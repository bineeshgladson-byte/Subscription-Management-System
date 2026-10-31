package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/brand"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
)

// An outbox row is committed with its business action, then delivered separately.
// SMTP cannot promise exactly-once delivery across a crash after acceptance, but
// event keys and row locks prevent normal retries/workers duplicating messages.
type EmailNotification struct {
	ID                       int64
	EventKey                 string
	CustomerID               int64
	InvoiceID                *int64
	Recipient, Subject, Body string
	Attempts                 int
	NextAttemptAt            time.Time
	SentAt                   *time.Time
	LastError                *string
	CreatedAt                time.Time
}

func queueCustomerEmail(tx *gorm.DB, key string, customerID int64, invoiceID *int64, subject, body string) error {
	var customer models.Customer
	if err := tx.Unscoped().First(&customer, customerID).Error; err != nil {
		return err
	}
	message := EmailNotification{EventKey: key, CustomerID: customerID, InvoiceID: invoiceID,
		Recipient: customer.Email, Subject: subject, Body: "Hi " + customer.Name + ",\n\n" + body + brand.EmailSignature,
		NextAttemptAt: time.Now()}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_key"}}, DoNothing: true}).Create(&message).Error
}

func queueSubscriptionEmail(tx *gorm.DB, id int64, event string) error {
	var sub models.Subscription
	if err := tx.Preload("Plan").First(&sub, id).Error; err != nil {
		return err
	}
	number := sub.QuotationNumber
	if sub.SubscriptionNumber != nil {
		number = *sub.SubscriptionNumber
	}
	name := "your plan"
	if sub.Plan != nil {
		name = sub.Plan.Name
	}
	body := fmt.Sprintf("%s\n\nReference: %s\nPlan: %s\nStatus: %s\nTotal per billing period (including tax): ₹%s\nStart date: %s\n",
		event, number, name, sub.Status, sub.Total.StringFixed(2), dateString(sub.StartDate))
	if sub.Plan != nil {
		body += "Billing: " + billingLabel(sub.Plan.BillingPeriod, sub.Plan.BillingInterval) + "\n"
	}
	if sub.NextBillingDate != nil && sub.Status == models.SubActive {
		body += "Next billing date: " + dateString(*sub.NextBillingDate) + "\n"
	}
	if sub.ExpiryDate != nil {
		body += "End date: " + dateString(*sub.ExpiryDate) + "\n"
	}
	if sub.QuotationValidUntil != nil && (sub.Status == models.SubDraft || sub.Status == models.SubQuotation) {
		body += "Quotation valid until: " + dateString(*sub.QuotationValidUntil) + "\n"
	}
	key := fmt.Sprintf("%s:%d:%d", event, id, sub.UpdatedAt.UnixNano())
	return queueCustomerEmail(tx, key, sub.CustomerID, nil, brand.Name+": "+event+" · "+number, body)
}

func queueInvoiceEmail(tx *gorm.DB, id int64, event string) error {
	var inv models.Invoice
	if err := tx.Preload("Items").First(&inv, id).Error; err != nil {
		return err
	}
	var b strings.Builder
	amountDue := inv.AmountDue
	if inv.Status == models.InvoiceCancelled {
		amountDue = inv.Total.Sub(inv.Total)
		b.WriteString("This invoice has been cancelled. No payment is due for this invoice.\n\n")
	}
	fmt.Fprintf(&b, "%s\n\nInvoice: %s\nInvoice date: %s\nDue date: %s\nStatus: %s\n", event, inv.InvoiceNumber, dateString(inv.InvoiceDate), dateString(inv.DueDate), inv.Status)
	if inv.PeriodStart != nil && inv.PeriodEnd != nil {
		fmt.Fprintf(&b, "Billing period: %s to %s\n", dateString(*inv.PeriodStart), dateString(*inv.PeriodEnd))
	}
	for _, item := range inv.Items {
		fmt.Fprintf(&b, "\n%d × %s @ ₹%s · line total ₹%s (discount ₹%s, GST %s%% ₹%s)", item.Quantity, item.Description, item.UnitPrice.StringFixed(2), item.LineTotal.StringFixed(2), item.DiscountAmount.StringFixed(2), item.TaxRate.StringFixed(2), item.TaxAmount.StringFixed(2))
	}
	fmt.Fprintf(&b, "\n\nSubtotal: ₹%s\nDiscounts: ₹%s\nTax: ₹%s\nTotal: ₹%s\nPaid: ₹%s\nAmount due: ₹%s\n", inv.Subtotal.StringFixed(2), inv.DiscountTotal.StringFixed(2), inv.TaxTotal.StringFixed(2), inv.Total.StringFixed(2), inv.AmountPaid.StringFixed(2), amountDue.StringFixed(2))
	return queueCustomerEmail(tx, fmt.Sprintf("%s:%d", event, id), inv.CustomerID, &inv.ID, brand.Name+": "+event+" · "+inv.InvoiceNumber, b.String())
}

// DispatchEmails retries due messages; each worker exclusively locks one row
// while sending. A bounded batch avoids monopolising the scheduler.
func (s *SalesService) DispatchEmails(ctx context.Context) { dispatchEmails(ctx, s.db, s.mailer) }

