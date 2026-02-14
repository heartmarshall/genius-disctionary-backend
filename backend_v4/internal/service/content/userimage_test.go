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
// Mock extensions for user image operations
// ---------------------------------------------------------------------------

type mockImageRepo struct {
	getUserByIDFunc func(ctx context.Context, imageID uuid.UUID) (*domain.UserImage, error)
	createUserFunc  func(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error)
	deleteUserFunc  func(ctx context.Context, imageID uuid.UUID) error
}

func (m *mockImageRepo) GetUserByID(ctx context.Context, imageID uuid.UUID) (*domain.UserImage, error) {
	if m.getUserByIDFunc != nil {
		return m.getUserByIDFunc(ctx, imageID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockImageRepo) CreateUser(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, entryID, url, caption)
	}
	return &domain.UserImage{ID: uuid.New(), EntryID: entryID, URL: url, Caption: caption}, nil
}

func (m *mockImageRepo) DeleteUser(ctx context.Context, imageID uuid.UUID) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, imageID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AddUserImage Tests (AI1-AI5)
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

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, nil, nil)
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

	svc := NewService(logger, entryRepo, nil, nil, nil, nil, nil, nil)
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

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, nil, nil)
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

func TestService_AddUserImage_WithoutCaption(t *testing.T) {
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
		createUserFunc: func(ctx context.Context, eid uuid.UUID, url string, cap *string) (*domain.UserImage, error) {
			if cap != nil {
				t.Error("expected caption to be nil")
			}
			return &domain.UserImage{ID: imageID, EntryID: eid, URL: url, Caption: nil}, nil
		},
	}

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, nil, nil)
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

	if img.Caption != nil {
		t.Errorf("expected nil caption, got %v", *img.Caption)
	}
}

// ---------------------------------------------------------------------------
// DeleteUserImage Tests (DI1-DI3)
// ---------------------------------------------------------------------------

func TestService_DeleteUserImage_HappyPath(t *testing.T) {
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
		getUserByIDFunc: func(ctx context.Context, iid uuid.UUID) (*domain.UserImage, error) {
			if iid != imageID {
				return nil, domain.ErrNotFound
			}
			return &domain.UserImage{ID: imageID, EntryID: entryID}, nil
		},
		deleteUserFunc: func(ctx context.Context, iid uuid.UUID) error {
			if iid != imageID {
				t.Errorf("unexpected imageID: got %v, want %v", iid, imageID)
			}
			return nil
		},
	}

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, nil, nil)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteUserImage(ctx, imageID)
	if err != nil {
		t.Fatalf("DeleteUserImage failed: %v", err)
	}
}

func TestService_DeleteUserImage_ImageNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	imageID := uuid.New()

	imageRepo := &mockImageRepo{
		getUserByIDFunc: func(ctx context.Context, iid uuid.UUID) (*domain.UserImage, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, nil, nil, nil, nil, imageRepo, nil, nil)
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
	otherUserID := uuid.New()
	entryID := uuid.New()
	imageID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			// Entry exists but belongs to different user
			if eid != entryID {
				return nil, domain.ErrNotFound
			}
			if uid != otherUserID {
				return nil, domain.ErrNotFound
			}
			return &domain.Entry{ID: entryID, UserID: otherUserID}, nil
		},
	}

	imageRepo := &mockImageRepo{
		getUserByIDFunc: func(ctx context.Context, iid uuid.UUID) (*domain.UserImage, error) {
			if iid != imageID {
				return nil, domain.ErrNotFound
			}
			return &domain.UserImage{ID: imageID, EntryID: entryID}, nil
		},
	}

	svc := NewService(logger, entryRepo, nil, nil, nil, imageRepo, nil, nil)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteUserImage(ctx, imageID)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound (ownership violation), got %v", err)
	}
}
