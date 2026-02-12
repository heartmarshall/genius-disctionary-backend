package topics

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository/base"
	"github.com/heartmarshall/my-english/internal/database/schema"
	"github.com/heartmarshall/my-english/internal/model"
)

// TopicWithEntryID используется для DataLoaders (M2M связь).
type TopicWithEntryID struct {
	model.Topic
	EntryID uuid.UUID `db:"entry_id"`
}

// TopicRepository предоставляет методы для работы с топиками.
type TopicRepository struct {
	*base.Base[model.Topic]
}

// NewTopicRepository создаёт новый репозиторий топиков.
func NewTopicRepository(q database.Querier) *TopicRepository {
	return &TopicRepository{
		Base: base.MustNewBase[model.Topic](q, base.Config{
			Table:   schema.Topics.Name.String(),
			Columns: schema.Topics.Columns(),
		}),
	}
}

// GetByID возвращает топик по ID.
func (r *TopicRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Topic, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Topics.ID.Bare(), id)
}

// Create создает новый топик.
func (r *TopicRepository) Create(ctx context.Context, topic *model.Topic) (*model.Topic, error) {
	if topic == nil {
		return nil, fmt.Errorf("%w: topic is required", database.ErrInvalidInput)
	}
	if err := base.ValidateString(topic.Name, "name"); err != nil {
		return nil, err
	}

	insert := r.InsertBuilder().
		Columns(schema.Topics.InsertColumns()...).
		Values(topic.Name, topic.Description)

	return r.InsertReturning(ctx, insert)
}

// Update обновляет топик.
func (r *TopicRepository) Update(ctx context.Context, id uuid.UUID, topic *model.Topic) (*model.Topic, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}

	update := r.UpdateBuilder().
		Set("name", topic.Name).
		Set("description", topic.Description).
		Where(squirrel.Eq{schema.Topics.ID.Bare(): id})

	return r.Base.Update(ctx, update)
}

// Delete удаляет топик по ID.
func (r *TopicRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Topics.ID.Bare(), id)
}

// ListAll возвращает все топики.
func (r *TopicRepository) ListAll(ctx context.Context) ([]model.Topic, error) {
	query := r.SelectBuilder().OrderBy(schema.Topics.NameColumn.Bare() + " ASC")
	return r.List(ctx, query)
}

// ============================================================================
// JUNCTION TABLE OPERATIONS (M2M)
// ============================================================================

// BindToEntry привязывает топик к слову.
func (r *TopicRepository) BindToEntry(ctx context.Context, entryID, topicID uuid.UUID) error {
	if err := base.ValidateUUID(entryID, "entry_id"); err != nil {
		return err
	}
	if err := base.ValidateUUID(topicID, "topic_id"); err != nil {
		return err
	}

	// Используем ON CONFLICT DO NOTHING, чтобы избежать ошибок дубликатов
	query := base.Builder().
		Insert(schema.DictionaryEntryTopics.Name.String()).
		Columns(schema.DictionaryEntryTopics.InsertColumns()...).
		Values(entryID, topicID).
		Suffix("ON CONFLICT (entry_id, topic_id) DO NOTHING")

	sql, args, err := query.ToSql()
	if err != nil {
		return database.WrapDBError(err)
	}

	_, err = r.Q().Exec(ctx, sql, args...)
	return database.WrapDBError(err)
}

// UnbindFromEntry удаляет связь топика со словом.
func (r *TopicRepository) UnbindFromEntry(ctx context.Context, entryID, topicID uuid.UUID) error {
	query := base.Builder().
		Delete(schema.DictionaryEntryTopics.Name.String()).
		Where(squirrel.Eq{
			schema.DictionaryEntryTopics.EntryID.Bare(): entryID,
			schema.DictionaryEntryTopics.TopicID.Bare(): topicID,
		})

	sql, args, err := query.ToSql()
	if err != nil {
		return database.WrapDBError(err)
	}

	_, err = r.Q().Exec(ctx, sql, args...)
	return database.WrapDBError(err)
}

// UnbindAllFromEntry удаляет все топики у слова (используется при обновлении).
func (r *TopicRepository) UnbindAllFromEntry(ctx context.Context, entryID uuid.UUID) error {
	query := base.Builder().
		Delete(schema.DictionaryEntryTopics.Name.String()).
		Where(squirrel.Eq{schema.DictionaryEntryTopics.EntryID.Bare(): entryID})

	sql, args, err := query.ToSql()
	if err != nil {
		return database.WrapDBError(err)
	}

	_, err = r.Q().Exec(ctx, sql, args...)
	return database.WrapDBError(err)
}

// ListByEntryIDs возвращает топики для списка слов.
// Используется в DataLoader. Возвращает расширенную структуру с EntryID.
func (r *TopicRepository) ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]TopicWithEntryID, error) {
	if len(entryIDs) == 0 {
		return []TopicWithEntryID{}, nil
	}

	// SELECT t.*, det.entry_id
	// FROM topics t
	// JOIN dictionary_entry_topics det ON t.id = det.topic_id
	// WHERE det.entry_id IN (...)
	query := r.SelectBuilder().
		Column(schema.DictionaryEntryTopics.EntryID.Qualified()). // Добавляем entry_id в выборку
		Join(fmt.Sprintf("%s ON %s = %s",
			schema.DictionaryEntryTopics.Name,
			schema.Topics.ID.Qualified(),
			schema.DictionaryEntryTopics.TopicID.Qualified(),
		)).
		Where(squirrel.Eq{schema.DictionaryEntryTopics.EntryID.Qualified(): entryIDs})

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	var result []TopicWithEntryID
	// Здесь нам придется использовать QueryRaw из base или scan вручную,
	// так как base.List возвращает []model.Topic.
	// Используем pgxscan через Querier напрямую.
	// Но у нас нет прямого доступа к pgxscan здесь без импорта.
	// Воспользуемся методом QueryRaw базового репозитория, он удобен.

	err = r.Base.QueryRaw(ctx, &result, sql, args...)
	if err != nil {
		return nil, err
	}

	return result, nil
}
