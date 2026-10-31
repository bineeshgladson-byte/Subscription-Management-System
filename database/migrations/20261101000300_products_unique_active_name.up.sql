-- A3: one ACTIVE product per name (case-insensitive). Archived products may
-- keep an old name, so a product can be replaced without renaming history.
BEGIN;
CREATE UNIQUE INDEX ux_products_name_active ON products (lower(name))
    WHERE deleted_at IS NULL AND status = 'active';
CREATE INDEX ix_products_type ON products (product_type);
-- Case-insensitive variant names per product (exact-case UNIQUE already exists).
CREATE UNIQUE INDEX ux_variants_product_attr_name_ci ON product_variants (product_id, lower(attribute), lower(name));
COMMIT;
