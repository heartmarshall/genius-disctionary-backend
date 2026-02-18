package auth

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/auth"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

//go:generate moq -out user_repo_mock_test.go -pkg auth . userRepo
//go:generate moq -out settings_repo_mock_test.go -pkg auth . settingsRepo
//go:generate moq -out token_repo_mock_test.go -pkg auth . tokenRepo
//go:generate moq -out auth_method_repo_mock_test.go -pkg auth . authMethodRepo
//go:generate moq -out tx_manager_mock_test.go -pkg auth . txManager
//go:generate moq -out oauth_verifier_mock_test.go -pkg auth . oauthVerifier
//go:generate moq -out jwt_manager_mock_test.go -pkg auth . jwtManager

// defaultCfg returns a config suitable for most tests.
func defaultCfg() config.AuthConfig {
	return config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    30 * 24 * time.Hour,
		PasswordHashCost:   4, // minimum cost for fast tests
	}
}

// hashPassword returns a bcrypt hash for testing.
func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		t.Fatalf("hashPassword: %v", err)
	}
	return string(hash)
}

// ─── OAuth Login Tests ──────────────────────────────────────────────────────

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

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			if p != provider || c != code {
				t.Errorf("VerifyCode called with wrong params: provider=%s, code=%s", p, c)
			}
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, am *domain.AuthMethod) (*domain.AuthMethod, error) {
			created := *am
			created.ID = uuid.New()
			return &created, nil
		},
	}

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
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
			return nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

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
		t.Errorf("RefreshToken: got=%s, want=%s", result.RefreshToken, "raw_refresh_123")
	}
	if result.User == nil {
		t.Fatal("User is nil")
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}

	// Verify mocks
	if len(oauthMock.VerifyCodeCalls()) != 1 {
		t.Errorf("VerifyCode called %d times, want 1", len(oauthMock.VerifyCodeCalls()))
	}
	if len(authMethodsMock.GetByOAuthCalls()) != 1 {
		t.Errorf("authMethods.GetByOAuth called %d times, want 1", len(authMethodsMock.GetByOAuthCalls()))
	}
	if len(usersMock.GetByEmailCalls()) != 1 {
		t.Errorf("GetByEmail called %d times, want 1", len(usersMock.GetByEmailCalls()))
	}
	if len(usersMock.CreateCalls()) != 1 {
		t.Errorf("Create called %d times, want 1", len(usersMock.CreateCalls()))
	}
	if len(authMethodsMock.CreateCalls()) != 1 {
		t.Errorf("authMethods.Create called %d times, want 1", len(authMethodsMock.CreateCalls()))
	}
	if len(settingsMock.CreateSettingsCalls()) != 1 {
		t.Errorf("CreateSettings called %d times, want 1", len(settingsMock.CreateSettingsCalls()))
	}
}

func TestService_Login_ExistingUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "test",
		Name:      "Test User",
		AvatarURL: ptrString("https://example.com/avatar.jpg"),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	existingAM := &domain.AuthMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Method:     domain.AuthMethodGoogle,
		ProviderID: ptrString("google_123"),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			return existingAM, nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

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
		ID:        userID,
		Email:     "test@example.com",
		Username:  "test",
		Name:      "Old Name",
		AvatarURL: ptrString("https://example.com/old_avatar.jpg"),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	existingAM := &domain.AuthMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Method:     domain.AuthMethodGoogle,
		ProviderID: ptrString("google_123"),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("New Name"),
		AvatarURL:  ptrString("https://example.com/new_avatar.jpg"),
	}

	updatedUser := &domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "test",
		Name:      "New Name",
		AvatarURL: ptrString("https://example.com/new_avatar.jpg"),
		CreatedAt: existingUser.CreatedAt,
		UpdatedAt: time.Now(),
	}

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			return existingAM, nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
			if id != userID {
				t.Errorf("Update called with wrong userID: got=%s, want=%s", id, userID)
			}
			if name == nil || *name != "New Name" {
				t.Errorf("Update called with wrong name: got=%v, want=%s", name, "New Name")
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.User.Name != "New Name" {
		t.Errorf("User.Name: got=%s, want=%s", result.User.Name, "New Name")
	}
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
		ID:        userID,
		Email:     "test@example.com",
		Username:  "test",
		Name:      "Same Name",
		AvatarURL: ptrString("https://example.com/same_avatar.jpg"),
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	existingAM := &domain.AuthMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Method:     domain.AuthMethodGoogle,
		ProviderID: ptrString("google_123"),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Same Name"),
		AvatarURL:  ptrString("https://example.com/same_avatar.jpg"),
	}

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			return existingAM, nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Login returned nil result")
	}
	if len(usersMock.UpdateCalls()) != 0 {
		t.Errorf("Update called %d times, want 0 (profile not changed)", len(usersMock.UpdateCalls()))
	}
}

