package dictionary

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ===========================================================================
// Manual mocks (moq-style with func fields)
// ===========================================================================

type mockEntryRepo struct {
	GetByIDFunc     func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	GetByTextFunc   func(ctx context.Context, userID uuid.UUID, textNormalized string) (*domain.Entry, error)
	GetByIDsFunc    func(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error)
	FindFunc        func(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, int, error)
	FindCursorFunc  func(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, bool, error)
	FindDeletedFunc func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Entry, int, error)
	CountByUserFunc func(ctx context.Context, userID uuid.UUID) (int, error)
	CreateFunc      func(ctx context.Context, entry *domain.Entry) (*domain.Entry, error)
	UpdateNotesFunc func(ctx context.Context, userID, entryID uuid.UUID, notes *string) (*domain.Entry, error)
	SoftDeleteFunc  func(ctx context.Context, userID, entryID uuid.UUID) error
	RestoreFunc     func(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	HardDeleteOldFunc func(ctx context.Context, threshold time.Time) (int64, error)
}

func (m *mockEntryRepo) GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, userID, entryID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockEntryRepo) GetByText(ctx context.Context, userID uuid.UUID, textNormalized string) (*domain.Entry, error) {
	if m.GetByTextFunc != nil {
		return m.GetByTextFunc(ctx, userID, textNormalized)
	}
	return nil, domain.ErrNotFound
}

func (m *mockEntryRepo) GetByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error) {
	if m.GetByIDsFunc != nil {
		return m.GetByIDsFunc(ctx, userID, ids)
	}
	return nil, nil
}

func (m *mockEntryRepo) Find(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, int, error) {
	if m.FindFunc != nil {
		return m.FindFunc(ctx, userID, filter)
	}
	return nil, 0, nil
}

func (m *mockEntryRepo) FindCursor(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, bool, error) {
	if m.FindCursorFunc != nil {
		return m.FindCursorFunc(ctx, userID, filter)
	}
	return nil, false, nil
}

func (m *mockEntryRepo) FindDeleted(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Entry, int, error) {
	if m.FindDeletedFunc != nil {
		return m.FindDeletedFunc(ctx, userID, limit, offset)
	}
	return nil, 0, nil
}

func (m *mockEntryRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.CountByUserFunc != nil {
		return m.CountByUserFunc(ctx, userID)
	}
	return 0, nil
}

func (m *mockEntryRepo) Create(ctx context.Context, entry *domain.Entry) (*domain.Entry, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, entry)
	}
	entry.ID = uuid.New()
	return entry, nil
}

func (m *mockEntryRepo) UpdateNotes(ctx context.Context, userID, entryID uuid.UUID, notes *string) (*domain.Entry, error) {
	if m.UpdateNotesFunc != nil {
		return m.UpdateNotesFunc(ctx, userID, entryID, notes)
	}
	return nil, nil
}

func (m *mockEntryRepo) SoftDelete(ctx context.Context, userID, entryID uuid.UUID) error {
	if m.SoftDeleteFunc != nil {
		return m.SoftDeleteFunc(ctx, userID, entryID)
	}
	return nil
}

func (m *mockEntryRepo) Restore(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
	if m.RestoreFunc != nil {
		return m.RestoreFunc(ctx, userID, entryID)
	}
	return nil, nil
}

func (m *mockEntryRepo) HardDeleteOld(ctx context.Context, threshold time.Time) (int64, error) {
	return 0, nil
}

type mockSenseRepo struct {
	GetByEntryIDsFunc func(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Sense, error)
	CreateFromRefFunc func(ctx context.Context, entryID, refSenseID uuid.UUID, sourceSlug string) (*domain.Sense, error)
	CreateCustomFunc  func(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error)
}

func (m *mockSenseRepo) GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Sense, error) {
	if m.GetByEntryIDsFunc != nil {
		return m.GetByEntryIDsFunc(ctx, entryIDs)
	}
	return nil, nil
}

func (m *mockSenseRepo) CreateFromRef(ctx context.Context, entryID, refSenseID uuid.UUID, sourceSlug string) (*domain.Sense, error) {
	if m.CreateFromRefFunc != nil {
		return m.CreateFromRefFunc(ctx, entryID, refSenseID, sourceSlug)
	}
	return &domain.Sense{ID: uuid.New(), EntryID: entryID}, nil
}

func (m *mockSenseRepo) CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error) {
	if m.CreateCustomFunc != nil {
		return m.CreateCustomFunc(ctx, entryID, definition, pos, cefr, sourceSlug)
	}
	return &domain.Sense{ID: uuid.New(), EntryID: entryID}, nil
}

type mockTranslationRepo struct {
	GetBySenseIDsFunc func(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Translation, error)
	CreateFromRefFunc func(ctx context.Context, senseID, refTranslationID uuid.UUID, sourceSlug string) (*domain.Translation, error)
	CreateCustomFunc  func(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
}

func (m *mockTranslationRepo) GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Translation, error) {
	if m.GetBySenseIDsFunc != nil {
		return m.GetBySenseIDsFunc(ctx, senseIDs)
	}
	return nil, nil
}

func (m *mockTranslationRepo) CreateFromRef(ctx context.Context, senseID, refTranslationID uuid.UUID, sourceSlug string) (*domain.Translation, error) {
	if m.CreateFromRefFunc != nil {
		return m.CreateFromRefFunc(ctx, senseID, refTranslationID, sourceSlug)
	}
	return &domain.Translation{ID: uuid.New(), SenseID: senseID}, nil
}

func (m *mockTranslationRepo) CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error) {
	if m.CreateCustomFunc != nil {
		return m.CreateCustomFunc(ctx, senseID, text, sourceSlug)
	}
	return &domain.Translation{ID: uuid.New(), SenseID: senseID}, nil
}

