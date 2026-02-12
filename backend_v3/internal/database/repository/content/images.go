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
// IMAGES REPOSITORY
// ============================================================================

// ImageRepository предоставляет методы для работы с изображениями.
type ImageRepository struct {
	*base.Base[model.Image]
}

// NewImageRepository создаёт новый репозиторий изображений.
func NewImageRepository(q database.Querier) *ImageRepository {
	return &ImageRepository{
		Base: base.MustNewBase[model.Image](q, base.Config{
			Table:   schema.Images.Name.String(),
			Columns: schema.Images.Columns(),
		}),
	}
}

// GetByID получает изображение по ID.
func (r *ImageRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Image, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Images.ID.Bare(), id)
}

// ListByEntryIDs получает изображения для списка записей словаря.
func (r *ImageRepository) ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Image, error) {
	if len(entryIDs) == 0 {
		return []model.Image{}, nil
	}
	return r.ListByUUIDs(ctx, schema.Images.EntryID.Bare(), entryIDs)
}

// BatchCreate создает несколько изображений за один запрос.
func (r *ImageRepository) BatchCreate(ctx context.Context, images []model.Image) ([]model.Image, error) {
	if len(images) == 0 {
		return []model.Image{}, nil
	}

	// Валидация всех элементов
	for i := range images {
		if err := base.ValidateUUID(images[i].EntryID, "entry_id"); err != nil {
			return nil, fmt.Errorf("image[%d]: %w", i, err)
		}
		if err := base.ValidateString(images[i].URL, "url"); err != nil {
			return nil, fmt.Errorf("image[%d]: %w", i, err)
		}
		if err := base.ValidateString(images[i].SourceSlug, "source_slug"); err != nil {
			return nil, fmt.Errorf("image[%d]: %w", i, err)
		}
	}

	columns := schema.Images.InsertColumns()
	valuesFunc := func(img model.Image) []any {
		return []any{
			img.EntryID,
			img.URL,
			img.Caption,
			img.SourceSlug,
		}
	}

	return r.BatchInsertReturning(ctx, columns, images, valuesFunc)
}

// Delete удаляет изображение.
func (r *ImageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Images.ID.Bare(), id)
}
