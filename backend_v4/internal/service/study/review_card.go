package study

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
)

// ReviewCard records a review and updates the card's SRS state using FSRS-5.
func (s *Service) ReviewCard(ctx context.Context, input ReviewCardInput) (*domain.Card, error) {
	userID, err := s.userID(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	now := s.clock.Now()

	// Load settings outside tx (read-only, no lock needed)
	settings, err := s.settings.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}

	params := s.buildFSRSParams(settings)
	rating := mapGradeToRating(input.Grade)

	var updatedCard *domain.Card

	// Transaction: lock card, compute FSRS, update card + create log + audit
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Lock card row inside transaction
		card, cardErr := s.cards.GetByIDForUpdate(txCtx, userID, input.CardID)
		if cardErr != nil {
			return fmt.Errorf("get card: %w", cardErr)
		}

		snapshot := snapshotFromCard(card)

		fsrsCard := cardToFSRS(card)
		fsrsCard.ElapsedDays = computeElapsedDays(card.LastReview, now)

		// Calculate new SRS state
		result, fsrsErr := fsrs.ReviewCard(params, fsrsCard, rating, now)
		if fsrsErr != nil {
			return fmt.Errorf("fsrs review: %w", fsrsErr)
		}

		var updateErr error
		updatedCard, updateErr = s.cards.UpdateSRS(txCtx, userID, card.ID, fsrsResultToUpdateParams(result))
		if updateErr != nil {
			return fmt.Errorf("update card: %w", updateErr)
		}

		// Create review log
		_, logErr := s.reviews.Create(txCtx, &domain.ReviewLog{
			ID:         uuid.New(),
			CardID:     card.ID,
			UserID:     userID,
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
		slog.String("card_id", input.CardID.String()),
		slog.String("grade", string(input.Grade)),
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
