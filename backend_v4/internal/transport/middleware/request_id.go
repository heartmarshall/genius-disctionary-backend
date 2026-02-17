package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// RequestIDHeader is the HTTP header used to propagate request IDs.
const RequestIDHeader = "X-Request-Id"

// RequestID returns middleware that extracts or generates a unique request ID.
// If the incoming request has an X-Request-Id header, it is reused;
// otherwise a new UUID is generated. The ID is stored in the context
// and set on the response header.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if id == "" {
				id = uuid.New().String()
			}
			ctx := ctxutil.WithRequestID(r.Context(), id)
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
