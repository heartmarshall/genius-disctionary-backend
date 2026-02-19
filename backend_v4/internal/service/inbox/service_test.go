package inbox

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// newTestService creates a Service with the given mock and a discard logger.
func newTestService(t *testing.T, mock *inboxRepoMock) *Service {
	t.Helper()
	return &Service{
		inbox: mock,
		log:   slog.Default(),
	}
}

// ---------------------------------------------------------------------------
// CreateItem Tests (11 tests)
// ---------------------------------------------------------------------------

func TestCreateItem_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()
	ctxVal := "some context"

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 10, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			return &domain.InboxItem{
				ID:        itemID,
				UserID:    uid,
				Text:      item.Text,
				Context:   item.Context,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateItem(ctx, CreateItemInput{
		Text:    "hello world",
		Context: &ctxVal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != itemID {
		t.Errorf("item ID: got %v, want %v", result.ID, itemID)
	}
	if result.Text != "hello world" {
		t.Errorf("text: got %q, want %q", result.Text, "hello world")
	}
	if result.Context == nil || *result.Context != ctxVal {
		t.Errorf("context: got %v, want %q", result.Context, ctxVal)
	}
	if len(mock.CountCalls()) != 1 {
		t.Errorf("Count calls: got %d, want 1", len(mock.CountCalls()))
	}
	if len(mock.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(mock.CreateCalls()))
	}
}

func TestCreateItem_WithoutContext(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			if item.Context != nil {
				t.Errorf("context should be nil, got %v", item.Context)
			}
			return &domain.InboxItem{
				ID:        itemID,
				UserID:    uid,
				Text:      item.Text,
				Context:   nil,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateItem(ctx, CreateItemInput{
		Text:    "hello",
		Context: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Context != nil {
		t.Errorf("context should be nil, got %v", result.Context)
	}
}

func TestCreateItem_EmptyText(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{
		Text: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if len(ve.Errors) != 1 {
		t.Fatalf("expected 1 field error, got %d", len(ve.Errors))
	}
	if ve.Errors[0].Field != "text" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "text")
	}
	if ve.Errors[0].Message != "required" {
		t.Errorf("message: got %q, want %q", ve.Errors[0].Message, "required")
	}
}

func TestCreateItem_WhitespaceOnlyText(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{
		Text: "   \t\n  ",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "text" || ve.Errors[0].Message != "required" {
		t.Errorf("expected text/required, got %s/%s", ve.Errors[0].Field, ve.Errors[0].Message)
	}
}

func TestCreateItem_TextTooLong(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	longText := strings.Repeat("a", 501)
	_, err := svc.CreateItem(ctx, CreateItemInput{
		Text: longText,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "text" && fe.Message == "max 500 characters" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected text/max 500 characters error, got %v", ve.Errors)
	}
}

func TestCreateItem_ContextTooLong(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	longCtx := strings.Repeat("b", 2001)
	_, err := svc.CreateItem(ctx, CreateItemInput{
		Text:    "hello",
		Context: &longCtx,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "context" && fe.Message == "max 2000 characters" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected context/max 2000 characters error, got %v", ve.Errors)
	}
}

func TestCreateItem_EmptyContextToNil(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()
	emptyCtx := "   "

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			if item.Context != nil {
				t.Errorf("expected context to be nil after trimming whitespace, got %q", *item.Context)
			}
			return &domain.InboxItem{
				ID:        itemID,
				UserID:    uid,
				Text:      item.Text,
				Context:   item.Context,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateItem(ctx, CreateItemInput{
		Text:    "hello",
		Context: &emptyCtx,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Context != nil {
		t.Errorf("expected nil context, got %v", result.Context)
	}
}

func TestCreateItem_TextTrimmed(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			if item.Text != "hello world" {
				t.Errorf("text not trimmed: got %q, want %q", item.Text, "hello world")
			}
			return &domain.InboxItem{
				ID:        itemID,
				UserID:    uid,
				Text:      item.Text,
				Context:   item.Context,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateItem(ctx, CreateItemInput{
		Text: "  hello world  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "hello world" {
		t.Errorf("text: got %q, want %q", result.Text, "hello world")
	}
}

func TestCreateItem_InboxFull(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return MaxInboxItems, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{
		Text: "hello",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "inbox" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "inbox")
	}
	if ve.Errors[0].Message != "inbox is full (max 500 items)" {
		t.Errorf("message: got %q, want %q", ve.Errors[0].Message, "inbox is full (max 500 items)")
	}
	// Create should not be called
	if len(mock.CreateCalls()) != 0 {
		t.Errorf("Create calls: got %d, want 0", len(mock.CreateCalls()))
	}
}

func TestCreateItem_DuplicateAllowed(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	callCount := 0

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			callCount++
			return &domain.InboxItem{
				ID:        uuid.New(),
				UserID:    uid,
				Text:      item.Text,
				Context:   item.Context,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{Text: "duplicate"})
	if err != nil {
		t.Fatalf("first create: unexpected error: %v", err)
	}

	_, err = svc.CreateItem(ctx, CreateItemInput{Text: "duplicate"})
	if err != nil {
		t.Fatalf("second create: unexpected error: %v", err)
	}

	if len(mock.CreateCalls()) != 2 {
		t.Errorf("Create calls: got %d, want 2", len(mock.CreateCalls()))
	}
}

func TestCreateItem_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &inboxRepoMock{})
	ctx := context.Background() // no user ID

	_, err := svc.CreateItem(ctx, CreateItemInput{Text: "hello"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// ListItems Tests (7 tests)
// ---------------------------------------------------------------------------

func TestListItems_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	items := []*domain.InboxItem{
		{ID: uuid.New(), UserID: userID, Text: "item1", CreatedAt: time.Now()},
		{ID: uuid.New(), UserID: userID, Text: "item2", CreatedAt: time.Now()},
	}

	mock := &inboxRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if limit != 10 {
				t.Errorf("limit: got %d, want 10", limit)
			}
			if offset != 0 {
				t.Errorf("offset: got %d, want 0", offset)
			}
			return items, 2, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, total, err := svc.ListItems(ctx, ListItemsInput{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("result length: got %d, want 2", len(result))
	}
	if total != 2 {
		t.Errorf("total: got %d, want 2", total)
	}
}

func TestListItems_Empty(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
			return []*domain.InboxItem{}, 0, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, total, err := svc.ListItems(ctx, ListItemsInput{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("result length: got %d, want 0", len(result))
	}
	if total != 0 {
		t.Errorf("total: got %d, want 0", total)
	}
}

func TestListItems_Pagination(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	item := &domain.InboxItem{ID: uuid.New(), UserID: userID, Text: "item3", CreatedAt: time.Now()}

	mock := &inboxRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
			if limit != 5 {
				t.Errorf("limit: got %d, want 5", limit)
			}
			if offset != 10 {
				t.Errorf("offset: got %d, want 10", offset)
			}
			return []*domain.InboxItem{item}, 15, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, total, err := svc.ListItems(ctx, ListItemsInput{Limit: 5, Offset: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("result length: got %d, want 1", len(result))
	}
	if total != 15 {
		t.Errorf("total: got %d, want 15", total)
	}
}

func TestListItems_DefaultLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
			if limit != DefaultLimit {
				t.Errorf("limit: got %d, want %d (DefaultLimit)", limit, DefaultLimit)
			}
			return []*domain.InboxItem{}, 0, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.ListCalls()) != 1 {
		t.Errorf("List calls: got %d, want 1", len(mock.ListCalls()))
	}
}

func TestListItems_InvalidLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: -1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "limit" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "limit")
	}
}

func TestListItems_LimitTooLarge(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: 201})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "limit" && fe.Message == "max 200" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected limit/max 200 error, got %v", ve.Errors)
	}
}

func TestListItems_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &inboxRepoMock{})
	ctx := context.Background()

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: 10})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// GetItem Tests (4 tests)
// ---------------------------------------------------------------------------

func TestGetItem_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()
	item := &domain.InboxItem{
		ID:        itemID,
		UserID:    userID,
		Text:      "test item",
		CreatedAt: time.Now(),
	}

	mock := &inboxRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.InboxItem, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if iid != itemID {
				t.Errorf("itemID: got %v, want %v", iid, itemID)
			}
			return item, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.GetItem(ctx, itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != itemID {
		t.Errorf("item ID: got %v, want %v", result.ID, itemID)
	}
	if result.Text != "test item" {
		t.Errorf("text: got %q, want %q", result.Text, "test item")
	}
}

func TestGetItem_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()

	mock := &inboxRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.InboxItem, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.GetItem(ctx, itemID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestGetItem_WrongUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserItemID := uuid.New()

	mock := &inboxRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, iid uuid.UUID) (*domain.InboxItem, error) {
			// Repo filters by userID, so wrong user gets ErrNotFound
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.GetItem(ctx, otherUserItemID)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestGetItem_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &inboxRepoMock{})
	ctx := context.Background()

	_, err := svc.GetItem(ctx, uuid.New())
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteItem Tests (5 tests)
// ---------------------------------------------------------------------------

func TestDeleteItem_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()

	mock := &inboxRepoMock{
		DeleteFunc: func(ctx context.Context, uid, iid uuid.UUID) error {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if iid != itemID {
				t.Errorf("itemID: got %v, want %v", iid, itemID)
			}
			return nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteItem(ctx, DeleteItemInput{ItemID: itemID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.DeleteCalls()) != 1 {
		t.Errorf("Delete calls: got %d, want 1", len(mock.DeleteCalls()))
	}
}

func TestDeleteItem_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	itemID := uuid.New()

	mock := &inboxRepoMock{
		DeleteFunc: func(ctx context.Context, uid, iid uuid.UUID) error {
			return domain.ErrNotFound
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteItem(ctx, DeleteItemInput{ItemID: itemID})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestDeleteItem_WrongUser(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	otherUserItemID := uuid.New()

	mock := &inboxRepoMock{
		DeleteFunc: func(ctx context.Context, uid, iid uuid.UUID) error {
			// Repo filters by userID, 0 rows affected → ErrNotFound
			return domain.ErrNotFound
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteItem(ctx, DeleteItemInput{ItemID: otherUserItemID})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestDeleteItem_NilID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteItem(ctx, DeleteItemInput{ItemID: uuid.Nil})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "item_id" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "item_id")
	}
	if ve.Errors[0].Message != "required" {
		t.Errorf("message: got %q, want %q", ve.Errors[0].Message, "required")
	}
}

func TestDeleteItem_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &inboxRepoMock{})
	ctx := context.Background()

	err := svc.DeleteItem(ctx, DeleteItemInput{ItemID: uuid.New()})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteAll Tests (3 tests)
// ---------------------------------------------------------------------------

func TestDeleteAll_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		DeleteAllFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			return 5, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	count, err := svc.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Errorf("deleted count: got %d, want 5", count)
	}
	if len(mock.DeleteAllCalls()) != 1 {
		t.Errorf("DeleteAll calls: got %d, want 1", len(mock.DeleteAllCalls()))
	}
}

func TestDeleteAll_EmptyInbox(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		DeleteAllFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	count, err := svc.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("deleted count: got %d, want 0", count)
	}
}

func TestDeleteAll_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, &inboxRepoMock{})
	ctx := context.Background()

	_, err := svc.DeleteAll(ctx)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestDeleteAll_RepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repoErr := errors.New("db connection lost")

	mock := &inboxRepoMock{
		DeleteAllFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, repoErr
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.DeleteAll(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error should wrap repo error: got %v", err)
	}
	if !strings.Contains(err.Error(), "delete all inbox items") {
		t.Errorf("error should contain context: got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// CreateItem — repo error paths
// ---------------------------------------------------------------------------

func TestCreateItem_CountRepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repoErr := errors.New("db timeout")

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, repoErr
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{Text: "hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error should wrap repo error: got %v", err)
	}
	if !strings.Contains(err.Error(), "count inbox items") {
		t.Errorf("error should contain context: got %q", err.Error())
	}
	if len(mock.CreateCalls()) != 0 {
		t.Error("Create should not be called when Count fails")
	}
}

func TestCreateItem_CreateRepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repoErr := errors.New("unique constraint violation")

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			return nil, repoErr
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{Text: "hello"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error should wrap repo error: got %v", err)
	}
	if !strings.Contains(err.Error(), "create inbox item") {
		t.Errorf("error should contain context: got %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// ListItems — repo error path + missing validation
// ---------------------------------------------------------------------------

func TestListItems_RepoError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	repoErr := errors.New("query failed")

	mock := &inboxRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
			return nil, 0, repoErr
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: 10})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Errorf("error should wrap repo error: got %v", err)
	}
	if !strings.Contains(err.Error(), "list inbox items") {
		t.Errorf("error should contain context: got %q", err.Error())
	}
}

