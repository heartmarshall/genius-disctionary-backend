package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

//go:generate moq -out user_repo_mock_test.go -pkg auth . userRepo
//go:generate moq -out settings_repo_mock_test.go -pkg auth . settingsRepo
//go:generate moq -out token_repo_mock_test.go -pkg auth . tokenRepo
//go:generate moq -out tx_manager_mock_test.go -pkg auth . txManager
//go:generate moq -out oauth_verifier_mock_test.go -pkg auth . oauthVerifier
//go:generate moq -out jwt_manager_mock_test.go -pkg auth . jwtManager

func TestService_Login_NewUserRegistration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			if p != provider || c != code {
				t.Errorf("VerifyCode called with wrong params: provider=%s, code=%s", p, c)
			}
			return identity, nil
		},
	}

	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			created := *user
			created.ID = userID
			created.CreatedAt = time.Now()
			created.UpdatedAt = time.Now()
			return &created, nil
		},
	}

	settingsMock := &settingsRepoMock{
		CreateSettingsFunc: func(ctx context.Context, settings *domain.UserSettings) error {
			if settings.UserID != userID {
				t.Errorf("CreateSettings called with wrong userID: got=%s, want=%s", settings.UserID, userID)
			}
			return nil
		},
	}

	txMock := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			if uid != userID {
				t.Errorf("GenerateAccessToken called with wrong userID: got=%s, want=%s", uid, userID)
			}
			return "access_token_123", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "raw_refresh_123", "hash_refresh_123", nil
		},
	}

	tokensMock := &tokenRepoMock{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			if token.UserID != userID {
				t.Errorf("tokens.Create called with wrong userID: got=%s, want=%s", token.UserID, userID)
			}
			if token.TokenHash != "hash_refresh_123" {
				t.Errorf("tokens.Create called with wrong hash: got=%s, want=%s", token.TokenHash, "hash_refresh_123")
			}
			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		settingsMock,
		tokensMock,
		txMock,
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}
	if result.AccessToken != "access_token_123" {
		t.Errorf("AccessToken: got=%s, want=%s", result.AccessToken, "access_token_123")
	}
	if result.RefreshToken != "raw_refresh_123" {
		t.Errorf("RefreshToken: got=%s, want=%s (should be raw, not hash)", result.RefreshToken, "raw_refresh_123")
	}
	if result.User == nil {
		t.Fatal("User is nil")
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}

	// Verify all mocks were called
	if len(oauthMock.VerifyCodeCalls()) != 1 {
		t.Errorf("VerifyCode called %d times, want 1", len(oauthMock.VerifyCodeCalls()))
	}
	if len(usersMock.GetByOAuthCalls()) != 1 {
		t.Errorf("GetByOAuth called %d times, want 1", len(usersMock.GetByOAuthCalls()))
	}
	if len(usersMock.CreateCalls()) != 1 {
		t.Errorf("Create called %d times, want 1", len(usersMock.CreateCalls()))
	}
	if len(settingsMock.CreateSettingsCalls()) != 1 {
		t.Errorf("CreateSettings called %d times, want 1", len(settingsMock.CreateSettingsCalls()))
	}
	if len(jwtMock.GenerateAccessTokenCalls()) != 1 {
		t.Errorf("GenerateAccessToken called %d times, want 1", len(jwtMock.GenerateAccessTokenCalls()))
	}
	if len(jwtMock.GenerateRefreshTokenCalls()) != 1 {
		t.Errorf("GenerateRefreshToken called %d times, want 1", len(jwtMock.GenerateRefreshTokenCalls()))
	}
	if len(tokensMock.CreateCalls()) != 1 {
		t.Errorf("tokens.Create called %d times, want 1", len(tokensMock.CreateCalls()))
	}
}

func TestService_Login_ExistingUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Test User",
		AvatarURL:     ptrString("https://example.com/avatar.jpg"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google_123",
		CreatedAt:     time.Now().Add(-24 * time.Hour),
		UpdatedAt:     time.Now().Add(-24 * time.Hour),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			return existingUser, nil
		},
	}

	settingsMock := &settingsRepoMock{}
	txMock := &txManagerMock{}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "access_token_123", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "raw_refresh_123", "hash_refresh_123", nil
		},
	}

	tokensMock := &tokenRepoMock{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		settingsMock,
		tokensMock,
		txMock,
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}

	// Verify Create was NOT called (existing user)
	if len(usersMock.CreateCalls()) != 0 {
		t.Errorf("Create called %d times, want 0 (existing user)", len(usersMock.CreateCalls()))
	}
	if len(settingsMock.CreateSettingsCalls()) != 0 {
		t.Errorf("CreateSettings called %d times, want 0 (existing user)", len(settingsMock.CreateSettingsCalls()))
	}
}

