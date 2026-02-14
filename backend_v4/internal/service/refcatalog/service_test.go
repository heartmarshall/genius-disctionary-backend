package refcatalog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Manual mocks (moq-style with func fields)
// ---------------------------------------------------------------------------

type mockRefEntryRepo struct {
	SearchFunc            func(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
	GetFullTreeByIDFunc   func(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error)
	GetFullTreeByTextFunc func(ctx context.Context, textNormalized string) (*domain.RefEntry, error)
	CreateWithTreeFunc    func(ctx context.Context, entry *domain.RefEntry) (*domain.RefEntry, error)
}

func (m *mockRefEntryRepo) Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	return m.SearchFunc(ctx, query, limit)
}

func (m *mockRefEntryRepo) GetFullTreeByID(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error) {
	return m.GetFullTreeByIDFunc(ctx, id)
}

func (m *mockRefEntryRepo) GetFullTreeByText(ctx context.Context, textNormalized string) (*domain.RefEntry, error) {
	return m.GetFullTreeByTextFunc(ctx, textNormalized)
}

func (m *mockRefEntryRepo) CreateWithTree(ctx context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
	return m.CreateWithTreeFunc(ctx, entry)
}

type mockTxManager struct {
	RunInTxFunc func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.RunInTxFunc != nil {
		return m.RunInTxFunc(ctx, fn)
	}
	// Default: pass-through (no real transaction).
	return fn(ctx)
}

type mockDictionaryProvider struct {
	FetchEntryFunc func(ctx context.Context, word string) (*provider.DictionaryResult, error)
}

func (m *mockDictionaryProvider) FetchEntry(ctx context.Context, word string) (*provider.DictionaryResult, error) {
	return m.FetchEntryFunc(ctx, word)
}

type mockTranslationProvider struct {
	FetchTranslationsFunc func(ctx context.Context, word string) ([]string, error)
}

