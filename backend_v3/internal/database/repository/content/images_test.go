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

func TestImageRepository_GetByID(t *testing.T) {
	imageID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "found",
			id:   imageID,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "url", "caption", "source_slug"}).
					AddRow(imageID, entryID, "https://example.com/img.png", nil, "user")
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   imageID,
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
			repo := NewImageRepository(querier)
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

func TestImageRepository_ListByEntryIDs(t *testing.T) {
	imageID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name     string
		entryIDs []uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "returns images",
			entryIDs: []uuid.UUID{entryID},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "url", "caption", "source_slug"}).
					AddRow(imageID, entryID, "https://example.com/img.png", nil, "user")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewImageRepository(querier)
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

func TestImageRepository_BatchCreate(t *testing.T) {
	imageID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name    string
		images  []model.Image
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name: "successful batch create",
			images: []model.Image{
				{EntryID: entryID, URL: "https://example.com/img.png", SourceSlug: "user"},
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "url", "caption", "source_slug"}).
					AddRow(imageID, entryID, "https://example.com/img.png", nil, "user")
				mock.ExpectQuery(`INSERT INTO images`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "empty input returns empty",
			images:  []model.Image{},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "validation error - zero entry_id",
			images: []model.Image{
				{EntryID: uuid.UUID{}, URL: "https://example.com/img.png", SourceSlug: "user"},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "validation error - empty url",
			images: []model.Image{
				{EntryID: entryID, URL: "", SourceSlug: "user"},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "validation error - empty source_slug",
			images: []model.Image{
				{EntryID: entryID, URL: "https://example.com/img.png", SourceSlug: ""},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewImageRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.BatchCreate(ctx, tt.images)

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

func TestImageRepository_Delete(t *testing.T) {
	imageID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   imageID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM images`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   imageID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM images`).
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
			repo := NewImageRepository(querier)
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
