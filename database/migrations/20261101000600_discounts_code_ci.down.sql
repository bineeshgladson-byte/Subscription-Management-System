BEGIN;
DROP INDEX IF EXISTS ux_discounts_code_ci;
ALTER TABLE discounts ADD CONSTRAINT discounts_code_key UNIQUE (code);
COMMIT;
