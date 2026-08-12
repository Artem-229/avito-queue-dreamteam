-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE catalog_items 
ADD COLUMN embedding vector(1024);

CREATE INDEX catalog_items_embedding_idx 
ON catalog_items USING hnsw (embedding vector_cosine_ops);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS catalog_items_embedding_idx;

ALTER TABLE catalog_items 
DROP COLUMN IF EXISTS embedding;

-- +goose StatementEnd