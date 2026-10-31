# Subly

**Subscriptions. Billing. Simplified.**

Subly is the subscription management system built for Brightloop Software: products, plans, quotations,
subscriptions, automatic recurring invoices, discounts, payments and revenue dashboards.

| Part      | Stack                                                     | Folder                  |
|-----------|-----------------------------------------------------------|-------------------------|
| Frontend  | Flutter · Riverpod · GoRouter · Dio                       | [`frontend/`](frontend) |
| Backend   | Go · Gin · GORM · JWT · bcrypt                            | [`backend/`](backend)   |
| Database  | PostgreSQL 16 · golang-migrate                            | [`database/`](database) |
| API spec  | OpenAPI 3 (contract-first)                                | [`docs/api/openapi.yaml`](docs/api/openapi.yaml) |

Architecture plan (read this first): see [`docs/architecture.md`](docs/architecture.md).

---

## Quick start (5 minutes)

**You need:** Docker + Docker Compose, Flutter 3.38+, `make`. Go is optional: all Go
commands run inside Docker through the Makefile.

```bash
git clone <repo-url> brightloop && cd brightloop
cp .env.example .env                 # then set JWT_SECRET (openssl rand -hex 32)
make up                              # postgres + migrations + api + mailpit + swagger
make seed                            # run seed scripts (currently no-op)
make fe-get && make fe-run           # Flutter web on http://localhost:5000
```

| What           | URL                                   |
|----------------|---------------------------------------|
| API            | http://localhost:8080/api/v1          |
| Health         | http://localhost:8080/health          |
| Swagger UI     | http://localhost:8090                 |
| Mailpit inbox  | http://localhost:8025                 |
| Postgres       | `localhost:5433` (credentials in `.env`) · or `make psql` |
| Mock API       | http://localhost:4010 (`make mock-api`) |

**Demo login:** `admin@brightloop.local` / `Admin#Brightloop1` (from `SEED_ADMIN_*` in `.env`;
created automatically on first API start). Customers can self-register in the app.

Run `make help` to see every command.

---

## What works today (Phase 1 + 2 scaffold)

- ✅ Docker dev stack, full database schema (all tables, constraints, indexes)
- ✅ Backend layering with a complete reference module: **register, login, `/auth/me`**
- ✅ JWT auth middleware + role checks (`adminOnly`, `adminOrStaff`, `customerOnly`)
- ✅ **Every endpoint from the plan is registered with its correct roles** and returns `501 NOT_IMPLEMENTED` until built
- ✅ Standard JSON envelope, typed errors, pagination helper, injectable clock
- ✅ Billing date math (31 Jan → 28 Feb) with tests; scheduler wired (job body is a TODO)
- ✅ Flutter: Dio client + token storage + 401 handling, Riverpod auth state, role-based
  router and sidebar, working Login / Register screens, placeholder screen per feature
- ⏳ Everything else: see the roadmap in the architecture plan and the TODOs in code

## Repository layout

```
backend/     Go API (cmd/api, internal/{config,database,models,dto,repository,service,handler,middleware,router,validator,jobs}, pkg/)
frontend/    Flutter app (lib/core = shared, lib/features/<module>/{data,application,presentation})
database/    migrations/ (golang-migrate SQL) and seeds/
docs/        api/openapi.yaml, postman/, architecture.md, decisions/
deployment/  production compose, nginx config (Phase 10)
.github/     CI workflows, CODEOWNERS, PR template
```

## How to add a feature (the pattern)

**Backend** (copy the auth module):
1. Spec the endpoint in `docs/api/openapi.yaml` (PR first if it changes the contract).
2. `internal/dto/<module>.go` – request/response structs with `binding` tags.
3. `internal/repository/<module>_repo.go` – queries; every method takes `*gorm.DB`.
4. `internal/service/<module>_service.go` – rules + `db.Transaction(...)`.
5. `internal/handler/<module>_handler.go` – bind → service → `response.OK/Created/Error`.
6. In `internal/router/routes_<module>.go` replace `todo("...")` with your handler,
   and add the handler to `router.Deps` + `cmd/api/main.go`.

**Frontend** (copy `features/auth`):
1. `features/<module>/data/` – model with `fromJson`, repository using `apiClientProvider`.
2. `features/<module>/application/` – Riverpod providers / notifiers.
3. `features/<module>/presentation/` – replace `PlaceholderView` with the real screen;
   use `AsyncView` for loading/error, `Fmt.inr` for money, `StatusChip` for statuses.

**Database:** never edit a merged migration. Add a new pair:
`database/migrations/<YYYYMMDDHHMMSS>_<name>.up.sql` and `.down.sql`, then `make migrate`.

See [CONTRIBUTING.md](CONTRIBUTING.md) for branches, PRs and ownership.

## Troubleshooting

- **Port already in use:** change `API_PORT`, `DB_HOST_PORT` or `SWAGGER_PORT` in `.env`.
- **API exits with `JWT_SECRET must be at least 32 characters`:** set it in `.env`.
- **Android emulator can't reach the API:** use `--dart-define=API_BASE_URL=http://10.0.2.2:8080/api/v1`.
- **Flutter web CORS error:** run the app on port 5000 (`make fe-run`) or add your origin to `CORS_ALLOWED_ORIGINS`.
- **Start over with an empty database:** `make reset` (deletes all data).

### Product plans and customer email updates

In **Plans**, choose a product to view its billing options. Admins can create several plans for the same product (including a different cadence or a price override); the selected product is prefilled. Staff can view the same groups. Customers see only active plans available today. Product details also offer **View plans**.

Customer emails are automatically queued for quotation creation/sending/cancellation/expiry, subscription confirmation/activation/pause/resume/closure, renewal requests, issued/cancelled/paid invoices and payment receipts. Issued invoice emails include the item, discount, tax and outstanding-balance breakdown. No admin/staff copies are sent.

Renewal reminders run with billing at **7, 3 and 1 day before** a renewable subscription ends, or before the next billing date if it has no contract end date. Pending renewal quotations suppress more reminders. Requests made early keep the existing subscription active until its confirmed replacement starts. Reminders do not automatically renew or charge the customer.

Migration `20261101000700` adds the durable email queue. Delivery starts after each successful action, retries every minute after SMTP failures and survives API restarts. Repeated scheduler runs share a unique reminder key. As with SMTP generally, a crash after server acceptance but before recording delivery can still cause a retry. The existing `SMTP_*`, `BILLING_CRON` and `INVOICE_AUTO_CONFIRM` settings apply; no new mail service is required.

For isolated backend integration verification without touching live customer records or sending real mail, create a separate migrated test database and point `TEST_DATABASE_URL` at it. The test harness uses a fake mailer.
