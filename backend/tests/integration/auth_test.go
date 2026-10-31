// Integration tests: real router + services + PostgreSQL, fake mailer.
//
// Run with a migrated database:
//
//	make be-itest                      (uses the docker-compose database)
//	TEST_DATABASE_URL=postgres://... go test ./tests/integration/...
//
// Skipped automatically when TEST_DATABASE_URL is not set.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/config"
	"github.com/brightloop/subscriptions/backend/internal/database"
	"github.com/brightloop/subscriptions/backend/internal/handler"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/router"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/hash"
	"github.com/brightloop/subscriptions/backend/pkg/jwtutil"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
)

// ---------- test harness ----------

type fakeMailer struct {
	mu   sync.Mutex
	sent []mailer.Message
}

func (f *fakeMailer) Send(_ context.Context, m mailer.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

var codeRe = regexp.MustCompile(`code \(for API testing\): ([0-9a-f]{64})`)

// waitForCode polls for an email to `to` (forgot-password sends in the background).
func (f *fakeMailer) waitForCode(t *testing.T, to string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		for i := len(f.sent) - 1; i >= 0; i-- {
			if f.sent[i].To == to {
				if m := codeRe.FindStringSubmatch(f.sent[i].Body); m != nil {
					f.mu.Unlock()
					return m[1]
				}
			}
		}
		f.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("no email with a code was sent to %s", to)
	return ""
}

func (f *fakeMailer) countTo(to string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, m := range f.sent {
		if m.To == to {
			n++
		}
	}
	return n
}

type env struct {
	t      *testing.T
	db     *gorm.DB
	engine *gin.Engine
	mail   *fakeMailer
	auth   *service.AuthService
	users  *repository.UserRepo
	hasher *hash.Hasher
	suffix string // makes emails unique per test run
}

func setup(t *testing.T) *env {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	gin.SetMode(gin.TestMode)

	db, err := database.Connect(dsn, false)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Kolkata")
	clk := clock.Real{Loc: loc}
	jwtMgr := jwtutil.New("test-secret-test-secret-test-secret-123", time.Hour)
	hasher := hash.New(4) // fast bcrypt for tests
	mail := &fakeMailer{}
	users := repository.NewUserRepo()

	authSvc := service.NewAuthService(service.AuthDeps{
		DB: db, Users: users, Customers: repository.NewCustomerRepo(), ResetTokens: repository.NewResetTokenRepo(),
		Hasher: hasher, JWT: jwtMgr, Clock: clk, Mailer: mail,
		FrontendURL: "http://localhost:5000", ResetTTL: 30 * time.Minute, ShowCodesInEmail: true,
	})
	staffSvc := service.NewStaffService(db, users, hasher, authSvc, 48*time.Hour)
	customerSvc := service.NewCustomerService(db, repository.NewCustomerRepo(), users, clk)
	taxSvc := service.NewTaxService(db, repository.NewTaxRepo())

	engine := router.New(router.Deps{
		Config:   config.Config{AppEnv: "test"},
		DB:       db,
		JWT:      jwtMgr,
		Sessions: authSvc,
		Auth:     handler.NewAuthHandler(authSvc),
		Staff:    handler.NewStaffHandler(staffSvc),

		Customers: handler.NewCustomerHandler(customerSvc),
		Taxes:     handler.NewTaxHandler(taxSvc),
	})
	return &env{t: t, db: db, engine: engine, mail: mail, auth: authSvc, users: users, hasher: hasher,
		suffix: fmt.Sprintf("%d", time.Now().UnixNano())}
}

func (e *env) email(name string) string { return fmt.Sprintf("%s+%s@test.brightloop", name, e.suffix) }

type resp struct {
	Status int
	Body   map[string]any
}

func (r resp) data() map[string]any { m, _ := r.Body["data"].(map[string]any); return m }

func (r resp) errCode() string {
	if e, ok := r.Body["error"].(map[string]any); ok {
		return fmt.Sprint(e["code"])
	}
	return ""
}

func (r resp) errDetail(field string) string {
	if e, ok := r.Body["error"].(map[string]any); ok {
		if d, ok := e["details"].(map[string]any); ok {
			return fmt.Sprint(d[field])
		}
	}
	return ""
}

func (e *env) call(method, path, token string, body any) resp {
	e.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, "/api/v1"+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	e.engine.ServeHTTP(w, req)
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return resp{Status: w.Code, Body: m}
}

func (e *env) expect(r resp, status int, what string) {
	e.t.Helper()
	if r.Status != status {
		e.t.Fatalf("%s: got HTTP %d, want %d. body=%v", what, r.Status, status, r.Body)
	}
}

func (e *env) login(email, password string) string {
	e.t.Helper()
	r := e.call("POST", "/auth/login", "", map[string]any{"email": email, "password": password})
	e.expect(r, 200, "login "+email)
	return r.data()["access_token"].(string)
}

func (e *env) register(email, password string) {
	e.t.Helper()
	r := e.call("POST", "/auth/register", "", map[string]any{"full_name": "Test Customer", "email": email, "password": password})
	e.expect(r, 201, "register "+email)
}

// createAdmin inserts an admin directly (only the seed creates admins).
func (e *env) createAdmin() (email, password string) {
	e.t.Helper()
	email, password = e.email("admin"), "Admin#Pass123"
	h, _ := e.hasher.Hash(password)
	db, _ := database.Connect(os.Getenv("TEST_DATABASE_URL"), false)
	if err := e.users.Create(db, &models.User{Email: email, PasswordHash: h, FullName: "Test Admin",
		Role: models.RoleAdmin, IsActive: true}); err != nil {
		e.t.Fatal(err)
	}
	return email, password
}

// ---------- tests ----------

func TestRegisterAndLogin(t *testing.T) {
	e := setup(t)
	email := e.email("rahul")

	r := e.call("POST", "/auth/register", "", map[string]any{"full_name": "Rahul", "email": email, "password": "short"})
	e.expect(r, 400, "weak password")
	if r.errDetail("password") == "" {
		t.Fatal("expected a password field error")
	}

	e.register(email, "Brightloop#2026")

	r = e.call("POST", "/auth/register", "", map[string]any{"full_name": "Rahul", "email": "  " + upper(email), "password": "Brightloop#2026"})
	e.expect(r, 409, "duplicate email (different case)")

	wrongPw := e.call("POST", "/auth/login", "", map[string]any{"email": email, "password": "Wrong#Pass123"})
	unknown := e.call("POST", "/auth/login", "", map[string]any{"email": e.email("nobody"), "password": "Wrong#Pass123"})
	e.expect(wrongPw, 401, "wrong password")
	e.expect(unknown, 401, "unknown email")
	if fmt.Sprint(wrongPw.Body) != fmt.Sprint(unknown.Body) {
		t.Fatal("wrong-password and unknown-email responses must be identical")
	}

	token := e.login(email, "Brightloop#2026")
	me := e.call("GET", "/auth/me", token, nil)
	e.expect(me, 200, "me")
	if me.data()["role"] != "customer" || me.data()["customer_id"] == nil {
		t.Fatalf("expected a customer with customer_id, got %v", me.data())
	}
}

func TestRBAC(t *testing.T) {
	e := setup(t)
	e.expect(e.call("GET", "/staff", "", nil), 401, "no token")
	e.expect(e.call("GET", "/staff", "not-a-jwt", nil), 401, "garbage token")

	email := e.email("cust")
	e.register(email, "Brightloop#2026")
	cust := e.login(email, "Brightloop#2026")
	e.expect(e.call("GET", "/staff", cust, nil), 403, "customer listing staff")
	e.expect(e.call("POST", "/staff", cust, map[string]any{"full_name": "X", "email": e.email("x")}), 403, "customer creating staff")
}

func TestForgotAndResetPassword(t *testing.T) {
	e := setup(t)
	email := e.email("forgot")
	e.register(email, "Brightloop#2026")
	oldToken := e.login(email, "Brightloop#2026")

	// Unknown email: same 200 and same message, but no email is sent.
	unknown := e.email("ghost")
	r1 := e.call("POST", "/auth/forgot-password", "", map[string]any{"email": unknown})
	r2 := e.call("POST", "/auth/forgot-password", "", map[string]any{"email": email})
	e.expect(r1, 200, "forgot unknown")
	e.expect(r2, 200, "forgot known")
	if fmt.Sprint(r1.Body) != fmt.Sprint(r2.Body) {
		t.Fatal("forgot-password must answer identically for known and unknown emails")
	}
	firstCode := e.mail.waitForCode(t, email)

	// Asking again invalidates the first link.
	e.call("POST", "/auth/forgot-password", "", map[string]any{"email": email})
	deadline := time.Now().Add(3 * time.Second)
	for e.mail.countTo(email) < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	code := e.mail.waitForCode(t, email)
	if code == firstCode {
		t.Fatal("expected a new code")
	}
	if e.mail.countTo(unknown) != 0 {
		t.Fatal("no email should go to an unregistered address")
	}
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"token": firstCode, "password": "NewPass#2026"}), 400, "superseded link")

	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"token": code, "password": "weak"}), 400, "weak new password")
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"token": code, "password": "NewPass#2026"}), 200, "reset")

	// Link is single-use; old sessions are revoked; new password works.
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"token": code, "password": "Other#Pass2026"}), 400, "reused link")
	e.expect(e.call("GET", "/auth/me", oldToken, nil), 401, "token from before the reset")
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": email, "password": "Brightloop#2026"}), 401, "old password")
	newToken := e.login(email, "NewPass#2026")
	e.expect(e.call("GET", "/auth/me", newToken, nil), 200, "token after the reset")
}

