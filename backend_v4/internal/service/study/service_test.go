package study

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// GetStudyQueue Tests (8 tests)
// ---------------------------------------------------------------------------

func TestService_GetStudyQueue_Success_MixOfDueAndNew(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now()

	dueCard1 := &domain.Card{
		ID:           uuid.New(),
		Status:       domain.LearningStatusReview,
		NextReviewAt: ptr(now.Add(-1 * time.Hour)),
	}
	dueCard2 := &domain.Card{
		ID:           uuid.New(),
		Status:       domain.LearningStatusLearning,
		NextReviewAt: ptr(now.Add(-30 * time.Minute)),
	}
	newCard := &domain.Card{
		ID:           uuid.New(),
		Status:       domain.LearningStatusNew,
		NextReviewAt: nil,
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			if uid != userID {
				t.Errorf("unexpected userID: got %v, want %v", uid, userID)
			}
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			if uid != userID {
				t.Errorf("unexpected userID: got %v, want %v", uid, userID)
			}
			return 5, nil // 5 new cards already reviewed today
		},
	}

	mockCards := &cardRepoMock{
		GetDueCardsFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time, limit int) ([]*domain.Card, error) {
			if uid != userID {
				t.Errorf("unexpected userID: got %v, want %v", uid, userID)
			}
			if limit != 50 {
				t.Errorf("unexpected limit: got %d, want 50", limit)
			}
			return []*domain.Card{dueCard1, dueCard2}, nil
		},
		GetNewCardsFunc: func(ctx context.Context, uid uuid.UUID, limit int) ([]*domain.Card, error) {
			if uid != userID {
				t.Errorf("unexpected userID: got %v, want %v", uid, userID)
			}
			// newRemaining = 20 - 5 = 15
			// newLimit = min(50 - 2, 15) = 15
			if limit != 15 {
				t.Errorf("unexpected new limit: got %d, want 15", limit)
			}
			return []*domain.Card{newCard}, nil
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 50}

	queue, err := svc.GetStudyQueue(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(queue) != 3 {
		t.Errorf("queue length: got %d, want 3", len(queue))
	}

	// Verify calls
	if len(mockSettings.GetByUserIDCalls()) != 1 {
		t.Errorf("GetByUserID calls: got %d, want 1", len(mockSettings.GetByUserIDCalls()))
	}
	if len(mockReviews.CountNewTodayCalls()) != 1 {
		t.Errorf("CountNewToday calls: got %d, want 1", len(mockReviews.CountNewTodayCalls()))
	}
	if len(mockCards.GetDueCardsCalls()) != 1 {
		t.Errorf("GetDueCards calls: got %d, want 1", len(mockCards.GetDueCardsCalls()))
	}
	if len(mockCards.GetNewCardsCalls()) != 1 {
		t.Errorf("GetNewCards calls: got %d, want 1", len(mockCards.GetNewCardsCalls()))
	}
}

func TestService_GetStudyQueue_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background() // No user ID
	input := GetQueueInput{Limit: 50}

	_, err := svc.GetStudyQueue(ctx, input)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestService_GetStudyQueue_InvalidInput(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &Service{
		log: slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 300} // Exceeds max 200

	_, err := svc.GetStudyQueue(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}
}

func TestService_GetStudyQueue_SettingsLoadError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return nil, errors.New("db error")
		},
	}

	svc := &Service{
		settings: mockSettings,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 50}

	_, err := svc.GetStudyQueue(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, errors.New("db error")) {
		// Check that error is wrapped
		if err.Error() == "" {
			t.Error("expected wrapped error")
		}
	}
}