type mockExampleRepo struct {
	GetBySenseIDsFunc func(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Example, error)
	CreateFromRefFunc func(ctx context.Context, senseID, refExampleID uuid.UUID, sourceSlug string) (*domain.Example, error)
	CreateCustomFunc  func(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
}

func (m *mockExampleRepo) GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Example, error) {
	if m.GetBySenseIDsFunc != nil {
		return m.GetBySenseIDsFunc(ctx, senseIDs)
	}
	return nil, nil
}

func (m *mockExampleRepo) CreateFromRef(ctx context.Context, senseID, refExampleID uuid.UUID, sourceSlug string) (*domain.Example, error) {
	if m.CreateFromRefFunc != nil {
		return m.CreateFromRefFunc(ctx, senseID, refExampleID, sourceSlug)
	}
	return &domain.Example{ID: uuid.New(), SenseID: senseID}, nil
}

func (m *mockExampleRepo) CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error) {
	if m.CreateCustomFunc != nil {
		return m.CreateCustomFunc(ctx, senseID, sentence, translation, sourceSlug)
	}
	return &domain.Example{ID: uuid.New(), SenseID: senseID}, nil
}

type mockPronunciationRepo struct {
	LinkFunc func(ctx context.Context, entryID, refPronunciationID uuid.UUID) error
}

func (m *mockPronunciationRepo) Link(ctx context.Context, entryID, refPronunciationID uuid.UUID) error {
	if m.LinkFunc != nil {
		return m.LinkFunc(ctx, entryID, refPronunciationID)
	}
	return nil
}

type mockImageRepo struct {
	LinkCatalogFunc func(ctx context.Context, entryID, refImageID uuid.UUID) error
}

func (m *mockImageRepo) LinkCatalog(ctx context.Context, entryID, refImageID uuid.UUID) error {
	if m.LinkCatalogFunc != nil {
		return m.LinkCatalogFunc(ctx, entryID, refImageID)
	}
	return nil
}

type mockCardRepo struct {
	GetByEntryIDsFunc func(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Card, error)
	CreateFunc        func(ctx context.Context, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) (*domain.Card, error)
}

func (m *mockCardRepo) GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Card, error) {
	if m.GetByEntryIDsFunc != nil {
		return m.GetByEntryIDsFunc(ctx, entryIDs)
	}
	return nil, nil
}

func (m *mockCardRepo) Create(ctx context.Context, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) (*domain.Card, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, userID, entryID, status, easeFactor)
	}
	return &domain.Card{ID: uuid.New(), UserID: userID, EntryID: entryID, Status: status, EaseFactor: easeFactor}, nil
}

type mockAuditRepo struct {
	CreateFunc func(ctx context.Context, record domain.AuditRecord) error
}

func (m *mockAuditRepo) Create(ctx context.Context, record domain.AuditRecord) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, record)
	}
	return nil
}

type mockTxManager struct {
	RunInTxFunc func(ctx context.Context, fn func(context.Context) error) error
}

func (m *mockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.RunInTxFunc != nil {
		return m.RunInTxFunc(ctx, fn)
	}
	return fn(ctx)
}

