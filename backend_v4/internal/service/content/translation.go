package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// Translation Operations
// ---------------------------------------------------------------------------

// AddTranslation adds a new translation to a sense.
func (s *Service) AddTranslation(ctx context.Context, input AddTranslationInput) (*domain.Translation, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check ownership
	_, entry, err := s.checkSenseOwnership(ctx, userID, input.SenseID)
	if err != nil {
		return nil, err
	}
	_ = entry

	// Check limit
	count, err := s.translations.CountBySense(ctx, input.SenseID)
	if err != nil {
		return nil, fmt.Errorf("count translations: %w", err)
	}
	if count >= MaxTranslationsPerSense {
		return nil, domain.NewValidationError("translations", fmt.Sprintf("limit reached (%d)", MaxTranslationsPerSense))
	}

	var translation *domain.Translation

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Create translation
		translation, err = s.translations.CreateCustom(txCtx, input.SenseID, input.Text, "user")
		if err != nil {
			return fmt.Errorf("create translation: %w", err)
		}

		// Audit on parent SENSE
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &input.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"translation_added": map[string]any{"new": input.Text},
			},
		})
	})

	if err != nil {
		return nil, err
	}

	return translation, nil
}

// UpdateTranslation updates a translation's text.
func (s *Service) UpdateTranslation(ctx context.Context, input UpdateTranslationInput) (*domain.Translation, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check ownership
	oldTranslation, _, err := s.checkTranslationOwnership(ctx, userID, input.TranslationID)
	if err != nil {
		return nil, err
	}

	var translation *domain.Translation

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Update translation (ref_translation_id NOT touched)
		translation, err = s.translations.Update(txCtx, input.TranslationID, input.Text)
		if err != nil {
			return fmt.Errorf("update translation: %w", err)
		}

		// Audit on parent SENSE
		oldText := ""
		if oldTranslation.Text != nil {
			oldText = *oldTranslation.Text
		}
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &oldTranslation.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"translation_text": map[string]any{
					"old": oldText,
					"new": input.Text,
				},
			},
		})
	})

	if err != nil {
		return nil, err
	}

	return translation, nil
}

// DeleteTranslation deletes a translation.
func (s *Service) DeleteTranslation(ctx context.Context, translationID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Check ownership
	translation, _, err := s.checkTranslationOwnership(ctx, userID, translationID)
	if err != nil {
		return err
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Delete translation
		err := s.translations.Delete(txCtx, translationID)
		if err != nil {
			return fmt.Errorf("delete translation: %w", err)
		}

		// Audit on parent SENSE
		oldText := ""
		if translation.Text != nil {
			oldText = *translation.Text
		}
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &translation.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"translation_deleted": map[string]any{"old": oldText},
			},
		})
	})
}

// ReorderTranslations reorders translations within a sense.
func (s *Service) ReorderTranslations(ctx context.Context, input ReorderTranslationsInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	// Check ownership
	_, _, err := s.checkSenseOwnership(ctx, userID, input.SenseID)
	if err != nil {
		return err
	}

	// Validate items belong to sense
	existingTranslations, err := s.translations.GetBySenseID(ctx, input.SenseID)
	if err != nil {
		return fmt.Errorf("get translations: %w", err)
	}

	existingIDs := make(map[uuid.UUID]bool)
	for _, tr := range existingTranslations {
		existingIDs[tr.ID] = true
	}

	for _, item := range input.Items {
		if !existingIDs[item.ID] {
			return domain.NewValidationError("items", fmt.Sprintf("translation does not belong to this sense: %s", item.ID))
		}
	}

	// Reorder (no audit for reorder operations)
	return s.translations.Reorder(ctx, input.Items)
}

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

	// Check ownership
	_, entry, err := s.checkSenseOwnership(ctx, userID, input.SenseID)
	if err != nil {
		return nil, err
	}
	_ = entry

	// Check limit
	count, err := s.examples.CountBySense(ctx, input.SenseID)
	if err != nil {
		return nil, fmt.Errorf("count examples: %w", err)
	}
	if count >= MaxExamplesPerSense {
		return nil, domain.NewValidationError("examples", fmt.Sprintf("limit reached (%d)", MaxExamplesPerSense))
	}

	var example *domain.Example

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Create example
		example, err = s.examples.CreateCustom(txCtx, input.SenseID, input.Sentence, input.Translation, "user")
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
				"example_added": map[string]any{"new": input.Sentence},
			},
		})
	})

	if err != nil {
		return nil, err
	}

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

	// Check ownership
	oldExample, _, err := s.checkExampleOwnership(ctx, userID, input.ExampleID)
	if err != nil {
		return nil, err
	}

	var example *domain.Example

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Update example (translation=nil → remove translation)
		example, err = s.examples.Update(txCtx, input.ExampleID, input.Sentence, input.Translation)
		if err != nil {
			return fmt.Errorf("update example: %w", err)
		}

		// Audit on parent SENSE
		changes := buildExampleChanges(oldExample, &input)
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

	return example, nil
}

// DeleteExample deletes an example.
func (s *Service) DeleteExample(ctx context.Context, exampleID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Check ownership
	example, _, err := s.checkExampleOwnership(ctx, userID, exampleID)
	if err != nil {
		return err
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Delete example
		err := s.examples.Delete(txCtx, exampleID)
		if err != nil {
			return fmt.Errorf("delete example: %w", err)
		}

		// Audit on parent SENSE
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &example.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"example_deleted": map[string]any{"old": *example.Sentence},
			},
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

	// Check ownership
	_, _, err := s.checkSenseOwnership(ctx, userID, input.SenseID)
	if err != nil {
		return err
	}

	// Validate items belong to sense
	existingExamples, err := s.examples.GetBySenseID(ctx, input.SenseID)
	if err != nil {
		return fmt.Errorf("get examples: %w", err)
	}

	existingIDs := make(map[uuid.UUID]bool)
	for _, ex := range existingExamples {
		existingIDs[ex.ID] = true
	}

	for _, item := range input.Items {
		if !existingIDs[item.ID] {
			return domain.NewValidationError("items", fmt.Sprintf("example does not belong to this sense: %s", item.ID))
		}
	}

	// Reorder (no audit for reorder operations)
	return s.examples.Reorder(ctx, input.Items)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildExampleChanges compares old and new example fields and returns audit changes.
func buildExampleChanges(old *domain.Example, input *UpdateExampleInput) map[string]any {
	changes := make(map[string]any)

	// Sentence
	oldSentence := ""
	if old.Sentence != nil {
		oldSentence = *old.Sentence
	}
	if oldSentence != input.Sentence {
		changes["sentence"] = map[string]any{"old": oldSentence, "new": input.Sentence}
	}

	// Translation
	oldTranslation := ""
	if old.Translation != nil {
		oldTranslation = *old.Translation
	}
	newTranslation := ""
	if input.Translation != nil {
		newTranslation = *input.Translation
	}
	if oldTranslation != newTranslation {
		changes["translation"] = map[string]any{"old": oldTranslation, "new": newTranslation}
	}

	return changes
}
