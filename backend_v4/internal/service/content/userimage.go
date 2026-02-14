package content

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// UserImage Operations
// ---------------------------------------------------------------------------

// AddUserImage adds a new user image to an entry.
func (s *Service) AddUserImage(ctx context.Context, input AddUserImageInput) (*domain.UserImage, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check entry ownership
	_, err := s.checkEntryOwnership(ctx, userID, input.EntryID)
	if err != nil {
		return nil, err
	}

	// Create user image
	image, err := s.images.CreateUser(ctx, input.EntryID, input.URL, input.Caption)
	if err != nil {
		return nil, fmt.Errorf("create user image: %w", err)
	}

	return image, nil
}

// DeleteUserImage deletes a user image.
func (s *Service) DeleteUserImage(ctx context.Context, imageID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Check ownership
	_, _, err := s.checkUserImageOwnership(ctx, userID, imageID)
	if err != nil {
		return err
	}

	// Delete image
	if err := s.images.DeleteUser(ctx, imageID); err != nil {
		return fmt.Errorf("delete user image: %w", err)
	}

	return nil
}
