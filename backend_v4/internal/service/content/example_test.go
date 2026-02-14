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
// AddExample Tests (AE1-AE4)
// ---------------------------------------------------------------------------

func TestService_AddExample_HappyPathWithTranslation(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "This is a test sentence."
	translation := "Это тестовое предложение."

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

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
	entryID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "Example without translation."

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

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
	entryID := uuid.New()
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	exampleRepo := &mockExampleRepo{
		countBySenseFunc: func(ctx context.Context, sid uuid.UUID) (int, error) {
			return MaxExamplesPerSense, nil // Limit reached
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

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
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
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
	entryID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	oldSentence := "Old sentence."
	newSentence := "New sentence."
	translation := "Перевод."

	exampleRepo := &mockExampleRepo{
		getByIDFunc: func(ctx context.Context, eid uuid.UUID) (*domain.Example, error) {
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

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

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
	entryID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "Sentence."
	oldTranslation := "Старый перевод."

	var receivedTranslation *string
	exampleRepo := &mockExampleRepo{
		getByIDFunc: func(ctx context.Context, eid uuid.UUID) (*domain.Example, error) {
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

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

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
		getByIDFunc: func(ctx context.Context, eid uuid.UUID) (*domain.Example, error) {
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
	entryID := uuid.New()
	senseID := uuid.New()
	exampleID := uuid.New()
	sentence := "Example to delete."

	exampleRepo := &mockExampleRepo{
		getByIDFunc: func(ctx context.Context, eid uuid.UUID) (*domain.Example, error) {
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

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, auditRepo, &mockTxManager{})

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
		getByIDFunc: func(ctx context.Context, eid uuid.UUID) (*domain.Example, error) {
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
	entryID := uuid.New()
	senseID := uuid.New()
	ex1 := uuid.New()
	ex2 := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	exampleRepo := &mockExampleRepo{
		getBySenseIDFunc: func(ctx context.Context, sid uuid.UUID) ([]domain.Example, error) {
			return []domain.Example{
				{ID: ex1, SenseID: sid},
				{ID: ex2, SenseID: sid},
			}, nil
		},
		reorderFunc: func(ctx context.Context, items []ReorderItem) error {
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %d", len(items))
			}
			return nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderExamplesInput{
		SenseID: senseID,
		Items: []ReorderItem{
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
	entryID := uuid.New()
	senseID := uuid.New()
	foreignExID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, exampleRepo, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderExamplesInput{
		SenseID: senseID,
		Items: []ReorderItem{
			{ID: foreignExID, Position: 0},
		},
	}

	err := svc.ReorderExamples(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for foreign example ID, got %v", err)
	}
}
