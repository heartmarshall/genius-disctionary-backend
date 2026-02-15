package user

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestService(users userRepo, settings settingsRepo, audit auditRepo, tx txManager) *Service {
	logger := slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewService(logger, users, settings, audit, tx)
}

func ptr[T any](v T) *T { return &v }

// ---------------------------------------------------------------------------
// GetProfile tests
// ---------------------------------------------------------------------------

func TestService_GetProfile_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	expected := domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Test User",
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google-123",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	users := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (domain.User, error) {
			assert.Equal(t, userID, id)
			return expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.GetProfile(ctx)

	require.NoError(t, err)
	assert.Equal(t, expected, user)
	assert.Len(t, users.GetByIDCalls(), 1)
}

func TestService_GetProfile_NoUserIDInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestService(nil, nil, nil, nil)

	user, err := svc.GetProfile(ctx)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Equal(t, domain.User{}, user)
}

func TestService_GetProfile_UserNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	users := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (domain.User, error) {
			return domain.User{}, domain.ErrNotFound
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.GetProfile(ctx)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Equal(t, domain.User{}, user)
}

// ---------------------------------------------------------------------------
// UpdateProfile tests
// ---------------------------------------------------------------------------

func TestService_UpdateProfile_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateProfileInput{
		Name:      "New Name",
		AvatarURL: ptr("https://example.com/avatar.jpg"),
	}

	expected := domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Name:      "New Name",
		AvatarURL: ptr("https://example.com/avatar.jpg"),
		UpdatedAt: time.Now().UTC(),
	}

	users := &userRepoMock{
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name string, avatarURL *string) (domain.User, error) {
			assert.Equal(t, userID, id)
			assert.Equal(t, "New Name", name)
			assert.Equal(t, ptr("https://example.com/avatar.jpg"), avatarURL)
			return expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.UpdateProfile(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, expected, user)
	assert.Len(t, users.UpdateCalls(), 1)
}

func TestService_UpdateProfile_ValidationError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	tests := []struct {
		name  string
		input UpdateProfileInput
	}{
		{
			name: "empty name",
			input: UpdateProfileInput{
				Name: "",
			},
		},
		{
			name: "name too long",
			input: UpdateProfileInput{
				Name: string(make([]byte, 256)),
			},
		},
		{
			name: "avatar_url too long",
			input: UpdateProfileInput{
				Name:      "Valid Name",
				AvatarURL: ptr(string(make([]byte, 513))),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestService(nil, nil, nil, nil)
			user, err := svc.UpdateProfile(ctx, tt.input)

			require.ErrorIs(t, err, domain.ErrValidation)
			assert.Equal(t, domain.User{}, user)
		})
	}
}

func TestService_UpdateProfile_NoUserIDInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	input := UpdateProfileInput{Name: "Valid Name"}

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.UpdateProfile(ctx, input)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Equal(t, domain.User{}, user)
}

// ---------------------------------------------------------------------------
// GetSettings tests
// ---------------------------------------------------------------------------

func TestService_GetSettings_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	expected := domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		ReviewsPerDay:   200,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
		UpdatedAt:       time.Now().UTC(),
	}

	settings := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (domain.UserSettings, error) {
			assert.Equal(t, userID, uid)
			return expected, nil
		},
	}

	svc := newTestService(nil, settings, nil, nil)
	result, err := svc.GetSettings(ctx)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Len(t, settings.GetSettingsCalls(), 1)
}

func TestService_GetSettings_NoUserIDInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestService(nil, nil, nil, nil)

	result, err := svc.GetSettings(ctx)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Equal(t, domain.UserSettings{}, result)
}

func TestService_GetSettings_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	settings := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (domain.UserSettings, error) {
			return domain.UserSettings{}, domain.ErrNotFound
		},
	}

	svc := newTestService(nil, settings, nil, nil)
	result, err := svc.GetSettings(ctx)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Equal(t, domain.UserSettings{}, result)
}

// ---------------------------------------------------------------------------
// UpdateSettings tests
// ---------------------------------------------------------------------------

