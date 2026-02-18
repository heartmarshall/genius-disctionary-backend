package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Refresh performs token rotation and returns new access/refresh tokens.
// If the refresh token is not found (revoked or reused), logs a warning and returns ErrUnauthorized.
// If the token is expired or the user is deleted, returns ErrUnauthorized.
func (s *Service) Refresh(ctx context.Context, input RefreshInput) (*AuthResult, error) {
	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Hash the refresh token
	hash := auth.HashToken(input.RefreshToken)

	// Step 3: Get token from DB
	token, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Token not found (reuse detection)
			s.log.WarnContext(ctx, "refresh token reuse attempted")
			return nil, domain.ErrUnauthorized
		}
		return nil, fmt.Errorf("auth.Refresh get token: %w", err)
	}

	// Step 4: Check if token is expired
	if token.IsExpired(time.Now()) {
		return nil, domain.ErrUnauthorized
	}

	// Step 5: Get user
	user, err := s.users.GetByID(ctx, token.UserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// User deleted
			s.log.WarnContext(ctx, "refresh for deleted user",
				slog.String("user_id", token.UserID.String()))
			return nil, domain.ErrUnauthorized
		}
		return nil, fmt.Errorf("auth.Refresh get user: %w", err)
	}

	// Step 6: Revoke old token
	if err := s.tokens.RevokeByID(ctx, token.ID); err != nil {
		return nil, fmt.Errorf("auth.Refresh revoke token: %w", err)
	}

	// Step 7: Issue new token pair
	result, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("auth.Refresh issue tokens: %w", err)
	}
	return result, nil
}
