BEGIN;
DROP INDEX IF EXISTS ux_variants_product_attr_name_ci;
DROP INDEX IF EXISTS ix_products_type;
DROP INDEX IF EXISTS ux_products_name_active;
COMMIT;
