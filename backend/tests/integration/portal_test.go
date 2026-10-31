package integration

import (
	"fmt"
	"testing"
	"time"
)

func TestCustomerPortal(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	productID, odooID, _ := e.catalog(admin)
	r := e.call("POST", "/plans", admin, map[string]any{"name": "Portal plan " + e.suffix, "billing_period": "monthly",
		"is_pausable": true, "products": []map[string]any{{"product_id": productID, "variant_id": odooID}}})
	planID := id(r)

	// Two customers who signed up; each gets a subscription.
	aEmail, bEmail := e.email("alice"), e.email("bob")
	e.register(aEmail, "Brightloop#2026")
	e.register(bEmail, "Brightloop#2026")
	alice, bob := e.login(aEmail, "Brightloop#2026"), e.login(bEmail, "Brightloop#2026")
	aCust := int64(e.call("GET", "/auth/me", alice, nil).data()["customer_id"].(float64))
	bCust := int64(e.call("GET", "/auth/me", bob, nil).data()["customer_id"].(float64))
	start := time.Now().Format("2006-01-02")

	// Customer can request a quotation from a plan.
	r = e.call("POST", "/me/quotations", alice, map[string]any{"plan_id": planID, "start_date": start})
	e.expect(r, 200, "customer requests quotation")
	if r.data()["status"] != "quotation" || int64(r.data()["customer_id"].(float64)) != aCust {
		t.Fatalf("expected own quotation in quotation status: %v", r.data())
	}
	if int64(r.data()["plan_id"].(float64)) != planID {
		t.Fatalf("quotation should use requested plan: %v", r.data())
	}
	e.expect(e.call("POST", "/me/quotations", bob, map[string]any{"plan_id": 99999999}), 400, "invalid plan in quotation request")

	sub := func(cust int64) int64 {
		r := e.call("POST", "/quotations", admin, map[string]any{"customer_id": cust, "plan_id": planID, "start_date": start})
		e.expect(r, 201, "quotation")
		e.expect(e.call("POST", fmt.Sprintf("/quotations/%d/confirm", id(r)), admin, nil), 200, "confirm")
		return id(r)
	}
	aSub, bSub := sub(aCust), sub(bCust)

	// Alice sees only her own subscription and invoice.
	r = e.call("GET", "/me/subscriptions", alice, nil)
	e.expect(r, 200, "my subscriptions")
	if list := r.Body["data"].([]any); len(list) != 2 {
		t.Fatalf("alice should see her quotation + subscription: %v", list)
	} else {
		seenSub, seenQuote := false, false
		for _, row := range list {
			rm := row.(map[string]any)
			id := int64(rm["id"].(float64))
			if id == aSub {
				seenSub = true
			}
			if rm["status"] == "quotation" {
				seenQuote = true
			}
		}
		if !seenSub || !seenQuote {
			t.Fatalf("alice portal list should include active sub and pending quotation: %v", list)
		}
	}
	r = e.call("GET", "/me/invoices", alice, nil)
	inv := r.Body["data"].([]any)
	if len(inv) != 1 {
		t.Fatalf("alice should see 1 invoice: %v", inv)
	}
	aInv := int64(inv[0].(map[string]any)["id"].(float64))
	e.expect(e.call("GET", fmt.Sprintf("/me/invoices/%d", aInv), alice, nil), 200, "own invoice")
	e.expect(e.call("GET", fmt.Sprintf("/me/subscriptions/%d", bSub), alice, nil), 404, "someone else's subscription")
	e.expect(e.call("GET", fmt.Sprintf("/me/invoices/%d", aInv), bob, nil), 404, "someone else's invoice")
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/pause", bSub), alice, nil), 404, "pause someone else's")

	// Payment history.
	e.expect(e.call("POST", fmt.Sprintf("/invoices/%d/payments", aInv), admin, map[string]any{"method": "upi", "amount": "100", "payment_date": start}), 201, "payment")
	r = e.call("GET", "/me/payments", alice, nil)
	if len(r.Body["data"].([]any)) != 1 {
		t.Fatal("alice should see her payment")
	}
	if len(e.call("GET", "/me/payments", bob, nil).Body["data"].([]any)) != 0 {
		t.Fatal("bob must not see alice's payment")
	}

	// Manage own subscription within the plan's rules.
	r = e.call("POST", fmt.Sprintf("/me/subscriptions/%d/pause", aSub), alice, nil)
	e.expect(r, 200, "pause own")
	if r.data()["status"] != "paused" {
		t.Fatal("should be paused")
	}
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/resume", aSub), alice, nil), 200, "resume own")
	r = e.call("POST", fmt.Sprintf("/me/subscriptions/%d/renew", aSub), alice, nil)
	e.expect(r, 200, "renew own (creates a renewal quotation)")
	e.expect(e.call("POST", fmt.Sprintf("/me/subscriptions/%d/close", bSub), bob, nil), 200, "close own")

	// Staff/admin can't use the portal; customers can't use staff routes.
	e.expect(e.call("GET", "/me/subscriptions", admin, nil), 403, "admin on portal")
	e.expect(e.call("GET", "/subscriptions", alice, nil), 403, "customer on staff route")
}
