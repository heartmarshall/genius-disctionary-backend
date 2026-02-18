package content

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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

	trimmedURL := strings.TrimSpace(input.URL)
	var trimmedCaption *string
	if input.Caption != nil {
		t := strings.TrimSpace(*input.Caption)
		trimmedCaption = &t
	}

	var image *domain.UserImage

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx
		_, err := s.entries.GetByID(txCtx, userID, input.EntryID)
		if err != nil {
			return err
		}

		// Check limit inside tx
		count, err := s.images.CountUserByEntry(txCtx, input.EntryID)
		if err != nil {
			return fmt.Errorf("count user images: %w", err)
		}
		if count >= MaxUserImagesPerEntry {
			return domain.NewValidationError("images", fmt.Sprintf("limit reached (%d)", MaxUserImagesPerEntry))
		}

		// Create user image (trimmed)
		image, err = s.images.CreateUser(txCtx, input.EntryID, trimmedURL, trimmedCaption)
		if err != nil {
			return fmt.Errorf("create user image: %w", err)
		}

		// Audit on parent ENTRY
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &input.EntryID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"user_image_added": map[string]any{"new": trimmedURL},
			},
		})
	})

	if err != nil {
		return nil, err
	}

	s.log.DebugContext(ctx, "user image added",
		slog.String("user_id", userID.String()),
		slog.String("entry_id", input.EntryID.String()),
		slog.String("image_id", image.ID.String()),
	)

	return image, nil
}

// UpdateUserImage updates a user image's caption.
func (s *Service) UpdateUserImage(ctx context.Context, input UpdateUserImageInput) (*domain.UserImage, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Trim optional text fields
	var trimmedCaption *string
	if input.Caption != nil {
		t := strings.TrimSpace(*input.Caption)
		trimmedCaption = &t
	}

	var image *domain.UserImage

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx (JOIN-based)
		oldImage, err := s.images.GetUserByIDForUser(txCtx, userID, input.ImageID)
		if err != nil {
			return err
		}

		// Update caption (trimmed)
		image, err = s.images.UpdateUser(txCtx, input.ImageID, trimmedCaption)
		if err != nil {
			return fmt.Errorf("update user image: %w", err)
		}

		// Audit on parent ENTRY — skip if nothing changed
		changes := buildImageCaptionChanges(oldImage.Caption, trimmedCaption)
		if len(changes) == 0 {
			return nil
		}

		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &oldImage.EntryID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
		})
	})

	if err != nil {
		return nil, err
	}

	s.log.DebugContext(ctx, "user image updated",
		slog.String("user_id", userID.String()),
		slog.String("image_id", input.ImageID.String()),
	)

	return image, nil
}

// DeleteUserImage deletes a user image.
func (s *Service) DeleteUserImage(ctx context.Context, imageID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	return s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Check ownership inside tx (JOIN-based)
		image, err := s.images.GetUserByIDForUser(txCtx, userID, imageID)
		if err != nil {
			return err
		}

		// Delete image
		if err := s.images.DeleteUser(txCtx, imageID); err != nil {
			return fmt.Errorf("delete user image: %w", err)
		}

		s.log.InfoContext(txCtx, "user image deleted",
			slog.String("user_id", userID.String()),
			slog.String("image_id", imageID.String()),
		)

		// Audit on parent ENTRY
		return s.audit.Log(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &image.EntryID,
			Action:     domain.AuditActionUpdate,
			Changes: map[string]any{
				"user_image_deleted": map[string]any{"old": image.URL},
			},
		})
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildImageCaptionChanges compares old and new caption and returns audit changes.
func buildImageCaptionChanges(oldCaption, newCaption *string) map[string]any {
	oldVal := ""
	if oldCaption != nil {
		oldVal = *oldCaption
	}
	newVal := ""
	if newCaption != nil {
		newVal = *newCaption
	}
	if oldVal == newVal {
		return nil
	}
	return map[string]any{
		"caption": map[string]any{"old": oldVal, "new": newVal},
	}
}
