package dictionary

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 8. DeleteEntry
// ---------------------------------------------------------------------------

// DeleteEntry soft-deletes an entry.
func (s *Service) DeleteEntry(ctx context.Context, entryID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Get entry for audit text.
	entry, err := s.entries.GetByID(ctx, userID, entryID)
	if err != nil {
		return err
	}

	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if delErr := s.entries.SoftDelete(txCtx, userID, entryID); delErr != nil {
			return fmt.Errorf("soft delete: %w", delErr)
		}

		_, auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &entryID,
			Action:     domain.AuditActionDelete,
			Changes:    map[string]any{"text": entry.Text},
		})
		if auditErr != nil {
			return fmt.Errorf("audit delete: %w", auditErr)
		}

		return nil
	})

	return txErr
}

// ---------------------------------------------------------------------------
// 10. RestoreEntry
// ---------------------------------------------------------------------------

// RestoreEntry restores a soft-deleted entry.
func (s *Service) RestoreEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	restored, err := s.entries.Restore(ctx, userID, entryID)
	if err != nil {
		return nil, err
	}

	return restored, nil
}

// ---------------------------------------------------------------------------
// 11. BatchDeleteEntries
// ---------------------------------------------------------------------------

// BatchDeleteEntries soft-deletes multiple entries. NOT transactional, partial failure OK.
func (s *Service) BatchDeleteEntries(ctx context.Context, entryIDs []uuid.UUID) (*BatchResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if len(entryIDs) == 0 {
		return nil, domain.NewValidationError("entry_ids", "required (at least 1)")
	}
	if len(entryIDs) > 200 {
		return nil, domain.NewValidationError("entry_ids", "too many (max 200)")
	}

	result := &BatchResult{}

	for _, eid := range entryIDs {
		if delErr := s.entries.SoftDelete(ctx, userID, eid); delErr != nil {
			result.Errors = append(result.Errors, BatchError{
				EntryID: eid,
				Error:   delErr.Error(),
			})
		} else {
			result.Deleted++
		}
	}

	// Write a single audit record if any were deleted.
	if result.Deleted > 0 {
		ids := make([]string, 0, result.Deleted)
		for _, eid := range entryIDs {
			// Only include successfully deleted IDs.
			failed := false
			for _, be := range result.Errors {
				if be.EntryID == eid {
					failed = true
					break
				}
			}
			if !failed {
				ids = append(ids, eid.String())
			}
		}

		_, auditErr := s.audit.Create(ctx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			Action:     domain.AuditActionDelete,
			Changes:    map[string]any{"batch_delete": ids, "count": result.Deleted},
		})
		if auditErr != nil {
			s.log.ErrorContext(ctx, "batch delete audit error",
				slog.String("error", auditErr.Error()),
			)
		}
	}

	return result, nil
}
