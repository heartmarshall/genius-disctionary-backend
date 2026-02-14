package study

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// Consumer-defined interfaces (private)
// ---------------------------------------------------------------------------

type cardRepo interface {
	GetByID(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
	GetByEntryID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error)
	Create(ctx context.Context, userID uuid.UUID, card *domain.Card) (*domain.Card, error)
	UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error)
	Delete(ctx context.Context, userID, cardID uuid.UUID) error
	GetDueCards(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*domain.Card, error)
	GetNewCards(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Card, error)
	CountByStatus(ctx context.Context, userID uuid.UUID) (domain.CardStatusCounts, error)
	CountDue(ctx context.Context, userID uuid.UUID, now time.Time) (int, error)
	CountNew(ctx context.Context, userID uuid.UUID) (int, error)
	ExistsByEntryIDs(ctx context.Context, userID uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type reviewLogRepo interface {
	Create(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error)
	GetByCardID(ctx context.Context, cardID uuid.UUID, limit, offset int) ([]*domain.ReviewLog, int, error)
	GetLastByCardID(ctx context.Context, cardID uuid.UUID) (*domain.ReviewLog, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
	CountNewToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
	GetStreakDays(ctx context.Context, userID uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error)
	GetByPeriod(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.ReviewLog, error)
}

type sessionRepo interface {
	Create(ctx context.Context, session *domain.StudySession) (*domain.StudySession, error)
	GetByID(ctx context.Context, userID, sessionID uuid.UUID) (*domain.StudySession, error)
	GetActive(ctx context.Context, userID uuid.UUID) (*domain.StudySession, error)
	Finish(ctx context.Context, userID, sessionID uuid.UUID, result domain.SessionResult) (*domain.StudySession, error)
	Abandon(ctx context.Context, userID, sessionID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.StudySession, int, error)
}

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	ExistByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

type senseRepo interface {
	CountByEntryID(ctx context.Context, entryID uuid.UUID) (int, error)
}

type settingsRepo interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
}

type auditLogger interface {
	Log(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the study business logic.
type Service struct {
	cards     cardRepo
	reviews   reviewLogRepo
	sessions  sessionRepo
	entries   entryRepo
	senses    senseRepo
	settings  settingsRepo
	audit     auditLogger
	tx        txManager
	log       *slog.Logger
	srsConfig domain.SRSConfig
}

// NewService creates a new Study service.
func NewService(
	log *slog.Logger,
	cards cardRepo,
	reviews reviewLogRepo,
	sessions sessionRepo,
	entries entryRepo,
	senses senseRepo,
	settings settingsRepo,
	audit auditLogger,
	tx txManager,
	srsConfig domain.SRSConfig,
) *Service {
	return &Service{
		cards:     cards,
		reviews:   reviews,
		sessions:  sessions,
		entries:   entries,
		senses:    senses,
		settings:  settings,
		audit:     audit,
		tx:        tx,
		log:       log.With("service", "study"),
		srsConfig: srsConfig,
	}
}

// ---------------------------------------------------------------------------
// Queue, Review & Undo Operations
// ---------------------------------------------------------------------------

// GetStudyQueue returns cards ready for review (due cards + new cards respecting daily limit).
func (s *Service) GetStudyQueue(ctx context.Context, input GetQueueInput) ([]*domain.Card, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	limit := input.Limit
	if limit == 0 {
		limit = 50
	}

	now := time.Now()

	// Load user settings for limits and timezone
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("load settings: %w", err)
	}

	tz := ParseTimezone(settings.Timezone)
	dayStart := DayStart(now, tz)

	// Count new cards reviewed today
	newToday, err := s.reviews.CountNewToday(ctx, userID, dayStart)
	if err != nil {
		return nil, fmt.Errorf("count new today: %w", err)
	}

	newRemaining := max(0, settings.NewCardsPerDay-newToday)

	// Get due cards (overdue not limited by reviews_per_day)
	dueCards, err := s.cards.GetDueCards(ctx, userID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("get due cards: %w", err)
	}

	// Fill remaining slots with new cards
	queue := dueCards
	if len(dueCards) < limit && newRemaining > 0 {
		newLimit := min(limit-len(dueCards), newRemaining)
		newCards, err := s.cards.GetNewCards(ctx, userID, newLimit)
		if err != nil {
			return nil, fmt.Errorf("get new cards: %w", err)
		}
		queue = append(queue, newCards...)
	}

	s.log.InfoContext(ctx, "study queue generated",
		slog.String("user_id", userID.String()),
		slog.Int("due_count", len(dueCards)),
		slog.Int("new_count", len(queue)-len(dueCards)),
		slog.Int("total", len(queue)),
	)

	return queue, nil
}

// ReviewCard records a review and updates the card's SRS state.
func (s *Service) ReviewCard(ctx context.Context, input ReviewCardInput) (*domain.Card, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()

	// Load card
	card, err := s.cards.GetByID(ctx, userID, input.CardID)
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}

	// Load settings
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	maxInterval := min(s.srsConfig.MaxIntervalDays, settings.MaxIntervalDays)

	// Snapshot state before review
	snapshot := &domain.CardSnapshot{
		Status:       card.Status,
		LearningStep: card.LearningStep,
		IntervalDays: card.IntervalDays,
		EaseFactor:   card.EaseFactor,
		NextReviewAt: card.NextReviewAt,
	}

	// Calculate new SRS state
	srsResult := CalculateSRS(SRSInput{
		CurrentStatus:   card.Status,
		CurrentInterval: card.IntervalDays,
		CurrentEase:     card.EaseFactor,
		LearningStep:    card.LearningStep,
		Grade:           input.Grade,
		Now:             now,
		Config:          s.srsConfig,
		MaxIntervalDays: maxInterval,
	})

	var updatedCard *domain.Card

	// Transaction: update card + create log + audit
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Update card
		nextReviewAt := srsResult.NextReviewAt
		var updateErr error
		updatedCard, updateErr = s.cards.UpdateSRS(txCtx, userID, card.ID, domain.SRSUpdateParams{
			Status:       srsResult.NewStatus,
			NextReviewAt: &nextReviewAt,
			IntervalDays: srsResult.NewInterval,
			EaseFactor:   srsResult.NewEase,
			LearningStep: srsResult.NewLearningStep,
		})
		if updateErr != nil {
			return fmt.Errorf("update card: %w", updateErr)
		}

		// Create review log
		_, logErr := s.reviews.Create(txCtx, &domain.ReviewLog{
			ID:         uuid.New(),
			CardID:     card.ID,
			Grade:      input.Grade,
			PrevState:  snapshot,
			DurationMs: input.DurationMs,
			ReviewedAt: now,
		})
		if logErr != nil {
			return fmt.Errorf("create review log: %w", logErr)
		}

		// Audit
		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeCard,
			EntityID:   &card.ID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"grade": map[string]any{"new": input.Grade},
				"status": map[string]any{
					"old": card.Status,
					"new": srsResult.NewStatus,
				},
				"interval": map[string]any{
					"old": card.IntervalDays,
					"new": srsResult.NewInterval,
				},
			},
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Safety check: ensure card was actually updated
	if updatedCard == nil {
		return nil, fmt.Errorf("card update failed: no result returned")
	}

	s.log.InfoContext(ctx, "card reviewed",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()),
		slog.String("grade", string(input.Grade)),
		slog.String("old_status", string(card.Status)),
		slog.String("new_status", string(srsResult.NewStatus)),
		slog.Int("new_interval", srsResult.NewInterval),
	)

	return updatedCard, nil
}

