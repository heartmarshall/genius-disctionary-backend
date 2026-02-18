package content

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// Mock extensions for example operations
// ---------------------------------------------------------------------------

type mockExampleRepo struct {
	getByIDForUserFunc func(ctx context.Context, userID, exampleID uuid.UUID) (*domain.Example, error)
	getBySenseIDFunc   func(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error)
	countBySenseFunc   func(ctx context.Context, senseID uuid.UUID) (int, error)
	createCustomFunc   func(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
	updateFunc         func(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error)
	deleteFunc         func(ctx context.Context, exampleID uuid.UUID) error
	reorderFunc        func(ctx context.Context, items []domain.ReorderItem) error
}

func (m *mockExampleRepo) GetByIDForUser(ctx context.Context, userID, exampleID uuid.UUID) (*domain.Example, error) {
	if m.getByIDForUserFunc != nil {
		return m.getByIDForUserFunc(ctx, userID, exampleID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockExampleRepo) GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error) {
	if m.getBySenseIDFunc != nil {
		return m.getBySenseIDFunc(ctx, senseID)
	}
	return nil, nil
}

func (m *mockExampleRepo) CountBySense(ctx context.Context, senseID uuid.UUID) (int, error) {
	if m.countBySenseFunc != nil {
		return m.countBySenseFunc(ctx, senseID)
	}
	return 0, nil
}

func (m *mockExampleRepo) CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error) {
	if m.createCustomFunc != nil {
		return m.createCustomFunc(ctx, senseID, sentence, translation, sourceSlug)
	}
	return &domain.Example{ID: uuid.New()}, nil
}

func (m *mockExampleRepo) Update(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, exampleID, sentence, translation)
	}
	return &domain.Example{ID: exampleID}, nil
}

func (m *mockExampleRepo) Delete(ctx context.Context, exampleID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, exampleID)
	}
	return nil
}

func (m *mockExampleRepo) Reorder(ctx context.Context, items []domain.ReorderItem) error {
	if m.reorderFunc != nil {
		return m.reorderFunc(ctx, items)
	}
	return nil
}

// ---------------------------------------------------------------------------
// NoUserID Tests for Example Operations
// ---------------------------------------------------------------------------

