// Package dictionary содержит репозиторий для работы со словарными записями.
package dictionary

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository/base"
	"github.com/heartmarshall/my-english/internal/database/schema"
	"github.com/heartmarshall/my-english/internal/model"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	// MinSearchLength — минимальная длина поискового запроса для использования триграмм.
	// Для коротких запросов используется prefix search (ILIKE 'query%').
	MinSearchLength = 3

	// DefaultLimit — лимит по умолчанию для списка записей.
	DefaultLimit = 50

	// MaxLimit — максимальный лимит для списка записей.
	// Ограничивает потенциально тяжелые запросы.
	MaxLimit = 1000
)

// ============================================================================
// FILTER
// ============================================================================

// DictionaryFilter содержит параметры фильтрации и пагинации для поиска слов.
type DictionaryFilter struct {
	// Search — поисковый запрос (prefix для коротких, trigram для длинных)
	Search string

	// PartOfSpeech — фильтр по части речи (через EXISTS подзапрос к senses)
	PartOfSpeech *model.PartOfSpeech

	// HasCard — фильтр по наличию карточки (true/false/nil)
	HasCard *bool

	// TopicIDs — фильтр по топикам (слова должны иметь хотя бы один из указанных топиков)
	TopicIDs []uuid.UUID

	// Пагинация
	Limit  int
	Offset int

	// Сортировка
	SortBy  *model.WordSortField
	SortDir *model.SortDirection
}

// Normalize нормализует и валидирует фильтр, применяя дефолтные значения.
func (f *DictionaryFilter) Normalize() {
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	f.Search = strings.TrimSpace(f.Search)
}

// ============================================================================
// REPOSITORY
// ============================================================================

// DictionaryRepository предоставляет методы для работы со словарными записями.
type DictionaryRepository struct {
	*base.Base[model.DictionaryEntry]
}

// NewDictionaryRepository создаёт новый репозиторий словаря.
func NewDictionaryRepository(q database.Querier) *DictionaryRepository {
	return &DictionaryRepository{
		Base: base.MustNewBase[model.DictionaryEntry](q, base.Config{
			Table:   schema.DictionaryEntries.Name.String(),
			Columns: schema.DictionaryEntries.Columns(),
		}),
	}
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

// GetByID получает словарную запись по ID.
func (r *DictionaryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DictionaryEntry, error) {
	if base.IsZeroUUID(id) {
		return nil, fmt.Errorf("%w: id is required", database.ErrInvalidInput)
	}
	return r.Base.GetByID(ctx, schema.DictionaryEntries.ID.Bare(), id)
}

// FindByNormalizedText находит запись по нормализованному тексту.
func (r *DictionaryRepository) FindByNormalizedText(ctx context.Context, text string) (*model.DictionaryEntry, error) {
	if text == "" {
		return nil, fmt.Errorf("%w: text is required", database.ErrInvalidInput)
	}
	return r.FindOneBy(ctx, schema.DictionaryEntries.TextNormalized.Bare(), text)
}

// ListByIDs получает записи по списку ID.
func (r *DictionaryRepository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]model.DictionaryEntry, error) {
	if len(ids) == 0 {
		return []model.DictionaryEntry{}, nil
	}
	return r.Base.ListByUUIDs(ctx, schema.DictionaryEntries.ID.Bare(), ids)
}

// ListAll получает все записи словаря.
// Для MVP можно выгрузить все. Если записей > 10k, это будет бутылочным горлышком, но пока ок.
func (r *DictionaryRepository) ListAll(ctx context.Context) ([]model.DictionaryEntry, error) {
	b := r.SelectBuilder()
	return r.List(ctx, b)
}

// ExistsByNormalizedText проверяет существование слова по нормализованному тексту.
func (r *DictionaryRepository) ExistsByNormalizedText(ctx context.Context, text string) (bool, error) {
	if text == "" {
		return false, fmt.Errorf("%w: text is required", database.ErrInvalidInput)
	}
	return r.Exists(ctx, schema.DictionaryEntries.TextNormalized.Bare(), text)
}

// ============================================================================
// SEARCH OPERATIONS
// ============================================================================

// Find выполняет поиск слов с фильтрацией, сортировкой и пагинацией.
func (r *DictionaryRepository) Find(ctx context.Context, f DictionaryFilter) ([]model.DictionaryEntry, error) {
	f.Normalize()

	b := r.SelectBuilder()
	var err error

	// Применяем фильтры
	b, err = r.applyFilters(b, f)
	if err != nil {
		return nil, err
	}

	// Применяем сортировку
	b = r.applySorting(b, f)

	// Применяем пагинацию
	b = b.Limit(uint64(f.Limit))
	if f.Offset > 0 {
		b = b.Offset(uint64(f.Offset))
	}

	return r.List(ctx, b)
}

// CountTotal возвращает общее количество записей по фильтру (без пагинации).
func (r *DictionaryRepository) CountTotal(ctx context.Context, f DictionaryFilter) (int64, error) {
	f.Normalize()

	b := r.SelectBuilder()
	var err error
	b, err = r.applyFilters(b, f)
	if err != nil {
		return 0, err
	}

	return r.Count(ctx, b)
}

