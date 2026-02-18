package user

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetSettings returns the authenticated user's settings.
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) GetSettings(ctx context.Context) (*domain.UserSettings, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	settings, err := s.settings.GetSettings(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user.GetSettings: %w", err)
	}

	return settings, nil
}

// UpdateSettings updates the authenticated user's settings with partial updates.
// Returns ErrUnauthorized if no userID is found in context.
// Creates an audit record for the changes in a transaction.
func (s *Service) UpdateSettings(ctx context.Context, input UpdateSettingsInput) (*domain.UserSettings, error) {
	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Extract userID from context
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	var updatedSettings *domain.UserSettings

	// Step 3: Update settings and create audit record in transaction
	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Get current settings
		current, err := s.settings.GetSettings(txCtx, userID)
		if err != nil {
			return fmt.Errorf("get current settings: %w", err)
		}

		// Apply changes
		newSettings := applySettingsChanges(*current, input)

		// Update settings
		updated, err := s.settings.UpdateSettings(txCtx, userID, newSettings)
		if err != nil {
			return fmt.Errorf("update settings: %w", err)
		}
		updatedSettings = updated

		// Build changes for audit
		changes := buildSettingsChanges(*current, newSettings)

		// Create audit record
		auditRecord := domain.AuditRecord{
			ID:         uuid.New(),
			UserID:     userID,
			EntityType: domain.EntityTypeUser,
			EntityID:   &userID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
			CreatedAt:  time.Now().UTC(),
		}
		if _, err := s.audit.Create(txCtx, auditRecord); err != nil {
			return fmt.Errorf("create audit record: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("user.UpdateSettings: %w", err)
	}

	// Step 4: Log the update
	s.log.InfoContext(ctx, "settings updated",
		slog.String("user_id", userID.String()))

	return updatedSettings, nil
}

// applySettingsChanges merges the input changes into current settings.
func applySettingsChanges(current domain.UserSettings, input UpdateSettingsInput) domain.UserSettings {
	result := current

	if input.NewCardsPerDay != nil {
		result.NewCardsPerDay = *input.NewCardsPerDay
	}
	if input.ReviewsPerDay != nil {
		result.ReviewsPerDay = *input.ReviewsPerDay
	}
	if input.MaxIntervalDays != nil {
		result.MaxIntervalDays = *input.MaxIntervalDays
	}
	if input.Timezone != nil {
		result.Timezone = *input.Timezone
	}

	return result
}

// buildSettingsChanges creates a map of field changes for audit logging.
func buildSettingsChanges(old, new domain.UserSettings) map[string]any {
	changes := make(map[string]any)

	if old.NewCardsPerDay != new.NewCardsPerDay {
		changes["new_cards_per_day"] = map[string]any{
			"old": old.NewCardsPerDay,
			"new": new.NewCardsPerDay,
		}
	}
	if old.ReviewsPerDay != new.ReviewsPerDay {
		changes["reviews_per_day"] = map[string]any{
			"old": old.ReviewsPerDay,
			"new": new.ReviewsPerDay,
		}
	}
	if old.MaxIntervalDays != new.MaxIntervalDays {
		changes["max_interval_days"] = map[string]any{
			"old": old.MaxIntervalDays,
			"new": new.MaxIntervalDays,
		}
	}
	if old.Timezone != new.Timezone {
		changes["timezone"] = map[string]any{
			"old": old.Timezone,
			"new": new.Timezone,
		}
	}

	return changes
}