func TestService_Login_AccountLinking(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	// User registered via password, now logging in via Google
	existingUser := &domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "testuser",
		Name:      "Test User",
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now().Add(-24 * time.Hour),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, am *domain.AuthMethod) (*domain.AuthMethod, error) {
			if am.UserID != userID {
				t.Errorf("authMethods.Create: UserID: got=%s, want=%s", am.UserID, userID)
			}
			if am.Method != domain.AuthMethodGoogle {
				t.Errorf("authMethods.Create: Method: got=%s, want=%s", am.Method, domain.AuthMethodGoogle)
			}
			created := *am
			created.ID = uuid.New()
			return &created, nil
		},
	}

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return existingUser, nil
		},
		UpdateFunc: func(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
			updated := *existingUser
			if avatarURL != nil {
				updated.AvatarURL = avatarURL
			}
			return &updated, nil
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}

	// Auth method was linked
	if len(authMethodsMock.CreateCalls()) != 1 {
		t.Errorf("authMethods.Create called %d times, want 1", len(authMethodsMock.CreateCalls()))
	}
	// No new user was created
	if len(usersMock.CreateCalls()) != 0 {
		t.Errorf("users.Create called %d times, want 0 (account linking)", len(usersMock.CreateCalls()))
	}
}

func TestService_Login_RaceCondition(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	provider := "google"
	code := "auth_code_123"

	existingUser := &domain.User{
		ID:        userID,
		Email:     "test@example.com",
		Username:  "test",
		Name:      "Test User",
		AvatarURL: ptrString("https://example.com/avatar.jpg"),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	existingAM := &domain.AuthMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Method:     domain.AuthMethodGoogle,
		ProviderID: ptrString("google_123"),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
		AvatarURL:  ptrString("https://example.com/avatar.jpg"),
	}

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	getByOAuthCallCount := 0
	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			getByOAuthCallCount++
			if getByOAuthCallCount == 1 {
				return nil, domain.ErrNotFound
			}
			// Retry after race: auth method now exists (created by concurrent request)
			return existingAM, nil
		},
	}

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			// Simulate race condition: another request already created the user
			return nil, domain.ErrAlreadyExists
		},
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}

	// GetByOAuth called twice (initial + retry after race)
	if len(authMethodsMock.GetByOAuthCalls()) != 2 {
		t.Errorf("authMethods.GetByOAuth called %d times, want 2 (initial + retry)", len(authMethodsMock.GetByOAuthCalls()))
	}
	if len(usersMock.CreateCalls()) != 1 {
		t.Errorf("users.Create called %d times, want 1", len(usersMock.CreateCalls()))
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

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			// Both initial and retry return ErrNotFound (email collision, not OAuth collision)
			return nil, domain.ErrNotFound
		},
	}

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			// Create fails due to username collision or similar unique constraint
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: provider, Code: code})

	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("Login error: got=%v, want=ErrAlreadyExists", err)
	}
	if result != nil {
		t.Fatal("Login should return nil result on email collision")
	}

	// GetByOAuth called twice (initial + retry)
	if len(authMethodsMock.GetByOAuthCalls()) != 2 {
		t.Errorf("authMethods.GetByOAuth called %d times, want 2 (initial + retry)", len(authMethodsMock.GetByOAuthCalls()))
	}
}

