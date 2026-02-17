package inbox_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*inbox.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return inbox.New(pool), pool
}

// buildInboxItem creates a domain.InboxItem for testing.
func buildInboxItem(userID uuid.UUID, text string, ctx *string) domain.InboxItem {
	return domain.InboxItem{
		ID:        uuid.New(),
		UserID:    userID,
		Text:      text,
		Context:   ctx,
		CreatedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
}

// ---------------------------------------------------------------------------
// Create tests
// ---------------------------------------------------------------------------

func TestRepo_Create_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	itemCtx := "saw it in a movie"
	input := buildInboxItem(user.ID, "serendipity", &itemCtx)

	got, err := repo.Create(ctx, user.ID, &input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.ID != input.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, input.ID)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, user.ID)
	}
	if got.Text != "serendipity" {
		t.Errorf("Text mismatch: got %q, want %q", got.Text, "serendipity")
	}
	if got.Context == nil || *got.Context != "saw it in a movie" {
		t.Errorf("Context mismatch: got %v, want %q", got.Context, "saw it in a movie")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestRepo_Create_NilContext(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	input := buildInboxItem(user.ID, "ephemeral", nil)

	got, err := repo.Create(ctx, user.ID, &input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.Context != nil {
		t.Errorf("Context should be nil, got %v", got.Context)
	}
}

func TestRepo_Create_InvalidUserID(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	bogusUserID := uuid.New()
	input := buildInboxItem(bogusUserID, "will fail", nil)

	_, err := repo.Create(ctx, bogusUserID, &input)
	assertIsDomainError(t, err, domain.ErrNotFound) // FK violation -> ErrNotFound
}

// ---------------------------------------------------------------------------
// GetByID tests
// ---------------------------------------------------------------------------

func TestRepo_GetByID_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	itemCtx := "from a podcast"
	input := buildInboxItem(user.ID, "ubiquitous", &itemCtx)

	created, err := repo.Create(ctx, user.ID, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: unexpected error: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.Text != created.Text {
		t.Errorf("Text mismatch: got %q, want %q", got.Text, created.Text)
	}
	if got.Context == nil || *got.Context != *created.Context {
		t.Errorf("Context mismatch: got %v, want %v", got.Context, created.Context)
	}
}

func TestRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	_, err := repo.GetByID(ctx, user.ID, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_GetByID_WrongUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	input := buildInboxItem(user1.ID, "private word", nil)
	created, err := repo.Create(ctx, user1.ID, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// user2 should not be able to access user1's item.
	_, err = repo.GetByID(ctx, user2.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// ListByUser tests
// ---------------------------------------------------------------------------

func TestRepo_ListByUser_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Create 3 items with staggered timestamps.
	for i := range 3 {
		item := buildInboxItem(user.ID, "word-"+uuid.New().String()[:8], nil)
		item.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, user.ID, &item); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	items, totalCount, err := repo.List(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}

	if totalCount != 3 {
		t.Errorf("TotalCount: got %d, want 3", totalCount)
	}
	if len(items) != 3 {
		t.Fatalf("Items count: got %d, want 3", len(items))
	}

	// Verify descending order by created_at.
	for i := 1; i < len(items); i++ {
		if items[i].CreatedAt.After(items[i-1].CreatedAt) {
			t.Errorf("Items not in DESC order: [%d].CreatedAt=%s > [%d].CreatedAt=%s",
				i, items[i].CreatedAt, i-1, items[i-1].CreatedAt)
		}
	}
}

func TestRepo_ListByUser_EmptyInbox(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	items, totalCount, err := repo.List(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}

	if totalCount != 0 {
		t.Errorf("TotalCount: got %d, want 0", totalCount)
	}
	if items == nil {
		t.Fatal("Items should not be nil (empty inbox should return empty slice)")
	}
	if len(items) != 0 {
		t.Errorf("Items count: got %d, want 0", len(items))
	}
}

func TestRepo_ListByUser_Pagination(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Create 5 items.
	for i := range 5 {
		item := buildInboxItem(user.ID, "paginate-"+uuid.New().String()[:8], nil)
		item.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, user.ID, &item); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Page 1: limit 2, offset 0.
	page1Items, page1Total, err := repo.List(ctx, user.ID, 2, 0)
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if page1Total != 5 {
		t.Errorf("page1 TotalCount: got %d, want 5", page1Total)
	}
	if len(page1Items) != 2 {
		t.Fatalf("page1 Items count: got %d, want 2", len(page1Items))
	}

	// Page 2: limit 2, offset 2.
	page2Items, _, err := repo.List(ctx, user.ID, 2, 2)
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2Items) != 2 {
		t.Fatalf("page2 Items count: got %d, want 2", len(page2Items))
	}

	// Page 3: limit 2, offset 4.
	page3Items, _, err := repo.List(ctx, user.ID, 2, 4)
	if err != nil {
		t.Fatalf("List page3: %v", err)
	}
	if len(page3Items) != 1 {
		t.Fatalf("page3 Items count: got %d, want 1", len(page3Items))
	}

	// Verify no overlap between pages.
	ids := make(map[uuid.UUID]bool)
	allItems := append(page1Items, page2Items...)
	allItems = append(allItems, page3Items...)
	for _, item := range allItems {
		if ids[item.ID] {
			t.Errorf("duplicate item ID %s across pages", item.ID)
		}
		ids[item.ID] = true
	}
	if len(ids) != 5 {
		t.Errorf("expected 5 unique items across pages, got %d", len(ids))
	}
}

func TestRepo_ListByUser_IsolationBetweenUsers(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	// Create items for user1 and user2.
	for i := range 3 {
		item := buildInboxItem(user1.ID, "user1-word-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user1.ID, &item); err != nil {
			t.Fatalf("Create user1[%d]: %v", i, err)
		}
	}
	for i := range 2 {
		item := buildInboxItem(user2.ID, "user2-word-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user2.ID, &item); err != nil {
			t.Fatalf("Create user2[%d]: %v", i, err)
		}
	}

	_, total1, err := repo.List(ctx, user1.ID, 10, 0)
	if err != nil {
		t.Fatalf("List user1: %v", err)
	}
	if total1 != 3 {
		t.Errorf("user1 TotalCount: got %d, want 3", total1)
	}

	_, total2, err := repo.List(ctx, user2.ID, 10, 0)
	if err != nil {
		t.Fatalf("List user2: %v", err)
	}
	if total2 != 2 {
		t.Errorf("user2 TotalCount: got %d, want 2", total2)
	}
}

// ---------------------------------------------------------------------------
// CountByUser tests
// ---------------------------------------------------------------------------

func TestRepo_CountByUser_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Initially empty.
	count, err := repo.Count(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByUser: %v", err)
	}
	if count != 0 {
		t.Errorf("initial count: got %d, want 0", count)
	}

	// Create 3 items.
	for i := range 3 {
		item := buildInboxItem(user.ID, "count-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user.ID, &item); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	count, err = repo.Count(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByUser after create: %v", err)
	}
	if count != 3 {
		t.Errorf("count after create: got %d, want 3", count)
	}
}

// ---------------------------------------------------------------------------
// Delete tests
// ---------------------------------------------------------------------------

func TestRepo_Delete_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	input := buildInboxItem(user.ID, "to-be-deleted", nil)
	created, err := repo.Create(ctx, user.ID, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.Delete(ctx, user.ID, created.ID); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}

	// Verify item is gone.
	_, err = repo.GetByID(ctx, user.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Delete_NotFound(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	err := repo.Delete(ctx, user.ID, uuid.New())
	assertIsDomainError(t, err, domain.ErrNotFound)
}

func TestRepo_Delete_WrongUser(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	input := buildInboxItem(user1.ID, "user1-only", nil)
	created, err := repo.Create(ctx, user1.ID, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// user2 should not be able to delete user1's item.
	err = repo.Delete(ctx, user2.ID, created.ID)
	assertIsDomainError(t, err, domain.ErrNotFound)

	// Verify item still exists for user1.
	got, err := repo.GetByID(ctx, user1.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID after wrong-user delete: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("item should still exist, ID mismatch: got %s, want %s", got.ID, created.ID)
	}
}

// ---------------------------------------------------------------------------
// DeleteAll tests
// ---------------------------------------------------------------------------

func TestRepo_DeleteAll_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Create several items.
	for i := range 5 {
		item := buildInboxItem(user.ID, "clearme-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user.ID, &item); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Delete all.
	if _, err := repo.DeleteAll(ctx, user.ID); err != nil {
		t.Fatalf("DeleteAll: unexpected error: %v", err)
	}

	// Verify inbox is empty.
	count, err := repo.Count(ctx, user.ID)
	if err != nil {
		t.Fatalf("CountByUser after DeleteAll: %v", err)
	}
	if count != 0 {
		t.Errorf("count after DeleteAll: got %d, want 0", count)
	}
}

func TestRepo_DeleteAll_Idempotent(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Calling DeleteAll on an empty inbox should not error.
	if _, err := repo.DeleteAll(ctx, user.ID); err != nil {
		t.Fatalf("DeleteAll on empty inbox: unexpected error: %v", err)
	}

	// Double call should also succeed.
	if _, err := repo.DeleteAll(ctx, user.ID); err != nil {
		t.Fatalf("DeleteAll second call: unexpected error: %v", err)
	}
}

func TestRepo_DeleteAll_UserIsolation(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	// Create items for both users.
	for i := range 3 {
		item1 := buildInboxItem(user1.ID, "u1-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user1.ID, &item1); err != nil {
			t.Fatalf("Create user1[%d]: %v", i, err)
		}
		item2 := buildInboxItem(user2.ID, "u2-"+uuid.New().String()[:8], nil)
		if _, err := repo.Create(ctx, user2.ID, &item2); err != nil {
			t.Fatalf("Create user2[%d]: %v", i, err)
		}
	}

	// Delete all for user1.
	if _, err := repo.DeleteAll(ctx, user1.ID); err != nil {
		t.Fatalf("DeleteAll user1: %v", err)
	}

	// user1 inbox should be empty.
	count1, err := repo.Count(ctx, user1.ID)
	if err != nil {
		t.Fatalf("CountByUser user1: %v", err)
	}
	if count1 != 0 {
		t.Errorf("user1 count after DeleteAll: got %d, want 0", count1)
	}

	// user2 inbox should be untouched.
	count2, err := repo.Count(ctx, user2.ID)
	if err != nil {
		t.Fatalf("CountByUser user2: %v", err)
	}
	if count2 != 3 {
		t.Errorf("user2 count after user1 DeleteAll: got %d, want 3", count2)
	}
}

// ---------------------------------------------------------------------------
// Round-trip test
// ---------------------------------------------------------------------------

func TestRepo_Create_ThenGetByID_RoundTrip(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	itemCtx := "heard in conversation"
	input := buildInboxItem(user.ID, "mellifluous", &itemCtx)

	created, err := repo.Create(ctx, user.ID, &input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByID(ctx, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetByID after Create: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, created.ID)
	}
	if got.Text != created.Text {
		t.Errorf("Text mismatch: got %q, want %q", got.Text, created.Text)
	}
	if got.Context == nil || created.Context == nil || *got.Context != *created.Context {
		t.Errorf("Context mismatch: got %v, want %v", got.Context, created.Context)
	}
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %s, want %s", got.CreatedAt, created.CreatedAt)
	}
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func assertIsDomainError(t *testing.T, err error, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error wrapping %v, got nil", target)
	}
	if !errors.Is(err, target) {
		t.Fatalf("expected error wrapping %v, got: %v", target, err)
	}
}
