package audit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/audit"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// newRepo sets up a test DB and returns a ready Repo + pool.
func newRepo(t *testing.T) (*audit.Repo, *pgxpool.Pool) {
	t.Helper()
	pool := testhelper.SetupTestDB(t)
	return audit.New(pool), pool
}

// buildAuditRecord creates a domain.AuditRecord for testing.
func buildAuditRecord(userID uuid.UUID, entityType domain.EntityType, entityID *uuid.UUID, action domain.AuditAction, changes map[string]any) domain.AuditRecord {
	return domain.AuditRecord{
		ID:         uuid.New(),
		UserID:     userID,
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Changes:    changes,
		CreatedAt:  time.Now().UTC().Truncate(time.Microsecond),
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

	entityID := uuid.New()
	changes := map[string]any{
		"text":     "hello",
		"old_text": "helo",
	}
	input := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionUpdate, changes)

	got, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.ID != input.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, input.ID)
	}
	if got.UserID != user.ID {
		t.Errorf("UserID mismatch: got %s, want %s", got.UserID, user.ID)
	}
	if got.EntityType != domain.EntityTypeEntry {
		t.Errorf("EntityType mismatch: got %s, want %s", got.EntityType, domain.EntityTypeEntry)
	}
	if got.EntityID == nil || *got.EntityID != entityID {
		t.Errorf("EntityID mismatch: got %v, want %s", got.EntityID, entityID)
	}
	if got.Action != domain.AuditActionUpdate {
		t.Errorf("Action mismatch: got %s, want %s", got.Action, domain.AuditActionUpdate)
	}
	if got.Changes["text"] != "hello" {
		t.Errorf("Changes[text] mismatch: got %v, want %q", got.Changes["text"], "hello")
	}
	if got.Changes["old_text"] != "helo" {
		t.Errorf("Changes[old_text] mismatch: got %v, want %q", got.Changes["old_text"], "helo")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestRepo_Create_NilEntityID(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	changes := map[string]any{"action": "bulk_import"}
	input := buildAuditRecord(user.ID, domain.EntityTypeEntry, nil, domain.AuditActionCreate, changes)

	got, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.EntityID != nil {
		t.Errorf("EntityID should be nil, got %v", got.EntityID)
	}
}

func TestRepo_Create_EmptyChanges(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()
	input := buildAuditRecord(user.ID, domain.EntityTypeCard, &entityID, domain.AuditActionDelete, map[string]any{})

	got, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	if got.Changes == nil {
		t.Error("Changes should not be nil (empty map expected)")
	}
	if len(got.Changes) != 0 {
		t.Errorf("Changes should be empty, got %v", got.Changes)
	}
}

func TestRepo_Create_NilChanges(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()
	input := buildAuditRecord(user.ID, domain.EntityTypeCard, &entityID, domain.AuditActionCreate, nil)

	got, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	// nil map marshals to JSON "null", which unmarshals back to nil map.
	// Depending on implementation, this could be nil or empty map.
	// Both are acceptable; we just verify no error occurred.
	_ = got
}

func TestRepo_Create_InvalidUserID(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	bogusUserID := uuid.New()
	input := buildAuditRecord(bogusUserID, domain.EntityTypeEntry, nil, domain.AuditActionCreate, map[string]any{})

	_, err := repo.Create(ctx, input)
	assertIsDomainError(t, err, domain.ErrNotFound) // FK violation -> ErrNotFound
}

func TestRepo_Create_ComplexChanges(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()
	changes := map[string]any{
		"text":     "hello world",
		"count":    float64(42),
		"nested":   map[string]any{"key": "value"},
		"list":     []any{"a", "b", "c"},
		"is_valid": true,
	}
	input := buildAuditRecord(user.ID, domain.EntityTypeSense, &entityID, domain.AuditActionUpdate, changes)

	got, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	// Verify complex types round-trip correctly.
	if got.Changes["text"] != "hello world" {
		t.Errorf("Changes[text] mismatch: got %v", got.Changes["text"])
	}
	if got.Changes["count"] != float64(42) {
		t.Errorf("Changes[count] mismatch: got %v", got.Changes["count"])
	}
	nested, ok := got.Changes["nested"].(map[string]any)
	if !ok {
		t.Fatalf("Changes[nested] is not map[string]any: %T", got.Changes["nested"])
	}
	if nested["key"] != "value" {
		t.Errorf("Changes[nested][key] mismatch: got %v", nested["key"])
	}
	list, ok := got.Changes["list"].([]any)
	if !ok {
		t.Fatalf("Changes[list] is not []any: %T", got.Changes["list"])
	}
	if len(list) != 3 {
		t.Errorf("Changes[list] length: got %d, want 3", len(list))
	}
	if got.Changes["is_valid"] != true {
		t.Errorf("Changes[is_valid] mismatch: got %v", got.Changes["is_valid"])
	}
}

// ---------------------------------------------------------------------------
// GetByEntity tests
// ---------------------------------------------------------------------------

func TestRepo_GetByEntity_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()

	// Create 3 audit records for the same entity with staggered timestamps.
	for i := range 3 {
		record := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionUpdate, map[string]any{
			"step": float64(i),
		})
		record.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, record); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	got, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, entityID, 10)
	if err != nil {
		t.Fatalf("GetByEntity: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 records, got %d", len(got))
	}

	// Verify descending order by created_at.
	for i := 1; i < len(got); i++ {
		if got[i].CreatedAt.After(got[i-1].CreatedAt) {
			t.Errorf("records not in DESC order: [%d].CreatedAt=%s > [%d].CreatedAt=%s",
				i, got[i].CreatedAt, i-1, got[i-1].CreatedAt)
		}
	}

	// Verify entity fields.
	for _, rec := range got {
		if rec.EntityType != domain.EntityTypeEntry {
			t.Errorf("EntityType mismatch: got %s, want %s", rec.EntityType, domain.EntityTypeEntry)
		}
		if rec.EntityID == nil || *rec.EntityID != entityID {
			t.Errorf("EntityID mismatch: got %v, want %s", rec.EntityID, entityID)
		}
	}
}