func TestService_Login_ValidationErrors(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{},
		&authMethodRepoMock{}, &txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
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
	oauthErr := errors.New("oauth provider error")

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return nil, oauthErr
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{},
		&authMethodRepoMock{}, &txManagerMock{}, oauthMock, &jwtManagerMock{}, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: "google", Code: "invalid_code"})

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

	existingUser := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "test",
		Name:     "Test User",
	}

	existingAM := &domain.AuthMethod{
		ID:         uuid.New(),
		UserID:     userID,
		Method:     domain.AuthMethodGoogle,
		ProviderID: ptrString("google_123"),
	}

	identity := &auth.OAuthIdentity{
		ProviderID: "google_123",
		Email:      "test@example.com",
		Name:       ptrString("Test User"),
	}

	accessTokenGenerated := false
	refreshTokenGenerated := false
	refreshTokenStored := false

	oauthMock := &oauthVerifierMock{
		VerifyCodeFunc: func(ctx context.Context, p, c string) (*auth.OAuthIdentity, error) {
			return identity, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByOAuthFunc: func(ctx context.Context, method domain.AuthMethodType, providerID string) (*domain.AuthMethod, error) {
			return existingAM, nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
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
				t.Errorf("tokens.Create: TokenHash: got=%s, want=%s", token.TokenHash, "hash_refresh_123")
			}
			expectedExpiry := time.Now().Add(refreshTokenTTL)
			diff := token.ExpiresAt.Sub(expectedExpiry)
			if diff < -time.Second || diff > time.Second {
				t.Errorf("tokens.Create: ExpiresAt: got=%s, want~%s (diff=%s)", token.ExpiresAt, expectedExpiry, diff)
			}
			return nil
		},
	}

	cfg := config.AuthConfig{
		GoogleClientID:     "google_client_id",
		GoogleClientSecret: "google_client_secret",
		RefreshTokenTTL:    refreshTokenTTL,
		PasswordHashCost:   4,
	}

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, authMethodsMock,
		&txManagerMock{}, oauthMock, jwtMock, cfg,
	)

	result, err := svc.Login(ctx, LoginInput{Provider: "google", Code: "auth_code_123"})

	if err != nil {
		t.Fatalf("Login returned error: %v", err)
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
		t.Errorf("RefreshToken: got=%s, want=%s", result.RefreshToken, "raw_refresh_123")
	}
}

// ─── Password Registration Tests ────────────────────────────────────────────

func TestService_Register_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()

	usersMock := &userRepoMock{
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			if user.Email != "new@example.com" {
				t.Errorf("Create email: got=%s, want=%s", user.Email, "new@example.com")
			}
			if user.Username != "newuser" {
				t.Errorf("Create username: got=%s, want=%s", user.Username, "newuser")
			}
			created := *user
			created.ID = userID
			created.CreatedAt = time.Now()
			created.UpdatedAt = time.Now()
			return &created, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		CreateFunc: func(ctx context.Context, am *domain.AuthMethod) (*domain.AuthMethod, error) {
			if am.Method != domain.AuthMethodPassword {
				t.Errorf("authMethods.Create method: got=%s, want=%s", am.Method, domain.AuthMethodPassword)
			}
			if am.PasswordHash == nil || *am.PasswordHash == "" {
				t.Error("authMethods.Create: PasswordHash should be set")
			}
			created := *am
			created.ID = uuid.New()
			return &created, nil
		},
	}

	settingsMock := &settingsRepoMock{
		CreateSettingsFunc: func(ctx context.Context, settings *domain.UserSettings) error {
			if settings.UserID != userID {
				t.Errorf("CreateSettings userID: got=%s, want=%s", settings.UserID, userID)
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, settingsMock, tokensMock, authMethodsMock,
		txMock, &oauthVerifierMock{}, jwtMock, cfg,
	)

	input := RegisterInput{
		Email:    "new@example.com",
		Username: "newuser",
		Password: "password123",
	}

	result, err := svc.Register(ctx, input)

	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Register returned nil result")
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}
	if result.AccessToken != "access_token_123" {
		t.Errorf("AccessToken: got=%s, want=%s", result.AccessToken, "access_token_123")
	}

	if len(usersMock.CreateCalls()) != 1 {
		t.Errorf("users.Create called %d times, want 1", len(usersMock.CreateCalls()))
	}
	if len(authMethodsMock.CreateCalls()) != 1 {
		t.Errorf("authMethods.Create called %d times, want 1", len(authMethodsMock.CreateCalls()))
	}
	if len(settingsMock.CreateSettingsCalls()) != 1 {
		t.Errorf("CreateSettings called %d times, want 1", len(settingsMock.CreateSettingsCalls()))
	}
}

