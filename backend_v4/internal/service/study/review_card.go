package study

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ReviewCard records a review and updates the card's SRS state using FSRS-5.
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

	// Build FSRS parameters
	params := fsrs.Parameters{
		W:                s.fsrsWeights,
		DesiredRetention: settings.DesiredRetention,
		MaxIntervalDays:  min(s.srsConfig.MaxIntervalDays, settings.MaxIntervalDays),
		EnableFuzz:       s.srsConfig.EnableFuzz,
		LearningSteps:    s.srsConfig.LearningSteps,
		RelearningSteps:  s.srsConfig.RelearningSteps,
	}

	// Snapshot state before review
	snapshot := &domain.CardSnapshot{
		State:         card.State,
		Step:          card.Step,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		Due:           card.Due,
		LastReview:    card.LastReview,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		ScheduledDays: card.ScheduledDays,
		ElapsedDays:   card.ElapsedDays,
	}

	// Map domain grade to FSRS rating
	rating := mapGradeToRating(input.Grade)

	// Convert domain card to FSRS card
	fsrsCard := fsrs.Card{
		State:         fsrs.CardState(card.State),
		Step:          card.Step,
		Stability:     card.Stability,
		Difficulty:    card.Difficulty,
		Due:           card.Due,
		LastReview:    card.LastReview,
		Reps:          card.Reps,
		Lapses:        card.Lapses,
		ScheduledDays: card.ScheduledDays,
		ElapsedDays:   card.ElapsedDays,
	}

	// Compute actual elapsed days since last review for FSRS retrievability calculation.
	// The DB stores elapsed_days=0 after each review; we must recompute it here.
	if card.LastReview != nil {
		elapsed := now.Sub(*card.LastReview)
		fsrsCard.ElapsedDays = max(0, int(elapsed.Hours()/24))
	}

	// Calculate new SRS state
	result := fsrs.ReviewCard(params, fsrsCard, rating, now)

	var updatedCard *domain.Card

	// Transaction: update card + create log + audit
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var lastReview *time.Time
		if result.LastReview != nil {
			t := *result.LastReview
			lastReview = &t
		}

		var updateErr error
		updatedCard, updateErr = s.cards.UpdateSRS(txCtx, userID, card.ID, domain.SRSUpdateParams{
			State:         domain.CardState(result.State),
			Step:          result.Step,
			Stability:     result.Stability,
			Difficulty:    result.Difficulty,
			Due:           result.Due,
			LastReview:    lastReview,
			Reps:          result.Reps,
			Lapses:        result.Lapses,
			ScheduledDays: result.ScheduledDays,
			ElapsedDays:   result.ElapsedDays,
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
				"state": map[string]any{
					"old": card.State,
					"new": updatedCard.State,
				},
				"stability": map[string]any{
					"old": card.Stability,
					"new": updatedCard.Stability,
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

	if updatedCard == nil {
		return nil, fmt.Errorf("card update failed: no result returned")
	}

	s.log.InfoContext(ctx, "card reviewed",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()),
		slog.String("grade", string(input.Grade)),
		slog.String("old_state", string(card.State)),
		slog.String("new_state", string(updatedCard.State)),
		slog.Float64("stability", updatedCard.Stability),
	)

	return updatedCard, nil
}

// mapGradeToRating maps domain ReviewGrade to FSRS Rating.
func mapGradeToRating(grade domain.ReviewGrade) fsrs.Rating {
	switch grade {
	case domain.ReviewGradeAgain:
		return fsrs.Again
	case domain.ReviewGradeHard:
		return fsrs.Hard
	case domain.ReviewGradeGood:
		return fsrs.Good
	case domain.ReviewGradeEasy:
		return fsrs.Easy
	default:
		return fsrs.Good
	}
}
