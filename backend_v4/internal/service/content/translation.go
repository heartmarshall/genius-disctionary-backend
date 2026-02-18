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

	trimmedText := strings.TrimSpace(input.Text)
	var translation *domain.Translation

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		_, err := s.senses.GetByIDForUser(txCtx, userID, input.SenseID)
		if err != nil {
			return err
		}

		// Check limit inside tx
		count, err := s.translations.CountBySense(txCtx, input.SenseID)
		if err != nil {
			return fmt.Errorf("count translations: %w", err)
		}
		if count >= MaxTranslationsPerSense {
			return domain.NewValidationError("translations", fmt.Sprintf("limit reached (%d)", MaxTranslationsPerSense))
		}

		// Create translation (trimmed)
		translation, err = s.translations.CreateCustom(txCtx, input.SenseID, trimmedText, "user")
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
				"translation_added": map[string]any{"new": trimmedText},
			},
		})
	})

	if err != nil {
		return nil, err
	}

	s.log.DebugContext(ctx, "translation added",
		slog.String("user_id", userID.String()),
		slog.String("sense_id", input.SenseID.String()),
	)

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

	trimmedText := strings.TrimSpace(input.Text)
	var translation *domain.Translation

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		oldTranslation, err := s.translations.GetByIDForUser(txCtx, userID, input.TranslationID)
		if err != nil {
			return err
		}

		// Update translation (ref_translation_id NOT touched)
		translation, err = s.translations.Update(txCtx, input.TranslationID, trimmedText)
		if err != nil {
			return fmt.Errorf("update translation: %w", err)
		}

		// Audit on parent SENSE — skip if nothing changed
		oldText := ""
		if oldTranslation.Text != nil {
			oldText = *oldTranslation.Text
		}
		if oldText == trimmedText {
			return nil
		}

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeSense,
			EntityID:   &oldTranslation.SenseID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"translation_text": map[string]any{
					"old": oldText,
					"new": trimmedText,
				},
			},
		})
	})

	if err != nil {
		return nil, err
	}

	s.log.DebugContext(ctx, "translation updated",
		slog.String("user_id", userID.String()),
		slog.String("translation_id", input.TranslationID.String()),
	)

	return translation, nil
}

// DeleteTranslation deletes a translation.
func (s *Service) DeleteTranslation(ctx context.Context, translationID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		translation, err := s.translations.GetByIDForUser(txCtx, userID, translationID)
		if err != nil {
			return err
		}

		// Delete translation
		if err := s.translations.Delete(txCtx, translationID); err != nil {
			return fmt.Errorf("delete translation: %w", err)
		}

		// Audit on parent SENSE
		oldText := ""
		if translation.Text != nil {
			oldText = *translation.Text
		}

		s.log.InfoContext(txCtx, "translation deleted",
			slog.String("user_id", userID.String()),
			slog.String("translation_id", translationID.String()),
		)

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

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		_, err := s.senses.GetByIDForUser(txCtx, userID, input.SenseID)
		if err != nil {
			return err
		}

		// Validate items belong to sense
		existingTranslations, err := s.translations.GetBySenseID(txCtx, input.SenseID)
		if err != nil {
			return fmt.Errorf("get translations: %w", err)
		}

		existingIDs := make(map[uuid.UUID]bool, len(existingTranslations))
		for _, tr := range existingTranslations {
			existingIDs[tr.ID] = true
		}

		for _, item := range input.Items {
			if !existingIDs[item.ID] {
				return domain.NewValidationError("items", fmt.Sprintf("translation does not belong to this sense: %s", item.ID))
			}
		}

		return s.translations.Reorder(txCtx, input.Items)
	})

	if err != nil {
		return err
	}

	s.log.DebugContext(ctx, "translations reordered",
		slog.String("user_id", userID.String()),
		slog.String("sense_id", input.SenseID.String()),
	)

	return nil
}
