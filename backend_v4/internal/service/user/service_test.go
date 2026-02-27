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
		Username: "testuser",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	users := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			assert.Equal(t, userID, id)
			return &expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.GetProfile(ctx)

	require.NoError(t, err)
	assert.Equal(t, &expected, user)
	assert.Len(t, users.GetByIDCalls(), 1)
}

func TestService_GetProfile_NoUserIDInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestService(nil, nil, nil, nil)

	user, err := svc.GetProfile(ctx)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Nil(t, user)
}

func TestService_GetProfile_UserNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	users := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.GetProfile(ctx)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, user)
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
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
			assert.Equal(t, userID, id)
			assert.Equal(t, ptr("New Name"), name)
			assert.Equal(t, ptr("https://example.com/avatar.jpg"), avatarURL)
			return &expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.UpdateProfile(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, &expected, user)
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
			assert.Nil(t, user)
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
	assert.Nil(t, user)
}

func TestService_UpdateProfile_RepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateProfileInput{Name: "Valid Name"}
	repoErr := errors.New("db connection lost")

	users := &userRepoMock{
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
			return nil, repoErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.UpdateProfile(ctx, input)

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, user)
}

func TestService_UpdateProfile_NilAvatarURL(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateProfileInput{
		Name:      "New Name",
		AvatarURL: nil,
	}

	expected := domain.User{
		ID:   userID,
		Name: "New Name",
	}

	users := &userRepoMock{
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
			assert.Nil(t, avatarURL, "nil AvatarURL should be passed through to repo")
			return &expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.UpdateProfile(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, &expected, user)
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
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			assert.Equal(t, userID, uid)
			return &expected, nil
		},
	}

	svc := newTestService(nil, settings, nil, nil)
	result, err := svc.GetSettings(ctx)

	require.NoError(t, err)
	assert.Equal(t, &expected, result)
	assert.Len(t, settings.GetSettingsCalls(), 1)
}

func TestService_GetSettings_NoUserIDInContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := newTestService(nil, nil, nil, nil)

	result, err := svc.GetSettings(ctx)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Nil(t, result)
}

func TestService_GetSettings_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	settings := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(nil, settings, nil, nil)
	result, err := svc.GetSettings(ctx)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, result)
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
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return &current, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, 30, s.NewCardsPerDay)
			assert.Equal(t, 250, s.ReviewsPerDay)
			assert.Equal(t, 400, s.MaxIntervalDays)
			assert.Equal(t, "America/New_York", s.Timezone)
			return &expected, nil
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
	assert.Equal(t, &expected, result)
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
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return &current, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error) {
			// Verify only NewCardsPerDay changed
			assert.Equal(t, 50, s.NewCardsPerDay)
			assert.Equal(t, 200, s.ReviewsPerDay)
			assert.Equal(t, 365, s.MaxIntervalDays)
			assert.Equal(t, "UTC", s.Timezone)
			return &s, nil
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
			assert.Nil(t, result)
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
	assert.Nil(t, result)
}

func TestService_UpdateSettings_TransactionRollback(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateSettingsInput{NewCardsPerDay: ptr(30)}

	settingsRepo := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return &domain.UserSettings{UserID: userID, NewCardsPerDay: 20}, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error) {
			return &s, nil
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
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "audit failed")
}

func TestService_UpdateSettings_GetSettingsRepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateSettingsInput{NewCardsPerDay: ptr(30)}
	repoErr := errors.New("settings repo down")

	settingsRepo := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return nil, repoErr
		},
	}

	txMgr := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := newTestService(nil, settingsRepo, nil, txMgr)
	result, err := svc.UpdateSettings(ctx, input)

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, result)
}

func TestService_UpdateSettings_UpdateSettingsRepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateSettingsInput{NewCardsPerDay: ptr(30)}
	repoErr := errors.New("update failed")

	settingsRepo := &settingsRepoMock{
		GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
			return &domain.UserSettings{UserID: userID, NewCardsPerDay: 20}, nil
		},
		UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error) {
			return nil, repoErr
		},
	}

	txMgr := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	svc := newTestService(nil, settingsRepo, nil, txMgr)
	result, err := svc.UpdateSettings(ctx, input)

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, result)
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

