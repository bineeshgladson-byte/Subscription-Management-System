# AGENT_HANDOFF.md

> **For any AI or developer picking this up:** read this whole file before touching code.
> It is updated after **every** completed step. The newest entry in the step log (section 7) is at the top.
> Last updated: **2026-10-31**, after **A2 Taxes** and the architecture diagrams.

---

## 0. Where to resume (read first)

| | |
|---|---|
| **Current step** | A2 Taxes ✅ done. **Next: A3 Products and variants** (spec in section 8). |
| **Branch** | `ashlin_dev` (Ashlin's backend branch). `main` is the team integration branch. |
| **Uncommitted work** | A1 Customers + A2 Taxes + this file are **not committed yet**. Ashlin commits by hand; see rule R1. Check with `git status`. |
| **Is it running?** | `make dev` = backend (hot reload) + Flutter. Health: `curl localhost:8080/health` |
| **Test before you claim done** | `make be-test` (unit) and `make be-itest` (integration, needs the DB up) |

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
| **A1 Customers** | ✅ built, **uncommitted** | `/customers` list/create/get(+stats)/update, DELETE = archive (admin) | sign-up links to an existing customer by email; archive blocked by open items |
| **A2 Taxes** | ✅ built, **uncommitted** | `/taxes` list/get/create/update, DELETE = deactivate (admin) | rate string 0–100, 2 dp; `TAX_IN_USE` guard |
| A3 Products + variants | ⏭ next | `/products`, `/products/:id/variants` | spec in section 8 |
| A4 Plans · A5 Templates · A6 Discounts + pricing | ⏳ planned (Ashlin) | | see the plan PDF |
| A7 Quotations · A8 Subscriptions | 🟡 Bineesh | | his branch has a subscription repo/service/handler |
| A9 Billing/invoices · A10 Payments | 🟡 Gayatri | | her branch has record-payment |
| A11 Dashboard · A12 Admin billing run | ⏳ planned (Ashlin) | | |
| Phase B Staff flow audit · Phase C Customer `/me/*` | ⏳ after Phase A | | |

**Tests right now:** 11 integration tests pass (`TestRegisterAndLogin, TestRBAC, TestForgotAndResetPassword,
TestChangePassword, TestUpdateProfile, TestStaffManagement, TestCustomerCRUDAndValidation, TestCustomerPermissions,
TestSignupLinksToExistingCustomer, TestCustomerArchive, TestTaxes`), plus unit tests (password, email, GSTIN, phone,
billing dates, tax rate parsing). Postman: 92 requests / 12 folders in `~/Documents/brightloop-postman/`.

**Migrations applied:** `20261101000000_init_schema` (20 tables), `20261101000100_customers_unique_email`,
`20261101000200_taxes_unique_name_ci`.

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
9. **Billing correctness comes from the DB:** transaction + `FOR UPDATE SKIP LOCKED` + unique `(subscription_id, period_start)`;
   month-end clamping via `billing_anchor_day` (`service/billing_dates.go`, tested).

---

## 7. Step log (newest first)

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

## 8. Next step: A3 Products and variants (detailed spec)

Brief: *"The admin adds it with its name, type, sales price, cost price and notes, and offers it in variants, for
example by brand, where choosing the Odoo brand adds 560 to the price."*

- **Tables (exist):** `products` (name, product_type ENUM software|service|addon, sales_price, cost_price,
  description, tax_id FK→taxes, status active|archived, soft delete) and `product_variants`
  (product_id FK CASCADE, attribute, name, extra_price, sku UNIQUE, is_active; UNIQUE (product_id, attribute, name)).
- **Routes:** in `routes_catalog.go` (currently `todo` stubs). Reads `adminOrStaff`, writes `adminOnly`:
  `GET/POST /products`, `GET/PUT/DELETE /products/:id` (DELETE = archive),
  `POST /products/:id/variants`, `PUT/DELETE /products/:id/variants/:vid`.
- **Rules:**
  - prices ≥ 0, money strings, at most 2 dp (reuse the `parseRate`-style parsing for money)
  - `tax_id`, if given, must be an **active** tax
  - POST may include `variants: [...]` (created in the same transaction)
  - each variant response has a computed `price = sales_price + extra_price`
  - variant attribute + name unique per product; SKU unique
  - archived products are hidden from lists by default (`?status=archived` to see them) and must not be usable in new quotations
  - list filters: `q`, `status`, `product_type`, `tax_id`; sort by name, sales_price, created_at
- **Consider:** a migration making the product name unique among active products (case-insensitive)? Ask the user first.
- **Done when:** the Odoo Brand variant shows ₹1,559 (999 + 560); staff get 403 on writes; customers 403 on everything;
  tax `products_using` counts update; integration tests, OpenAPI, Postman folder "12. Products (A3)" and this file updated.

---

## 9. Known issues and gotchas

- **Teammate branches edit `router.go` and `cmd/api/main.go`** → expect merge conflicts there; merge one PR at a time.
- **Postman folders 6 and 8** read codes from Mailpit and fail while `.env` uses Gmail.
- **Stale bind mounts:** if `make seed` says `No such file`, run `docker compose up -d --force-recreate db`.
- **After editing `backend/.air.toml`**, restart the container (`docker compose --profile dev restart api-dev`); Air reads its config only at start.
- **`backend/tmp/`** is a root-owned empty dir created by Docker for api-dev; it's git-ignored and harmless.
- **The email format check is lenient** (`a@b` and `rahul@gmail` are accepted). A stricter check is optional, not done.
- **Not done (optional):** rate limiting on login/forgot, email verification, Google login, Flutter `/reset-password` and `/set-password` pages (Flutter team).
- **Integration tests and Newman runs add test rows to the dev DB** (timestamped emails). `make reset && make seed` cleans up.

## 10. Open decisions (team)
1. Confirm module ownership (A0). 2. Staff see no financial totals? 3. Scheduled invoices auto-confirmed (`INVOICE_AUTO_CONFIRM=true`)?
4. Quotations stored in `subscriptions` (ADR-001)? 5. Unique product names?
