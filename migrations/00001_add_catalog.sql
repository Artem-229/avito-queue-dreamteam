-- +goose Up
-- +goose StatementBegin
CREATE TABLE catalog_items
(
    id                UUID PRIMARY KEY        DEFAULT gen_random_uuid(),
    name              TEXT           NOT NULL,
    price_kopecks     BIGINT         NOT NULL CHECK (price_kopecks > 0),
    total_stock       INTEGER        NOT NULL CHECK (total_stock >= 0),
    hold_ttl_seconds  INTEGER        NOT NULL DEFAULT 120 CHECK (hold_ttl_seconds BETWEEN 15 AND 900),
    granted_count     INTEGER        NOT NULL DEFAULT 0 CHECK (granted_count >= 0),
    used_count        INTEGER        NOT NULL DEFAULT 0 CHECK (used_count >= 0),
    category          TEXT           NOT NULL,
    seller_name       TEXT           NOT NULL,
    created_at        TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT catalog_items_stock_invariant CHECK (granted_count + used_count <= total_stock)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE catalog_items;
-- +goose StatementEnd