type mockRefCatalogService struct {
	GetOrFetchEntryFunc func(ctx context.Context, text string) (*domain.RefEntry, error)
	GetRefEntryFunc     func(ctx context.Context, refEntryID uuid.UUID) (*domain.RefEntry, error)
	SearchFunc          func(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
}

func (m *mockRefCatalogService) GetOrFetchEntry(ctx context.Context, text string) (*domain.RefEntry, error) {
	if m.GetOrFetchEntryFunc != nil {
		return m.GetOrFetchEntryFunc(ctx, text)
	}
	return nil, domain.ErrNotFound
}

func (m *mockRefCatalogService) GetRefEntry(ctx context.Context, refEntryID uuid.UUID) (*domain.RefEntry, error) {
	if m.GetRefEntryFunc != nil {
		return m.GetRefEntryFunc(ctx, refEntryID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockRefCatalogService) Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(ctx, query, limit)
	}
	return nil, nil
}

// ===========================================================================
// Helpers
// ===========================================================================

func defaultCfg() config.DictionaryConfig {
	return config.DictionaryConfig{
		MaxEntriesPerUser:       10000,
		DefaultEaseFactor:       2.5,
		ImportChunkSize:         50,
		ExportMaxEntries:        10000,
		HardDeleteRetentionDays: 30,
	}
}

type testDeps struct {
	entries        *mockEntryRepo
	senses         *mockSenseRepo
	translations   *mockTranslationRepo
	examples       *mockExampleRepo
	pronunciations *mockPronunciationRepo
	images         *mockImageRepo
	cards          *mockCardRepo
	audit          *mockAuditRepo
	tx             *mockTxManager
	refCatalog     *mockRefCatalogService
}

func newTestService(cfg config.DictionaryConfig) (*Service, *testDeps) {
	deps := &testDeps{
		entries:        &mockEntryRepo{},
		senses:         &mockSenseRepo{},
		translations:   &mockTranslationRepo{},
		examples:       &mockExampleRepo{},
		pronunciations: &mockPronunciationRepo{},
		images:         &mockImageRepo{},
		cards:          &mockCardRepo{},
		audit:          &mockAuditRepo{},
		tx:             &mockTxManager{},
		refCatalog:     &mockRefCatalogService{},
	}
	svc := NewService(
		slog.Default(),
		deps.entries,
		deps.senses,
		deps.translations,
		deps.examples,
		deps.pronunciations,
		deps.images,
		deps.cards,
		deps.audit,
		deps.tx,
		deps.refCatalog,
		cfg,
	)
	return svc, deps
}

func authCtx() (context.Context, uuid.UUID) {
	userID := uuid.New()
	return ctxutil.WithUserID(context.Background(), userID), userID
}

func ptrString(s string) *string    { return &s }
func ptrBool(b bool) *bool          { return &b }

func makeRefEntry(text string, senses ...domain.RefSense) *domain.RefEntry {
	return &domain.RefEntry{
		ID:             uuid.New(),
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		Senses:         senses,
	}
}

func makeRefSense(def string) domain.RefSense {
	return domain.RefSense{
		ID:         uuid.New(),
		Definition: def,
		SourceSlug: "freedict",
		Translations: []domain.RefTranslation{
			{ID: uuid.New(), Text: "перевод", SourceSlug: "translate"},
		},
		Examples: []domain.RefExample{
			{ID: uuid.New(), Sentence: "example sentence", SourceSlug: "freedict"},
		},
	}
}

// ===========================================================================
// 1. SearchCatalog Tests
// ===========================================================================

func TestService_SearchCatalog_EmptyQuery(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	results, err := svc.SearchCatalog(ctx, "", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestService_SearchCatalog_NormalQuery(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	expected := []domain.RefEntry{{ID: uuid.New(), Text: "hello"}}
	deps.refCatalog.SearchFunc = func(_ context.Context, q string, l int) ([]domain.RefEntry, error) {
		assert.Equal(t, "hel", q)
		assert.Equal(t, 10, l)
		return expected, nil
	}

	results, err := svc.SearchCatalog(ctx, "hel", 10)
	require.NoError(t, err)
	assert.Equal(t, expected, results)
}

func TestService_SearchCatalog_LimitClamp(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	var capturedLimit int
	deps.refCatalog.SearchFunc = func(_ context.Context, _ string, l int) ([]domain.RefEntry, error) {
		capturedLimit = l
		return nil, nil
	}

	_, _ = svc.SearchCatalog(ctx, "test", 999)
	assert.Equal(t, 50, capturedLimit)
}

func TestService_SearchCatalog_NoAuth(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())

	_, err := svc.SearchCatalog(context.Background(), "test", 10)
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// ===========================================================================
// 2. PreviewRefEntry Tests
// ===========================================================================

func TestService_PreviewRefEntry_Success(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	expected := makeRefEntry("hello")
	deps.refCatalog.GetOrFetchEntryFunc = func(_ context.Context, text string) (*domain.RefEntry, error) {
		assert.Equal(t, "hello", text)
		return expected, nil
	}

	result, err := svc.PreviewRefEntry(ctx, "hello")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_PreviewRefEntry_FetchOK(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	fetched := makeRefEntry("world")
	deps.refCatalog.GetOrFetchEntryFunc = func(_ context.Context, _ string) (*domain.RefEntry, error) {
		return fetched, nil
	}

	result, err := svc.PreviewRefEntry(ctx, "World")
	require.NoError(t, err)
	assert.Equal(t, fetched.ID, result.ID)
}

func TestService_PreviewRefEntry_APIError(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	apiErr := errors.New("API timeout")
	deps.refCatalog.GetOrFetchEntryFunc = func(_ context.Context, _ string) (*domain.RefEntry, error) {
		return nil, apiErr
	}

	_, err := svc.PreviewRefEntry(ctx, "hello")
	require.ErrorIs(t, err, apiErr)
}

// ===========================================================================
// 3. CreateEntryFromCatalog Tests
// ===========================================================================

func TestService_CreateFromCatalog_AllSenses(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, userID := authCtx()

	sense1 := makeRefSense("definition 1")
	sense2 := makeRefSense("definition 2")
	refEntry := makeRefEntry("hello", sense1, sense2)
	refEntry.Pronunciations = []domain.RefPronunciation{{ID: uuid.New()}}
	refEntry.Images = []domain.RefImage{{ID: uuid.New()}}

	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, id uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	senseCount := 0
	deps.senses.CreateFromRefFunc = func(_ context.Context, entryID, refSenseID uuid.UUID, slug string) (*domain.Sense, error) {
		senseCount++
		return &domain.Sense{ID: uuid.New(), EntryID: entryID}, nil
	}

	pronLinked := false
	deps.pronunciations.LinkFunc = func(_ context.Context, _, _ uuid.UUID) error {
		pronLinked = true
		return nil
	}

	imgLinked := false
	deps.images.LinkCatalogFunc = func(_ context.Context, _, _ uuid.UUID) error {
		imgLinked = true
		return nil
	}

	auditCreated := false
	deps.audit.CreateFunc = func(_ context.Context, rec domain.AuditRecord) error {
		assert.Equal(t, userID, rec.UserID)
		assert.Equal(t, domain.AuditActionCreate, rec.Action)
		assert.Equal(t, domain.EntityTypeEntry, rec.EntityType)
		auditCreated = true
		return nil
	}

	result, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 2, senseCount)
	assert.True(t, pronLinked)
	assert.True(t, imgLinked)
	assert.True(t, auditCreated)
}

func TestService_CreateFromCatalog_SelectedSenses(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	sense1 := makeRefSense("def1")
	sense2 := makeRefSense("def2")
	refEntry := makeRefEntry("hello", sense1, sense2)

	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	var capturedRefSenseIDs []uuid.UUID
	deps.senses.CreateFromRefFunc = func(_ context.Context, entryID, refSenseID uuid.UUID, _ string) (*domain.Sense, error) {
		capturedRefSenseIDs = append(capturedRefSenseIDs, refSenseID)
		return &domain.Sense{ID: uuid.New(), EntryID: entryID}, nil
	}

	result, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
		SenseIDs:   []uuid.UUID{sense2.ID},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, capturedRefSenseIDs, 1)
	assert.Equal(t, sense2.ID, capturedRefSenseIDs[0])
}

func TestService_CreateFromCatalog_WithCard(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, userID := authCtx()

	refEntry := makeRefEntry("hello")
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	cardCreated := false
	deps.cards.CreateFunc = func(_ context.Context, uid, eid uuid.UUID, status domain.LearningStatus, ef float64) (*domain.Card, error) {
		assert.Equal(t, userID, uid)
		assert.Equal(t, domain.LearningStatusNew, status)
		assert.Equal(t, 2.5, ef)
		cardCreated = true
		return &domain.Card{ID: uuid.New()}, nil
	}

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
		CreateCard: true,
	})

	require.NoError(t, err)
	assert.True(t, cardCreated)
}

