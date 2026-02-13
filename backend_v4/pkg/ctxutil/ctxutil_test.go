package ctxutil

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestWithUserID_And_UserIDFromCtx(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	ctx := WithUserID(context.Background(), id)

	got, ok := UserIDFromCtx(ctx)
	if !ok {
		t.Fatal("expected ok=true for valid UUID")
	}
	if got != id {
		t.Fatalf("expected %s, got %s", id, got)
	}
}

func TestUserIDFromCtx_EmptyContext(t *testing.T) {
	t.Parallel()

	got, ok := UserIDFromCtx(context.Background())
	if ok {
		t.Fatal("expected ok=false for empty context")
	}
	if got != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %s", got)
	}
}

func TestUserIDFromCtx_NilUUID(t *testing.T) {
	t.Parallel()

	ctx := WithUserID(context.Background(), uuid.Nil)

	got, ok := UserIDFromCtx(ctx)
	if ok {
		t.Fatal("expected ok=false for uuid.Nil")
	}
	if got != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %s", got)
	}
}

func TestUserIDFromCtx_WrongType(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), ctxKey("user_id"), "not-a-uuid")

	got, ok := UserIDFromCtx(ctx)
	if ok {
		t.Fatal("expected ok=false for wrong type")
	}
	if got != uuid.Nil {
		t.Fatalf("expected uuid.Nil, got %s", got)
	}
}

func TestWithRequestID_And_RequestIDFromCtx(t *testing.T) {
	t.Parallel()

	ctx := WithRequestID(context.Background(), "req-123")

	got := RequestIDFromCtx(ctx)
	if got != "req-123" {
		t.Fatalf("expected req-123, got %s", got)
	}
}

func TestRequestIDFromCtx_EmptyContext(t *testing.T) {
	t.Parallel()

	got := RequestIDFromCtx(context.Background())
	if got != "" {
		t.Fatalf("expected empty string, got %s", got)
	}
}

func TestRequestIDFromCtx_WrongType(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), ctxKey("request_id"), 12345)

	got := RequestIDFromCtx(ctx)
	if got != "" {
		t.Fatalf("expected empty string, got %s", got)
	}
}
