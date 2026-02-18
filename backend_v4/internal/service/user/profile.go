package user

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetProfile returns the authenticated user's profile.
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) GetProfile(ctx context.Context) (*domain.User, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.GetProfile: %w", err)
	}

	return user, nil
}

// UpdateProfile updates the authenticated user's profile (name and avatar).
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) UpdateProfile(ctx context.Context, input UpdateProfileInput) (*domain.User, error) {
	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Extract userID from context
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	// Step 3: Update profile
	user, err := s.users.Update(ctx, userID, &input.Name, input.AvatarURL)
	if err != nil {
		return nil, fmt.Errorf("user.UpdateProfile: %w", err)
	}

	// Step 4: Log the update
	s.log.InfoContext(ctx, "profile updated",
		slog.String("user_id", userID.String()))

	return user, nil
}
