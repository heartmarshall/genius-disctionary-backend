package dictionary

import (
	"context"
	"fmt"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 7. UpdateNotes
// ---------------------------------------------------------------------------

// UpdateNotes updates the notes for an entry.
func (s *Service) UpdateNotes(ctx context.Context, input UpdateNotesInput) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get old entry for audit diff.
	oldEntry, err := s.entries.GetByID(ctx, userID, input.EntryID)
	if err != nil {
		return nil, err
	}

	var updated *domain.Entry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var updateErr error
		updated, updateErr = s.entries.UpdateNotes(txCtx, userID, input.EntryID, input.Notes)
		if updateErr != nil {
			return fmt.Errorf("update notes: %w", updateErr)
		}

		changes := map[string]any{
			"old_notes": oldEntry.Notes,
			"new_notes": input.Notes,
		}

		_, auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &input.EntryID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
		})
		if auditErr != nil {
			return fmt.Errorf("audit update: %w", auditErr)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return updated, nil
}
