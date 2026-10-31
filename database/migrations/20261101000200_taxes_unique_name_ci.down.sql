BEGIN;
DROP INDEX IF EXISTS ux_taxes_name_ci;
ALTER TABLE taxes ADD CONSTRAINT taxes_name_key UNIQUE (name);
COMMIT;
