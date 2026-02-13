package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDefaultUserSettings(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	s := DefaultUserSettings(id)

	if s.UserID != id {
		t.Fatalf("expected UserID %s, got %s", id, s.UserID)
	}
	if s.NewCardsPerDay != 20 {
		t.Errorf("NewCardsPerDay = %d, want 20", s.NewCardsPerDay)
	}
	if s.ReviewsPerDay != 200 {
		t.Errorf("ReviewsPerDay = %d, want 200", s.ReviewsPerDay)
	}
	if s.MaxIntervalDays != 365 {
		t.Errorf("MaxIntervalDays = %d, want 365", s.MaxIntervalDays)
	}
	if s.Timezone != "UTC" {
		t.Errorf("Timezone = %q, want UTC", s.Timezone)
	}
}

func TestRefreshToken_IsRevoked(t *testing.T) {
	t.Parallel()

	t.Run("not revoked", func(t *testing.T) {
		t.Parallel()
		token := &RefreshToken{RevokedAt: nil}
		if token.IsRevoked() {
			t.Error("expected not revoked")
		}
	})

	t.Run("revoked", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		token := &RefreshToken{RevokedAt: &now}
		if !token.IsRevoked() {
			t.Error("expected revoked")
		}
	})
}

func TestRefreshToken_IsExpired(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 2, 13, 12, 0, 0, 0, time.UTC)

	t.Run("not expired", func(t *testing.T) {
		t.Parallel()
		future := now.Add(time.Hour)
		token := &RefreshToken{ExpiresAt: future}
		if token.IsExpired(now) {
			t.Error("expected not expired")
		}
	})

	t.Run("expired", func(t *testing.T) {
		t.Parallel()
		past := now.Add(-time.Hour)
		token := &RefreshToken{ExpiresAt: past}
		if !token.IsExpired(now) {
			t.Error("expected expired")
		}
	})

	t.Run("exactly now", func(t *testing.T) {
		t.Parallel()
		token := &RefreshToken{ExpiresAt: now}
		// ExpiresAt == now means not yet expired (Before returns false).
		if token.IsExpired(now) {
			t.Error("expected not expired when ExpiresAt == now")
		}
	})
}
