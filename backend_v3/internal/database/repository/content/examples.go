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
// EXAMPLES REPOSITORY
// ============================================================================

// ExampleRepository предоставляет методы для работы с примерами.
type ExampleRepository struct {
	*base.Base[model.Example]
}

// NewExampleRepository создаёт новый репозиторий примеров.
func NewExampleRepository(q database.Querier) *ExampleRepository {
	return &ExampleRepository{
		Base: base.MustNewBase[model.Example](q, base.Config{
			Table:   schema.Examples.Name.String(),
			Columns: schema.Examples.Columns(),
		}),
	}
}

// GetByID получает пример по ID.
func (r *ExampleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Example, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Examples.ID.Bare(), id)
}

// ListBySenseIDs получает примеры для списка смыслов.
func (r *ExampleRepository) ListBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]model.Example, error) {
	if len(senseIDs) == 0 {
		return []model.Example{}, nil
	}
	return r.ListByUUIDs(ctx, schema.Examples.SenseID.Bare(), senseIDs)
}

// BatchCreate создает несколько примеров за один запрос.
func (r *ExampleRepository) BatchCreate(ctx context.Context, examples []model.Example) ([]model.Example, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	if len(examples) == 0 {
		return []model.Example{}, nil
	}

	// Валидация всех элементов перед вставкой
	for i := range examples {
		if err := base.ValidateUUID(examples[i].SenseID, "sense_id"); err != nil {
			return nil, fmt.Errorf("example[%d]: %w", i, err)
		}
		if err := base.ValidateString(examples[i].Sentence, "sentence"); err != nil {
			return nil, fmt.Errorf("example[%d]: %w", i, err)
		}
		if err := base.ValidateString(examples[i].SourceSlug, "source_slug"); err != nil {
			return nil, fmt.Errorf("example[%d]: %w", i, err)
		}
	}

	columns := schema.Examples.InsertColumns()
	valuesFunc := func(e model.Example) []any {
		return []any{
			e.SenseID,
			e.Sentence,
			e.Translation,
			e.SourceSlug,
		}
	}

	return r.BatchInsertReturning(ctx, columns, examples, valuesFunc)
}

// Delete удаляет пример.
func (r *ExampleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Examples.ID.Bare(), id)
}