func TestChangePassword(t *testing.T) {
	e := setup(t)
	email := e.email("change")
	e.register(email, "Brightloop#2026")
	token := e.login(email, "Brightloop#2026")

	r := e.call("POST", "/auth/change-password", token, map[string]any{"current_password": "Wrong#123456", "new_password": "Changed#2026"})
	e.expect(r, 400, "wrong current password")
	if r.errDetail("current_password") == "" {
		t.Fatal("expected current_password field error")
	}
	e.expect(e.call("POST", "/auth/change-password", token, map[string]any{"current_password": "Brightloop#2026", "new_password": "Brightloop#2026"}), 400, "same password")

	r = e.call("POST", "/auth/change-password", token, map[string]any{"current_password": "Brightloop#2026", "new_password": "Changed#2026"})
	e.expect(r, 200, "change password")
	fresh := r.data()["access_token"].(string)

	e.expect(e.call("GET", "/auth/me", token, nil), 401, "old token after change")
	e.expect(e.call("GET", "/auth/me", fresh, nil), 200, "fresh token after change")
	e.login(email, "Changed#2026")
}

func TestUpdateProfile(t *testing.T) {
	e := setup(t)
	email := e.email("profile")
	e.register(email, "Brightloop#2026")
	token := e.login(email, "Brightloop#2026")

	e.expect(e.call("PUT", "/auth/me", token, map[string]any{"gstin": "NOT-A-GSTIN"}), 400, "bad GSTIN")
	r := e.call("PUT", "/auth/me", token, map[string]any{"full_name": "Rahul Sharma", "company_name": "Acme Pvt Ltd", "gstin": "29abcde1234f1z5"})
	e.expect(r, 200, "update profile")
	p := r.data()["profile"].(map[string]any)
	if r.data()["full_name"] != "Rahul Sharma" || p["gstin"] != "29ABCDE1234F1Z5" || p["company_name"] != "Acme Pvt Ltd" {
		t.Fatalf("profile not updated: %v", r.data())
	}
}

