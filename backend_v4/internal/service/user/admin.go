package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// SetUserRole changes the role of a user (admin only).
func (s *Service) SetUserRole(ctx context.Context, targetUserID uuid.UUID, role domain.UserRole) (*domain.User, error) {
	if !ctxutil.IsAdminCtx(ctx) {
		return nil, domain.ErrForbidden
	}

	if !role.IsValid() {
		return nil, domain.NewValidationError("role", "invalid role: must be 'user' or 'admin'")
	}

	// Prevent admin from demoting themselves.
	callerID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}
	if callerID == targetUserID && role == domain.UserRoleUser {
		return nil, domain.NewValidationError("role", "cannot demote yourself")
	}

	user, err := s.users.UpdateRole(ctx, targetUserID, role.String())
	if err != nil {
		return nil, fmt.Errorf("user.SetUserRole: %w", err)
	}

	s.log.InfoContext(ctx, "user role updated",
		slog.String("target_user_id", targetUserID.String()),
		slog.String("new_role", role.String()),
	)

	return user, nil
}

// ListUsers returns a paginated list of all users (admin only).
func (s *Service) ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error) {
	if !ctxutil.IsAdminCtx(ctx) {
		return nil, 0, domain.ErrForbidden
	}

	if limit <= 0 {
		limit = 50
	}

	users, err := s.users.ListUsers(ctx, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("user.ListUsers: %w", err)
	}

	total, err := s.users.CountUsers(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("user.CountUsers: %w", err)
	}

	return users, total, nil
}
