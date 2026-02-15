package auth

import (
	"context"
	"errors"
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
	GetByOAuth(ctx context.Context, provider domain.OAuthProvider, oauthID string) (*domain.User, error)
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
	GenerateAccessToken(userID uuid.UUID) (string, error)
	ValidateAccessToken(token string) (uuid.UUID, error)
	GenerateRefreshToken() (raw string, hash string, err error)
}

// Service implements auth operations.
type Service struct {
	log      *slog.Logger
	users    userRepo
	settings settingsRepo
	tokens   tokenRepo
	tx       txManager
	oauth    oauthVerifier
	jwt      jwtManager
	cfg      config.AuthConfig
}

// NewService creates a new auth service instance.
func NewService(
	logger *slog.Logger,
	users userRepo,
	settings settingsRepo,
	tokens tokenRepo,
	tx txManager,
	oauth oauthVerifier,
	jwt jwtManager,
	cfg config.AuthConfig,
) *Service {
	return &Service{
		log:      logger.With("service", "auth"),
		users:    users,
		settings: settings,
		tokens:   tokens,
		tx:       tx,
		oauth:    oauth,
		jwt:      jwt,
		cfg:      cfg,
	}
}

// derefOrEmpty returns the dereferenced value or empty string if nil.
// Used because domain.User.Name is string, but OAuthIdentity.Name is *string.
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

// Login performs OAuth authentication and returns access/refresh tokens.
// If the user doesn't exist, creates a new user with default settings in a transaction.
// If the user exists, updates their profile if it changed.
func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	// Step 1: Validate input
	if err := input.Validate(s.cfg.AllowedProviders()); err != nil {
		return nil, err
	}

	// Step 2: Verify OAuth code with provider
	identity, err := s.oauth.VerifyCode(ctx, input.Provider, input.Code)
	if err != nil {
		s.log.ErrorContext(ctx, "oauth verification failed",
			slog.String("provider", input.Provider),
			slog.String("error", err.Error()))
		return nil, fmt.Errorf("auth.Login oauth verification: %w", err)
	}

	provider := domain.OAuthProvider(input.Provider)

	// Step 3: Check if user exists
	user, err := s.users.GetByOAuth(ctx, provider, identity.ProviderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("auth.Login get user: %w", err)
	}

	isNewUser := errors.Is(err, domain.ErrNotFound)

	// Step 4: Handle new user registration
	if isNewUser {
		user, err = s.registerNewUser(ctx, identity, provider)
		if err != nil {
			return nil, err
		}
	} else {
		// Step 5: Update existing user profile if changed
		if profileChanged(user, identity) {
			user, err = s.users.Update(ctx, user.ID, identity.Name, identity.AvatarURL)
			if err != nil {
				return nil, fmt.Errorf("auth.Login update profile: %w", err)
			}
		}
	}

	// Step 6: Generate access token
	accessToken, err := s.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Login generate access token: %w", err)
	}

	// Step 7: Generate refresh token
	rawRefresh, hashRefresh, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("auth.Login generate refresh token: %w", err)
	}

	// Step 8: Store refresh token hash in DB
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashRefresh,
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.tokens.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("auth.Login store refresh token: %w", err)
	}

	// Step 9: Log the appropriate event
	if isNewUser {
		s.log.InfoContext(ctx, "user registered",
			slog.String("user_id", user.ID.String()),
			slog.String("provider", input.Provider))
	} else {
		s.log.InfoContext(ctx, "user logged in",
			slog.String("user_id", user.ID.String()),
			slog.String("provider", input.Provider))
	}

	// Step 10: Return result
	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		User:         user,
	}, nil
}

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

	// Step 7: Generate new access token
	accessToken, err := s.jwt.GenerateAccessToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("auth.Refresh generate access token: %w", err)
	}

	// Step 8: Generate new refresh token
	rawRefresh, hashRefresh, err := s.jwt.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("auth.Refresh generate refresh token: %w", err)
	}

	// Step 9: Store new refresh token hash in DB
	refreshToken := &domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashRefresh,
		ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.tokens.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("auth.Refresh store refresh token: %w", err)
	}

	// Step 10: Return result
	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		User:         user,
	}, nil
}

// registerNewUser creates a new user and default settings in a transaction.
// Handles race condition: if Create returns ErrAlreadyExists, retries GetByOAuth.
// If retry also returns ErrNotFound, it's an email collision → returns ErrAlreadyExists.
func (s *Service) registerNewUser(ctx context.Context, identity *auth.OAuthIdentity, provider domain.OAuthProvider) (*domain.User, error) {
	var createdUser *domain.User

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Create user
		newUser := &domain.User{
			Email:         identity.Email,
			Name:          derefOrEmpty(identity.Name),
			AvatarURL:     identity.AvatarURL,
			OAuthProvider: provider,
			OAuthID:       identity.ProviderID,
		}

		user, err := s.users.Create(txCtx, newUser)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		// Create default settings
		settings := domain.DefaultUserSettings(user.ID)
		if err := s.settings.CreateSettings(txCtx, &settings); err != nil {
			return fmt.Errorf("create settings: %w", err)
		}

		createdUser = user
		return nil
	})

	if err == nil {
		return createdUser, nil
	}

	// Handle race condition: another request created the user concurrently
	if errors.Is(err, domain.ErrAlreadyExists) {
		// Retry: try to fetch the user again
		user, retryErr := s.users.GetByOAuth(ctx, provider, identity.ProviderID)
		if retryErr == nil {
			// Success: user was created by concurrent request
			return user, nil
		}

		// If still not found, it's an email collision (different provider)
		if errors.Is(retryErr, domain.ErrNotFound) {
			return nil, domain.ErrAlreadyExists
		}

		// Other error during retry
		return nil, fmt.Errorf("auth.Login retry get user: %w", retryErr)
	}

	// Other transaction error
	return nil, fmt.Errorf("auth.Login register user: %w", err)
}
