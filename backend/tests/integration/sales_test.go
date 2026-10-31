package integration

import (
	"fmt"
	"testing"
	"time"
)

// TestAdminFlow walks the whole brief: quotation → subscription → automatic
// invoice → payments → paid, plus "nothing billed twice or skipped".
func TestAdminFlow(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)
	productID, odooID, planID := e.catalog(admin) // yearly plan, 1 × Odoo Brand (1559), GST 18%
	// A monthly plan for billing-cycle checks.
	r := e.call("POST", "/plans", admin, map[string]any{"name": "Monthly " + e.suffix, "billing_period": "monthly",
		"is_pausable": true, "products": []map[string]any{{"product_id": productID, "variant_id": odooID}}})
	e.expect(r, 201, "monthly plan")
	monthlyID := id(r)
	r = e.call("POST", "/customers", admin, map[string]any{"name": "Rahul " + e.suffix, "email": e.email("rahul")})
	e.expect(r, 201, "customer")
	custID := id(r)
	today := time.Now()
	start := today.Format("2006-01-02")

	// ---- quotation ----
	e.expect(e.call("POST", "/quotations", admin, map[string]any{"customer_id": custID, "plan_id": 999999999}), 400, "unknown plan")
	r = e.call("POST", "/quotations", staff, map[string]any{"customer_id": custID, "plan_id": monthlyID, "start_date": start})
	e.expect(r, 201, "create quotation (staff)")
	q := r.data()
	qID := id(r)
	if q["status"] != "draft" || q["total"] != "1839.62" || len(q["items"].([]any)) != 1 {
		t.Fatalf("quotation: status=%v total=%v (want draft, 1559 + 18%% = 1839.62)", q["status"], q["total"])
	}
	e.expect(e.call("POST", fmt.Sprintf("/quotations/%d/send", qID), staff, nil), 200, "send")

	// ---- confirm → active + first invoice right away ----
	r = e.call("POST", fmt.Sprintf("/quotations/%d/confirm", qID), staff, nil)
	e.expect(r, 200, "confirm")
	sub := r.data()
	if sub["status"] != "active" || sub["subscription_number"] == nil || len(sub["invoices"].([]any)) != 1 {
		t.Fatalf("confirm should activate and bill the first period: %v", sub)
	}
	inv := sub["invoices"].([]any)[0].(map[string]any)
	invID := int64(inv["id"].(float64))
	if inv["total"] != "1839.62" || inv["status"] != "confirmed" || inv["period_start"] != start {
		t.Fatalf("first invoice wrong: %v", inv)
	}
	e.expect(e.call("POST", fmt.Sprintf("/quotations/%d/confirm", qID), staff, nil), 409, "confirm twice")

	// ---- nothing billed twice ----
	e.expect(e.call("POST", fmt.Sprintf("/subscriptions/%d/bill-now", qID), admin, nil), 422, "bill-now when nothing is due")
	for i := 0; i < 3; i++ {
		e.expect(e.call("POST", "/admin/billing/run", admin, map[string]any{}), 200, "billing run (today)")
	}
	r = e.call("GET", fmt.Sprintf("/invoices?customer_id=%d", custID), admin, nil)
	if n := len(r.Body["data"].([]any)); n != 1 {
		t.Fatalf("running billing again must not duplicate: %d invoices", n)
	}

	// ---- payments: partial, overpay, then paid ----
	e.expect(e.call("POST", fmt.Sprintf("/invoices/%d/payments", invID), staff, map[string]any{"method": "upi", "amount": "5000", "payment_date": start}), 422, "overpayment")
	e.expect(e.call("POST", fmt.Sprintf("/invoices/%d/payments", invID), staff, map[string]any{"method": "upi", "amount": "1000", "payment_date": start, "reference": "UTR1"}), 201, "partial payment")
	r = e.call("GET", fmt.Sprintf("/invoices/%d", invID), admin, nil)
	if r.data()["payment_status"] != "partial" || r.data()["amount_due"] != "839.62" || len(r.data()["payments"].([]any)) != 1 {
		t.Fatalf("after partial: %v", r.data())
	}
	e.expect(e.call("POST", fmt.Sprintf("/invoices/%d/cancel", invID), admin, nil), 409, "cancel a part-paid invoice")
	e.expect(e.call("POST", fmt.Sprintf("/invoices/%d/payments", invID), staff, map[string]any{"method": "bank_transfer", "amount": "839.62", "payment_date": start}), 201, "rest")
	r = e.call("GET", fmt.Sprintf("/invoices/%d", invID), admin, nil)
	if r.data()["status"] != "paid" || r.data()["payment_status"] != "paid" {
		t.Fatalf("should be paid: %v", r.data())
	}
	r = e.call("GET", fmt.Sprintf("/payments?invoice_id=%d", invID), admin, nil)
	e.expect(r, 200, "payments list")
	if len(r.Body["data"].([]any)) != 2 {
		t.Fatal("expected 2 payments")
	}

	// ---- no skipped cycles: billing 3 months ahead creates 3 more invoices, once ----
	ahead := today.AddDate(0, 3, 0).Format("2006-01-02")
	r = e.call("POST", "/admin/billing/run", admin, map[string]any{"as_of": ahead})
	e.expect(r, 200, "billing run 3 months ahead")
	e.call("POST", "/admin/billing/run", admin, map[string]any{"as_of": ahead})
	r = e.call("GET", fmt.Sprintf("/invoices?customer_id=%d", custID), admin, nil)
	if n := len(r.Body["data"].([]any)); n != 4 {
		t.Fatalf("expected 4 invoices (1 + 3 caught-up periods), got %d", n)
	}

	// ---- lifecycle: pause (allowed) / close; yearly plan isn't pausable ----
	e.expect(e.call("POST", fmt.Sprintf("/subscriptions/%d/pause", qID), staff, nil), 200, "pause")
	e.expect(e.call("POST", fmt.Sprintf("/subscriptions/%d/resume", qID), staff, nil), 200, "resume")
	r = e.call("POST", "/quotations", admin, map[string]any{"customer_id": custID, "plan_id": planID, "start_date": start})
	yID := id(r)
	e.call("POST", fmt.Sprintf("/quotations/%d/confirm", yID), admin, nil)
	r = e.call("POST", fmt.Sprintf("/subscriptions/%d/pause", yID), admin, nil)
	e.expect(r, 422, "pause a non-pausable plan")
	e.expect(e.call("POST", fmt.Sprintf("/subscriptions/%d/close", yID), admin, nil), 200, "close early")

	// ---- cancel an unpaid invoice (draft quotation flow) ----
	r = e.call("GET", fmt.Sprintf("/invoices?customer_id=%d&status=confirmed", custID), admin, nil)
	unpaid := int64(r.Body["data"].([]any)[0].(map[string]any)["id"].(float64))
	r = e.call("POST", fmt.Sprintf("/invoices/%d/cancel", unpaid), admin, map[string]any{"reason": "test"})
	e.expect(r, 200, "cancel unpaid invoice")
	if r.data()["status"] != "cancelled" {
		t.Fatal("should be cancelled")
	}

	// ---- dashboard: admin sees money, staff doesn't ----
	r = e.call("GET", "/dashboard/summary", admin, nil)
	e.expect(r, 200, "dashboard admin")
	if r.data()["mrr"] == nil || r.data()["revenue_received"] == nil || r.data()["active_subscriptions"].(float64) < 1 {
		t.Fatalf("admin dashboard: %v", r.data())
	}
	r = e.call("GET", "/dashboard/summary", staff, nil)
	if r.data()["mrr"] != nil || r.data()["revenue_received"] != nil {
		t.Fatalf("staff must not see money: %v", r.data())
	}
	e.expect(e.call("POST", "/admin/billing/run", staff, nil), 403, "staff runs billing")
	e.expect(e.call("GET", "/quotations", e.customerToken(), nil), 403, "customer lists quotations")
}
