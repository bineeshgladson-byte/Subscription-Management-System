-- Brightloop Subscription Management System — initial schema.
-- Owned by SQL migrations (golang-migrate). Never use GORM AutoMigrate.
-- Requires PostgreSQL 15+ (UNIQUE NULLS NOT DISTINCT).

BEGIN;

-- ============ ENUM types (mirror backend/internal/models/enums.go) ============
CREATE TYPE user_role           AS ENUM ('admin', 'staff', 'customer');
CREATE TYPE product_type        AS ENUM ('software', 'service', 'addon');
CREATE TYPE record_status       AS ENUM ('active', 'archived');
CREATE TYPE billing_period      AS ENUM ('daily', 'weekly', 'monthly', 'yearly');
CREATE TYPE subscription_status AS ENUM ('draft', 'quotation', 'confirmed', 'active', 'paused', 'closed', 'renewed');
CREATE TYPE invoice_status      AS ENUM ('draft', 'confirmed', 'paid', 'cancelled');
CREATE TYPE discount_type       AS ENUM ('fixed', 'percentage');
CREATE TYPE discount_scope      AS ENUM ('subscription', 'product');
CREATE TYPE payment_method      AS ENUM ('upi', 'bank_transfer', 'card', 'cash');
CREATE TYPE payment_status      AS ENUM ('pending', 'completed', 'failed', 'refunded');
CREATE TYPE email_status        AS ENUM ('pending', 'sent', 'failed');

-- ============ Document number sequences (formatted in Go: INV/2026/00042) ============
CREATE SEQUENCE quotation_no_seq;
CREATE SEQUENCE subscription_no_seq;
CREATE SEQUENCE invoice_no_seq;
CREATE SEQUENCE payment_no_seq;

-- ============ Auth & people ============
CREATE TABLE users (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email               VARCHAR(255) NOT NULL,
    password_hash       VARCHAR(100) NOT NULL,
    full_name           VARCHAR(120) NOT NULL,
    role                user_role    NOT NULL,
    is_active           BOOLEAN      NOT NULL DEFAULT TRUE,
    password_changed_at TIMESTAMPTZ,
    last_login_at       TIMESTAMPTZ,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ
);
CREATE UNIQUE INDEX ux_users_email ON users (lower(email));
CREATE INDEX ix_users_role ON users (role);

