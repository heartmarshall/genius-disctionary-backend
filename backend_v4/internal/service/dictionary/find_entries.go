package dictionary

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 5. FindEntries
// ---------------------------------------------------------------------------

// FindEntries searches and paginates user dictionary entries.
func (s *Service) FindEntries(ctx context.Context, input FindInput) (*FindResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Normalize search text.
	var normalizedSearch *string
	if input.Search != nil {
		n := domain.NormalizeText(*input.Search)
		if n != "" {
			normalizedSearch = &n
		}
	}

	// Clamp limit.
	limit := clampLimit(input.Limit, 1, 200, 20)

	// Defaults.
	sortBy := input.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := input.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	filter := domain.EntryFilter{
		Search:       normalizedSearch,
		HasCard:      input.HasCard,
		PartOfSpeech: input.PartOfSpeech,
		TopicID:      input.TopicID,
		Status:       input.Status,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
		Cursor:       input.Cursor,
		Offset:       input.Offset,
	}

	result := &FindResult{}

	if input.Cursor != nil {
		// Cursor-based pagination.
		entries, hasNext, err := s.entries.FindCursor(ctx, userID, filter)
		if err != nil {
			return nil, fmt.Errorf("find entries (cursor): %w", err)
		}
		result.Entries = entries
		result.HasNextPage = hasNext

		if len(entries) > 0 {
			startID := entries[0].ID.String()
			endID := entries[len(entries)-1].ID.String()
			result.PageInfo = &PageInfo{
				StartCursor: &startID,
				EndCursor:   &endID,
			}
		}
	} else {
		// Offset-based pagination.
		entries, total, err := s.entries.Find(ctx, userID, filter)
		if err != nil {
			return nil, fmt.Errorf("find entries (offset): %w", err)
		}
		result.Entries = entries
		result.TotalCount = total

		offset := 0
		if input.Offset != nil {
			offset = *input.Offset
		}
		result.HasNextPage = offset+len(entries) < total

		if len(entries) > 0 {
			startID := entries[0].ID.String()
			endID := entries[len(entries)-1].ID.String()
			result.PageInfo = &PageInfo{
				StartCursor: &startID,
				EndCursor:   &endID,
			}
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// 6. GetEntry
// ---------------------------------------------------------------------------

// GetEntry returns a single entry by ID.
func (s *Service) GetEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	return s.entries.GetByID(ctx, userID, entryID)
}

// ---------------------------------------------------------------------------
// 7. FindDeletedEntries
// ---------------------------------------------------------------------------

// FindDeletedEntries returns soft-deleted entries for the user.
func (s *Service) FindDeletedEntries(ctx context.Context, limit, offset int) ([]domain.Entry, int, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, 0, domain.ErrUnauthorized
	}

	limit = clampLimit(limit, 1, 200, 20)

	return s.entries.FindDeleted(ctx, userID, limit, offset)
}