func TestService_GetStudyQueue_CountNewTodayError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	settings := &domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, errors.New("count error")
		},
	}

	svc := &Service{
		settings: mockSettings,
		reviews:  mockReviews,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 50}

	_, err := svc.GetStudyQueue(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetStudyQueue_DueCardsError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	settings := &domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 5, nil
		},
	}

	mockCards := &cardRepoMock{
		GetDueCardsFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time, limit int) ([]*domain.Card, error) {
			return nil, errors.New("due cards error")
		},
	}

	svc := &Service{
		settings: mockSettings,
		reviews:  mockReviews,
		cards:    mockCards,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 50}

	_, err := svc.GetStudyQueue(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_GetStudyQueue_DailyLimitReached(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now()

	dueCard := &domain.Card{
		ID:           uuid.New(),
		Status:       domain.LearningStatusReview,
		NextReviewAt: ptr(now.Add(-1 * time.Hour)),
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 20, nil // Limit reached
		},
	}

	mockCards := &cardRepoMock{
		GetDueCardsFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time, limit int) ([]*domain.Card, error) {
			return []*domain.Card{dueCard}, nil
		},
		GetNewCardsFunc: func(ctx context.Context, uid uuid.UUID, limit int) ([]*domain.Card, error) {
			t.Error("GetNewCards should not be called when limit reached")
			return nil, nil
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 50}

	queue, err := svc.GetStudyQueue(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only have due cards, no new cards
	if len(queue) != 1 {
		t.Errorf("queue length: got %d, want 1", len(queue))
	}

	if len(mockCards.GetNewCardsCalls()) != 0 {
		t.Error("GetNewCards should not be called")
	}
}

func TestService_GetStudyQueue_OnlyDueCards(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	now := time.Now()

	// 50 due cards (fills the limit)
	dueCards := make([]*domain.Card, 50)
	for i := 0; i < 50; i++ {
		dueCards[i] = &domain.Card{
			ID:           uuid.New(),
			Status:       domain.LearningStatusReview,
			NextReviewAt: ptr(now.Add(-1 * time.Hour)),
		}
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
	}

	mockCards := &cardRepoMock{
		GetDueCardsFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time, limit int) ([]*domain.Card, error) {
			return dueCards, nil
		},
		GetNewCardsFunc: func(ctx context.Context, uid uuid.UUID, limit int) ([]*domain.Card, error) {
			t.Error("GetNewCards should not be called when queue is full")
			return nil, nil
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 50}

	queue, err := svc.GetStudyQueue(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(queue) != 50 {
		t.Errorf("queue length: got %d, want 50", len(queue))
	}

	// Should not call GetNewCards because queue is already full
	if len(mockCards.GetNewCardsCalls()) != 0 {
		t.Error("GetNewCards should not be called")
	}
}

// ---------------------------------------------------------------------------
// ReviewCard Tests (10 tests)
// ---------------------------------------------------------------------------

func TestService_ReviewCard_Success_NewToLearning(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: ptr(now.Add(1 * time.Minute)),
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		MaxIntervalDays: 365,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			if uid != userID || cid != cardID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, cid, userID, cardID)
			}
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			if params.Status != domain.LearningStatusLearning {
				t.Errorf("status: got %v, want Learning", params.Status)
			}
			return updatedCard, nil
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CreateFunc: func(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error) {
			if log.CardID != cardID {
				t.Errorf("log.CardID: got %v, want %v", log.CardID, cardID)
			}
			if log.Grade != domain.ReviewGradeGood {
				t.Errorf("log.Grade: got %v, want Good", log.Grade)
			}
			if log.PrevState == nil {
				t.Error("PrevState is nil")
			}
			if log.PrevState.Status != domain.LearningStatusNew {
				t.Errorf("PrevState.Status: got %v, want New", log.PrevState.Status)
			}
			return log, nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			if record.EntityType != domain.EntityTypeCard {
				t.Errorf("EntityType: got %v, want Card", record.EntityType)
			}
			if record.Action != domain.AuditActionUpdate {
				t.Errorf("Action: got %v, want Update", record.Action)
			}
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx) // Execute immediately
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		audit:    mockAudit,
		tx:       mockTx,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	result, err := svc.ReviewCard(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != cardID {
		t.Errorf("result.ID: got %v, want %v", result.ID, cardID)
	}
	if result.Status != domain.LearningStatusLearning {
		t.Errorf("result.Status: got %v, want Learning", result.Status)
	}

	// Verify calls
	if len(mockCards.GetByIDCalls()) != 1 {
		t.Errorf("GetByID calls: got %d, want 1", len(mockCards.GetByIDCalls()))
	}
	if len(mockCards.UpdateSRSCalls()) != 1 {
		t.Errorf("UpdateSRS calls: got %d, want 1", len(mockCards.UpdateSRSCalls()))
	}
	if len(mockReviews.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(mockReviews.CreateCalls()))
	}
	if len(mockAudit.LogCalls()) != 1 {
		t.Errorf("Audit Log calls: got %d, want 1", len(mockAudit.LogCalls()))
	}
	if len(mockTx.RunInTxCalls()) != 1 {
		t.Errorf("RunInTx calls: got %d, want 1", len(mockTx.RunInTxCalls()))
	}
}