CREATE TABLE customers (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id         BIGINT UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    name            VARCHAR(150) NOT NULL,
    company_name    VARCHAR(150),
    email           VARCHAR(255) NOT NULL,
    phone           VARCHAR(20),
    billing_address TEXT,
    gstin           VARCHAR(15),
    created_by      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX ix_customers_name ON customers (lower(name));
CREATE INDEX ix_customers_email ON customers (lower(email));
CREATE INDEX ix_customers_created_by ON customers (created_by);

CREATE TABLE password_reset_tokens (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_prt_user ON password_reset_tokens (user_id);

-- ============ Catalogue ============
CREATE TABLE taxes (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       VARCHAR(60) NOT NULL UNIQUE,
    rate       NUMERIC(5,2) NOT NULL CHECK (rate BETWEEN 0 AND 100),
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name         VARCHAR(150) NOT NULL,
    product_type product_type NOT NULL,
    sales_price  NUMERIC(12,2) NOT NULL CHECK (sales_price >= 0),
    cost_price   NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
    description  TEXT,
    tax_id       BIGINT REFERENCES taxes(id) ON DELETE RESTRICT,
    status       record_status NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX ix_products_tax ON products (tax_id);
CREATE INDEX ix_products_status ON products (status);

CREATE TABLE product_variants (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute   VARCHAR(60)  NOT NULL,
    name        VARCHAR(100) NOT NULL,
    extra_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    sku         VARCHAR(60) UNIQUE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (product_id, attribute, name)
);
CREATE INDEX ix_variants_product ON product_variants (product_id);

CREATE TABLE subscription_plans (
    id                      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                    VARCHAR(120) NOT NULL UNIQUE,
    description             TEXT,
    price                   NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (price >= 0),
    billing_period          billing_period NOT NULL,
    billing_interval        INT NOT NULL DEFAULT 1 CHECK (billing_interval > 0),
    min_quantity            INT NOT NULL DEFAULT 1 CHECK (min_quantity > 0),
    available_from          DATE NOT NULL DEFAULT CURRENT_DATE,
    available_until         DATE,
    auto_close_enabled      BOOLEAN NOT NULL DEFAULT FALSE,
    auto_close_after_cycles INT CHECK (auto_close_after_cycles IS NULL OR auto_close_after_cycles > 0),
    is_pausable             BOOLEAN NOT NULL DEFAULT FALSE,
    is_closable             BOOLEAN NOT NULL DEFAULT TRUE,
    is_renewable            BOOLEAN NOT NULL DEFAULT TRUE,
    status                  record_status NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (available_until IS NULL OR available_until >= available_from)
);

CREATE TABLE plan_products (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    plan_id        BIGINT NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
    product_id     BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    variant_id     BIGINT REFERENCES product_variants(id) ON DELETE RESTRICT,
    quantity       INT NOT NULL DEFAULT 1 CHECK (quantity > 0),
    price_override NUMERIC(12,2) CHECK (price_override IS NULL OR price_override >= 0),
    UNIQUE NULLS NOT DISTINCT (plan_id, product_id, variant_id)
);
CREATE INDEX ix_plan_products_product ON plan_products (product_id);
CREATE INDEX ix_plan_products_variant ON plan_products (variant_id);

CREATE TABLE quotation_templates (
    id                 BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name               VARCHAR(120) NOT NULL UNIQUE,
    plan_id            BIGINT REFERENCES subscription_plans(id) ON DELETE SET NULL,
    validity_days      INT NOT NULL DEFAULT 15 CHECK (validity_days > 0),
    payment_terms_days INT NOT NULL DEFAULT 15 CHECK (payment_terms_days >= 0),
    terms              TEXT,
    is_active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_qt_plan ON quotation_templates (plan_id);

CREATE TABLE quotation_template_lines (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    template_id BIGINT NOT NULL REFERENCES quotation_templates(id) ON DELETE CASCADE,
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    variant_id  BIGINT REFERENCES product_variants(id) ON DELETE RESTRICT,
    quantity    INT NOT NULL DEFAULT 1 CHECK (quantity > 0)
);
CREATE INDEX ix_qtl_template ON quotation_template_lines (template_id);
CREATE INDEX ix_qtl_product ON quotation_template_lines (product_id);
CREATE INDEX ix_qtl_variant ON quotation_template_lines (variant_id);

-- ============ Subscriptions (a quotation is a subscription in draft/quotation state) ============
CREATE TABLE subscriptions (
    id                     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    quotation_number       VARCHAR(30) NOT NULL UNIQUE,
    subscription_number    VARCHAR(30) UNIQUE,
    customer_id            BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    plan_id                BIGINT NOT NULL REFERENCES subscription_plans(id) ON DELETE RESTRICT,
    template_id            BIGINT REFERENCES quotation_templates(id) ON DELETE SET NULL,
    status                 subscription_status NOT NULL DEFAULT 'draft',
    start_date             DATE NOT NULL,
    expiry_date            DATE,
    quotation_valid_until  DATE,
    payment_terms_days     INT NOT NULL DEFAULT 15 CHECK (payment_terms_days >= 0),
    next_billing_date      DATE,
    billing_anchor_day     SMALLINT NOT NULL CHECK (billing_anchor_day BETWEEN 1 AND 31),
    cycles_billed          INT NOT NULL DEFAULT 0,
    parent_subscription_id BIGINT REFERENCES subscriptions(id) ON DELETE RESTRICT,
    close_reason           VARCHAR(60),
    subtotal               NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_total         NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax_total              NUMERIC(12,2) NOT NULL DEFAULT 0,
    total                  NUMERIC(12,2) NOT NULL DEFAULT 0,
    confirmed_at           TIMESTAMPTZ,
    activated_at           TIMESTAMPTZ,
    paused_at              TIMESTAMPTZ,
    closed_at              TIMESTAMPTZ,
    created_by             BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (expiry_date IS NULL OR expiry_date >= start_date)
);
CREATE INDEX ix_subs_customer ON subscriptions (customer_id);
CREATE INDEX ix_subs_plan ON subscriptions (plan_id);
CREATE INDEX ix_subs_status ON subscriptions (status);
CREATE INDEX ix_subs_template ON subscriptions (template_id);
CREATE INDEX ix_subs_parent ON subscriptions (parent_subscription_id);
CREATE INDEX ix_subs_created_by ON subscriptions (created_by);
-- The billing scheduler's "who is due?" query. Partial index stays small.
CREATE INDEX ix_subs_due ON subscriptions (next_billing_date) WHERE status = 'active';

CREATE TABLE subscription_items (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subscription_id BIGINT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    product_id      BIGINT NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    variant_id      BIGINT REFERENCES product_variants(id) ON DELETE RESTRICT,
    description     VARCHAR(255) NOT NULL,
    quantity        INT NOT NULL CHECK (quantity > 0),
    unit_price      NUMERIC(12,2) NOT NULL CHECK (unit_price >= 0),
    tax_id          BIGINT REFERENCES taxes(id) ON DELETE RESTRICT,
    tax_rate        NUMERIC(5,2) NOT NULL DEFAULT 0,
    line_subtotal   NUMERIC(12,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_sub_items_sub ON subscription_items (subscription_id);
CREATE INDEX ix_sub_items_product ON subscription_items (product_id);
CREATE INDEX ix_sub_items_variant ON subscription_items (variant_id);
CREATE INDEX ix_sub_items_tax ON subscription_items (tax_id);

-- ============ Invoices ============
CREATE TABLE invoices (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_number  VARCHAR(30) NOT NULL UNIQUE,
    customer_id     BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    subscription_id BIGINT REFERENCES subscriptions(id) ON DELETE RESTRICT,
    period_start    DATE,
    period_end      DATE,
    invoice_date    DATE NOT NULL,
    due_date        DATE NOT NULL,
    status          invoice_status NOT NULL DEFAULT 'draft',
    subtotal        NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_total  NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax_total       NUMERIC(12,2) NOT NULL DEFAULT 0,
    total           NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (total >= 0),
    amount_paid     NUMERIC(12,2) NOT NULL DEFAULT 0,
    amount_due      NUMERIC(12,2) GENERATED ALWAYS AS (total - amount_paid) STORED,
    customer_name   VARCHAR(150) NOT NULL,
    billing_address TEXT,
    customer_gstin  VARCHAR(15),
    sent_at         TIMESTAMPTZ,
    confirmed_at    TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ,
    cancel_reason   VARCHAR(255),
    created_by      BIGINT REFERENCES users(id) ON DELETE SET NULL, -- NULL = billing scheduler
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (amount_paid >= 0 AND amount_paid <= total),
    CHECK (due_date >= invoice_date),
    CHECK (period_end IS NULL OR period_end >= period_start)
);
-- THE anti-duplicate-billing guarantee: one live invoice per subscription per period.
CREATE UNIQUE INDEX ux_invoices_sub_period ON invoices (subscription_id, period_start)
    WHERE status <> 'cancelled' AND subscription_id IS NOT NULL;
CREATE INDEX ix_invoices_customer_date ON invoices (customer_id, invoice_date DESC);
CREATE INDEX ix_invoices_status_due ON invoices (status, due_date);
CREATE INDEX ix_invoices_subscription ON invoices (subscription_id);
CREATE INDEX ix_invoices_created_by ON invoices (created_by);

CREATE TABLE invoice_items (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    invoice_id      BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    product_id      BIGINT REFERENCES products(id) ON DELETE RESTRICT,
    variant_id      BIGINT REFERENCES product_variants(id) ON DELETE RESTRICT,
    description     VARCHAR(255) NOT NULL,
    quantity        INT NOT NULL CHECK (quantity > 0),
    unit_price      NUMERIC(12,2) NOT NULL,
    line_subtotal   NUMERIC(12,2) NOT NULL,
    discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    tax_rate        NUMERIC(5,2)  NOT NULL DEFAULT 0,
    tax_amount      NUMERIC(12,2) NOT NULL DEFAULT 0,
    line_total      NUMERIC(12,2) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_invoice_items_invoice ON invoice_items (invoice_id);
CREATE INDEX ix_invoice_items_product ON invoice_items (product_id);
CREATE INDEX ix_invoice_items_variant ON invoice_items (variant_id);

-- ============ Discounts ============
CREATE TABLE discounts (
    id                  BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name                VARCHAR(120) NOT NULL,
    code                VARCHAR(40) UNIQUE,
    discount_type       discount_type NOT NULL,
    value               NUMERIC(12,2) NOT NULL CHECK (value > 0),
    scope               discount_scope NOT NULL,
    min_purchase_amount NUMERIC(12,2) CHECK (min_purchase_amount IS NULL OR min_purchase_amount >= 0),
    min_quantity        INT CHECK (min_quantity IS NULL OR min_quantity > 0),
    starts_on           DATE NOT NULL,
    ends_on             DATE,
    max_usage           INT CHECK (max_usage IS NULL OR max_usage > 0),
    usage_count         INT NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_by          BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (discount_type <> 'percentage' OR value <= 100),
    CHECK (max_usage IS NULL OR usage_count <= max_usage),
    CHECK (ends_on IS NULL OR ends_on >= starts_on)
);
CREATE INDEX ix_discounts_active_window ON discounts (starts_on, ends_on) WHERE is_active;
CREATE INDEX ix_discounts_created_by ON discounts (created_by);

CREATE TABLE discount_products (
    discount_id BIGINT NOT NULL REFERENCES discounts(id) ON DELETE CASCADE,
    product_id  BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    PRIMARY KEY (discount_id, product_id)
);
CREATE INDEX ix_discount_products_product ON discount_products (product_id);

CREATE TABLE discount_usages (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    discount_id     BIGINT NOT NULL REFERENCES discounts(id) ON DELETE RESTRICT,
    invoice_id      BIGINT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    invoice_item_id BIGINT REFERENCES invoice_items(id) ON DELETE CASCADE,
    customer_id     BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (discount_id, invoice_id, invoice_item_id)
);
CREATE INDEX ix_discount_usages_invoice ON discount_usages (invoice_id);
CREATE INDEX ix_discount_usages_item ON discount_usages (invoice_item_id);
CREATE INDEX ix_discount_usages_customer ON discount_usages (customer_id);

-- ============ Payments ============
CREATE TABLE payments (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    payment_number  VARCHAR(30) NOT NULL UNIQUE,
    invoice_id      BIGINT NOT NULL REFERENCES invoices(id) ON DELETE RESTRICT,
    customer_id     BIGINT NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
    method          payment_method NOT NULL,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    payment_date    DATE NOT NULL,
    reference       VARCHAR(100),
    notes           TEXT,
    status          payment_status NOT NULL DEFAULT 'completed',
    idempotency_key VARCHAR(64) UNIQUE,
    recorded_by     BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_payments_invoice ON payments (invoice_id);
CREATE INDEX ix_payments_customer_date ON payments (customer_id, payment_date DESC);
CREATE INDEX ix_payments_completed_date ON payments (payment_date) WHERE status = 'completed';
CREATE INDEX ix_payments_recorded_by ON payments (recorded_by);

-- ============ Infrastructure ============
CREATE TABLE email_outbox (
    id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    to_email       VARCHAR(255) NOT NULL,
    subject        VARCHAR(255) NOT NULL,
    body           TEXT NOT NULL,
    attachment_ref VARCHAR(255),
    status         email_status NOT NULL DEFAULT 'pending',
    attempts       INT NOT NULL DEFAULT 0,
    last_error     TEXT,
    sent_at        TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_outbox_pending ON email_outbox (created_at) WHERE status = 'pending';

CREATE TABLE audit_logs (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
    action     VARCHAR(60) NOT NULL,
    entity     VARCHAR(60) NOT NULL,
    entity_id  BIGINT,
    changes    JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ix_audit_entity ON audit_logs (entity, entity_id);
CREATE INDEX ix_audit_user ON audit_logs (user_id);

COMMIT;
