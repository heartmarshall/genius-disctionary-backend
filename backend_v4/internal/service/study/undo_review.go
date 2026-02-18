package study

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

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
