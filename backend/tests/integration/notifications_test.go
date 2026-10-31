package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
)

func TestActionEmailsAndRenewalReminders(t *testing.T) {
	e := setup(t)
	adminEmail, pw := e.createAdmin()
	admin := e.login(adminEmail, pw)
	productID, _, _ := e.catalog(admin)
	r := e.call("POST", "/plans", admin, map[string]any{"name": "Email plan " + e.suffix, "billing_period": "monthly", "is_pausable": true, "products": []map[string]any{{"product_id": productID}}})
	e.expect(r, 201, "create reminder plan")
	planID := id(r)
	email := e.email("notifications")
	e.register(email, "Brightloop#2026")
	customer := e.login(email, "Brightloop#2026")
	custID := int64(e.call("GET", "/auth/me", customer, nil).data()["customer_id"].(float64))
	today := time.Now().UTC().Truncate(24 * time.Hour)
	r = e.call("POST", "/me/quotations", customer, map[string]any{"plan_id": planID, "start_date": today.Format("2006-01-02")})
	e.expect(r, 200, "request quotation")
	subID := id(r)
	r = e.call("POST", fmt.Sprintf("/quotations/%d/confirm", subID), admin, nil)
	e.expect(r, 200, "confirm and invoice")
	invID := int64(r.data()["invoices"].([]any)[0].(map[string]any)["id"].(float64))
	var issued int64
	e.db.Model(&service.EmailNotification{}).Where("event_key = ?", fmt.Sprintf("Invoice issued:%d", invID)).Count(&issued)
	if issued != 1 {
		t.Fatal("issued invoice must be queued once")
	}
	for _, action := range []string{"pause", "resume"} {
		e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/%s", subID, action), customer, nil), 200, action)
	}
	var invoices int64
	e.db.Model(&models.Invoice{}).Where("subscription_id = ?", subID).Count(&invoices)
	if invoices != 1 {
		t.Fatal("same-day resume must not create another invoice")
	}
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/pause", subID), customer, nil), 200, "pause again")
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/pause", subID), customer, nil), 409, "invalid repeat pause")
	var pauses int64
	e.db.Model(&service.EmailNotification{}).Where("customer_id = ? AND subject LIKE ?", custID, "%Subscription paused%").Count(&pauses)
	if pauses != 2 {
		t.Fatalf("expected one email per actual pause, got %d", pauses)
	}
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/resume", subID), customer, nil), 200, "resume again")

	payment := map[string]any{"method": "upi", "amount": "100.00", "payment_date": today.Format("2006-01-02")}
	// No idempotency header in e.call: test receipt event count against each actual payment.
	e.expect(e.call("POST", fmt.Sprintf("/invoices/%d/payments", invID), admin, payment), 201, "record payment")
	var receipt service.EmailNotification
	if err := e.db.Where("customer_id = ? AND event_key LIKE 'payment-received:%'", custID).First(&receipt).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(receipt.Body, "₹100.00") || !strings.Contains(receipt.Body, "Remaining amount due") {
		t.Fatal("receipt missing payment breakdown")
	}

	// Reminders are durable, repeatable across concurrent scheduler runs and due dates.
	due := today.AddDate(0, 0, 7)
	e.db.Model(&models.Subscription{}).Where("id = ?", subID).Update("expiry_date", due)
	for _, advance := range []int{0, 4, 6} {
		asOf := today.AddDate(0, 0, advance)
		var wg sync.WaitGroup
		for range 3 {
			wg.Go(func() {
				if err := e.sales.QueueRenewalReminders(context.Background(), asOf); err != nil {
					t.Error(err)
				}
			})
		}
		wg.Wait()
	}
	var reminders []service.EmailNotification
	e.db.Where("customer_id = ? AND event_key LIKE 'renewal-reminder:%'", custID).Find(&reminders)
	if len(reminders) != 3 {
		t.Fatalf("want 7/3/1 reminder once each, got %d", len(reminders))
	}
	for _, m := range reminders {
		if m.Recipient != email || !strings.Contains(m.Body, due.Format("2006-01-02")) {
			t.Fatal("reminder recipient/date incorrect")
		}
	}
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/renew", subID), customer, nil), 200, "renew")
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/renew", subID), customer, nil), 409, "renew twice")
	var children []models.Subscription
	e.db.Where("parent_subscription_id = ?", subID).Find(&children)
	if len(children) != 1 || children[0].Status != models.SubQuotation {
		t.Fatal("renewal must make exactly one customer-visible quotation")
	}
	var original models.Subscription
	e.db.First(&original, subID)
	if original.Status != models.SubActive {
		t.Fatal("advance renewal request must preserve the current subscription")
	}
	if err := e.sales.QueueRenewalReminders(context.Background(), today); err != nil {
		t.Fatal(err)
	}
	var count int64
	e.db.Model(&service.EmailNotification{}).Where("customer_id = ? AND event_key LIKE 'renewal-reminder:%'", custID).Count(&count)
	if count > 3 {
		t.Fatal("renewed subscriptions must not create further reminders")
	}
	// Confirming the future renewal keeps the original active until activation.
	e.expect(e.call("POST", fmt.Sprintf("/quotations/%d/confirm", children[0].ID), admin, nil), 200, "confirm future renewal")
	e.db.First(&original, subID)
	if original.Status != models.SubActive {
		t.Fatal("future confirmation must not stop current service")
	}
	restarted := service.NewSalesService(e.db, repository.NewProductRepo(), repository.NewDiscountRepo(), clock.Fixed{T: children[0].StartDate}, e.mail, true)
	restarted.RunBilling(context.Background(), children[0].StartDate)
	e.db.First(&original, subID)
	if original.Status != models.SubRenewed {
		t.Fatal("replacement activation must retire the original subscription")
	}
	stale := service.EmailNotification{EventKey: fmt.Sprintf("renewal-reminder:%d:%s:7", subID, due.AddDate(0, 0, 10).Format("2006-01-02")), CustomerID: custID, Recipient: email, Subject: "Obsolete reminder", Body: "Must not send", NextAttemptAt: time.Now()}
	if err := e.db.Create(&stale).Error; err != nil {
		t.Fatal(err)
	}
	restarted.DispatchEmails(context.Background())
	var staleCount int64
	e.db.Model(&service.EmailNotification{}).Where("id = ?", stale.ID).Count(&staleCount)
	if staleCount != 0 {
		t.Fatal("obsolete unsent reminders must be discarded")
	}
	e.mail.mu.Lock()
	for _, m := range e.mail.sent {
		if m.Subject == "Obsolete reminder" {
			t.Error("obsolete reminder was sent")
		}
	}
	e.mail.mu.Unlock()

	// Retry records survive a service restart; workers skip delivered messages.
	e.sales.DispatchEmails(context.Background())
	e.mail.mu.Lock()
	e.mail.fail = true
	e.mail.mu.Unlock()
	retry := service.EmailNotification{EventKey: "retry-" + e.suffix, CustomerID: custID, Recipient: email, Subject: "Retry proof", Body: "A saved email", NextAttemptAt: time.Now()}
	if err := e.db.Create(&retry).Error; err != nil {
		t.Fatal(err)
	}
	restarted.DispatchEmails(context.Background())
	e.db.First(&retry, retry.ID)
	if retry.SentAt != nil || retry.Attempts < 1 || retry.LastError == nil {
		t.Fatal("failed send must remain pending")
	}
	e.mail.mu.Lock()
	e.mail.fail = false
	e.mail.mu.Unlock()
	e.db.Model(&retry).Update("next_attempt_at", time.Now())
	e.sales.DispatchEmails(context.Background())
	e.db.First(&retry, retry.ID)
	if retry.SentAt == nil {
		t.Fatal("retry was not delivered")
	}
	before := e.mail.countTo(email)
	e.sales.DispatchEmails(context.Background())
	if e.mail.countTo(email) != before {
		t.Fatal("delivered messages were sent again")
	}
}
