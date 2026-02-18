package refcatalog

import (
	"context"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Search finds reference entries matching the query. The catalog is shared (no userID required).
// An empty query returns an empty result. Limit is clamped to [1, 50], defaulting to 20.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	if query == "" {
		return []domain.RefEntry{}, nil
	}

	limit = clampLimit(limit)

	return s.refEntries.Search(ctx, query, limit)
}
