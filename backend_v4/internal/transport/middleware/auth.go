package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

type tokenValidator interface {
	ValidateToken(ctx context.Context, token string) (uuid.UUID, string, error)
}

func Auth(validator tokenValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				next.ServeHTTP(w, r) // Anonymous
				return
			}
			userID, role, err := validator.ValidateToken(r.Context(), token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			ctx := ctxutil.WithUserID(r.Context(), userID)
			ctx = ctxutil.WithUserRole(ctx, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}
