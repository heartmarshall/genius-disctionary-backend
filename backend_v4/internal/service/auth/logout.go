package auth

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// Logout revokes all refresh tokens for the authenticated user.
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) Logout(ctx context.Context) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	if err := s.tokens.RevokeAllByUser(ctx, userID); err != nil {
		return fmt.Errorf("auth.Logout: %w", err)
	}

	s.log.InfoContext(ctx, "user logged out", slog.String("user_id", userID.String()))
	return nil
}

// ValidateToken validates an access token and returns the user ID and role.
// Returns ErrUnauthorized if the token is invalid or expired.
func (s *Service) ValidateToken(ctx context.Context, token string) (uuid.UUID, string, error) {
	userID, role, err := s.jwt.ValidateAccessToken(token)
	if err != nil {
		return uuid.Nil, "", domain.ErrUnauthorized
	}
	return userID, role, nil
}

// CleanupExpiredTokens removes all expired refresh tokens from the database.
// Returns the number of tokens deleted. This is a maintenance operation.
func (s *Service) CleanupExpiredTokens(ctx context.Context) (int, error) {
	count, err := s.tokens.DeleteExpired(ctx)
	if err != nil {
		s.log.ErrorContext(ctx, "token cleanup failed", slog.String("error", err.Error()))
		return 0, fmt.Errorf("auth.CleanupExpiredTokens: %w", err)
	}

	if count > 0 {
		s.log.InfoContext(ctx, "cleaned up expired tokens", slog.Int("count", count))
	}

	return count, nil
}