func TestService_CreateFromCatalog_NoCard(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello")
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	cardCreated := false
	deps.cards.CreateFunc = func(_ context.Context, _, _ uuid.UUID, _ domain.LearningStatus, _ float64) (*domain.Card, error) {
		cardCreated = true
		return &domain.Card{ID: uuid.New()}, nil
	}

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
		CreateCard: false,
	})

	require.NoError(t, err)
	assert.False(t, cardCreated)
}

func TestService_CreateFromCatalog_WithNotes(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello")
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	var capturedNotes *string
	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		capturedNotes = entry.Notes
		entry.ID = uuid.New()
		return entry, nil
	}

	notes := "my notes"
	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
		Notes:      &notes,
	})

	require.NoError(t, err)
	require.NotNil(t, capturedNotes)
	assert.Equal(t, "my notes", *capturedNotes)
}

func TestService_CreateFromCatalog_Duplicate(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello")
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}
	deps.entries.GetByTextFunc = func(_ context.Context, _ uuid.UUID, _ string) (*domain.Entry, error) {
		return &domain.Entry{ID: uuid.New()}, nil // already exists
	}

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestService_CreateFromCatalog_DuplicateUniqueConstraint(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello")
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}
	deps.entries.CreateFunc = func(_ context.Context, _ *domain.Entry) (*domain.Entry, error) {
		return nil, domain.ErrAlreadyExists
	}

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestService_CreateFromCatalog_LimitReached(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello")
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}
	deps.entries.CountByUserFunc = func(_ context.Context, _ uuid.UUID) (int, error) {
		return 10000, nil
	}

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "entries", ve.Errors[0].Field)
}

func TestService_CreateFromCatalog_RefNotFound(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return nil, domain.ErrNotFound
	}

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: uuid.New(),
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "ref_entry_id", ve.Errors[0].Field)
}

func TestService_CreateFromCatalog_InvalidSenseID(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello", makeRefSense("def1"))
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	badSenseID := uuid.New()
	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
		SenseIDs:   []uuid.UUID{badSenseID},
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "sense_ids", ve.Errors[0].Field)
}

func TestService_CreateFromCatalog_InvalidInput(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	_, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: uuid.Nil, // required
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "ref_entry_id", ve.Errors[0].Field)
}

func TestService_CreateFromCatalog_NoAuth(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())

	_, err := svc.CreateEntryFromCatalog(context.Background(), CreateFromCatalogInput{
		RefEntryID: uuid.New(),
	})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestService_CreateFromCatalog_RefWithoutSenses(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	refEntry := makeRefEntry("hello") // no senses
	deps.refCatalog.GetRefEntryFunc = func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
		return refEntry, nil
	}

	senseCount := 0
	deps.senses.CreateFromRefFunc = func(_ context.Context, _, _ uuid.UUID, _ string) (*domain.Sense, error) {
		senseCount++
		return &domain.Sense{ID: uuid.New()}, nil
	}

	result, err := svc.CreateEntryFromCatalog(ctx, CreateFromCatalogInput{
		RefEntryID: refEntry.ID,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, senseCount, "no senses should be created for a ref without senses")
}

// ===========================================================================
// 4. CreateEntryCustom Tests
// ===========================================================================

func TestService_CreateCustom_HappyPath(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	pos := domain.PartOfSpeechNoun
	var capturedSlug string
	deps.senses.CreateCustomFunc = func(_ context.Context, _ uuid.UUID, _ *string, _ *domain.PartOfSpeech, _ *string, slug string) (*domain.Sense, error) {
		capturedSlug = slug
		return &domain.Sense{ID: uuid.New()}, nil
	}

	result, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "hello",
		Senses: []SenseInput{
			{
				Definition:   ptrString("a greeting"),
				PartOfSpeech: &pos,
				Translations: []string{"привет"},
				Examples:     []ExampleInput{{Sentence: "Hello, World!"}},
			},
		},
		CreateCard: true,
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "user", capturedSlug)
}

func TestService_CreateCustom_EmptySenses(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	result, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "hello",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestService_CreateCustom_NoDefinition(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	var capturedDef *string
	deps.senses.CreateCustomFunc = func(_ context.Context, _ uuid.UUID, def *string, _ *domain.PartOfSpeech, _ *string, _ string) (*domain.Sense, error) {
		capturedDef = def
		return &domain.Sense{ID: uuid.New()}, nil
	}

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "hello",
		Senses: []SenseInput{
			{Translations: []string{"привет"}},
		},
	})

	require.NoError(t, err)
	assert.Nil(t, capturedDef)
}

func TestService_CreateCustom_WithCard(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	cardCreated := false
	deps.cards.CreateFunc = func(_ context.Context, _, _ uuid.UUID, status domain.LearningStatus, ef float64) (*domain.Card, error) {
		assert.Equal(t, domain.LearningStatusNew, status)
		assert.Equal(t, 2.5, ef)
		cardCreated = true
		return &domain.Card{ID: uuid.New()}, nil
	}

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text:       "hello",
		CreateCard: true,
	})

	require.NoError(t, err)
	assert.True(t, cardCreated)
}

func TestService_CreateCustom_NormalizesText(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	var capturedNormalized string
	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		capturedNormalized = entry.TextNormalized
		entry.ID = uuid.New()
		return entry, nil
	}

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "  Hello  World  ",
	})

	require.NoError(t, err)
	assert.Equal(t, "hello world", capturedNormalized)
}

func TestService_CreateCustom_Duplicate(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.GetByTextFunc = func(_ context.Context, _ uuid.UUID, _ string) (*domain.Entry, error) {
		return &domain.Entry{ID: uuid.New()}, nil
	}

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "hello",
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
}

func TestService_CreateCustom_InvalidText(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "",
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "text", ve.Errors[0].Field)
}

func TestService_CreateCustom_TooLongText(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	longText := make([]byte, 501)
	for i := range longText {
		longText[i] = 'a'
	}

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: string(longText),
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "text", ve.Errors[0].Field)
}