func (m *mockTranslationProvider) FetchTranslations(ctx context.Context, word string) ([]string, error) {
	return m.FetchTranslationsFunc(ctx, word)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestService(
	repo *mockRefEntryRepo,
	tx *mockTxManager,
	dict *mockDictionaryProvider,
	trans *mockTranslationProvider,
) *Service {
	logger := slog.Default()
	if tx == nil {
		tx = &mockTxManager{}
	}
	return NewService(logger, repo, tx, dict, trans)
}

func ptrString(s string) *string { return &s }

func makeDictResult(word string, senses []provider.SenseResult, prons []provider.PronunciationResult) *provider.DictionaryResult {
	return &provider.DictionaryResult{
		Word:           word,
		Senses:         senses,
		Pronunciations: prons,
	}
}

func makeRefEntry(text string) *domain.RefEntry {
	return &domain.RefEntry{
		ID:             uuid.New(),
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		Senses:         []domain.RefSense{},
		Pronunciations: []domain.RefPronunciation{},
	}
}

// ---------------------------------------------------------------------------
// Search tests
// ---------------------------------------------------------------------------

func TestService_Search_EmptyQuery(t *testing.T) {
	t.Parallel()

	searchCalled := false
	repo := &mockRefEntryRepo{
		SearchFunc: func(_ context.Context, _ string, _ int) ([]domain.RefEntry, error) {
			searchCalled = true
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, nil, nil)
	results, err := svc.Search(context.Background(), "", 10)

	require.NoError(t, err)
	assert.Empty(t, results)
	assert.False(t, searchCalled, "Search should NOT be called for empty query")
}

func TestService_Search_NormalQuery(t *testing.T) {
	t.Parallel()

	expected := []domain.RefEntry{
		{ID: uuid.New(), Text: "hello"},
		{ID: uuid.New(), Text: "help"},
	}
	repo := &mockRefEntryRepo{
		SearchFunc: func(_ context.Context, query string, limit int) ([]domain.RefEntry, error) {
			assert.Equal(t, "hel", query)
			assert.Equal(t, 10, limit)
			return expected, nil
		},
	}

	svc := newTestService(repo, nil, nil, nil)
	results, err := svc.Search(context.Background(), "hel", 10)

	require.NoError(t, err)
	assert.Equal(t, expected, results)
}

func TestService_Search_LimitClampedToMax(t *testing.T) {
	t.Parallel()

	var capturedLimit int
	repo := &mockRefEntryRepo{
		SearchFunc: func(_ context.Context, _ string, limit int) ([]domain.RefEntry, error) {
			capturedLimit = limit
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, nil, nil)
	_, err := svc.Search(context.Background(), "test", 999)

	require.NoError(t, err)
	assert.Equal(t, 50, capturedLimit)
}

func TestService_Search_LimitClampedToMin(t *testing.T) {
	t.Parallel()

	var capturedLimit int
	repo := &mockRefEntryRepo{
		SearchFunc: func(_ context.Context, _ string, limit int) ([]domain.RefEntry, error) {
			capturedLimit = limit
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, nil, nil)
	_, err := svc.Search(context.Background(), "test", 0)

	require.NoError(t, err)
	assert.Equal(t, 20, capturedLimit)
}

// ---------------------------------------------------------------------------
// GetOrFetchEntry tests
// ---------------------------------------------------------------------------

func TestService_GetOrFetchEntry_WordInCatalog(t *testing.T) {
	t.Parallel()

	existing := makeRefEntry("hello")
	dictCalled := false
	transCalled := false

	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, text string) (*domain.RefEntry, error) {
			assert.Equal(t, "hello", text)
			return existing, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			dictCalled = true
			return nil, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			transCalled = true
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	assert.Equal(t, existing, result)
	assert.False(t, dictCalled, "dictionary provider should NOT be called when entry exists")
	assert.False(t, transCalled, "translation provider should NOT be called when entry exists")
}

func TestService_GetOrFetchEntry_FetchSuccessNoTranslations(t *testing.T) {
	t.Parallel()

	dictResult := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting", PartOfSpeech: ptrString("noun")},
	}, nil)

	var createdEntry *domain.RefEntry
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
		CreateWithTreeFunc: func(_ context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
			createdEntry = entry
			return entry, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "hello", createdEntry.TextNormalized)
	assert.Len(t, createdEntry.Senses, 1)
	assert.Equal(t, "greeting", createdEntry.Senses[0].Definition)
	assert.Empty(t, createdEntry.Senses[0].Translations)
}

func TestService_GetOrFetchEntry_FetchSuccessWithTranslations(t *testing.T) {
	t.Parallel()

	dictResult := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting", PartOfSpeech: ptrString("noun")},
		{Definition: "an expression of greeting", PartOfSpeech: ptrString("interjection")},
	}, nil)

	var createdEntry *domain.RefEntry
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
		CreateWithTreeFunc: func(_ context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
			createdEntry = entry
			return entry, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"привет", "здравствуйте"}, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Translations attached to first sense only.
	assert.Len(t, createdEntry.Senses[0].Translations, 2)
	assert.Equal(t, "привет", createdEntry.Senses[0].Translations[0].Text)
	assert.Equal(t, "здравствуйте", createdEntry.Senses[0].Translations[1].Text)
	assert.Equal(t, "translate", createdEntry.Senses[0].Translations[0].SourceSlug)
	// Second sense has no translations.
	assert.Empty(t, createdEntry.Senses[1].Translations)
}

func TestService_GetOrFetchEntry_WordNotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return nil, nil // nil result = word not found
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	_, err := svc.GetOrFetchEntry(context.Background(), "xyznonexistent")

	require.ErrorIs(t, err, ErrWordNotFound)
}

func TestService_GetOrFetchEntry_DictionaryProviderError(t *testing.T) {
	t.Parallel()

	providerErr := errors.New("API timeout")
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return nil, providerErr
		},
	}

	svc := newTestService(repo, nil, dict, nil)
	_, err := svc.GetOrFetchEntry(context.Background(), "hello")

	require.Error(t, err)
	assert.ErrorIs(t, err, providerErr)
}

