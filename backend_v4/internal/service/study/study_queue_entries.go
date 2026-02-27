package study

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetStudyQueueEntries returns entries for the study queue, using batch loading
// to avoid N+1 queries. It preserves the card ordering (due cards first, then new).
func (s *Service) GetStudyQueueEntries(ctx context.Context, input GetQueueInput) ([]*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	cards, err := s.GetStudyQueue(ctx, input)
	if err != nil {
		return nil, err
	}

	if len(cards) == 0 {
		return nil, nil
	}

	// Collect entry IDs preserving card order.
	entryIDs := make([]uuid.UUID, len(cards))
	for i, c := range cards {
		entryIDs[i] = c.EntryID
	}

	// Single batch query instead of N individual queries.
	entriesList, err := s.entries.GetByIDs(ctx, userID, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("batch load entries: %w", err)
	}

	// Index by ID for O(1) lookup.
	byID := make(map[uuid.UUID]*domain.Entry, len(entriesList))
	for i := range entriesList {
		byID[entriesList[i].ID] = &entriesList[i]
	}

	// Preserve card ordering.
	result := make([]*domain.Entry, 0, len(cards))
	for _, c := range cards {
		if e, ok := byID[c.EntryID]; ok {
			result = append(result, e)
		}
	}

	return result, nil
}
