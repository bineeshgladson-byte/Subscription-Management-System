package integration

import (
	"fmt"
	"testing"
	"time"
)

// catalog creates a taxed product with an Odoo variant and a plan using it.
func (e *env) catalog(admin string) (productID, odooID, planID int64) {
	e.t.Helper()
	taxID := e.newTax(admin, "18")
	r := e.call("POST", "/products", admin, map[string]any{"name": "Seat " + e.suffix, "product_type": "software",
		"sales_price": "999", "tax_id": taxID,
		"variants": []map[string]any{{"attribute": "Brand", "name": "Odoo Brand", "extra_price": "560"}}})
	e.expect(r, 201, "product")
	productID = id(r)
	odooID = int64(variantByName(e.t, r, "Odoo Brand")["id"].(float64))
	r = e.call("POST", "/plans", admin, map[string]any{"name": "Plan " + e.suffix, "billing_period": "yearly",
		"products": []map[string]any{{"product_id": productID, "variant_id": odooID, "quantity": 1}}})
	e.expect(r, 201, "plan")
	return productID, odooID, id(r)
}

func TestQuotationTemplates(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)
	cust := e.customerToken()
	productID, odooID, planID := e.catalog(admin)
	name := "Standard yearly offer " + e.suffix

	e.expect(e.call("POST", "/quotation-templates", admin, map[string]any{"name": name}), 400, "no lines and no plan")
	e.expect(e.call("POST", "/quotation-templates", admin, map[string]any{"name": name, "plan_id": 999999999}), 400, "unknown plan")

	// Lines default to the plan's products.
	r := e.call("POST", "/quotation-templates", admin, map[string]any{"name": name, "plan_id": planID, "validity_days": 30,
		"terms": "Prices exclusive of GST."})
	e.expect(r, 201, "template from plan")
	tplID := id(r)
	d := r.data()
	lines := d["lines"].([]any)
	if len(lines) != 1 || lines[0].(map[string]any)["unit_price"] != "1559.00" || d["estimated_total"] != "1559.00" {
		t.Fatalf("lines should come from the plan: %v", d)
	}
	if d["validity_days"].(float64) != 30 || d["payment_terms_days"].(float64) != 15 || d["plan"].(map[string]any)["billing_label"] != "Yearly" {
		t.Fatalf("defaults/plan summary wrong: %v", d)
	}
	e.expect(e.call("POST", "/quotation-templates", admin, map[string]any{"name": upper(name), "plan_id": planID}), 409, "duplicate name")

	// Explicit lines replace them; duplicates rejected.
	e.expect(e.call("PUT", fmt.Sprintf("/quotation-templates/%d", tplID), admin, map[string]any{"lines": []map[string]any{
		{"product_id": productID}, {"product_id": productID}}}), 400, "duplicate lines")
	r = e.call("PUT", fmt.Sprintf("/quotation-templates/%d", tplID), admin, map[string]any{"lines": []map[string]any{
		{"product_id": productID, "variant_id": odooID, "quantity": 10}}, "payment_terms_days": 30})
	e.expect(r, 200, "update lines")
	if r.data()["estimated_total"] != "15590.00" || r.data()["payment_terms_days"].(float64) != 30 {
		t.Fatalf("update not applied: %v", r.data())
	}

	// Permissions
	e.expect(e.call("GET", "/quotation-templates?q="+e.suffix, staff, nil), 200, "staff lists")
	e.expect(e.call("POST", "/quotation-templates", staff, map[string]any{"name": "x", "plan_id": planID}), 403, "staff creates")
	e.expect(e.call("GET", "/quotation-templates", cust, nil), 403, "customer lists")

	// Deactivate + filter
	r = e.call("DELETE", fmt.Sprintf("/quotation-templates/%d", tplID), admin, nil)
	e.expect(r, 200, "deactivate")
	if r.data()["is_active"] != false {
		t.Fatal("should be inactive")
	}
	r = e.call("GET", fmt.Sprintf("/quotation-templates?is_active=false&plan_id=%d", planID), admin, nil)
	if len(r.Body["data"].([]any)) != 1 {
		t.Fatalf("filter should find it: %v", r.Body["data"])
	}
	e.expect(e.call("PUT", fmt.Sprintf("/quotation-templates/%d", tplID), admin, map[string]any{"plan_id": 0, "is_active": true}), 200, "remove plan + reactivate")
	e.expect(e.call("GET", "/quotation-templates/999999999", admin, nil), 404, "unknown template")
}

