BEGIN;
DROP INDEX IF EXISTS ix_plans_status_availability;
DROP INDEX IF EXISTS ux_plans_name_ci;
ALTER TABLE subscription_plans ADD CONSTRAINT subscription_plans_name_key UNIQUE (name);
COMMIT;