func TestService_UpdateSettings_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateSettingsInput{
		NewCardsPerDay:  ptr(30),
		ReviewsPerDay:   ptr(250),
		MaxIntervalDays: ptr(400),
		Timezone:        ptr("America/New_York"),
	}

	current := domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		ReviewsPerDay:   200,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	expected := domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  30,
		ReviewsPerDay:   250,
		MaxIntervalDays: 400,
		Timezone:        "America/New_York",
		UpdatedAt:       time.Now().UTC(),
	}

	settingsRepo := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (domain.UserSettings, error) {
			return current, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (domain.UserSettings, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, 30, s.NewCardsPerDay)
			assert.Equal(t, 250, s.ReviewsPerDay)
			assert.Equal(t, 400, s.MaxIntervalDays)
			assert.Equal(t, "America/New_York", s.Timezone)
			return expected, nil
		},
	}

	auditRepo := &auditRepoMock{
		CreateFunc: func(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
			assert.Equal(t, userID, record.UserID)
			assert.Equal(t, domain.EntityTypeUser, record.EntityType)
			assert.Equal(t, &userID, record.EntityID)
			assert.Equal(t, domain.AuditActionUpdate, record.Action)
			assert.NotEmpty(t, record.Changes)
			return record, nil
		},
	}

	txMgr := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := newTestService(nil, settingsRepo, auditRepo, txMgr)
	result, err := svc.UpdateSettings(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Len(t, settingsRepo.GetSettingsCalls(), 1)
	assert.Len(t, settingsRepo.UpdateSettingsCalls(), 1)
	assert.Len(t, auditRepo.CreateCalls(), 1)
	assert.Len(t, txMgr.RunInTxCalls(), 1)
}

func TestService_UpdateSettings_PartialUpdate(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateSettingsInput{
		NewCardsPerDay: ptr(50),
		// Other fields nil - should not change
	}

	current := domain.UserSettings{
		UserID:          userID,
		NewCardsPerDay:  20,
		ReviewsPerDay:   200,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	settingsRepo := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (domain.UserSettings, error) {
			return current, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (domain.UserSettings, error) {
			// Verify only NewCardsPerDay changed
			assert.Equal(t, 50, s.NewCardsPerDay)
			assert.Equal(t, 200, s.ReviewsPerDay)
			assert.Equal(t, 365, s.MaxIntervalDays)
			assert.Equal(t, "UTC", s.Timezone)
			return s, nil
		},
	}

	auditRepo := &auditRepoMock{
		CreateFunc: func(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
			// Verify only one field in changes
			assert.Len(t, record.Changes, 1)
			assert.Contains(t, record.Changes, "new_cards_per_day")
			return record, nil
		},
	}

	txMgr := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := newTestService(nil, settingsRepo, auditRepo, txMgr)
	_, err := svc.UpdateSettings(ctx, input)

	require.NoError(t, err)
}

func TestService_UpdateSettings_ValidationError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	tests := []struct {
		name  string
		input UpdateSettingsInput
	}{
		{
			name: "new_cards_per_day too low",
			input: UpdateSettingsInput{
				NewCardsPerDay: ptr(0),
			},
		},
		{
			name: "new_cards_per_day too high",
			input: UpdateSettingsInput{
				NewCardsPerDay: ptr(1000),
			},
		},
		{
			name: "reviews_per_day too low",
			input: UpdateSettingsInput{
				ReviewsPerDay: ptr(0),
			},
		},
		{
			name: "reviews_per_day too high",
			input: UpdateSettingsInput{
				ReviewsPerDay: ptr(10000),
			},
		},
		{
			name: "max_interval_days too low",
			input: UpdateSettingsInput{
				MaxIntervalDays: ptr(0),
			},
		},
		{
			name: "max_interval_days too high",
			input: UpdateSettingsInput{
				MaxIntervalDays: ptr(36501),
			},
		},
		{
			name: "timezone empty",
			input: UpdateSettingsInput{
				Timezone: ptr(""),
			},
		},
		{
			name: "timezone too long",
			input: UpdateSettingsInput{
				Timezone: ptr(string(make([]byte, 65))),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newTestService(nil, nil, nil, nil)
			result, err := svc.UpdateSettings(ctx, tt.input)

			require.ErrorIs(t, err, domain.ErrValidation)
			assert.Equal(t, domain.UserSettings{}, result)
		})
	}
}

func TestService_UpdateSettings_NoUserIDInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	input := UpdateSettingsInput{NewCardsPerDay: ptr(30)}

	svc := newTestService(nil, nil, nil, nil)
	result, err := svc.UpdateSettings(ctx, input)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Equal(t, domain.UserSettings{}, result)
}

