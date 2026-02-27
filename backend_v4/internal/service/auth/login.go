package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Login performs OAuth authentication and returns access/refresh tokens.
// If the user doesn't exist, creates a new user with default settings in a transaction.
// If the user exists, updates their profile if it changed.
// If a user with the same email already exists (password-registered), links the OAuth method.
func (s *Service) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	// Step 1: Validate input
	if err := input.Validate(s.cfg.AllowedProviders()); err != nil {
		return nil, err
	}

	// Step 2: Verify OAuth code with provider
	identity, err := s.oauth.VerifyCode(ctx, input.Provider, input.Code)
	if err != nil {
		return nil, fmt.Errorf("auth.Login oauth verification: %w", err)
	}

	method := domain.AuthMethodType(input.Provider)

	// Step 3: Check if an auth method already exists for this OAuth identity
	am, err := s.authMethods.GetByOAuth(ctx, method, identity.ProviderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("auth.Login get auth method: %w", err)
	}

	if am != nil {
		// Existing OAuth user — load and optionally update profile
		user, err := s.users.GetByID(ctx, am.UserID)
		if err != nil {
			return nil, fmt.Errorf("auth.Login get user: %w", err)
		}

		if profileChanged(user, identity) {
			user, err = s.users.Update(ctx, user.ID, identity.Name, identity.AvatarURL)
			if err != nil {
				return nil, fmt.Errorf("auth.Login update profile: %w", err)
			}
		}

		result, err := s.issueTokens(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("auth.Login issue tokens: %w", err)
		}

		s.log.InfoContext(ctx, "user logged in via oauth",
			slog.String("user_id", user.ID.String()),
			slog.String("provider", input.Provider))

		return result, nil
	}

	// Step 4: No existing auth method — check if user with same email exists (account linking)
	identity.Email = strings.ToLower(strings.TrimSpace(identity.Email))
	user, err := s.users.GetByEmail(ctx, identity.Email)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("auth.Login get user by email: %w", err)
	}

	if user != nil {
		// Account linking: create OAuth auth method for existing user in a transaction.
		err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
			newAM := &domain.AuthMethod{
				UserID:     user.ID,
				Method:     method,
				ProviderID: &identity.ProviderID,
			}
			if _, err := s.authMethods.Create(txCtx, newAM); err != nil {
				return fmt.Errorf("link oauth: %w", err)
			}

			if profileChanged(user, identity) {
				user, err = s.users.Update(txCtx, user.ID, identity.Name, identity.AvatarURL)
				if err != nil {
					return fmt.Errorf("update profile: %w", err)
				}
			}
			return nil
		})
		if err != nil {
			if !errors.Is(err, domain.ErrAlreadyExists) {
				return nil, fmt.Errorf("auth.Login link oauth: %w", err)
			}
			// Concurrent link — the method already exists, just proceed to issue tokens.
		}

		result, err := s.issueTokens(ctx, user)
		if err != nil {
			return nil, fmt.Errorf("auth.Login issue tokens: %w", err)
		}

		s.log.InfoContext(ctx, "oauth linked to existing account",
			slog.String("user_id", user.ID.String()),
			slog.String("provider", input.Provider))

		return result, nil
	}

	// Step 5: Completely new user — register in a transaction
	user, err = s.registerOAuthUser(ctx, identity, method)
	if err != nil {
		return nil, err
	}

	result, err := s.issueTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("auth.Login issue tokens: %w", err)
	}

	s.log.InfoContext(ctx, "user registered via oauth",
		slog.String("user_id", user.ID.String()),
		slog.String("provider", input.Provider))

	return result, nil
}

// emailPrefix extracts the part before @ from an email address.
func emailPrefix(email string) string {
	if idx := strings.IndexByte(email, '@'); idx > 0 {
		return email[:idx]
	}
	return email
}

// registerOAuthUser creates a new user + auth method + default settings in a transaction.
func (s *Service) registerOAuthUser(ctx context.Context, identity *auth.OAuthIdentity, method domain.AuthMethodType) (*domain.User, error) {
	var createdUser *domain.User

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		// Use email prefix as username for OAuth users
		identity.Email = strings.ToLower(strings.TrimSpace(identity.Email))
		username := emailPrefix(identity.Email)

		now := time.Now()
		newUser := &domain.User{
			ID:        uuid.New(),
			Email:     identity.Email,
			Username:  username,
			Name:      derefOrEmpty(identity.Name),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if identity.AvatarURL != nil {
			newUser.AvatarURL = identity.AvatarURL
		}

		user, err := s.users.Create(txCtx, newUser)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		// Create auth method
		am := &domain.AuthMethod{
			UserID:     user.ID,
			Method:     method,
			ProviderID: &identity.ProviderID,
		}
		if _, err := s.authMethods.Create(txCtx, am); err != nil {
			return fmt.Errorf("create auth method: %w", err)
		}

		// Create default settings
		settings := domain.DefaultUserSettings(user.ID)
		if err := s.settings.CreateSettings(txCtx, &settings); err != nil {
			return fmt.Errorf("create settings: %w", err)
		}

		createdUser = user
		return nil
	})

	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			// Race condition: retry lookup
			am, retryErr := s.authMethods.GetByOAuth(ctx, method, identity.ProviderID)
			if retryErr == nil {
				user, retryErr := s.users.GetByID(ctx, am.UserID)
				if retryErr == nil {
					return user, nil
				}
			}
			return nil, domain.ErrAlreadyExists
		}
		return nil, fmt.Errorf("auth.Login register user: %w", err)
	}

	return createdUser, nil
}
