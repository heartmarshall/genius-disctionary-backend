package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuthMethodType represents the type of authentication credential.
type AuthMethodType string

const (
	AuthMethodPassword AuthMethodType = "password"
	AuthMethodGoogle   AuthMethodType = "google"
	AuthMethodApple    AuthMethodType = "apple"
)

func (m AuthMethodType) String() string { return string(m) }

// IsValid returns true if the method type is a known value.
func (m AuthMethodType) IsValid() bool {
	switch m {
	case AuthMethodPassword, AuthMethodGoogle, AuthMethodApple:
		return true
	}
	return false
}

// IsOAuth returns true if this method type is an OAuth provider.
func (m AuthMethodType) IsOAuth() bool {
	return m == AuthMethodGoogle || m == AuthMethodApple
}

// AuthMethod represents a single authentication credential for a user.
// A user may have multiple AuthMethods (e.g., Google + password).
type AuthMethod struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Method       AuthMethodType
	ProviderID   *string
	PasswordHash *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
