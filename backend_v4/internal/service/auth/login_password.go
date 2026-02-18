package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// LoginWithPassword authenticates a user with email + password.
// Returns ErrUnauthorized if the email is not found or the password is wrong.
func (s *Service) LoginWithPassword(ctx context.Context, input LoginPasswordInput) (*AuthResult, error) {
	// Normalize input before validation.
	input.Email = strings.TrimSpace(input.Email)

	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Find user by email
	user, err := s.users.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, fmt.Errorf("auth.LoginWithPassword get user: %w", err)
	}

	// Step 3: Find password auth method for this user
	am, err := s.authMethods.GetByUserAndMethod(ctx, user.ID, domain.AuthMethodPassword)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// User exists but has no password method (OAuth-only)
			return nil, domain.ErrUnauthorized
		}
		return nil, fmt.Errorf("auth.LoginWithPassword get auth method: %w", err)
	}

	// Step 4: Verify password
	if am.PasswordHash == nil {
		return nil, domain.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*am.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	// Step 5: Issue tokens
	result, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("auth.LoginWithPassword issue tokens: %w", err)
	}

	s.log.InfoContext(ctx, "user logged in via password",
		slog.String("user_id", user.ID.String()))

	return result, nil
}
