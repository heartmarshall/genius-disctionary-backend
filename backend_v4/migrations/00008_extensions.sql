-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX ix_ref_entries_text_trgm ON ref_entries USING GIN (text_normalized gin_trgm_ops);
CREATE INDEX ix_entries_text_trgm ON entries USING GIN (text_normalized gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS ix_entries_text_trgm;
DROP INDEX IF EXISTS ix_ref_entries_text_trgm;
DROP EXTENSION IF EXISTS pg_trgm;
