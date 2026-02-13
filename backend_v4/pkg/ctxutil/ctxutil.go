package ctxutil

import (
	"context"

	"github.com/google/uuid"
)

type ctxKey string

const (
	userIDKey    ctxKey = "user_id"
	requestIDKey ctxKey = "request_id"
)

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserIDFromCtx extracts the user ID from the context.
// Returns uuid.Nil and false if the value is missing, nil UUID, or wrong type.
func UserIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok || id == uuid.Nil {
		return uuid.Nil, false
	}
	return id, true
}

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromCtx extracts the request ID from the context.
// Returns an empty string if absent.
func RequestIDFromCtx(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}
