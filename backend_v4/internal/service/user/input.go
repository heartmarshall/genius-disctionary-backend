package user

import (
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// UpdateProfileInput holds parameters for profile update operation.
type UpdateProfileInput struct {
	Name      string
	AvatarURL *string
}

// Validate validates the update profile input.
func (i UpdateProfileInput) Validate() error {
	var errs []domain.FieldError

	if i.Name == "" {
		errs = append(errs, domain.FieldError{Field: "name", Message: "required"})
	} else if len(i.Name) > 255 {
		errs = append(errs, domain.FieldError{Field: "name", Message: "too long"})
	}

	if i.AvatarURL != nil && len(*i.AvatarURL) > 512 {
		errs = append(errs, domain.FieldError{Field: "avatar_url", Message: "too long"})
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}

// UpdateSettingsInput holds parameters for settings update operation.
// All fields are optional (nil = don't change).
type UpdateSettingsInput struct {
	NewCardsPerDay   *int
	ReviewsPerDay    *int
	MaxIntervalDays  *int
	Timezone         *string
	DesiredRetention *float64
}

// Validate validates the update settings input.
func (i UpdateSettingsInput) Validate() error {
	var errs []domain.FieldError

	if i.NewCardsPerDay != nil {
		if *i.NewCardsPerDay < 1 {
			errs = append(errs, domain.FieldError{Field: "new_cards_per_day", Message: "must be at least 1"})
		} else if *i.NewCardsPerDay > 999 {
			errs = append(errs, domain.FieldError{Field: "new_cards_per_day", Message: "must be at most 999"})
		}
	}

	if i.ReviewsPerDay != nil {
		if *i.ReviewsPerDay < 1 {
			errs = append(errs, domain.FieldError{Field: "reviews_per_day", Message: "must be at least 1"})
		} else if *i.ReviewsPerDay > 9999 {
			errs = append(errs, domain.FieldError{Field: "reviews_per_day", Message: "must be at most 9999"})
		}
	}

	if i.MaxIntervalDays != nil {
		if *i.MaxIntervalDays < 1 {
			errs = append(errs, domain.FieldError{Field: "max_interval_days", Message: "must be at least 1"})
		} else if *i.MaxIntervalDays > 36500 {
			errs = append(errs, domain.FieldError{Field: "max_interval_days", Message: "must be at most 36500"})
		}
	}

	if i.DesiredRetention != nil {
		if *i.DesiredRetention < 0.70 {
			errs = append(errs, domain.FieldError{Field: "desired_retention", Message: "must be at least 0.70"})
		} else if *i.DesiredRetention > 0.99 {
			errs = append(errs, domain.FieldError{Field: "desired_retention", Message: "must be at most 0.99"})
		}
	}

	if i.Timezone != nil {
		if *i.Timezone == "" {
			errs = append(errs, domain.FieldError{Field: "timezone", Message: "cannot be empty"})
		} else if len(*i.Timezone) > 64 {
			errs = append(errs, domain.FieldError{Field: "timezone", Message: "too long"})
		} else if _, err := time.LoadLocation(*i.Timezone); err != nil {
			errs = append(errs, domain.FieldError{Field: "timezone", Message: "invalid IANA timezone"})
		}
	}

	if len(errs) > 0 {
		return &domain.ValidationError{Errors: errs}
	}
	return nil
}
