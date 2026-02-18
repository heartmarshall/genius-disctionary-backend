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
// 3. CreateEntryFromCatalog
// ---------------------------------------------------------------------------

// CreateEntryFromCatalog creates a new dictionary entry from a reference catalog entry.
func (s *Service) CreateEntryFromCatalog(ctx context.Context, input CreateFromCatalogInput) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get reference entry.
	refEntry, err := s.refCatalog.GetRefEntry(ctx, input.RefEntryID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewValidationError("ref_entry_id", "reference entry not found")
		}
		return nil, fmt.Errorf("get ref entry: %w", err)
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
	_, err = s.entries.GetByText(ctx, userID, refEntry.TextNormalized)
	if err == nil {
		return nil, domain.ErrAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}

	// Determine which senses to use.
	var selectedSenses []domain.RefSense
	if len(input.SenseIDs) == 0 {
		// Use all senses from the reference entry.
		selectedSenses = refEntry.Senses
	} else {
		// Filter to only requested senses, preserving order.
		senseMap := make(map[uuid.UUID]domain.RefSense, len(refEntry.Senses))
		for _, rs := range refEntry.Senses {
			senseMap[rs.ID] = rs
		}
		for _, senseID := range input.SenseIDs {
			rs, found := senseMap[senseID]
			if !found {
				return nil, domain.NewValidationError("sense_ids", "sense not found: "+senseID.String())
			}
			selectedSenses = append(selectedSenses, rs)
		}
	}

	// Create entry in transaction.
	var created *domain.Entry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		now := time.Now().UTC()
		entry := &domain.Entry{
			ID:             uuid.New(),
			UserID:         userID,
			RefEntryID:     &refEntry.ID,
			Text:           refEntry.Text,
			TextNormalized: refEntry.TextNormalized,
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
		for _, rs := range selectedSenses {
			sense, senseErr := s.senses.CreateFromRef(txCtx, created.ID, rs.ID, rs.SourceSlug)
			if senseErr != nil {
				return fmt.Errorf("create sense from ref: %w", senseErr)
			}

			// Translations for this sense.
			for _, rt := range rs.Translations {
				if _, trErr := s.translations.CreateFromRef(txCtx, sense.ID, rt.ID, rt.SourceSlug); trErr != nil {
					return fmt.Errorf("create translation from ref: %w", trErr)
				}
			}

			// Examples for this sense.
			for _, re := range rs.Examples {
				if _, exErr := s.examples.CreateFromRef(txCtx, sense.ID, re.ID, re.SourceSlug); exErr != nil {
					return fmt.Errorf("create example from ref: %w", exErr)
				}
			}
		}

		// Link pronunciations.
		for _, rp := range refEntry.Pronunciations {
			if linkErr := s.pronunciations.Link(txCtx, created.ID, rp.ID); linkErr != nil {
				return fmt.Errorf("link pronunciation: %w", linkErr)
			}
		}

		// Link images.
		for _, ri := range refEntry.Images {
			if linkErr := s.images.LinkCatalog(txCtx, created.ID, ri.ID); linkErr != nil {
				return fmt.Errorf("link image: %w", linkErr)
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
			Changes:    map[string]any{"text": created.Text, "source": "catalog"},
		})
		if auditErr != nil {
			return fmt.Errorf("audit create: %w", auditErr)
		}

		return nil
	})

	if txErr != nil {
		// Handle unique constraint violation from concurrent create.
		if errors.Is(txErr, domain.ErrAlreadyExists) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, txErr
	}

	return created, nil
}