func TestService_AddExample_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, &mockExampleRepo{}, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := AddExampleInput{
		SenseID:  uuid.New(),
		Sentence: "Test sentence.",
	}

	_, err := svc.AddExample(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_UpdateExample_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, &mockExampleRepo{}, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := UpdateExampleInput{
		ExampleID: uuid.New(),
		Sentence:  "Test sentence.",
	}

	_, err := svc.UpdateExample(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_DeleteExample_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, &mockExampleRepo{}, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()

	err := svc.DeleteExample(ctx, uuid.New())

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_ReorderExamples_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, &mockExampleRepo{}, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := ReorderExamplesInput{
		SenseID: uuid.New(),
		Items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
	}

	err := svc.ReorderExamples(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteExample with Nil Sentence (regression test for panic fix)
// ---------------------------------------------------------------------------

func TestService_DeleteExample_NilSentence(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()

	exampleRepo := &mockExampleRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Example, error) {
			return &domain.Example{
				ID:       exampleID,
				SenseID:  senseID,
				Sentence: nil, // Nil sentence — previously caused panic
			}, nil
		},
		deleteFunc: func(ctx context.Context, eid uuid.UUID) error {
			return nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteExample(ctx, exampleID)

	if err != nil {
		t.Fatalf("expected no error for nil sentence, got %v", err)
	}

	// Verify audit does not contain example_deleted key when sentence is nil
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	if _, ok := auditRepo.records[0].Changes["example_deleted"]; ok {
		t.Error("expected no 'example_deleted' in audit changes when sentence is nil")
	}
}

// ---------------------------------------------------------------------------
// AddExample Tests (AE1-AE4)
// ---------------------------------------------------------------------------

func TestService_AddExample_HappyPathWithTranslation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "This is a test sentence."
	translation := "Это тестовое предложение."

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	var createdSourceSlug string
	exampleRepo := &mockExampleRepo{
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, sent string, trans *string, sourceSlug string) (*domain.Example, error) {
			createdSourceSlug = sourceSlug
			return &domain.Example{
				ID:          exampleID,
				SenseID:     sid,
				Sentence:    &sent,
				Translation: trans,
			}, nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddExampleInput{
		SenseID:     senseID,
		Sentence:    sentence,
		Translation: &translation,
	}

	example, err := svc.AddExample(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if example == nil {
		t.Fatal("expected example, got nil")
	}
	if example.ID != exampleID {
		t.Errorf("expected example ID %v, got %v", exampleID, example.ID)
	}
	if createdSourceSlug != "user" {
		t.Errorf("expected source_slug 'user', got %q", createdSourceSlug)
	}

	// Verify audit
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	audit := auditRepo.records[0]
	if audit.EntityType != domain.EntityTypeSense {
		t.Errorf("expected EntityType SENSE, got %v", audit.EntityType)
	}
	if *audit.EntityID != senseID {
		t.Errorf("expected EntityID %v, got %v", senseID, *audit.EntityID)
	}
	if audit.Action != domain.AuditActionUpdate {
		t.Errorf("expected Action UPDATE, got %v", audit.Action)
	}
	if addedChange, ok := audit.Changes["example_added"].(map[string]any); ok {
		if addedChange["new"] != sentence {
			t.Errorf("expected new sentence in audit, got %v", addedChange["new"])
		}
	} else {
		t.Error("expected 'example_added' in audit changes")
	}
}

func TestService_AddExample_WithoutTranslation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "Example without translation."

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	var receivedTranslation *string
	exampleRepo := &mockExampleRepo{
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, sent string, trans *string, sourceSlug string) (*domain.Example, error) {
			receivedTranslation = trans
			return &domain.Example{
				ID:          exampleID,
				SenseID:     sid,
				Sentence:    &sent,
				Translation: trans,
			}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddExampleInput{
		SenseID:     senseID,
		Sentence:    sentence,
		Translation: nil, // No translation
	}

	example, err := svc.AddExample(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if example == nil {
		t.Fatal("expected example, got nil")
	}
	if receivedTranslation != nil {
		t.Errorf("expected nil translation passed to repo, got %v", *receivedTranslation)
	}
}

func TestService_AddExample_LimitReached(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	exampleRepo := &mockExampleRepo{
		countBySenseFunc: func(ctx context.Context, sid uuid.UUID) (int, error) {
			return MaxExamplesPerSense, nil // Limit reached
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddExampleInput{
		SenseID:  senseID,
		Sentence: "Test sentence.",
	}

	_, err := svc.AddExample(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for limit reached, got %v", err)
	}
}

func TestService_AddExample_SenseNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, &mockExampleRepo{}, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddExampleInput{
		SenseID:  senseID,
		Sentence: "Test sentence.",
	}

	_, err := svc.AddExample(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateExample Tests (UE1-UE3)
// ---------------------------------------------------------------------------

func TestService_UpdateExample_ChangeSentence(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	oldSentence := "Old sentence."
	newSentence := "New sentence."
	translation := "Перевод."

	exampleRepo := &mockExampleRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Example, error) {
			return &domain.Example{
				ID:          exampleID,
				SenseID:     senseID,
				Sentence:    &oldSentence,
				Translation: &translation,
			}, nil
		},
		updateFunc: func(ctx context.Context, eid uuid.UUID, sent string, trans *string) (*domain.Example, error) {
			return &domain.Example{
				ID:          eid,
				SenseID:     senseID,
				Sentence:    &sent,
				Translation: trans,
			}, nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateExampleInput{
		ExampleID:   exampleID,
		Sentence:    newSentence,
		Translation: &translation,
	}

	example, err := svc.UpdateExample(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if example == nil {
		t.Fatal("expected example, got nil")
	}
	if example.Sentence == nil || *example.Sentence != newSentence {
		t.Errorf("expected sentence %q, got %v", newSentence, example.Sentence)
	}

	// Verify audit
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	audit := auditRepo.records[0]
	if audit.EntityType != domain.EntityTypeSense {
		t.Errorf("expected EntityType SENSE, got %v", audit.EntityType)
	}
	if *audit.EntityID != senseID {
		t.Errorf("expected EntityID %v (senseID), got %v", senseID, *audit.EntityID)
	}
}

func TestService_UpdateExample_RemoveTranslation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "Sentence."
	oldTranslation := "Старый перевод."

	var receivedTranslation *string
	exampleRepo := &mockExampleRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Example, error) {
			return &domain.Example{
				ID:          exampleID,
				SenseID:     senseID,
				Sentence:    &sentence,
				Translation: &oldTranslation,
			}, nil
		},
		updateFunc: func(ctx context.Context, eid uuid.UUID, sent string, trans *string) (*domain.Example, error) {
			receivedTranslation = trans
			return &domain.Example{
				ID:          eid,
				SenseID:     senseID,
				Sentence:    &sent,
				Translation: trans,
			}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateExampleInput{
		ExampleID:   exampleID,
		Sentence:    sentence,
		Translation: nil, // Remove translation
	}

	example, err := svc.UpdateExample(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if example == nil {
		t.Fatal("expected example, got nil")
	}
	if receivedTranslation != nil {
		t.Errorf("expected nil translation passed to repo, got %v", *receivedTranslation)
	}
	if example.Translation != nil {
		t.Errorf("expected nil translation in result, got %v", *example.Translation)
	}
}

func TestService_UpdateExample_ExampleNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	exampleID := uuid.New()

	exampleRepo := &mockExampleRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Example, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateExampleInput{
		ExampleID: exampleID,
		Sentence:  "New sentence.",
	}

	_, err := svc.UpdateExample(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteExample Tests (DEX1-DEX2)
// ---------------------------------------------------------------------------

func TestService_DeleteExample_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "Example to delete."

	exampleRepo := &mockExampleRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Example, error) {
			return &domain.Example{
				ID:       exampleID,
				SenseID:  senseID,
				Sentence: &sentence,
			}, nil
		},
		deleteFunc: func(ctx context.Context, eid uuid.UUID) error {
			return nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteExample(ctx, exampleID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify audit
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	audit := auditRepo.records[0]
	if audit.EntityType != domain.EntityTypeSense {
		t.Errorf("expected EntityType SENSE, got %v", audit.EntityType)
	}
	if *audit.EntityID != senseID {
		t.Errorf("expected EntityID %v (senseID), got %v", senseID, *audit.EntityID)
	}
	if audit.Action != domain.AuditActionUpdate {
		t.Errorf("expected Action UPDATE, got %v", audit.Action)
	}
	if deletedChange, ok := audit.Changes["example_deleted"].(map[string]any); ok {
		if deletedChange["old"] != sentence {
			t.Errorf("expected old sentence %q in audit, got %v", sentence, deletedChange["old"])
		}
	} else {
		t.Error("expected 'example_deleted' in audit changes")
	}
}

func TestService_DeleteExample_ExampleNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	exampleID := uuid.New()

	exampleRepo := &mockExampleRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Example, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteExample(ctx, exampleID)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ReorderExamples Tests (REX1-REX2)
// ---------------------------------------------------------------------------

func TestService_ReorderExamples_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	ex1 := uuid.New()
	ex2 := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	exampleRepo := &mockExampleRepo{
		getBySenseIDFunc: func(ctx context.Context, sid uuid.UUID) ([]domain.Example, error) {
			return []domain.Example{
				{ID: ex1, SenseID: sid},
				{ID: ex2, SenseID: sid},
			}, nil
		},
		reorderFunc: func(ctx context.Context, items []domain.ReorderItem) error {
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %d", len(items))
			}
			return nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderExamplesInput{
		SenseID: senseID,
		Items: []domain.ReorderItem{
			{ID: ex2, Position: 0},
			{ID: ex1, Position: 1},
		},
	}

	err := svc.ReorderExamples(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_ReorderExamples_ForeignExampleID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	foreignExID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	exampleRepo := &mockExampleRepo{
		getBySenseIDFunc: func(ctx context.Context, sid uuid.UUID) ([]domain.Example, error) {
			// Return examples that DON'T include foreign ID
			return []domain.Example{
				{ID: uuid.New(), SenseID: sid},
			}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderExamplesInput{
		SenseID: senseID,
		Items: []domain.ReorderItem{
			{ID: foreignExID, Position: 0},
		},
	}

	err := svc.ReorderExamples(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for foreign example ID, got %v", err)
	}
}