func TestService_CreateCustom_SourceSlugUser(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	var senseSlug, trSlug, exSlug string
	deps.senses.CreateCustomFunc = func(_ context.Context, _ uuid.UUID, _ *string, _ *domain.PartOfSpeech, _ *string, slug string) (*domain.Sense, error) {
		senseSlug = slug
		return &domain.Sense{ID: uuid.New()}, nil
	}
	deps.translations.CreateCustomFunc = func(_ context.Context, _ uuid.UUID, _ string, slug string) (*domain.Translation, error) {
		trSlug = slug
		return &domain.Translation{ID: uuid.New()}, nil
	}
	deps.examples.CreateCustomFunc = func(_ context.Context, _ uuid.UUID, _ string, _ *string, slug string) (*domain.Example, error) {
		exSlug = slug
		return &domain.Example{ID: uuid.New()}, nil
	}

	_, err := svc.CreateEntryCustom(ctx, CreateCustomInput{
		Text: "hello",
		Senses: []SenseInput{
			{
				Translations: []string{"привет"},
				Examples:     []ExampleInput{{Sentence: "Hello!"}},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "user", senseSlug)
	assert.Equal(t, "user", trSlug)
	assert.Equal(t, "user", exSlug)
}

// ===========================================================================
// 5. FindEntries Tests
// ===========================================================================

func TestService_FindEntries_NoFiltersOffset(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	entries := []domain.Entry{{ID: uuid.New(), Text: "hello"}, {ID: uuid.New(), Text: "world"}}
	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		assert.Equal(t, "created_at", f.SortBy)
		assert.Equal(t, "DESC", f.SortOrder)
		return entries, 2, nil
	}

	result, err := svc.FindEntries(ctx, FindInput{Limit: 20})
	require.NoError(t, err)
	assert.Len(t, result.Entries, 2)
	assert.Equal(t, 2, result.TotalCount)
	assert.False(t, result.HasNextPage)
}

func TestService_FindEntries_WithSearch(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		require.NotNil(t, f.Search)
		assert.Equal(t, "hello", *f.Search)
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{Search: ptrString("Hello"), Limit: 20})
	require.NoError(t, err)
}

func TestService_FindEntries_SearchSpaces(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		// Spaces-only input normalizes to "", so search should be nil.
		assert.Nil(t, f.Search)
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{Search: ptrString("   "), Limit: 20})
	require.NoError(t, err)
}

func TestService_FindEntries_HasCard(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		require.NotNil(t, f.HasCard)
		assert.True(t, *f.HasCard)
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{HasCard: ptrBool(true), Limit: 20})
	require.NoError(t, err)
}

func TestService_FindEntries_TopicID(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	topicID := uuid.New()
	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		require.NotNil(t, f.TopicID)
		assert.Equal(t, topicID, *f.TopicID)
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{TopicID: &topicID, Limit: 20})
	require.NoError(t, err)
}

func TestService_FindEntries_ComboFilters(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	pos := domain.PartOfSpeechNoun
	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		require.NotNil(t, f.Search)
		assert.Equal(t, "hello", *f.Search)
		require.NotNil(t, f.HasCard)
		assert.True(t, *f.HasCard)
		require.NotNil(t, f.PartOfSpeech)
		assert.Equal(t, domain.PartOfSpeechNoun, *f.PartOfSpeech)
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{
		Search:       ptrString("Hello"),
		HasCard:      ptrBool(true),
		PartOfSpeech: &pos,
		Limit:        20,
	})
	require.NoError(t, err)
}

func TestService_FindEntries_Cursor(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	entries := []domain.Entry{{ID: uuid.New()}}
	deps.entries.FindCursorFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, bool, error) {
		require.NotNil(t, f.Cursor)
		assert.Equal(t, "cursor123", *f.Cursor)
		return entries, true, nil
	}

	cursor := "cursor123"
	result, err := svc.FindEntries(ctx, FindInput{Cursor: &cursor, Limit: 20})
	require.NoError(t, err)
	assert.True(t, result.HasNextPage)
	require.NotNil(t, result.PageInfo)
}

func TestService_FindEntries_CursorWithOffsetUsesCursor(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	cursorCalled := false
	deps.entries.FindCursorFunc = func(_ context.Context, _ uuid.UUID, _ domain.EntryFilter) ([]domain.Entry, bool, error) {
		cursorCalled = true
		return nil, false, nil
	}

	cursor := "cursor123"
	offset := 10
	_, err := svc.FindEntries(ctx, FindInput{Cursor: &cursor, Offset: &offset, Limit: 20})
	require.NoError(t, err)
	assert.True(t, cursorCalled)
}

func TestService_FindEntries_LimitClamp(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	var capturedLimit int
	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		capturedLimit = f.Limit
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{Limit: 999})
	require.NoError(t, err)
	assert.Equal(t, 200, capturedLimit)
}

func TestService_FindEntries_DefaultSort(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		assert.Equal(t, "created_at", f.SortBy)
		assert.Equal(t, "DESC", f.SortOrder)
		return nil, 0, nil
	}

	_, err := svc.FindEntries(ctx, FindInput{Limit: 20})
	require.NoError(t, err)
}

func TestService_FindEntries_InvalidSortBy(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	_, err := svc.FindEntries(ctx, FindInput{SortBy: "invalid", Limit: 20})
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "sort_by", ve.Errors[0].Field)
}

// ===========================================================================
// 6. GetEntry Tests
// ===========================================================================

