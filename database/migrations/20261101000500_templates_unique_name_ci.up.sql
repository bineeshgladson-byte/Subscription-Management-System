-- A5: quotation template names unique ignoring case.
BEGIN;
ALTER TABLE quotation_templates DROP CONSTRAINT IF EXISTS quotation_templates_name_key;
CREATE UNIQUE INDEX ux_templates_name_ci ON quotation_templates (lower(name));
COMMIT;
