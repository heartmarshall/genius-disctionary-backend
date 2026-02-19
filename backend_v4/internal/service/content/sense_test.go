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
// Test mocks (minimal, inline)
// ---------------------------------------------------------------------------

type mockEntryRepo struct {
	getByIDFunc func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
}

func (m *mockEntryRepo) GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, userID, entryID)
	}
	return nil, domain.ErrNotFound
}

type mockSenseRepo struct {
	getByIDForUserFunc func(ctx context.Context, userID, senseID uuid.UUID) (*domain.Sense, error)
	getByEntryIDFunc   func(ctx context.Context, entryID uuid.UUID) ([]domain.Sense, error)
	countByEntryFunc   func(ctx context.Context, entryID uuid.UUID) (int, error)
	createCustomFunc   func(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error)
	updateFunc         func(ctx context.Context, senseID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error)
	deleteFunc         func(ctx context.Context, senseID uuid.UUID) error
	reorderFunc        func(ctx context.Context, items []domain.ReorderItem) error
}

func (m *mockSenseRepo) GetByIDForUser(ctx context.Context, userID, senseID uuid.UUID) (*domain.Sense, error) {
	if m.getByIDForUserFunc != nil {
		return m.getByIDForUserFunc(ctx, userID, senseID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockSenseRepo) GetByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.Sense, error) {
	if m.getByEntryIDFunc != nil {
		return m.getByEntryIDFunc(ctx, entryID)
	}
	return nil, nil
}

func (m *mockSenseRepo) CountByEntry(ctx context.Context, entryID uuid.UUID) (int, error) {
	if m.countByEntryFunc != nil {
		return m.countByEntryFunc(ctx, entryID)
	}
	return 0, nil
}

func (m *mockSenseRepo) CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error) {
	if m.createCustomFunc != nil {
		return m.createCustomFunc(ctx, entryID, definition, pos, cefr, sourceSlug)
	}
	return &domain.Sense{ID: uuid.New()}, nil
}

func (m *mockSenseRepo) Update(ctx context.Context, senseID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, senseID, definition, pos, cefr)
	}
	return &domain.Sense{ID: senseID}, nil
}

func (m *mockSenseRepo) Delete(ctx context.Context, senseID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, senseID)
	}
	return nil
}

func (m *mockSenseRepo) Reorder(ctx context.Context, items []domain.ReorderItem) error {
	if m.reorderFunc != nil {
		return m.reorderFunc(ctx, items)
	}
	return nil
}

type mockTranslationRepo struct {
	getByIDForUserFunc func(ctx context.Context, userID, translationID uuid.UUID) (*domain.Translation, error)
	getBySenseIDFunc   func(ctx context.Context, senseID uuid.UUID) ([]domain.Translation, error)
	countBySenseFunc   func(ctx context.Context, senseID uuid.UUID) (int, error)
	createCustomFunc   func(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
	updateFunc         func(ctx context.Context, translationID uuid.UUID, text string) (*domain.Translation, error)
	deleteFunc         func(ctx context.Context, translationID uuid.UUID) error
	reorderFunc        func(ctx context.Context, items []domain.ReorderItem) error
}

func (m *mockTranslationRepo) GetByIDForUser(ctx context.Context, userID, translationID uuid.UUID) (*domain.Translation, error) {
	if m.getByIDForUserFunc != nil {
		return m.getByIDForUserFunc(ctx, userID, translationID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockTranslationRepo) GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Translation, error) {
	if m.getBySenseIDFunc != nil {
		return m.getBySenseIDFunc(ctx, senseID)
	}
	return nil, nil
}

func (m *mockTranslationRepo) CountBySense(ctx context.Context, senseID uuid.UUID) (int, error) {
	if m.countBySenseFunc != nil {
		return m.countBySenseFunc(ctx, senseID)
	}
	return 0, nil
}

func (m *mockTranslationRepo) CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
	if m.createCustomFunc != nil {
		return m.createCustomFunc(ctx, senseID, text, sourceSlug)
	}
	return &domain.Translation{ID: uuid.New()}, nil
}

func (m *mockTranslationRepo) Update(ctx context.Context, translationID uuid.UUID, text string) (*domain.Translation, error) {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, translationID, text)
	}
	return &domain.Translation{ID: translationID}, nil
}

