package integration

import (
	"fmt"
	"testing"
	"time"
)

func TestPlans(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)
	cust := e.customerToken()

	// Catalogue: a seat product with an Odoo variant (999 + 560) and a support add-on.
	r := e.call("POST", "/products", admin, map[string]any{"name": "Seat " + e.suffix, "product_type": "software", "sales_price": "999",
		"variants": []map[string]any{{"attribute": "Brand", "name": "Odoo Brand", "extra_price": "560"}}})
	e.expect(r, 201, "seat product")
	seatID := id(r)
	odooID := int64(variantByName(t, r, "Odoo Brand")["id"].(float64))
	r = e.call("POST", "/products", admin, map[string]any{"name": "Support " + e.suffix, "product_type": "addon", "sales_price": "499"})
	e.expect(r, 201, "support product")
	supportID := id(r)

	name := "Business Quarterly " + e.suffix
	lines := []map[string]any{
		{"product_id": seatID, "variant_id": odooID, "quantity": 2}, // 2 × 1559 = 3118
		{"product_id": supportID, "price_override": "100"},          // 1 × 100  =  100
	}

	// ---- validation ----
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "monthly"}), 400, "no products")
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "hourly", "products": lines}), 400, "bad period")
	r = e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "monthly", "products": lines,
		"available_from": "2026-12-01", "available_until": "2026-11-01"})
	e.expect(r, 400, "until before from")
	if r.errDetail("available_until") == "" {
		t.Fatalf("expected available_until error: %v", r.Body)
	}
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "monthly", "products": lines,
		"auto_close_enabled": true}), 400, "auto-close without cycles")
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "monthly",
		"products": []map[string]any{{"product_id": seatID, "variant_id": 999999999}}}), 400, "variant of another product")
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "monthly",
		"products": []map[string]any{{"product_id": supportID}, {"product_id": supportID}}}), 400, "duplicate lines")

	// ---- create ----
	r = e.call("POST", "/plans", admin, map[string]any{"name": name, "billing_period": "monthly", "billing_interval": 3,
		"min_quantity": 5, "is_pausable": true, "products": lines})
	e.expect(r, 201, "create plan")
	planID := id(r)
	d := r.data()
	if d["price"] != "3218.00" || d["billing_label"] != "Every 3 months" || d["min_quantity"].(float64) != 5 {
		t.Fatalf("unexpected plan: price=%v label=%v min=%v", d["price"], d["billing_label"], d["min_quantity"])
	}
	if d["is_pausable"] != true || d["is_closable"] != true || d["is_renewable"] != true || d["is_available_now"] != true {
		t.Fatalf("defaults wrong: %v", d)
	}
	if l := d["products"].([]any)[0].(map[string]any); l["unit_price"] != "1559.00" || l["line_total"] != "3118.00" {
		t.Fatalf("line wrong: %v", l)
	}
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": upper(name), "billing_period": "yearly", "products": lines}), 409, "duplicate name")

	// ---- who sees what ----
	e.expect(e.call("POST", "/plans", staff, map[string]any{"name": "x", "billing_period": "monthly", "products": lines}), 403, "staff creates plan")
	e.expect(e.call("POST", "/plans", cust, map[string]any{"name": "x", "billing_period": "monthly", "products": lines}), 403, "customer creates plan")
	r = e.call("GET", fmt.Sprintf("/plans/%d", planID), staff, nil)
	e.expect(r, 200, "staff reads plan")
	if r.data()["active_subscriptions"].(float64) != 0 {
		t.Fatal("staff should see active_subscriptions")
	}
	r = e.call("GET", fmt.Sprintf("/plans/%d", planID), cust, nil)
	e.expect(r, 200, "customer reads available plan")
	if _, has := r.data()["active_subscriptions"]; has {
		t.Fatal("customers must not see active_subscriptions")
	}

	// A plan that starts in the future: admin sees it, customer doesn't.
	future := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	r = e.call("POST", "/plans", admin, map[string]any{"name": "Future " + e.suffix, "billing_period": "yearly",
		"available_from": future, "products": []map[string]any{{"product_id": supportID}}})
	e.expect(r, 201, "future plan")
	futureID := id(r)
	if r.data()["is_available_now"] != false {
		t.Fatal("future plan isn't available now")
	}
	e.expect(e.call("GET", fmt.Sprintf("/plans/%d", futureID), cust, nil), 404, "customer opens future plan")
	r = e.call("GET", "/plans?q="+e.suffix, cust, nil)
	e.expect(r, 200, "customer catalogue")
	for _, p := range r.Body["data"].([]any) {
		if int64(p.(map[string]any)["id"].(float64)) == futureID {
			t.Fatal("customer catalogue must not include the future plan")
		}
	}
	r = e.call("GET", "/plans?q="+e.suffix+"&status=all&available_on="+future, admin, nil)
	e.expect(r, 200, "admin filter by date")
	if len(r.Body["data"].([]any)) != 2 {
		t.Fatalf("both plans are available next month: %v", len(r.Body["data"].([]any)))
	}
	e.expect(e.call("GET", "/plans?available_on=tomorrow", admin, nil), 400, "bad date filter")

	// ---- price follows the catalogue ----
	e.expect(e.call("PUT", fmt.Sprintf("/products/%d", seatID), admin, map[string]any{"sales_price": "1000"}), 200, "seat price change")
	if p := e.call("GET", fmt.Sprintf("/plans/%d", planID), admin, nil).data()["price"]; p != "3220.00" {
		t.Fatalf("plan price should be 2 × 1560 + 100 = 3220.00, got %v", p)
	}

	// Products in an active plan can't be archived / variants deactivated.
	r = e.call("DELETE", fmt.Sprintf("/products/%d", seatID), admin, nil)
	e.expect(r, 422, "archive product used by a plan")
	if r.errDetail("rule") != "PRODUCT_IN_USE" {
		t.Fatalf("expected PRODUCT_IN_USE: %v", r.Body)
	}
	r = e.call("DELETE", fmt.Sprintf("/products/%d/variants/%d", seatID, odooID), admin, nil)
	e.expect(r, 422, "deactivate variant used by a plan")
	if r.errDetail("rule") != "VARIANT_IN_USE" {
		t.Fatalf("expected VARIANT_IN_USE: %v", r.Body)
	}

	// ---- update: replace products, dates, auto-close ----
	r = e.call("PUT", fmt.Sprintf("/plans/%d", planID), admin, map[string]any{
		"products": []map[string]any{{"product_id": supportID, "quantity": 3}}, "auto_close_enabled": true, "auto_close_after_cycles": 4,
		"available_until": "2099-12-31"})
	e.expect(r, 200, "update plan")
	if r.data()["price"] != "1497.00" || r.data()["auto_close_after_cycles"].(float64) != 4 || r.data()["available_until"] != "2099-12-31" {
		t.Fatalf("update not applied: %v", r.data())
	}
	r = e.call("PUT", fmt.Sprintf("/plans/%d", planID), admin, map[string]any{"available_until": "", "auto_close_enabled": false})
	e.expect(r, 200, "clear end date and auto-close")
	if r.data()["available_until"] != nil || r.data()["auto_close_after_cycles"] != nil {
		t.Fatalf("should be cleared: %v", r.data())
	}

	// Billing frequency is locked once someone subscribes.
	if err := e.db.Exec(`INSERT INTO subscriptions (quotation_number, customer_id, plan_id, status, start_date, billing_anchor_day)
		VALUES (?, (SELECT id FROM customers LIMIT 1), ?, 'active', CURRENT_DATE, 1)`, "QT/PLAN/"+e.suffix, planID).Error; err != nil {
		t.Fatal(err)
	}
	r = e.call("PUT", fmt.Sprintf("/plans/%d", planID), admin, map[string]any{"billing_period": "yearly"})
	e.expect(r, 422, "change billing period with subscribers")
	if r.errDetail("rule") != "PLAN_IN_USE" {
		t.Fatalf("expected PLAN_IN_USE: %v", r.Body)
	}
	e.expect(e.call("PUT", fmt.Sprintf("/plans/%d", planID), admin, map[string]any{"min_quantity": 1}), 200, "other edits still allowed")
	e.expect(e.call("PUT", fmt.Sprintf("/plans/%d", futureID), admin, map[string]any{"billing_period": "monthly"}), 200, "change period with no subscribers")

	// ---- archive / reactivate ----
	r = e.call("PATCH", fmt.Sprintf("/plans/%d/status", futureID), admin, map[string]any{"status": "archived"})
	e.expect(r, 200, "archive plan")
	e.expect(e.call("PATCH", fmt.Sprintf("/plans/%d/status", planID), admin, map[string]any{"status": "archived"}), 200, "archive plan 2")
	e.expect(e.call("GET", fmt.Sprintf("/plans/%d", planID), cust, nil), 404, "customer opens archived plan")
	// With no active plans left using it, the support product can be archived…
	e.expect(e.call("DELETE", fmt.Sprintf("/products/%d", supportID), admin, nil), 200, "archive unused product")
	// …and then the plan can't come back until its products are fixed.
	r = e.call("PATCH", fmt.Sprintf("/plans/%d/status", planID), admin, map[string]any{"status": "active"})
	e.expect(r, 422, "reactivate plan with archived product")
	if r.errDetail("rule") != "PLAN_HAS_INACTIVE_PRODUCTS" {
		t.Fatalf("expected PLAN_HAS_INACTIVE_PRODUCTS: %v", r.Body)
	}
	e.expect(e.call("PATCH", fmt.Sprintf("/plans/%d/status", planID), admin, map[string]any{"status": "paused"}), 400, "bad status")
	e.expect(e.call("GET", "/plans/999999999", admin, nil), 404, "unknown plan")
}
