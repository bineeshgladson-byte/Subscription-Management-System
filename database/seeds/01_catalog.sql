-- Demo catalogue: taxes, products with variants, plans, a template and a discount.
-- Safe to run more than once. The first ADMIN is created by the API on startup
-- from SEED_ADMIN_EMAIL / SEED_ADMIN_PASSWORD (it needs bcrypt, so not done in SQL).

BEGIN;

INSERT INTO taxes (name, rate) VALUES
    ('GST 18%', 18.00),
    ('GST 0%', 0.00)
ON CONFLICT (name) DO NOTHING;

INSERT INTO products (name, product_type, sales_price, cost_price, description, tax_id)
SELECT 'Software Subscription', 'software', 999.00, 300.00, 'Cloud ERP seat',
       (SELECT id FROM taxes WHERE name = 'GST 18%')
WHERE NOT EXISTS (SELECT 1 FROM products WHERE name = 'Software Subscription');

INSERT INTO products (name, product_type, sales_price, cost_price, description, tax_id)
SELECT 'Priority Support', 'addon', 499.00, 100.00, '4-hour response SLA',
       (SELECT id FROM taxes WHERE name = 'GST 18%')
WHERE NOT EXISTS (SELECT 1 FROM products WHERE name = 'Priority Support');

INSERT INTO product_variants (product_id, attribute, name, extra_price)
SELECT p.id, 'Brand', v.name, v.extra
FROM products p
CROSS JOIN (VALUES ('Brand A', 0.00), ('Odoo Brand', 560.00)) AS v(name, extra)
WHERE p.name = 'Software Subscription'
ON CONFLICT (product_id, attribute, name) DO NOTHING;

INSERT INTO subscription_plans (name, description, price, billing_period, billing_interval, min_quantity,
                                available_from, is_pausable, is_closable, is_renewable)
VALUES
    ('Starter Monthly', 'One seat, billed monthly', 999.00, 'monthly', 1, 1, '2026-01-01', TRUE, TRUE, TRUE),
    ('Business Yearly', 'Odoo Brand seats, billed yearly', 1559.00, 'yearly', 1, 5, '2026-01-01', FALSE, FALSE, TRUE)
ON CONFLICT (name) DO NOTHING;

INSERT INTO plan_products (plan_id, product_id, variant_id, quantity)
SELECT sp.id, p.id, v.id, 1
FROM subscription_plans sp
JOIN products p ON p.name = 'Software Subscription'
JOIN product_variants v ON v.product_id = p.id
WHERE (sp.name = 'Starter Monthly' AND v.name = 'Brand A')
   OR (sp.name = 'Business Yearly' AND v.name = 'Odoo Brand')
ON CONFLICT DO NOTHING;

INSERT INTO quotation_templates (name, plan_id, validity_days, payment_terms_days, terms)
SELECT 'Standard SME offer', id, 15, 15, 'Prices exclusive of GST. Payment due within 15 days of invoice.'
FROM subscription_plans WHERE name = 'Starter Monthly'
ON CONFLICT (name) DO NOTHING;

INSERT INTO discounts (name, discount_type, value, scope, min_purchase_amount, starts_on, ends_on, max_usage)
SELECT '10% off orders over 10,000', 'percentage', 10, 'subscription', 10000.00, '2026-11-01', '2026-12-31', 50
WHERE NOT EXISTS (SELECT 1 FROM discounts WHERE name = '10% off orders over 10,000');

COMMIT;