func TestService_Login_ProfileChanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Old Name",
		AvatarURL:     ptrString("https://example.com/old_avatar.jpg"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google_123",
		CreatedAt:     time.Now().Add(-24 * time.Hour),
		UpdatedAt:     time.Now().Add(-24 * time.Hour),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("New Name"),
		AvatarURL:  ptrString("https://example.com/new_avatar.jpg"),
	}

	updatedUser := &domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "New Name",
		AvatarURL:     ptrString("https://example.com/new_avatar.jpg"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google_123",
		CreatedAt:     existingUser.CreatedAt,
		UpdatedAt:     time.Now(),
	}

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			return existingUser, nil
		},
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
			if id != userID {
				t.Errorf("Update called with wrong userID: got=%s, want=%s", id, userID)
			}
			if name == nil || *name != "New Name" {
				t.Errorf("Update called with wrong name: got=%v, want=%s", name, "New Name")
			}
			if avatarURL == nil || *avatarURL != "https://example.com/new_avatar.jpg" {
				t.Errorf("Update called with wrong avatarURL: got=%v, want=%s", avatarURL, "https://example.com/new_avatar.jpg")
			}
			return updatedUser, nil
		},
	}

	settingsMock := &settingsRepoMock{}
	txMock := &txManagerMock{}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "access_token_123", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "raw_refresh_123", "hash_refresh_123", nil
		},
	}

	tokensMock := &tokenRepoMock{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		settingsMock,
		tokensMock,
		txMock,
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}
	if result.User.Name != "New Name" {
		t.Errorf("User.Name: got=%s, want=%s", result.User.Name, "New Name")
	}

	// Verify Update was called
	if len(usersMock.UpdateCalls()) != 1 {
		t.Errorf("Update called %d times, want 1", len(usersMock.UpdateCalls()))
	}
}

func TestService_Login_ProfileNotChanged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Same Name",
		AvatarURL:     ptrString("https://example.com/same_avatar.jpg"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google_123",
		CreatedAt:     time.Now().Add(-24 * time.Hour),
		UpdatedAt:     time.Now().Add(-24 * time.Hour),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Same Name"),
		AvatarURL:  ptrString("https://example.com/same_avatar.jpg"),
	}

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			return existingUser, nil
		},
	}

	settingsMock := &settingsRepoMock{}
	txMock := &txManagerMock{}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "access_token_123", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "raw_refresh_123", "hash_refresh_123", nil
		},
	}

	tokensMock := &tokenRepoMock{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		settingsMock,
		tokensMock,
		txMock,
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}

	// Verify Update was NOT called (profile unchanged)
	if len(usersMock.UpdateCalls()) != 0 {
		t.Errorf("Update called %d times, want 0 (profile not changed)", len(usersMock.UpdateCalls()))
	}
}

func TestService_Login_RaceCondition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Test User",
		AvatarURL:     ptrString("https://example.com/avatar.jpg"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google_123",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	getByOAuthCallCount := 0
	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			getByOAuthCallCount++
			if getByOAuthCallCount == 1 {
				// First call: user not found (both requests arrive at same time)
				return nil, domain.ErrNotFound
			}
			// Second call (retry after race): user found (created by concurrent request)
			return existingUser, nil
		},
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			// Simulate race condition: another request already created the user
			return nil, domain.ErrAlreadyExists
		},
	}

	settingsMock := &settingsRepoMock{}

	txMock := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "access_token_123", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "raw_refresh_123", "hash_refresh_123", nil
		},
	}

	tokensMock := &tokenRepoMock{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		settingsMock,
		tokensMock,
		txMock,
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}

	// Verify GetByOAuth was called twice (initial + retry)
	if len(usersMock.GetByOAuthCalls()) != 2 {
		t.Errorf("GetByOAuth called %d times, want 2 (initial + retry)", len(usersMock.GetByOAuthCalls()))
	}
	// Verify Create was called once (failed with ErrAlreadyExists)
	if len(usersMock.CreateCalls()) != 1 {
		t.Errorf("Create called %d times, want 1", len(usersMock.CreateCalls()))
	}
	// Verify CreateSettings was NOT called (transaction rolled back)
	if len(settingsMock.CreateSettingsCalls()) != 0 {
		t.Errorf("CreateSettings called %d times, want 0 (tx rolled back)", len(settingsMock.CreateSettingsCalls()))
	}
}

func TestService_Login_EmailCollision(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider := "google"
	code := "auth_code_123"

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	getByOAuthCallCount := 0
	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			getByOAuthCallCount++
			// Both initial and retry return ErrNotFound (email collision from different provider)
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			// Create fails due to email collision (ux_users_email constraint)
			return nil, domain.ErrAlreadyExists
		},
	}

	settingsMock := &settingsRepoMock{}

	txMock := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	jwtMock := &jwtManagerMock{}
	tokensMock := &tokenRepoMock{}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		settingsMock,
		tokensMock,
		txMock,
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("Login error: got=%v, want=ErrAlreadyExists", err)
	}
	if result != nil {
		t.Fatal("Login should return nil result on email collision")
	}

	// Verify GetByOAuth was called twice (initial + retry)
	if len(usersMock.GetByOAuthCalls()) != 2 {
		t.Errorf("GetByOAuth called %d times, want 2 (initial + retry)", len(usersMock.GetByOAuthCalls()))
	}
}

