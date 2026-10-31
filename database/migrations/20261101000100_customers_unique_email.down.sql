BEGIN;
DROP INDEX IF EXISTS ix_customers_created_at;
DROP INDEX IF EXISTS ux_customers_email_live;
CREATE INDEX ix_customers_email ON customers (lower(email));
COMMIT;
