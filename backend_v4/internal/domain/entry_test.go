package domain

import (
	"testing"
	"time"
)

func TestEntry_IsDeleted(t *testing.T) {
	t.Parallel()

	t.Run("nil DeletedAt", func(t *testing.T) {
		t.Parallel()
		e := &Entry{DeletedAt: nil}
		if e.IsDeleted() {
			t.Error("expected not deleted")
		}
	})

	t.Run("non-nil DeletedAt", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		e := &Entry{DeletedAt: &now}
		if !e.IsDeleted() {
			t.Error("expected deleted")
		}
	})
}
