-- +goose Up
-- Миграция для добавления индексов, оптимизирующих производительность запросов
-- Создана на основе рекомендаций из internal/database/repository/PERFORMANCE.md

-- ============================================================================
-- DICTIONARY ENTRIES
-- ============================================================================

-- Индекс для prefix search (короткие запросы через ILIKE 'text%')
-- Используется text_pattern_ops для оптимизации prefix поиска
-- Это дополняет существующий GIN индекс для триграмм (длинные запросы)
CREATE INDEX IF NOT EXISTS idx_dictionary_entries_text_prefix 
ON dictionary_entries(text text_pattern_ops);

-- Примечание: 
-- - UNIQUE индекс на text_normalized уже существует (ux_dictionary_entries_text_norm)
-- - GIN индекс для триграмм уже создан в миграции 20260113170000_add_trgm_extension.sql

-- ============================================================================
-- SENSES
-- ============================================================================

-- Композитный индекс для фильтрации по entry_id и part_of_speech одновременно
-- Используется в DictionaryFilter для поиска слов по части речи
-- Улучшает производительность EXISTS подзапроса в applyFilters
CREATE INDEX IF NOT EXISTS idx_senses_entry_part_of_speech 
ON senses(entry_id, part_of_speech);

-- Примечание:
-- - Индекс на entry_id уже существует (ix_senses_entry_id)
-- - Индекс на part_of_speech уже существует (ix_senses_pos)
-- - Этот композитный индекс оптимизирует запросы с обоими условиями

-- ============================================================================
-- CARDS
-- ============================================================================

-- Частичный индекс для получения карточек к повторению
-- WHERE next_review_at IS NOT NULL уменьшает размер индекса и ускоряет запросы
-- Это улучшение существующего индекса ix_cards_next_review
CREATE INDEX IF NOT EXISTS idx_cards_next_review_at 
ON cards(next_review_at) 
WHERE next_review_at IS NOT NULL;

-- Индекс для фильтрации по статусу
-- Используется в GetDashboardStats и других запросах с фильтрацией по статусу
CREATE INDEX IF NOT EXISTS idx_cards_status 
ON cards(status);

-- Композитный индекс для дашборда с частичным условием
-- Оптимизирует запросы, которые фильтруют по статусу и next_review_at одновременно
-- Это улучшение существующего индекса ix_cards_status_next_review
CREATE INDEX IF NOT EXISTS idx_cards_status_next_review 
ON cards(status, next_review_at) 
WHERE next_review_at IS NOT NULL;

-- Примечание:
-- - UNIQUE constraint на entry_id уже существует (автоматически создаёт индекс)
-- - Индекс ix_cards_next_review уже существует, но без частичного условия
-- - Индекс ix_cards_status_next_review уже существует, но без частичного условия

-- ============================================================================
-- REVIEW LOGS
-- ============================================================================

-- Композитный индекс для получения истории повторений карточки
-- DESC порядок для reviewed_at оптимизирует ORDER BY reviewed_at DESC
-- Используется в ListByCardID для получения последних повторений
CREATE INDEX IF NOT EXISTS idx_review_logs_card_id_reviewed_at 
ON review_logs(card_id, reviewed_at DESC);

-- Примечание:
-- - Индекс на card_id уже существует (ix_review_logs_card_id)
-- - Индекс на reviewed_at уже существует (ix_review_logs_reviewed_at)
-- - Этот композитный индекс оптимизирует запросы с обоими условиями и сортировкой

-- ============================================================================
-- INBOX ITEMS
-- ============================================================================

-- Индекс для пагинации по дате создания (DESC порядок)
-- Используется в ListAll и ListPaginated для быстрой сортировки
CREATE INDEX IF NOT EXISTS idx_inbox_items_created_at 
ON inbox_items(created_at DESC);

-- ============================================================================
-- ОПТИМИЗАЦИЯ СУЩЕСТВУЮЩИХ ИНДЕКСОВ
-- ============================================================================

-- Удаляем старые индексы, которые заменены более оптимизированными версиями
-- ВАЖНО: новые индексы созданы выше, поэтому старые можно безопасно удалить
-- Частичные индексы (WHERE next_review_at IS NOT NULL) более эффективны,
-- так как они меньше по размеру и быстрее для запросов, которые фильтруют по next_review_at

-- Удаляем старый индекс на next_review_at (заменён частичным idx_cards_next_review_at)
-- Старый индекс покрывает все строки, новый - только с next_review_at IS NOT NULL
-- Это безопасно, так как запросы с WHERE next_review_at IS NULL редки и могут использовать seq scan
DROP INDEX IF EXISTS ix_cards_next_review;

-- Удаляем старый композитный индекс (заменён частичным idx_cards_status_next_review)
-- Новый индекс с частичным условием более эффективен для типичных запросов
DROP INDEX IF EXISTS ix_cards_status_next_review;

-- +goose Down
-- Откат миграции: удаляем новые индексы и восстанавливаем старые

-- Удаляем новые индексы
DROP INDEX IF EXISTS idx_dictionary_entries_text_prefix;
DROP INDEX IF EXISTS idx_senses_entry_part_of_speech;
DROP INDEX IF EXISTS idx_cards_next_review_at;
DROP INDEX IF EXISTS idx_cards_status;
DROP INDEX IF EXISTS idx_cards_status_next_review;
DROP INDEX IF EXISTS idx_review_logs_card_id_reviewed_at;
DROP INDEX IF EXISTS idx_inbox_items_created_at;

-- Восстанавливаем старые индексы
CREATE INDEX IF NOT EXISTS ix_cards_next_review ON cards(next_review_at);
CREATE INDEX IF NOT EXISTS ix_cards_status_next_review ON cards(status, next_review_at);

