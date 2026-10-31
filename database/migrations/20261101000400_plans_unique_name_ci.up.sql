-- A4: plan names unique ignoring case; fast lookups for the customer catalogue.
BEGIN;
ALTER TABLE subscription_plans DROP CONSTRAINT IF EXISTS subscription_plans_name_key;
CREATE UNIQUE INDEX ux_plans_name_ci ON subscription_plans (lower(name));
CREATE INDEX ix_plans_status_availability ON subscription_plans (status, available_from, available_until);
COMMIT;