func TestService_GetEntry_Found(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, userID := authCtx()

	entryID := uuid.New()
	expected := &domain.Entry{ID: entryID, UserID: userID, Text: "hello"}
	deps.entries.GetByIDFunc = func(_ context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
		assert.Equal(t, userID, uid)
		assert.Equal(t, entryID, eid)
		return expected, nil
	}

	result, err := svc.GetEntry(ctx, entryID)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_GetEntry_NotFound(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.GetByIDFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return nil, domain.ErrNotFound
	}

	_, err := svc.GetEntry(ctx, uuid.New())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestService_GetEntry_NoAuth(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())

	_, err := svc.GetEntry(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// ===========================================================================
// 7. UpdateNotes Tests
// ===========================================================================

func TestService_UpdateNotes_Set(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, userID := authCtx()

	entryID := uuid.New()
	oldEntry := &domain.Entry{ID: entryID, UserID: userID, Notes: nil}
	deps.entries.GetByIDFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return oldEntry, nil
	}

	newNotes := "new notes"
	deps.entries.UpdateNotesFunc = func(_ context.Context, uid, eid uuid.UUID, notes *string) (*domain.Entry, error) {
		assert.Equal(t, userID, uid)
		assert.Equal(t, entryID, eid)
		return &domain.Entry{ID: entryID, Notes: notes}, nil
	}

	var auditChanges map[string]any
	deps.audit.CreateFunc = func(_ context.Context, rec domain.AuditRecord) error {
		assert.Equal(t, domain.AuditActionUpdate, rec.Action)
		auditChanges = rec.Changes
		return nil
	}

	result, err := svc.UpdateNotes(ctx, UpdateNotesInput{EntryID: entryID, Notes: &newNotes})
	require.NoError(t, err)
	require.NotNil(t, result.Notes)
	assert.Equal(t, "new notes", *result.Notes)

	// Check audit diff.
	assert.Nil(t, auditChanges["old_notes"])
	assert.Equal(t, &newNotes, auditChanges["new_notes"])
}

func TestService_UpdateNotes_Clear(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	entryID := uuid.New()
	oldNotes := "old"
	deps.entries.GetByIDFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return &domain.Entry{ID: entryID, Notes: &oldNotes}, nil
	}
	deps.entries.UpdateNotesFunc = func(_ context.Context, _, _ uuid.UUID, notes *string) (*domain.Entry, error) {
		return &domain.Entry{ID: entryID, Notes: notes}, nil
	}

	result, err := svc.UpdateNotes(ctx, UpdateNotesInput{EntryID: entryID, Notes: nil})
	require.NoError(t, err)
	assert.Nil(t, result.Notes)
}

func TestService_UpdateNotes_NotFound(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.GetByIDFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return nil, domain.ErrNotFound
	}

	_, err := svc.UpdateNotes(ctx, UpdateNotesInput{EntryID: uuid.New()})
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestService_UpdateNotes_Validation(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	_, err := svc.UpdateNotes(ctx, UpdateNotesInput{EntryID: uuid.Nil})
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "entry_id", ve.Errors[0].Field)
}

// ===========================================================================
// 8. DeleteEntry Tests
// ===========================================================================

func TestService_DeleteEntry_Happy(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, userID := authCtx()

	entryID := uuid.New()
	deps.entries.GetByIDFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return &domain.Entry{ID: entryID, UserID: userID, Text: "hello"}, nil
	}

	deleted := false
	deps.entries.SoftDeleteFunc = func(_ context.Context, _, _ uuid.UUID) error {
		deleted = true
		return nil
	}

	auditCreated := false
	deps.audit.CreateFunc = func(_ context.Context, rec domain.AuditRecord) error {
		assert.Equal(t, domain.AuditActionDelete, rec.Action)
		assert.Equal(t, "hello", rec.Changes["text"])
		auditCreated = true
		return nil
	}

	err := svc.DeleteEntry(ctx, entryID)
	require.NoError(t, err)
	assert.True(t, deleted)
	assert.True(t, auditCreated)
}

func TestService_DeleteEntry_NotFound(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.GetByIDFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return nil, domain.ErrNotFound
	}

	err := svc.DeleteEntry(ctx, uuid.New())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestService_DeleteEntry_NoAuth(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())

	err := svc.DeleteEntry(context.Background(), uuid.New())
	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// ===========================================================================
// 9. FindDeletedEntries Tests
// ===========================================================================

func TestService_FindDeletedEntries_WithEntries(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	entries := []domain.Entry{{ID: uuid.New()}, {ID: uuid.New()}}
	deps.entries.FindDeletedFunc = func(_ context.Context, _ uuid.UUID, limit, offset int) ([]domain.Entry, int, error) {
		assert.Equal(t, 20, limit)
		assert.Equal(t, 0, offset)
		return entries, 2, nil
	}

	result, total, err := svc.FindDeletedEntries(ctx, 20, 0)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
}

func TestService_FindDeletedEntries_Empty(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.FindDeletedFunc = func(_ context.Context, _ uuid.UUID, _, _ int) ([]domain.Entry, int, error) {
		return nil, 0, nil
	}

	result, total, err := svc.FindDeletedEntries(ctx, 20, 0)
	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
}

func TestService_FindDeletedEntries_LimitClamp(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	var capturedLimit int
	deps.entries.FindDeletedFunc = func(_ context.Context, _ uuid.UUID, limit, _ int) ([]domain.Entry, int, error) {
		capturedLimit = limit
		return nil, 0, nil
	}

	_, _, err := svc.FindDeletedEntries(ctx, 999, 0)
	require.NoError(t, err)
	assert.Equal(t, 200, capturedLimit)
}

// ===========================================================================
// 10. RestoreEntry Tests
// ===========================================================================

func TestService_RestoreEntry_Happy(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	entryID := uuid.New()
	restored := &domain.Entry{ID: entryID, Text: "hello"}
	deps.entries.RestoreFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return restored, nil
	}

	result, err := svc.RestoreEntry(ctx, entryID)
	require.NoError(t, err)
	assert.Equal(t, restored, result)
}

func TestService_RestoreEntry_NotFound(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.RestoreFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return nil, domain.ErrNotFound
	}

	_, err := svc.RestoreEntry(ctx, uuid.New())
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestService_RestoreEntry_TextConflict(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.RestoreFunc = func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
		return nil, domain.ErrAlreadyExists
	}

	_, err := svc.RestoreEntry(ctx, uuid.New())
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "text", ve.Errors[0].Field)
	assert.Contains(t, ve.Errors[0].Message, "active entry")
}

