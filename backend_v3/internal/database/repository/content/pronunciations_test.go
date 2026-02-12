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

func TestPronunciationRepository_GetByID(t *testing.T) {
	pronID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "found",
			id:   pronID,
			setup: func(mock pgxmock.PgxPoolIface) {
				audioURL := "https://example.com/audio.mp3"
				rows := pgxmock.NewRows([]string{"id", "entry_id", "audio_url", "transcription", "region", "source_slug"}).
					AddRow(pronID, entryID, &audioURL, "/həˈloʊ/", nil, "freedict")
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   pronID,
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
			repo := NewPronunciationRepository(querier)
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

func TestPronunciationRepository_ListByEntryIDs(t *testing.T) {
	pronID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name     string
		entryIDs []uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "returns pronunciations",
			entryIDs: []uuid.UUID{entryID},
			setup: func(mock pgxmock.PgxPoolIface) {
				audioURL := "https://example.com/audio.mp3"
				rows := pgxmock.NewRows([]string{"id", "entry_id", "audio_url", "transcription", "region", "source_slug"}).
					AddRow(pronID, entryID, &audioURL, "/həˈloʊ/", nil, "freedict")
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
			repo := NewPronunciationRepository(querier)
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

func TestPronunciationRepository_BatchCreate(t *testing.T) {
	pronID := uuid.New()
	entryID := uuid.New()

	tests := []struct {
		name           string
		pronunciations []model.Pronunciation
		setup          func(mock pgxmock.PgxPoolIface)
		wantLen        int
		wantErr        bool
	}{
		{
			name: "successful batch create",
			pronunciations: []model.Pronunciation{
				{EntryID: entryID, Transcription: "/həˈloʊ/", SourceSlug: "freedict"},
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "audio_url", "transcription", "region", "source_slug"}).
					AddRow(pronID, entryID, nil, "/həˈloʊ/", nil, "freedict")
				mock.ExpectQuery(`INSERT INTO pronunciations`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:           "empty input returns empty",
			pronunciations: []model.Pronunciation{},
			setup:          func(mock pgxmock.PgxPoolIface) {},
			wantLen:        0,
			wantErr:        false,
		},
		{
			name: "validation error - zero entry_id",
			pronunciations: []model.Pronunciation{
				{EntryID: uuid.UUID{}, Transcription: "/həˈloʊ/", SourceSlug: "freedict"},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
		{
			name: "validation error - empty source_slug",
			pronunciations: []model.Pronunciation{
				{EntryID: entryID, Transcription: "/həˈloʊ/", SourceSlug: ""},
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantLen: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewPronunciationRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.BatchCreate(ctx, tt.pronunciations)

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

func TestPronunciationRepository_Delete(t *testing.T) {
	pronID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   pronID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM pronunciations`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   pronID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM pronunciations`).
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
			repo := NewPronunciationRepository(querier)
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