func TestListItems_NegativeOffset(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newTestService(t, &inboxRepoMock{})
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: 10, Offset: -1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "offset" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "offset")
	}
}

// ---------------------------------------------------------------------------
// Boundary tests
// ---------------------------------------------------------------------------

func TestCreateItem_TextExactlyAtLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	text := strings.Repeat("a", 500) // exactly at limit

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			return &domain.InboxItem{
				ID:        uuid.New(),
				UserID:    uid,
				Text:      item.Text,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateItem(ctx, CreateItemInput{Text: text})
	if err != nil {
		t.Fatalf("500-char text should be accepted, got error: %v", err)
	}
	if len(result.Text) != 500 {
		t.Errorf("text length: got %d, want 500", len(result.Text))
	}
}

func TestCreateItem_ContextExactlyAtLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctxText := strings.Repeat("b", 2000) // exactly at limit

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			return &domain.InboxItem{
				ID:        uuid.New(),
				UserID:    uid,
				Text:      item.Text,
				Context:   item.Context,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{
		Text:    "hello",
		Context: &ctxText,
	})
	if err != nil {
		t.Fatalf("2000-char context should be accepted, got error: %v", err)
	}
}

func TestCreateItem_InboxAtCapacityMinusOne(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return MaxInboxItems - 1, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error) {
			return &domain.InboxItem{
				ID:        uuid.New(),
				UserID:    uid,
				Text:      item.Text,
				CreatedAt: time.Now(),
			}, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateItem(ctx, CreateItemInput{Text: "last slot"})
	if err != nil {
		t.Fatalf("item 500 (count=499) should be accepted, got error: %v", err)
	}
	if len(mock.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(mock.CreateCalls()))
	}
}

func TestListItems_LimitExactlyAtMax(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mock := &inboxRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error) {
			if limit != 200 {
				t.Errorf("limit: got %d, want 200", limit)
			}
			return []*domain.InboxItem{}, 0, nil
		},
	}

	svc := newTestService(t, mock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, _, err := svc.ListItems(ctx, ListItemsInput{Limit: 200})
	if err != nil {
		t.Fatalf("limit=200 should be accepted, got error: %v", err)
	}
}