// ===========================================================================
// 11. BatchDeleteEntries Tests
// ===========================================================================

func TestService_BatchDelete_AllOK(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	deps.entries.SoftDeleteFunc = func(_ context.Context, _, _ uuid.UUID) error {
		return nil
	}

	auditCreated := false
	deps.audit.CreateFunc = func(_ context.Context, rec domain.AuditRecord) error {
		assert.Equal(t, domain.AuditActionDelete, rec.Action)
		auditCreated = true
		return nil
	}

	result, err := svc.BatchDeleteEntries(ctx, ids)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Deleted)
	assert.Empty(t, result.Errors)
	assert.True(t, auditCreated)
}

func TestService_BatchDelete_Partial(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	failID := uuid.New()
	ids := []uuid.UUID{uuid.New(), failID, uuid.New()}
	deps.entries.SoftDeleteFunc = func(_ context.Context, _, eid uuid.UUID) error {
		if eid == failID {
			return domain.ErrNotFound
		}
		return nil
	}

	result, err := svc.BatchDeleteEntries(ctx, ids)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Deleted)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, failID, result.Errors[0].EntryID)
}

func TestService_BatchDelete_Empty(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	_, err := svc.BatchDeleteEntries(ctx, []uuid.UUID{})
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "entry_ids", ve.Errors[0].Field)
}

func TestService_BatchDelete_TooMany(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	ids := make([]uuid.UUID, 201)
	for i := range ids {
		ids[i] = uuid.New()
	}

	_, err := svc.BatchDeleteEntries(ctx, ids)
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "entry_ids", ve.Errors[0].Field)
}

func TestService_BatchDelete_AuditOnlyOnSuccess(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	// All fail.
	deps.entries.SoftDeleteFunc = func(_ context.Context, _, _ uuid.UUID) error {
		return domain.ErrNotFound
	}

	auditCreated := false
	deps.audit.CreateFunc = func(_ context.Context, _ domain.AuditRecord) error {
		auditCreated = true
		return nil
	}

	result, err := svc.BatchDeleteEntries(ctx, []uuid.UUID{uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Deleted)
	assert.False(t, auditCreated, "audit should not be created when nothing was deleted")
}

// ===========================================================================
// 12. ImportEntries Tests
// ===========================================================================

func TestService_ImportEntries_Happy(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		entry.ID = uuid.New()
		return entry, nil
	}

	var trSlug string
	deps.translations.CreateCustomFunc = func(_ context.Context, _ uuid.UUID, _ string, slug string) (*domain.Translation, error) {
		trSlug = slug
		return &domain.Translation{ID: uuid.New()}, nil
	}

	result, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{
			{Text: "hello", Translations: []string{"привет"}},
			{Text: "world"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Imported)
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.Errors)
	assert.Equal(t, "import", trSlug)
}

func TestService_ImportEntries_DuplicateInFile(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		entry.ID = uuid.New()
		return entry, nil
	}

	result, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{
			{Text: "hello"},
			{Text: "Hello"}, // duplicate after normalization
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Skipped)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "duplicate within import", result.Errors[0].Reason)
}

func TestService_ImportEntries_ExistingEntry(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.GetByTextFunc = func(_ context.Context, _ uuid.UUID, text string) (*domain.Entry, error) {
		if text == "hello" {
			return &domain.Entry{ID: uuid.New()}, nil
		}
		return nil, domain.ErrNotFound
	}
	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		entry.ID = uuid.New()
		return entry, nil
	}

	result, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{
			{Text: "hello"}, // already exists
			{Text: "world"}, // new
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Skipped)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "entry already exists", result.Errors[0].Reason)
}

func TestService_ImportEntries_EmptyText(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		entry.ID = uuid.New()
		return entry, nil
	}

	result, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{
			{Text: "   "}, // normalizes to empty
			{Text: "hello"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, result.Imported)
	assert.Equal(t, 1, result.Skipped)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "empty text after normalization", result.Errors[0].Reason)
}

func TestService_ImportEntries_ChunkFail(t *testing.T) {
	t.Parallel()
	cfg := defaultCfg()
	cfg.ImportChunkSize = 2

	svc, deps := newTestService(cfg)
	ctx, _ := authCtx()

	callCount := 0
	deps.tx.RunInTxFunc = func(ctx context.Context, fn func(context.Context) error) error {
		callCount++
		if callCount == 2 {
			return errors.New("db connection lost")
		}
		return fn(ctx)
	}

	deps.entries.CreateFunc = func(_ context.Context, entry *domain.Entry) (*domain.Entry, error) {
		entry.ID = uuid.New()
		return entry, nil
	}

	result, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{
			{Text: "a"},
			{Text: "b"},
			{Text: "c"}, // chunk 2 — will fail
			{Text: "d"}, // chunk 2 — will fail
		},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Imported, "only first chunk should succeed")
	assert.Equal(t, 2, result.Skipped, "second chunk items should be skipped")
}

func TestService_ImportEntries_LimitExceeded(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.CountByUserFunc = func(_ context.Context, _ uuid.UUID) (int, error) {
		return 9999, nil
	}

	_, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{
			{Text: "a"},
			{Text: "b"},
		},
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "items", ve.Errors[0].Field)
}

func TestService_ImportEntries_EmptyItems(t *testing.T) {
	t.Parallel()
	svc, _ := newTestService(defaultCfg())
	ctx, _ := authCtx()

	_, err := svc.ImportEntries(ctx, ImportInput{
		Items: []ImportItem{},
	})

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "items", ve.Errors[0].Field)
}

// ===========================================================================
// 13. ExportEntries Tests
// ===========================================================================

