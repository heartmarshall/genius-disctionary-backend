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

	// Check topic ownership
	if _, err := s.topics.GetByID(ctx, userID, input.TopicID); err != nil {
		return fmt.Errorf("get topic: %w", err)
	}

	// Check entry ownership (also filters soft-deleted)
	if _, err := s.entries.GetByID(ctx, userID, input.EntryID); err != nil {
		return fmt.Errorf("get entry: %w", err)
	}

	// ON CONFLICT DO NOTHING — idempotent
	if err := s.topics.LinkEntry(ctx, input.EntryID, input.TopicID); err != nil {
		return fmt.Errorf("link entry: %w", err)
	}

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

	// Check topic ownership only — no entry check needed
	if _, err := s.topics.GetByID(ctx, userID, input.TopicID); err != nil {
		return fmt.Errorf("get topic: %w", err)
	}

	// Idempotent — 0 affected rows is not an error
	if err := s.topics.UnlinkEntry(ctx, input.EntryID, input.TopicID); err != nil {
		return fmt.Errorf("unlink entry: %w", err)
	}

	return nil
}

// BatchLinkEntries links multiple entries to a topic. Entries that do not exist are skipped.
func (s *Service) BatchLinkEntries(ctx context.Context, input BatchLinkEntriesInput) (*BatchLinkResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check topic ownership
	if _, err := s.topics.GetByID(ctx, userID, input.TopicID); err != nil {
		return nil, fmt.Errorf("get topic: %w", err)
	}

	// Filter to existing entries only
	existing, err := s.entries.ExistByIDs(ctx, userID, input.EntryIDs)
	if err != nil {
		return nil, fmt.Errorf("check entries: %w", err)
	}

	var validEntryIDs []uuid.UUID
	for _, id := range input.EntryIDs {
		if existing[id] {
			validEntryIDs = append(validEntryIDs, id)
		}
	}

	if len(validEntryIDs) == 0 {
		return &BatchLinkResult{Linked: 0, Skipped: len(input.EntryIDs)}, nil
	}

	linked, err := s.topics.BatchLinkEntries(ctx, validEntryIDs, input.TopicID)
	if err != nil {
		return nil, fmt.Errorf("batch link entries: %w", err)
	}

	skipped := len(input.EntryIDs) - linked

	s.log.InfoContext(ctx, "entries batch linked",
		slog.String("user_id", userID.String()),
		slog.String("topic_id", input.TopicID.String()),
		slog.Int("requested", len(input.EntryIDs)),
		slog.Int("linked", linked),
		slog.Int("skipped", skipped),
	)

	return &BatchLinkResult{Linked: linked, Skipped: skipped}, nil
}
