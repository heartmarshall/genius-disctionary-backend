package study

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

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