func TestService_Register_EmailAlreadyTaken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	usersMock := &userRepoMock{
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
			return nil, domain.ErrAlreadyExists
		},
	}

	txMock := &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, &tokenRepoMock{},
		&authMethodRepoMock{}, txMock, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.Register(ctx, RegisterInput{
		Email:    "taken@example.com",
		Username: "newuser",
		Password: "password123",
	})

	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("Register error: got=%v, want=ErrAlreadyExists", err)
	}
	if result != nil {
		t.Fatal("Register should return nil result when email is taken")
	}
}

func TestService_Register_ValidationErrors(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{},
		&authMethodRepoMock{}, &txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	tests := []struct {
		name      string
		input     RegisterInput
		wantField string
		wantMsg   string
	}{
		{
			name:      "empty email",
			input:     RegisterInput{Email: "", Username: "user", Password: "password123"},
			wantField: "email",
			wantMsg:   "required",
		},
		{
			name:      "invalid email",
			input:     RegisterInput{Email: "notanemail", Username: "user", Password: "password123"},
			wantField: "email",
			wantMsg:   "invalid email",
		},
		{
			name:      "empty username",
			input:     RegisterInput{Email: "a@b.com", Username: "", Password: "password123"},
			wantField: "username",
			wantMsg:   "required",
		},
		{
			name:      "username too short",
			input:     RegisterInput{Email: "a@b.com", Username: "a", Password: "password123"},
			wantField: "username",
			wantMsg:   "must be between 2 and 50 characters",
		},
		{
			name:      "empty password",
			input:     RegisterInput{Email: "a@b.com", Username: "user", Password: ""},
			wantField: "password",
			wantMsg:   "required",
		},
		{
			name:      "password too short",
			input:     RegisterInput{Email: "a@b.com", Username: "user", Password: "short"},
			wantField: "password",
			wantMsg:   "must be at least 8 characters",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := svc.Register(context.Background(), tt.input)
			if result != nil {
				t.Error("Register should return nil result on validation error")
			}

			var valErr *domain.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("Register error: got=%v, want=ValidationError", err)
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

// ─── Password Login Tests ───────────────────────────────────────────────────

func TestService_LoginWithPassword_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	password := "correct_password"
	passHash := hashPassword(t, password)

	existingUser := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "testuser",
		Name:     "Test User",
	}

	existingAM := &domain.AuthMethod{
		ID:           uuid.New(),
		UserID:       userID,
		Method:       domain.AuthMethodPassword,
		PasswordHash: &passHash,
	}

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			if email != "test@example.com" {
				t.Errorf("GetByEmail email: got=%s, want=%s", email, "test@example.com")
			}
			return existingUser, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByUserAndMethodFunc: func(ctx context.Context, uid uuid.UUID, method domain.AuthMethodType) (*domain.AuthMethod, error) {
			if uid != userID {
				t.Errorf("GetByUserAndMethod userID: got=%s, want=%s", uid, userID)
			}
			if method != domain.AuthMethodPassword {
				t.Errorf("GetByUserAndMethod method: got=%s, want=%s", method, domain.AuthMethodPassword)
			}
			return existingAM, nil
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

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, authMethodsMock,
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	result, err := svc.LoginWithPassword(ctx, LoginPasswordInput{
		Email:    "test@example.com",
		Password: password,
	})

	if err != nil {
		t.Fatalf("LoginWithPassword returned error: %v", err)
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}
	if result.AccessToken != "access_token_123" {
		t.Errorf("AccessToken: got=%s, want=%s", result.AccessToken, "access_token_123")
	}
}

func TestService_LoginWithPassword_UserNotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, &tokenRepoMock{},
		&authMethodRepoMock{}, &txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.LoginWithPassword(ctx, LoginPasswordInput{
		Email:    "nobody@example.com",
		Password: "password123",
	})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("LoginWithPassword error: got=%v, want=ErrUnauthorized", err)
	}
	if result != nil {
		t.Fatal("LoginWithPassword should return nil result when user not found")
	}
}

