-- +goose Up
-- +goose StatementBegin
-- Слияние queue и purchase_rights: одна строка — одно участие пользователя в
-- очереди на товар, от входа до финального исхода. Раньше состояние покупки
-- жило в двух таблицах плюс счётчиках товара, и согласованность трёх мест
-- держалась на аккуратности кода.
--
-- Данные старых таблиц не переносятся: там демонстрационные участия, а
-- каталог живёт в catalog_items и миграцию переживает.
CREATE TABLE queue_entries
(
    id               UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    item_id          UUID        NOT NULL REFERENCES catalog_items (id),
    user_id          UUID        NOT NULL,
    status           TEXT        NOT NULL CHECK (status IN
                                                 ('waiting', 'granted', 'purchased', 'expired', 'cancelled', 'sold_out')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(), -- ключ FIFO
    granted_at       TIMESTAMPTZ,
    expires_at       TIMESTAMPTZ,
    resolved_at      TIMESTAMPTZ,
    initial_position INT,

    CONSTRAINT granted_has_deadline CHECK (
        status <> 'granted' OR (expires_at IS NOT NULL AND granted_at IS NOT NULL))
);

-- Одно активное участие на пару пользователь-товар. Предикат обязан дословно
-- совпадать с ON CONFLICT в QueueEntryRepository.Enter, иначе Postgres не
-- выведет этот индекс и вставка упадёт.
CREATE UNIQUE INDEX queue_entries_active
    ON queue_entries (user_id, item_id) WHERE status IN ('waiting', 'granted');

CREATE INDEX queue_entries_fifo ON queue_entries (item_id, status, created_at, user_id);
CREATE INDEX queue_entries_expiry ON queue_entries (item_id, expires_at) WHERE status = 'granted';
-- +goose StatementEnd

-- +goose StatementBegin
-- Счётчики хранят состояние от старых таблиц, а участий в новой ещё нет.
-- Без обнуления CHECK (granted_count + used_count <= total_stock) отклонит
-- первую же выдачу права на товаре, где раньше кто-то стоял.
UPDATE catalog_items SET granted_count = 0, used_count = 0;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE purchase_rights;
DROP TABLE queue;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE TABLE queue
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    item_id    UUID        NOT NULL REFERENCES catalog_items (id),
    status     TEXT        NOT NULL DEFAULT 'waiting' CHECK (status IN
                                                             ('waiting', 'sold_out', 'cancelled', 'expired', 'granted', 'purchased')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_queue_user_item_active
    ON queue (user_id, item_id) WHERE status IN ('waiting', 'granted');
CREATE INDEX idx_queue_item_status_created ON queue (item_id, status, created_at);

CREATE TABLE purchase_rights
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    item_id    UUID        NOT NULL REFERENCES catalog_items (id),
    user_id    UUID        NOT NULL,
    status     VARCHAR(20) NOT NULL CHECK (status IN ('granted', 'used', 'cancelled', 'expired')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_purchase_rights_user_item_status ON purchase_rights (user_id, item_id, status);
CREATE UNIQUE INDEX idx_purchase_rights_user_item_active
    ON purchase_rights (user_id, item_id) WHERE status = 'granted';
CREATE INDEX idx_purchase_rights_item_status_expires ON purchase_rights (item_id, status, expires_at);

-- Участия обратно не разворачиваются: миграция вперёд их не переносила,
-- разворачивать нечего.
UPDATE catalog_items SET granted_count = 0, used_count = 0;

DROP TABLE queue_entries;
-- +goose StatementEnd