func TestService_Login_ValidationErrors(t *testing.T) {
	t.Parallel()

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		&userRepoMock{},
		&settingsRepoMock{},
		&tokenRepoMock{},
		&txManagerMock{},
		&oauthVerifierMock{},
		&jwtManagerMock{},
		cfg,
	)

	tests := []struct {
		name      string
		input     LoginInput
		wantField string
		wantMsg   string
	}{
		{
			name:      "empty provider",
			input:     LoginInput{Provider: "", Code: "abc"},
			wantField: "provider",
			wantMsg:   "required",
		},
		{
			name:      "unsupported provider",
			input:     LoginInput{Provider: "facebook", Code: "abc"},
			wantField: "provider",
			wantMsg:   "unsupported provider",
		},
		{
			name:      "empty code",
			input:     LoginInput{Provider: "google", Code: ""},
			wantField: "code",
			wantMsg:   "required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			result, err := svc.Login(ctx, tt.input)

			if result != nil {
				t.Errorf("Login should return nil result on validation error")
			}

			var valErr *domain.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("Login error: got=%v, want=ValidationError", err)
			}

			if len(valErr.Errors) == 0 {
				t.Fatal("ValidationError.Errors is empty")
			}

			found := false
			for _, fieldErr := range valErr.Errors {
				if fieldErr.Field == tt.wantField && fieldErr.Message == tt.wantMsg {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("ValidationError missing: field=%s, message=%s. Got: %v", tt.wantField, tt.wantMsg, valErr.Errors)
			}
		})
	}
}

func TestService_Login_OAuthVerificationFailed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	provider := "google"
	code := "invalid_code"

	oauthErr := errors.New("oauth provider error")

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return nil, oauthErr
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
	}

	svc := NewService(
		slog.Default(),
		&userRepoMock{},
		&settingsRepoMock{},
		&tokenRepoMock{},
		&txManagerMock{},
		oauthMock,
		&jwtManagerMock{},
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err == nil {
		t.Fatal("Login should return error when OAuth verification fails")
	}
	if result != nil {
		t.Fatal("Login should return nil result on OAuth error")
	}
	if !errors.Is(err, oauthErr) {
		t.Errorf("Login error should wrap oauth error: got=%v, want=%v", err, oauthErr)
	}
}

func TestService_Login_TokensGeneratedCorrectly(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:            userID,
		Email:         "test@example.com",
		Name:          "Test User",
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google_123",
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
	}

	accessTokenGenerated := false
	refreshTokenGenerated := false
	refreshTokenStored := false

	// Setup mocks
	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	usersMock := &userRepoMock{
		GetByOAuthFunc: func(ctx context.Context, p domain.OAuthProvider, oauthID string) (*domain.User, error) {
			return existingUser, nil
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			if uid != userID {
				t.Errorf("GenerateAccessToken called with wrong userID: got=%s, want=%s", uid, userID)
			}
			accessTokenGenerated = true
			return "access_token_123", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			refreshTokenGenerated = true
			return "raw_refresh_123", "hash_refresh_123", nil
		},
	}

	refreshTokenTTL := 30 * 24 * time.Hour
	tokensMock := &tokenRepoMock{
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			refreshTokenStored = true

			if token.UserID != userID {
				t.Errorf("tokens.Create: UserID: got=%s, want=%s", token.UserID, userID)
			}
			if token.TokenHash != "hash_refresh_123" {
				t.Errorf("tokens.Create: TokenHash: got=%s, want=%s (should be hash, not raw)", token.TokenHash, "hash_refresh_123")
			}

			// Check ExpiresAt is approximately now + RefreshTokenTTL
			expectedExpiry := time.Now().Add(refreshTokenTTL)
			diff := token.ExpiresAt.Sub(expectedExpiry)
			if diff < -time.Second || diff > time.Second {
				t.Errorf("tokens.Create: ExpiresAt: got=%s, want≈%s (diff=%s)", token.ExpiresAt, expectedExpiry, diff)
			}

			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    refreshTokenTTL,
	}

	svc := NewService(
		slog.Default(),
		usersMock,
		&settingsRepoMock{},
		tokensMock,
		&txManagerMock{},
		oauthMock,
		jwtMock,
		cfg,
	)

	// Execute
	input := LoginInput{Provider: provider, Code: code}
	result, err := svc.Login(ctx, input)

	// Assert
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}

	if !accessTokenGenerated {
		t.Error("Access token was not generated")
	}
	if !refreshTokenGenerated {
		t.Error("Refresh token was not generated")
	}
	if !refreshTokenStored {
		t.Error("Refresh token was not stored")
	}

	if result.AccessToken != "access_token_123" {
		t.Errorf("AccessToken: got=%s, want=%s", result.AccessToken, "access_token_123")
	}
	if result.RefreshToken != "raw_refresh_123" {
		t.Errorf("RefreshToken: got=%s, want=%s (should be raw, not hash)", result.RefreshToken, "raw_refresh_123")
	}
}

// Helper function to create *string
func ptrString(s string) *string {
	return &s
}