func TestService_LoginWithPassword_NoPasswordMethod(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: email, Username: "oauthuser"}, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByUserAndMethodFunc: func(ctx context.Context, uid uuid.UUID, method domain.AuthMethodType) (*domain.AuthMethod, error) {
			return nil, domain.ErrNotFound // OAuth-only user
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, &tokenRepoMock{},
		authMethodsMock, &txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.LoginWithPassword(ctx, LoginPasswordInput{
		Email:    "oauth@example.com",
		Password: "password123",
	})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("LoginWithPassword error: got=%v, want=ErrUnauthorized", err)
	}
	if result != nil {
		t.Fatal("LoginWithPassword should return nil result for OAuth-only user")
	}
}

func TestService_LoginWithPassword_WrongPassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	correctHash := hashPassword(t, "correct_password")

	usersMock := &userRepoMock{
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
			return &domain.User{ID: userID, Email: email, Username: "testuser"}, nil
		},
	}

	authMethodsMock := &authMethodRepoMock{
		GetByUserAndMethodFunc: func(ctx context.Context, uid uuid.UUID, method domain.AuthMethodType) (*domain.AuthMethod, error) {
			return &domain.AuthMethod{
				ID:           uuid.New(),
				UserID:       uid,
				Method:       domain.AuthMethodPassword,
				PasswordHash: &correctHash,
			}, nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, &tokenRepoMock{},
		authMethodsMock, &txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.LoginWithPassword(ctx, LoginPasswordInput{
		Email:    "test@example.com",
		Password: "wrong_password",
	})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("LoginWithPassword error: got=%v, want=ErrUnauthorized", err)
	}
	if result != nil {
		t.Fatal("LoginWithPassword should return nil result on wrong password")
	}
}

func TestService_LoginWithPassword_ValidationErrors(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{},
		&authMethodRepoMock{}, &txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	tests := []struct {
		name      string
		input     LoginPasswordInput
		wantField string
		wantMsg   string
	}{
		{
			name:      "empty email",
			input:     LoginPasswordInput{Email: "", Password: "password123"},
			wantField: "email",
			wantMsg:   "required",
		},
		{
			name:      "empty password",
			input:     LoginPasswordInput{Email: "a@b.com", Password: ""},
			wantField: "password",
			wantMsg:   "required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := svc.LoginWithPassword(context.Background(), tt.input)
			if result != nil {
				t.Error("LoginWithPassword should return nil result on validation error")
			}

			var valErr *domain.ValidationError
			if !errors.As(err, &valErr) {
				t.Fatalf("error: got=%v, want=ValidationError", err)
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

// ─── Refresh Tests ──────────────────────────────────────────────────────────

func TestService_Refresh_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	tokenID := uuid.New()
	oldRefreshRaw := "old_refresh_raw"
	oldRefreshHash := auth.HashToken(oldRefreshRaw)

	existingToken := &domain.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: oldRefreshHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	existingUser := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "test",
		Name:     "Test User",
	}

	oldTokenRevoked := false
	newTokenCreated := false

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			if hash != oldRefreshHash {
				t.Errorf("GetByHash called with wrong hash: got=%s, want=%s", hash, oldRefreshHash)
			}
			return existingToken, nil
		},
		RevokeByIDFunc: func(ctx context.Context, id uuid.UUID) error {
			if id != tokenID {
				t.Errorf("RevokeByID called with wrong ID: got=%s, want=%s", id, tokenID)
			}
			oldTokenRevoked = true
			return nil
		},
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			if token.UserID != userID {
				t.Errorf("tokens.Create: UserID: got=%s, want=%s", token.UserID, userID)
			}
			if token.TokenHash == oldRefreshHash {
				t.Errorf("tokens.Create: TokenHash should be different from old hash")
			}
			newTokenCreated = true
			return nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			if id != userID {
				t.Errorf("GetByID called with wrong ID: got=%s, want=%s", id, userID)
			}
			return existingUser, nil
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "new_access_token", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "new_refresh_raw", "new_refresh_hash", nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	result, err := svc.Refresh(ctx, RefreshInput{RefreshToken: oldRefreshRaw})

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if result.AccessToken != "new_access_token" {
		t.Errorf("AccessToken: got=%s, want=%s", result.AccessToken, "new_access_token")
	}
	if result.RefreshToken != "new_refresh_raw" {
		t.Errorf("RefreshToken: got=%s, want=%s", result.RefreshToken, "new_refresh_raw")
	}
	if result.User.ID != userID {
		t.Errorf("User.ID: got=%s, want=%s", result.User.ID, userID)
	}
	if !oldTokenRevoked {
		t.Error("Old token was not revoked")
	}
	if !newTokenCreated {
		t.Error("New token was not created")
	}
}

func TestService_Refresh_TokenNotFound(t *testing.T) {
	t.Parallel()

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			return nil, domain.ErrNotFound
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: "invalid_token"})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Refresh error: got=%v, want=ErrUnauthorized", err)
	}
	if result != nil {
		t.Fatal("Refresh should return nil result on token not found")
	}
}

