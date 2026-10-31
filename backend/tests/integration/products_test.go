package integration

import (
	"fmt"
	"testing"
)

// customerToken registers a new customer and logs them in.
func (e *env) customerToken() string {
	e.t.Helper()
	email := e.email("cust")
	e.register(email, "Brightloop#2026")
	return e.login(email, "Brightloop#2026")
}

// newTax creates an active tax through the API and returns its id.
func (e *env) newTax(admin, rate string) int64 {
	e.t.Helper()
	r := e.call("POST", "/taxes", admin, map[string]any{"name": "GST " + rate + " " + e.suffix + fmt.Sprint(len(rate)), "rate": rate})
	e.expect(r, 201, "create tax")
	return id(r)
}

func variantByName(t *testing.T, r resp, name string) map[string]any {
	t.Helper()
	for _, v := range r.data()["variants"].([]any) {
		if m := v.(map[string]any); m["name"] == name {
			return m
		}
	}
	t.Fatalf("variant %q not found in %v", name, r.data()["variants"])
	return nil
}

func TestProducts(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)
	cust := e.customerToken()
	taxID := e.newTax(admin, "18")
	name := "Software Subscription " + e.suffix

	// ---- validation ----
	r := e.call("POST", "/products", admin, map[string]any{"name": name, "product_type": "software", "sales_price": "9.999"})
	e.expect(r, 400, "3-decimal price")
	if r.errDetail("sales_price") == "" {
		t.Fatalf("expected sales_price error: %v", r.Body)
	}
	e.expect(e.call("POST", "/products", admin, map[string]any{"name": name, "product_type": "hardware", "sales_price": "1"}), 400, "bad product_type")
	e.expect(e.call("POST", "/products", admin, map[string]any{"name": name, "product_type": "software", "sales_price": "1", "tax_id": 999999999}), 400, "unknown tax")
	r = e.call("POST", "/products", admin, map[string]any{"name": name, "product_type": "software", "sales_price": "999",
		"variants": []map[string]any{{"attribute": "Brand", "name": "Odoo"}, {"attribute": "brand", "name": "ODOO"}}})
	e.expect(r, 400, "duplicate variants (ignoring case)")

	// ---- create with variants: "choosing the Odoo brand adds 560" ----
	r = e.call("POST", "/products", admin, map[string]any{
		"name": name, "product_type": "software", "sales_price": "999", "cost_price": "300",
		"description": "Cloud ERP seat", "tax_id": taxID,
		"variants": []map[string]any{
			{"attribute": "Brand", "name": "Brand A", "extra_price": "0"},
			{"attribute": "Brand", "name": "Odoo Brand", "extra_price": "560", "sku": "ODOO-" + e.suffix},
		}})
	e.expect(r, 201, "create product")
	productID := id(r)
	if v := variantByName(t, r, "Odoo Brand"); v["price"] != "1559.00" || v["extra_price"] != "560.00" {
		t.Fatalf("Odoo Brand should cost 1559.00: %v", v)
	}
	if variantByName(t, r, "Brand A")["price"] != "999.00" {
		t.Fatal("Brand A should cost 999.00")
	}
	if r.data()["cost_price"] != "300.00" || r.data()["margin"] != "699.00" {
		t.Fatalf("admin should see cost and margin: %v", r.data())
	}
	if tax := r.data()["tax"].(map[string]any); tax["rate"] != "18.00" {
		t.Fatalf("tax summary wrong: %v", tax)
	}
	e.expect(e.call("POST", "/products", admin, map[string]any{"name": upper(name), "product_type": "service", "sales_price": "1"}), 409, "duplicate active name")

	// Tax now shows it's in use
	if n := e.call("GET", fmt.Sprintf("/taxes/%d", taxID), admin, nil).data()["products_using"]; n.(float64) != 1 {
		t.Fatalf("tax products_using should be 1, got %v", n)
	}

	// ---- permissions ----
	r = e.call("GET", fmt.Sprintf("/products/%d", productID), staff, nil)
	e.expect(r, 200, "staff reads product")
	if r.data()["cost_price"] != nil || r.data()["margin"] != nil {
		t.Fatalf("staff must not see cost/margin: %v", r.data())
	}
	e.expect(e.call("POST", "/products", staff, map[string]any{"name": "X", "product_type": "addon", "sales_price": "1"}), 403, "staff creates")
	e.expect(e.call("PUT", fmt.Sprintf("/products/%d", productID), staff, map[string]any{"sales_price": "1"}), 403, "staff edits")
	e.expect(e.call("GET", "/products", cust, nil), 403, "customer lists products")

	// ---- list + filters ----
	r = e.call("GET", fmt.Sprintf("/products?q=%s&product_type=software&tax_id=%d", e.suffix, taxID), admin, nil)
	e.expect(r, 200, "filtered list")
	if len(r.Body["data"].([]any)) != 1 {
		t.Fatalf("expected exactly our product: %v", r.Body["data"])
	}
	e.expect(e.call("GET", "/products?status=bogus", admin, nil), 400, "bad status filter")

	// ---- variants ----
	vpath := fmt.Sprintf("/products/%d/variants", productID)
	e.expect(e.call("POST", vpath, admin, map[string]any{"attribute": "BRAND", "name": "odoo brand"}), 409, "duplicate variant (ignoring case)")
	e.expect(e.call("POST", vpath, admin, map[string]any{"attribute": "Brand", "name": "Other", "sku": "ODOO-" + e.suffix}), 409, "duplicate SKU")
	e.expect(e.call("POST", vpath, admin, map[string]any{"attribute": "Brand", "name": "Too cheap", "extra_price": "-1000"}), 400, "negative variant price")
	r = e.call("POST", vpath, admin, map[string]any{"attribute": "Brand", "name": "Lite", "extra_price": "-200"})
	e.expect(r, 201, "cheaper variant")
	lite := variantByName(t, r, "Lite")
	if lite["price"] != "799.00" {
		t.Fatalf("Lite should cost 799.00: %v", lite)
	}
	liteID := int64(lite["id"].(float64))
	e.expect(e.call("PUT", fmt.Sprintf("%s/%d", vpath, liteID), admin, map[string]any{"extra_price": "-100"}), 200, "edit variant")
	e.expect(e.call("PUT", fmt.Sprintf("/products/%d", productID), admin, map[string]any{"sales_price": "50"}), 400, "price that makes a variant negative")

	// Sales price change flows through to variant prices
	r = e.call("PUT", fmt.Sprintf("/products/%d", productID), admin, map[string]any{"sales_price": "1000"})
	e.expect(r, 200, "change sales price")
	if variantByName(t, r, "Odoo Brand")["price"] != "1560.00" {
		t.Fatal("variant price should follow the sales price")
	}

	r = e.call("DELETE", fmt.Sprintf("%s/%d", vpath, liteID), admin, nil)
	e.expect(r, 200, "deactivate variant")
	if variantByName(t, r, "Lite")["is_active"] != false {
		t.Fatal("variant should be inactive")
	}
	e.expect(e.call("PUT", fmt.Sprintf("%s/999999999", vpath), admin, map[string]any{"name": "x"}), 404, "unknown variant")
	e.expect(e.call("PUT", fmt.Sprintf("%s/abc", vpath), admin, map[string]any{"name": "x"}), 400, "bad variant id")

	// ---- archive ----
	r = e.call("DELETE", fmt.Sprintf("/products/%d", productID), admin, nil)
	e.expect(r, 200, "archive product")
	if r.data()["status"] != "archived" {
		t.Fatal("product should be archived")
	}
	r = e.call("GET", fmt.Sprintf("/products?q=%s", e.suffix), admin, nil)
	if len(r.Body["data"].([]any)) != 0 {
		t.Fatal("archived products are hidden by default")
	}
	r = e.call("GET", fmt.Sprintf("/products?q=%s&status=archived", e.suffix), admin, nil)
	if len(r.Body["data"].([]any)) != 1 {
		t.Fatal("status=archived should show it")
	}
	// The name is free again for a new active product, so the old one can't be restored.
	e.expect(e.call("POST", "/products", admin, map[string]any{"name": name, "product_type": "software", "sales_price": "1"}), 201, "reuse name after archive")
	e.expect(e.call("PUT", fmt.Sprintf("/products/%d", productID), admin, map[string]any{"status": "active"}), 409, "restore with a taken name")

	// Removing the default tax
	r = e.call("PUT", fmt.Sprintf("/products/%d", productID), admin, map[string]any{"tax_id": 0})
	e.expect(r, 200, "remove tax")
	if r.data()["tax"] != nil {
		t.Fatal("tax should be removed")
	}
	e.expect(e.call("GET", "/products/999999999", admin, nil), 404, "unknown product")
}