func TestStaffManagement(t *testing.T) {
	e := setup(t)
	adminEmail, adminPw := e.createAdmin()
	admin := e.login(adminEmail, adminPw)

	// Admin creates staff with a password.
	staffEmail := e.email("priya")
	r := e.call("POST", "/staff", admin, map[string]any{"full_name": "Priya Nair", "email": staffEmail, "password": "Staff#Pass2026"})
	e.expect(r, 201, "create staff")
	staffID := int64(r.data()["staff"].(map[string]any)["id"].(float64))
	e.expect(e.call("POST", "/staff", admin, map[string]any{"full_name": "Dup", "email": staffEmail}), 409, "duplicate staff email")

	// Staff can log in, but cannot manage staff (admin-only).
	staff := e.login(staffEmail, "Staff#Pass2026")
	e.expect(e.call("POST", "/staff", staff, map[string]any{"full_name": "X", "email": e.email("x")}), 403, "staff creating staff")
	e.expect(e.call("GET", "/staff", staff, nil), 403, "staff listing staff")
	e.expect(e.call("PUT", "/auth/me", staff, map[string]any{"company_name": "Nope"}), 400, "staff setting customer fields")

	// Admin creates staff WITHOUT a password → invite email → staff sets password.
	invEmail := e.email("invite")
	r = e.call("POST", "/staff", admin, map[string]any{"full_name": "Invited Staff", "email": invEmail})
	e.expect(r, 201, "create invited staff")
	if r.data()["invite_sent"] != true {
		t.Fatal("expected invite_sent=true")
	}
	code := e.mail.waitForCode(t, invEmail)
	e.expect(e.call("POST", "/auth/reset-password", "", map[string]any{"token": code, "password": "Invited#2026"}), 200, "set password from invite")
	e.login(invEmail, "Invited#2026")

	// List, search, get, update.
	r = e.call("GET", "/staff?q=priya+nair&page_size=5", admin, nil)
	e.expect(r, 200, "list staff")
	if n := len(r.Body["data"].([]any)); n < 1 {
		t.Fatal("search should find Priya")
	}
	e.expect(e.call("GET", fmt.Sprintf("/staff/%d", staffID), admin, nil), 200, "get staff")
	e.expect(e.call("GET", "/staff/abc", admin, nil), 400, "bad id")
	r = e.call("PUT", fmt.Sprintf("/staff/%d", staffID), admin, map[string]any{"full_name": "Priya N."})
	e.expect(r, 200, "update staff")
	if r.data()["full_name"] != "Priya N." {
		t.Fatal("name not updated")
	}

	// /staff/:id must not reach customers or admins.
	custEmail := e.email("notstaff")
	e.register(custEmail, "Brightloop#2026")
	cust := e.call("GET", "/auth/me", e.login(custEmail, "Brightloop#2026"), nil)
	custUserID := int64(cust.data()["id"].(float64))
	e.expect(e.call("GET", fmt.Sprintf("/staff/%d", custUserID), admin, nil), 404, "customer via /staff/:id")

	// Deactivation is immediate: existing token rejected, login refused.
	e.expect(e.call("PATCH", fmt.Sprintf("/staff/%d/status", staffID), admin, map[string]any{}), 400, "missing is_active")
	e.expect(e.call("PATCH", fmt.Sprintf("/staff/%d/status", staffID), admin, map[string]any{"is_active": false}), 200, "deactivate")
	e.expect(e.call("GET", "/auth/me", staff, nil), 401, "deactivated staff's existing token")
	e.expect(e.call("POST", "/auth/login", "", map[string]any{"email": staffEmail, "password": "Staff#Pass2026"}), 403, "deactivated login")

	// No reset email for deactivated accounts.
	before := e.mail.countTo(staffEmail)
	e.expect(e.call("POST", "/auth/forgot-password", "", map[string]any{"email": staffEmail}), 200, "forgot (deactivated)")
	time.Sleep(200 * time.Millisecond)
	if e.mail.countTo(staffEmail) != before {
		t.Fatal("deactivated accounts must not receive reset emails")
	}

	e.expect(e.call("PATCH", fmt.Sprintf("/staff/%d/status", staffID), admin, map[string]any{"is_active": true}), 200, "reactivate")
	e.login(staffEmail, "Staff#Pass2026")
}

func upper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}
