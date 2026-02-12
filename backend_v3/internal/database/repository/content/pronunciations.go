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
// PRONUNCIATIONS REPOSITORY
// ============================================================================

// PronunciationRepository предоставляет методы для работы с произношениями.
type PronunciationRepository struct {
	*base.Base[model.Pronunciation]
}

// NewPronunciationRepository создаёт новый репозиторий произношений.
func NewPronunciationRepository(q database.Querier) *PronunciationRepository {
	return &PronunciationRepository{
		Base: base.MustNewBase[model.Pronunciation](q, base.Config{
			Table:   schema.Pronunciations.Name.String(),
			Columns: schema.Pronunciations.Columns(),
		}),
	}
}

// GetByID получает произношение по ID.
func (r *PronunciationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Pronunciation, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Pronunciations.ID.Bare(), id)
}

// ListByEntryIDs получает произношения для списка записей словаря.
func (r *PronunciationRepository) ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Pronunciation, error) {
	if len(entryIDs) == 0 {
		return []model.Pronunciation{}, nil
	}
	return r.ListByUUIDs(ctx, schema.Pronunciations.EntryID.Bare(), entryIDs)
}

// BatchCreate создает несколько произношений за один запрос.
func (r *PronunciationRepository) BatchCreate(ctx context.Context, pronunciations []model.Pronunciation) ([]model.Pronunciation, error) {
	if len(pronunciations) == 0 {
		return []model.Pronunciation{}, nil
	}

	// Валидация всех элементов
	for i := range pronunciations {
		if err := base.ValidateUUID(pronunciations[i].EntryID, "entry_id"); err != nil {
			return nil, fmt.Errorf("pronunciation[%d]: %w", i, err)
		}
		if err := base.ValidateString(pronunciations[i].SourceSlug, "source_slug"); err != nil {
			return nil, fmt.Errorf("pronunciation[%d]: %w", i, err)
		}
	}

	columns := schema.Pronunciations.InsertColumns()
	valuesFunc := func(p model.Pronunciation) []any {
		return []any{
			p.EntryID,
			p.AudioURL,
			p.Transcription,
			p.Region,
			p.SourceSlug,
		}
	}

	return r.BatchInsertReturning(ctx, columns, pronunciations, valuesFunc)
}

// Delete удаляет произношение.
func (r *PronunciationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Pronunciations.ID.Bare(), id)
}
