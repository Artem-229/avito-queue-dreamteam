-- +goose Up
-- +goose StatementBegin
CREATE TABLE catalog_items
(
    id          UUID PRIMARY KEY        DEFAULT gen_random_uuid(),
    name        TEXT           NOT NULL,
    price       NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    total_stock INTEGER        NOT NULL CHECK (total_stock >= 0),
    category    TEXT           NOT NULL,
    seller_name TEXT           NOT NULL,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE catalog_items;
-- +goose StatementEnd