func (m *mockTranslationRepo) Delete(ctx context.Context, translationID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, translationID)
	}
	return nil
}

func (m *mockTranslationRepo) Reorder(ctx context.Context, items []domain.ReorderItem) error {
	if m.reorderFunc != nil {
		return m.reorderFunc(ctx, items)
	}
	return nil
}

type mockAuditRepo struct {
	logFunc func(ctx context.Context, record domain.AuditRecord) error
	records []domain.AuditRecord
}

func (m *mockAuditRepo) Log(ctx context.Context, record domain.AuditRecord) error {
	if m.logFunc != nil {
		return m.logFunc(ctx, record)
	}
	m.records = append(m.records, record)
	return nil
}

type mockTxManager struct {
	runInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.runInTxFunc != nil {
		return m.runInTxFunc(ctx, fn)
	}
	// Default: pass-through
	return fn(ctx)
}

// ---------------------------------------------------------------------------
// AddSense Tests
// ---------------------------------------------------------------------------

func TestService_AddSense_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(
		logger,
		&mockEntryRepo{},
		&mockSenseRepo{},
		&mockTranslationRepo{},
		nil, // examples
		nil, // images
		&mockAuditRepo{},
		&mockTxManager{},
	)

	ctx := context.Background()
	input := AddSenseInput{
		EntryID:    uuid.New(),
		Definition: strPtr("test definition"),
	}

	_, err := svc.AddSense(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_AddSense_InvalidInput(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := AddSenseInput{
		EntryID: uuid.Nil, // Invalid
	}

	_, err := svc.AddSense(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_AddSense_EntryNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(logger, entryRepo, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := AddSenseInput{
		EntryID:    uuid.New(),
		Definition: strPtr("test"),
	}

	_, err := svc.AddSense(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_AddSense_EntryDeleted(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
			// Soft-deleted entries are treated as not found
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(logger, entryRepo, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := AddSenseInput{
		EntryID:    uuid.New(),
		Definition: strPtr("test"),
	}

	_, err := svc.AddSense(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for deleted entry, got %v", err)
	}
}

func TestService_AddSense_LimitReached(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entryID := uuid.New()
	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}
	senseRepo := &mockSenseRepo{
		countByEntryFunc: func(ctx context.Context, entryID uuid.UUID) (int, error) {
			return MaxSensesPerEntry, nil // Limit reached
		},
	}
	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := AddSenseInput{
		EntryID:    entryID,
		Definition: strPtr("test"),
	}

	_, err := svc.AddSense(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for limit reached, got %v", err)
	}
}

func TestService_AddSense_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			if uid != userID || eid != entryID {
				return nil, domain.ErrNotFound
			}
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	var createdSenseSourceSlug string
	senseRepo := &mockSenseRepo{
		countByEntryFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 5, nil
		},
		createCustomFunc: func(ctx context.Context, eid uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error) {
			createdSenseSourceSlug = sourceSlug
			return &domain.Sense{ID: senseID, EntryID: eid}, nil
		},
	}

	var translationCalls int
	var translationSourceSlug string
	translationRepo := &mockTranslationRepo{
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			translationCalls++
			translationSourceSlug = sourceSlug
			return &domain.Translation{ID: uuid.New(), SenseID: sid}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddSenseInput{
		EntryID:      entryID,
		Definition:   strPtr("test definition"),
		Translations: []string{"перевод 1", "перевод 2"},
	}

	sense, err := svc.AddSense(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sense == nil {
		t.Fatal("expected sense, got nil")
	}
	if sense.ID != senseID {
		t.Errorf("expected sense ID %v, got %v", senseID, sense.ID)
	}
	if createdSenseSourceSlug != "user" {
		t.Errorf("expected source_slug 'user', got %q", createdSenseSourceSlug)
	}
	if translationCalls != 2 {
		t.Errorf("expected 2 translation calls, got %d", translationCalls)
	}
	if translationSourceSlug != "user" {
		t.Errorf("expected translation source_slug 'user', got %q", translationSourceSlug)
	}
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	if auditRepo.records[0].EntityType != domain.EntityTypeSense {
		t.Errorf("expected EntityType SENSE, got %v", auditRepo.records[0].EntityType)
	}
	if auditRepo.records[0].Action != domain.AuditActionCreate {
		t.Errorf("expected Action CREATE, got %v", auditRepo.records[0].Action)
	}
}

func TestService_AddSense_NoDefinition(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	senseRepo := &mockSenseRepo{
		countByEntryFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 0, nil
		},
		createCustomFunc: func(ctx context.Context, eid uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error) {
			if definition != nil {
				t.Errorf("expected nil definition, got %v", *definition)
			}
			return &domain.Sense{ID: uuid.New(), EntryID: eid}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddSenseInput{
		EntryID:    entryID,
		Definition: nil, // No definition
	}

	_, err := svc.AddSense(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_AddSense_NoTranslations(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: userID}, nil
		},
	}

	senseRepo := &mockSenseRepo{
		countByEntryFunc: func(ctx context.Context, eid uuid.UUID) (int, error) {
			return 0, nil
		},
		createCustomFunc: func(ctx context.Context, eid uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error) {
			return &domain.Sense{ID: uuid.New(), EntryID: eid}, nil
		},
	}

	translationCallCount := 0
	translationRepo := &mockTranslationRepo{
		createCustomFunc: func(ctx context.Context, sid uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
			translationCallCount++
			return &domain.Translation{ID: uuid.New()}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, entryRepo, senseRepo, translationRepo, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := AddSenseInput{
		EntryID:      entryID,
		Definition:   strPtr("test"),
		Translations: []string{}, // Empty
	}

	_, err := svc.AddSense(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if translationCallCount != 0 {
		t.Errorf("expected 0 translation calls, got %d", translationCallCount)
	}
}

// ---------------------------------------------------------------------------
// UpdateSense Tests
// ---------------------------------------------------------------------------

func TestService_UpdateSense_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := UpdateSenseInput{
		SenseID:    uuid.New(),
		Definition: strPtr("new definition"),
	}

	_, err := svc.UpdateSense(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_UpdateSense_InvalidInput(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := UpdateSenseInput{
		SenseID: uuid.Nil, // Invalid
	}

	_, err := svc.UpdateSense(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_UpdateSense_SenseNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, userID, senseID uuid.UUID) (*domain.Sense, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := UpdateSenseInput{
		SenseID:    uuid.New(),
		Definition: strPtr("new"),
	}

	_, err := svc.UpdateSense(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_UpdateSense_SenseFromForeignEntry(t *testing.T) {
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

	ctx := withUser(context.Background(), userID)
	input := UpdateSenseInput{
		SenseID:    senseID,
		Definition: strPtr("new"),
	}

	// This should succeed since userID matches
	_, err := svc.UpdateSense(ctx, input)
	if err != nil {
		t.Fatalf("expected no error for own sense, got %v", err)
	}

	// Different user should get ErrNotFound
	otherCtx := withUser(context.Background(), uuid.New())
	_, err = svc.UpdateSense(otherCtx, input)
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound for foreign sense, got %v", err)
	}
}

func TestService_UpdateSense_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()

	oldDef := "old definition"
	oldPOS := domain.PartOfSpeechNoun
	newDef := "new definition"

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{
				ID:           senseID,
				EntryID:      entryID,
				Definition:   &oldDef,
				PartOfSpeech: &oldPOS,
			}, nil
		},
		updateFunc: func(ctx context.Context, sid uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error) {
			return &domain.Sense{
				ID:           sid,
				Definition:   definition,
				PartOfSpeech: pos,
			}, nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateSenseInput{
		SenseID:    senseID,
		Definition: &newDef,
	}

	sense, err := svc.UpdateSense(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sense == nil {
		t.Fatal("expected sense, got nil")
	}
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	if auditRepo.records[0].EntityType != domain.EntityTypeSense {
		t.Errorf("expected EntityType SENSE, got %v", auditRepo.records[0].EntityType)
	}
	if auditRepo.records[0].Action != domain.AuditActionUpdate {
		t.Errorf("expected Action UPDATE, got %v", auditRepo.records[0].Action)
	}
}

func TestService_UpdateSense_PartialUpdate(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()

	oldDef := "old definition"
	oldPOS := domain.PartOfSpeechNoun

	var updateCallDef *string
	var updateCallPOS *domain.PartOfSpeech

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{
				ID:           senseID,
				EntryID:      entryID,
				Definition:   &oldDef,
				PartOfSpeech: &oldPOS,
			}, nil
		},
		updateFunc: func(ctx context.Context, sid uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error) {
			updateCallDef = definition
			updateCallPOS = pos
			return &domain.Sense{ID: sid}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateSenseInput{
		SenseID:      senseID,
		PartOfSpeech: nil, // Don't change
	}

	_, err := svc.UpdateSense(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if updateCallDef != nil {
		t.Errorf("expected nil definition (no change), got %v", *updateCallDef)
	}
	if updateCallPOS != nil {
		t.Errorf("expected nil pos (no change), got %v", *updateCallPOS)
	}
}

func TestService_UpdateSense_AllFieldsNil(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()

	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{ID: senseID, EntryID: entryID}, nil
		},
		updateFunc: func(ctx context.Context, sid uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error) {
			return &domain.Sense{ID: sid}, nil
		},
	}

	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := UpdateSenseInput{
		SenseID:      senseID,
		Definition:   nil,
		PartOfSpeech: nil,
		CEFRLevel:    nil,
	}

	_, err := svc.UpdateSense(ctx, input)

	if err != nil {
		t.Fatalf("expected no error for all-nil update, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteSense Tests
// ---------------------------------------------------------------------------

func TestService_DeleteSense_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()

	err := svc.DeleteSense(ctx, uuid.New())

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_DeleteSense_SenseNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())

	err := svc.DeleteSense(ctx, uuid.New())

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_DeleteSense_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	senseID := uuid.New()

	def := "test definition"
	senseRepo := &mockSenseRepo{
		getByIDForUserFunc: func(ctx context.Context, uid, sid uuid.UUID) (*domain.Sense, error) {
			return &domain.Sense{
				ID:         senseID,
				EntryID:    entryID,
				Definition: &def,
			}, nil
		},
		deleteFunc: func(ctx context.Context, sid uuid.UUID) error {
			return nil
		},
	}

	auditRepo := &mockAuditRepo{}
	svc := NewService(logger, &mockEntryRepo{}, senseRepo, &mockTranslationRepo{}, nil, nil, auditRepo, &mockTxManager{})

	ctx := withUser(context.Background(), userID)

	err := svc.DeleteSense(ctx, senseID)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(auditRepo.records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(auditRepo.records))
	}
	if auditRepo.records[0].Action != domain.AuditActionDelete {
		t.Errorf("expected Action DELETE, got %v", auditRepo.records[0].Action)
	}
	changes := auditRepo.records[0].Changes
	if changes == nil {
		t.Fatal("expected changes in audit record")
	}
	if defChange, ok := changes["definition"].(map[string]any); ok {
		if defChange["old"] != def {
			t.Errorf("expected old definition %q in audit, got %v", def, defChange["old"])
		}
	}
}

// ---------------------------------------------------------------------------
// ReorderSenses Tests
// ---------------------------------------------------------------------------

func TestService_ReorderSenses_NoUserID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := context.Background()
	input := ReorderSensesInput{
		EntryID: uuid.New(),
		Items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
	}

	err := svc.ReorderSenses(ctx, input)

	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestService_ReorderSenses_InvalidInput(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewService(logger, &mockEntryRepo{}, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := ReorderSensesInput{
		EntryID: uuid.Nil, // Invalid
		Items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
	}

	err := svc.ReorderSenses(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

func TestService_ReorderSenses_EntryNotFound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := NewService(logger, entryRepo, &mockSenseRepo{}, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), uuid.New())
	input := ReorderSensesInput{
		EntryID: uuid.New(),
		Items:   []domain.ReorderItem{{ID: uuid.New(), Position: 0}},
	}

	err := svc.ReorderSenses(ctx, input)

	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_ReorderSenses_ForeignSenseID(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	foreignSenseID := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: uid}, nil
		},
	}

	senseRepo := &mockSenseRepo{
		getByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) ([]domain.Sense, error) {
			// Return senses that DON'T include the foreign ID
			return []domain.Sense{
				{ID: uuid.New(), EntryID: eid},
			}, nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderSensesInput{
		EntryID: entryID,
		Items: []domain.ReorderItem{
			{ID: foreignSenseID, Position: 0},
		},
	}

	err := svc.ReorderSenses(ctx, input)

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for foreign sense ID, got %v", err)
	}
}

func TestService_ReorderSenses_HappyPath(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	sense1 := uuid.New()
	sense2 := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: uid}, nil
		},
	}

	senseRepo := &mockSenseRepo{
		getByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) ([]domain.Sense, error) {
			return []domain.Sense{
				{ID: sense1, EntryID: eid},
				{ID: sense2, EntryID: eid},
			}, nil
		},
		reorderFunc: func(ctx context.Context, items []domain.ReorderItem) error {
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %d", len(items))
			}
			return nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderSensesInput{
		EntryID: entryID,
		Items: []domain.ReorderItem{
			{ID: sense2, Position: 0},
			{ID: sense1, Position: 1},
		},
	}

	err := svc.ReorderSenses(ctx, input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestService_ReorderSenses_PartialList(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	userID := uuid.New()
	entryID := uuid.New()
	sense1 := uuid.New()
	sense2 := uuid.New()
	sense3 := uuid.New()

	entryRepo := &mockEntryRepo{
		getByIDFunc: func(ctx context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, UserID: uid}, nil
		},
	}

	senseRepo := &mockSenseRepo{
		getByEntryIDFunc: func(ctx context.Context, eid uuid.UUID) ([]domain.Sense, error) {
			return []domain.Sense{
				{ID: sense1, EntryID: eid},
				{ID: sense2, EntryID: eid},
				{ID: sense3, EntryID: eid},
			}, nil
		},
		reorderFunc: func(ctx context.Context, items []domain.ReorderItem) error {
			return nil
		},
	}

	svc := NewService(logger, entryRepo, senseRepo, &mockTranslationRepo{}, nil, nil, &mockAuditRepo{}, &mockTxManager{})

	ctx := withUser(context.Background(), userID)
	input := ReorderSensesInput{
		EntryID: entryID,
		Items: []domain.ReorderItem{
			{ID: sense1, Position: 0},
			{ID: sense3, Position: 1},
			// sense2 not included - partial reorder is allowed
		},
	}

	err := svc.ReorderSenses(ctx, input)

	if err != nil {
		t.Fatalf("expected no error for partial reorder, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Reorder Validation Edge Cases
// ---------------------------------------------------------------------------

func TestValidation_ReorderSenses_DuplicateIDs(t *testing.T) {
	t.Parallel()

	id := uuid.New()
	input := ReorderSensesInput{
		EntryID: uuid.New(),
		Items: []domain.ReorderItem{
			{ID: id, Position: 0},
			{ID: id, Position: 1}, // Duplicate
		},
	}

	err := input.Validate()

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for duplicate IDs, got %v", err)
	}
}

func TestValidation_ReorderSenses_NegativePosition(t *testing.T) {
	t.Parallel()

	input := ReorderSensesInput{
		EntryID: uuid.New(),
		Items: []domain.ReorderItem{
			{ID: uuid.New(), Position: -1}, // Invalid
		},
	}

	err := input.Validate()

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for negative position, got %v", err)
	}
}

func TestValidation_AddSense_TooManyTranslations(t *testing.T) {
	t.Parallel()

	translations := make([]string, 21)
	for i := range translations {
		translations[i] = "перевод"
	}

	input := AddSenseInput{
		EntryID:      uuid.New(),
		Translations: translations,
	}

	err := input.Validate()

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for too many translations, got %v", err)
	}
}

func TestValidation_AddSense_EmptyTranslationString(t *testing.T) {
	t.Parallel()

	input := AddSenseInput{
		EntryID:      uuid.New(),
		Translations: []string{"valid", "  ", "also valid"},
	}

	err := input.Validate()

	if !errors.Is(err, domain.ErrValidation) {
		t.Errorf("expected ErrValidation for empty translation, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// buildSenseChanges Tests
// ---------------------------------------------------------------------------

func TestBuildSenseChanges_DefinitionChanged(t *testing.T) {
	t.Parallel()

	oldDef := "old definition"
	newDef := "new definition"

	old := &domain.Sense{Definition: &oldDef}
	input := &UpdateSenseInput{SenseID: uuid.New(), Definition: &newDef}

	changes := buildSenseChanges(old, input)

	defChange, ok := changes["definition"].(map[string]any)
	if !ok {
		t.Fatal("expected 'definition' in changes")
	}
	if defChange["old"] != oldDef {
		t.Errorf("expected old=%q, got %v", oldDef, defChange["old"])
	}
	if defChange["new"] != newDef {
		t.Errorf("expected new=%q, got %v", newDef, defChange["new"])
	}
}

func TestBuildSenseChanges_PartOfSpeechChanged(t *testing.T) {
	t.Parallel()

	oldPOS := domain.PartOfSpeechNoun
	newPOS := domain.PartOfSpeechVerb

	old := &domain.Sense{PartOfSpeech: &oldPOS}
	input := &UpdateSenseInput{SenseID: uuid.New(), PartOfSpeech: &newPOS}

	changes := buildSenseChanges(old, input)

	posChange, ok := changes["part_of_speech"].(map[string]any)
	if !ok {
		t.Fatal("expected 'part_of_speech' in changes")
	}
	if posChange["old"] != string(oldPOS) {
		t.Errorf("expected old=%q, got %v", oldPOS, posChange["old"])
	}
	if posChange["new"] != string(newPOS) {
		t.Errorf("expected new=%q, got %v", newPOS, posChange["new"])
	}
}

func TestBuildSenseChanges_CEFRLevelChanged(t *testing.T) {
	t.Parallel()

	oldCEFR := "A1"
	newCEFR := "C2"

	old := &domain.Sense{CEFRLevel: &oldCEFR}
	input := &UpdateSenseInput{SenseID: uuid.New(), CEFRLevel: &newCEFR}

	changes := buildSenseChanges(old, input)

	cefrChange, ok := changes["cefr_level"].(map[string]any)
	if !ok {
		t.Fatal("expected 'cefr_level' in changes")
	}
	if cefrChange["old"] != oldCEFR {
		t.Errorf("expected old=%q, got %v", oldCEFR, cefrChange["old"])
	}
	if cefrChange["new"] != newCEFR {
		t.Errorf("expected new=%q, got %v", newCEFR, cefrChange["new"])
	}
}

func TestBuildSenseChanges_NoChanges(t *testing.T) {
	t.Parallel()

	def := "same definition"
	pos := domain.PartOfSpeechNoun
	cefr := "B1"

	old := &domain.Sense{
		Definition:   &def,
		PartOfSpeech: &pos,
		CEFRLevel:    &cefr,
	}
	input := &UpdateSenseInput{
		SenseID:      uuid.New(),
		Definition:   &def,
		PartOfSpeech: &pos,
		CEFRLevel:    &cefr,
	}

	changes := buildSenseChanges(old, input)

	if len(changes) != 0 {
		t.Errorf("expected empty changes for identical values, got %v", changes)
	}
}

func TestBuildSenseChanges_NilFieldsNotTracked(t *testing.T) {
	t.Parallel()

	def := "old definition"
	old := &domain.Sense{Definition: &def}
	input := &UpdateSenseInput{
		SenseID:      uuid.New(),
		Definition:   nil, // Not changing
		PartOfSpeech: nil,
		CEFRLevel:    nil,
	}

	changes := buildSenseChanges(old, input)

	if len(changes) != 0 {
		t.Errorf("expected empty changes for nil input fields, got %v", changes)
	}
}

func TestBuildSenseChanges_OldNilNewSet(t *testing.T) {
	t.Parallel()

	newDef := "brand new"
	newPOS := domain.PartOfSpeechAdverb
	newCEFR := "C1"

	old := &domain.Sense{
		Definition:   nil,
		PartOfSpeech: nil,
		CEFRLevel:    nil,
	}
	input := &UpdateSenseInput{
		SenseID:      uuid.New(),
		Definition:   &newDef,
		PartOfSpeech: &newPOS,
		CEFRLevel:    &newCEFR,
	}

	changes := buildSenseChanges(old, input)

	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d: %v", len(changes), changes)
	}

	// Definition: old was nil → treated as ""
	if defChange, ok := changes["definition"].(map[string]any); ok {
		if defChange["old"] != "" {
			t.Errorf("expected old definition '', got %v", defChange["old"])
		}
		if defChange["new"] != newDef {
			t.Errorf("expected new definition %q, got %v", newDef, defChange["new"])
		}
	} else {
		t.Error("expected 'definition' in changes")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func strPtr(s string) *string {
	return &s
}

func withUser(ctx context.Context, userID uuid.UUID) context.Context {
	return ctxutil.WithUserID(ctx, userID)
}
