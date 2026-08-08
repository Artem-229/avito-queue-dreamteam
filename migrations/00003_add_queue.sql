-- +goose Up
-- +goose StatementBegin
CREATE TABLE queue
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    item_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'waiting' CHECK (status IN ('waiting', 'sold_out', 'cancelled', 'expired', 'granted', 'purchased')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX idx_queue_user_item_active
    ON queue (user_id, item_id)
    WHERE status IN ('waiting', 'granted');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE queue;
-- +goose StatementEnd
