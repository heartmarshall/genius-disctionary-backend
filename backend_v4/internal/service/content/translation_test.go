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
// NoUserID Tests for Translation Operations
// ---------------------------------------------------------------------------

func TestService_AddTranslation_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := AddTranslationInput{
		SenseID: uuid.New(),
		Text:    "перевод",
	}

	_, err := svc.AddTranslation(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_UpdateTranslation_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := UpdateTranslationInput{
		TranslationID: uuid.New(),
		Text:          "перевод",
	}

	_, err := svc.UpdateTranslation(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_DeleteTranslation_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()

	err := svc.DeleteTranslation(ctx, uuid.New())

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_ReorderTranslations_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := ReorderTranslationsInput{
		SenseID: uuid.New(),
		Items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
	}

	err := svc.ReorderTranslations(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateTranslation No-Op Test
// ---------------------------------------------------------------------------

func TestService_UpdateTranslation_NoChangeSkipsAudit(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()
	sameText := "одинаковый текст"

	translationRepo := &mockTranslationRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &sameText,
			}, nil
		},
		updateFunc: func(ctx context.Context, tid uuid.UUID, text string) (*domain.Translation, error) {
			return &domain.Translation{ID: tid, SenseID: senseID, Text: &text}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateTranslationInput{
		TranslationID: translationID,
		Text:          sameText,
	}

	_, err := svc.UpdateTranslation(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// No audit when nothing changed
	if len(auditRepo.records) != 0 {
		t.Errorf("expected 0 audit records for no-change update, got %d", len(auditRepo.records))
	}
}

// ---------------------------------------------------------------------------
// AddTranslation Tests (AT1-AT4)
// ---------------------------------------------------------------------------

func TestService_AddTranslation_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	senseID := uuid.New()
	translationID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			if uid != userID || sid != senseID {
				return nil, domain.ErrNotFound
			}
			return &domain.Sense{ID: senseID}, nil
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

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

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
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	translationRepo := &mockTranslationRepo{
		countBySenseFunc: func(ctx context.Context, sid uuid.UUID) (int, error) {
			return MaxTranslationsPerSense, nil // Limit reached
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

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
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
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
	senseID := uuid.New()

	// GetByIDForUser returns ErrNotFound for foreign user (JOIN-based ownership)
	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			if uid != userID {
				return nil, domain.ErrNotFound
			}
			return &domain.Sense{ID: senseID}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	// Different user should get ErrNotFound
	otherCtx := withUser(context.Background(), uuid.New())
	input := AddTranslationInput{
		SenseID: senseID,
		Text:    "перевод",
	}

	_, err := svc.AddTranslation(otherCtx, input)

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
	senseID := uuid.New()
	translationID := uuid.New()
	oldText := "старый перевод"
	newText := "новый перевод"

	translationRepo := &mockTranslationRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			if uid != userID || tid != translationID {
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
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, auditRepo, &mockTxManager{})

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
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			return nil, domain.ErrNotFound
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
	senseID := uuid.New()
	translationID := uuid.New()
	oldText := "old"
	newText := "new"

	translationRepo := &mockTranslationRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &oldText,
			}, nil
		},
		updateFunc: func(ctx context.Context, tid uuid.UUID, text string) (*domain.Translation, error) {
			return &domain.Translation{ID: tid, Text: &text}, nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, auditRepo, &mockTxManager{})

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
	senseID := uuid.New()
	translationID := uuid.New()
	text := "перевод для удаления"

	translationRepo := &mockTranslationRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &text,
			}, nil
		},
		deleteFunc: func(ctx context.Context, tid uuid.UUID) error {
			return nil
		},
	}

	auditRepo := &mockAuditRepo{}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, auditRepo, &mockTxManager{})

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
	senseID := uuid.New()
	translationID := uuid.New()
	lastText := "последний перевод"

	translationRepo := &mockTranslationRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			return &domain.Translation{
				ID:      translationID,
				SenseID: senseID,
				Text:    &lastText,
			}, nil
		},
		deleteFunc: func(ctx context.Context, tid uuid.UUID) error {
			return nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

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
		getByIDForUserFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Translation, error) {
			return nil, domain.ErrNotFound
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
	senseID := uuid.New()
	tr1 := uuid.New()
	tr2 := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
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
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

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
	senseID := uuid.New()
	foreignTrID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID}, nil
		},
	}

	translationRepo := &mockTranslationRepo{
		getBySenseIDFunc: func(ctx context.Context, sid uuid.UUID) ([]domain.Translation, error) {
			// Return translations that DON'T include foreign ID
			return []domain.Translation{
				{ID: uuid.New(), SenseID: sid},
			}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, translationRepo, nil, nil, &mockAuditRepo{}, &mockTxManager{})

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
