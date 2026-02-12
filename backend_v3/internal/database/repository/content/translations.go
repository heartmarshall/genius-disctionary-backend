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
// TRANSLATIONS REPOSITORY
// ============================================================================

// TranslationRepository предоставляет методы для работы с переводами.
type TranslationRepository struct {
	*base.Base[model.Translation]
}

// NewTranslationRepository создаёт новый репозиторий переводов.
func NewTranslationRepository(q database.Querier) *TranslationRepository {
	return &TranslationRepository{
		Base: base.MustNewBase[model.Translation](q, base.Config{
			Table:   schema.Translations.Name.String(),
			Columns: schema.Translations.Columns(),
		}),
	}
}

// GetByID получает перевод по ID.
func (r *TranslationRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Translation, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Translations.ID.Bare(), id)
}

// ListBySenseIDs получает переводы для списка смыслов.
func (r *TranslationRepository) ListBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]model.Translation, error) {
	if len(senseIDs) == 0 {
		return []model.Translation{}, nil
	}
	return r.ListByUUIDs(ctx, schema.Translations.SenseID.Bare(), senseIDs)
}

// BatchCreate создает несколько переводов за один запрос.
func (r *TranslationRepository) BatchCreate(ctx context.Context, translations []model.Translation) ([]model.Translation, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	if len(translations) == 0 {
		return []model.Translation{}, nil
	}

	// Валидация всех элементов перед вставкой
	for i := range translations {
		if err := base.ValidateUUID(translations[i].SenseID, "sense_id"); err != nil {
			return nil, fmt.Errorf("translation[%d]: %w", i, err)
		}
		if err := base.ValidateString(translations[i].Text, "text"); err != nil {
			return nil, fmt.Errorf("translation[%d]: %w", i, err)
		}
		if err := base.ValidateString(translations[i].SourceSlug, "source_slug"); err != nil {
			return nil, fmt.Errorf("translation[%d]: %w", i, err)
		}
	}

	columns := schema.Translations.InsertColumns()
	valuesFunc := func(t model.Translation) []any {
		return []any{
			t.SenseID,
			t.Text,
			t.SourceSlug,
		}
	}

	return r.BatchInsertReturning(ctx, columns, translations, valuesFunc)
}

// Delete удаляет перевод.
func (r *TranslationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Translations.ID.Bare(), id)
}
