package user_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/user"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo is a test helper that sets up the DB and returns a ready Repo.
func newRepo(t *testing.T) (*user.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return user.New(pool), pool
}

// ---------------------------------------------------------------------------
// User CRUD
// ---------------------------------------------------------------------------

func TestRepo_Create_HappyPath(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	u := domain.User{
		ID:            uuid.New(),
		Email:         "create-happy-" + uuid.New().String()[:8] + "@example.com",
		Name:          "Happy User",
		AvatarURL:     ptrStr("https://example.com/avatar.png"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "google-" + uuid.New().String()[:8],
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	got, err := repo.Create(ctx, u)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	assertUserEqual(t, u, got)
}

func TestRepo_Create_DuplicateEmail(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	email := "dup-email-" + uuid.New().String()[:8] + "@example.com"
	now := time.Now().UTC().Truncate(time.Microsecond)

	u1 := domain.User{
		ID:            uuid.New(),
		Email:         email,
		Name:          "User 1",
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "oauth-dup-email-1-" + uuid.New().String()[:8],
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := repo.Create(ctx, u1); err != nil {
		t.Fatalf("Create first user: %v", err)
	}

	u2 := domain.User{
		ID:            uuid.New(),
		Email:         email, // same email
		Name:          "User 2",
		OAuthProvider: domain.OAuthProviderApple,
		OAuthID:       "oauth-dup-email-2-" + uuid.New().String()[:8],
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err := repo.Create(ctx, u2)
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

func TestRepo_Create_DuplicateOAuth(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	provider := domain.OAuthProviderGoogle
	oauthID := "oauth-dup-" + uuid.New().String()[:8]
	now := time.Now().UTC().Truncate(time.Microsecond)

	u1 := domain.User{
		ID:            uuid.New(),
		Email:         "dup-oauth-1-" + uuid.New().String()[:8] + "@example.com",
		Name:          "User 1",
		OAuthProvider: provider,
		OAuthID:       oauthID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := repo.Create(ctx, u1); err != nil {
		t.Fatalf("Create first user: %v", err)
	}

	u2 := domain.User{
		ID:            uuid.New(),
		Email:         "dup-oauth-2-" + uuid.New().String()[:8] + "@example.com",
		Name:          "User 2",
		OAuthProvider: provider,
		OAuthID:       oauthID, // same OAuth
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err := repo.Create(ctx, u2)
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

func TestRepo_GetByID_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedUser(t, pool)

	got, err := repo.GetByID(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != seeded.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, seeded.ID)
	}
	if got.Email != seeded.Email {
		t.Errorf("Email mismatch: got %s, want %s", got.Email, seeded.Email)
	}
}

func TestRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByOAuth_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedUser(t, pool)

	got, err := repo.GetByOAuth(ctx, seeded.OAuthProvider, seeded.OAuthID)
	if err != nil {
		t.Fatalf("GetByOAuth: unexpected error: %v", err)
	}

	if got.ID != seeded.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, seeded.ID)
	}
}

func TestRepo_GetByOAuth_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetByOAuth(ctx, domain.OAuthProviderGoogle, "nonexistent-oauth-id")
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByEmail_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedUser(t, pool)

	got, err := repo.GetByEmail(ctx, seeded.Email)
	if err != nil {
		t.Fatalf("GetByEmail: unexpected error: %v", err)
	}

	if got.ID != seeded.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, seeded.ID)
	}
}

func TestRepo_GetByEmail_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetByEmail(ctx, "nonexistent-"+uuid.New().String()[:8]+"@example.com")
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Update_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedUser(t, pool)

	newName := "Updated Name"
	newAvatar := "https://example.com/new-avatar.png"

	got, err := repo.Update(ctx, seeded.ID, newName, &newAvatar)
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}

	if got.Name != newName {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, newName)
	}
	if got.AvatarURL == nil || *got.AvatarURL != newAvatar {
		t.Errorf("AvatarURL mismatch: got %v, want %q", got.AvatarURL, newAvatar)
	}
	if !got.UpdatedAt.After(seeded.UpdatedAt) {
		t.Errorf("UpdatedAt should be newer: got %v, seeded %v", got.UpdatedAt, seeded.UpdatedAt)
	}
}

