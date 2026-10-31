package integration

import (
	"fmt"
	"testing"
)

func TestTaxes(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)
	name := "Test GST " + e.suffix

	// Validation
	for _, rate := range []string{"abc", "-1", "101", "18.555"} {
		r := e.call("POST", "/taxes", admin, map[string]any{"name": name, "rate": rate})
		e.expect(r, 400, "bad rate "+rate)
		if r.errDetail("rate") == "" {
			t.Fatalf("expected rate error for %q: %v", rate, r.Body)
		}
	}
	e.expect(e.call("POST", "/taxes", admin, map[string]any{"rate": "18"}), 400, "missing name")

	// Create: rate normalised to 2 decimals
	r := e.call("POST", "/taxes", admin, map[string]any{"name": name, "rate": "18"})
	e.expect(r, 201, "create tax")
	taxID := id(r)
	if r.data()["rate"] != "18.00" || r.data()["is_active"] != true || r.data()["products_using"].(float64) != 0 {
		t.Fatalf("unexpected tax: %v", r.data())
	}
	e.expect(e.call("POST", "/taxes", admin, map[string]any{"name": upper(name), "rate": "5"}), 409, "duplicate name (caps)")

	// Permissions: staff read, can't write; customers can't read
	e.expect(e.call("GET", "/taxes", staff, nil), 200, "staff lists taxes")
	e.expect(e.call("GET", fmt.Sprintf("/taxes/%d", taxID), staff, nil), 200, "staff reads a tax")
	e.expect(e.call("POST", "/taxes", staff, map[string]any{"name": "Staff tax", "rate": "1"}), 403, "staff creates tax")
	e.expect(e.call("PUT", fmt.Sprintf("/taxes/%d", taxID), staff, map[string]any{"rate": "1"}), 403, "staff edits tax")
	e.expect(e.call("DELETE", fmt.Sprintf("/taxes/%d", taxID), staff, nil), 403, "staff deactivates tax")
	custEmail := e.email("taxcust")
	e.register(custEmail, "Brightloop#2026")
	e.expect(e.call("GET", "/taxes", e.login(custEmail, "Brightloop#2026"), nil), 403, "customer lists taxes")

	// Update rate + name
	r = e.call("PUT", fmt.Sprintf("/taxes/%d", taxID), admin, map[string]any{"rate": "12.5", "name": name + " (new)"})
	e.expect(r, 200, "update tax")
	if r.data()["rate"] != "12.50" {
		t.Fatalf("rate not updated: %v", r.data())
	}

	// A product uses it as its default tax → can't deactivate
	var productID int64
	if err := e.db.Raw(`INSERT INTO products (name, product_type, sales_price, tax_id) VALUES (?, 'software', 999, ?) RETURNING id`,
		"Tax test product "+e.suffix, taxID).Scan(&productID).Error; err != nil {
		t.Fatal(err)
	}
	r = e.call("GET", fmt.Sprintf("/taxes/%d", taxID), admin, nil)
	if r.data()["products_using"].(float64) != 1 {
		t.Fatalf("expected products_using=1: %v", r.data())
	}
	r = e.call("DELETE", fmt.Sprintf("/taxes/%d", taxID), admin, nil)
	e.expect(r, 422, "deactivate tax used by an active product")
	if r.errDetail("rule") != "TAX_IN_USE" || r.errDetail("products_using") != "1" {
		t.Fatalf("expected TAX_IN_USE: %v", r.Body)
	}
	e.expect(e.call("PUT", fmt.Sprintf("/taxes/%d", taxID), admin, map[string]any{"is_active": false}), 422, "deactivate via PUT too")

	// Archive the product → now it can be deactivated, then reactivated
	e.db.Exec(`UPDATE products SET status = 'archived' WHERE id = ?`, productID)
	r = e.call("DELETE", fmt.Sprintf("/taxes/%d", taxID), admin, nil)
	e.expect(r, 200, "deactivate unused tax")
	if r.data()["is_active"] != false {
		t.Fatal("tax should be inactive")
	}
	r = e.call("GET", "/taxes?is_active=false&q="+e.suffix, admin, nil)
	e.expect(r, 200, "filter inactive")
	if len(r.Body["data"].([]any)) != 1 {
		t.Fatalf("expected the inactive tax in the filtered list: %v", r.Body["data"])
	}
	r = e.call("PUT", fmt.Sprintf("/taxes/%d", taxID), admin, map[string]any{"is_active": true})
	e.expect(r, 200, "reactivate")
	if r.data()["is_active"] != true {
		t.Fatal("tax should be active again")
	}

	e.expect(e.call("GET", "/taxes/999999999", admin, nil), 404, "unknown tax")
	e.expect(e.call("GET", "/taxes?is_active=maybe", admin, nil), 400, "bad filter")
}