func TestService_ExportEntries_Happy(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	entryID := uuid.New()
	entries := []domain.Entry{{ID: entryID, Text: "hello"}}
	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, f domain.EntryFilter) ([]domain.Entry, int, error) {
		assert.Equal(t, "created_at", f.SortBy)
		assert.Equal(t, "ASC", f.SortOrder)
		return entries, 1, nil
	}

	senseID := uuid.New()
	deps.senses.GetByEntryIDsFunc = func(_ context.Context, ids []uuid.UUID) ([]domain.Sense, error) {
		return []domain.Sense{{ID: senseID, EntryID: entryID, Definition: ptrString("greeting")}}, nil
	}

	trText := "привет"
	deps.translations.GetBySenseIDsFunc = func(_ context.Context, ids []uuid.UUID) ([]domain.Translation, error) {
		return []domain.Translation{{ID: uuid.New(), SenseID: senseID, Text: &trText}}, nil
	}

	sentence := "Hello!"
	deps.examples.GetBySenseIDsFunc = func(_ context.Context, ids []uuid.UUID) ([]domain.Example, error) {
		return []domain.Example{{ID: uuid.New(), SenseID: senseID, Sentence: &sentence}}, nil
	}

	status := domain.LearningStatusNew
	deps.cards.GetByEntryIDsFunc = func(_ context.Context, ids []uuid.UUID) ([]domain.Card, error) {
		return []domain.Card{{EntryID: entryID, Status: status}}, nil
	}

	result, err := svc.ExportEntries(ctx)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "hello", result.Items[0].Text)
	require.NotNil(t, result.Items[0].CardStatus)
	assert.Equal(t, domain.LearningStatusNew, *result.Items[0].CardStatus)
	require.Len(t, result.Items[0].Senses, 1)
	assert.Equal(t, "greeting", *result.Items[0].Senses[0].Definition)
	require.Len(t, result.Items[0].Senses[0].Translations, 1)
	assert.Equal(t, "привет", result.Items[0].Senses[0].Translations[0])
	require.Len(t, result.Items[0].Senses[0].Examples, 1)
	assert.Equal(t, "Hello!", result.Items[0].Senses[0].Examples[0].Sentence)
}

func TestService_ExportEntries_Empty(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, _ domain.EntryFilter) ([]domain.Entry, int, error) {
		return nil, 0, nil
	}

	result, err := svc.ExportEntries(ctx)
	require.NoError(t, err)
	assert.Empty(t, result.Items)
	assert.False(t, result.ExportedAt.IsZero())
}

func TestService_ExportEntries_WithData(t *testing.T) {
	t.Parallel()
	svc, deps := newTestService(defaultCfg())
	ctx, _ := authCtx()

	id1, id2 := uuid.New(), uuid.New()
	entries := []domain.Entry{
		{ID: id1, Text: "hello", Notes: ptrString("note1")},
		{ID: id2, Text: "world"},
	}
	deps.entries.FindFunc = func(_ context.Context, _ uuid.UUID, _ domain.EntryFilter) ([]domain.Entry, int, error) {
		return entries, 2, nil
	}

	senseID1, senseID2 := uuid.New(), uuid.New()
	deps.senses.GetByEntryIDsFunc = func(_ context.Context, _ []uuid.UUID) ([]domain.Sense, error) {
		return []domain.Sense{
			{ID: senseID1, EntryID: id1},
			{ID: senseID2, EntryID: id2},
		}, nil
	}

	tr1 := "привет"
	deps.translations.GetBySenseIDsFunc = func(_ context.Context, _ []uuid.UUID) ([]domain.Translation, error) {
		return []domain.Translation{
			{SenseID: senseID1, Text: &tr1},
		}, nil
	}

	deps.examples.GetBySenseIDsFunc = func(_ context.Context, _ []uuid.UUID) ([]domain.Example, error) {
		return nil, nil
	}

	deps.cards.GetByEntryIDsFunc = func(_ context.Context, _ []uuid.UUID) ([]domain.Card, error) {
		return nil, nil
	}

	result, err := svc.ExportEntries(ctx)
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	assert.Equal(t, "hello", result.Items[0].Text)
	require.NotNil(t, result.Items[0].Notes)
	assert.Equal(t, "note1", *result.Items[0].Notes)
	assert.Equal(t, "world", result.Items[1].Text)
	assert.Nil(t, result.Items[1].Notes)
}

// ===========================================================================
// Input Validation Tests
// ===========================================================================

func TestCreateFromCatalogInput_Validate_CollectsAllErrors(t *testing.T) {
	t.Parallel()

	longNotes := make([]byte, 5001)
	for i := range longNotes {
		longNotes[i] = 'a'
	}
	notesStr := string(longNotes)

	ids := make([]uuid.UUID, 21)
	for i := range ids {
		ids[i] = uuid.New()
	}

	input := CreateFromCatalogInput{
		RefEntryID: uuid.Nil,
		SenseIDs:   ids,
		Notes:      &notesStr,
	}

	err := input.Validate()
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Len(t, ve.Errors, 3, "should collect all 3 errors")
}

func TestCreateCustomInput_Validate_CollectsAllErrors(t *testing.T) {
	t.Parallel()

	longNotes := make([]byte, 5001)
	for i := range longNotes {
		longNotes[i] = 'a'
	}
	notesStr := string(longNotes)

	input := CreateCustomInput{
		Text:  "", // required
		Notes: &notesStr,
	}

	err := input.Validate()
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Len(t, ve.Errors, 2, "should collect both errors")
}

func TestFindInput_Validate_CollectsAllErrors(t *testing.T) {
	t.Parallel()

	input := FindInput{
		SortBy:    "invalid",
		SortOrder: "invalid",
	}

	err := input.Validate()
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Len(t, ve.Errors, 2, "should collect both errors")
}

func TestImportInput_Validate_CollectsAllErrors(t *testing.T) {
	t.Parallel()

	input := ImportInput{
		Items: []ImportItem{
			{Text: ""}, // required
			{Text: "ok", Translations: make([]string, 21)}, // too many translations
		},
	}

	err := input.Validate()
	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Len(t, ve.Errors, 2)
}