func TestRepo_GetByEntity_LimitRespected(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()

	// Create 5 audit records.
	for i := range 5 {
		record := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionUpdate, map[string]any{
			"step": float64(i),
		})
		record.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, record); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	got, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, entityID, 2)
	if err != nil {
		t.Fatalf("GetByEntity: unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Errorf("expected 2 records (limit), got %d", len(got))
	}
}

func TestRepo_GetByEntity_EmptyResult(t *testing.T) {
	t.Parallel()
	repo, _ := newRepo(t)
	ctx := context.Background()

	got, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, uuid.New(), 10)
	if err != nil {
		t.Fatalf("GetByEntity: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("result should not be nil (empty result should return empty slice)")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 records, got %d", len(got))
	}
}

func TestRepo_GetByEntity_DifferentEntitiesIsolated(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID1 := uuid.New()
	entityID2 := uuid.New()

	// Create records for two different entities.
	for i := range 3 {
		r1 := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID1, domain.AuditActionCreate, map[string]any{"e": float64(1)})
		r1.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, r1); err != nil {
			t.Fatalf("Create entity1[%d]: %v", i, err)
		}
	}
	for i := range 2 {
		r2 := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID2, domain.AuditActionUpdate, map[string]any{"e": float64(2)})
		r2.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, r2); err != nil {
			t.Fatalf("Create entity2[%d]: %v", i, err)
		}
	}

	got1, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, entityID1, 10)
	if err != nil {
		t.Fatalf("GetByEntity entity1: %v", err)
	}
	if len(got1) != 3 {
		t.Errorf("entity1: expected 3 records, got %d", len(got1))
	}

	got2, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, entityID2, 10)
	if err != nil {
		t.Fatalf("GetByEntity entity2: %v", err)
	}
	if len(got2) != 2 {
		t.Errorf("entity2: expected 2 records, got %d", len(got2))
	}
}

func TestRepo_GetByEntity_DifferentEntityTypesIsolated(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()

	// Same entity_id but different entity_type.
	r1 := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionCreate, map[string]any{})
	if _, err := repo.Create(ctx, r1); err != nil {
		t.Fatalf("Create ENTRY: %v", err)
	}

	r2 := buildAuditRecord(user.ID, domain.EntityTypeCard, &entityID, domain.AuditActionCreate, map[string]any{})
	if _, err := repo.Create(ctx, r2); err != nil {
		t.Fatalf("Create CARD: %v", err)
	}

	gotEntry, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, entityID, 10)
	if err != nil {
		t.Fatalf("GetByEntity ENTRY: %v", err)
	}
	if len(gotEntry) != 1 {
		t.Errorf("ENTRY: expected 1 record, got %d", len(gotEntry))
	}

	gotCard, err := repo.GetByEntity(ctx, domain.EntityTypeCard, entityID, 10)
	if err != nil {
		t.Fatalf("GetByEntity CARD: %v", err)
	}
	if len(gotCard) != 1 {
		t.Errorf("CARD: expected 1 record, got %d", len(gotCard))
	}
}

