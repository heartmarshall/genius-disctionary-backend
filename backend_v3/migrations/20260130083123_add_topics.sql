-- +goose Up
-- +goose StatementBegin
-- ============================================================================
-- TOPICS
-- ============================================================================
CREATE TABLE topics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Название топика (например, "Travel", "IT"). 
    -- Делаем уникальным, чтобы не было дублей.
    name TEXT NOT NULL UNIQUE, 
    
    description TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Триггер для обновления updated_at
CREATE TRIGGER trg_topics_updated
BEFORE UPDATE ON topics
FOR EACH ROW EXECUTE FUNCTION touch_updated_at();

-- ============================================================================
-- DICTIONARY ENTRY TOPICS (Junction Table)
-- ============================================================================
CREATE TABLE dictionary_entry_topics (
    entry_id UUID NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    
    -- Композитный первичный ключ предотвращает дублирование связи одной пары
    PRIMARY KEY (entry_id, topic_id)
);

-- Индекс для быстрого поиска слов по топику (обратный поиск).
-- Прямой поиск (топики по слову) покрывается первичным ключом, 
-- так как entry_id идет первым.
CREATE INDEX ix_dictionary_entry_topics_topic_id ON dictionary_entry_topics(topic_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS ix_dictionary_entry_topics_topic_id;
DROP TABLE IF EXISTS dictionary_entry_topics;
DROP TRIGGER IF EXISTS trg_topics_updated ON topics;
DROP TABLE IF EXISTS topics;
-- +goose StatementEnd
