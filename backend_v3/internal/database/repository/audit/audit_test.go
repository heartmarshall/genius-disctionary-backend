package audit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/testutil"
	"github.com/heartmarshall/my-english/internal/model"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func TestAuditRepository_Create(t *testing.T) {
	auditID := uuid.New()
	entityID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		audit   *model.AuditRecord
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful creation",
			audit: &model.AuditRecord{
				EntityType: model.EntityEntry,
				EntityID:   &entityID,
				Action:     model.ActionCreate,
				Changes:    model.JSON{"text": "hello"},
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entity_type", "entity_id", "action", "changes", "created_at"}).
					AddRow(auditID, model.EntityEntry, &entityID, model.ActionCreate, []byte(`{"text":"hello"}`), now)
				mock.ExpectQuery(`INSERT INTO audit_records`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "successful creation without changes",
			audit: &model.AuditRecord{
				EntityType: model.EntityCard,
				EntityID:   &entityID,
				Action:     model.ActionDelete,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entity_type", "entity_id", "action", "changes", "created_at"}).
					AddRow(auditID, model.EntityCard, &entityID, model.ActionDelete, nil, now)
				mock.ExpectQuery(`INSERT INTO audit_records`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			audit:   nil,
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "empty entity_type",
			audit: &model.AuditRecord{
				EntityType: "",
				EntityID:   &entityID,
				Action:     model.ActionCreate,
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "nil entity_id",
			audit: &model.AuditRecord{
				EntityType: model.EntityEntry,
				EntityID:   nil,
				Action:     model.ActionCreate,
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "zero entity_id",
			audit: func() *model.AuditRecord {
				zeroID := uuid.UUID{}
				return &model.AuditRecord{
					EntityType: model.EntityEntry,
					EntityID:   &zeroID,
					Action:     model.ActionCreate,
				}
			}(),
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "empty action",
			audit: &model.AuditRecord{
				EntityType: model.EntityEntry,
				EntityID:   &entityID,
				Action:     "",
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewAuditRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Create(ctx, tt.audit)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Create() returned nil result")
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestNewCreateRecord(t *testing.T) {
	entityID := uuid.New()
	changes := model.JSON{"text": "hello"}

	record := NewCreateRecord(model.EntityEntry, entityID, changes)

	if record.EntityType != model.EntityEntry {
		t.Errorf("NewCreateRecord() entity_type = %v, want %v", record.EntityType, model.EntityEntry)
	}
	if record.EntityID == nil || *record.EntityID != entityID {
		t.Errorf("NewCreateRecord() entity_id = %v, want %v", record.EntityID, entityID)
	}
	if record.Action != model.ActionCreate {
		t.Errorf("NewCreateRecord() action = %v, want %v", record.Action, model.ActionCreate)
	}
	if record.Changes["text"] != "hello" {
		t.Errorf("NewCreateRecord() changes = %v, want text=hello", record.Changes)
	}
}

func TestNewUpdateRecord(t *testing.T) {
	entityID := uuid.New()
	changes := model.JSON{"old": "foo", "new": "bar"}

	record := NewUpdateRecord(model.EntitySense, entityID, changes)

	if record.EntityType != model.EntitySense {
		t.Errorf("NewUpdateRecord() entity_type = %v, want %v", record.EntityType, model.EntitySense)
	}
	if record.EntityID == nil || *record.EntityID != entityID {
		t.Errorf("NewUpdateRecord() entity_id = %v, want %v", record.EntityID, entityID)
	}
	if record.Action != model.ActionUpdate {
		t.Errorf("NewUpdateRecord() action = %v, want %v", record.Action, model.ActionUpdate)
	}
}

func TestNewDeleteRecord(t *testing.T) {
	entityID := uuid.New()

	record := NewDeleteRecord(model.EntityCard, entityID)

	if record.EntityType != model.EntityCard {
		t.Errorf("NewDeleteRecord() entity_type = %v, want %v", record.EntityType, model.EntityCard)
	}
	if record.EntityID == nil || *record.EntityID != entityID {
		t.Errorf("NewDeleteRecord() entity_id = %v, want %v", record.EntityID, entityID)
	}
	if record.Action != model.ActionDelete {
		t.Errorf("NewDeleteRecord() action = %v, want %v", record.Action, model.ActionDelete)
	}
	if record.Changes != nil {
		t.Errorf("NewDeleteRecord() changes = %v, want nil", record.Changes)
	}
}
