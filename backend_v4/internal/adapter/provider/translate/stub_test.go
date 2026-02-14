package translate

import (
	"context"
	"testing"
)

func TestStub_FetchTranslations_ReturnsNil(t *testing.T) {
	t.Parallel()

	stub := NewStub()

	got, err := stub.FetchTranslations(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil translations, got %v", got)
	}
}