func TestService_ReviewCard_Success_LearningToReview(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 1,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: ptr(now.Add(-5 * time.Minute)),
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		MaxIntervalDays: 365,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			if params.Status != domain.LearningStatusReview {
				t.Errorf("status: got %v, want Review", params.Status)
			}
			if params.IntervalDays != 1 {
				t.Errorf("interval: got %d, want 1", params.IntervalDays)
			}
			// Return card with params applied
			return &domain.Card{
				ID:           cardID,
				Status:       params.Status,
				LearningStep: params.LearningStep,
				IntervalDays: params.IntervalDays,
				EaseFactor:   params.EaseFactor,
				NextReviewAt: params.NextReviewAt,
			}, nil
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CreateFunc: func(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error) {
			return log, nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		audit:    mockAudit,
		tx:       mockTx,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(3000),
	}

	result, err := svc.ReviewCard(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.LearningStatusReview {
		t.Errorf("result.Status: got %v, want Review", result.Status)
	}
	if result.IntervalDays != 1 {
		t.Errorf("result.IntervalDays: got %d, want 1", result.IntervalDays)
	}
}

func TestService_ReviewCard_Success_ReviewIntervalIncrease(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusReview,
		LearningStep: 0,
		IntervalDays: 7,
		EaseFactor:   2.5,
		NextReviewAt: ptr(now.Add(-1 * time.Hour)),
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusReview,
		LearningStep: 0,
		IntervalDays: 17, // 7 * 2.5 = 17.5 → 17
		EaseFactor:   2.6,
		NextReviewAt: ptr(now.Add(17 * 24 * time.Hour)),
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		MaxIntervalDays: 365,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			if params.Status != domain.LearningStatusReview {
				t.Errorf("status: got %v, want Review", params.Status)
			}
			if params.IntervalDays < 7 {
				t.Errorf("interval should increase: got %d", params.IntervalDays)
			}
			return updatedCard, nil
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CreateFunc: func(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error) {
			return log, nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		audit:    mockAudit,
		tx:       mockTx,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(8000),
	}

	result, err := svc.ReviewCard(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IntervalDays <= 7 {
		t.Errorf("interval should have increased: got %d", result.IntervalDays)
	}
}

func TestService_ReviewCard_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background() // No user ID
	input := ReviewCardInput{
		CardID:     uuid.New(),
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestService_ReviewCard_InvalidInput(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &Service{
		log: slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     uuid.Nil, // Invalid
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}
}

func TestService_ReviewCard_CardNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		cards: mockCards,
		log:   slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestService_ReviewCard_SettingsLoadError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return nil, errors.New("settings error")
		},
	}

	svc := &Service{
		cards:    mockCards,
		settings: mockSettings,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_ReviewCard_UpdateSRSError_TxRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		MaxIntervalDays: 365,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			return nil, errors.New("update error")
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CreateFunc: func(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error) {
			t.Error("Create should not be called after UpdateSRS error")
			return nil, nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx) // Will propagate the error
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		tx:       mockTx,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify Create was not called
	if len(mockReviews.CreateCalls()) != 0 {
		t.Error("Create should not be called after UpdateSRS error")
	}
}

func TestService_ReviewCard_CreateReviewLogError_TxRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		MaxIntervalDays: 365,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			return updatedCard, nil
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CreateFunc: func(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error) {
			return nil, errors.New("create log error")
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			t.Error("Audit should not be called after CreateLog error")
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		audit:    mockAudit,
		tx:       mockTx,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify Audit was not called
	if len(mockAudit.LogCalls()) != 0 {
		t.Error("Audit should not be called after CreateLog error")
	}
}

func TestService_ReviewCard_AuditError_TxRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	settings := &domain.UserSettings{
		UserID:          userID,
		MaxIntervalDays: 365,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			return updatedCard, nil
		},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CreateFunc: func(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error) {
			return log, nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			return errors.New("audit error")
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		audit:    mockAudit,
		tx:       mockTx,
		log:      slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := ReviewCardInput{
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		DurationMs: ptr(5000),
	}

	_, err := svc.ReviewCard(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// UndoReview Tests (10 tests)
// ---------------------------------------------------------------------------

func TestService_UndoReview_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	logID := uuid.New()
	now := time.Now()
	reviewedAt := now.Add(-5 * time.Minute)

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: ptr(now.Add(1 * time.Minute)),
	}

	prevState := &domain.CardSnapshot{
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	reviewLog := &domain.ReviewLog{
		ID:         logID,
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		PrevState:  prevState,
		DurationMs: ptr(5000),
		ReviewedAt: reviewedAt,
	}

	restoredCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			if uid != userID || cid != cardID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, cid, userID, cardID)
			}
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			if params.Status != domain.LearningStatusNew {
				t.Errorf("restored status: got %v, want New", params.Status)
			}
			return restoredCard, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			if cid != cardID {
				t.Errorf("cardID: got %v, want %v", cid, cardID)
			}
			return reviewLog, nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			if id != logID {
				t.Errorf("logID: got %v, want %v", id, logID)
			}
			return nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			if record.Action != domain.AuditActionUpdate {
				t.Errorf("Action: got %v, want Update", record.Action)
			}
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	result, err := svc.UndoReview(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != cardID {
		t.Errorf("result.ID: got %v, want %v", result.ID, cardID)
	}
	if result.Status != domain.LearningStatusNew {
		t.Errorf("result.Status: got %v, want New", result.Status)
	}

	// Verify calls
	if len(mockCards.GetByIDCalls()) != 1 {
		t.Errorf("GetByID calls: got %d, want 1", len(mockCards.GetByIDCalls()))
	}
	if len(mockReviews.GetLastByCardIDCalls()) != 1 {
		t.Errorf("GetLastByCardID calls: got %d, want 1", len(mockReviews.GetLastByCardIDCalls()))
	}
	if len(mockCards.UpdateSRSCalls()) != 1 {
		t.Errorf("UpdateSRS calls: got %d, want 1", len(mockCards.UpdateSRSCalls()))
	}
	if len(mockReviews.DeleteCalls()) != 1 {
		t.Errorf("Delete calls: got %d, want 1", len(mockReviews.DeleteCalls()))
	}
	if len(mockAudit.LogCalls()) != 1 {
		t.Errorf("Audit Log calls: got %d, want 1", len(mockAudit.LogCalls()))
	}
}

func TestService_UndoReview_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background() // No user ID
	input := UndoReviewInput{CardID: uuid.New()}

	_, err := svc.UndoReview(ctx, input)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestService_UndoReview_InvalidInput(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := &Service{
		log: slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: uuid.Nil} // Invalid

	_, err := svc.UndoReview(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}
}

func TestService_UndoReview_CardNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		cards: mockCards,
		log:   slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestService_UndoReview_NoReviewLog(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}
}

