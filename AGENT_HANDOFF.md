# AGENT_HANDOFF.md

> **For any AI or developer picking this up:** read this whole file before touching code.
> It is updated after **every** completed step. The newest entry in the step log (section 7) is at the top.
> Last updated: **2026-10-31**, after **A7–A12 (quotations → subscriptions → invoices → payments → dashboard → billing run)**, backend + admin panel.

---

## 0. Where to resume (read first)

| | |
|---|---|
| **Current step** | **Product-specific plans, customer action emails and 7/3/1-day renewal reminders enabled.** Admin creates tiers within a selected product; staff/customers browse product groups. Customer portal includes subscriptions, invoices, payments, Plans, Profile and Dashboard. Remaining work is listed in section 8. |
| **Branch** | `ashlin_dev` (Ashlin's backend branch). `main` is the team integration branch. |
| **Uncommitted work** | Everything from A3 handlers through A6 plus the frontend wiring (see the step log for files). Also modified but **not by the backend/wiring agent**: `frontend/lib/core/theme/app_theme.dart`, `features/auth/presentation/{auth_card,login_screen}.dart`, `auth/presentation/widgets/{brand_logo,logo_video_loader}.dart` (someone else's auth-UI work; leave as is). Ashlin commits by hand; see R1 and `git status`. |
| **Is it running?** | `make dev` = backend (hot reload) + Flutter. Health: `curl localhost:8080/health` |
| **Test before you claim done** | Backend: `make be-test` + `make be-itest`. Frontend: `cd frontend && flutter analyze && flutter test && flutter build web`. |

---

## 1. Rules from the user (must follow)

| # | Rule |
|---|------|
| **R1** | **Never run `git add`, `git commit`, `git push`, merge, rebase or switch branches.** Ashlin does all Git by hand. Read-only Git (`status`, `diff`, `log`, `fetch`) is fine. At the end of a step, list the changed files. |
| **R2** | Always git-ignore `.env` and every secret (keys, credentials, app passwords). Only `*.example` files are tracked. `.env` currently holds a **real Gmail app password**; never print it, copy it or commit it. |
| **R3** | Guides, test collections, PDFs and diagrams for Ashlin go in `~/Documents/...`, **not** in the project, unless asked. |
| **R4** | Update **this file** after every completed step (status board, step log, next step). |
| **R5** | Explain decisions in plain, beginner-friendly language. Ashlin owns the **backend**; teammates own other parts. |
| **R6** | Don't touch teammates' code or branches (Abrar: Flutter UI; Bineesh: subscriptions; Gayatri: billing/payments) without asking. |

---

## 2. Project snapshot

- **What:** Subscription Management System for the company **Brightloop Software** (hackathon).
  **App name: Subly**, tagline *"Subscriptions. Billing. Simplified."* Brightloop is the company; Subly is the product users see.
  Constants: `backend/pkg/brand/brand.go`, `frontend/lib/core/widgets/brand_mark.dart`.
- **Official brief** (pasted by the user): sign-up with unique email and a password of more than 8 characters
  with upper, lower and special; reset link to the registered email; admin / staff (limited) / customer (own data
  via portal); only the admin adds staff; products with variants (e.g. "Odoo brand adds 560"); plans (daily,
  weekly, monthly, yearly, min quantity, availability dates, auto-close, pause/close/renew flags); quotation
  templates; subscriptions Draft→Quotation→Confirmed→Active→Closed; automatic invoices (no duplicates, no
  skips) with automatic tax; invoices Draft→Confirmed→Paid; admin-only discounts (fixed or %, minimums,
  dates, usage limit, product or subscription scope); payments by method, amount, date and note; dashboard
  (active subs, revenue, payments, overdue, filters), every page < 2 s with thousands of subscriptions.
- **Team's own spec additionally requires:** bcrypt, JWT, backend RBAC, Go/Gin/GORM, PostgreSQL, Flutter
  (Riverpod, GoRouter, Dio), Docker Compose, Swagger, migrations (not AutoMigrate).
- **Repo:** https://github.com/bineeshgladson-byte/Subscription-Management-System (public). Local: `~/hackathon`.
- **Team / ownership (A0 split, to be confirmed by the team):**
  - **Ashlin** (this machine, `ashlin_dev`): auth ✅, staff ✅, customers ✅, taxes ✅, products, plans, templates, discounts + pricing engine, dashboard/reports, admin settings
  - **Bineesh** (`products-subscriptions`): quotations → subscriptions, lifecycle (list/get/activate/pause/resume/close in progress)
  - **Gayatri** (`gayatri_billing`): billing engine, invoices, payments (record payment in progress)
  - **Abrar** (`Abrar-Anas`, merged): Flutter login/register UI, logo animation
- **Architecture plan** (full design, ER diagram, API list, billing design):
  https://claude.ai/artifact/GoBJt3Bxh6jgRx6KNjo2N3 · implementation plan PDF: `~/Documents/Subly-Backend-Plan.pdf`
- **Diagrams generated from the live DB:** `~/Documents/subly-diagrams/` (architecture, ERD, business flow,
  auth and lifecycles). Regenerate after migrations: `~/Documents/subly-diagrams/generator/regenerate.sh`

---

## 3. Environment and commands

| Thing | Value |
|---|---|
| Go | **Not installed locally.** All Go runs in Docker (`golang:1.26-alpine`, `GOTOOLCHAIN=local`). Makefile handles it. |
| Flutter | 3.47.9 / Dart 3.13.7 (upgraded 2026-10-31; Abrar's `video_player ^2.14.1` needs Dart ≥ 3.12) |
| Ports | API `8080` · Postgres `5433` (host) · Mailpit `8025` · Swagger `8090` · Flutter web `5000` · Prism mock `4010` |
| Admin login | `admin@brightloop.local` / `Admin#Brightloop1` (from `SEED_ADMIN_*` in `.env`, created on API start) |
| Email | `.env` currently sends **real email via Gmail SMTP** (ashlinbiju2006@gmail.com). Backup of the Mailpit `.env` was in a temp dir; to revert set `SMTP_HOST=mailpit SMTP_PORT=1025 SMTP_USERNAME= SMTP_PASSWORD=` and `docker compose up -d --force-recreate api-dev` |

```bash
make dev            # backend with Air hot reload (container api-dev) + Flutter web on :5000
make dev-backend    # backend only, logs in terminal      make logs-dev   # follow backend logs
make dev-stop       # stop the dev stack                  make up         # backend from built image (no hot reload)
make be-test        # Go unit tests (Docker)              make be-itest   # integration tests vs the dev DB
make be-fmt be-vet  # gofmt / go vet (Docker)             make test-email TO=x@y.com   # one test email via .env SMTP
make migrate        # apply migrations                    make migrate-down   # roll back the last one
make seed           # demo catalogue (idempotent)         make psql / make reset (DANGER: wipes the DB)
```

Format/vet/unit-test in one go (what the agent has been using):
```bash
docker run --rm --user $(id -u):$(id -g) -e GOCACHE=/tmp/gc -e GOTOOLCHAIN=local -e GOMODCACHE=/tmp/mod \
  -v gomodcache:/tmp/mod -v "$PWD/backend":/src -w /src golang:1.26-alpine \
  sh -c 'gofmt -w . && go vet ./... && go test ./...'
```

---

## 4. Architecture and conventions (copy these for every module)

```
backend/
  cmd/api/main.go            wiring: config → db → repos → services → handlers → router.Deps
  cmd/mailtest/              `make test-email`
  internal/config            env vars (see .env.example)
  internal/models            GORM structs = tables (enums.go mirrors Postgres ENUMs)
  internal/dto               JSON request/response structs (keep in sync with docs/api/openapi.yaml)
  internal/repository        all SQL; every method takes *gorm.DB (so it works inside a transaction)
  internal/service           business rules + transactions; no HTTP here
  internal/handler           bind → validate → ONE service call → response.*  (handler.go: bind(), pathID())
  internal/middleware        RequestID, Logger, CORS, Auth (JWT + CheckSession), RequireRoles
  internal/router            router.go + ONE routes_<module>.go per module (avoids merge conflicts)
  internal/validator         Password, NormalizeEmail, GSTIN, Phone
  internal/jobs              cron scheduler (billing job is a TODO stub)
  pkg/                       response, apperr-mapping, jwtutil, hash, mailer, pagination, clock, brand
  tests/integration          real HTTP + real Postgres; skipped if TEST_DATABASE_URL unset
database/migrations          golang-migrate SQL, timestamp names YYYYMMDDHHMMSS_name.{up,down}.sql
database/seeds               idempotent demo data (make seed)
docs/api/openapi.yaml        THE API contract (validated with openapi-spec-validator)
```

**Rules every module follows**
- **Response envelope:** `{"success":true,"data":…,"meta":{page,page_size,total}}` and errors
  `{"success":false,"error":{"code","message","details":{field:msg}}}`. Codes: `VALIDATION_ERROR` 400,
  `UNAUTHENTICATED` 401, `FORBIDDEN` 403, `NOT_FOUND` 404, `CONFLICT`/`INVALID_STATE` 409, `BUSINESS_RULE` 422
  (`details.rule` holds a specific code such as `TAX_IN_USE`), `NOT_IMPLEMENTED` 501.
- **Errors:** services return `apperr.*`; handlers call `response.Error(c, err)`. Unknown errors become a 500 and are logged, never leaked.
- **Money and rates** are `NUMERIC` in Postgres, `decimal.Decimal` in Go, and **strings in JSON** (`"999.00"`, `StringFixed(2)`). Never float.
- **Lists:** `pagination.Parse(c, defaultSort, whitelist...)`, with `?page&page_size(max 100)&sort=-col`. Search via `escapeLike`. Use repo `scope` funcs for Count + Find.
- **Partial updates:** PUT DTOs use pointer fields; `""` clears an optional field (`nullIfEmpty`).
- **Emails:** `validator.NormalizeEmail` in services (not Gin's `email` binding, which rejects leading spaces).
- **RBAC guards** (router.go): `adminOnly`, `adminOrStaff`, `customerOnly`, `anyLoggedInUser`. Unbuilt routes use `todo("…")`, which returns 501.
- **Customer self-service** goes under `/me/*` and takes the customer id only from `middleware.CurrentCustomerID(c)`. Return 404 (not 403) for other people's records.
- **Migrations:** never edit a merged migration; add a new timestamped pair; test `up → down 1 → up`. When an index changes, check the seeds' `ON CONFLICT` targets.
- **Definition of done per module:** DTO, repo, service, handler, routes, `main.go` + `tests/integration/auth_test.go setup()` wiring,
  `openapi.yaml`, integration tests (`make be-itest`), a Postman folder (generator in `~/Documents/brightloop-postman/generator/`),
  seed data if useful, then **update this file**.

---

## 5. Status board

| Module | Status | Endpoints | Notes |
|---|---|---|---|
| Scaffold, Docker, CI, docs | ✅ merged | `/health` | PRs #1–#3 |
| Auth | ✅ merged | register, login, me (GET/PUT), change-password, forgot/reset | bcrypt 12, JWT HS256 8 h, ms-precision `iat`, `CheckSession` per request |
| Staff (admin) | ✅ merged | `/staff` list/create/get/update, `PATCH /staff/:id/status` | invite email if no password |
| Email + branding | ✅ merged | `make test-email` | Gmail SMTP works; `EMAIL_SHOW_CODES` dev-only codes |
| `make dev` (Air in Docker) | ✅ merged | — | `backend/.air.toml`, `Dockerfile.dev`, service `api-dev` (profile dev) |
| **A1 Customers** | ✅ committed | `/customers` list/create/get(+stats)/update, DELETE = archive (admin) | sign-up links to an existing customer by email; archive blocked by open items |
| **A2 Taxes** | ✅ committed | `/taxes` list/get/create/update, DELETE = deactivate (admin) | rate string 0–100, 2 dp; `TAX_IN_USE` guard |
| Reference Flutter UI | ✅ built | full-viewport workspace shell | Styling kept; the sample-data plans/dashboard bodies were replaced by live-API screens |
| **A3 Products + variants** | ✅ built (partly committed) | `/products` list/get/create/update, DELETE = archive; `/products/:id/variants` add/update, DELETE = deactivate | variant price = sales + extra; staff see no cost/margin; `PRODUCT_IN_USE` / `VARIANT_IN_USE` guards; form close lifecycle fixed and regression tested |
| **A4 Plans** | ✅ built (partly committed) | `/plans` list/get (all roles), create/update (admin), `PATCH /plans/:id/status` | product-based browsing across roles; `product_id` filter; selected product prefilled on create; price computed from lines; customer availability/permissions preserved |
| Customer action emails + renewal reminders | ✅ enabled | automatic transactional queue + scheduler | customer only; detailed invoice emails and receipts; 7/3/1-day reminders; retries and deduplication; early renewal keeps current contract active |
| **A5 Quotation templates** | ✅ built | `/quotation-templates` list/get/create/update, DELETE = deactivate | lines default to the plan's products; `estimated_total` |
| **A6 Discounts + pricing engine** | ✅ built | `/discounts` CRUD (admin), `/discounts/:id/usages`, `POST /pricing/preview` (admin + staff) | pure `service.Price()`; `DiscountRepo.ReserveUse/ReleaseUse` ready for invoices (A9) |
| **Frontend wired to the API** | ✅ built | Dashboard, Customers, Staff, Products (+variants), Taxes, Plans, Templates, Discounts, Price calculator, Profile | quotations/subscriptions/invoices/payments/reports/settings stay placeholders until their APIs exist |
| **A7 Quotations · A8 Subscriptions** | ✅ built by Ashlin's agent (Bineesh's branch NOT merged; it would conflict in routes_subscriptions.go/main.go) | `/quotations` list/create/get/send/confirm/cancel; `/subscriptions` list/get/pause/resume/close/renew/bill-now | `service/sales_service.go` |
| **A9 Billing/invoices · A10 Payments** | ✅ | `/invoices` list/get/confirm/cancel/send, `POST /invoices/:id/payments` (Gayatri's files copied verbatim from `gayatri_billing`), `GET /payments` | billing engine in `sales_service.go` (`billNextPeriod`, `RunBilling`) |
| **A11 Dashboard · A12 Admin billing run** | ✅ | `GET /dashboard/summary?from&to` (money admin-only), `POST /admin/billing/run {as_of}` (as_of dev-only); scheduler calls `RunBilling` hourly + daily | |
| **Phase C Customer portal** | ✅ | `/me/subscriptions[/:id]`, `POST /me/subscriptions/:id/{pause,resume,close,renew}`, `/me/invoices[/:id]`, `/me/payments` | `service/portal.go`, `handler/portal_handler.go`, `routes_me.go`; frontend `features/portal/presentation/portal_screens.dart`; `TestCustomerPortal`; all customer sections styled consistently; immediate workspace navigation with brief account-scoped caching |
| Phase B Staff flow audit | ⏳ not formally audited | | staff use the same screens; RBAC enforced per route |

**Tests right now:** backend 19 integration tests pass (latest feature run in a separate migrated database with a fake mailer; no live customer/test mail),
Go unit tests and vet pass. Earlier coverage included `TestQuotationTemplates`,
`TestDiscountsAndPricing`, unit tests incl. pricing engine (Acme ₹16,556.58, Rahul ₹1,178.82, best-of, minimums, spread, cap).
Frontend: 45 widget/unit tests pass (`flutter test`; 1 opt-in live API test skipped; FakeApi uses API-shaped JSON),
plus opt-in `LIVE_API=http://localhost:8080/api/v1 flutter test test/live_api_test.dart` (real repos vs the live backend, passes).
Postman: 140 requests / 16 folders; folders 0,1,2,5,11–15 = 90 requests / 165 assertions all pass.

**Migrations applied:** `20261101000000_init_schema` (20 tables), `20261101000100_customers_unique_email`,
`20261101000200_taxes_unique_name_ci`, `20261101000300_products_unique_active_name`, `20261101000400_plans_unique_name_ci`,
`20261101000500_templates_unique_name_ci`, `20261101000600_discounts_code_ci`, `20261101000700_email_notifications`.

---

## 6. Key decisions (with reasons)

1. **Quotations are rows in `subscriptions`** (status draft/quotation). No copying on confirmation. (ADR-001, not yet confirmed by the team.)
2. **Roles and payment methods are Postgres ENUMs**, not tables.
3. **Passwords:** 9+ characters ("more than 8"), with upper, lower and special. Max 72 bytes (bcrypt limit).
4. **Forgot-password always returns 200 with the same message** and sends in the background (no account enumeration).
   Reset tokens are 32 random bytes, only the SHA-256 is stored, 30 min, single use; a new request invalidates the old token.
5. **Every request re-checks the DB** (`users.is_active`, `iat ≥ password_changed_at`), so deactivation and password resets take effect immediately.
6. **Staff can't see financial reports** (MRR/ARR/revenue) — to be confirmed by the team.
7. **Customers:** one live customer per email (partial unique index); a sign-up attaches to the existing customer
   record; a linked customer's email can't be changed by staff; archiving deactivates their login.
8. **Taxes are never hard-deleted;** invoice lines snapshot `tax_rate`. Can't deactivate while active products use the tax as their default.
10. **Product names are unique among ACTIVE products** (ignoring case). Archived products may keep old names. Decided by the agent; reversible by dropping `ux_products_name_active`.
11. **Plan price is never typed in:** it's Σ quantity × (price_override or sales_price + variant extra_price), stored in
    `subscription_plans.price` and refreshed in SQL (`repository.recalcSQL`) when lines, product prices or variant prices change.
12. **Staff see products without `cost_price`/`margin`**, and plans with `active_subscriptions`. Customers see plans only (no counts).
13. **Catalogue rows are never hard-deleted:** products archive, variants deactivate, plans archive. Existing subscriptions keep billing.
14. **Pricing engine order:** line subtotal → single best product discount per line → single best order-wide discount
    (spread proportionally, cents to largest lines) → tax per line rounded half-up → total. Minimums: product discounts use the
    line's qty/subtotal, order discounts use the whole order's qty/subtotal (before discounts). Eligibility (active, dates, uses
    left) is a DB query (`DiscountRepo.Eligible`). A7–A9 must price through `service.Price()` so quotation = subscription = invoice.
15. **Discount usage** is counted when an invoice is created: `DiscountRepo.ReserveUse(tx, id)` (atomic, refuses when used up) +
    a `discount_usages` row; cancelling an invoice calls `ReleaseUse`. Not called anywhere yet (needs A9).
16. **Frontend data pattern:** `features/<x>/data/<x>_api.dart` = model `fromJson` + repository (ApiClient) + Riverpod
    `FutureProvider.autoDispose(.family)` keyed by a query record; mutate via repository then `ref.invalidate(provider)`.
    Shared UI in `core/widgets/ui_kit.dart` (`PageFrame`, `SearchBox`, `FilterDropdown`, `Pager`, `ResourceTile`, `Pill`,
    `showApiForm` which maps API `details` to fields, `runAction`, `confirmAction`, `DateField`). `ProviderScope(retry: null)`.
9. **Billing correctness comes from the DB:** transaction + `FOR UPDATE SKIP LOCKED` + unique `(subscription_id, period_start)`;
   month-end clamping via `billing_anchor_day` (`service/billing_dates.go`, tested).

---

## 7. Step log (newest first)

### 2026-10-31 · Product plans + customer action emails + renewal reminders ✅ (uncommitted)
- User requested multiple plans per product across admin/staff/customer, action/invoice emails and automatic renewal reminders. User explicitly chose **7, 3 and 1 day before renewal** and **customer emails only**.
- Plans now browse by product for every role. Admin can select a product (including one with zero plans) and create different tiers with its product line prefilled. Staff view groups; customers see only eligible plans. Product detail and dashboard plan previews link directly to the product's plans. Product pickers load all pages; product edits invalidate cached plan prices.
- API: `GET /plans?product_id=...` validates the filter and retains customer availability rules. Existing bundle/override pricing and admin-only writes preserved; no catalogue schema redesign.
- New `email_notifications` outbox committed with business actions; immediate asynchronous delivery plus one-minute retry worker. Quotation creation/sending/cancellation/expiry, subscription confirmation/activation/pause/resume/closure, renewal requests, issued/cancelled/paid invoices and payment receipts covered. Composite customer/renewal actions suppress redundant internal creation emails. SMTP now has connection/read deadlines and cancellation support.
- Reminders use contract expiry when present, otherwise next recurring billing date, for renewable subscriptions. Unique subscription/date/lead-time keys prevent repeated cron notifications. Existing renewal requests suppress more reminders; unsent reminders invalidated by lifecycle changes are discarded. No admin/staff copies and no automatic renewal/charge.
- Renewal requests are atomic, visible as quotations in the portal and reject duplicates. The existing subscription stays active until the confirmed replacement starts. Same-day pause/resume reuses its existing billing period instead of attempting a duplicate invoice.
- Migration `20261101000700` tested up/down/up in an isolated `subly_feature_test` database, then applied to the running dev stack. Integration tests used that separate DB and a fake mailer; the temporary database was dropped after verification. Production customer records were not populated with test fixtures, and no test emails were sent to real recipients.
- Validation: Go unit tests + vet pass; **19 integration tests pass** including concurrent reminder deduplication, 7/3/1 schedule, delivery retry/restart, obsolete reminder suppression, advance renewal and product filters across roles. SMTP tests use a local fake server. Flutter analyze clean, **45 tests pass / 1 opt-in skipped**, release web build + Wasm dry run pass; existing Cupertino font warning remains. API/database health OK. OpenAPI YAML parses; product filter documented.
- Files changed for this step: `backend/cmd/api/main.go`; `backend/internal/{handler/plan_handler.go,jobs/scheduler.go,repository/plan_repo.go,service/{notifications,payment_service,plan_service,portal,sales_service}.go}`; `backend/pkg/mailer/{mailer,mailer_test}.go`; `backend/tests/integration/{auth_test,notifications_test,product_plans_test}.go`; migration `database/migrations/20261101000700_email_notifications.{up,down}.sql`; `docs/api/openapi.yaml`; `frontend/lib/core/router/app_router.dart`; `frontend/lib/features/{dashboard/presentation/customer_dashboard.dart,plans/{data/plans_api,presentation/plans_screen}.dart,products/{data/products_api,presentation/products_screen}.dart}`; `frontend/test/workspace_test.dart`; `README.md`, `database/README.md`, this handoff. Other existing changes preserved.
- Next: live customer demo of product-tier selection, quotation confirmation, invoice/receipt delivery and renewal with the existing SMTP settings. Backend must be running for scheduled jobs. SMTP has the normal at-least-once crash boundary (accepted email before delivery acknowledgement is persisted can be retried); this is documented in README.

### 2026-10-31 · Immediate workspace navigation + customer section polish ✅ (uncommitted)
- User requested removal of delayed page-switch animations and application of the dashboard style across customer sections.
- `core/router/app_router.dart`: shell and all workspace destinations use `NoTransitionPage` with route keys. Login/auth routes retain their existing behavior.
- New `core/network/navigation_cache.dart`: successful requests stay alive for two minutes after their last listener leaves. Account id/role changes invalidate them; errors are not retained. Applied to plan queries, customer lists/details and Profile. Dashboard now shares the Plans query and warms the first subscriptions/invoices/payments/Profile pages independently of dashboard loading. Refresh buttons and existing mutation invalidations fetch current records.
- New `core/widgets/customer_widgets.dart`: shared lavender intro panels, record cards, responsive facts, sections and empty states. Applied to customer Plans browsing, Subscriptions, Invoices, Payments and Profile. Subscription actions, invoice details, pagination, quotation requests, editing and password changes remain wired to existing APIs.
- Validation: analyzer clean; 44 tests passed / 1 opt-in live API skipped; release web build passed. Production-router test proves a single-frame page replacement, no transition pages, shared catalogue fetch, cached revisits and explicit refresh. Account-change test proves refetching. Desktop/phone/2× text and empty-state captures reviewed in `~/Documents/subly-ui-preview/customer-*.png`.
- Changed files: `core/{router/app_router.dart,network/navigation_cache.dart,widgets/customer_widgets.dart}`, `features/{dashboard/presentation/dashboard_screen.dart,plans/data/plans_api.dart,plans/presentation/plans_screen.dart,portal/presentation/portal_screens.dart,profile/data/profile_api.dart,profile/presentation/profile_screen.dart,sales/data/sales_api.dart}`, `test/workspace_test.dart` (all under `frontend/`), and this handoff. Existing unrelated changes preserved.
- Next: continue live browser demo checks, including customer navigation and lifecycle actions. Initial uncached loads still depend on the API; customer pages provide loading/error/retry feedback.

### 2026-10-31 · Customer dashboard visual redesign ✅ (uncommitted)
- User requested improvement to the customer Overview screenshot. Added `customer_dashboard.dart` and routed the existing customer branch of `dashboard_screen.dart` into it; admin/staff dashboards retain their existing behavior.
- Lavender welcome panel with a decorative subscription-card illustration, account identity/profile panel, plan previews with live INR prices/cadence/included products, and navigation shortcuts to subscriptions, invoices, and payments. No fake statistics or new API dependencies.
- Responsive single/two-column layout; a single available plan uses a horizontal card on desktop. Empty catalogue, narrow screens, landscape and 2× text supported. Uses existing Poppins, Material icons and workspace colors.
- Validation: analyzer clean; 24 tests passed / 1 opt-in live API skipped; release web build and Wasm dry run passed (existing Cupertino font warning). Six new dashboard checks cover desktop, phone, landscape, large text, zero/one plans and navigation to catalogue. Rendered captures visually reviewed in `~/Documents/subly-ui-preview/customer-dashboard-*.png`.
- Running Flutter web process on :5000 hot-restarted successfully. Changed files: `frontend/lib/features/dashboard/presentation/{customer_dashboard,dashboard_screen}.dart`, `frontend/test/workspace_test.dart`, `AGENT_HANDOFF.md`.
- Next: continue live browser demo checks in section 8.

### 2026-10-31 · Products red error screen fixed ✅ (uncommitted)
- Reproduced by typing in New product and cancelling: text controllers were disposed while the closing dialog still used them, cascading into Flutter's `_dependents.isEmpty` assertion.
- `frontend/lib/core/widgets/ui_kit.dart`: `showApiForm` now waits for the dialog route's `completed` future before returning, so callers dispose controllers after its fields leave the tree. Shared fix also protects variant and other API forms.
- `frontend/test/workspace_test.dart`: Cancel and Save regression cases fill a product form, close it, and reopen it without widget errors; fake product creation returns API-shaped JSON.
- Validation: `flutter analyze` clean; `flutter test` 18 passed / 1 opt-in live API test skipped; `flutter build web` succeeded. Existing Cupertino icon font warning remains. Running Flutter process on :5000 hot-restarted successfully.
- Changed files for this fix: `frontend/lib/core/widgets/ui_kit.dart`, `frontend/test/workspace_test.dart`, `AGENT_HANDOFF.md`. Other existing work preserved.
- Next: continue browser demo checks in section 8, including product and variant form closure.

### 2026-10-31 · Plans screen redesign — product-first browse + tier cards ✅ (uncommitted)
- **Customer** now lands on a product browse grid, derived from the plans the customer can already see (no backend change — plans endpoint already embeds `products`). Each product card: tinted tier icon, product name, "from ₹XXX", plan count pill, billing cadences pill, "View plans" CTA.
- Picking a product shows every active+available plan that bundles it, rendered as the tier-style pricing cards (sparkle / lightning / crown / enterprise icon). Second tier gets the "Most popular" overlay badge. CTA stays `Request quotation` → `POST /me/quotations`.
- A back/breadcrumb button ("All products") returns to the browse grid. Search in the browse view filters by product name; Billing filter still re-queries plans.
- **Admin / staff** keep the full plan list (no product gate) but now use the same tier cards, with Edit / Archive in a top-right overflow menu (admin), and a subscribers / MRR-contribution footer (staff + admin). Top of the admin page gets a 4-tile metrics row: Total plans · Available now · Active subscribers · Total MRR from plans.
- Updated `test/workspace_test.dart`: admin/staff/customer plan tests rewritten against the new UX (customer flow now opens via the product card first). 16 tests pass, 1 skipped (unchanged); analyzer clean.
- Hot-restart signalled to the running Flutter dev process on :5000.
- Files: `frontend/lib/features/plans/presentation/plans_screen.dart` (rewritten), `frontend/test/workspace_test.dart` (plan tests updated), `AGENT_HANDOFF.md`.

### 2026-10-31 · Full stack restarted ✅
- `make dev-stop` + `docker compose --profile dev up -d --build db migrate mailpit swagger api-dev` → API `/health` ok, DB ok. Flutter web (debug) on `0.0.0.0:5000` via `flutter run -d web-server`, backgrounded. Data unchanged from the clean-up (2 users, 1 customer).

### 2026-10-31 · Removed all fake data from the dev DB ✅
- Code and seeds were already clean (seed file is a no-op; only form hint text like "Jane Doe" remains in the login screen).
- Dev DB held integration-test leftovers (`*@test.brightloop` users/customers, "Seat 1793…" products, test plans/taxes,
  subscriptions, invoices, payments). Truncated all catalogue/sales tables (`RESTART IDENTITY`) and deleted the test users/customers.
- Kept: `admin@brightloop.local` (seed admin) and `abraranas@gmail.com` (Abrar's real sign-up, customer 22, had no subscriptions).
- **Note:** `make be-itest` runs against the dev DB and will re-create `@test.brightloop` rows; repeat the clean-up after running it.

### 2026-10-31 · Removed Price calculator + demo seed data ✅ (uncommitted)
- Frontend: removed Price calculator from navigation and router (`/pricing` no longer reachable from the app menu/routes).
- Sidebar wording updated: plan menu label now stays `Plans` (no `Pricing & Plans`).
- UI tests updated to remove the Price calculator screen route/import/test case.
- Database seeds: `database/seeds/01_catalog.sql` is now intentionally no-op, so `make seed` no longer inserts fake/demo catalogue rows.
- Docs updated so seed behavior is explicit (`README.md`, `database/README.md`, Makefile help text).

### 2026-10-31 · Customer plan → quotation request ✅ (uncommitted)
- Added customer self-service endpoint `POST /me/quotations` (JWT customer only). Input is minimal (`plan_id`, optional dates); customer id comes from token.
- Service creates the quotation using existing pricing/catalog rules and immediately moves it to `quotation` state so staff can process it. Customers now see `quotation` rows in **My subscriptions** (still never `draft`).
- Frontend plans page now shows **Request quotation** for customer users on available plans. Action calls `/me/quotations`; success refreshes portal data.
- Integration coverage extended in `TestCustomerPortal` (request quote success + invalid plan). `make be-itest` passes; `flutter analyze` passes.

### 2026-10-31 · Customer portal ✅ (uncommitted)
- Only what the brief asks: customers see ONLY their own subscriptions, invoices (confirmed/paid only) and payments, and can
  pause / resume / renew / close their own subscription where the plan allows (renew creates a renewal quotation for staff).
  Customer id from the JWT; other people's records → 404. Removed the unbuilt /me stubs (request plan, accept, simulated pay, pdf, me/dashboard).
- Frontend: customer menu = Dashboard, Plans, My subscriptions, My invoices, Payment history, Profile.
- 17 integration tests pass (`TestCustomerPortal` new); 17 frontend tests pass; web build OK. DB wiped again (only admin).

### 2026-10-31 · Menu clean-up ✅ (uncommitted)
- Removed the Reports page (placeholder; the dashboard is the summary view) and the notifications bell (no feature behind it).
- Customers no longer see Subscriptions / Invoices / Payments in the menu: those pages use staff endpoints and the customer
  portal (`/me/*`) isn't built. Customer menu = Dashboard, Plans, Profile. Route test updated.

### 2026-10-31 · A7–A12 + admin panel (deadline push) ✅ (uncommitted)
- **Backend:** `dto/sales.go`, `service/sales_service.go` (quotations, lifecycle, billing engine, invoices, payments list,
  dashboard), `handler/sales_handler.go`; routes `routes_{quotations,subscriptions,invoices,payments,reports,admin}.go`;
  `jobs/scheduler.go` now takes a `BillingFunc` (main.go passes `salesSvc.RunBilling`).
- **Gayatri's files copied unchanged** from `origin/gayatri_billing` (so a later merge of her branch has no add/add conflicts):
  `dto/payment.go`, `handler/payment_handler.go`, `repository/{invoice_repo,payment_repo}.go`, `service/payment_service.go`.
  Router field name `Payment` matches her branch.
- **Billing:** confirm → if start ≤ today: activate + bill the first period. Each period is billed in one tx: lock (SKIP LOCKED) →
  price via `Price()` with `Eligible(periodStart)` → `ReserveUse` (re-price if a discount ran out) → invoice + items +
  discount_usages → advance `next_billing_date` / `cycles_billed` / auto-close. Catch-up loop (cap 400). Invoices are
  auto-confirmed when `INVOICE_AUTO_CONFIRM=true` (default), due = invoice date + payment_terms_days; emailed in the background.
  `RunBilling` also activates due confirmed subs and closes expired quotations. Cancel invoice (unpaid only) releases discount uses.
- **Test:** `tests/integration/sales_test.go` `TestAdminFlow` (quote → confirm → invoice 1839.62 → no duplicates on 3 reruns →
  overpay 422 → partial → paid → 3-month catch-up = exactly 3 more invoices even when run twice → pause/resume → non-pausable
  422 → close → cancel invoice → dashboard money admin-only → staff 403 billing run). All 16 integration tests pass.
- **Frontend:** `features/sales/data/sales_api.dart`, `sales/presentation/sales_widgets.dart`; screens Quotations (create from
  plan/template/custom lines, send, confirm, cancel), Subscriptions (detail with bill now/pause/resume/renew/close + invoices),
  Invoices (filters, overdue chip, detail, confirm, cancel, send, printable view, record payment with Idempotency-Key),
  Payments list, Settings (run billing as of a date), Dashboard finance block (`_Finance`, date range).
  17 widget tests + live API test (`LIVE_API=…`) pass; web build OK.
- DB wiped again afterwards (only admin left).


### 2026-10-31 · A5 + A6 backend, frontend wired to the live API ✅ (uncommitted)
- **Backend A5:** migration `20261101000500_templates_unique_name_ci`; seed copies the plan's products into the
  "Standard SME offer" template (idempotent). Models: `QuotationTemplate.Plan`, `QuotationTemplateLine.Product/Variant`.
  New `dto/template.go`, `repository/template_repo.go`, `service/template_service.go`, `handler/template_handler.go`; routes in `routes_plans.go`.
- **Shared line validation** `service/catalog_lines.go` (`resolveLines`, `resolvedLine.UnitPrice/TaxRate`) is used by plans, templates and pricing.
- **Backend A6:** migration `20261101000600_discounts_code_ci` (codes upper-case, `ux_discounts_code_ci`). New `service/pricing.go`
  (+ `pricing_test.go`), `dto/discount.go`, `repository/discount_repo.go` (List with states, `Eligible`, `ReserveUse`, `ReleaseUse`,
  `Usages`), `service/discount_service.go` (rules + `PricingService.Preview`), `handler/discount_handler.go`, `routes_discounts.go`
  (adds `POST /pricing/preview`). Discount states: active | scheduled | expired | used_up | inactive.
- `TestDiscountsAndPricing` switches its discounts off in `t.Cleanup` (discounts affect every price in the shared dev DB).
- **Live check:** seed "10% off orders over 10,000" starts 2026-11-01. A preview of 10 × Odoo Brand dated 2026-11-01 = 15590 − 1559 + 2525.58 = **16556.58**.
- **OpenAPI:** templates, discounts, usages, pricing preview (30 paths, valid). YAML gotcha: quote flow values that contain commas.
- **Frontend (user asked: connect everything, no fake data):**
  - New: `core/widgets/ui_kit.dart`, `core/network/json.dart`, data layers in `features/{customers,staff,taxes,products,plans,templates,discounts,profile}/data/`.
  - New: `products/presentation/line_editor.dart`, screens `taxes/`, `templates/`, `pricing/`.
  - Rewritten on the API: `dashboard_screen.dart`, `customers_screen.dart`, `staff_screen.dart`, `products_screen.dart`, `plans_screen.dart`,
    `discounts_screen.dart`, `profile_screen.dart`.
  - Nav and routes: `/taxes`, `/templates`, `/pricing`. `AuthController.refreshUser/replaceToken` added; `PreviewNotice` removed.
  - `main.dart`: `ProviderScope(retry: (_, _) => null)`, so Riverpod 3 doesn't auto-retry failed (e.g. 403) requests.
  - Bug caught by tests: the Status filter crashed because "Active" was a null option while the value was 'active' → fixed.
  - UI captures: `~/Documents/subly-ui-preview/` (`SUBLY_UI_CAPTURE=… flutter test test/workspace_test.dart`).
  - The running dev app was hot-restarted with `kill -USR2 <flutter_tools pid>` (works for any `flutter run`).


### 2026-10-31 · A3 Products + A4 Plans ✅ (partly committed in `80a4dad`)
- **Migrations:** `20261101000300_products_unique_active_name` (`ux_products_name_active ON lower(name) WHERE deleted_at IS NULL AND status='active'`,
  `ix_products_type`, `ux_variants_product_attr_name_ci`) and `20261101000400_plans_unique_name_ci` (drops `subscription_plans_name_key`,
  adds `ux_plans_name_ci`, `ix_plans_status_availability`). Both tested up → down → up. Seed plans insert now `ON CONFLICT ((lower(name)))`.
- **New:** `service/money.go` (`parseMoney`, `parseDate`/`ParseDate`, `dateString`; dates stored as UTC midnight) + `money_test.go`,
  `dto/product.go`, `dto/plan.go`, `repository/product_repo.go`, `repository/plan_repo.go` (`recalcSQL`, `RecalculatePrice`,
  `RecalculateForProduct`, `ActiveSubscriptions`), `service/product_service.go`, `service/plan_service.go`,
  `handler/product_handler.go`, `handler/plan_handler.go`, `tests/integration/products_test.go`, `plans_test.go`.
- **Changed:** `routes_catalog.go` and `routes_plans.go` (real handlers; quotation-template routes are still `todo`), `router.go` Deps
  (`Products`, `Plans`), `main.go` wiring, `handler.go` (`pathParamID` for `:vid`), test `setup()`.
- **Rules:**
  - Products: prices ≥ 0, at most 2 dp; the tax must be active; variants may be created inline on POST; variant `extra_price` may be negative but the price may not go below 0.
  - Archive is blocked while an active plan includes the product (`PRODUCT_IN_USE`).
  - Variants: deactivation is blocked while an active plan uses the variant (`VARIANT_IN_USE`).
  - Plans: at least 1 product line; the variant must belong to the product and be active; no duplicate lines; `available_until ≥ available_from`;
    `auto_close_enabled` requires `auto_close_after_cycles`.
  - Plans: the billing period/interval is locked while confirmed/active/paused subscriptions exist (`PLAN_IN_USE`); reactivating is blocked if a line is inactive.
- **Live check:** seed "Software Subscription" variants show Brand A 999.00 and Odoo Brand 1559.00; "Starter Monthly" price is 999.00, labelled Monthly.
- **Postman generator fix:** id variables in JSON bodies are now unquoted (`{{productId}}`, not `"{{productId}}"`), otherwise the API rejects them as strings.
- **OpenAPI:** `/products*`, `/plans*` fully specified (24 paths, validated).


### 2026-10-31 · Browser-fit workspace ✅ (uncommitted)
- User requested removal of the large outer margins visible in the browser.
- `frontend/lib/core/widgets/app_shell.dart`: removed desktop horizontal/vertical frame padding; the sidebar and content now fill the viewport. Content keeps rounded left corners beside the sidebar, with square right corners at the browser edge. Removing the frame also gives the navigation more vertical room.
- Validation: Flutter analysis clean, all 8 workspace widget tests pass (desktop, mobile, tablet, landscape and large text). Hot reload requested for the running dev app.
- Changed files for this adjustment: `frontend/lib/core/widgets/app_shell.dart`, `AGENT_HANDOFF.md`.
- Existing UI captures show the earlier framed layout; regenerate them before using them to review the full-viewport layout.
- Next backend steps remain A3/A4; no backend changes in this adjustment.

### 2026-10-31 · Reference UI implementation ✅ (uncommitted)
- User explicitly asked to copy the supplied pricing dashboard into the Flutter project; frontend edits are authorized for this task.
- `frontend/lib/core/widgets/app_shell.dart`: light workspace regardless of OS dark mode, lavender sidebar, pill selection, rounded white panel, searchable role-filtered navigation, account menu/log out, notification dialog, mobile drawer. Original role guards remain.
- New `core/theme/workspace_theme.dart`, `core/widgets/workspace_widgets.dart`: shared colors, typography, headings, metrics and preview notice. Auth screens retain their existing theme.
- `features/plans/presentation/plans_screen.dart`: four pricing cards matching the reference (USD preview prices), Professional popular badge/gradient border, feature checks, subscribers, MRR and revenue bars. Admin-only create/edit/archive operate in session only, with validation and archive confirmation; no server writes. Staff cannot see financial totals; customers cannot see company subscriber metrics.
- `features/dashboard/presentation/dashboard_screen.dart`: overview cards, growth chart, plan breakdown and activity in the same style; sample content is explicitly labeled. Other feature bodies remain existing placeholders inside the updated shell.
- `frontend/pubspec.yaml` + `assets/fonts/{Poppins-Regular,Poppins-Medium,Poppins-SemiBold}.ttf` + `OFL.txt`: bundled Poppins font and license, no new package dependencies.
- `frontend/test/workspace_test.dart`: 8 widget tests for desktop/mobile/tablet/landscape, 2x text, role restrictions, plan creation/validation/archive and navigation. Together with existing tests, 14 pass; analyzer clean; release web build passes (Wasm dry run also succeeded).
- UI captures: `~/Documents/subly-ui-preview/{pricing-desktop,pricing-phone,pricing-tablet,pricing-landscape,overview-desktop,overview-phone}.png`. Regenerate with `SUBLY_UI_CAPTURE="$HOME/Documents/subly-ui-preview" flutter test` from `frontend/`.
- Browser plugin had no connected browsers; screenshots were generated with Flutter rendering and visually inspected. A hot restart was requested from the existing Flutter dev process on port 5000.
- Backend APIs for plans/dashboard remain unwired; connect these views to live API data once the backend modules land. Existing untracked backend files/migrations and seed edit were not modified.

### 2026-10-31 · Architecture / DB diagrams (outside the repo)
- `~/Documents/subly-diagrams/{01-architecture,02-database-erd,03-business-flow,04-auth-and-lifecycles}.{png,svg}`.
  The ERD is generated from the live schema (20 tables, 225 columns, PK/FK/UQ, status badges).
- Graphviz 2.42 spline bug: the generator falls back to `curved` edges and converts labels to `xlabel`; an empty `<FONT>` crashes HTML labels.

### 2026-10-31 · A2 Taxes ✅ (uncommitted)
- New: `dto/tax.go`, `repository/tax_repo.go`, `service/tax_service.go` (+ `_test.go`), `handler/tax_handler.go`,
  `router/routes_taxes.go` (moved out of `routes_catalog.go`, added `GET /taxes/:id`), `tests/integration/taxes_test.go`,
  migration `20261101000200_taxes_unique_name_ci` (drops `taxes_name_key`, adds `ux_taxes_name_ci ON lower(name)`).
- Seed fix: the taxes insert uses `ON CONFLICT ((lower(name)))` (the old `(name)` target broke after the migration).
- Responses include `products_using` (one grouped query, no N+1).
- Incident: the db container's `/seeds` bind mount went stale (folder recreated on the host) → fixed with `docker compose up -d --force-recreate db` (data kept).

### 2026-10-31 · A1 Customers ✅ (uncommitted)
- New: `dto/customer.go`, `service/customer_service.go`, `handler/customer_handler.go`, `validator/phone.go`,
  `tests/integration/customers_test.go`, migration `20261101000100_customers_unique_email`
  (`ux_customers_email_live ON lower(email) WHERE deleted_at IS NULL`).
- Changed: `repository/customer_repo.go` (FindByID, FindLiveByEmail, EmailTaken, List, Stats, Archive),
  `service/auth_service.go` (Register links to an existing customer; phone validation), routes, `main.go`, test setup (`env.db` added).
- `GET /customers/:id` returns `stats` (active/total subscriptions, open/overdue invoices, amount_due).
- Archive → 422 `CUSTOMER_HAS_OPEN_ITEMS` while there are draft/quotation/confirmed/active/paused subscriptions or unpaid invoices.

### 2026-10-31 · `make dev` + Flutter upgrade (merged, PR #3)
- Air hot reload in Docker (`Dockerfile.dev`, `.air.toml` uses `bin=` and `poll=true`), Makefile `dev*`/`run`/`setup`/`wait-api`, fixed `make help`.
- Flutter upgraded 3.38.5 → 3.47.9.

### 2026-10-31 · Subly branding + Gmail SMTP + test-email (merged)
- `pkg/brand`, `cmd/mailtest`, `MAIL_FROM_NAME`, `EMAIL_SHOW_CODES`; Flutter title, manifest and Android label → Subly.

### 2026-10-31 · Auth module (merged, PR #1)
- Forgot/reset, change password, profile, staff CRUD, `CheckSession` middleware, mailer, 6 integration tests, Postman collection.

### 2026-10-31 · Scaffold (merged)
- Monorepo, docker-compose, full schema, OpenAPI, CI, Flutter Riverpod/GoRouter skeleton, every planned route registered with roles (501 stubs).

---

## 8. Next steps (priority order — see also `~/Documents/ANTIGRAVITY_PROMPT.md`)

1. **Manual demo run in the browser** (`make dev`, admin login) following the demo script in the prompt; fix anything broken. Product form close regression is fixed/tested; confirm product and variant forms in the live browser. Customer sections now share the dashboard styling; check instant page navigation and Refresh with live records.
2. **OpenAPI + Postman** for A7–A12 and `/me/*` (endpoints exist; docs don't).
3. **Staff flow check:** log in as staff; confirm admin-only actions are hidden and return 403 (taxes/products/plans/discounts writes, staff, settings, money on dashboard).
4. Optional if time: invoice PDF (currently a printable view + Ctrl+P), editing a draft quotation (currently cancel + recreate), load test.

---

## 9. Known issues and gotchas

- **Teammate branches edit `router.go` and `cmd/api/main.go`** → expect merge conflicts there; merge one PR at a time.
- **Postman folders 6 and 8** read codes from Mailpit and fail while `.env` uses Gmail.
- **Stale bind mounts:** if `make seed` says `No such file`, run `docker compose up -d --force-recreate db`.
- **After editing `backend/.air.toml`**, restart the container (`docker compose --profile dev restart api-dev`); Air reads its config only at start.
- **`backend/tmp/`** is a root-owned empty dir created by Docker for api-dev; it's git-ignored and harmless.
- **The email format check is lenient** (`a@b` and `rahul@gmail` are accepted). A stricter check is optional, not done.
- **Not done (optional):** rate limiting on login/forgot, email verification, Google login, Flutter `/reset-password` and `/set-password` pages (Flutter team).
- **Postman bodies:** numeric ids must be unquoted in raw JSON (`"product_id": {{productId}}`). The generator does this for every variable ending in `Id`.
- **Frontend plans screen** (other session) still uses sample USD data; the real API returns INR money strings from `GET /plans`.
- **Dev DB was wiped on 2026-10-31** at the user's request (all tables truncated, ids and document sequences reset; only the admin
  remains, recreated by the API on start). `make seed` brings back the demo catalogue; integration tests/Newman add test rows again.
- **Riverpod 3 auto-retries failing providers** unless `ProviderScope(retry:)` is set (it is, in `main.dart` and the tests).
- **Integration tests and Newman runs add test rows to the dev DB** (timestamped emails). `make reset && make seed` cleans up.

## 10. Open decisions (team)
1. Confirm module ownership (A0). 2. Staff see no financial totals? 3. Scheduled invoices auto-confirmed (`INVOICE_AUTO_CONFIRM=true`)?
4. Quotations stored in `subscriptions` (ADR-001)? 5. ~~Unique product names?~~ Decided: unique among active products (decision 10); reversible.

---

## 11. Implemented Features (Quotations, Subscriptions, Invoices, Payments & Razorpay)
- **Quotations:** Backend CRUD, NextSequence generator (`QT/YYYY/00001`), line resolution, calculate subtotal/tax/discount/total, list/filter, send, confirm-to-subscription, cancel. Frontend full UI screen with dialogs.
- **Subscriptions:** Backend NextSequence generator (`SUB/YYYY/00001`), lifecycle transitions (Draft → Quotation → Confirmed → Active → Closed/Paused/Renewed), and on-demand `BillNow` invoice generation. Frontend full UI screen with lifecycle action controls.
- **Invoices:** Backend NextSequence generator (`INV/YYYY/00001`), period start/end tracking, overdue calculation, line item generation, confirm, cancel, send, and PDF/HTML generation. Frontend full UI screen with overdue badge and payment actions.
- **Payments & Razorpay:**
  - Manual payment recording: UPI, Bank Transfer, Card, Cash with auto-application against invoice due amounts (`PAY/YYYY/00001`).
  - Razorpay Online Checkout: integrated with key `rzp_test_TuNhl7wCahhafO` and secret `LwRFRG06o0FB5duWLv76e8cd`.
  - Backend creates Razorpay order via Razorpay API (`POST /invoices/:id/razorpay/order`) and verifies HMAC SHA256 signature (`POST /invoices/:id/razorpay/verify`).
  - Frontend triggers browser Razorpay modal popup via `dart:js` and refreshes invoice to Paid on completion.