func TestService_Refresh_TokenExpired(t *testing.T) {
	t.Parallel()

	expiredToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "some_hash",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			return expiredToken, nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: "expired_token"})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Refresh error: got=%v, want=ErrUnauthorized", err)
	}
	if result != nil {
		t.Fatal("Refresh should return nil result on expired token")
	}
}

func TestService_Refresh_UserDeleted(t *testing.T) {
	t.Parallel()

	validToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		TokenHash: "some_hash",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			return validToken, nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: "valid_token_deleted_user"})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Refresh error: got=%v, want=ErrUnauthorized", err)
	}
	if result != nil {
		t.Fatal("Refresh should return nil result when user is deleted")
	}
}

func TestService_Refresh_ValidationEmptyToken(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{}, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	result, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: ""})

	if result != nil {
		t.Error("Refresh should return nil result on validation error")
	}

	var valErr *domain.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("Refresh error: got=%v, want=ValidationError", err)
	}

	found := false
	for _, fieldErr := range valErr.Errors {
		if fieldErr.Field == "refresh_token" && fieldErr.Message == "required" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ValidationError missing: field=refresh_token, message=required. Got: %v", valErr.Errors)
	}
}

func TestService_Refresh_ValidationTooLong(t *testing.T) {
	t.Parallel()

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{}, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	longToken := string(make([]byte, 513))
	result, err := svc.Refresh(context.Background(), RefreshInput{RefreshToken: longToken})

	if result != nil {
		t.Error("Refresh should return nil result on validation error")
	}

	var valErr *domain.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("Refresh error: got=%v, want=ValidationError", err)
	}

	found := false
	for _, fieldErr := range valErr.Errors {
		if fieldErr.Field == "refresh_token" && fieldErr.Message == "too long" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ValidationError missing: field=refresh_token, message=too long. Got: %v", valErr.Errors)
	}
}

func TestService_Refresh_OldTokenRevoked(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	tokenID := uuid.New()
	oldRefreshRaw := "old_refresh_raw"

	existingToken := &domain.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: auth.HashToken(oldRefreshRaw),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	existingUser := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "test",
		Name:     "Test User",
	}

	revokeCalledWithID := uuid.Nil

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			return existingToken, nil
		},
		RevokeByIDFunc: func(ctx context.Context, id uuid.UUID) error {
			revokeCalledWithID = id
			return nil
		},
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "new_access_token", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "new_refresh_raw", "new_refresh_hash", nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	_, err := svc.Refresh(ctx, RefreshInput{RefreshToken: oldRefreshRaw})

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if revokeCalledWithID != tokenID {
		t.Errorf("RevokeByID called with ID: got=%s, want=%s", revokeCalledWithID, tokenID)
	}
}