func TestService_GetOrFetchEntry_TranslationProviderError(t *testing.T) {
	t.Parallel()

	dictResult := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting", PartOfSpeech: ptrString("noun")},
	}, nil)

	var createdEntry *domain.RefEntry
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
		CreateWithTreeFunc: func(_ context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
			createdEntry = entry
			return entry, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return nil, fmt.Errorf("translation API unavailable")
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err, "translation error should not propagate")
	require.NotNil(t, result)
	// Entry saved without translations.
	assert.Empty(t, createdEntry.Senses[0].Translations)
}

func TestService_GetOrFetchEntry_ConcurrentCreate(t *testing.T) {
	t.Parallel()

	existing := makeRefEntry("hello")
	dictResult := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting"},
	}, nil)

	getByTextCallCount := 0
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			getByTextCallCount++
			if getByTextCallCount == 1 {
				return nil, domain.ErrNotFound // first call: not found
			}
			return existing, nil // second call (after conflict): found
		},
		CreateWithTreeFunc: func(_ context.Context, _ *domain.RefEntry) (*domain.RefEntry, error) {
			return nil, domain.ErrAlreadyExists // concurrent insert
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	assert.Equal(t, existing.ID, result.ID)
	assert.Equal(t, 2, getByTextCallCount, "GetFullTreeByText should be called twice")
}

func TestService_GetOrFetchEntry_EmptyText(t *testing.T) {
	t.Parallel()

	svc := newTestService(&mockRefEntryRepo{}, nil, nil, nil)
	_, err := svc.GetOrFetchEntry(context.Background(), "")

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "text", ve.Errors[0].Field)
	assert.Equal(t, "required", ve.Errors[0].Message)
}

func TestService_GetOrFetchEntry_TextOfSpaces(t *testing.T) {
	t.Parallel()

	svc := newTestService(&mockRefEntryRepo{}, nil, nil, nil)
	_, err := svc.GetOrFetchEntry(context.Background(), "   ")

	require.Error(t, err)
	var ve *domain.ValidationError
	require.ErrorAs(t, err, &ve)
	assert.Equal(t, "text", ve.Errors[0].Field)
}

func TestService_GetOrFetchEntry_ProviderReturnsNoSenses(t *testing.T) {
	t.Parallel()

	dictResult := makeDictResult("hello", nil, nil) // no senses

	var createdEntry *domain.RefEntry
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
		CreateWithTreeFunc: func(_ context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
			createdEntry = entry
			return entry, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, createdEntry.Senses)
}

func TestService_GetOrFetchEntry_SensesWithoutExamples(t *testing.T) {
	t.Parallel()

	dictResult := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting", PartOfSpeech: ptrString("noun"), Examples: nil},
	}, nil)

	var createdEntry *domain.RefEntry
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
		CreateWithTreeFunc: func(_ context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
			createdEntry = entry
			return entry, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return nil, nil
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, createdEntry.Senses, 1)
	assert.Empty(t, createdEntry.Senses[0].Examples)
}

