package content

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// Mock for image operations
// ---------------------------------------------------------------------------

type mockImageRepo struct {
	getUserByIDForUserFunc func(ctx context.Context, userID, imageID uuid.UUID) (*domain.UserImage, error)
	countUserByEntryFunc   func(ctx context.Context, entryID uuid.UUID) (int, error)
	createUserFunc         func(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error)
	updateUserFunc         func(ctx context.Context, imageID uuid.UUID, caption *string) (*domain.UserImage, error)
	deleteUserFunc         func(ctx context.Context, imageID uuid.UUID) error
}

func (m *mockImageRepo) GetUserByIDForUser(ctx context.Context, userID, imageID uuid.UUID) (*domain.UserImage, error) {
	if m.getUserByIDForUserFunc != nil {
		return m.getUserByIDForUserFunc(ctx, userID, imageID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockImageRepo) CountUserByEntry(ctx context.Context, entryID uuid.UUID) (int, error) {
	if m.countUserByEntryFunc != nil {
		return m.countUserByEntryFunc(ctx, entryID)
	}
	return 0, nil
}

func (m *mockImageRepo) CreateUser(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, entryID, url, caption)
	}
	return &domain.UserImage{ID: uuid.New(), EntryID: entryID, URL: url, Caption: caption}, nil
}

func (m *mockImageRepo) UpdateUser(ctx context.Context, imageID uuid.UUID, caption *string) (*domain.UserImage, error) {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, imageID, caption)
	}
	return &domain.UserImage{ID: imageID, Caption: caption}, nil
}

func (m *mockImageRepo) DeleteUser(ctx context.Context, imageID uuid.UUID) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, imageID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AddUserImage Tests
// ---------------------------------------------------------------------------