func TestRepo_Update_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.Update(ctx, uuid.New(), "name", nil)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Update_ClearAvatar(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)
	u := domain.User{
		ID:            uuid.New(),
		Email:         "clear-avatar-" + uuid.New().String()[:8] + "@example.com",
		Name:          "With Avatar",
		AvatarURL:     ptrStr("https://example.com/old.png"),
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "oauth-clear-" + uuid.New().String()[:8],
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Update(ctx, u.ID, "With Avatar", nil)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if got.AvatarURL != nil {
		t.Errorf("AvatarURL should be nil after clearing, got %v", *got.AvatarURL)
	}
}

// ---------------------------------------------------------------------------
// UserSettings CRUD
// ---------------------------------------------------------------------------

func TestRepo_CreateSettings_HappyPath(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	// Create a user first (without using SeedUser to avoid auto settings creation).
	now := time.Now().UTC().Truncate(time.Microsecond)
	u := domain.User{
		ID:            uuid.New(),
		Email:         "settings-create-" + uuid.New().String()[:8] + "@example.com",
		Name:          "Settings User",
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "oauth-settings-" + uuid.New().String()[:8],
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if _, err := repo.Create(ctx, u); err != nil {
		t.Fatalf("Create user: %v", err)
	}

	s := domain.UserSettings{
		UserID:          u.ID,
		NewCardsPerDay:  30,
		ReviewsPerDay:   150,
		MaxIntervalDays: 180,
		Timezone:        "Europe/Moscow",
	}

	got, err := repo.CreateSettings(ctx, s)
	if err != nil {
		t.Fatalf("CreateSettings: unexpected error: %v", err)
	}

	if got.UserID != s.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, s.UserID)
	}
	if got.NewCardsPerDay != s.NewCardsPerDay {
		t.Errorf("NewCardsPerDay mismatch: got %d, want %d", got.NewCardsPerDay, s.NewCardsPerDay)
	}
	if got.ReviewsPerDay != s.ReviewsPerDay {
		t.Errorf("ReviewsPerDay mismatch: got %d, want %d", got.ReviewsPerDay, s.ReviewsPerDay)
	}
	if got.MaxIntervalDays != s.MaxIntervalDays {
		t.Errorf("MaxIntervalDays mismatch: got %d, want %d", got.MaxIntervalDays, s.MaxIntervalDays)
	}
	if got.Timezone != s.Timezone {
		t.Errorf("Timezone mismatch: got %s, want %s", got.Timezone, s.Timezone)
	}
}

func TestRepo_CreateSettings_DuplicateUserID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	// SeedUser already creates settings, so creating again should conflict.
	seeded := testhelper.SeedUser(t, pool)

	s := domain.DefaultUserSettings(seeded.ID)
	_, err := repo.CreateSettings(ctx, s)
	assertIsDomainError(t, err, domain.ErrAlreadyExists)
}

func TestRepo_GetSettings_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedUser(t, pool)

	got, err := repo.GetSettings(ctx, seeded.ID)
	if err != nil {
		t.Fatalf("GetSettings: unexpected error: %v", err)
	}

	defaults := domain.DefaultUserSettings(seeded.ID)
	if got.NewCardsPerDay != defaults.NewCardsPerDay {
		t.Errorf("NewCardsPerDay mismatch: got %d, want %d", got.NewCardsPerDay, defaults.NewCardsPerDay)
	}
	if got.ReviewsPerDay != defaults.ReviewsPerDay {
		t.Errorf("ReviewsPerDay mismatch: got %d, want %d", got.ReviewsPerDay, defaults.ReviewsPerDay)
	}
}

