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
// Mock extensions for translation operations
// ---------------------------------------------------------------------------

type mockExampleRepo struct {
	getByIDFunc      func(ctx context.Context, exampleID uuid.UUID) (*domain.Example, error)
	getBySenseIDFunc func(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error)
	countBySenseFunc func(ctx context.Context, senseID uuid.UUID) (int, error)
	createCustomFunc func(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
	updateFunc       func(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error)
	deleteFunc       func(ctx context.Context, exampleID uuid.UUID) error
	reorderFunc      func(ctx context.Context, items []domain.ReorderItem) error
}

func (m *mockExampleRepo) GetByID(ctx context.Context, exampleID uuid.UUID) (*domain.Example, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, exampleID)
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
// AddTranslation Tests (AT1-AT4)
// ---------------------------------------------------------------------------

func TestService_AddTranslation_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			if sid != senseID {
				return nil, domain.ErrNotFound
			}
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			if uid != userID || eid != entryID {
				return nil, domain.ErrNotFound
			}
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	var createdSourceSlug string
	translationRepo := &mockTranslationRepo{
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			createdSourceSlug = sourceSlug
			return &domain.Translation{
				ID:      translationID,
				SenseID: sid,
				Text:    &text,
			}, nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddTranslationInput{
		SenseID: senseID,
		Text:    "перевод",
	}

	translation, err := svc.AddTranslation(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if translation == nil {
		t.Fatal("expected translation, got nil")
	}
	if translation.ID != translationID {
		t.Errorf("expected translation ID %v, got %v", translationID, translation.ID)
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
	if audit.Changes == nil {
		t.Fatal("expected Changes in audit record")
	}
	if addedChange, ok := audit.Changes["translation_added"].(map[string]any); ok {
		if addedChange["new"] != "перевод" {
			t.Errorf("expected new translation text in audit, got %v", addedChange["new"])
		}
	} else {
		t.Error("expected 'translation_added' in audit changes")
	}
}

func TestService_AddTranslation_LimitReached(t *testing.T) {
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

	translationRepo := &mockTranslationRepo{
		countBySenseFunc: func(ctx context.Context, sid uuid.UUID) (int, error) {
			return MaxTranslationsPerSense, nil // Limit reached
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddTranslationInput{
		SenseID: senseID,
		Text:    "перевод",
	}

	_, err := svc.AddTranslation(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for limit reached, got %v", err)
	}
}

func TestService_AddTranslation_SenseNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddTranslationInput{
		SenseID: senseID,
		Text:    "перевод",
	}

	_, err := svc.AddTranslation(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_AddTranslation_SenseFromForeignEntry(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	foreignUserID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDFunc: func(ctx context.Context, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
	}

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			// Entry belongs to different user
			if uid == foreignUserID {
				return &domain.Entry{ID: entryID, UserID: foreignUserID}, nil
			}
			return nil, domain.ErrNotFound
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddTranslationInput{
		SenseID: senseID,
		Text:    "перевод",
	}

	_, err := svc.AddTranslation(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for sense from foreign entry, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateTranslation Tests (UT1-UT3)
// ---------------------------------------------------------------------------

func TestService_UpdateTranslation_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()
	oldText := "старый перевод"
	newText := "новый перевод"

	translationRepo := &mockTranslationRepo{
		getByIDFunc: func(ctx context.Context, tid uuid.UUID) (*domain.Translation, error) {
			if tid != translationID {
				return nil, domain.ErrNotFound
			}
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &oldText,
			}, nil
		},
		updateFunc: func(ctx context.Context, tid uuid.UUID, text string) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      tid,
				SenseID: senseID,
				Text:    &text,
			}, nil
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateTranslationInput{
		TranslationID: translationID,
		Text:          newText,
	}

	translation, err := svc.UpdateTranslation(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if translation == nil {
		t.Fatal("expected translation, got nil")
	}
	if translation.Text == nil || *translation.Text != newText {
		t.Errorf("expected text %q, got %v", newText, translation.Text)
	}

	// Verify audit contains old and new text
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
	if textChange, ok := audit.Changes["translation_text"].(map[string]any); ok {
		if textChange["old"] != oldText {
			t.Errorf("expected old text %q in audit, got %v", oldText, textChange["old"])
		}
		if textChange["new"] != newText {
			t.Errorf("expected new text %q in audit, got %v", newText, textChange["new"])
		}
	} else {
		t.Error("expected 'translation_text' in audit changes")
	}
}

func TestService_UpdateTranslation_TranslationNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	translationID := uuid.New()

	translationRepo := &mockTranslationRepo{
		getByIDFunc: func(ctx context.Context, tid uuid.UUID) (*domain.Translation, error) {
			return nil, domain.ErrNotFound
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateTranslationInput{
		TranslationID: translationID,
		Text:          "новый перевод",
	}

	_, err := svc.UpdateTranslation(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateTranslation_AuditContainsOldAndNew(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()
	oldText := "old"
	newText := "new"

	translationRepo := &mockTranslationRepo{
		getByIDFunc: func(ctx context.Context, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &oldText,
			}, nil
		},
		updateFunc: func(ctx context.Context, tid uuid.UUID, text string) (*domain.Translation, error) {
			return &domain.Translation{ID: tid, Text: &text}, nil
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateTranslationInput{
		TranslationID: translationID,
		Text:          newText,
	}

	_, err := svc.UpdateTranslation(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify audit structure
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	changes := auditRepo.records[0].Changes
	if textChange, ok := changes["translation_text"].(map[string]any); ok {
		if textChange["old"] != oldText || textChange["new"] != newText {
			t.Errorf("expected old=%q new=%q, got old=%v new=%v", oldText, newText, textChange["old"], textChange["new"])
		}
	} else {
		t.Error("expected 'translation_text' in changes")
	}
}

// ---------------------------------------------------------------------------
// DeleteTranslation Tests (DT1-DT3)
// ---------------------------------------------------------------------------

func TestService_DeleteTranslation_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()
	text := "перевод для удаления"

	translationRepo := &mockTranslationRepo{
		getByIDFunc: func(ctx context.Context, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &text,
			}, nil
		},
		deleteFunc: func(ctx context.Context, tid uuid.UUID) error {
			return nil
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteTranslation(ctx, translationID)

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
	if deletedChange, ok := audit.Changes["translation_deleted"].(map[string]any); ok {
		if deletedChange["old"] != text {
			t.Errorf("expected old text %q in audit, got %v", text, deletedChange["old"])
		}
	} else {
		t.Error("expected 'translation_deleted' in audit changes")
	}
}

func TestService_DeleteTranslation_LastTranslationAllowed(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()
	lastText := "последний перевод"

	translationRepo := &mockTranslationRepo{
		getByIDFunc: func(ctx context.Context, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &lastText,
			}, nil
		},
		deleteFunc: func(ctx context.Context, tid uuid.UUID) error {
			return nil
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
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

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteTranslation(ctx, translationID)

	if err != nil {
		t.Errorf("expected no error for deleting last translation, got %v", err)
	}
}

func TestService_DeleteTranslation_TranslationNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	translationID := uuid.New()

	translationRepo := &mockTranslationRepo{
		getByIDFunc: func(ctx context.Context, tid uuid.UUID) (*domain.Translation, error) {
			return nil, domain.ErrNotFound
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteTranslation(ctx, translationID)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ReorderTranslations Tests (RT1-RT2)
// ---------------------------------------------------------------------------

func TestService_ReorderTranslations_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	tr1 := uuid.New()
	tr2 := uuid.New()

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

	translationRepo := &mockTranslationRepo{
		getBySenseIDFunc: func(ctx context.Context, sid uuid.UUID) ([]domain.Translation, error) {
			return []domain.Translation{
				{ID: tr1, SenseID: sid},
				{ID: tr2, SenseID: sid},
			}, nil
		},
		reorderFunc: func(ctx context.Context, items []domain.ReorderItem) error {
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %d", len(items))
			}
			return nil
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderTranslationsInput{
		SenseID: senseID,
		Items: []domain.ReorderItem{
			{ID: tr2, Position: 0},
			{ID: tr1, Position: 1},
		},
	}

	err := svc.ReorderTranslations(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_ReorderTranslations_ForeignTranslationID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()
	foreignTrID := uuid.New()

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

	translationRepo := &mockTranslationRepo{
		getBySenseIDFunc: func(ctx context.Context, sid uuid.UUID) ([]domain.Translation, error) {
			// Return translations that DON'T include foreign ID
			return []domain.Translation{
				{ID: uuid.New(), SenseID: sid},
			}, nil
		},
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			return &domain.Translation{ID: uuid.New()}, nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderTranslationsInput{
		SenseID: senseID,
		Items: []domain.ReorderItem{
			{ID: foreignTrID, Position: 0},
		},
	}

	err := svc.ReorderTranslations(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for foreign translation ID, got %v", err)
	}
}