func TestService_GetOrFetchEntry_TranslationsExistSensesEmpty(t *testing.T) {
	t.Parallel()

	dictResult := makeDictResult("hello", nil, nil) // no senses

	var createdEntry *domain.RefEntry
	repo := &mockRefEntryRepo{
		GetFullTreeByTextFunc: func(_ context.Context, _ string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
		CreateWithTreeFunc: func(_ context.Context, entry *domain.RefEntry) (*domain.RefEntry, error) {
			createdEntry = entry
			return entry, nil
		},
	}
	dict := &mockDictionaryProvider{
		FetchEntryFunc: func(_ context.Context, _ string) (*provider.DictionaryResult, error) {
			return dictResult, nil
		},
	}
	trans := &mockTranslationProvider{
		FetchTranslationsFunc: func(_ context.Context, _ string) ([]string, error) {
			return []string{"привет"}, nil // translations exist
		},
	}

	svc := newTestService(repo, nil, dict, trans)
	result, err := svc.GetOrFetchEntry(context.Background(), "Hello")

	require.NoError(t, err)
	require.NotNil(t, result)
	// Translations should be ignored because there are no senses.
	assert.Empty(t, createdEntry.Senses)
}

// ---------------------------------------------------------------------------
// GetRefEntry tests
// ---------------------------------------------------------------------------

func TestService_GetRefEntry_Found(t *testing.T) {
	t.Parallel()

	expected := makeRefEntry("hello")
	repo := &mockRefEntryRepo{
		GetFullTreeByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.RefEntry, error) {
			assert.Equal(t, expected.ID, id)
			return expected, nil
		},
	}

	svc := newTestService(repo, nil, nil, nil)
	result, err := svc.GetRefEntry(context.Background(), expected.ID)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestService_GetRefEntry_NotFound(t *testing.T) {
	t.Parallel()

	repo := &mockRefEntryRepo{
		GetFullTreeByIDFunc: func(_ context.Context, _ uuid.UUID) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(repo, nil, nil, nil)
	_, err := svc.GetRefEntry(context.Background(), uuid.New())

	require.ErrorIs(t, err, domain.ErrNotFound)
}

// ---------------------------------------------------------------------------
// mapToRefEntry tests (table-driven)
// ---------------------------------------------------------------------------

func TestMapToRefEntry_FullResult(t *testing.T) {
	t.Parallel()

	dict := &provider.DictionaryResult{
		Word: "Hello",
		Senses: []provider.SenseResult{
			{
				Definition:   "greeting",
				PartOfSpeech: ptrString("noun"),
				Examples: []provider.ExampleResult{
					{Sentence: "Hello, world!", Translation: ptrString("Привет, мир!")},
				},
			},
		},
		Pronunciations: []provider.PronunciationResult{
			{Transcription: ptrString("/hɛˈloʊ/"), AudioURL: ptrString("https://example.com/hello.mp3"), Region: ptrString("us")},
		},
	}
	translations := []string{"привет", "здравствуйте"}

	entry := mapToRefEntry("hello", dict, translations)

	assert.Equal(t, "Hello", entry.Text)
	assert.Equal(t, "hello", entry.TextNormalized)
	require.Len(t, entry.Senses, 1)
	assert.Equal(t, "greeting", entry.Senses[0].Definition)
	assert.Equal(t, domain.PartOfSpeechNoun, *entry.Senses[0].PartOfSpeech)
	assert.Equal(t, "freedict", entry.Senses[0].SourceSlug)
	assert.Equal(t, 0, entry.Senses[0].Position)

	// Examples
	require.Len(t, entry.Senses[0].Examples, 1)
	assert.Equal(t, "Hello, world!", entry.Senses[0].Examples[0].Sentence)
	assert.Equal(t, "Привет, мир!", *entry.Senses[0].Examples[0].Translation)
	assert.Equal(t, "freedict", entry.Senses[0].Examples[0].SourceSlug)
	assert.Equal(t, 0, entry.Senses[0].Examples[0].Position)

	// Translations
	require.Len(t, entry.Senses[0].Translations, 2)
	assert.Equal(t, "привет", entry.Senses[0].Translations[0].Text)
	assert.Equal(t, "translate", entry.Senses[0].Translations[0].SourceSlug)
	assert.Equal(t, 0, entry.Senses[0].Translations[0].Position)
	assert.Equal(t, "здравствуйте", entry.Senses[0].Translations[1].Text)
	assert.Equal(t, 1, entry.Senses[0].Translations[1].Position)

	// Pronunciations
	require.Len(t, entry.Pronunciations, 1)
	assert.Equal(t, "/hɛˈloʊ/", *entry.Pronunciations[0].Transcription)
	assert.Equal(t, "https://example.com/hello.mp3", *entry.Pronunciations[0].AudioURL)
	assert.Equal(t, "us", *entry.Pronunciations[0].Region)
	assert.Equal(t, "freedict", entry.Pronunciations[0].SourceSlug)

	// Parent references
	assert.Equal(t, entry.ID, entry.Senses[0].RefEntryID)
	assert.Equal(t, entry.Senses[0].ID, entry.Senses[0].Examples[0].RefSenseID)
	assert.Equal(t, entry.Senses[0].ID, entry.Senses[0].Translations[0].RefSenseID)
	assert.Equal(t, entry.ID, entry.Pronunciations[0].RefEntryID)
}

func TestMapToRefEntry_WithoutTranslations(t *testing.T) {
	t.Parallel()

	dict := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting"},
	}, nil)

	entry := mapToRefEntry("hello", dict, nil)

	require.Len(t, entry.Senses, 1)
	assert.Empty(t, entry.Senses[0].Translations)
}