// UndoReview reverts the last review of a card within the undo window.
func (s *Service) UndoReview(ctx context.Context, input UndoReviewInput) (*domain.Card, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	now := time.Now()

	// Load card
	card, err := s.cards.GetByID(ctx, userID, input.CardID)
	if err != nil {
		return nil, fmt.Errorf("get card: %w", err)
	}

	// Load last review log
	lastLog, err := s.reviews.GetLastByCardID(ctx, input.CardID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewValidationError("card_id", "card has no reviews to undo")
		}
		return nil, fmt.Errorf("get last review: %w", err)
	}

	// Check prev_state exists
	if lastLog.PrevState == nil {
		return nil, domain.NewValidationError("review", "review cannot be undone")
	}

	// Check undo window
	undoWindow := time.Duration(s.srsConfig.UndoWindowMinutes) * time.Minute
	if now.Sub(lastLog.ReviewedAt) > undoWindow {
		return nil, domain.NewValidationError("review", "undo window expired")
	}

	var restoredCard *domain.Card

	// Transaction: restore card + delete log + audit
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Restore prev state (NextReviewAt might be nil for NEW cards)
		var nextReview *time.Time
		if lastLog.PrevState.NextReviewAt != nil {
			t := *lastLog.PrevState.NextReviewAt
			nextReview = &t
		}

		var restoreErr error
		restoredCard, restoreErr = s.cards.UpdateSRS(txCtx, userID, card.ID, domain.SRSUpdateParams{
			Status:       lastLog.PrevState.Status,
			NextReviewAt: nextReview,
			IntervalDays: lastLog.PrevState.IntervalDays,
			EaseFactor:   lastLog.PrevState.EaseFactor,
			LearningStep: lastLog.PrevState.LearningStep,
		})
		if restoreErr != nil {
			return fmt.Errorf("restore card: %w", restoreErr)
		}

		// Delete review log
		if deleteErr := s.reviews.Delete(txCtx, lastLog.ID); deleteErr != nil {
			return fmt.Errorf("delete review log: %w", deleteErr)
		}

		// Audit
		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeCard,
			EntityID:   &card.ID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"undo": map[string]any{"old": lastLog.Grade},
				"status": map[string]any{
					"old": card.Status,
					"new": lastLog.PrevState.Status,
				},
			},
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Safety check: ensure card was actually restored
	if restoredCard == nil {
		return nil, fmt.Errorf("card restore failed: no result returned")
	}

	s.log.InfoContext(ctx, "review undone",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()),
		slog.String("undone_grade", string(lastLog.Grade)),
		slog.String("restored_status", string(lastLog.PrevState.Status)),
	)

	return restoredCard, nil
}

