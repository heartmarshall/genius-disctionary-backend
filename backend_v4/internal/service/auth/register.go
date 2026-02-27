package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Register creates a new user with email + password authentication.
// Returns ErrAlreadyExists if the email or username is already taken.
func (s *Service) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	// Normalize input before validation.
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Username = strings.TrimSpace(input.Username)

	// Step 1: Validate input
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Step 2: Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), s.cfg.PasswordHashCost)
	if err != nil {
		return nil, fmt.Errorf("auth.Register hash password: %w", err)
	}
	hashStr := string(hash)

	// Step 3: Create user + auth method + settings in a transaction.
	// Email and username uniqueness are enforced by DB constraints.
	var createdUser *domain.User

	err = s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		now := time.Now()
		newUser := &domain.User{
			ID:        uuid.New(),
			Email:     input.Email,
			Username:  input.Username,
			Name:      input.Username,
			CreatedAt: now,
			UpdatedAt: now,
		}

		user, err := s.users.Create(txCtx, newUser)
		if err != nil {
			return fmt.Errorf("create user: %w", err)
		}

		am := &domain.AuthMethod{
			UserID:       user.ID,
			Method:       domain.AuthMethodPassword,
			PasswordHash: &hashStr,
		}
		if _, err := s.authMethods.Create(txCtx, am); err != nil {
			return fmt.Errorf("create auth method: %w", err)
		}

		settings := domain.DefaultUserSettings(user.ID)
		if err := s.settings.CreateSettings(txCtx, &settings); err != nil {
			return fmt.Errorf("create settings: %w", err)
		}

		createdUser = user
		return nil
	})

	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil, fmt.Errorf("auth.Register: %w", domain.ErrAlreadyExists)
		}
		return nil, fmt.Errorf("auth.Register: %w", err)
	}

	// Step 4: Issue tokens
	result, err := s.issueTokens(ctx, createdUser)
	if err != nil {
		return nil, fmt.Errorf("auth.Register issue tokens: %w", err)
	}

	s.log.InfoContext(ctx, "user registered via password",
		slog.String("user_id", createdUser.ID.String()))

	return result, nil
}
