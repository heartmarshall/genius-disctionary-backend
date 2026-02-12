// Package audit содержит репозиторий для работы с аудит-логами.
package audit

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
// REPOSITORY
// ============================================================================

// AuditRepository предоставляет методы для работы с аудит-логами.
// Аудит-логи используются для отслеживания изменений сущностей в системе.
type AuditRepository struct {
	*base.Base[model.AuditRecord]
}

// NewAuditRepository создаёт новый репозиторий аудит-логов.
func NewAuditRepository(q database.Querier) *AuditRepository {
	return &AuditRepository{
		Base: base.MustNewBase[model.AuditRecord](q, base.Config{
			Table:   schema.AuditRecords.Name.String(),
			Columns: schema.AuditRecords.Columns(),
		}),
	}
}

// ============================================================================
// WRITE OPERATIONS
// ============================================================================

// Create создает новую запись аудита.
//
// Параметры:
//   - audit.EntityType: тип сущности (ENTRY, SENSE, CARD и т.д.)
//   - audit.EntityID: ID сущности (не может быть nil или zero UUID)
//   - audit.Action: действие (CREATE, UPDATE, DELETE)
//   - audit.Changes: опциональные детали изменений (JSONB)
func (r *AuditRepository) Create(ctx context.Context, audit *model.AuditRecord) (*model.AuditRecord, error) {
	if audit == nil {
		return nil, fmt.Errorf("%w: audit is required", database.ErrInvalidInput)
	}
	if audit.EntityType == "" {
		return nil, fmt.Errorf("%w: entity_type is required", database.ErrInvalidInput)
	}
	if audit.EntityID == nil {
		return nil, fmt.Errorf("%w: entity_id is required", database.ErrInvalidInput)
	}
	if err := base.ValidateUUID(*audit.EntityID, "entity_id"); err != nil {
		return nil, err
	}
	if audit.Action == "" {
		return nil, fmt.Errorf("%w: action is required", database.ErrInvalidInput)
	}

	insert := r.InsertBuilder().
		Columns(schema.AuditRecords.InsertColumns()...).
		Values(
			audit.EntityType,
			audit.EntityID,
			audit.Action,
			audit.Changes,
		)

	return r.InsertReturning(ctx, insert)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// NewCreateRecord создаёт запись аудита для действия CREATE.
func NewCreateRecord(entityType model.EntityType, entityID uuid.UUID, changes model.JSON) *model.AuditRecord {
	return &model.AuditRecord{
		EntityType: entityType,
		EntityID:   &entityID,
		Action:     model.ActionCreate,
		Changes:    changes,
	}
}

// NewUpdateRecord создаёт запись аудита для действия UPDATE.
func NewUpdateRecord(entityType model.EntityType, entityID uuid.UUID, changes model.JSON) *model.AuditRecord {
	return &model.AuditRecord{
		EntityType: entityType,
		EntityID:   &entityID,
		Action:     model.ActionUpdate,
		Changes:    changes,
	}
}

// NewDeleteRecord создаёт запись аудита для действия DELETE.
func NewDeleteRecord(entityType model.EntityType, entityID uuid.UUID) *model.AuditRecord {
	return &model.AuditRecord{
		EntityType: entityType,
		EntityID:   &entityID,
		Action:     model.ActionDelete,
	}
}
