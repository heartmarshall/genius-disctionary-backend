package user

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
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
