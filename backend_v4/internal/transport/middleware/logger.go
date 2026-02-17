package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// Logger returns middleware that logs each HTTP request with method, path,
// status code, duration, and context identifiers (request_id, user_id).
func Logger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			duration := time.Since(start)
			requestID := ctxutil.RequestIDFromCtx(r.Context())
			userID, _ := ctxutil.UserIDFromCtx(r.Context())

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Duration("duration", duration),
				slog.String("request_id", requestID),
			}
			if userID.String() != "00000000-0000-0000-0000-000000000000" {
				attrs = append(attrs, slog.String("user_id", userID.String()))
			}

			level := slog.LevelInfo
			if sw.status >= 500 {
				level = slog.LevelError
			}
			logger.LogAttrs(r.Context(), level, "http.request", attrs...)
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the response status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}
