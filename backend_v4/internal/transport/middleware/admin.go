package middleware

import (
	"context"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// RequireAdmin returns domain.ErrForbidden if the context user is not admin.
// Use in resolver methods or REST handlers, not as HTTP middleware.
func RequireAdmin(ctx context.Context) error {
	if !ctxutil.IsAdminCtx(ctx) {
		return domain.ErrForbidden
	}
	return nil
}
