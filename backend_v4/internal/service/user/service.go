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

// userRepo defines the user repository interface needed by user service.
type userRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error)
}

// settingsRepo defines the settings repository interface needed by user service.
type settingsRepo interface {
	GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
	UpdateSettings(ctx context.Context, userID uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error)
}

// auditRepo defines the audit repository interface needed by user service.
type auditRepo interface {
	Create(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error)
}

// txManager defines the transaction manager interface needed by user service.
type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// Service implements user profile and settings operations.
type Service struct {
	log      *slog.Logger
	users    userRepo
	settings settingsRepo
	audit    auditRepo
	tx       txManager
}

// NewService creates a new user service instance.
func NewService(
	logger *slog.Logger,
	users userRepo,
	settings settingsRepo,
	audit auditRepo,
	tx txManager,
) *Service {
	return &Service{
		log:      logger.With("service", "user"),
		users:    users,
		settings: settings,
		audit:    audit,
		tx:       tx,
	}
}

// GetProfile returns the authenticated user's profile.
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) GetProfile(ctx context.Context) (domain.User, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.User{}, domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("user.GetProfile: %w", err)
	}

	return *user, nil
}

// UpdateProfile updates the authenticated user's profile (name and avatar).
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) UpdateProfile(ctx context.Context, input UpdateProfileInput) (domain.User, error) {
	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return domain.User{}, err
	}

	// Step 2: Extract userID from context
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.User{}, domain.ErrUnauthorized
	}

	// Step 3: Update profile
	user, err := s.users.Update(ctx, userID, &input.Name, input.AvatarURL)
	if err != nil {
		return domain.User{}, fmt.Errorf("user.UpdateProfile: %w", err)
	}

	// Step 4: Log the update
	s.log.InfoContext(ctx, "profile updated",
		slog.String("user_id", userID.String()))

	return *user, nil
}

// GetSettings returns the authenticated user's settings.
// Returns ErrUnauthorized if no userID is found in context.
func (s *Service) GetSettings(ctx context.Context) (domain.UserSettings, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.UserSettings{}, domain.ErrUnauthorized
	}

	settings, err := s.settings.GetSettings(ctx, userID)
	if err != nil {
		return domain.UserSettings{}, fmt.Errorf("user.GetSettings: %w", err)
	}

	return *settings, nil
}

// UpdateSettings updates the authenticated user's settings with partial updates.
// Returns ErrUnauthorized if no userID is found in context.
// Creates an audit record for the changes in a transaction.
func (s *Service) UpdateSettings(ctx context.Context, input UpdateSettingsInput) (domain.UserSettings, error) {
	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return domain.UserSettings{}, err
	}

	// Step 2: Extract userID from context
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.UserSettings{}, domain.ErrUnauthorized
	}

	var updatedSettings domain.UserSettings

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
		updatedSettings = *updated

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
		return domain.UserSettings{}, fmt.Errorf("user.UpdateSettings: %w", err)
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
