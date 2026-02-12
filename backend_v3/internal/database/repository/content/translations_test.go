package content

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/testutil"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func TestTranslationRepository_GetByID(t *testing.T) {
	translationID := uuid.New()
	senseID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "found",
			id:   translationID,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "sense_id", "text", "source_slug"}).
					AddRow(translationID, senseID, "Привет", "freedict")
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   translationID,
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
			repo := NewTranslationRepository(querier)
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

func TestTranslationRepository_ListBySenseIDs(t *testing.T) {
	translationID := uuid.New()
	senseID := uuid.New()

	tests := []struct {
		name     string
		senseIDs []uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "returns translations",
			senseIDs: []uuid.UUID{senseID},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "sense_id", "text", "source_slug"}).
					AddRow(translationID, senseID, "Привет", "freedict")
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:     "empty input returns empty",
			senseIDs: []uuid.UUID{},
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantLen:  0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTranslationRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListBySenseIDs(ctx, tt.senseIDs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListBySenseIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("ListBySenseIDs() returned %d items, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTranslationRepository_BatchCreate(t *testing.T) {
	translationID := uuid.New()
	senseID := uuid.New()

	tests := []struct {
		name         string
		translations []model.Translation
		setup        func(mock pgxmock.PgxPoolIface)
		wantLen      int
		wantErr      bool
	}{
		{
			name: "successful batch create",
			translations: []model.Translation{
				{SenseID: senseID, Text: "Привет", SourceSlug: "freedict"},
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "sense_id", "text", "source_slug"}).
					AddRow(translationID, senseID, "Привет", "freedict")
				mock.ExpectQuery(`INSERT INTO translations`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:         "empty input returns empty",
			translations: []model.Translation{},
			setup:        func(mock pgxmock.PgxPoolIface) {},
			wantLen:      0,
			wantErr:      false,
		},
		{
			name: "validation error - zero sense_id",
			translations: []model.Translation{
				{SenseID: uuid.UUID{}, Text: "Hello", SourceSlug: "freedict"},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "validation error - empty text",
			translations: []model.Translation{
				{SenseID: senseID, Text: "", SourceSlug: "freedict"},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "validation error - empty source_slug",
			translations: []model.Translation{
				{SenseID: senseID, Text: "Hello", SourceSlug: ""},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTranslationRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.BatchCreate(ctx, tt.translations)

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

func TestTranslationRepository_Delete(t *testing.T) {
	translationID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   translationID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM translations`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   translationID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM translations`).
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
			repo := NewTranslationRepository(querier)
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
