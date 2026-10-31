# Database

- `migrations/` — schema, applied by the `migrate` service on `make up` (golang-migrate).
- `seeds/` — optional SQL scripts, loaded with `make seed`.
   Current default seed script is intentionally empty (no demo/fake data).

## Rules

1. **Never edit a migration that has been merged.** Add a new one.
2. Name files with a timestamp so parallel branches never clash:
   `20261102143000_add_invoice_notes.up.sql` + `20261102143000_add_invoice_notes.down.sql`
3. Every `.up.sql` needs a `.down.sql` that undoes it.
4. Index every foreign key column (Postgres does not do it automatically).
5. Money is `NUMERIC(12,2)`. Never `REAL`/`DOUBLE PRECISION`.
6. When you add an ENUM value or column, update `backend/internal/models/` too.

## Commands

```bash
make migrate         # apply pending migrations
make migrate-down    # roll back the last one
make psql            # SQL shell
make reset           # DANGER: drop all data and recreate
```

The first admin user is created by the API at startup from `SEED_ADMIN_EMAIL` /
`SEED_ADMIN_PASSWORD`, because the password must be bcrypt-hashed.

`20261101000700_email_notifications` stores customer email events alongside successful business actions. Unique event keys deduplicate reminders; pending rows retain retry times and delivery errors. Apply this migration before starting an API build with automatic notifications. Its down migration removes the queue and its delivery history.
