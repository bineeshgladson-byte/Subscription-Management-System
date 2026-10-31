-- A6: discount codes are stored upper-case; uniqueness ignores case for safety.
BEGIN;
ALTER TABLE discounts DROP CONSTRAINT IF EXISTS discounts_code_key;
CREATE UNIQUE INDEX ux_discounts_code_ci ON discounts (upper(code)) WHERE code IS NOT NULL;
COMMIT;
