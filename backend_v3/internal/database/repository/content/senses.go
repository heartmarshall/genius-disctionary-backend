package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository/base"
	"github.com/heartmarshall/my-english/internal/database/schema"
	"github.com/heartmarshall/my-english/internal/model"
)

// ============================================================================
// SENSES REPOSITORY
// ============================================================================

// SenseRepository предоставляет методы для работы со смыслами слов.
type SenseRepository struct {
	*base.Base[model.Sense]
}

// NewSenseRepository создаёт новый репозиторий смыслов.
func NewSenseRepository(q database.Querier) *SenseRepository {
	return &SenseRepository{
		Base: base.MustNewBase[model.Sense](q, base.Config{
			Table:   schema.Senses.Name.String(),
			Columns: schema.Senses.Columns(),
		}),
	}
}

// GetByID получает смысл по ID.
func (r *SenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Sense, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Senses.ID.Bare(), id)
}

// ListByEntryIDs получает смыслы для списка записей словаря.
func (r *SenseRepository) ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Sense, error) {
	if len(entryIDs) == 0 {
		return []model.Sense{}, nil
	}
	return r.ListByUUIDs(ctx, schema.Senses.EntryID.Bare(), entryIDs)
}

// Create создает новый смысл.
func (r *SenseRepository) Create(ctx context.Context, sense *model.Sense) (*model.Sense, error) {
	if sense == nil {
		return nil, fmt.Errorf("%w: sense is required", database.ErrInvalidInput)
	}
	if err := base.ValidateUUID(sense.EntryID, "entry_id"); err != nil {
		return nil, err
	}
	if err := base.ValidateString(sense.SourceSlug, "source_slug"); err != nil {
		return nil, err
	}

	insert := r.InsertBuilder().
		Columns(schema.Senses.InsertColumns()...).
		Values(
			sense.EntryID,
			sense.Definition,
			sense.PartOfSpeech,
			sense.SourceSlug,
			sense.CefrLevel,
		)

	return r.InsertReturning(ctx, insert)
}

// BatchCreate создает несколько смыслов за один запрос.
//
// Производительность:
//   - Использует batch insert для минимизации round-trips к БД
//   - Автоматически разбивает большие батчи на чанки
//   - Рекомендуется использовать в транзакциях для атомарности
func (r *SenseRepository) BatchCreate(ctx context.Context, senses []model.Sense) ([]model.Sense, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	if len(senses) == 0 {
		return []model.Sense{}, nil
	}

	// Валидация всех элементов перед вставкой
	// Это позволяет вернуть ошибку до начала транзакции
	for i := range senses {
		if err := base.ValidateUUID(senses[i].EntryID, "entry_id"); err != nil {
			return nil, fmt.Errorf("sense[%d]: %w", i, err)
		}
		if err := base.ValidateString(senses[i].SourceSlug, "source_slug"); err != nil {
			return nil, fmt.Errorf("sense[%d]: %w", i, err)
		}
	}

	columns := schema.Senses.InsertColumns()
	valuesFunc := func(s model.Sense) []any {
		return []any{
			s.EntryID,
			s.Definition,
			s.PartOfSpeech,
			s.SourceSlug,
			s.CefrLevel,
		}
	}

	return r.BatchInsertReturning(ctx, columns, senses, valuesFunc)
}

// Delete удаляет смысл.
func (r *SenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Senses.ID.Bare(), id)
}
