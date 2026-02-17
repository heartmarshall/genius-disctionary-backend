package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery returns middleware that recovers from panics, logs the error
// with a stack trace, and responds with 500 Internal Server Error.
func Recovery(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					logger.ErrorContext(r.Context(), "panic recovered",
						slog.Any("error", err),
						slog.String("stack", string(stack)),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