func TestMapToRefEntry_WithTranslations(t *testing.T) {
	t.Parallel()

	dict := makeDictResult("hello", []provider.SenseResult{
		{Definition: "first sense"},
		{Definition: "second sense"},
	}, nil)

	entry := mapToRefEntry("hello", dict, []string{"trans1", "trans2"})

	// Translations only on first sense.
	require.Len(t, entry.Senses[0].Translations, 2)
	assert.Empty(t, entry.Senses[1].Translations)
}

func TestMapToRefEntry_WithoutPronunciations(t *testing.T) {
	t.Parallel()

	dict := makeDictResult("hello", []provider.SenseResult{
		{Definition: "greeting"},
	}, nil)

	entry := mapToRefEntry("hello", dict, nil)

	assert.Empty(t, entry.Pronunciations)
}

func TestMapToRefEntry_MultipleSensesPositions(t *testing.T) {
	t.Parallel()

	dict := makeDictResult("run", []provider.SenseResult{
		{Definition: "to move fast"},
		{Definition: "to operate"},
		{Definition: "a period of running"},
	}, nil)

	entry := mapToRefEntry("run", dict, nil)

	require.Len(t, entry.Senses, 3)
	for i, s := range entry.Senses {
		assert.Equal(t, i, s.Position, "sense position should be sequential")
	}
}

func TestMapToRefEntry_UUIDUniqueness(t *testing.T) {
	t.Parallel()

	dict := &provider.DictionaryResult{
		Word: "test",
		Senses: []provider.SenseResult{
			{
				Definition:   "first",
				PartOfSpeech: ptrString("noun"),
				Examples:     []provider.ExampleResult{{Sentence: "ex1"}, {Sentence: "ex2"}},
			},
			{
				Definition: "second",
				Examples:   []provider.ExampleResult{{Sentence: "ex3"}},
			},
		},
		Pronunciations: []provider.PronunciationResult{
			{Transcription: ptrString("/tɛst/")},
		},
	}

	entry := mapToRefEntry("test", dict, []string{"тест"})

	ids := make(map[uuid.UUID]struct{})
	ids[entry.ID] = struct{}{}
	for _, s := range entry.Senses {
		_, dup := ids[s.ID]
		assert.False(t, dup, "duplicate UUID found: %s", s.ID)
		ids[s.ID] = struct{}{}
		for _, ex := range s.Examples {
			_, dup := ids[ex.ID]
			assert.False(t, dup, "duplicate UUID found: %s", ex.ID)
			ids[ex.ID] = struct{}{}
		}
		for _, tr := range s.Translations {
			_, dup := ids[tr.ID]
			assert.False(t, dup, "duplicate UUID found: %s", tr.ID)
			ids[tr.ID] = struct{}{}
		}
	}
	for _, p := range entry.Pronunciations {
		_, dup := ids[p.ID]
		assert.False(t, dup, "duplicate UUID found: %s", p.ID)
		ids[p.ID] = struct{}{}
	}

	// entry(1) + senses(2) + examples(3) + translations(1) + pronunciations(1) = 8
	assert.Len(t, ids, 8, "expected 8 unique UUIDs")
}

// ---------------------------------------------------------------------------
// mapPartOfSpeech tests (table-driven)
// ---------------------------------------------------------------------------

func TestMapPartOfSpeech(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *string
		expected *domain.PartOfSpeech
	}{
		{
			name:     "noun lowercase",
			input:    ptrString("noun"),
			expected: ptrPOS(domain.PartOfSpeechNoun),
		},
		{
			name:     "verb lowercase",
			input:    ptrString("verb"),
			expected: ptrPOS(domain.PartOfSpeechVerb),
		},
		{
			name:     "unknown maps to OTHER",
			input:    ptrString("unknown"),
			expected: ptrPOS(domain.PartOfSpeechOther),
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := mapPartOfSpeech(tc.input)
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tc.expected, *result)
			}
		})
	}
}

func ptrPOS(p domain.PartOfSpeech) *domain.PartOfSpeech { return &p }
