-- +goose Up
-- +goose StatementBegin
CREATE TABLE purchase_rights (
    id_purchase     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_item         UUID NOT NULL REFERENCES items(id_item) ON DELET CASCADE,
    id_user         UUID NOT NULL,
    purchase_status VARCHAR(20) NOT NULL CHECK (purchase_status IN ('granted', 'used', 'cancelled', 'expired')),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_rights_user_item_status
    ON purchase_rights (id_user, id_item, purchase_status)

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS purchase_rights;
-- +goose StatementEnd