func TestDiscountsAndPricing(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)
	productID, odooID, _ := e.catalog(admin)
	today := time.Now().Format("2006-01-02")
	future := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	// Discounts apply to every price in the shared dev DB, so switch ours off afterwards.
	t.Cleanup(func() { e.db.Exec("UPDATE discounts SET is_active = false WHERE name LIKE ?", "%"+e.suffix) })

	// ---- validation ----
	base := map[string]any{"name": "Bulk " + e.suffix, "discount_type": "percentage", "scope": "subscription"}
	with := func(extra map[string]any) map[string]any {
		m := map[string]any{}
		for k, v := range base {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	e.expect(e.call("POST", "/discounts", admin, with(map[string]any{"value": "120"})), 400, "percent over 100")
	e.expect(e.call("POST", "/discounts", admin, with(map[string]any{"value": "0"})), 400, "zero value")
	e.expect(e.call("POST", "/discounts", admin, with(map[string]any{"value": "10", "starts_on": future, "ends_on": today})), 400, "ends before starts")
	e.expect(e.call("POST", "/discounts", admin, with(map[string]any{"value": "10", "scope": "product"})), 400, "product scope without products")

	// ---- permissions ----
	e.expect(e.call("POST", "/discounts", staff, with(map[string]any{"value": "10"})), 403, "staff creates discount")
	e.expect(e.call("GET", "/discounts", staff, nil), 403, "staff lists discounts")

	// ---- the plan's example: "10% off orders over 10,000", max 1 use ----
	code := "BULK" + e.suffix[len(e.suffix)-6:]
	r := e.call("POST", "/discounts", admin, with(map[string]any{"value": "10", "min_purchase_amount": "10000",
		"max_usage": 1, "code": code}))
	e.expect(r, 201, "create subscription discount")
	bulkID := id(r)
	if r.data()["state"] != "active" || r.data()["label"] != "10% off" || r.data()["remaining_uses"].(float64) != 1 {
		t.Fatalf("unexpected discount: %v", r.data())
	}
	e.expect(e.call("POST", "/discounts", admin, with(map[string]any{"name": "Other", "value": "5", "code": code})), 409, "duplicate code")

	// Acme: 10 × Odoo Brand (1559), GST 18% → 16,556.58 (this discount must be the one applied)
	acme := map[string]any{"lines": []map[string]any{{"product_id": productID, "variant_id": odooID, "quantity": 10}}}
	r = e.call("POST", "/pricing/preview", staff, acme)
	e.expect(r, 200, "price preview (staff)")
	applied := false
	for _, a := range r.data()["discounts"].([]any) {
		if int64(a.(map[string]any)["discount_id"].(float64)) == bulkID {
			applied = true
		}
	}
	if !applied {
		t.Fatalf("bulk discount should apply automatically: %v", r.data())
	}
	if r.data()["subtotal"] != "15590.00" {
		t.Fatalf("subtotal: %v", r.data())
	}
	// Only our discount can apply to this brand-new product unless other subscription-wide
	// discounts exist in the shared dev DB, so check the exact total only when it's the only one.
	if len(r.data()["discounts"].([]any)) == 1 && r.data()["total"] != "16556.58" {
		t.Fatalf("Acme total should be 16556.58: %v", r.data()["total"])
	}

	// Below the minimum purchase → not applied
	r = e.call("POST", "/pricing/preview", staff, map[string]any{"lines": []map[string]any{{"product_id": productID, "quantity": 1}}})
	for _, a := range r.data()["discounts"].([]any) {
		if int64(a.(map[string]any)["discount_id"].(float64)) == bulkID {
			t.Fatal("bulk discount must not apply below 10,000")
		}
	}

	// ---- product discount ----
	r = e.call("POST", "/discounts", admin, map[string]any{"name": "Seat promo " + e.suffix, "discount_type": "fixed",
		"value": "500", "scope": "product", "product_ids": []int64{productID}, "min_quantity": 2})
	e.expect(r, 201, "product discount")
	promoID := id(r)
	if r.data()["label"] != "₹500.00 off" || len(r.data()["products"].([]any)) != 1 {
		t.Fatalf("unexpected product discount: %v", r.data())
	}
	r = e.call("POST", "/pricing/preview", admin, map[string]any{"lines": []map[string]any{{"product_id": productID, "quantity": 2}}})
	line := r.data()["lines"].([]any)[0].(map[string]any)
	if line["product_discount"] != "500.00" || line["product_discount_name"] == nil {
		t.Fatalf("product discount should apply to 2 seats: %v", line)
	}

	// ---- usage limit: once used up it stops applying ----
	if err := e.db.Exec("UPDATE discounts SET usage_count = 1 WHERE id = ?", bulkID).Error; err != nil {
		t.Fatal(err)
	}
	r = e.call("GET", fmt.Sprintf("/discounts/%d", bulkID), admin, nil)
	if r.data()["state"] != "used_up" || r.data()["remaining_uses"].(float64) != 0 {
		t.Fatalf("should be used up: %v", r.data())
	}
	r = e.call("POST", "/pricing/preview", admin, acme)
	for _, a := range r.data()["discounts"].([]any) {
		if int64(a.(map[string]any)["discount_id"].(float64)) == bulkID {
			t.Fatal("a used-up discount must stop applying")
		}
	}
	e.expect(e.call("PUT", fmt.Sprintf("/discounts/%d", bulkID), admin, map[string]any{"max_usage": 0}), 200, "remove the limit")
	e.expect(e.call("PUT", fmt.Sprintf("/discounts/%d", promoID), admin, map[string]any{"max_usage": 1, "ends_on": today}), 200, "limit + end today")
	if err := e.db.Exec("UPDATE discounts SET usage_count = 1 WHERE id = ?", promoID).Error; err != nil {
		t.Fatal(err)
	}
	e.expect(e.call("PUT", fmt.Sprintf("/discounts/%d", promoID), admin, map[string]any{"max_usage": 0}), 200, "clear limit")
	e.db.Exec("UPDATE discounts SET usage_count = 2 WHERE id = ?", promoID)
	e.expect(e.call("PUT", fmt.Sprintf("/discounts/%d", promoID), admin, map[string]any{"max_usage": 1}), 400, "limit below uses so far")

	// ---- dates: scheduled / previewing a future date ----
	r = e.call("POST", "/discounts", admin, map[string]any{"name": "New year " + e.suffix, "discount_type": "percentage",
		"value": "50", "scope": "subscription", "starts_on": future})
	e.expect(r, 201, "scheduled discount")
	nyID := id(r)
	if r.data()["state"] != "scheduled" {
		t.Fatalf("should be scheduled: %v", r.data())
	}
	r = e.call("POST", "/pricing/preview", admin, map[string]any{"date": future,
		"lines": []map[string]any{{"product_id": productID, "quantity": 1}}})
	found := false
	for _, a := range r.data()["discounts"].([]any) {
		if int64(a.(map[string]any)["discount_id"].(float64)) == nyID {
			found = true
		}
	}
	if !found {
		t.Fatal("previewing the future date should apply the scheduled discount")
	}

	// ---- deactivate, filters, usages ----
	r = e.call("DELETE", fmt.Sprintf("/discounts/%d", nyID), admin, nil)
	e.expect(r, 200, "deactivate")
	if r.data()["state"] != "inactive" {
		t.Fatal("should be inactive")
	}
	r = e.call("GET", "/discounts?state=inactive&q="+e.suffix, admin, nil)
	e.expect(r, 200, "filter inactive")
	if len(r.Body["data"].([]any)) != 1 {
		t.Fatalf("expected 1 inactive: %v", len(r.Body["data"].([]any)))
	}
	e.expect(e.call("GET", "/discounts?state=bogus", admin, nil), 400, "bad state filter")
	r = e.call("GET", fmt.Sprintf("/discounts/%d/usages", bulkID), admin, nil)
	e.expect(r, 200, "usages")
	e.expect(e.call("GET", "/discounts/999999999", admin, nil), 404, "unknown discount")
	e.expect(e.call("POST", "/pricing/preview", admin, map[string]any{"lines": []map[string]any{{"product_id": 999999999}}}), 400, "preview unknown product")
}
