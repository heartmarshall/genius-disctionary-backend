package dictionary

import (
	"context"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 1. SearchCatalog
// ---------------------------------------------------------------------------

// SearchCatalog searches the reference catalog for entries matching the query.
func (s *Service) SearchCatalog(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	if _, ok := ctxutil.UserIDFromCtx(ctx); !ok {
		return nil, domain.ErrUnauthorized
	}

	if query == "" {
		return []domain.RefEntry{}, nil
	}

	limit = clampLimit(limit, 1, 50, 20)

	return s.refCatalog.Search(ctx, query, limit)
}

// ---------------------------------------------------------------------------
// 2. PreviewRefEntry
// ---------------------------------------------------------------------------

// PreviewRefEntry fetches or retrieves a reference entry by text.
func (s *Service) PreviewRefEntry(ctx context.Context, text string) (*domain.RefEntry, error) {
	if _, ok := ctxutil.UserIDFromCtx(ctx); !ok {
		return nil, domain.ErrUnauthorized
	}

	return s.refCatalog.GetOrFetchEntry(ctx, text)
}