func TestService_Refresh_NewTokenDifferentFromOld(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	tokenID := uuid.New()
	oldRefreshRaw := "old_refresh_raw"
	oldRefreshHash := auth.HashToken(oldRefreshRaw)

	existingToken := &domain.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: oldRefreshHash,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	existingUser := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "test",
		Name:     "Test User",
	}

	var createdTokenHash string

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			return existingToken, nil
		},
		RevokeByIDFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			createdTokenHash = token.TokenHash
			return nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "new_access_token", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "new_refresh_raw", "new_refresh_hash", nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	result, err := svc.Refresh(ctx, RefreshInput{RefreshToken: oldRefreshRaw})

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if result.RefreshToken == oldRefreshRaw {
		t.Error("New refresh token should be different from old")
	}
	if createdTokenHash == oldRefreshHash {
		t.Error("New token hash should be different from old token hash")
	}
	if createdTokenHash != "new_refresh_hash" {
		t.Errorf("Stored token hash: got=%s, want=%s", createdTokenHash, "new_refresh_hash")
	}
}

func TestService_Refresh_UserDataInResponse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	tokenID := uuid.New()
	oldRefreshRaw := "old_refresh_raw"

	existingToken := &domain.RefreshToken{
		ID:        tokenID,
		UserID:    userID,
		TokenHash: auth.HashToken(oldRefreshRaw),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now().Add(-1 * time.Hour),
	}

	existingUser := &domain.User{
		ID:        userID,
		Email:     "john@example.com",
		Username:  "johndoe",
		Name:      "John Doe",
		AvatarURL: ptrString("https://example.com/john.jpg"),
	}

	tokensMock := &tokenRepoMock{
		GetByHashFunc: func(ctx context.Context, hash string) (*domain.RefreshToken, error) {
			return existingToken, nil
		},
		RevokeByIDFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
		CreateFunc: func(ctx context.Context, token *domain.RefreshToken) error {
			return nil
		},
	}

	usersMock := &userRepoMock{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*domain.User, error) {
			return existingUser, nil
		},
	}

	jwtMock := &jwtManagerMock{
		GenerateAccessTokenFunc: func(uid uuid.UUID) (string, error) {
			return "new_access_token", nil
		},
		GenerateRefreshTokenFunc: func() (string, string, error) {
			return "new_refresh_raw", "new_refresh_hash", nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), usersMock, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	result, err := svc.Refresh(ctx, RefreshInput{RefreshToken: oldRefreshRaw})

	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if result.User.Name != "John Doe" {
		t.Errorf("User.Name: got=%s, want=%s", result.User.Name, "John Doe")
	}
	if result.User.Email != "john@example.com" {
		t.Errorf("User.Email: got=%s, want=%s", result.User.Email, "john@example.com")
	}
}

// ─── Logout Tests ───────────────────────────────────────────────────────────

func TestService_Logout_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	revokedForUserID := uuid.Nil

	tokensMock := &tokenRepoMock{
		RevokeAllByUserFunc: func(ctx context.Context, uid uuid.UUID) error {
			revokedForUserID = uid
			return nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	err := svc.Logout(ctx)

	if err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if revokedForUserID != userID {
		t.Errorf("RevokeAllByUser called with userID: got=%s, want=%s", revokedForUserID, userID)
	}
	if len(tokensMock.RevokeAllByUserCalls()) != 1 {
		t.Errorf("RevokeAllByUser called %d times, want 1", len(tokensMock.RevokeAllByUserCalls()))
	}
}

func TestService_Logout_NoUserID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tokensMock := &tokenRepoMock{}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	err := svc.Logout(ctx)

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("Logout error: got=%v, want=ErrUnauthorized", err)
	}
	if len(tokensMock.RevokeAllByUserCalls()) != 0 {
		t.Errorf("RevokeAllByUser called %d times, want 0", len(tokensMock.RevokeAllByUserCalls()))
	}
}

