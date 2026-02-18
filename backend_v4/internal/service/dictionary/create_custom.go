package dictionary

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 4. CreateEntryCustom
// ---------------------------------------------------------------------------

// CreateEntryCustom creates a new custom dictionary entry.
func (s *Service) CreateEntryCustom(ctx context.Context, input CreateCustomInput) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	normalized := domain.NormalizeText(input.Text)
	if normalized == "" {
		return nil, domain.NewValidationError("text", "required")
	}

	// Check entry limit.
	count, err := s.entries.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	if count >= s.cfg.MaxEntriesPerUser {
		return nil, domain.NewValidationError("entries", "limit reached")
	}

	// Duplicate check.
	_, err = s.entries.GetByText(ctx, userID, normalized)
	if err == nil {
		return nil, domain.ErrAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}

	const sourceSlug = "user"

	var created *domain.Entry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		now := time.Now().UTC()
		entry := &domain.Entry{
			ID:             uuid.New(),
			UserID:         userID,
			Text:           input.Text,
			TextNormalized: normalized,
			Notes:          input.Notes,
			CreatedAt:      now,
			UpdatedAt:      now,
		}

		var createErr error
		created, createErr = s.entries.Create(txCtx, entry)
		if createErr != nil {
			return fmt.Errorf("create entry: %w", createErr)
		}

		// Create senses and their children.
		for _, si := range input.Senses {
			sense, senseErr := s.senses.CreateCustom(txCtx, created.ID, si.Definition, si.PartOfSpeech, nil, sourceSlug)
			if senseErr != nil {
				return fmt.Errorf("create custom sense: %w", senseErr)
			}

			for _, tr := range si.Translations {
				if _, trErr := s.translations.CreateCustom(txCtx, sense.ID, tr, sourceSlug); trErr != nil {
					return fmt.Errorf("create custom translation: %w", trErr)
				}
			}

			for _, ex := range si.Examples {
				if _, exErr := s.examples.CreateCustom(txCtx, sense.ID, ex.Sentence, ex.Translation, sourceSlug); exErr != nil {
					return fmt.Errorf("create custom example: %w", exErr)
				}
			}
		}

		// Create card if requested.
		if input.CreateCard {
			if _, cardErr := s.cards.Create(txCtx, userID, created.ID, domain.LearningStatusNew, s.cfg.DefaultEaseFactor); cardErr != nil {
				return fmt.Errorf("create card: %w", cardErr)
			}
		}

		// Audit.
		_, auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &created.ID,
			Action:     domain.AuditActionCreate,
			Changes:    map[string]any{"text": created.Text, "source": sourceSlug},
		})
		if auditErr != nil {
			return fmt.Errorf("audit create: %w", auditErr)
		}

		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, domain.ErrAlreadyExists) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, txErr
	}

	return created, nil
}
