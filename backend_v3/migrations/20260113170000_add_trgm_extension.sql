-- +goose Up
-- Включаем расширение для работы с триграммами (fuzzy search)
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Создаем GIN индекс для столбца text.
-- gin_trgm_ops позволяет использовать операторы LIKE, ILIKE, %, <-> очень быстро.
CREATE INDEX ix_dictionary_entries_text_trgm 
ON dictionary_entries 
USING GIN (text gin_trgm_ops);

-- +goose Down
DROP INDEX IF EXISTS ix_dictionary_entries_text_trgm;
DROP EXTENSION IF EXISTS pg_trgm;