// applyFilters добавляет условия WHERE к билдеру.
//
// Производительность:
//   - PartOfSpeech: требует индекса на senses(entry_id, part_of_speech)
//   - HasCard: требует индекса на cards(entry_id)
//   - Search: требует GIN индекса на text_normalized для триграмм
func (r *DictionaryRepository) applyFilters(b squirrel.SelectBuilder, f DictionaryFilter) (squirrel.SelectBuilder, error) {
	// 1. Фильтр по PartOfSpeech (через подзапрос EXISTS)
	// Оптимизация: EXISTS обычно быстрее JOIN для проверки наличия
	if f.PartOfSpeech != nil {
		// Используем корреляционный подзапрос с параметризованным запросом
		// ВАЖНО: используем ? — squirrel автоматически нумерует параметры ($1, $2, ...)
		b = b.Where(squirrel.Expr(
			fmt.Sprintf("EXISTS (SELECT 1 FROM %s s WHERE s.entry_id = dictionary_entries.id AND s.part_of_speech = ?)",
				schema.Senses.Name.String()),
			*f.PartOfSpeech,
		))
	}

	// 2. Фильтр по HasCard (через подзапрос EXISTS)
	// Оптимизация: EXISTS быстрее COUNT(*) > 0
	if f.HasCard != nil {
		cardsExistsSQL := fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s c WHERE c.entry_id = dictionary_entries.id)",
			schema.Cards.Name.String(),
		)
		if *f.HasCard {
			b = b.Where(squirrel.Expr(cardsExistsSQL))
		} else {
			b = b.Where(squirrel.Expr("NOT " + cardsExistsSQL))
		}
	}

	// 3. Фильтр по TopicIDs (через подзапрос EXISTS)
	// Оптимизация: EXISTS быстрее JOIN для проверки наличия
	// Используем индекс ix_dictionary_entry_topics_topic_id для производительности
	if len(f.TopicIDs) > 0 {
		// EXISTS (SELECT 1 FROM dictionary_entry_topics det WHERE det.entry_id = dictionary_entries.id AND det.topic_id IN (...))
		// Используем IN с подзапросом для совместимости с squirrel
		// Конвертируем []uuid.UUID в []any для использования в IN
		topicIDsAny := make([]any, len(f.TopicIDs))
		for i, id := range f.TopicIDs {
			topicIDsAny[i] = id
		}
		// Создаем плейсхолдеры для IN: $1, $2, $3, ...
		placeholders := make([]string, len(f.TopicIDs))
		for i := range f.TopicIDs {
			placeholders[i] = "?"
		}
		topicsExistsSQL := fmt.Sprintf(
			"EXISTS (SELECT 1 FROM %s det WHERE det.entry_id = dictionary_entries.id AND det.topic_id IN (%s))",
			schema.DictionaryEntryTopics.Name.String(),
			strings.Join(placeholders, ","),
		)
		// Передаем параметры через variadic arguments
		b = b.Where(squirrel.Expr(topicsExistsSQL, topicIDsAny...))
	}

	// 4. Поиск (Prefix для коротких слов, Trigram для длинных)
	if f.Search != "" {
		textCol := schema.DictionaryEntries.Text.Bare()
		queryLen := utf8.RuneCountInString(f.Search)

		if queryLen < MinSearchLength {
			// Prefix search для коротких запросов
			// Используем ILIKE с индексом для производительности
			// Рекомендуется индекс: CREATE INDEX idx_text_prefix ON dictionary_entries(text text_pattern_ops);
			b = b.Where(squirrel.ILike{textCol: f.Search + "%"})
		} else {
			// Fuzzy search через pg_trgm для длинных запросов
			// Используем оператор similarity (%)
			// Требует расширения: CREATE EXTENSION IF NOT EXISTS pg_trgm;
			// Рекомендуется GIN индекс: CREATE INDEX idx_text_trgm ON dictionary_entries USING GIN(text gin_trgm_ops);
			// Используем ? вместо $1, чтобы squirrel автоматически нумеровал параметры
			b = b.Where(squirrel.Expr(textCol+" % ?", f.Search))
		}
	}

	return b, nil
}

