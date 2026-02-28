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
	var restoredCard *domain.Card
	var undoneGrade domain.ReviewGrade
	var restoredState domain.CardState

	// Transaction: lock card, validate, restore, delete log, audit
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Lock card row
		card, cardErr := s.cards.GetByIDForUpdate(txCtx, userID, input.CardID)
		if cardErr != nil {
			return fmt.Errorf("get card: %w", cardErr)
		}

		// Load last review log
		lastLog, logErr := s.reviews.GetLastByCardID(txCtx, input.CardID)
		if logErr != nil {
			if errors.Is(logErr, domain.ErrNotFound) {
				return domain.NewValidationError("card_id", "card has no reviews to undo")
			}
			return fmt.Errorf("get last review: %w", logErr)
		}

		// Check prev_state exists
		if lastLog.PrevState == nil {
			return domain.NewValidationError("review", "review cannot be undone")
		}

		// Check undo window
		undoWindow := time.Duration(s.srsConfig.UndoWindowMinutes) * time.Minute
		if now.Sub(lastLog.ReviewedAt) > undoWindow {
			return domain.NewValidationError("review", "undo window expired")
		}

		undoneGrade = lastLog.Grade
		ps := lastLog.PrevState
		restoredState = ps.State

		var restoreErr error
		restoredCard, restoreErr = s.cards.UpdateSRS(txCtx, userID, card.ID, domain.SRSUpdateParams{
			State:         ps.State,
			Step:          ps.Step,
			Stability:     ps.Stability,
			Difficulty:    ps.Difficulty,
			Due:           ps.Due,
			LastReview:    ps.LastReview,
			Reps:          ps.Reps,
			Lapses:        ps.Lapses,
			ScheduledDays: ps.ScheduledDays,
			ElapsedDays:   ps.ElapsedDays,
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
				"state": map[string]any{
					"old": card.State,
					"new": ps.State,
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

	if restoredCard == nil {
		return nil, fmt.Errorf("card restore failed: no result returned")
	}

	s.log.InfoContext(ctx, "review undone",
		slog.String("user_id", userID.String()),
		slog.String("card_id", input.CardID.String()),
		slog.String("undone_grade", string(undoneGrade)),
		slog.String("restored_state", string(restoredState)),
	)

	return restoredCard, nil
}
