-- +goose Up
-- +goose StatementBegin
-- Индекс под ExpireAllOverdue (services/stats.go, GetStats): запрос ищет
-- просроченные granted-записи БЕЗ фильтра по item_id (по всем товарам
-- сразу), поэтому существующий queue_entries_expiry (item_id, expires_at)
-- ему не помогает — Postgres не может использовать первый столбец индекса,
-- если условия на него нет. Этот индекс — без item_id, чтобы поиск по
-- expires_at был прямым.
CREATE INDEX queue_entries_expiry_global ON queue_entries (expires_at) WHERE status = 'granted';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS queue_entries_expiry_global;
-- +goose StatementEnd
