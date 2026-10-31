-- One live (non-archived) customer per email, case-insensitive.
-- Archived customers keep their email so old invoices still show it, and a new
-- customer can reuse the address after archiving.
BEGIN;
DROP INDEX IF EXISTS ix_customers_email;
CREATE UNIQUE INDEX ux_customers_email_live ON customers (lower(email)) WHERE deleted_at IS NULL;
CREATE INDEX ix_customers_created_at ON customers (created_at DESC) WHERE deleted_at IS NULL;
COMMIT;
