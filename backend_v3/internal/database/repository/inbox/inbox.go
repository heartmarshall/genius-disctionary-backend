// Package inbox содержит репозиторий для работы с inbox (списком слов для изучения).
package inbox

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

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	// MaxInboxItems — максимальное количество элементов в inbox.
	// Предотвращает слишком большие выборки.
	MaxInboxItems = 1000
)

// ============================================================================
// REPOSITORY
// ============================================================================

// InboxRepository предоставляет методы для работы с inbox.
type InboxRepository struct {
	*base.Base[model.InboxItem]
}

// NewInboxRepository создаёт новый репозиторий inbox.
func NewInboxRepository(q database.Querier) *InboxRepository {
	return &InboxRepository{
		Base: base.MustNewBase[model.InboxItem](q, base.Config{
			Table:   schema.InboxItems.Name.String(),
			Columns: schema.InboxItems.Columns(),
		}),
	}
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

// GetByID получает элемент inbox по ID.
func (r *InboxRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.InboxItem, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.InboxItems.ID.Bare(), id)
}

// ListAll возвращает все элементы inbox, отсортированные по дате создания (новые первыми).
//
// Внимание: для больших объемов данных используйте ListPaginated.
func (r *InboxRepository) ListAll(ctx context.Context) ([]model.InboxItem, error) {
	query := r.SelectBuilder().
		OrderBy(schema.InboxItems.CreatedAt.Bare() + " DESC").
		Limit(MaxInboxItems)

	return r.Base.List(ctx, query)
}

// ListPaginated возвращает элементы inbox с пагинацией.
//
// Производительность:
//   - Использует LIMIT/OFFSET для пагинации
//   - Для больших offset (>10000) рассмотрите курсорную пагинацию
//   - Рекомендуется индекс на (created_at DESC) для быстрой сортировки
func (r *InboxRepository) ListPaginated(ctx context.Context, limit, offset int) ([]model.InboxItem, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	// Нормализуем параметры пагинации
	if limit <= 0 {
		limit = 50
	}
	if limit > MaxInboxItems {
		limit = MaxInboxItems
	}
	if offset < 0 {
		offset = 0
	}

	query := r.SelectBuilder().
		OrderBy(schema.InboxItems.CreatedAt.Bare() + " DESC").
		Limit(uint64(limit)).
		Offset(uint64(offset))

	return r.Base.List(ctx, query)
}

// Count возвращает общее количество элементов в inbox.
func (r *InboxRepository) Count(ctx context.Context) (int64, error) {
	return r.CountAll(ctx)
}

// ============================================================================
// WRITE OPERATIONS
// ============================================================================

// Create создает новый элемент inbox.
func (r *InboxRepository) Create(ctx context.Context, item *model.InboxItem) (*model.InboxItem, error) {
	if item == nil {
		return nil, fmt.Errorf("%w: item is required", database.ErrInvalidInput)
	}
	if err := base.ValidateString(item.Text, "text"); err != nil {
		return nil, err
	}

	insert := r.InsertBuilder().
		Columns(schema.InboxItems.InsertColumns()...).
		Values(item.Text, item.Context)

	return r.InsertReturning(ctx, insert)
}

// Delete удаляет элемент inbox по ID.
func (r *InboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.InboxItems.ID.Bare(), id)
}

// DeleteAll удаляет все элементы inbox.
// Возвращает количество удалённых записей.
func (r *InboxRepository) DeleteAll(ctx context.Context) (int64, error) {
	return r.DeleteWhere(ctx, squirrel.Expr("1=1"))
}
