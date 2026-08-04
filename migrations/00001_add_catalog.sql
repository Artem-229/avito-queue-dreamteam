-- +goose Up
-- +goose StatementBegin
CREATE TABLE catalog_items
(
    id          UUID PRIMARY KEY,
    name        TEXT           NOT NULL,
    price       NUMERIC(10, 2) NOT NULL CHECK (price >= 0),
    total_stock INTEGER        NOT NULL CHECK (total_stock >= 0),
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE catalog_items;
-- +goose StatementEnd
