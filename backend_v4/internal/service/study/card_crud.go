package study

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// CreateCard creates a study card for an entry. Entry must have at least one sense.
func (s *Service) CreateCard(ctx context.Context, input CreateCardInput) (*domain.Card, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check entry exists
	_, err := s.entries.GetByID(ctx, userID, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("get entry: %w", err)
	}

	// Check entry has senses
	senseCount, err := s.senses.CountByEntryID(ctx, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("count senses: %w", err)
	}
	if senseCount == 0 {
		return nil, domain.NewValidationError("entry_id", "entry must have at least one sense to create a card")
	}

	var card *domain.Card

	// Transaction: create card + audit
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var createErr error
		card, createErr = s.cards.Create(txCtx, userID, input.EntryID)
		if createErr != nil {
			return fmt.Errorf("create card: %w", createErr)
		}

		// Audit
		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeCard,
			EntityID:   &card.ID,
			Action:     domain.AuditActionCreate,
			Changes: map[string]any{
				"entry_id": map[string]any{"new": input.EntryID},
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

	s.log.InfoContext(ctx, "card created",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()),
		slog.String("entry_id", input.EntryID.String()),
	)

	return card, nil
}

// DeleteCard deletes a study card. Entry remains in dictionary.
func (s *Service) DeleteCard(ctx context.Context, input DeleteCardInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	// Load card to check ownership
	card, err := s.cards.GetByID(ctx, userID, input.CardID)
	if err != nil {
		return fmt.Errorf("get card: %w", err)
	}

	// Transaction: delete card + audit
	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Delete card (CASCADE deletes review_logs)
		if deleteErr := s.cards.Delete(txCtx, userID, input.CardID); deleteErr != nil {
			return fmt.Errorf("delete card: %w", deleteErr)
		}

		// Audit
		auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeCard,
			EntityID:   &card.ID,
			Action:     domain.AuditActionDelete,
			Changes: map[string]any{
				"entry_id": map[string]any{"old": card.EntryID},
			},
		})
		if auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})

	if err != nil {
		return err
	}

	s.log.InfoContext(ctx, "card deleted",
		slog.String("user_id", userID.String()),
		slog.String("card_id", card.ID.String()),
		slog.String("entry_id", card.EntryID.String()),
	)

	return nil
}

// BatchCreateCards creates cards for multiple entries in batch with partial success.
func (s *Service) BatchCreateCards(ctx context.Context, input BatchCreateCardsInput) (BatchCreateResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return BatchCreateResult{}, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return BatchCreateResult{}, err
	}

	result := BatchCreateResult{
		Errors: []BatchCreateError{},
	}

	// Check which entries exist
	existMap, err := s.entries.ExistByIDs(ctx, userID, input.EntryIDs)
	if err != nil {
		return result, fmt.Errorf("check entries exist: %w", err)
	}

	// Filter to existing entries only
	existingEntryIDs := []uuid.UUID{}
	for _, entryID := range input.EntryIDs {
		if exists, ok := existMap[entryID]; !ok || !exists {
			result.Errors = append(result.Errors, BatchCreateError{
				EntryID: entryID,
				Reason:  "entry not found",
			})
		} else {
			existingEntryIDs = append(existingEntryIDs, entryID)
		}
	}

	if len(existingEntryIDs) == 0 {
		// All entries not found - return result with errors
		return result, nil
	}

	// Check which entries already have cards
	cardExistsMap, err := s.cards.ExistsByEntryIDs(ctx, userID, existingEntryIDs)
	if err != nil {
		return result, fmt.Errorf("check cards exist: %w", err)
	}

	// Filter to entries without cards
	entriesToCreate := []uuid.UUID{}
	for _, entryID := range existingEntryIDs {
		if exists, ok := cardExistsMap[entryID]; ok && exists {
			result.SkippedExisting++
		} else {
			entriesToCreate = append(entriesToCreate, entryID)
		}
	}

	if len(entriesToCreate) == 0 {
		// All entries already have cards
		return result, nil
	}

	// Batch count senses for all entries at once (eliminates N+1)
	senseCounts, err := s.senses.CountByEntryIDs(ctx, entriesToCreate)
	if err != nil {
		return result, fmt.Errorf("count senses batch: %w", err)
	}

	finalEntriesToCreate := []uuid.UUID{}
	for _, entryID := range entriesToCreate {
		if cnt, ok := senseCounts[entryID]; !ok || cnt == 0 {
			result.SkippedNoSenses++
		} else {
			finalEntriesToCreate = append(finalEntriesToCreate, entryID)
		}
	}

	// Create cards for valid entries
	for _, entryID := range finalEntriesToCreate {
		err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
			createdCard, createErr := s.cards.Create(txCtx, userID, entryID)
			if createErr != nil {
				return fmt.Errorf("create card: %w", createErr)
			}

			// Audit
			auditErr := s.audit.Log(txCtx, domain.AuditRecord{
				UserID:     userID,
				EntityType: domain.EntityTypeCard,
				EntityID:   &createdCard.ID,
				Action:     domain.AuditActionCreate,
				Changes: map[string]any{
					"entry_id": map[string]any{"new": entryID},
				},
			})
			if auditErr != nil {
				return fmt.Errorf("audit log: %w", auditErr)
			}

			return nil
		})

		if err != nil {
			result.Errors = append(result.Errors, BatchCreateError{
				EntryID: entryID,
				Reason:  err.Error(),
			})
		} else {
			result.Created++
		}
	}

	s.log.InfoContext(ctx, "batch card creation completed",
		slog.String("user_id", userID.String()),
		slog.Int("created", result.Created),
		slog.Int("skipped_existing", result.SkippedExisting),
		slog.Int("skipped_no_senses", result.SkippedNoSenses),
		slog.Int("errors", len(result.Errors)),
	)

	return result, nil
}