func TestService_AddUserImage_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	imageID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			if uid != userID || eid != entryID {
				return nil, domain.ErrNotFound
			}
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	imageRepo := &mockImageRepo{
		createUserFunc: func(ctx context.Context, eid uuid.UUID, url string, caption *string) (*domain.UserImage, error) {
			if eid != entryID {
				t.Errorf("unexpected entryID: got %v, want %v", eid, entryID)
			}
			return &domain.UserImage{ID: imageID, EntryID: eid, URL: url, Caption: caption}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, auditRepo, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := AddUserImageInput{
		EntryID: entryID,
		URL:     "https://example.com/image.jpg",
		Caption: nil,
	}

	img, err := svc.AddUserImage(ctx, input)
	if err != nil {
		t.Fatalf("AddUserImage failed: %v", err)
	}

	if img == nil {
		t.Fatal("expected image, got nil")
	}
	if img.ID != imageID {
		t.Errorf("unexpected image ID: got %v, want %v", img.ID, imageID)
	}
	if img.EntryID != entryID {
		t.Errorf("unexpected entry ID: got %v, want %v", img.EntryID, entryID)
	}
	if img.URL != input.URL {
		t.Errorf("unexpected URL: got %v, want %v", img.URL, input.URL)
	}

	// Verify audit
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	audit := auditRepo.records[0]
	if audit.EntityType != domain.EntityTypeEntry {
		t.Errorf("expected EntityType ENTRY, got %v", audit.EntityType)
	}
	if *audit.EntityID != entryID {
		t.Errorf("expected EntityID %v, got %v", entryID, *audit.EntityID)
	}
}

func TestService_AddUserImage_EntryNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, entryRepo, nil, nil, nil, &mockImageRepo{}, &mockAuditRepo{}, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := AddUserImageInput{
		EntryID: entryID,
		URL:     "https://example.com/image.jpg",
	}

	_, err := svc.AddUserImage(ctx, input)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_AddUserImage_InvalidURL(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()

	svc := NewService(logger, nil, nil, nil, nil, nil, nil, nil)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	testCases := []struct {
		name string
		url  string
	}{
		{"empty URL", ""},
		{"not HTTP(S)", "ftp://example.com/image.jpg"},
		{"no scheme", "example.com/image.jpg"},
		{"too long", "https://example.com/" + string(make([]byte, 2000))},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := AddUserImageInput{
				EntryID: entryID,
				URL:     tc.url,
			}

			_, err := svc.AddUserImage(ctx, input)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}

			if !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestService_AddUserImage_WithCaption(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	imageID := uuid.New()
	caption := "Test caption"

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			if uid != userID || eid != entryID {
				return nil, domain.ErrNotFound
			}
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	imageRepo := &mockImageRepo{
		createUserFunc: func(ctx context.Context, eid uuid.UUID, url string, cap *string) (*domain.UserImage, error) {
			if cap == nil {
				t.Error("expected caption to be set")
			} else if *cap != caption {
				t.Errorf("unexpected caption: got %v, want %v", *cap, caption)
			}
			return &domain.UserImage{ID: imageID, EntryID: eid, URL: url, Caption: cap}, nil
		},
	}

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, &mockAuditRepo{}, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := AddUserImageInput{
		EntryID: entryID,
		URL:     "https://example.com/image.jpg",
		Caption: &caption,
	}

	img, err := svc.AddUserImage(ctx, input)
	if err != nil {
		t.Fatalf("AddUserImage failed: %v", err)
	}

	if img.Caption == nil {
		t.Fatal("expected caption, got nil")
	}

	if *img.Caption != caption {
		t.Errorf("unexpected caption: got %v, want %v", *img.Caption, caption)
	}
}

func TestService_AddUserImage_LimitReached(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	imageRepo := &mockImageRepo{
		countUserByEntryFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return MaxUserImagesPerEntry, nil // Limit reached
		},
	}

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, &mockAuditRepo{}, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := AddUserImageInput{
		EntryID: entryID,
		URL:     "https://example.com/image.jpg",
	}

	_, err := svc.AddUserImage(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for limit reached, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateUserImage Tests
// ---------------------------------------------------------------------------

func TestService_UpdateUserImage_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	imageID := uuid.New()
	oldCaption := "old caption"
	newCaption := "new caption"

	imageRepo := &mockImageRepo{
		getUserByIDForUserFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.UserImage, error) {
			if uid != userID || iid != imageID {
				return nil, domain.ErrNotFound
			}
			return &domain.UserImage{ID: imageID, EntryID: entryID, URL: "https://example.com/img.jpg", Caption: &oldCaption}, nil
		},
		updateUserFunc: func(ctx context.Context, iid uuid.UUID, cap *string) (*domain.UserImage, error) {
			return &domain.UserImage{ID: iid, EntryID: entryID, URL: "https://example.com/img.jpg", Caption: cap}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, nil, nil, nil, nil, imageRepo, auditRepo, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateUserImageInput{
		ImageID: imageID,
		Caption: &newCaption,
	}

	img, err := svc.UpdateUserImage(ctx, input)
	if err != nil {
		t.Fatalf("UpdateUserImage failed: %v", err)
	}
	if img == nil {
		t.Fatal("expected image, got nil")
	}
	if img.Caption == nil || *img.Caption != newCaption {
		t.Errorf("expected caption %q, got %v", newCaption, img.Caption)
	}

	// Verify audit
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	audit := auditRepo.records[0]
	if audit.EntityType != domain.EntityTypeEntry {
		t.Errorf("expected EntityType ENTRY, got %v", audit.EntityType)
	}
	captionChange, ok := audit.Changes["caption"].(map[string]any)
	if !ok {
		t.Fatal("expected 'caption' in audit changes")
	}
	if captionChange["old"] != oldCaption {
		t.Errorf("expected old caption %q, got %v", oldCaption, captionChange["old"])
	}
	if captionChange["new"] != newCaption {
		t.Errorf("expected new caption %q, got %v", newCaption, captionChange["new"])
	}
}

func TestService_UpdateUserImage_ImageNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	imageID := uuid.New()

	imageRepo := &mockImageRepo{
		getUserByIDForUserFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.UserImage, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, nil, nil, nil, nil, imageRepo, &mockAuditRepo{}, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	newCaption := "new"
	input := UpdateUserImageInput{
		ImageID: imageID,
		Caption: &newCaption,
	}

	_, err := svc.UpdateUserImage(ctx, input)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateUserImage_NoChange(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	imageID := uuid.New()
	caption := "same caption"

	imageRepo := &mockImageRepo{
		getUserByIDForUserFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.UserImage, error) {
			return &domain.UserImage{ID: imageID, EntryID: entryID, Caption: &caption}, nil
		},
		updateUserFunc: func(ctx context.Context, iid uuid.UUID, cap *string) (*domain.UserImage, error) {
			return &domain.UserImage{ID: iid, EntryID: entryID, Caption: cap}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, nil, nil, nil, nil, imageRepo, auditRepo, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	input := UpdateUserImageInput{
		ImageID: imageID,
		Caption: &caption, // Same as old
	}

	_, err := svc.UpdateUserImage(ctx, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// No audit when nothing changed
	if len(auditRepo.records) != 0 {
		t.Errorf("expected 0 audit records for no-change update, got %d", len(auditRepo.records))
	}
}

// ---------------------------------------------------------------------------
// NoUserID Tests for UserImage Operations
// ---------------------------------------------------------------------------

func TestService_AddUserImage_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, nil, nil, nil, &mockImageRepo{}, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := AddUserImageInput{
		EntryID: uuid.New(),
		URL:     "https://example.com/image.jpg",
	}

	_, err := svc.AddUserImage(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_UpdateUserImage_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, nil, nil, nil, nil, &mockImageRepo{}, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	caption := "test"
	input := UpdateUserImageInput{
		ImageID: uuid.New(),
		Caption: &caption,
	}

	_, err := svc.UpdateUserImage(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_DeleteUserImage_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, nil, nil, nil, nil, &mockImageRepo{}, nil, &mockTxManager{})

	ctx := context.Background()

	err := svc.DeleteUserImage(ctx, uuid.New())

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteUserImage Tests
// ---------------------------------------------------------------------------

func TestService_DeleteUserImage_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	imageID := uuid.New()

	imageRepo := &mockImageRepo{
		getUserByIDForUserFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.UserImage, error) {
			if uid != userID || iid != imageID {
				return nil, domain.ErrNotFound
			}
			return &domain.UserImage{ID: imageID, EntryID: entryID, URL: "https://example.com/img.jpg"}, nil
		},
		deleteUserFunc: func(ctx context.Context, iid uuid.UUID) error {
			if iid != imageID {
				t.Errorf("unexpected imageID: got %v, want %v", iid, imageID)
			}
			return nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, nil, nil, nil, nil, imageRepo, auditRepo, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteUserImage(ctx, imageID)
	if err != nil {
		t.Fatalf("DeleteUserImage failed: %v", err)
	}

	// Verify audit
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	audit := auditRepo.records[0]
	if audit.EntityType != domain.EntityTypeEntry {
		t.Errorf("expected EntityType ENTRY, got %v", audit.EntityType)
	}
	if *audit.EntityID != entryID {
		t.Errorf("expected EntityID %v, got %v", entryID, *audit.EntityID)
	}
	if audit.Action != domain.AuditActionUpdate {
		t.Errorf("expected Action UPDATE, got %v", audit.Action)
	}
}

func TestService_DeleteUserImage_ImageNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	imageID := uuid.New()

	imageRepo := &mockImageRepo{
		getUserByIDForUserFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.UserImage, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, nil, nil, nil, nil, imageRepo, nil, &mockTxManager{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteUserImage(ctx, imageID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_DeleteUserImage_ImageFromForeignEntry(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	imageID := uuid.New()

	// GetUserByIDForUser returns ErrNotFound for foreign user (JOIN-based ownership)
	imageRepo := &mockImageRepo{
		getUserByIDForUserFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.UserImage, error) {
			if uid != userID {
				return nil, domain.ErrNotFound
			}
			return &domain.UserImage{ID: imageID}, nil
		},
	}

	svc := NewService(logger, nil, nil, nil, nil, imageRepo, nil, &mockTxManager{})

	// Different user should get ErrNotFound
	otherCtx := ctxutil.WithUserID(context.Background(), uuid.New())
	err := svc.DeleteUserImage(otherCtx, imageID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound (ownership violation), got %v", err)
	}
}
