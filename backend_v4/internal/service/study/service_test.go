package study

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
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
		State:        domain.CardStateReview,
		Due:          now.Add(-1 * time.Hour),
	}
	dueCard2 := &domain.Card{
		ID:           uuid.New(),
		State:        domain.CardStateLearning,
		Due:          now.Add(-30 * time.Minute),
	}
	newCard := &domain.Card{
		ID:           uuid.New(),
		State:        domain.CardStateNew,
		Due:          time.Time{},
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
	// Verify error wrapping: should contain context about the operation
	if got := err.Error(); got == "" {
		t.Error("expected non-empty error message")
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
		State:        domain.CardStateReview,
		Due:          now.Add(-1 * time.Hour),
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
			State:        domain.CardStateReview,
			Due:          now.Add(-1 * time.Hour),
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          now.Add(1 * time.Minute),
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
			if params.State != domain.CardStateLearning {
				t.Errorf("status: got %v, want Learning", params.State)
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
			if log.PrevState.State != domain.CardStateNew {
				t.Errorf("PrevState.State: got %v, want New", log.PrevState.State)
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
	if result.State != domain.CardStateLearning {
		t.Errorf("result.State: got %v, want Learning", result.State)
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
		State:        domain.CardStateLearning,
		Step:         1,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          now.Add(-5 * time.Minute),
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
			if params.State != domain.CardStateReview {
				t.Errorf("status: got %v, want Review", params.State)
			}
			if params.ScheduledDays != 1 {
				t.Errorf("interval: got %d, want 1", params.ScheduledDays)
			}
			// Return card with params applied
			return &domain.Card{
				ID:            cardID,
				State:         params.State,
				Step:          params.Step,
				ScheduledDays: params.ScheduledDays,
				Stability:     params.Stability,
				Due:           params.Due,
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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

	if result.State != domain.CardStateReview {
		t.Errorf("result.State: got %v, want Review", result.State)
	}
	if result.ScheduledDays != 1 {
		t.Errorf("result.ScheduledDays: got %d, want 1", result.ScheduledDays)
	}
}

func TestService_ReviewCard_Success_ReviewIntervalIncrease(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()
	lastReview := now.Add(-15 * 24 * time.Hour)

	card := &domain.Card{
		ID:            cardID,
		State:         domain.CardStateReview,
		Step:          0,
		ScheduledDays: 15,
		ElapsedDays:   15,
		Stability:     15.0,
		Difficulty:    5.0,
		Due:           now.Add(-1 * time.Hour),
		LastReview:    &lastReview,
		Reps:          5,
	}

	settings := &domain.UserSettings{
		UserID:           userID,
		MaxIntervalDays:  365,
		DesiredRetention: 0.9,
	}

	var capturedParams domain.SRSUpdateParams
	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			capturedParams = params
			if params.State != domain.CardStateReview {
				t.Errorf("state: got %v, want Review", params.State)
			}
			if params.ScheduledDays <= card.ScheduledDays {
				t.Errorf("interval should increase: got %d, was %d", params.ScheduledDays, card.ScheduledDays)
			}
			return &domain.Card{
				ID:            cardID,
				State:         domain.CardState(params.State),
				ScheduledDays: params.ScheduledDays,
				Stability:     params.Stability,
				Difficulty:    params.Difficulty,
				Due:           params.Due,
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
		cards:       mockCards,
		reviews:     mockReviews,
		settings:    mockSettings,
		audit:       mockAudit,
		tx:          mockTx,
		log:         slog.Default(),
		fsrsWeights: fsrs.DefaultWeights,
		srsConfig: domain.SRSConfig{
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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

	if result.ScheduledDays <= card.ScheduledDays {
		t.Errorf("interval should have increased: got %d, was %d", result.ScheduledDays, card.ScheduledDays)
	}
	_ = capturedParams
}

func TestService_ReviewCard_ElapsedDaysComputedFromLastReview(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	card := &domain.Card{
		ID:            cardID,
		UserID:        userID,
		EntryID:       uuid.New(),
		State:         domain.CardStateReview,
		Stability:     10.0,
		Difficulty:    5.0,
		Due:           sevenDaysAgo,
		LastReview:    &sevenDaysAgo,
		Reps:          5,
		ElapsedDays:   0, // DB stores 0 (the bug)
		ScheduledDays: 10,
	}

	settings := &domain.UserSettings{
		UserID:           userID,
		MaxIntervalDays:  365,
		DesiredRetention: 0.9,
	}

	var capturedParams domain.SRSUpdateParams
	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			capturedParams = params
			return &domain.Card{
				ID:            cardID,
				State:         domain.CardState(params.State),
				Stability:     params.Stability,
				Difficulty:    params.Difficulty,
				ScheduledDays: params.ScheduledDays,
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
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error { return nil },
	}
	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		cards:       mockCards,
		reviews:     mockReviews,
		settings:    mockSettings,
		audit:       mockAudit,
		tx:          mockTx,
		log:         slog.Default(),
		fsrsWeights: fsrs.DefaultWeights,
		srsConfig: domain.SRSConfig{
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	_, err := svc.ReviewCard(ctx, ReviewCardInput{
		CardID: cardID,
		Grade:  domain.ReviewGradeGood,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With the fix, the scheduler should receive elapsed_days ≈ 7 (not 0).
	// A stability computed with elapsed=7 differs from elapsed=1 (which max(1,0) produces).
	// Compare against what the scheduler would return with ElapsedDays=1 (the bug):
	buggyCard := fsrs.Card{
		State: fsrs.StateReview, Stability: 10.0, Difficulty: 5.0,
		ElapsedDays: 1, Reps: 5,
	}
	buggyResult := fsrs.ReviewCard(fsrs.Parameters{
		W:                fsrs.DefaultWeights,
		DesiredRetention: 0.9,
		MaxIntervalDays:  365,
	}, buggyCard, fsrs.Good, now)

	if capturedParams.Stability == buggyResult.Stability {
		t.Errorf("stability matches buggy value (%f), elapsed_days was likely not recomputed",
			capturedParams.Stability)
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	updatedCard := &domain.Card{
		ID:           cardID,
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          now.Add(1 * time.Minute),
	}

	prevState := &domain.CardSnapshot{
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			if uid != userID || cid != cardID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, cid, userID, cardID)
			}
			return card, nil
		},
		UpdateSRSFunc: func(ctx context.Context, uid, cid uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error) {
			if params.State != domain.CardStateNew {
				t.Errorf("restored status: got %v, want New", params.State)
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
	if result.State != domain.CardStateNew {
		t.Errorf("result.State: got %v, want New", result.State)
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	prevState := &domain.CardSnapshot{
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	prevState := &domain.CardSnapshot{
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	prevState := &domain.CardSnapshot{
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
		State:        domain.CardStateLearning,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	prevState := &domain.CardSnapshot{
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
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
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
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
			LearningSteps:    []time.Duration{1 * time.Minute, 10 * time.Minute},
			DefaultRetention: 0.9,
			MaxIntervalDays:  365,
			UndoWindowMinutes: 15,
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
// Session Operations Tests (8 tests)
// ---------------------------------------------------------------------------

func TestService_StartSession_Success_CreatesNew(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			return nil, domain.ErrNotFound // No active session
		},
		CreateFunc: func(ctx context.Context, session *domain.StudySession) (*domain.StudySession, error) {
			if session.UserID != userID {
				t.Errorf("session.UserID: got %v, want %v", session.UserID, userID)
			}
			if session.Status != domain.SessionStatusActive {
				t.Errorf("session.Status: got %v, want ACTIVE", session.Status)
			}
			return session, nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.StartSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
	if result.UserID != userID {
		t.Errorf("result.UserID: got %v, want %v", result.UserID, userID)
	}
	if result.Status != domain.SessionStatusActive {
		t.Errorf("result.Status: got %v, want ACTIVE", result.Status)
	}

	// Verify calls
	if len(mockSessions.GetActiveCalls()) != 1 {
		t.Errorf("GetActive calls: got %d, want 1", len(mockSessions.GetActiveCalls()))
	}
	if len(mockSessions.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(mockSessions.CreateCalls()))
	}
}

func TestService_StartSession_ReturnsExisting_Idempotent(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	existingSession := &domain.StudySession{
		ID:        sessionID,
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: now.Add(-10 * time.Minute),
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return existingSession, nil
		},
		CreateFunc: func(ctx context.Context, session *domain.StudySession) (*domain.StudySession, error) {
			t.Error("Create should not be called when active session exists")
			return nil, nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.StartSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != sessionID {
		t.Errorf("result.ID: got %v, want %v", result.ID, sessionID)
	}
	if result.Status != domain.SessionStatusActive {
		t.Errorf("result.Status: got %v, want ACTIVE", result.Status)
	}

	// Verify only GetActive was called
	if len(mockSessions.GetActiveCalls()) != 1 {
		t.Errorf("GetActive calls: got %d, want 1", len(mockSessions.GetActiveCalls()))
	}
	if len(mockSessions.CreateCalls()) != 0 {
		t.Error("Create should not be called")
	}
}

func TestService_FinishSession_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	cardID1 := uuid.New()
	cardID2 := uuid.New()
	now := time.Now()
	startedAt := now.Add(-30 * time.Minute)

	session := &domain.StudySession{
		ID:        sessionID,
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: startedAt,
	}

	// 2 reviews: 1 NEW → LEARNING (GOOD), 1 REVIEW → REVIEW (EASY)
	logs := []*domain.ReviewLog{
		{
			ID:     uuid.New(),
			CardID: cardID1,
			Grade:  domain.ReviewGradeGood,
			PrevState: &domain.CardSnapshot{
				State:  domain.CardStateNew,
			},
			ReviewedAt: now.Add(-20 * time.Minute),
		},
		{
			ID:     uuid.New(),
			CardID: cardID2,
			Grade:  domain.ReviewGradeEasy,
			PrevState: &domain.CardSnapshot{
				State:  domain.CardStateReview,
			},
			ReviewedAt: now.Add(-10 * time.Minute),
		},
	}

	finishedSession := &domain.StudySession{
		ID:         sessionID,
		UserID:     userID,
		Status:     domain.SessionStatusFinished,
		StartedAt:  startedAt,
		FinishedAt: &now,
		Result: &domain.SessionResult{
			TotalReviewed: 2,
			NewReviewed:   1,
			DueReviewed:   1,
			GradeCounts: domain.GradeCounts{
				Good: 1,
				Easy: 1,
			},
			DurationMs:   30 * 60 * 1000, // 30 minutes
			AccuracyRate: 100.0,          // (1+1)/2 * 100
		},
	}

	mockSessions := &sessionRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.StudySession, error) {
			if uid != userID || sid != sessionID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, sid, userID, sessionID)
			}
			return session, nil
		},
		FinishFunc: func(ctx context.Context, uid, sid uuid.UUID, result domain.SessionResult) (*domain.StudySession, error) {
			if result.TotalReviewed != 2 {
				t.Errorf("TotalReviewed: got %d, want 2", result.TotalReviewed)
			}
			if result.NewReviewed != 1 {
				t.Errorf("NewReviewed: got %d, want 1", result.NewReviewed)
			}
			if result.DueReviewed != 1 {
				t.Errorf("DueReviewed: got %d, want 1", result.DueReviewed)
			}
			if result.GradeCounts.Good != 1 {
				t.Errorf("GradeCounts.Good: got %d, want 1", result.GradeCounts.Good)
			}
			if result.GradeCounts.Easy != 1 {
				t.Errorf("GradeCounts.Easy: got %d, want 1", result.GradeCounts.Easy)
			}
			if result.AccuracyRate != 100.0 {
				t.Errorf("AccuracyRate: got %.2f, want 100.00", result.AccuracyRate)
			}
			return finishedSession, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetByPeriodFunc: func(ctx context.Context, uid uuid.UUID, from, to time.Time) ([]*domain.ReviewLog, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if !from.Equal(startedAt) {
				t.Errorf("from: got %v, want %v", from, startedAt)
			}
			return logs, nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		reviews:  mockReviews,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := FinishSessionInput{SessionID: sessionID}

	result, err := svc.FinishSession(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Status != domain.SessionStatusFinished {
		t.Errorf("result.Status: got %v, want FINISHED", result.Status)
	}
	if result.Result == nil {
		t.Fatal("result.Result is nil")
	}
	if result.Result.TotalReviewed != 2 {
		t.Errorf("result.Result.TotalReviewed: got %d, want 2", result.Result.TotalReviewed)
	}

	// Verify calls
	if len(mockSessions.GetByIDCalls()) != 1 {
		t.Errorf("GetByID calls: got %d, want 1", len(mockSessions.GetByIDCalls()))
	}
	if len(mockReviews.GetByPeriodCalls()) != 1 {
		t.Errorf("GetByPeriod calls: got %d, want 1", len(mockReviews.GetByPeriodCalls()))
	}
	if len(mockSessions.FinishCalls()) != 1 {
		t.Errorf("Finish calls: got %d, want 1", len(mockSessions.FinishCalls()))
	}
}

func TestService_FinishSession_AlreadyFinished_ValidationError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	finishedSession := &domain.StudySession{
		ID:         sessionID,
		UserID:     userID,
		Status:     domain.SessionStatusFinished,
		StartedAt:  now.Add(-1 * time.Hour),
		FinishedAt: &now,
	}

	mockSessions := &sessionRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.StudySession, error) {
			return finishedSession, nil
		},
		FinishFunc: func(ctx context.Context, uid, sid uuid.UUID, result domain.SessionResult) (*domain.StudySession, error) {
			t.Error("Finish should not be called for already finished session")
			return nil, nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := FinishSessionInput{SessionID: sessionID}

	_, err := svc.FinishSession(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}

	// Verify Finish was not called
	if len(mockSessions.FinishCalls()) != 0 {
		t.Error("Finish should not be called")
	}
}

func TestService_FinishSession_EmptySession_NoReviews(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()
	startedAt := now.Add(-10 * time.Minute)

	session := &domain.StudySession{
		ID:        sessionID,
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: startedAt,
	}

	emptyLogs := []*domain.ReviewLog{}

	finishedSession := &domain.StudySession{
		ID:         sessionID,
		UserID:     userID,
		Status:     domain.SessionStatusFinished,
		StartedAt:  startedAt,
		FinishedAt: &now,
		Result: &domain.SessionResult{
			TotalReviewed: 0,
			NewReviewed:   0,
			DueReviewed:   0,
			GradeCounts:   domain.GradeCounts{},
			DurationMs:    10 * 60 * 1000,
			AccuracyRate:  0.0,
		},
	}

	mockSessions := &sessionRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.StudySession, error) {
			return session, nil
		},
		FinishFunc: func(ctx context.Context, uid, sid uuid.UUID, result domain.SessionResult) (*domain.StudySession, error) {
			if result.TotalReviewed != 0 {
				t.Errorf("TotalReviewed: got %d, want 0", result.TotalReviewed)
			}
			if result.AccuracyRate != 0.0 {
				t.Errorf("AccuracyRate: got %.2f, want 0.00", result.AccuracyRate)
			}
			return finishedSession, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetByPeriodFunc: func(ctx context.Context, uid uuid.UUID, from, to time.Time) ([]*domain.ReviewLog, error) {
			return emptyLogs, nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		reviews:  mockReviews,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := FinishSessionInput{SessionID: sessionID}

	result, err := svc.FinishSession(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Result.TotalReviewed != 0 {
		t.Errorf("result.Result.TotalReviewed: got %d, want 0", result.Result.TotalReviewed)
	}
	if result.Result.AccuracyRate != 0.0 {
		t.Errorf("result.Result.AccuracyRate: got %.2f, want 0.00", result.Result.AccuracyRate)
	}
}

func TestService_FinishSession_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()

	mockSessions := &sessionRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := FinishSessionInput{SessionID: sessionID}

	_, err := svc.FinishSession(ctx, input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestService_AbandonSession_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	activeSession := &domain.StudySession{
		ID:        sessionID,
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: now.Add(-15 * time.Minute),
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			return activeSession, nil
		},
		AbandonFunc: func(ctx context.Context, uid, sid uuid.UUID) error {
			if uid != userID || sid != sessionID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, sid, userID, sessionID)
			}
			return nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.AbandonSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify calls
	if len(mockSessions.GetActiveCalls()) != 1 {
		t.Errorf("GetActive calls: got %d, want 1", len(mockSessions.GetActiveCalls()))
	}
	if len(mockSessions.AbandonCalls()) != 1 {
		t.Errorf("Abandon calls: got %d, want 1", len(mockSessions.AbandonCalls()))
	}
}

func TestService_AbandonSession_NoActive_IdempotentNoop(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
		AbandonFunc: func(ctx context.Context, uid, sid uuid.UUID) error {
			t.Error("Abandon should not be called when no active session")
			return nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.AbandonSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify only GetActive was called
	if len(mockSessions.GetActiveCalls()) != 1 {
		t.Errorf("GetActive calls: got %d, want 1", len(mockSessions.GetActiveCalls()))
	}
	if len(mockSessions.AbandonCalls()) != 0 {
		t.Error("Abandon should not be called")
	}
}

// ---------------------------------------------------------------------------
// CreateCard Tests (6 tests)
// ---------------------------------------------------------------------------

func TestService_CreateCard_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	cardID := uuid.New()

	entry := &domain.Entry{
		ID:     entryID,
		UserID: userID,
		Text:   "hello",
	}

	createdCard := &domain.Card{
		ID:           cardID,
		UserID:       userID,
		EntryID:      entryID,
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
		Due:          time.Time{},
	}

	mockEntries := &entryRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			if uid != userID || eid != entryID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, eid, userID, entryID)
			}
			return entry, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			if eid != entryID {
				t.Errorf("entryID: got %v, want %v", eid, entryID)
			}
			return 3, nil // Entry has 3 senses
		},
	}

	mockCards := &cardRepoMock{
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if eid != entryID {
				t.Errorf("entryID: got %v, want %v", eid, entryID)
			}
			return createdCard, nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			if record.EntityType != domain.EntityTypeCard {
				t.Errorf("EntityType: got %v, want Card", record.EntityType)
			}
			if record.Action != domain.AuditActionCreate {
				t.Errorf("Action: got %v, want Create", record.Action)
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
		entries: mockEntries,
		senses:  mockSenses,
		cards:   mockCards,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := CreateCardInput{EntryID: entryID}

	result, err := svc.CreateCard(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != cardID {
		t.Errorf("result.ID: got %v, want %v", result.ID, cardID)
	}
	if result.State != domain.CardStateNew {
		t.Errorf("result.State: got %v, want New", result.State)
	}

	// Verify calls
	if len(mockEntries.GetByIDCalls()) != 1 {
		t.Errorf("GetByID calls: got %d, want 1", len(mockEntries.GetByIDCalls()))
	}
	if len(mockSenses.CountByEntryIDCalls()) != 1 {
		t.Errorf("CountByEntryID calls: got %d, want 1", len(mockSenses.CountByEntryIDCalls()))
	}
	if len(mockCards.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(mockCards.CreateCalls()))
	}
	if len(mockAudit.LogCalls()) != 1 {
		t.Errorf("Audit Log calls: got %d, want 1", len(mockAudit.LogCalls()))
	}
}

func TestService_CreateCard_EntryNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()

	mockEntries := &entryRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		entries: mockEntries,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := CreateCardInput{EntryID: entryID}

	_, err := svc.CreateCard(ctx, input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestService_CreateCard_EntryHasNoSenses(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()

	entry := &domain.Entry{
		ID:     entryID,
		UserID: userID,
		Text:   "hello",
	}

	mockEntries := &entryRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return entry, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 0, nil // No senses
		},
	}

	svc := &Service{
		entries: mockEntries,
		senses:  mockSenses,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := CreateCardInput{EntryID: entryID}

	_, err := svc.CreateCard(ctx, input)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}

	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		if len(validationErr.Errors) == 0 {
			t.Error("validation error has no errors")
		} else if validationErr.Errors[0].Field != "entry_id" {
			t.Errorf("validation error field: got %v, want entry_id", validationErr.Errors[0].Field)
		}
	}
}

func TestService_CreateCard_CardAlreadyExists(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()

	entry := &domain.Entry{
		ID:     entryID,
		UserID: userID,
		Text:   "hello",
	}

	mockEntries := &entryRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return entry, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 2, nil
		},
	}

	mockCards := &cardRepoMock{
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return nil, domain.ErrAlreadyExists
		},
	}

	mockTx := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := &Service{
		entries: mockEntries,
		senses:  mockSenses,
		cards:   mockCards,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := CreateCardInput{EntryID: entryID}

	_, err := svc.CreateCard(ctx, input)
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("error: got %v, want ErrAlreadyExists", err)
	}
}

func TestService_CreateCard_TransactionRollback_AuditError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()

	entry := &domain.Entry{
		ID:     entryID,
		UserID: userID,
		Text:   "hello",
	}

	createdCard := &domain.Card{
		ID:           uuid.New(),
		UserID:       userID,
		EntryID:      entryID,
		State:        domain.CardStateNew,
		Step:         0,
		ScheduledDays: 0,
		Stability:    2.5,
	}

	mockEntries := &entryRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return entry, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 1, nil
		},
	}

	mockCards := &cardRepoMock{
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return createdCard, nil
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
		entries: mockEntries,
		senses:  mockSenses,
		cards:   mockCards,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := CreateCardInput{EntryID: entryID}

	_, err := svc.CreateCard(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestService_CreateCard_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background() // No user ID
	input := CreateCardInput{EntryID: uuid.New()}

	_, err := svc.CreateCard(ctx, input)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteCard Tests (4 tests)
// ---------------------------------------------------------------------------

func TestService_DeleteCard_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()
	entryID := uuid.New()

	card := &domain.Card{
		ID:      cardID,
		UserID:  userID,
		EntryID: entryID,
		State:   domain.CardStateReview,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			if uid != userID || cid != cardID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, cid, userID, cardID)
			}
			return card, nil
		},
		DeleteFunc: func(ctx context.Context, uid, cid uuid.UUID) error {
			if uid != userID || cid != cardID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, cid, userID, cardID)
			}
			return nil
		},
	}

	mockAudit := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			if record.EntityType != domain.EntityTypeCard {
				t.Errorf("EntityType: got %v, want Card", record.EntityType)
			}
			if record.Action != domain.AuditActionDelete {
				t.Errorf("Action: got %v, want Delete", record.Action)
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
		cards: mockCards,
		audit: mockAudit,
		tx:    mockTx,
		log:   slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := DeleteCardInput{CardID: cardID}

	err := svc.DeleteCard(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify calls
	if len(mockCards.GetByIDCalls()) != 1 {
		t.Errorf("GetByID calls: got %d, want 1", len(mockCards.GetByIDCalls()))
	}
	if len(mockCards.DeleteCalls()) != 1 {
		t.Errorf("Delete calls: got %d, want 1", len(mockCards.DeleteCalls()))
	}
	if len(mockAudit.LogCalls()) != 1 {
		t.Errorf("Audit Log calls: got %d, want 1", len(mockAudit.LogCalls()))
	}
}

func TestService_DeleteCard_CardNotFound(t *testing.T) {
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
	input := DeleteCardInput{CardID: cardID}

	err := svc.DeleteCard(ctx, input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestService_DeleteCard_TransactionRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:     cardID,
		UserID: userID,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
		DeleteFunc: func(ctx context.Context, uid, cid uuid.UUID) error {
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
		cards: mockCards,
		audit: mockAudit,
		tx:    mockTx,
		log:   slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := DeleteCardInput{CardID: cardID}

	err := svc.DeleteCard(ctx, input)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify Audit was not called
	if len(mockAudit.LogCalls()) != 0 {
		t.Error("Audit should not be called after delete error")
	}
}

func TestService_DeleteCard_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background() // No user ID
	input := DeleteCardInput{CardID: uuid.New()}

	err := svc.DeleteCard(ctx, input)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// BatchCreateCards Tests (6 tests)
// ---------------------------------------------------------------------------

func TestService_BatchCreateCards_Success_AllCreated(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()

	mockEntries := &entryRepoMock{
		ExistByIDsFunc: func(ctx context.Context, uid uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,
				entryID2: true,
			}, nil
		},
	}

	mockCards := &cardRepoMock{
		ExistsByEntryIDsFunc: func(ctx context.Context, uid uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: false,
				entryID2: false,
			}, nil
		},
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return &domain.Card{UserID: uid, EntryID: eid, State: domain.CardStateNew, Stability: 0}, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 1, nil // All entries have senses
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
		entries: mockEntries,
		cards:   mockCards,
		senses:  mockSenses,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := BatchCreateCardsInput{EntryIDs: []uuid.UUID{entryID1, entryID2}}

	result, err := svc.BatchCreateCards(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Created != 2 {
		t.Errorf("Created: got %d, want 2", result.Created)
	}
	if result.SkippedExisting != 0 {
		t.Errorf("SkippedExisting: got %d, want 0", result.SkippedExisting)
	}
	if result.SkippedNoSenses != 0 {
		t.Errorf("SkippedNoSenses: got %d, want 0", result.SkippedNoSenses)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors: got %d, want 0", len(result.Errors))
	}
}

func TestService_BatchCreateCards_SomeEntriesNotExist(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()
	entryID3 := uuid.New()

	mockEntries := &entryRepoMock{
		ExistByIDsFunc: func(ctx context.Context, uid uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,
				entryID2: false, // Does not exist
				entryID3: true,
			}, nil
		},
	}

	mockCards := &cardRepoMock{
		ExistsByEntryIDsFunc: func(ctx context.Context, uid uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: false,
				entryID3: false,
			}, nil
		},
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return &domain.Card{UserID: uid, EntryID: eid, State: domain.CardStateNew, Stability: 0}, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 1, nil
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
		entries: mockEntries,
		cards:   mockCards,
		senses:  mockSenses,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := BatchCreateCardsInput{EntryIDs: []uuid.UUID{entryID1, entryID2, entryID3}}

	result, err := svc.BatchCreateCards(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Created != 2 {
		t.Errorf("Created: got %d, want 2", result.Created)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors: got %d, want 1", len(result.Errors))
	} else if result.Errors[0].EntryID != entryID2 {
		t.Errorf("Error EntryID: got %v, want %v", result.Errors[0].EntryID, entryID2)
	}
}

func TestService_BatchCreateCards_SomeEntriesNoSenses(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()

	mockEntries := &entryRepoMock{
		ExistByIDsFunc: func(ctx context.Context, uid uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,
				entryID2: true,
			}, nil
		},
	}

	mockCards := &cardRepoMock{
		ExistsByEntryIDsFunc: func(ctx context.Context, uid uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: false,
				entryID2: false,
			}, nil
		},
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return &domain.Card{UserID: uid, EntryID: eid, State: domain.CardStateNew, Stability: 0}, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			if eid == entryID1 {
				return 1, nil
			}
			return 0, nil // entryID2 has no senses
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
		entries: mockEntries,
		cards:   mockCards,
		senses:  mockSenses,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := BatchCreateCardsInput{EntryIDs: []uuid.UUID{entryID1, entryID2}}

	result, err := svc.BatchCreateCards(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("Created: got %d, want 1", result.Created)
	}
	if result.SkippedNoSenses != 1 {
		t.Errorf("SkippedNoSenses: got %d, want 1", result.SkippedNoSenses)
	}
}

func TestService_BatchCreateCards_SomeEntriesAlreadyHaveCards(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()

	mockEntries := &entryRepoMock{
		ExistByIDsFunc: func(ctx context.Context, uid uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,
				entryID2: true,
			}, nil
		},
	}

	mockCards := &cardRepoMock{
		ExistsByEntryIDsFunc: func(ctx context.Context, uid uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,  // Already has card
				entryID2: false, // No card yet
			}, nil
		},
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return &domain.Card{UserID: uid, EntryID: eid, State: domain.CardStateNew, Stability: 0}, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 1, nil
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
		entries: mockEntries,
		cards:   mockCards,
		senses:  mockSenses,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := BatchCreateCardsInput{EntryIDs: []uuid.UUID{entryID1, entryID2}}

	result, err := svc.BatchCreateCards(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("Created: got %d, want 1", result.Created)
	}
	if result.SkippedExisting != 1 {
		t.Errorf("SkippedExisting: got %d, want 1", result.SkippedExisting)
	}
}

func TestService_BatchCreateCards_MixedScenario(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID1 := uuid.New()
	entryID2 := uuid.New()
	entryID3 := uuid.New()
	entryID4 := uuid.New()

	mockEntries := &entryRepoMock{
		ExistByIDsFunc: func(ctx context.Context, uid uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,
				entryID2: false, // Not exist
				entryID3: true,
				entryID4: true,
			}, nil
		},
	}

	mockCards := &cardRepoMock{
		ExistsByEntryIDsFunc: func(ctx context.Context, uid uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				entryID1: true,  // Already has card
				entryID3: false, // No card
				entryID4: false, // No card
			}, nil
		},
		CreateFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Card, error) {
			return &domain.Card{UserID: uid, EntryID: eid, State: domain.CardStateNew, Stability: 0}, nil
		},
	}

	mockSenses := &senseRepoMock{
		CountByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			if eid == entryID3 {
				return 1, nil
			}
			return 0, nil // entryID4 has no senses
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
		entries: mockEntries,
		cards:   mockCards,
		senses:  mockSenses,
		audit:   mockAudit,
		tx:      mockTx,
		log:     slog.Default(),
		srsConfig: domain.SRSConfig{
			DefaultRetention: 0.9,
		},
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := BatchCreateCardsInput{EntryIDs: []uuid.UUID{entryID1, entryID2, entryID3, entryID4}}

	result, err := svc.BatchCreateCards(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Created != 1 {
		t.Errorf("Created: got %d, want 1", result.Created)
	}
	if result.SkippedExisting != 1 {
		t.Errorf("SkippedExisting: got %d, want 1", result.SkippedExisting)
	}
	if result.SkippedNoSenses != 1 {
		t.Errorf("SkippedNoSenses: got %d, want 1", result.SkippedNoSenses)
	}
	if len(result.Errors) != 1 {
		t.Errorf("Errors: got %d, want 1 (entry not found)", len(result.Errors))
	}
}

func TestService_BatchCreateCards_ValidationError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	// Test empty list
	input1 := BatchCreateCardsInput{EntryIDs: []uuid.UUID{}}
	_, err := svc.BatchCreateCards(ctx, input1)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}

	// Test > 100 entries
	tooManyIDs := make([]uuid.UUID, 101)
	for i := 0; i < 101; i++ {
		tooManyIDs[i] = uuid.New()
	}
	input2 := BatchCreateCardsInput{EntryIDs: tooManyIDs}
	_, err = svc.BatchCreateCards(ctx, input2)
	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("error: got %v, want ErrValidation", err)
	}
}

// ---------------------------------------------------------------------------
// GetDashboard Tests (7 tests)
// ---------------------------------------------------------------------------

func TestService_GetDashboard_Success_AllCountersCorrect(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	statusCounts := domain.CardStatusCounts{
		New:      10,
		Learning: 5,
		Review:   20,
		Relearning: 15,
		Total:    50,
	}

	// Use current date for streak data (use UTC to match service)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	streakDays := []domain.DayReviewCount{
		{Date: today, Count: 5},
		{Date: today.AddDate(0, 0, -1), Count: 3},
		{Date: today.AddDate(0, 0, -2), Count: 7},
	}

	activeSession := &domain.StudySession{
		ID:     sessionID,
		UserID: userID,
		Status: domain.SessionStatusActive,
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 8, nil
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 10, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return statusCounts, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 3, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 5, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 2, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return streakDays, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return activeSession, nil
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.DueCount != 8 {
		t.Errorf("DueCount: got %d, want 8", dashboard.DueCount)
	}
	if dashboard.NewCount != 10 {
		t.Errorf("NewCount: got %d, want 10", dashboard.NewCount)
	}
	if dashboard.ReviewedToday != 5 {
		t.Errorf("ReviewedToday: got %d, want 5", dashboard.ReviewedToday)
	}
	if dashboard.NewToday != 2 {
		t.Errorf("NewToday: got %d, want 2", dashboard.NewToday)
	}
	if dashboard.Streak != 3 {
		t.Errorf("Streak: got %d, want 3", dashboard.Streak)
	}
	if dashboard.StatusCounts.Total != 50 {
		t.Errorf("StatusCounts.Total: got %d, want 50", dashboard.StatusCounts.Total)
	}
	if dashboard.ActiveSession == nil {
		t.Error("ActiveSession should not be nil")
	} else if *dashboard.ActiveSession != sessionID {
		t.Errorf("ActiveSession: got %v, want %v", *dashboard.ActiveSession, sessionID)
	}
}

func TestService_GetDashboard_NoCards_AllZeros(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 0, nil
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return domain.CardStatusCounts{}, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return []domain.DayReviewCount{}, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.DueCount != 0 {
		t.Errorf("DueCount: got %d, want 0", dashboard.DueCount)
	}
	if dashboard.NewCount != 0 {
		t.Errorf("NewCount: got %d, want 0", dashboard.NewCount)
	}
	if dashboard.Streak != 0 {
		t.Errorf("Streak: got %d, want 0", dashboard.Streak)
	}
	if dashboard.ActiveSession != nil {
		t.Error("ActiveSession should be nil")
	}
}

func TestService_GetDashboard_StreakCalculation_FiveDays(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	// Use current date for streak data (use UTC to match service)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	// 5 consecutive days including today
	streakDays := []domain.DayReviewCount{
		{Date: today, Count: 5},
		{Date: today.AddDate(0, 0, -1), Count: 3},
		{Date: today.AddDate(0, 0, -2), Count: 7},
		{Date: today.AddDate(0, 0, -3), Count: 2},
		{Date: today.AddDate(0, 0, -4), Count: 4},
		{Date: today.AddDate(0, 0, -6), Count: 1}, // Gap here
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 0, nil
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return domain.CardStatusCounts{}, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return streakDays, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.Streak != 5 {
		t.Errorf("Streak: got %d, want 5", dashboard.Streak)
	}
}

func TestService_GetDashboard_StreakBroken_Gap(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	// Use current date for streak data (use UTC to match service)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	// Gap between day -1 and day -3
	streakDays := []domain.DayReviewCount{
		{Date: today, Count: 5},
		{Date: today.AddDate(0, 0, -1), Count: 3},
		// Missing day -2
		{Date: today.AddDate(0, 0, -3), Count: 7},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 0, nil
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return domain.CardStatusCounts{}, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return streakDays, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Streak should be 2 (today + yesterday)
	if dashboard.Streak != 2 {
		t.Errorf("Streak: got %d, want 2", dashboard.Streak)
	}
}

func TestService_GetDashboard_StreakStartsFromYesterday(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	// Use current date for streak data (use UTC to match service)
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	// No review today, but yesterday and before
	streakDays := []domain.DayReviewCount{
		// Today missing
		{Date: today.AddDate(0, 0, -1), Count: 3},
		{Date: today.AddDate(0, 0, -2), Count: 7},
		{Date: today.AddDate(0, 0, -3), Count: 2},
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 0, nil
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return domain.CardStatusCounts{}, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return streakDays, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Streak should be 3 (yesterday + 2 days before)
	if dashboard.Streak != 3 {
		t.Errorf("Streak: got %d, want 3", dashboard.Streak)
	}
}

func TestService_GetDashboard_OverdueCount(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 25, nil // 25 due cards
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return domain.CardStatusCounts{}, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 12, nil // 12 overdue cards
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return []domain.DayReviewCount{}, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.OverdueCount != 12 {
		t.Errorf("OverdueCount: got %d, want 12", dashboard.OverdueCount)
	}
}

func TestService_GetDashboard_ActiveSessionPresent(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()

	settings := &domain.UserSettings{
		UserID:   userID,
		Timezone: "UTC",
	}

	activeSession := &domain.StudySession{
		ID:     sessionID,
		UserID: userID,
		Status: domain.SessionStatusActive,
	}

	mockSettings := &settingsRepoMock{
		GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return settings, nil
		},
	}

	mockCards := &cardRepoMock{
		CountDueFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time) (int, error) {
			return 0, nil
		},
		CountNewFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CountByStatusFunc: func(ctx context.Context, uid uuid.UUID) (domain.CardStatusCounts, error) {
			return domain.CardStatusCounts{}, nil
		},
		CountOverdueFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		CountTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
			return 0, nil
		},
		GetStreakDaysFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error) {
			return []domain.DayReviewCount{}, nil
		},
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return activeSession, nil
		},
	}

	svc := &Service{
		settings: mockSettings,
		cards:    mockCards,
		reviews:  mockReviews,
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	dashboard, err := svc.GetDashboard(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if dashboard.ActiveSession == nil {
		t.Error("ActiveSession should not be nil")
	} else if *dashboard.ActiveSession != sessionID {
		t.Errorf("ActiveSession: got %v, want %v", *dashboard.ActiveSession, sessionID)
	}
}

// ---------------------------------------------------------------------------
// GetCardHistory Tests (3 tests)
// ---------------------------------------------------------------------------

func TestService_GetCardHistory_Success_WithPagination(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:     cardID,
		UserID: userID,
		State:  domain.CardStateReview,
	}

	logs := []*domain.ReviewLog{
		{ID: uuid.New(), CardID: cardID, Grade: domain.ReviewGradeGood},
		{ID: uuid.New(), CardID: cardID, Grade: domain.ReviewGradeEasy},
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			if uid != userID || cid != cardID {
				t.Errorf("unexpected IDs: got (%v, %v), want (%v, %v)", uid, cid, userID, cardID)
			}
			return card, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetByCardIDFunc: func(ctx context.Context, cid uuid.UUID, limit, offset int) ([]*domain.ReviewLog, int, error) {
			if cid != cardID {
				t.Errorf("cardID: got %v, want %v", cid, cardID)
			}
			if limit != 10 {
				t.Errorf("limit: got %d, want 10", limit)
			}
			if offset != 5 {
				t.Errorf("offset: got %d, want 5", offset)
			}
			return logs, 25, nil // 2 logs, 25 total
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetCardHistoryInput{
		CardID: cardID,
		Limit:  10,
		Offset: 5,
	}

	result, total, err := svc.GetCardHistory(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("result length: got %d, want 2", len(result))
	}
	if total != 25 {
		t.Errorf("total: got %d, want 25", total)
	}

	// Verify calls
	if len(mockCards.GetByIDCalls()) != 1 {
		t.Errorf("GetByID calls: got %d, want 1", len(mockCards.GetByIDCalls()))
	}
	if len(mockReviews.GetByCardIDCalls()) != 1 {
		t.Errorf("GetByCardID calls: got %d, want 1", len(mockReviews.GetByCardIDCalls()))
	}
}

func TestService_GetCardHistory_CardNotFound_OwnershipCheck(t *testing.T) {
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
	input := GetCardHistoryInput{
		CardID: cardID,
		Limit:  50,
		Offset: 0,
	}

	_, _, err := svc.GetCardHistory(ctx, input)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestService_GetCardHistory_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background() // No user ID
	input := GetCardHistoryInput{
		CardID: uuid.New(),
		Limit:  50,
		Offset: 0,
	}

	_, _, err := svc.GetCardHistory(ctx, input)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// GetCardStats Tests (2 tests)
// ---------------------------------------------------------------------------

func TestService_GetCardStats_Success_WithStats(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:            cardID,
		UserID:        userID,
		State:         domain.CardStateReview,
		ScheduledDays: 14,
		Stability:     2.6,
	}

	avgDur := 5333
	agg := domain.ReviewLogAggregation{
		TotalReviews:  4,
		AgainCount:    1,
		HardCount:     0,
		GoodCount:     2,
		EasyCount:     1,
		AvgDurationMs: &avgDur,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetStatsByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (domain.ReviewLogAggregation, error) {
			return agg, nil
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetCardHistoryInput{
		CardID: cardID,
		Limit:  0,
		Offset: 0,
	}

	stats, err := svc.GetCardStats(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalReviews != 4 {
		t.Errorf("TotalReviews: got %d, want 4", stats.TotalReviews)
	}
	// AccuracyRate = (2 GOOD + 1 EASY) / 4 * 100 = 75%
	if stats.AccuracyRate != 75.0 {
		t.Errorf("AccuracyRate: got %.2f, want 75.00", stats.AccuracyRate)
	}
	// AverageTimeMs = 5333 (from SQL aggregation)
	if stats.AverageTimeMs == nil {
		t.Error("AverageTimeMs should not be nil")
	} else if *stats.AverageTimeMs != 5333 {
		t.Errorf("AverageTimeMs: got %d, want 5333", *stats.AverageTimeMs)
	}
	if stats.CurrentState != domain.CardStateReview {
		t.Errorf("CurrentState: got %v, want Review", stats.CurrentState)
	}
	if stats.ScheduledDays != 14 {
		t.Errorf("ScheduledDays: got %d, want 14", stats.ScheduledDays)
	}
	if stats.Stability != 2.6 {
		t.Errorf("Stability: got %.1f, want 2.6", stats.Stability)
	}
}

func TestService_GetCardStats_NoReviews_ZerosAndNil(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	card := &domain.Card{
		ID:            cardID,
		UserID:        userID,
		State:         domain.CardStateNew,
		ScheduledDays: 0,
		Stability:     2.5,
	}

	mockCards := &cardRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, cid uuid.UUID) (*domain.Card, error) {
			return card, nil
		},
	}

	mockReviews := &reviewLogRepoMock{
		GetStatsByCardIDFunc: func(ctx context.Context, cid uuid.UUID) (domain.ReviewLogAggregation, error) {
			return domain.ReviewLogAggregation{}, nil
		},
	}

	svc := &Service{
		cards:   mockCards,
		reviews: mockReviews,
		log:     slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetCardHistoryInput{
		CardID: cardID,
		Limit:  0,
		Offset: 0,
	}

	stats, err := svc.GetCardStats(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.TotalReviews != 0 {
		t.Errorf("TotalReviews: got %d, want 0", stats.TotalReviews)
	}
	if stats.AccuracyRate != 0.0 {
		t.Errorf("AccuracyRate: got %.2f, want 0.00", stats.AccuracyRate)
	}
	if stats.AverageTimeMs != nil {
		t.Errorf("AverageTimeMs should be nil, got %d", *stats.AverageTimeMs)
	}
	if stats.CurrentState != domain.CardStateNew {
		t.Errorf("CurrentState: got %v, want New", stats.CurrentState)
	}
}

// ---------------------------------------------------------------------------
// GetActiveSession Tests
// ---------------------------------------------------------------------------

func TestService_GetActiveSession_Success_ReturnsSession(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	activeSession := &domain.StudySession{
		ID:        sessionID,
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: now.Add(-10 * time.Minute),
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			return activeSession, nil
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected session, got nil")
	}
	if result.ID != sessionID {
		t.Errorf("result.ID: got %v, want %v", result.ID, sessionID)
	}
}

func TestService_GetActiveSession_NoActiveSession_ReturnsNil(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.GetActiveSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestService_GetActiveSession_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background()

	_, err := svc.GetActiveSession(ctx)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestService_GetActiveSession_RepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			return nil, errors.New("db connection error")
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.GetActiveSession(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---------------------------------------------------------------------------
// StartSession Race Condition Test
// ---------------------------------------------------------------------------

func TestService_StartSession_RaceCondition_ErrAlreadyExists(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()
	now := time.Now()

	raceSession := &domain.StudySession{
		ID:        sessionID,
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: now,
	}

	mockSessions := &sessionRepoMock{
		GetActiveFunc: func() func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
			callCount := 0
			return func(ctx context.Context, uid uuid.UUID) (*domain.StudySession, error) {
				callCount++
				if callCount == 1 {
					// First call: no active session
					return nil, domain.ErrNotFound
				}
				// Second call (after race): session now exists
				return raceSession, nil
			}
		}(),
		CreateFunc: func(ctx context.Context, session *domain.StudySession) (*domain.StudySession, error) {
			// Simulate race: another request created between check and create
			return nil, domain.ErrAlreadyExists
		},
	}

	svc := &Service{
		sessions: mockSessions,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.StartSession(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("result is nil")
	}
	if result.ID != sessionID {
		t.Errorf("result.ID: got %v, want %v", result.ID, sessionID)
	}

	// GetActive should be called twice: initial check + retry after race
	if len(mockSessions.GetActiveCalls()) != 2 {
		t.Errorf("GetActive calls: got %d, want 2", len(mockSessions.GetActiveCalls()))
	}
	if len(mockSessions.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(mockSessions.CreateCalls()))
	}
}

func TestService_StartSession_NoUserID(t *testing.T) {
	t.Parallel()

	svc := &Service{
		log: slog.Default(),
	}

	ctx := context.Background()

	_, err := svc.StartSession(ctx)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// GetStudyQueue Default Limit Test
// ---------------------------------------------------------------------------

func TestService_GetStudyQueue_DefaultLimit_WhenZero(t *testing.T) {
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
			return 0, nil
		},
	}

	mockCards := &cardRepoMock{
		GetDueCardsFunc: func(ctx context.Context, uid uuid.UUID, nowTime time.Time, limit int) ([]*domain.Card, error) {
			if limit != 50 {
				t.Errorf("expected default limit 50, got %d", limit)
			}
			return []*domain.Card{}, nil
		},
		GetNewCardsFunc: func(ctx context.Context, uid uuid.UUID, limit int) ([]*domain.Card, error) {
			return []*domain.Card{}, nil
		},
	}

	svc := &Service{
		cards:    mockCards,
		reviews:  mockReviews,
		settings: mockSettings,
		log:      slog.Default(),
	}

	ctx := ctxutil.WithUserID(context.Background(), userID)
	input := GetQueueInput{Limit: 0} // Should use default 50

	_, err := svc.GetStudyQueue(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test Helpers
// ---------------------------------------------------------------------------

func ptr[T any](v T) *T {
	return &v
}
