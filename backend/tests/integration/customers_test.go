package integration

import (
	"fmt"
	"testing"
)

// staffToken creates a staff member through the API and logs them in.
func (e *env) staffToken(admin string) string {
	e.t.Helper()
	email := e.email("staff")
	e.expect(e.call("POST", "/staff", admin, map[string]any{"full_name": "Staff Member", "email": email, "password": "Staff#Pass2026"}), 201, "create staff")
	return e.login(email, "Staff#Pass2026")
}

func id(r resp) int64 { return int64(r.data()["id"].(float64)) }

func TestCustomerCRUDAndValidation(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	email := e.email("acme")

	// Validation
	e.expect(e.call("POST", "/customers", admin, map[string]any{"email": email}), 400, "missing name")
	r := e.call("POST", "/customers", admin, map[string]any{"name": "Acme", "email": "not-an-email", "phone": "12ab", "gstin": "BAD"})
	e.expect(r, 400, "bad email/phone/gstin")
	for _, f := range []string{"email", "phone", "gstin"} {
		if r.errDetail(f) == "" {
			t.Fatalf("expected a %s error, got %v", f, r.Body)
		}
	}

	// Create without a login
	r = e.call("POST", "/customers", admin, map[string]any{
		"name": "Acme Pvt Ltd", "email": email, "phone": "+91 98765 43210",
		"company_name": "Acme Pvt Ltd", "billing_address": "12 MG Road, Bengaluru", "gstin": "29abcde1234f1z5"})
	e.expect(r, 201, "create customer")
	custID := id(r)
	if r.data()["has_login"] != false || r.data()["gstin"] != "29ABCDE1234F1Z5" {
		t.Fatalf("unexpected customer: %v", r.data())
	}
	e.expect(e.call("POST", "/customers", admin, map[string]any{"name": "Dup", "email": upper(email)}), 409, "duplicate email (caps)")

	// Detail with stats
	r = e.call("GET", fmt.Sprintf("/customers/%d", custID), admin, nil)
	e.expect(r, 200, "get customer")
	st := r.data()["stats"].(map[string]any)
	if st["amount_due"] != "0.00" || st["active_subscriptions"].(float64) != 0 {
		t.Fatalf("unexpected stats: %v", st)
	}
	e.expect(e.call("GET", "/customers/999999999", admin, nil), 404, "unknown customer")
	e.expect(e.call("GET", "/customers/abc", admin, nil), 400, "bad id")

	// List + search by company + filter
	r = e.call("GET", "/customers?q=acme+pvt&has_login=false&page_size=5", admin, nil)
	e.expect(r, 200, "list customers")
	found := false
	for _, c := range r.Body["data"].([]any) {
		if int64(c.(map[string]any)["id"].(float64)) == custID {
			found = true
		}
	}
	if !found {
		t.Fatal("search should find the customer by company name")
	}
	e.expect(e.call("GET", "/customers?has_login=maybe", admin, nil), 400, "bad filter")

	// Update: change fields, clear one, reject duplicate email
	other := e.email("other")
	e.expect(e.call("POST", "/customers", admin, map[string]any{"name": "Other Co", "email": other}), 201, "second customer")
	e.expect(e.call("PUT", fmt.Sprintf("/customers/%d", custID), admin, map[string]any{"email": other}), 409, "email of another customer")
	r = e.call("PUT", fmt.Sprintf("/customers/%d", custID), admin, map[string]any{"name": "Acme India", "company_name": ""})
	e.expect(r, 200, "update customer")
	if r.data()["name"] != "Acme India" || r.data()["company_name"] != nil {
		t.Fatalf("update not applied: %v", r.data())
	}
}

func TestCustomerPermissions(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	staff := e.staffToken(admin)

	// Staff: list, create, edit yes; archive no.
	r := e.call("POST", "/customers", staff, map[string]any{"name": "Shop by Staff", "email": e.email("shop")})
	e.expect(r, 201, "staff creates customer")
	custID := id(r)
	e.expect(e.call("GET", "/customers", staff, nil), 200, "staff lists")
	e.expect(e.call("PUT", fmt.Sprintf("/customers/%d", custID), staff, map[string]any{"phone": "9876543210"}), 200, "staff edits")
	e.expect(e.call("DELETE", fmt.Sprintf("/customers/%d", custID), staff, nil), 403, "staff archives")

	// Customers can't use /customers at all.
	custEmail := e.email("portal")
	e.register(custEmail, "Brightloop#2026")
	cust := e.login(custEmail, "Brightloop#2026")
	e.expect(e.call("GET", "/customers", cust, nil), 403, "customer lists customers")
	e.expect(e.call("GET", fmt.Sprintf("/customers/%d", custID), cust, nil), 403, "customer reads a customer")
	e.expect(e.call("GET", "/customers", "", nil), 401, "no token")
}

