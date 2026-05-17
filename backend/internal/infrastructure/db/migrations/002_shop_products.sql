-- +goose Up

CREATE TABLE products (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_id          UUID        REFERENCES channels(id) ON DELETE SET NULL,
    platform            VARCHAR(20) NOT NULL,
    platform_product_id TEXT        NOT NULL,
    name                TEXT        NOT NULL,
    description         TEXT        NOT NULL DEFAULT '',
    price               NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency            CHAR(3)     NOT NULL DEFAULT '',
    cover_image_url     TEXT        NOT NULL DEFAULT '',
    product_url         TEXT        NOT NULL DEFAULT '',
    status              VARCHAR(20) NOT NULL DEFAULT 'active',
    raw_data            JSONB,
    synced_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, platform, platform_product_id)
);

CREATE INDEX idx_products_user_id   ON products(user_id);
CREATE INDEX idx_products_platform  ON products(user_id, platform);
CREATE INDEX idx_products_status    ON products(user_id, status);

CREATE TABLE publish_job_products (
    publish_job_id UUID NOT NULL REFERENCES publish_jobs(id) ON DELETE CASCADE,
    product_id     UUID NOT NULL REFERENCES products(id)    ON DELETE CASCADE,
    PRIMARY KEY (publish_job_id, product_id)
);

-- +goose Down

DROP TABLE IF EXISTS publish_job_products;
DROP TABLE IF EXISTS products;
