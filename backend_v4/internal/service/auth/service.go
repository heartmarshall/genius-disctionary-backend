package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// userRepo defines the user repository interface needed by auth service.
type userRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	Update(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error)
}

// settingsRepo defines the settings repository interface needed by auth service.
type settingsRepo interface {
	CreateSettings(ctx context.Context, settings *domain.UserSettings) error
}

// tokenRepo defines the refresh token repository interface needed by auth service.
type tokenRepo interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeByID(ctx context.Context, id uuid.UUID) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) (int, error)
}

// authMethodRepo defines the auth method repository interface needed by auth service.
type authMethodRepo interface {
	GetByOAuth(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error)
	GetByUserAndMethod(ctx context.Context, userID uuid.UUID, method domain.AuthMethodType) (*domain.AuthMethod, error)
	Create(ctx context.Context, am *domain.AuthMethod) (*domain.AuthMethod, error)
}

// txManager defines the transaction manager interface needed by auth service.
type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// oauthVerifier defines the OAuth verification interface needed by auth service.
type oauthVerifier interface {
	VerifyCode(ctx context.Context, provider, code string) (*auth.OAuthIdentity, error)
}

// jwtManager defines the JWT token management interface needed by auth service.
type jwtManager interface {
	GenerateAccessToken(userID uuid.UUID, role string) (string, error)
	ValidateAccessToken(token string) (uuid.UUID, string, error)
	GenerateRefreshToken() (raw string, hash string, err error)
}

// Service implements auth operations.
type Service struct {
	log         *slog.Logger
	users       userRepo
	settings    settingsRepo
	tokens      tokenRepo
	authMethods authMethodRepo
	tx          txManager
	oauth       oauthVerifier
	jwt         jwtManager
	cfg         config.AuthConfig
}

// NewService creates a new auth service instance.
func NewService(
	logger *slog.Logger,
	users userRepo,
	settings settingsRepo,
	tokens tokenRepo,
	authMethods authMethodRepo,
	tx txManager,
	oauth oauthVerifier,
	jwt jwtManager,
	cfg config.AuthConfig,
) *Service {
	return &Service{
		log:         logger.With("service", "auth"),
		users:       users,
		settings:    settings,
		tokens:      tokens,
		authMethods: authMethods,
		tx:          tx,
		oauth:       oauth,
		jwt:         jwt,
		cfg:         cfg,
	}
}

// issueTokens generates access and refresh tokens for the given user, stores
// the refresh token hash in DB, and returns an AuthResult.
func (s *Service) issueTokens(ctx context.Context, user *domain.User) (*AuthResult, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Role.String())
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefresh, hashRefresh, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashRefresh,
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.tokens.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		User:         user,
	}, nil
}

// derefOrEmpty returns the dereferenced value or empty string if nil.
func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ptrStringNotEqual compares *string with *string, treating nil as distinct from "".
func ptrStringNotEqual(a, b *string) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

// profileChanged checks if the OAuth identity profile differs from the stored user profile.
func profileChanged(user *domain.User, identity *auth.OAuthIdentity) bool {
	if identity.Name != nil && *identity.Name != user.Name {
		return true
	}
	if identity.AvatarURL != nil && ptrStringNotEqual(identity.AvatarURL, user.AvatarURL) {
		return true
	}
	return false
}
