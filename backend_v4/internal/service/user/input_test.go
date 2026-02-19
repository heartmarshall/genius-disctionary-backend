package user

import (
	"strings"
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// UpdateProfileInput.Validate boundary tests
// ---------------------------------------------------------------------------

func TestUpdateProfileInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   UpdateProfileInput
		wantErr bool
	}{
		{
			name:    "valid: name at max length (255)",
			input:   UpdateProfileInput{Name: strings.Repeat("a", 255)},
			wantErr: false,
		},
		{
			name:    "invalid: name at 256",
			input:   UpdateProfileInput{Name: strings.Repeat("a", 256)},
			wantErr: true,
		},
		{
			name:    "valid: avatar_url at max length (512)",
			input:   UpdateProfileInput{Name: "ok", AvatarURL: ptr(strings.Repeat("u", 512))},
			wantErr: false,
		},
		{
			name:    "invalid: avatar_url at 513",
			input:   UpdateProfileInput{Name: "ok", AvatarURL: ptr(strings.Repeat("u", 513))},
			wantErr: true,
		},
		{
			name:    "valid: nil avatar_url",
			input:   UpdateProfileInput{Name: "ok", AvatarURL: nil},
			wantErr: false,
		},
		{
			name:    "valid: single-char name",
			input:   UpdateProfileInput{Name: "A"},
			wantErr: false,
		},
		{
			name:    "invalid: empty name",
			input:   UpdateProfileInput{Name: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, domain.ErrValidation)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UpdateSettingsInput.Validate boundary tests
// ---------------------------------------------------------------------------

func TestUpdateSettingsInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   UpdateSettingsInput
		wantErr bool
	}{
		// NewCardsPerDay boundaries
		{
			name:    "valid: new_cards_per_day at min (1)",
			input:   UpdateSettingsInput{NewCardsPerDay: ptr(1)},
			wantErr: false,
		},
		{
			name:    "valid: new_cards_per_day at max (999)",
			input:   UpdateSettingsInput{NewCardsPerDay: ptr(999)},
			wantErr: false,
		},
		{
			name:    "invalid: new_cards_per_day below min (0)",
			input:   UpdateSettingsInput{NewCardsPerDay: ptr(0)},
			wantErr: true,
		},
		{
			name:    "invalid: new_cards_per_day above max (1000)",
			input:   UpdateSettingsInput{NewCardsPerDay: ptr(1000)},
			wantErr: true,
		},
		{
			name:    "invalid: new_cards_per_day negative",
			input:   UpdateSettingsInput{NewCardsPerDay: ptr(-1)},
			wantErr: true,
		},
		// ReviewsPerDay boundaries
		{
			name:    "valid: reviews_per_day at min (1)",
			input:   UpdateSettingsInput{ReviewsPerDay: ptr(1)},
			wantErr: false,
		},
		{
			name:    "valid: reviews_per_day at max (9999)",
			input:   UpdateSettingsInput{ReviewsPerDay: ptr(9999)},
			wantErr: false,
		},
		{
			name:    "invalid: reviews_per_day below min (0)",
			input:   UpdateSettingsInput{ReviewsPerDay: ptr(0)},
			wantErr: true,
		},
		{
			name:    "invalid: reviews_per_day above max (10000)",
			input:   UpdateSettingsInput{ReviewsPerDay: ptr(10000)},
			wantErr: true,
		},
		// MaxIntervalDays boundaries
		{
			name:    "valid: max_interval_days at min (1)",
			input:   UpdateSettingsInput{MaxIntervalDays: ptr(1)},
			wantErr: false,
		},
		{
			name:    "valid: max_interval_days at max (36500)",
			input:   UpdateSettingsInput{MaxIntervalDays: ptr(36500)},
			wantErr: false,
		},
		{
			name:    "invalid: max_interval_days below min (0)",
			input:   UpdateSettingsInput{MaxIntervalDays: ptr(0)},
			wantErr: true,
		},
		{
			name:    "invalid: max_interval_days above max (36501)",
			input:   UpdateSettingsInput{MaxIntervalDays: ptr(36501)},
			wantErr: true,
		},
		// Timezone boundaries
		{
			name:    "valid: timezone at max length (64)",
			input:   UpdateSettingsInput{Timezone: ptr(strings.Repeat("z", 64))},
			wantErr: false,
		},
		{
			name:    "invalid: timezone at 65",
			input:   UpdateSettingsInput{Timezone: ptr(strings.Repeat("z", 65))},
			wantErr: true,
		},
		{
			name:    "invalid: timezone empty",
			input:   UpdateSettingsInput{Timezone: ptr("")},
			wantErr: true,
		},
		{
			name:    "valid: timezone normal value",
			input:   UpdateSettingsInput{Timezone: ptr("America/New_York")},
			wantErr: false,
		},
		// All nil = no error
		{
			name:    "valid: all fields nil",
			input:   UpdateSettingsInput{},
			wantErr: false,
		},
		// Multiple errors at once
		{
			name: "invalid: multiple fields invalid",
			input: UpdateSettingsInput{
				NewCardsPerDay: ptr(0),
				ReviewsPerDay:  ptr(0),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.input.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, domain.ErrValidation)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpdateSettingsInput_Validate_MultipleErrors(t *testing.T) {
	t.Parallel()

	input := UpdateSettingsInput{
		NewCardsPerDay:  ptr(0),
		ReviewsPerDay:   ptr(0),
		MaxIntervalDays: ptr(0),
		Timezone:        ptr(""),
	}

	err := input.Validate()
	require.ErrorIs(t, err, domain.ErrValidation)

	var valErr *domain.ValidationError
	require.ErrorAs(t, err, &valErr)
	assert.Len(t, valErr.Errors, 4, "each invalid field should produce a separate error")
}
