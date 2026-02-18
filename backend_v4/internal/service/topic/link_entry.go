package topic

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// LinkEntry links a dictionary entry to a topic. Idempotent — re-linking is not an error.
func (s *Service) LinkEntry(ctx context.Context, input LinkEntryInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check topic ownership
		if _, err := s.topics.GetByID(txCtx, userID, input.TopicID); err != nil {
			return fmt.Errorf("get topic: %w", err)
		}

		// Check entry ownership (also filters soft-deleted)
		if _, err := s.entries.GetByID(txCtx, userID, input.EntryID); err != nil {
			return fmt.Errorf("get entry: %w", err)
		}

		// ON CONFLICT DO NOTHING — idempotent
		if err := s.topics.LinkEntry(txCtx, input.EntryID, input.TopicID); err != nil {
			return fmt.Errorf("link entry: %w", err)
		}

		if auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeTopic,
			EntityID:   &input.TopicID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"linked_entry": map[string]any{"new": input.EntryID},
			},
		}); auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.log.InfoContext(ctx, "entry linked to topic",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
		slog.String("entry_id", input.EntryID.String()),
	)

	return nil
}

// UnlinkEntry removes a link between an entry and a topic. Idempotent — unlinking a non-existent link is not an error.
func (s *Service) UnlinkEntry(ctx context.Context, input UnlinkEntryInput) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return err
	}

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check topic ownership only — no entry check needed
		if _, err := s.topics.GetByID(txCtx, userID, input.TopicID); err != nil {
			return fmt.Errorf("get topic: %w", err)
		}

		// Idempotent — 0 affected rows is not an error
		if err := s.topics.UnlinkEntry(txCtx, input.EntryID, input.TopicID); err != nil {
			return fmt.Errorf("unlink entry: %w", err)
		}

		if auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeTopic,
			EntityID:   &input.TopicID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"unlinked_entry": map[string]any{"old": input.EntryID},
			},
		}); auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.log.InfoContext(ctx, "entry unlinked from topic",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
		slog.String("entry_id", input.EntryID.String()),
	)

	return nil
}

// BatchLinkEntries links multiple entries to a topic.
// Duplicate and non-existent entries are skipped.
func (s *Service) BatchLinkEntries(ctx context.Context, input BatchLinkEntriesInput) (*BatchLinkResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Deduplicate entry IDs before any DB work.
	seen := make(map[uuid.UUID]struct{}, len(input.EntryIDs))
	var uniqueIDs []uuid.UUID
	for _, id := range input.EntryIDs {
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			uniqueIDs = append(uniqueIDs, id)
		}
	}

	var result *BatchLinkResult
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check topic ownership
		if _, err := s.topics.GetByID(txCtx, userID, input.TopicID); err != nil {
			return fmt.Errorf("get topic: %w", err)
		}

		// Filter to existing entries only
		existing, err := s.entries.ExistByIDs(txCtx, userID, uniqueIDs)
		if err != nil {
			return fmt.Errorf("check entries: %w", err)
		}

		var validEntryIDs []uuid.UUID
		for _, id := range uniqueIDs {
			if existing[id] {
				validEntryIDs = append(validEntryIDs, id)
			}
		}

		if len(validEntryIDs) == 0 {
			result = &BatchLinkResult{Linked: 0, Skipped: len(input.EntryIDs)}
			return nil
		}

		linked, linkErr := s.topics.BatchLinkEntries(txCtx, validEntryIDs, input.TopicID)
		if linkErr != nil {
			return fmt.Errorf("batch link entries: %w", linkErr)
		}

		skipped := len(input.EntryIDs) - linked
		result = &BatchLinkResult{Linked: linked, Skipped: skipped}

		if auditErr := s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeTopic,
			EntityID:   &input.TopicID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"batch_linked_entries": map[string]any{
					"entry_ids": validEntryIDs,
					"linked":    linked,
				},
			},
		}); auditErr != nil {
			return fmt.Errorf("audit log: %w", auditErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	s.log.InfoContext(ctx, "entries batch linked",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
		slog.Int("requested", len(input.EntryIDs)),
		slog.Int("linked", result.Linked),
		slog.Int("skipped", result.Skipped),
	)

	return result, nil
}
