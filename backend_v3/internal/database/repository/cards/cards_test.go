package cards

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/testutil"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func TestCardRepository_Create(t *testing.T) {
	entryID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		card    *model.Card
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
		check   func(t *testing.T, result *model.Card)
	}{
		{
			name: "successful creation with defaults",
			card: &model.Card{
				EntryID: entryID,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"}).
					AddRow(cardID, entryID, model.StatusNew, nil, 0, 2.5, now, now)
				mock.ExpectQuery(`INSERT INTO cards`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
			check: func(t *testing.T, result *model.Card) {
				if result.Status != model.StatusNew {
					t.Errorf("Expected status %v, got %v", model.StatusNew, result.Status)
				}
				if result.EaseFactor != 2.5 {
					t.Errorf("Expected ease_factor 2.5, got %v", result.EaseFactor)
				}
			},
		},
		{
			name: "successful creation with custom values",
			card: &model.Card{
				EntryID:      entryID,
				Status:       model.StatusLearning,
				NextReviewAt: timePtr(now.Add(24 * time.Hour)),
				IntervalDays: 1,
				EaseFactor:   2.6,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				nextReview := now.Add(24 * time.Hour)
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"}).
					AddRow(cardID, entryID, model.StatusLearning, &nextReview, 1, 2.6, now, now)
				mock.ExpectQuery(`INSERT INTO cards`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
			check: func(t *testing.T, result *model.Card) {
				if result.Status != model.StatusLearning {
					t.Errorf("Expected status %v, got %v", model.StatusLearning, result.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewCardRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Create(ctx, tt.card)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Create() returned nil result")
					return
				}
				if tt.check != nil {
					tt.check(t, result)
				}
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestCardRepository_Update(t *testing.T) {
	cardID := uuid.New()
	entryID := uuid.New()
	now := time.Now()
	nextReview := now.Add(24 * time.Hour)

	tests := []struct {
		name    string
		id      uuid.UUID
		card    *model.Card
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful update",
			id:   cardID,
			card: &model.Card{
				Status:       model.StatusReview,
				NextReviewAt: &nextReview,
				IntervalDays: 7,
				EaseFactor:   2.7,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"}).
					AddRow(cardID, entryID, model.StatusReview, &nextReview, 7, 2.7, now, now)
				mock.ExpectQuery(`UPDATE cards`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   cardID,
			card: &model.Card{
				Status: model.StatusNew,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`UPDATE cards`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewCardRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Update(ctx, tt.id, tt.card)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Update() returned nil result")
			}

			if tt.wantErr && err != database.ErrNotFound {
				t.Errorf("Update() expected ErrNotFound, got %v", err)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestCardRepository_UpdateSRSFields(t *testing.T) {
	cardID := uuid.New()
	now := time.Now()
	nextReview := now.Add(24 * time.Hour)

	tests := []struct {
		name         string
		id           uuid.UUID
		status       model.LearningStatus
		nextReviewAt *time.Time
		intervalDays int
		easeFactor   float64
		setup        func(mock pgxmock.PgxPoolIface)
		wantErr      bool
	}{
		{
			name:         "successful update",
			id:           cardID,
			status:       model.StatusReview,
			nextReviewAt: &nextReview,
			intervalDays: 7,
			easeFactor:   2.7,
			setup: func(mock pgxmock.PgxPoolIface) {
				entryID := uuid.New()
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"}).
					AddRow(cardID, entryID, model.StatusReview, &nextReview, 7, 2.7, now, now)
				mock.ExpectQuery(`UPDATE cards`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:         "not found",
			id:           cardID,
			status:       model.StatusNew,
			nextReviewAt: nil,
			intervalDays: 0,
			easeFactor:   2.5,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`UPDATE cards`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true, // Update возвращает ErrNotFound при 0 rows
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewCardRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			err := repo.UpdateSRSFields(ctx, tt.id, tt.status, tt.nextReviewAt, tt.intervalDays, tt.easeFactor)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSRSFields() error = %v, wantErr %v", err, tt.wantErr)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestCardRepository_Delete(t *testing.T) {
	cardID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   cardID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM cards`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   cardID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM cards`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewCardRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			err := repo.Delete(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && err != nil {
				if err != database.ErrNotFound {
					t.Errorf("Delete() unexpected error = %v", err)
				}
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestCardRepository_GetByEntryID(t *testing.T) {
	cardID := uuid.New()
	entryID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		entryID uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name:    "found",
			entryID: entryID,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"}).
					AddRow(cardID, entryID, model.StatusNew, nil, 0, 2.5, now, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:    "not found",
			entryID: entryID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewCardRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.GetByEntryID(ctx, tt.entryID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByEntryID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("GetByEntryID() returned nil result")
			}

			if tt.wantErr && err != database.ErrNotFound {
				t.Errorf("GetByEntryID() expected ErrNotFound, got %v", err)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestCardRepository_GetDueCards(t *testing.T) {
	cardID1 := uuid.New()
	cardID2 := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()
	now := time.Now()
	dueTime := now.Add(-1 * time.Hour)

	tests := []struct {
		name    string
		now     time.Time
		limit   int
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name:  "returns due cards",
			now:   now,
			limit: 10,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"}).
					AddRow(cardID1, entryID1, model.StatusReview, &dueTime, 7, 2.5, now, now).
					AddRow(cardID2, entryID2, model.StatusLearning, &dueTime, 1, 2.5, now, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:  "returns empty when no due cards",
			now:   now,
			limit: 10,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "entry_id", "status", "next_review_at", "interval_days", "ease_factor", "created_at", "updated_at"})
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewCardRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.GetDueCards(ctx, tt.now, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDueCards() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("GetDueCards() returned %d cards, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestReviewLogRepository_Create(t *testing.T) {
	logID := uuid.New()
	cardID := uuid.New()
	now := time.Now()
	duration := 5000

	tests := []struct {
		name    string
		log     *model.ReviewLog
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful creation",
			log: &model.ReviewLog{
				CardID:     cardID,
				Grade:      model.GradeGood,
				DurationMs: &duration,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "card_id", "grade", "duration_ms", "reviewed_at"}).
					AddRow(logID, cardID, model.GradeGood, &duration, now)
				mock.ExpectQuery(`INSERT INTO review_logs`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "successful creation without duration",
			log: &model.ReviewLog{
				CardID: cardID,
				Grade:  model.GradeEasy,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "card_id", "grade", "duration_ms", "reviewed_at"}).
					AddRow(logID, cardID, model.GradeEasy, nil, now)
				mock.ExpectQuery(`INSERT INTO review_logs`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewReviewLogRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Create(ctx, tt.log)

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

func TestReviewLogRepository_ListByCardID(t *testing.T) {
	cardID := uuid.New()
	logID1 := uuid.New()
	logID2 := uuid.New()
	now := time.Now()
	duration1 := 5000
	duration2 := 3000

	tests := []struct {
		name    string
		cardID  uuid.UUID
		limit   int
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name:   "returns logs with limit",
			cardID: cardID,
			limit:  10,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "card_id", "grade", "duration_ms", "reviewed_at"}).
					AddRow(logID1, cardID, model.GradeGood, &duration1, now.Add(-1*time.Hour)).
					AddRow(logID2, cardID, model.GradeEasy, &duration2, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 2,
			wantErr: false,
		},
		{
			name:   "returns empty when no logs",
			cardID: cardID,
			limit:  10,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "card_id", "grade", "duration_ms", "reviewed_at"})
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:   "respects limit",
			cardID: cardID,
			limit:  1,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "card_id", "grade", "duration_ms", "reviewed_at"}).
					AddRow(logID1, cardID, model.GradeGood, &duration1, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewReviewLogRepository(querier)

			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListByCardID(ctx, tt.cardID, tt.limit)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListByCardID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("ListByCardID() returned %d logs, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

// Helper function
func timePtr(t time.Time) *time.Time {
	return &t
}
