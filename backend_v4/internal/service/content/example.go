package content

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// Example Operations
// ---------------------------------------------------------------------------

// AddExample adds a new example to a sense.
func (s *Service) AddExample(ctx context.Context, input AddExampleInput) (*domain.Example, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	trimmedSentence := strings.TrimSpace(input.Sentence)
	var trimmedTranslation *string
	if input.Translation != nil {
		t := strings.TrimSpace(*input.Translation)
		trimmedTranslation = &t
	}

	var example *domain.Example

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		_, err := s.senses.GetByIDForUser(txCtx, userID, input.SenseID)
		if err != nil {
			return err
		}

		// Check limit inside tx
		count, err := s.examples.CountBySense(txCtx, input.SenseID)
		if err != nil {
			return fmt.Errorf("count examples: %w", err)
		}
		if count >= MaxExamplesPerSense {
			return domain.NewValidationError("examples", fmt.Sprintf("limit reached (%d)", MaxExamplesPerSense))
		}

		// Create example (trimmed)
		example, err = s.examples.CreateCustom(txCtx, input.SenseID, trimmedSentence, trimmedTranslation, "user")
		if err != nil {
			return fmt.Errorf("create example: %w", err)
		}

		// Audit on parent SENSE
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &input.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"example_added": map[string]any{"new": trimmedSentence},
			},
		})
	})

	if err != nil {
		return nil, err
	}

	s.log.DebugContext(ctx, "example added",
		slog.String("user_id", userID.String()),
		slog.String("sense_id", input.SenseID.String()),
	)

	return example, nil
}

// UpdateExample updates an example's sentence and/or translation.
// translation=nil means remove translation (set to NULL in DB).
func (s *Service) UpdateExample(ctx context.Context, input UpdateExampleInput) (*domain.Example, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	trimmedSentence := strings.TrimSpace(input.Sentence)
	var trimmedTranslation *string
	if input.Translation != nil {
		t := strings.TrimSpace(*input.Translation)
		trimmedTranslation = &t
	}

	var example *domain.Example

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		oldExample, err := s.examples.GetByIDForUser(txCtx, userID, input.ExampleID)
		if err != nil {
			return err
		}

		// Update example (translation=nil → remove translation)
		example, err = s.examples.Update(txCtx, input.ExampleID, trimmedSentence, trimmedTranslation)
		if err != nil {
			return fmt.Errorf("update example: %w", err)
		}

		// Audit on parent SENSE — skip if nothing changed
		changes := buildExampleChanges(oldExample, trimmedSentence, trimmedTranslation)
		if len(changes) == 0 {
			return nil
		}

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &oldExample.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
		})
	})

	if err != nil {
		return nil, err
	}

	s.log.DebugContext(ctx, "example updated",
		slog.String("user_id", userID.String()),
		slog.String("example_id", input.ExampleID.String()),
	)

	return example, nil
}

// DeleteExample deletes an example.
func (s *Service) DeleteExample(ctx context.Context, exampleID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		example, err := s.examples.GetByIDForUser(txCtx, userID, exampleID)
		if err != nil {
			return err
		}

		// Delete example
		if err := s.examples.Delete(txCtx, exampleID); err != nil {
			return fmt.Errorf("delete example: %w", err)
		}

		// Audit on parent SENSE
		changes := map[string]any{}
		if example.Sentence != nil {
			changes["example_deleted"] = map[string]any{"old": *example.Sentence}
		}

		s.log.InfoContext(txCtx, "example deleted",
			slog.String("user_id", userID.String()),
			slog.String("example_id", exampleID.String()),
		)

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &example.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
		})
	})
}

// ReorderExamples reorders examples within a sense.
func (s *Service) ReorderExamples(ctx context.Context, input ReorderExamplesInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		_, err := s.senses.GetByIDForUser(txCtx, userID, input.SenseID)
		if err != nil {
			return err
		}

		// Validate items belong to sense
		existingExamples, err := s.examples.GetBySenseID(txCtx, input.SenseID)
		if err != nil {
			return fmt.Errorf("get examples: %w", err)
		}

		existingIDs := make(map[uuid.UUID]bool, len(existingExamples))
		for _, ex := range existingExamples {
			existingIDs[ex.ID] = true
		}

		for _, item := range input.Items {
			if !existingIDs[item.ID] {
				return domain.NewValidationError("items", fmt.Sprintf("example does not belong to this sense: %s", item.ID))
			}
		}

		return s.examples.Reorder(txCtx, input.Items)
	})

	if err != nil {
		return err
	}

	s.log.DebugContext(ctx, "examples reordered",
		slog.String("user_id", userID.String()),
		slog.String("sense_id", input.SenseID.String()),
	)

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildExampleChanges compares old and new example fields and returns audit changes.
func buildExampleChanges(old *domain.Example, newSentence string, newTranslation *string) map[string]any {
	changes := make(map[string]any)

	// Sentence
	oldSentence := ""
	if old.Sentence != nil {
		oldSentence = *old.Sentence
	}
	if oldSentence != newSentence {
		changes["sentence"] = map[string]any{"old": oldSentence, "new": newSentence}
	}

	// Translation
	oldTranslation := ""
	if old.Translation != nil {
		oldTranslation = *old.Translation
	}
	newTr := ""
	if newTranslation != nil {
		newTr = *newTranslation
	}
	if oldTranslation != newTr {
		changes["translation"] = map[string]any{"old": oldTranslation, "new": newTr}
	}

	return changes
}