func TestService_Logout_RevokeError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	revokeErr := errors.New("database error")

	tokensMock := &tokenRepoMock{
		RevokeAllByUserFunc: func(ctx context.Context, uid uuid.UUID) error {
			return revokeErr
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	err := svc.Logout(ctx)

	if err == nil {
		t.Fatal("Logout should return error when revoke fails")
	}
	if !errors.Is(err, revokeErr) {
		t.Errorf("Logout error should wrap revoke error: got=%v, want=%v", err, revokeErr)
	}
}

// ─── ValidateToken Tests ────────────────────────────────────────────────────

func TestService_ValidateToken_ValidToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	userID := uuid.New()
	token := "valid_access_token"

	jwtMock := &jwtManagerMock{
		ValidateAccessTokenFunc: func(t string) (uuid.UUID, error) {
			if t != token {
				return uuid.Nil, errors.New("invalid token")
			}
			return userID, nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{}, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	resultUserID, err := svc.ValidateToken(ctx, token)

	if err != nil {
		t.Fatalf("ValidateToken returned error: %v", err)
	}
	if resultUserID != userID {
		t.Errorf("ValidateToken userID: got=%s, want=%s", resultUserID, userID)
	}
}

func TestService_ValidateToken_InvalidToken(t *testing.T) {
	t.Parallel()

	jwtMock := &jwtManagerMock{
		ValidateAccessTokenFunc: func(t string) (uuid.UUID, error) {
			return uuid.Nil, errors.New("jwt validation failed")
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{}, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	resultUserID, err := svc.ValidateToken(context.Background(), "invalid_token")

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ValidateToken error: got=%v, want=ErrUnauthorized", err)
	}
	if resultUserID != uuid.Nil {
		t.Errorf("ValidateToken should return uuid.Nil for invalid token, got=%s", resultUserID)
	}
}

func TestService_ValidateToken_MalformedToken(t *testing.T) {
	t.Parallel()

	jwtMock := &jwtManagerMock{
		ValidateAccessTokenFunc: func(t string) (uuid.UUID, error) {
			return uuid.Nil, errors.New("malformed token")
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, &tokenRepoMock{}, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, jwtMock, cfg,
	)

	resultUserID, err := svc.ValidateToken(context.Background(), "malformed.jwt.token")

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ValidateToken error: got=%v, want=ErrUnauthorized", err)
	}
	if resultUserID != uuid.Nil {
		t.Errorf("ValidateToken should return uuid.Nil for malformed token, got=%s", resultUserID)
	}
}

// ─── CleanupExpiredTokens Tests ─────────────────────────────────────────────

func TestService_CleanupExpiredTokens_TokensDeleted(t *testing.T) {
	t.Parallel()

	tokensMock := &tokenRepoMock{
		DeleteExpiredFunc: func(ctx context.Context) (int, error) {
			return 5, nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	count, err := svc.CleanupExpiredTokens(context.Background())

	if err != nil {
		t.Fatalf("CleanupExpiredTokens returned error: %v", err)
	}
	if count != 5 {
		t.Errorf("CleanupExpiredTokens count: got=%d, want=%d", count, 5)
	}
	if len(tokensMock.DeleteExpiredCalls()) != 1 {
		t.Errorf("DeleteExpired called %d times, want 1", len(tokensMock.DeleteExpiredCalls()))
	}
}

func TestService_CleanupExpiredTokens_NoTokensDeleted(t *testing.T) {
	t.Parallel()

	tokensMock := &tokenRepoMock{
		DeleteExpiredFunc: func(ctx context.Context) (int, error) {
			return 0, nil
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	count, err := svc.CleanupExpiredTokens(context.Background())

	if err != nil {
		t.Fatalf("CleanupExpiredTokens returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("CleanupExpiredTokens count: got=%d, want=%d", count, 0)
	}
}

func TestService_CleanupExpiredTokens_Error(t *testing.T) {
	t.Parallel()

	deleteErr := errors.New("database error")

	tokensMock := &tokenRepoMock{
		DeleteExpiredFunc: func(ctx context.Context) (int, error) {
			return 0, deleteErr
		},
	}

	cfg := defaultCfg()

	svc := NewService(
		slog.Default(), &userRepoMock{}, &settingsRepoMock{}, tokensMock, &authMethodRepoMock{},
		&txManagerMock{}, &oauthVerifierMock{}, &jwtManagerMock{}, cfg,
	)

	count, err := svc.CleanupExpiredTokens(context.Background())

	if err == nil {
		t.Fatal("CleanupExpiredTokens should return error when delete fails")
	}
	if !errors.Is(err, deleteErr) {
		t.Errorf("CleanupExpiredTokens error should wrap delete error: got=%v, want=%v", err, deleteErr)
	}
	if count != 0 {
		t.Errorf("CleanupExpiredTokens count: got=%d, want=%d (should be 0 on error)", count, 0)
	}
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func ptrString(s string) *string {
	return &s
}
