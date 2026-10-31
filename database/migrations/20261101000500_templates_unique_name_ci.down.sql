BEGIN;
DROP INDEX IF EXISTS ux_templates_name_ci;
ALTER TABLE quotation_templates ADD CONSTRAINT quotation_templates_name_key UNIQUE (name);
COMMIT;