func TestSignupLinksToExistingCustomer(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	email := e.email("rahul")

	// Staff/admin create the customer first (e.g. after a sales call)…
	r := e.call("POST", "/customers", admin, map[string]any{"name": "Rahul Sharma", "email": email})
	e.expect(r, 201, "create customer")
	custID := id(r)

	// …then Rahul signs up himself with the same email (different case).
	e.expect(e.call("POST", "/auth/register", "", map[string]any{"full_name": "Rahul S", "email": upper(email),
		"password": "Brightloop#2026", "phone": "9876543210"}), 201, "register")
	token := e.login(email, "Brightloop#2026")
	me := e.call("GET", "/auth/me", token, nil)
	if int64(me.data()["customer_id"].(float64)) != custID {
		t.Fatalf("login should be linked to the existing customer %d, got %v", custID, me.data()["customer_id"])
	}

	r = e.call("GET", fmt.Sprintf("/customers/%d", custID), admin, nil)
	if r.data()["has_login"] != true || r.data()["phone"] != "9876543210" {
		t.Fatalf("customer should now have a login and the phone from sign-up: %v", r.data())
	}

	// Their email is now their login, so staff can't change it…
	e.expect(e.call("PUT", fmt.Sprintf("/customers/%d", custID), admin, map[string]any{"email": e.email("new")}), 400, "change linked email")
	// …but renaming updates the login's name too.
	e.expect(e.call("PUT", fmt.Sprintf("/customers/%d", custID), admin, map[string]any{"name": "Rahul Sharma"}), 200, "rename")
	if n := e.call("GET", "/auth/me", token, nil).data()["full_name"]; n != "Rahul Sharma" {
		t.Fatalf("login name not synced: %v", n)
	}
}

func TestCustomerArchive(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)
	email := e.email("archive")
	e.register(email, "Brightloop#2026")
	token := e.login(email, "Brightloop#2026")
	custID := int64(e.call("GET", "/auth/me", token, nil).data()["customer_id"].(float64))

	// Give them an active subscription directly in the database (the
	// subscriptions module isn't merged yet).
	var planID, subID int64
	if err := e.db.Raw(`INSERT INTO subscription_plans (name, billing_period) VALUES (?, 'monthly') RETURNING id`,
		"Archive test plan "+e.suffix).Scan(&planID).Error; err != nil {
		t.Fatal(err)
	}
	if err := e.db.Raw(`INSERT INTO subscriptions (quotation_number, customer_id, plan_id, status, start_date, billing_anchor_day)
		VALUES (?, ?, ?, 'active', CURRENT_DATE, 1) RETURNING id`, "QT/TEST/"+e.suffix, custID, planID).Scan(&subID).Error; err != nil {
		t.Fatal(err)
	}

	r := e.call("DELETE", fmt.Sprintf("/customers/%d", custID), admin, nil)
	e.expect(r, 422, "archive with an active subscription")
	if r.errDetail("live_subscriptions") != "1" {
		t.Fatalf("expected live_subscriptions=1, got %v", r.Body)
	}

	// Close it; now archiving works, hides the customer and blocks their login.
	e.db.Exec(`UPDATE subscriptions SET status = 'closed' WHERE id = ?`, subID)
	e.expect(e.call("DELETE", fmt.Sprintf("/customers/%d", custID), admin, nil), 200, "archive")
	e.expect(e.call("GET", fmt.Sprintf("/customers/%d", custID), admin, nil), 404, "archived customer")
	e.expect(e.call("GET", "/auth/me", token, nil), 401, "archived customer's token")
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": email, "password": "Brightloop#2026"}), 403, "archived customer's login")

	// The email can be used for a new customer afterwards.
	e.expect(e.call("POST", "/customers", admin, map[string]any{"name": "New Owner", "email": email}), 201, "reuse archived email")
}
