package auth

import (
	"slices"

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
