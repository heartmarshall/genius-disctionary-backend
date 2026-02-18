package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents an authenticated application user.
type User struct {
	ID        uuid.UUID
	Email     string
	Username  string
	Name      string
	AvatarURL *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserSettings holds per-user SRS and display preferences.
type UserSettings struct {
	UserID          uuid.UUID
	NewCardsPerDay  int
	ReviewsPerDay   int
	MaxIntervalDays int
	Timezone        string
	UpdatedAt       time.Time
}

// DefaultUserSettings returns UserSettings with sensible defaults.
func DefaultUserSettings(userID uuid.UUID) UserSettings {
	return UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		ReviewsPerDay:   200,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}
}

// RefreshToken represents a hashed refresh token stored in the database.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	RevokedAt *time.Time
}

// IsRevoked returns true if the token has been revoked.
func (t *RefreshToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

// IsExpired returns true if the token has expired relative to now.
func (t *RefreshToken) IsExpired(now time.Time) bool {
	return t.ExpiresAt.Before(now)
}