// ---------------------------------------------------------------------------
// Session Operations
// ---------------------------------------------------------------------------

// StartSession starts a new study session or returns existing ACTIVE session (idempotent).
func (s *Service) StartSession(ctx context.Context) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	// Check for existing ACTIVE session first
	existing, err := s.sessions.GetActive(ctx, userID)
	if err == nil {
		// Found existing ACTIVE session - return it (idempotent)
		s.log.InfoContext(ctx, "returning existing session",
			slog.String("user_id", userID.String()),
			slog.String("session_id", existing.ID.String()),
		)
		return existing, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check active session: %w", err)
	}

	// No active session - create new one
	session := &domain.StudySession{
		ID:        uuid.New(),
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: time.Now(),
	}

	created, err := s.sessions.Create(ctx, session)
	if err != nil {
		// Race condition: another request created session between check and create
		if errors.Is(err, domain.ErrAlreadyExists) {
			// Retry: fetch the session that was just created
			existing, getErr := s.sessions.GetActive(ctx, userID)
			if getErr != nil {
				return nil, fmt.Errorf("get active after race: %w", getErr)
			}
			s.log.InfoContext(ctx, "race condition detected, returning existing session",
				slog.String("user_id", userID.String()),
				slog.String("session_id", existing.ID.String()),
			)
			return existing, nil
		}
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.log.InfoContext(ctx, "session started",
		slog.String("user_id", userID.String()),
		slog.String("session_id", created.ID.String()),
	)

	return created, nil
}

// FinishSession finishes an ACTIVE session, aggregating review logs and calculating stats.
func (s *Service) FinishSession(ctx context.Context, input FinishSessionInput) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Load session
	session, err := s.sessions.GetByID(ctx, userID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	// Check status: must be ACTIVE
	if session.Status != domain.SessionStatusActive {
		return nil, domain.NewValidationError("session", "session already finished")
	}

	now := time.Now()

	// Aggregate review logs for period [session.StartedAt, now]
	logs, err := s.reviews.GetByPeriod(ctx, userID, session.StartedAt, now)
	if err != nil {
		return nil, fmt.Errorf("get review logs: %w", err)
	}

	// Calculate stats
	totalReviewed := len(logs)
	newReviewed := 0
	gradeCounts := domain.GradeCounts{}

	for _, log := range logs {
		// Count new reviews (cards that were NEW before this review)
		if log.PrevState != nil && log.PrevState.Status == domain.LearningStatusNew {
			newReviewed++
		}

		// Count grades
		switch log.Grade {
		case domain.ReviewGradeAgain:
			gradeCounts.Again++
		case domain.ReviewGradeHard:
			gradeCounts.Hard++
		case domain.ReviewGradeGood:
			gradeCounts.Good++
		case domain.ReviewGradeEasy:
			gradeCounts.Easy++
		}
	}

	dueReviewed := totalReviewed - newReviewed
	durationMs := now.Sub(session.StartedAt).Milliseconds()

	accuracyRate := 0.0
	if totalReviewed > 0 {
		accuracyRate = float64(gradeCounts.Good+gradeCounts.Easy) / float64(totalReviewed) * 100
	}

	// Create SessionResult
	result := domain.SessionResult{
		TotalReviewed: totalReviewed,
		NewReviewed:   newReviewed,
		DueReviewed:   dueReviewed,
		GradeCounts:   gradeCounts,
		DurationMs:    durationMs,
		AccuracyRate:  accuracyRate,
	}

	// Finish session
	finishedSession, err := s.sessions.Finish(ctx, userID, session.ID, result)
	if err != nil {
		return nil, fmt.Errorf("finish session: %w", err)
	}

	s.log.InfoContext(ctx, "session finished",
		slog.String("user_id", userID.String()),
		slog.String("session_id", session.ID.String()),
		slog.Int("total_reviewed", totalReviewed),
		slog.Int("new_reviewed", newReviewed),
		slog.Float64("accuracy_rate", accuracyRate),
	)

	return finishedSession, nil
}

// AbandonSession abandons the current ACTIVE session (idempotent noop if no active session).
func (s *Service) AbandonSession(ctx context.Context) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Try to get active session
	session, err := s.sessions.GetActive(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// No active session - idempotent noop
			return nil
		}
		return fmt.Errorf("get active session: %w", err)
	}

	// Abandon the active session
	if err := s.sessions.Abandon(ctx, userID, session.ID); err != nil {
		return fmt.Errorf("abandon session: %w", err)
	}

	s.log.InfoContext(ctx, "session abandoned",
		slog.String("user_id", userID.String()),
		slog.String("session_id", session.ID.String()),
	)

	return nil
}
