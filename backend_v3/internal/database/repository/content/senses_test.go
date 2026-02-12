package content

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/testutil"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func TestSenseRepository_GetByID(t *testing.T) {
	senseID := uuid.New()
	entryID := uuid.New()
	now := time.Now()
	pos := model.PosNoun

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "found",
			id:   senseID,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "definition", "part_of_speech", "source_slug", "cefr_level", "created_at"}).
					AddRow(senseID, entryID, nil, &pos, "freedict", nil, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   senseID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:    "zero uuid",
			id:      uuid.UUID{},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewSenseRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.GetByID(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("GetByID() returned nil result")
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestSenseRepository_ListByEntryIDs(t *testing.T) {
	senseID := uuid.New()
	entryID := uuid.New()
	now := time.Now()
	pos := model.PosNoun

	tests := []struct {
		name     string
		entryIDs []uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "returns senses",
			entryIDs: []uuid.UUID{entryID},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "definition", "part_of_speech", "source_slug", "cefr_level", "created_at"}).
					AddRow(senseID, entryID, nil, &pos, "freedict", nil, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:     "empty input returns empty",
			entryIDs: []uuid.UUID{},
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "nil input returns empty",
			entryIDs: nil,
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantLen:  0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewSenseRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListByEntryIDs(ctx, tt.entryIDs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListByEntryIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("ListByEntryIDs() returned %d items, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestSenseRepository_Create(t *testing.T) {
	senseID := uuid.New()
	entryID := uuid.New()
	now := time.Now()
	pos := model.PosNoun

	tests := []struct {
		name    string
		sense   *model.Sense
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful creation",
			sense: &model.Sense{
				EntryID:    entryID,
				SourceSlug: "freedict",
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "definition", "part_of_speech", "source_slug", "cefr_level", "created_at"}).
					AddRow(senseID, entryID, nil, &pos, "freedict", nil, now)
				mock.ExpectQuery(`INSERT INTO senses`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			sense:   nil,
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "zero entry_id",
			sense: &model.Sense{
				EntryID:    uuid.UUID{},
				SourceSlug: "freedict",
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "empty source_slug",
			sense: &model.Sense{
				EntryID:    entryID,
				SourceSlug: "",
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewSenseRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Create(ctx, tt.sense)

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

func TestSenseRepository_BatchCreate(t *testing.T) {
	senseID := uuid.New()
	entryID := uuid.New()
	now := time.Now()
	pos := model.PosNoun

	tests := []struct {
		name    string
		senses  []model.Sense
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name: "successful batch create",
			senses: []model.Sense{
				{EntryID: entryID, SourceSlug: "freedict"},
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "definition", "part_of_speech", "source_slug", "cefr_level", "created_at"}).
					AddRow(senseID, entryID, nil, &pos, "freedict", nil, now)
				mock.ExpectQuery(`INSERT INTO senses`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty input returns empty",
			senses:  []model.Sense{},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "validation error - zero entry_id",
			senses: []model.Sense{
				{EntryID: uuid.UUID{}, SourceSlug: "freedict"},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "validation error - empty source_slug",
			senses: []model.Sense{
				{EntryID: entryID, SourceSlug: ""},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewSenseRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.BatchCreate(ctx, tt.senses)

			if (err != nil) != tt.wantErr {
				t.Errorf("BatchCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(result) != tt.wantLen {
				t.Errorf("BatchCreate() returned %d items, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestSenseRepository_Delete(t *testing.T) {
	senseID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   senseID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM senses`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   senseID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM senses`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: true,
		},
		{
			name:    "zero uuid",
			id:      uuid.UUID{},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewSenseRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			err := repo.Delete(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}