// ---------------------------------------------------------------------------
// GetByUser tests
// ---------------------------------------------------------------------------

func TestRepo_GetByUser_HappyPath(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Create 3 audit records for the user with staggered timestamps.
	for i := range 3 {
		entityID := uuid.New()
		record := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionCreate, map[string]any{
			"step": float64(i),
		})
		record.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, record); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	got, err := repo.GetByUser(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByUser: unexpected error: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 records, got %d", len(got))
	}

	// Verify descending order by created_at.
	for i := 1; i < len(got); i++ {
		if got[i].CreatedAt.After(got[i-1].CreatedAt) {
			t.Errorf("records not in DESC order: [%d].CreatedAt=%s > [%d].CreatedAt=%s",
				i, got[i].CreatedAt, i-1, got[i-1].CreatedAt)
		}
	}

	// Verify user_id.
	for _, rec := range got {
		if rec.UserID != user.ID {
			t.Errorf("UserID mismatch: got %s, want %s", rec.UserID, user.ID)
		}
	}
}

func TestRepo_GetByUser_EmptyResult(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	got, err := repo.GetByUser(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByUser: unexpected error: %v", err)
	}

	if got == nil {
		t.Fatal("result should not be nil (empty result should return empty slice)")
	}
	if len(got) != 0 {
		t.Errorf("expected 0 records, got %d", len(got))
	}
}

func TestRepo_GetByUser_Pagination(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	// Create 5 records.
	for i := range 5 {
		entityID := uuid.New()
		record := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionCreate, map[string]any{
			"step": float64(i),
		})
		record.CreatedAt = time.Now().UTC().Truncate(time.Microsecond).Add(time.Duration(i) * time.Millisecond)
		if _, err := repo.Create(ctx, record); err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	// Page 1: limit 2, offset 0.
	page1, err := repo.GetByUser(ctx, user.ID, 2, 0)
	if err != nil {
		t.Fatalf("GetByUser page1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page1: expected 2 records, got %d", len(page1))
	}

	// Page 2: limit 2, offset 2.
	page2, err := repo.GetByUser(ctx, user.ID, 2, 2)
	if err != nil {
		t.Fatalf("GetByUser page2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page2: expected 2 records, got %d", len(page2))
	}

	// Page 3: limit 2, offset 4.
	page3, err := repo.GetByUser(ctx, user.ID, 2, 4)
	if err != nil {
		t.Fatalf("GetByUser page3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page3: expected 1 record, got %d", len(page3))
	}

	// Verify no overlap between pages.
	ids := make(map[uuid.UUID]bool)
	allRecords := append(page1, page2...)
	allRecords = append(allRecords, page3...)
	for _, rec := range allRecords {
		if ids[rec.ID] {
			t.Errorf("duplicate record ID %s across pages", rec.ID)
		}
		ids[rec.ID] = true
	}
	if len(ids) != 5 {
		t.Errorf("expected 5 unique records across pages, got %d", len(ids))
	}
}

func TestRepo_GetByUser_IsolationBetweenUsers(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()

	user1 := testhelper.SeedUser(t, pool)
	user2 := testhelper.SeedUser(t, pool)

	// Create records for user1 and user2.
	for i := range 3 {
		entityID := uuid.New()
		r := buildAuditRecord(user1.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionCreate, map[string]any{})
		if _, err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create user1[%d]: %v", i, err)
		}
	}
	for i := range 2 {
		entityID := uuid.New()
		r := buildAuditRecord(user2.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionCreate, map[string]any{})
		if _, err := repo.Create(ctx, r); err != nil {
			t.Fatalf("Create user2[%d]: %v", i, err)
		}
	}

	got1, err := repo.GetByUser(ctx, user1.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByUser user1: %v", err)
	}
	if len(got1) != 3 {
		t.Errorf("user1: expected 3 records, got %d", len(got1))
	}

	got2, err := repo.GetByUser(ctx, user2.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByUser user2: %v", err)
	}
	if len(got2) != 2 {
		t.Errorf("user2: expected 2 records, got %d", len(got2))
	}
}

// ---------------------------------------------------------------------------
// Round-trip test
// ---------------------------------------------------------------------------

