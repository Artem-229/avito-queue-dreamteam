-- +goose Up
-- +goose StatementBegin
CREATE TABLE purchase_rights (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    item_id         UUID NOT NULL REFERENCES catalog_items(id),
    user_id         UUID NOT NULL,
    status          VARCHAR(20) NOT NULL CHECK (status IN ('granted', 'used', 'cancelled', 'expired')),
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_rights_user_item_status
    ON purchase_rights (user_id, item_id, status);

CREATE UNIQUE INDEX idx_purchase_rights_user_item_active
    ON purchase_rights (user_id, item_id)
    WHERE status = 'granted';

CREATE INDEX idx_purchase_rights_item_status_expires
    ON purchase_rights (item_id, status, expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS purchase_rights;
-- +goose StatementEnd