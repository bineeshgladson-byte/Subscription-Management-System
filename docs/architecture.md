# Architecture

The full architecture plan (problem statement, permission matrix, database design and
ER diagram, API list, billing workflow, roadmap, MVP scope) is published here:

**https://claude.ai/artifact/GoBJt3Bxh6jgRx6KNjo2N3** (shared with the team by the repo owner)

## Key decisions (see `docs/decisions/`)

1. Modular monolith: one Go service, one PostgreSQL database.
2. Riverpod (no codegen) + GoRouter + Dio in Flutter.
3. Quotations are rows in `subscriptions` with status `draft`/`quotation`; confirming changes the status.
4. Roles and payment methods are Postgres ENUMs, not tables.
5. Money is `NUMERIC(12,2)` in Postgres, `decimal.Decimal` in Go, `Decimal` in Dart, and a string in JSON.
6. Schema is owned by SQL migrations (`database/migrations`); GORM AutoMigrate is never used.
7. Billing runs from an in-process cron; correctness comes from the database:
   transaction + `FOR UPDATE SKIP LOCKED` + unique index on `invoices (subscription_id, period_start)`.
8. Contract-first API: `docs/api/openapi.yaml` is the source of truth.
9. Customer endpoints live under `/me` and take the customer id from the JWT only.

## Open questions (team to confirm)

- Password minimum: implemented as 9+ characters ("more than 8").
- Staff do not see MRR / ARR / revenue totals (financial reports are admin-only).
- Scheduled invoices are created as Confirmed (`INVOICE_AUTO_CONFIRM=true`).