// applySorting применяет сортировку к запросу.
func (r *DictionaryRepository) applySorting(b squirrel.SelectBuilder, f DictionaryFilter) squirrel.SelectBuilder {
	textCol := schema.DictionaryEntries.Text.Bare()

	// A. Явная сортировка от пользователя
	if f.SortBy != nil {
		direction := "ASC"
		if f.SortDir != nil && *f.SortDir == model.SortDirDesc {
			direction = "DESC"
		}

		switch *f.SortBy {
		case model.SortFieldText:
			return b.OrderBy(textCol + " " + direction)
		case model.SortFieldCreatedAt:
			return b.OrderBy(schema.DictionaryEntries.CreatedAt.Bare() + " " + direction)
		case model.SortFieldUpdatedAt:
			return b.OrderBy(schema.DictionaryEntries.UpdatedAt.Bare() + " " + direction)
		}
	}

	// B. Сортировка по релевантности при поиске
	if f.Search != "" {
		queryLen := utf8.RuneCountInString(f.Search)
		if queryLen < MinSearchLength {
			// Короткие запросы: сначала короткие совпадения, потом алфавит
			return b.OrderBy("LENGTH("+textCol+") ASC", textCol+" ASC")
		}
		// Длинные запросы: сортировка по similarity distance
		// Используем OrderByClause с ? - squirrel автоматически преобразует ? в правильный $N
		// Проблема была в том, что orderArgs из squirrel.Expr конфликтовал с параметрами WHERE
		// Поэтому используем простой подход: передаем SQL строку с ? и значение напрямую
		return b.OrderByClause(fmt.Sprintf("%s <-> ? ASC", textCol), f.Search)
	}

	// C. Дефолтная сортировка: новые первыми
	return b.OrderBy(schema.DictionaryEntries.CreatedAt.Bare() + " DESC")
}

// ============================================================================
// WRITE OPERATIONS
// ============================================================================

// Create создает новое слово.
//
// Возвращает:
//   - ErrDuplicate: если слово с таким text_normalized уже существует
//   - ErrInvalidInput: если text или text_normalized пусты
func (r *DictionaryRepository) Create(ctx context.Context, entry *model.DictionaryEntry) (*model.DictionaryEntry, error) {
	if entry == nil {
		return nil, fmt.Errorf("%w: entry is required", database.ErrInvalidInput)
	}
	if err := base.ValidateString(entry.Text, "text"); err != nil {
		return nil, err
	}
	if err := base.ValidateString(entry.TextNormalized, "text_normalized"); err != nil {
		return nil, err
	}

	insert := r.InsertBuilder().
		Columns(schema.DictionaryEntries.InsertColumns()...).
		Values(entry.Text, entry.TextNormalized, entry.Notes)

	return r.InsertReturning(ctx, insert)
}

// CreateOrGet создает новое слово или возвращает существующее по text_normalized.
//
// Использует атомарную операцию INSERT ... ON CONFLICT для предотвращения race condition.
// Это идемпотентная операция — безопасна для повторных вызовов.
//
// Производительность:
//   - Требует UNIQUE индекса на text_normalized
//   - Оптимизирован для конкурентных вставок
//   - Использует минимальное обновление для минимизации блокировок
func (r *DictionaryRepository) CreateOrGet(ctx context.Context, entry *model.DictionaryEntry) (*model.DictionaryEntry, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	if entry == nil {
		return nil, fmt.Errorf("%w: entry is required", database.ErrInvalidInput)
	}
	if err := base.ValidateString(entry.Text, "text"); err != nil {
		return nil, err
	}
	if err := base.ValidateString(entry.TextNormalized, "text_normalized"); err != nil {
		return nil, err
	}

	// ON CONFLICT ... DO UPDATE SET id = EXCLUDED.id гарантирует возврат записи
	// Используем минимальное обновление (id = id) чтобы сработал RETURNING
	// Это минимизирует блокировки и overhead при конфликтах
	insert := r.InsertBuilder().
		Columns(schema.DictionaryEntries.InsertColumns()...).
		Values(entry.Text, entry.TextNormalized, entry.Notes).
		Suffix("ON CONFLICT (text_normalized) DO UPDATE SET id = dictionary_entries.id RETURNING *")

	sql, args, err := insert.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	// Используем QueryRowRaw из базового репозитория, который сам обрабатывает таймауты
	var result model.DictionaryEntry
	if err := r.QueryRowRaw(ctx, &result, sql, args...); err != nil {
		return nil, err
	}

	return &result, nil
}

// Update обновляет словарную запись.
//
// Возвращает:
//   - ErrNotFound: если запись не найдена
//   - ErrInvalidInput: если id пустой или entry nil
func (r *DictionaryRepository) Update(ctx context.Context, id uuid.UUID, entry *model.DictionaryEntry) (*model.DictionaryEntry, error) {
	if entry == nil {
		return nil, fmt.Errorf("%w: entry is required", database.ErrInvalidInput)
	}
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	if err := base.ValidateString(entry.Text, "text"); err != nil {
		return nil, err
	}
	if err := base.ValidateString(entry.TextNormalized, "text_normalized"); err != nil {
		return nil, err
	}

	update := r.UpdateBuilder().
		Set("text", entry.Text).
		Set("text_normalized", entry.TextNormalized).
		Set("notes", entry.Notes).
		Where(squirrel.Eq{schema.DictionaryEntries.ID.Bare(): id})

	return r.Base.Update(ctx, update)
}

// Delete удаляет словарную запись.
// CASCADE удалит связанные senses, examples, translations и т.д.
func (r *DictionaryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.DictionaryEntries.ID.Bare(), id)
}
