package transport

import (
	"context"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID добавляет request ID в контекст.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID извлекает request ID из контекста.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