// ---------------------------------------------------------------------------
// SetUserRole tests
// ---------------------------------------------------------------------------

func TestService_SetUserRole_Success(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	targetID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	expected := domain.User{
		ID:    targetID,
		Email: "target@example.com",
		Name:  "Target User",
		Role:  domain.UserRoleAdmin,
	}

	users := &userRepoMock{
		UpdateRoleFunc: func(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
			assert.Equal(t, targetID, id)
			assert.Equal(t, "admin", role)
			return &expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, targetID, domain.UserRoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, &expected, user)
	assert.Len(t, users.UpdateRoleCalls(), 1)
}

func TestService_SetUserRole_NotAdmin(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), userID), "user")

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRoleAdmin)

	require.ErrorIs(t, err, domain.ErrForbidden)
	assert.Nil(t, user)
}

func TestService_SetUserRole_SelfDemotion(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, callerID, domain.UserRoleUser)

	require.ErrorIs(t, err, domain.ErrValidation)
	assert.Nil(t, user)
}

func TestService_SetUserRole_InvalidRole(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRole("superadmin"))

	require.ErrorIs(t, err, domain.ErrValidation)
	assert.Nil(t, user)
}

func TestService_SetUserRole_RepoError(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	repoErr := errors.New("db connection lost")

	users := &userRepoMock{
		UpdateRoleFunc: func(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
			return nil, repoErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRoleAdmin)

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, user)
}

func TestService_SetUserRole_TargetNotFound(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	users := &userRepoMock{
		UpdateRoleFunc: func(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRoleAdmin)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, user)
}

// ---------------------------------------------------------------------------
// ListUsers tests
// ---------------------------------------------------------------------------

func TestService_ListUsers_Success(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	expectedUsers := []domain.User{
		{ID: uuid.New(), Email: "a@example.com", Name: "Alice", Role: domain.UserRoleUser},
		{ID: uuid.New(), Email: "b@example.com", Name: "Bob", Role: domain.UserRoleAdmin},
	}

	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit int, offset int) ([]domain.User, error) {
			assert.Equal(t, 10, limit)
			assert.Equal(t, 5, offset)
			return expectedUsers, nil
		},
		CountUsersFunc: func(ctx context.Context) (int, error) {
			return 42, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 5)

	require.NoError(t, err)
	assert.Equal(t, expectedUsers, result)
	assert.Equal(t, 42, total)
	assert.Len(t, users.ListUsersCalls(), 1)
	assert.Len(t, users.CountUsersCalls(), 1)
}

func TestService_ListUsers_NotAdmin(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), userID), "user")

	svc := newTestService(nil, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 0)

	require.ErrorIs(t, err, domain.ErrForbidden)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}

func TestService_ListUsers_DefaultLimit(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit int, offset int) ([]domain.User, error) {
			assert.Equal(t, 50, limit, "limit=0 should default to 50")
			return nil, nil
		},
		CountUsersFunc: func(ctx context.Context) (int, error) {
			return 0, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	_, _, err := svc.ListUsers(ctx, 0, 0)

	require.NoError(t, err)
	assert.Len(t, users.ListUsersCalls(), 1)
}

func TestService_ListUsers_RepoError(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	repoErr := errors.New("db connection lost")

	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit int, offset int) ([]domain.User, error) {
			return nil, repoErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 0)

	require.Error(t, err)
	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}

func TestService_ListUsers_CountError(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserRole(ctxutil.WithUserID(context.Background(), callerID), "admin")

	countErr := errors.New("count query failed")

	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit int, offset int) ([]domain.User, error) {
			return []domain.User{}, nil
		},
		CountUsersFunc: func(ctx context.Context) (int, error) {
			return 0, countErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 0)

	require.Error(t, err)
	require.ErrorIs(t, err, countErr)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}