func TestService_UpdateSettings_TransactionRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateSettingsInput{NewCardsPerDay: ptr(30)}

	settingsRepo := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (domain.UserSettings, error) {
			return domain.UserSettings{UserID: userID, NewCardsPerDay: 20}, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (domain.UserSettings, error) {
			return s, nil
		},
	}

	auditRepo := &auditRepoMock{
		CreateFunc: func(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
			return domain.AuditRecord{}, errors.New("audit failed")
		},
	}

	txMgr := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx) // Return error from fn
		},
	}

	svc := newTestService(nil, settingsRepo, auditRepo, txMgr)
	result, err := svc.UpdateSettings(ctx, input)

	require.Error(t, err)
	assert.Equal(t, domain.UserSettings{}, result)
	assert.Contains(t, err.Error(), "audit failed")
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestApplySettingsChanges(t *testing.T) {
	t.Parallel()

	current := domain.UserSettings{
		UserID:          uuid.New(),
		NewCardsPerDay:  20,
		ReviewsPerDay:   200,
		MaxIntervalDays: 365,
		Timezone:        "UTC",
	}

	tests := []struct {
		name     string
		input    UpdateSettingsInput
		expected domain.UserSettings
	}{
		{
			name: "change all fields",
			input: UpdateSettingsInput{
				NewCardsPerDay:  ptr(30),
				ReviewsPerDay:   ptr(250),
				MaxIntervalDays: ptr(400),
				Timezone:        ptr("America/New_York"),
			},
			expected: domain.UserSettings{
				UserID:          current.UserID,
				NewCardsPerDay:  30,
				ReviewsPerDay:   250,
				MaxIntervalDays: 400,
				Timezone:        "America/New_York",
			},
		},
		{
			name: "change only new_cards_per_day",
			input: UpdateSettingsInput{
				NewCardsPerDay: ptr(50),
			},
			expected: domain.UserSettings{
				UserID:          current.UserID,
				NewCardsPerDay:  50,
				ReviewsPerDay:   200,
				MaxIntervalDays: 365,
				Timezone:        "UTC",
			},
		},
		{
			name:     "no changes",
			input:    UpdateSettingsInput{},
			expected: current,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := applySettingsChanges(current, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildSettingsChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		old      domain.UserSettings
		new      domain.UserSettings
		expected map[string]any
	}{
		{
			name: "all fields changed",
			old: domain.UserSettings{
				NewCardsPerDay:  20,
				ReviewsPerDay:   200,
				MaxIntervalDays: 365,
				Timezone:        "UTC",
			},
			new: domain.UserSettings{
				NewCardsPerDay:  30,
				ReviewsPerDay:   250,
				MaxIntervalDays: 400,
				Timezone:        "America/New_York",
			},
			expected: map[string]any{
				"new_cards_per_day": map[string]any{"old": 20, "new": 30},
				"reviews_per_day":   map[string]any{"old": 200, "new": 250},
				"max_interval_days": map[string]any{"old": 365, "new": 400},
				"timezone":          map[string]any{"old": "UTC", "new": "America/New_York"},
			},
		},
		{
			name: "only new_cards_per_day changed",
			old: domain.UserSettings{
				NewCardsPerDay:  20,
				ReviewsPerDay:   200,
				MaxIntervalDays: 365,
				Timezone:        "UTC",
			},
			new: domain.UserSettings{
				NewCardsPerDay:  50,
				ReviewsPerDay:   200,
				MaxIntervalDays: 365,
				Timezone:        "UTC",
			},
			expected: map[string]any{
				"new_cards_per_day": map[string]any{"old": 20, "new": 50},
			},
		},
		{
			name: "no changes",
			old: domain.UserSettings{
				NewCardsPerDay:  20,
				ReviewsPerDay:   200,
				MaxIntervalDays: 365,
				Timezone:        "UTC",
			},
			new: domain.UserSettings{
				NewCardsPerDay:  20,
				ReviewsPerDay:   200,
				MaxIntervalDays: 365,
				Timezone:        "UTC",
			},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := buildSettingsChanges(tt.old, tt.new)
			assert.Equal(t, tt.expected, result)
		})
	}
}