func TestService_UndoReview_PrevStateNil(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	reviewLog := &domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		PrevState:  nil, // No prev state
		DurationMs: ptr(5000),
		ReviewedAt: now,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			return reviewLog, nil
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}
}

func TestService_UndoReview_UndoWindowExpired(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()
	reviewedAt := now.Add(-20 * time.Minute) // Beyond 15 min window

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	prevState := &domain.CardSnapshot{
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	reviewLog := &domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		PrevState:  prevState,
		DurationMs: ptr(5000),
		ReviewedAt: reviewedAt,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			return reviewLog, nil
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}

	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		if len(validationErr.Errors) == 0 {
			t.Error("validation error has no errors")
		} else if validationErr.Errors[0].Field != "review" || validationErr.Errors[0].Message != "undo window expired" {
			t.Errorf("validation error: got %+v", validationErr.Errors[0])
		}
	} else {
		t.Error("error is not ValidationError")
	}
}

func TestService_UndoReview_RestoreError_TxRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()
	reviewedAt := now.Add(-5 * time.Minute)

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	prevState := &domain.CardSnapshot{
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	reviewLog := &domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		PrevState:  prevState,
		DurationMs: ptr(5000),
		ReviewedAt: reviewedAt,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			return nil, errors.New("restore error")
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			return reviewLog, nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			t.Error("Delete should not be called after restore error")
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify Delete was not called
	if len(mockReviews.DeleteCalls()) != 0 {
		t.Error("Delete should not be called after restore error")
	}
}

func TestService_UndoReview_DeleteLogError_TxRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()
	reviewedAt := now.Add(-5 * time.Minute)

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	prevState := &domain.CardSnapshot{
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	reviewLog := &domain.ReviewLog{
		ID:         uuid.New(),
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		PrevState:  prevState,
		DurationMs: ptr(5000),
		ReviewedAt: reviewedAt,
	}

	restoredCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			return restoredCard, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			return reviewLog, nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			return errors.New("delete error")
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			t.Error("Audit should not be called after delete error")
			return nil
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify Audit was not called
	if len(mockAudit.LogCalls()) != 0 {
		t.Error("Audit should not be called after delete error")
	}
}

func TestService_UndoReview_AuditError_TxRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	logID := uuid.New()
	now := time.Now()
	reviewedAt := now.Add(-5 * time.Minute)

	card := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusLearning,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	prevState := &domain.CardSnapshot{
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		NextReviewAt: nil,
	}

	reviewLog := &domain.ReviewLog{
		ID:         logID,
		CardID:     cardID,
		Grade:      domain.ReviewGradeGood,
		PrevState:  prevState,
		DurationMs: ptr(5000),
		ReviewedAt: reviewedAt,
	}

	restoredCard := &domain.Card{
		ID:           cardID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			return restoredCard, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetLastByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (*domain.ReviewLog, error) {
			return reviewLog, nil
		},
		DeleteFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			return errors.New("audit error")
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			LearningSteps:      []time.Duration{1 * time.Minute, 10 * time.Minute},
			GraduatingInterval: 1,
			DefaultEaseFactor:  2.5,
			MaxIntervalDays:    365,
			UndoWindowMinutes:  15,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := UndoReviewInput{CardID: cardID}

	_, err := svc.UndoReview(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Test Helpers
// ---------------------------------------------------------------------------

func ptr[T any](v T) *T {
	return &v
}
