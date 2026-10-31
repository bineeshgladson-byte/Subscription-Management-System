BEGIN;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS email_outbox;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS discount_usages;
DROP TABLE IF EXISTS discount_products;
DROP TABLE IF EXISTS discounts;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscription_items;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS quotation_template_lines;
DROP TABLE IF EXISTS quotation_templates;
DROP TABLE IF EXISTS plan_products;
DROP TABLE IF EXISTS subscription_plans;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS taxes;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS users;

DROP SEQUENCE IF EXISTS payment_no_seq;
DROP SEQUENCE IF EXISTS invoice_no_seq;
DROP SEQUENCE IF EXISTS subscription_no_seq;
DROP SEQUENCE IF EXISTS quotation_no_seq;

DROP TYPE IF EXISTS email_status;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_method;
DROP TYPE IF EXISTS discount_scope;
DROP TYPE IF EXISTS discount_type;
DROP TYPE IF EXISTS invoice_status;
DROP TYPE IF EXISTS subscription_status;
DROP TYPE IF EXISTS billing_period;
DROP TYPE IF EXISTS record_status;
DROP TYPE IF EXISTS product_type;
DROP TYPE IF EXISTS user_role;

COMMIT;
