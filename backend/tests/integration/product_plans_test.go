package integration

import (
	"fmt"
	"testing"
)

func TestProductScopedPlansAcrossRoles(t *testing.T) {
	e := setup(t)
	email, pw := e.createAdmin()
	admin := e.login(email, pw)
	product, _, _ := e.catalog(admin)
	r := e.call("POST", "/plans", admin, map[string]any{"name": "Monthly tier " + e.suffix, "billing_period": "monthly", "products": []map[string]any{{"product_id": product, "price_override": "199"}}})
	e.expect(r, 201, "second tier for same product")
	r = e.call("POST", "/products", admin, map[string]any{"name": "Unrelated product " + e.suffix, "product_type": "service", "sales_price": "100"})
	e.expect(r, 201, "other product")
	other := id(r)
	e.expect(e.call("POST", "/plans", admin, map[string]any{"name": "Unrelated tier " + e.suffix, "billing_period": "weekly", "products": []map[string]any{{"product_id": other}}}), 201, "unrelated plan")
	for _, token := range []string{admin, e.staffToken(admin), e.customerToken()} {
		r := e.call("GET", fmt.Sprintf("/plans?product_id=%d", product), token, nil)
		e.expect(r, 200, "product plan catalogue")
		if len(r.Body["data"].([]any)) != 2 {
			t.Fatalf("only the two tiers for selected product should appear: %v", r.Body)
		}
		e.expect(e.call("GET", "/plans?product_id=bad", token, nil), 400, "bad product filter")
	}
}