func dispatchEmails(ctx context.Context, db *gorm.DB, mail mailer.Mailer) {
	if mail == nil {
		return
	}
	for i := 0; i < 100 && ctx.Err() == nil; i++ {
		found := false
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var message EmailNotification
			result := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("sent_at IS NULL AND next_attempt_at <= ?", time.Now()).Order("id").Limit(1).Find(&message)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return nil
			}
			found = true
			if strings.HasPrefix(message.EventKey, "renewal-reminder:") {
				parts := strings.Split(message.EventKey, ":")
				if len(parts) != 4 {
					return fmt.Errorf("invalid reminder event key")
				}
				id, err := strconv.ParseInt(parts[1], 10, 64)
				if err != nil {
					return err
				}
				var applicable int64
				if err := tx.Table("subscriptions s").Joins("JOIN subscription_plans p ON p.id = s.plan_id").
					Where("s.id = ? AND s.status IN ('active','paused') AND p.is_renewable", id).
					Where("coalesce(s.expiry_date,s.next_billing_date) = ?", parts[2]).
					Where("s.expiry_date IS NOT NULL OR s.status = 'active'").
					Where("NOT EXISTS (SELECT 1 FROM subscriptions r WHERE r.parent_subscription_id = s.id AND r.status IN ('draft','quotation','confirmed','active','paused'))").Count(&applicable).Error; err != nil {
					return err
				}
				// No mail was sent: discard a queued reminder superseded by a
				// renewal, closure, pause or change of due date before its retry.
				if applicable == 0 {
					return tx.Delete(&message).Error
				}
			}
			if err := mail.Send(ctx, mailer.Message{To: message.Recipient, Subject: message.Subject, Body: message.Body}); err != nil {
				delay := time.Minute * time.Duration(min(message.Attempts+1, 60))
				slog.Warn("email delivery will retry", "notification_id", message.ID, "attempt", message.Attempts+1)
				return tx.Model(&message).Updates(map[string]any{"attempts": message.Attempts + 1, "next_attempt_at": time.Now().Add(delay), "last_error": err.Error()}).Error
			}
			now := time.Now()
			if err := tx.Model(&message).Updates(map[string]any{"sent_at": now, "attempts": message.Attempts + 1, "last_error": nil}).Error; err != nil {
				return err
			}
			if message.InvoiceID != nil {
				return tx.Model(&models.Invoice{}).Where("id = ?", *message.InvoiceID).Update("sent_at", now).Error
			}
			return nil
		})
		if err != nil {
			slog.Error("email queue failed", "err", err)
			return
		}
		if !found {
			return
		}
	}
}

func (s *SalesService) kickEmails() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		s.DispatchEmails(ctx)
	}()
}

// QueueRenewalReminders uses contract expiry when present, otherwise the next
// recurring billing date. A reminder never creates/renews a subscription.
func (s *SalesService) QueueRenewalReminders(ctx context.Context, asOf time.Time) error {
	var subs []models.Subscription
	err := s.db.WithContext(ctx).Preload("Plan").Where("status IN ('active','paused')").
		Where("NOT EXISTS (SELECT 1 FROM subscriptions renewal WHERE renewal.parent_subscription_id = subscriptions.id AND renewal.status IN ('draft','quotation','confirmed','active','paused'))").
		Where("((expiry_date IS NOT NULL AND expiry_date BETWEEN ? AND ?) OR (expiry_date IS NULL AND status = 'active' AND next_billing_date BETWEEN ? AND ?))", dateString(asOf.AddDate(0, 0, 1)), dateString(asOf.AddDate(0, 0, 7)), dateString(asOf.AddDate(0, 0, 1)), dateString(asOf.AddDate(0, 0, 7))).Find(&subs).Error
	if err != nil {
		return err
	}
	for _, sub := range subs {
		if sub.Plan == nil || !sub.Plan.IsRenewable {
			continue
		}
		due := sub.NextBillingDate
		if sub.ExpiryDate != nil {
			due = sub.ExpiryDate
		}
		if due == nil {
			continue
		}
		day, _ := time.Parse("2006-01-02", dateString(*due))
		today, _ := time.Parse("2006-01-02", dateString(asOf))
		days := int(day.Sub(today).Hours() / 24)
		if days != 7 && days != 3 && days != 1 {
			continue
		}
		num := sub.QuotationNumber
		if sub.SubscriptionNumber != nil {
			num = *sub.SubscriptionNumber
		}
		kind := "next billing date"
		if sub.ExpiryDate != nil {
			kind = "subscription end date"
		}
		body := fmt.Sprintf("Your %s for %s is %s, in %d day(s).\n\nSubscription: %s\nCurrent total per billing period (including tax): ₹%s\nReview your subscription in Subly to check the terms or request a renewal. This reminder does not charge you or renew your subscription.", kind, sub.Plan.Name, dateString(*due), days, num, sub.Total.StringFixed(2))
		if err := queueCustomerEmail(s.db.WithContext(ctx), fmt.Sprintf("renewal-reminder:%d:%s:%d", sub.ID, dateString(*due), days), sub.CustomerID, nil, brand.Name+": renewal reminder · "+num, body); err != nil {
			return err
		}
	}
	return nil
}