func TestRepo_GetSettings_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.GetSettings(ctx, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_UpdateSettings_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	seeded := testhelper.SeedUser(t, pool)

	updated := domain.UserSettings{
		NewCardsPerDay:  50,
		ReviewsPerDay:   300,
		MaxIntervalDays: 730,
		Timezone:        "America/New_York",
	}

	got, err := repo.UpdateSettings(ctx, seeded.ID, updated)
	if err != nil {
		t.Fatalf("UpdateSettings: unexpected error: %v", err)
	}

	if got.NewCardsPerDay != updated.NewCardsPerDay {
		t.Errorf("NewCardsPerDay mismatch: got %d, want %d", got.NewCardsPerDay, updated.NewCardsPerDay)
	}
	if got.ReviewsPerDay != updated.ReviewsPerDay {
		t.Errorf("ReviewsPerDay mismatch: got %d, want %d", got.ReviewsPerDay, updated.ReviewsPerDay)
	}
	if got.MaxIntervalDays != updated.MaxIntervalDays {
		t.Errorf("MaxIntervalDays mismatch: got %d, want %d", got.MaxIntervalDays, updated.MaxIntervalDays)
	}
	if got.Timezone != updated.Timezone {
		t.Errorf("Timezone mismatch: got %s, want %s", got.Timezone, updated.Timezone)
	}
}

func TestRepo_UpdateSettings_NotFound(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	_, err := repo.UpdateSettings(ctx, uuid.New(), domain.DefaultUserSettings(uuid.New()))
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// OAuthProvider mapping
// ---------------------------------------------------------------------------

func TestRepo_OAuthProviderMapping(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Microsecond)

	tests := []struct {
		name     string
		provider domain.OAuthProvider
	}{
		{"google", domain.OAuthProviderGoogle},
		{"apple", domain.OAuthProviderApple},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := domain.User{
				ID:            uuid.New(),
				Email:         "oauth-map-" + tt.name + "-" + uuid.New().String()[:8] + "@example.com",
				Name:          "OAuth " + tt.name,
				OAuthProvider: tt.provider,
				OAuthID:       "oauth-map-" + tt.name + "-" + uuid.New().String()[:8],
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			created, err := repo.Create(ctx, u)
			if err != nil {
				t.Fatalf("Create: %v", err)
			}

			if created.OAuthProvider != tt.provider {
				t.Errorf("OAuthProvider mismatch: got %s, want %s", created.OAuthProvider, tt.provider)
			}

			// Also verify via GetByOAuth.
			got, err := repo.GetByOAuth(ctx, tt.provider, u.OAuthID)
			if err != nil {
				t.Fatalf("GetByOAuth: %v", err)
			}
			if got.OAuthProvider != tt.provider {
				t.Errorf("GetByOAuth OAuthProvider mismatch: got %s, want %s", got.OAuthProvider, tt.provider)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func ptrStr(s string) *string {
	return &s
}

func assertUserEqual(t *testing.T, want, got domain.User) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, want.ID)
	}
	if got.Email != want.Email {
		t.Errorf("Email mismatch: got %s, want %s", got.Email, want.Email)
	}
	if got.Name != want.Name {
		t.Errorf("Name mismatch: got %q, want %q", got.Name, want.Name)
	}
	if (got.AvatarURL == nil) != (want.AvatarURL == nil) {
		t.Errorf("AvatarURL nil mismatch: got %v, want %v", got.AvatarURL, want.AvatarURL)
	} else if got.AvatarURL != nil && *got.AvatarURL != *want.AvatarURL {
		t.Errorf("AvatarURL mismatch: got %s, want %s", *got.AvatarURL, *want.AvatarURL)
	}
	if got.OAuthProvider != want.OAuthProvider {
		t.Errorf("OAuthProvider mismatch: got %s, want %s", got.OAuthProvider, want.OAuthProvider)
	}
	if got.OAuthID != want.OAuthID {
		t.Errorf("OAuthID mismatch: got %s, want %s", got.OAuthID, want.OAuthID)
	}
}

func assertIsDomainError(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error wrapping %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error wrapping %v, got: %v", target, err)
	}
}
