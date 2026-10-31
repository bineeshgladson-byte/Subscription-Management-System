-- Tax names are unique ignoring case ("GST 18%" = "gst 18%").
BEGIN;
ALTER TABLE taxes DROP CONSTRAINT IF EXISTS taxes_name_key;
CREATE UNIQUE INDEX ux_taxes_name_ci ON taxes (lower(name));
COMMIT;