func TestRepo_Create_ThenGetByEntity_RoundTrip(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()
	changes := map[string]any{
		"field":  "text",
		"old":    "before",
		"new":    "after",
		"number": float64(123),
	}
	input := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, domain.AuditActionUpdate, changes)

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByEntity(ctx, domain.EntityTypeEntry, entityID, 10)
	if err != nil {
		t.Fatalf("GetByEntity: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 record, got %d", len(got))
	}

	rec := got[0]
	if rec.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", rec.ID, created.ID)
	}
	if rec.UserID != created.UserID {
		t.Errorf("UserID mismatch: got %s, want %s", rec.UserID, created.UserID)
	}
	if rec.EntityType != created.EntityType {
		t.Errorf("EntityType mismatch: got %s, want %s", rec.EntityType, created.EntityType)
	}
	if rec.EntityID == nil || created.EntityID == nil || *rec.EntityID != *created.EntityID {
		t.Errorf("EntityID mismatch: got %v, want %v", rec.EntityID, created.EntityID)
	}
	if rec.Action != created.Action {
		t.Errorf("Action mismatch: got %s, want %s", rec.Action, created.Action)
	}
	if !rec.CreatedAt.Equal(created.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %s, want %s", rec.CreatedAt, created.CreatedAt)
	}

	// Verify changes map content.
	if rec.Changes["field"] != "text" {
		t.Errorf("Changes[field] mismatch: got %v, want %q", rec.Changes["field"], "text")
	}
	if rec.Changes["old"] != "before" {
		t.Errorf("Changes[old] mismatch: got %v, want %q", rec.Changes["old"], "before")
	}
	if rec.Changes["new"] != "after" {
		t.Errorf("Changes[new] mismatch: got %v, want %q", rec.Changes["new"], "after")
	}
	if rec.Changes["number"] != float64(123) {
		t.Errorf("Changes[number] mismatch: got %v, want %v", rec.Changes["number"], float64(123))
	}
}

func TestRepo_Create_ThenGetByUser_RoundTrip(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityID := uuid.New()
	input := buildAuditRecord(user.ID, domain.EntityTypeTopic, &entityID, domain.AuditActionCreate, map[string]any{
		"name": "travel",
	})

	created, err := repo.Create(ctx, input)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByUser(ctx, user.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetByUser: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 record, got %d", len(got))
	}

	rec := got[0]
	if rec.ID != created.ID {
		t.Errorf("ID mismatch: got %s, want %s", rec.ID, created.ID)
	}
	if rec.EntityType != domain.EntityTypeTopic {
		t.Errorf("EntityType mismatch: got %s, want %s", rec.EntityType, domain.EntityTypeTopic)
	}
	if rec.Changes["name"] != "travel" {
		t.Errorf("Changes[name] mismatch: got %v, want %q", rec.Changes["name"], "travel")
	}
}

// ---------------------------------------------------------------------------
// All entity types test
// ---------------------------------------------------------------------------

func TestRepo_Create_AllEntityTypes(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	entityTypes := []domain.EntityType{
		domain.EntityTypeEntry,
		domain.EntityTypeSense,
		domain.EntityTypeExample,
		domain.EntityTypeImage,
		domain.EntityTypePronunciation,
		domain.EntityTypeCard,
		domain.EntityTypeTopic,
	}

	for _, et := range entityTypes {
		entityID := uuid.New()
		record := buildAuditRecord(user.ID, et, &entityID, domain.AuditActionCreate, map[string]any{
			"type": string(et),
		})

		got, err := repo.Create(ctx, record)
		if err != nil {
			t.Fatalf("Create entity_type=%s: %v", et, err)
		}

		if got.EntityType != et {
			t.Errorf("EntityType mismatch: got %s, want %s", got.EntityType, et)
		}
	}
}

// ---------------------------------------------------------------------------
// All audit actions test
// ---------------------------------------------------------------------------

func TestRepo_Create_AllAuditActions(t *testing.T) {
	t.Parallel()
	repo, pool := newRepo(t)
	ctx := context.Background()
	user := testhelper.SeedUser(t, pool)

	actions := []domain.AuditAction{
		domain.AuditActionCreate,
		domain.AuditActionUpdate,
		domain.AuditActionDelete,
	}

	for _, action := range actions {
		entityID := uuid.New()
		record := buildAuditRecord(user.ID, domain.EntityTypeEntry, &entityID, action, map[string]any{
			"action": string(action),
		})

		got, err := repo.Create(ctx, record)
		if err != nil {
			t.Fatalf("Create action=%s: %v", action, err)
		}

		if got.Action != action {
			t.Errorf("Action mismatch: got %s, want %s", got.Action, action)
		}
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
