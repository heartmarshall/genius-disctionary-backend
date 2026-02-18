package auth

import (
	"slices"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// LoginInput holds parameters for OAuth login operation.
type LoginInput struct {
	Provider string
	Code     string
}

// Validate validates the login input.
func (i LoginInput) Validate(allowedProviders []string) error {
	var errs []domain.FieldError

	if i.Provider == "" {
		errs = append(errs, domain.FieldError{Field: "provider", Message: "required"})
	} else if !slices.Contains(allowedProviders, i.Provider) {
		errs = append(errs, domain.FieldError{Field: "provider", Message: "unsupported provider"})
	}

	if i.Code == "" {
		errs = append(errs, domain.FieldError{Field: "code", Message: "required"})
	} else if len(i.Code) > 4096 {
		errs = append(errs, domain.FieldError{Field: "code", Message: "too long"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// RefreshInput holds parameters for token refresh operation.
type RefreshInput struct {
	RefreshToken string
}

// Validate validates the refresh input.
func (i RefreshInput) Validate() error {
	var errs []domain.FieldError

	if i.RefreshToken == "" {
		errs = append(errs, domain.FieldError{Field: "refresh_token", Message: "required"})
	} else if len(i.RefreshToken) > 512 {
		errs = append(errs, domain.FieldError{Field: "refresh_token", Message: "too long"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// RegisterInput holds parameters for password registration.
type RegisterInput struct {
	Email    string
	Username string
	Password string
}

// Validate validates the register input.
func (i RegisterInput) Validate() error {
	var errs []domain.FieldError

	email := strings.TrimSpace(i.Email)
	if email == "" {
		errs = append(errs, domain.FieldError{Field: "email", Message: "required"})
	} else if !strings.Contains(email, "@") || len(email) > 254 {
		errs = append(errs, domain.FieldError{Field: "email", Message: "invalid email"})
	}

	username := strings.TrimSpace(i.Username)
	if username == "" {
		errs = append(errs, domain.FieldError{Field: "username", Message: "required"})
	} else if len(username) < 2 || len(username) > 50 {
		errs = append(errs, domain.FieldError{Field: "username", Message: "must be between 2 and 50 characters"})
	}

	if i.Password == "" {
		errs = append(errs, domain.FieldError{Field: "password", Message: "required"})
	} else if len(i.Password) < 8 {
		errs = append(errs, domain.FieldError{Field: "password", Message: "must be at least 8 characters"})
	} else if len(i.Password) > 72 {
		errs = append(errs, domain.FieldError{Field: "password", Message: "must be at most 72 characters"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// LoginPasswordInput holds parameters for email + password login.
type LoginPasswordInput struct {
	Email    string
	Password string
}

// Validate validates the login-with-password input.
func (i LoginPasswordInput) Validate() error {
	var errs []domain.FieldError

	if strings.TrimSpace(i.Email) == "" {
		errs = append(errs, domain.FieldError{Field: "email", Message: "required"})
	}

	if i.Password == "" {
		errs = append(errs, domain.FieldError{Field: "password", Message: "required"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}
