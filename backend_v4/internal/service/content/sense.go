package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// AddSense adds a new sense to an entry with optional translations.
func (s *Service) AddSense(ctx context.Context, input AddSenseInput) (*domain.Sense, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check ownership
	_, err := s.checkEntryOwnership(ctx, userID, input.EntryID)
	if err != nil {
		return nil, err
	}

	// Check limit
	count, err := s.senses.CountByEntry(ctx, input.EntryID)
	if err != nil {
		return nil, fmt.Errorf("count senses: %w", err)
	}
	if count >= MaxSensesPerEntry {
		return nil, domain.NewValidationError("senses", fmt.Sprintf("limit reached (%d)", MaxSensesPerEntry))
	}

	var sense *domain.Sense

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Create sense
		sense, err = s.senses.CreateCustom(txCtx, input.EntryID, input.Definition, input.PartOfSpeech, input.CEFRLevel, "user")
		if err != nil {
			return fmt.Errorf("create sense: %w", err)
		}

		// Create translations
		for _, text := range input.Translations {
			_, err = s.translations.CreateCustom(txCtx, sense.ID, text, "user")
			if err != nil {
				return fmt.Errorf("create translation: %w", err)
			}
		}

		// Audit
		changes := map[string]any{
			"entry_id": map[string]any{"new": input.EntryID},
		}
		if input.Definition != nil {
			changes["definition"] = map[string]any{"new": *input.Definition}
		}
		if len(input.Translations) > 0 {
			changes["translations_count"] = map[string]any{"new": len(input.Translations)}
		}

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &sense.ID,
			Action:     domain.AuditActionCreate,
			Changes:    changes,
		})
	})

	if err != nil {
		return nil, err
	}

	return sense, nil
}

// UpdateSense updates a sense's fields. Nil fields are not changed.
func (s *Service) UpdateSense(ctx context.Context, input UpdateSenseInput) (*domain.Sense, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check ownership
	oldSense, _, err := s.checkSenseOwnership(ctx, userID, input.SenseID)
	if err != nil {
		return nil, err
	}

	var sense *domain.Sense

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Update sense
		sense, err = s.senses.Update(txCtx, input.SenseID, input.Definition, input.PartOfSpeech, input.CEFRLevel)
		if err != nil {
			return fmt.Errorf("update sense: %w", err)
		}

		// Audit with changes
		changes := buildSenseChanges(oldSense, &input)

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &input.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
		})
	})

	if err != nil {
		return nil, err
	}

	return sense, nil
}

// DeleteSense deletes a sense (CASCADE deletes translations/examples via FK).
func (s *Service) DeleteSense(ctx context.Context, senseID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Check ownership
	sense, _, err := s.checkSenseOwnership(ctx, userID, senseID)
	if err != nil {
		return err
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Delete sense
		err := s.senses.Delete(txCtx, senseID)
		if err != nil {
			return fmt.Errorf("delete sense: %w", err)
		}

		// Audit
		changes := map[string]any{
			"entry_id": map[string]any{"old": sense.EntryID},
		}
		if sense.Definition != nil {
			changes["definition"] = map[string]any{"old": *sense.Definition}
		}

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &senseID,
			Action:     domain.AuditActionDelete,
			Changes:    changes,
		})
	})
}

// ReorderSenses reorders senses within an entry.
func (s *Service) ReorderSenses(ctx context.Context, input ReorderSensesInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	// Check ownership
	_, err := s.checkEntryOwnership(ctx, userID, input.EntryID)
	if err != nil {
		return err
	}

	// Validate items belong to entry
	existingSenses, err := s.senses.GetByEntryID(ctx, input.EntryID)
	if err != nil {
		return fmt.Errorf("get senses: %w", err)
	}

	existingIDs := make(map[uuid.UUID]bool)
	for _, sense := range existingSenses {
		existingIDs[sense.ID] = true
	}

	for _, item := range input.Items {
		if !existingIDs[item.ID] {
			return domain.NewValidationError("items", fmt.Sprintf("sense does not belong to this entry: %s", item.ID))
		}
	}

	// Reorder
	return s.senses.Reorder(ctx, input.Items)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildSenseChanges compares old and new sense fields and returns audit changes.
func buildSenseChanges(old *domain.Sense, input *UpdateSenseInput) map[string]any {
	changes := make(map[string]any)

	// Definition
	if input.Definition != nil {
		oldVal := ""
		if old.Definition != nil {
			oldVal = *old.Definition
		}
		newVal := *input.Definition
		if oldVal != newVal {
			changes["definition"] = map[string]any{"old": oldVal, "new": newVal}
		}
	}

	// PartOfSpeech
	if input.PartOfSpeech != nil {
		var oldVal string
		if old.PartOfSpeech != nil {
			oldVal = string(*old.PartOfSpeech)
		}
		newVal := string(*input.PartOfSpeech)
		if oldVal != newVal {
			changes["part_of_speech"] = map[string]any{"old": oldVal, "new": newVal}
		}
	}

	// CEFRLevel
	if input.CEFRLevel != nil {
		oldVal := ""
		if old.CEFRLevel != nil {
			oldVal = *old.CEFRLevel
		}
		newVal := *input.CEFRLevel
		if oldVal != newVal {
			changes["cefr_level"] = map[string]any{"old": oldVal, "new": newVal}
		}
	}

	return changes
}
