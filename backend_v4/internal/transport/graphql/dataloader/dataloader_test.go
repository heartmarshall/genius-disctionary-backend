package dataloader_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/card"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/example"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/image"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/pronunciation"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/reviewlog"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/topic"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/translation"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	dl "github.com/heartmarshall/myenglish-backend/internal/transport/graphql/dataloader"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// Mock repos
// ---------------------------------------------------------------------------

type mockSenseRepo struct {
	result []domain.Sense
	err    error
}

func (m *mockSenseRepo) GetByEntryIDs(_ context.Context, _ []uuid.UUID) ([]domain.Sense, error) {
	return m.result, m.err
}

type mockTranslationRepo struct {
	result []translation.TranslationWithSenseID
	err    error
}

func (m *mockTranslationRepo) GetBySenseIDs(_ context.Context, _ []uuid.UUID) ([]translation.TranslationWithSenseID, error) {
	return m.result, m.err
}

type mockExampleRepo struct {
	result []example.ExampleWithSenseID
	err    error
}

func (m *mockExampleRepo) GetBySenseIDs(_ context.Context, _ []uuid.UUID) ([]example.ExampleWithSenseID, error) {
	return m.result, m.err
}

type mockPronunciationRepo struct {
	result []pronunciation.PronunciationWithEntryID
	err    error
}

func (m *mockPronunciationRepo) GetByEntryIDs(_ context.Context, _ []uuid.UUID) ([]pronunciation.PronunciationWithEntryID, error) {
	return m.result, m.err
}

type mockImageRepo struct {
	catalogResult []image.CatalogImageWithEntryID
	userResult    []image.UserImageWithEntryID
	err           error
}

func (m *mockImageRepo) GetCatalogByEntryIDs(_ context.Context, _ []uuid.UUID) ([]image.CatalogImageWithEntryID, error) {
	return m.catalogResult, m.err
}

func (m *mockImageRepo) GetUserByEntryIDs(_ context.Context, _ []uuid.UUID) ([]image.UserImageWithEntryID, error) {
	return m.userResult, m.err
}

type mockCardRepo struct {
	result []card.CardWithEntryID
	err    error
}

func (m *mockCardRepo) GetByEntryIDs(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]card.CardWithEntryID, error) {
	return m.result, m.err
}

type mockTopicRepo struct {
	result []topic.TopicWithEntryID
	err    error
}

func (m *mockTopicRepo) GetByEntryIDs(_ context.Context, _ []uuid.UUID) ([]topic.TopicWithEntryID, error) {
	return m.result, m.err
}

type mockReviewLogRepo struct {
	result []reviewlog.ReviewLogWithCardID
	err    error
}

func (m *mockReviewLogRepo) GetByCardIDs(_ context.Context, _ []uuid.UUID) ([]reviewlog.ReviewLogWithCardID, error) {
	return m.result, m.err
}

func emptyRepos() *dl.Repos {
	return &dl.Repos{
		Sense:         &mockSenseRepo{},
		Translation:   &mockTranslationRepo{},
		Example:       &mockExampleRepo{},
		Pronunciation: &mockPronunciationRepo{},
		Image:         &mockImageRepo{},
		Card:          &mockCardRepo{},
		Topic:         &mockTopicRepo{},
		ReviewLog:     &mockReviewLogRepo{},
	}
}

// ---------------------------------------------------------------------------
// Context / Middleware tests
// ---------------------------------------------------------------------------

func TestFromContext_ReturnsLoaders(t *testing.T) {
	loaders := dl.NewLoaders(emptyRepos())
	ctx := dl.WithLoaders(context.Background(), loaders)

	got := dl.FromContext(ctx)
	assert.NotNil(t, got)
	assert.Equal(t, loaders, got)
}

func TestFromContext_PanicsWhenMissing(t *testing.T) {
	assert.Panics(t, func() {
		dl.FromContext(context.Background())
	})
}

func TestMiddleware_InjectsLoaders(t *testing.T) {
	repos := emptyRepos()
	mw := dl.Middleware(repos)

	var gotLoaders *dl.Loaders
	handler := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotLoaders = dl.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/graphql", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, gotLoaders)
	assert.NotNil(t, gotLoaders.SensesByEntryID)
	assert.NotNil(t, gotLoaders.TranslationsBySenseID)
	assert.NotNil(t, gotLoaders.ExamplesBySenseID)
	assert.NotNil(t, gotLoaders.PronunciationsByEntryID)
	assert.NotNil(t, gotLoaders.CatalogImagesByEntryID)
	assert.NotNil(t, gotLoaders.UserImagesByEntryID)
	assert.NotNil(t, gotLoaders.CardByEntryID)
	assert.NotNil(t, gotLoaders.TopicsByEntryID)
	assert.NotNil(t, gotLoaders.ReviewLogsByCardID)
}

// ---------------------------------------------------------------------------
// Batch function tests — verify grouping and empty results
// ---------------------------------------------------------------------------

func TestSensesLoader_GroupsByEntryID(t *testing.T) {
	entry1 := uuid.New()
	entry2 := uuid.New()

	repos := emptyRepos()
	repos.Sense = &mockSenseRepo{
		result: []domain.Sense{
			{ID: uuid.New(), EntryID: entry1},
			{ID: uuid.New(), EntryID: entry1},
			{ID: uuid.New(), EntryID: entry2},
		},
	}

	loaders := dl.NewLoaders(repos)
	ctx := context.Background()

	result1, err := loaders.SensesByEntryID.Load(ctx, entry1)()
	require.NoError(t, err)
	assert.Len(t, result1, 2)

	result2, err := loaders.SensesByEntryID.Load(ctx, entry2)()
	require.NoError(t, err)
	assert.Len(t, result2, 1)
}

func TestSensesLoader_EmptyResult(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	result, err := loaders.SensesByEntryID.Load(context.Background(), uuid.New())()
	require.NoError(t, err)
	assert.NotNil(t, result, "should return empty slice, not nil")
	assert.Empty(t, result)
}

func TestTranslationsLoader_GroupsBySenseID(t *testing.T) {
	sense1 := uuid.New()
	sense2 := uuid.New()

	repos := emptyRepos()
	repos.Translation = &mockTranslationRepo{
		result: []translation.TranslationWithSenseID{
			{SenseID: sense1, Translation: domain.Translation{ID: uuid.New()}},
			{SenseID: sense2, Translation: domain.Translation{ID: uuid.New()}},
			{SenseID: sense2, Translation: domain.Translation{ID: uuid.New()}},
		},
	}

	loaders := dl.NewLoaders(repos)
	ctx := context.Background()

	result1, err := loaders.TranslationsBySenseID.Load(ctx, sense1)()
	require.NoError(t, err)
	assert.Len(t, result1, 1)

	result2, err := loaders.TranslationsBySenseID.Load(ctx, sense2)()
	require.NoError(t, err)
	assert.Len(t, result2, 2)
}

func TestCardLoader_NullableResult(t *testing.T) {
	entry1 := uuid.New()
	entry2 := uuid.New() // no card for this entry
	userID := uuid.New()

	repos := emptyRepos()
	repos.Card = &mockCardRepo{
		result: []card.CardWithEntryID{
			{EntryID: entry1, Card: domain.Card{ID: uuid.New()}},
		},
	}

	loaders := dl.NewLoaders(repos)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result1, err := loaders.CardByEntryID.Load(ctx, entry1)()
	require.NoError(t, err)
	assert.NotNil(t, result1, "should return card for entry with card")

	result2, err := loaders.CardByEntryID.Load(ctx, entry2)()
	require.NoError(t, err)
	assert.Nil(t, result2, "should return nil for entry without card")
}

func TestCardLoader_ErrorOnMissingUserID(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	// No userID in context.
	_, err := loaders.CardByEntryID.Load(context.Background(), uuid.New())()
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestReviewLogsLoader_GroupsByCardID(t *testing.T) {
	card1 := uuid.New()
	card2 := uuid.New()

	repos := emptyRepos()
	repos.ReviewLog = &mockReviewLogRepo{
		result: []reviewlog.ReviewLogWithCardID{
			{CardID: card1, ReviewLog: domain.ReviewLog{ID: uuid.New()}},
			{CardID: card1, ReviewLog: domain.ReviewLog{ID: uuid.New()}},
			{CardID: card2, ReviewLog: domain.ReviewLog{ID: uuid.New()}},
		},
	}

	loaders := dl.NewLoaders(repos)
	ctx := context.Background()

	result1, err := loaders.ReviewLogsByCardID.Load(ctx, card1)()
	require.NoError(t, err)
	assert.Len(t, result1, 2)

	result2, err := loaders.ReviewLogsByCardID.Load(ctx, card2)()
	require.NoError(t, err)
	assert.Len(t, result2, 1)
}

func TestTopicsLoader_EmptyResult(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	result, err := loaders.TopicsByEntryID.Load(context.Background(), uuid.New())()
	require.NoError(t, err)
	assert.NotNil(t, result, "should return empty slice, not nil")
	assert.Empty(t, result)
}

func TestPronunciationsLoader_EmptyResult(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	result, err := loaders.PronunciationsByEntryID.Load(context.Background(), uuid.New())()
	require.NoError(t, err)
	assert.NotNil(t, result, "should return empty slice, not nil")
	assert.Empty(t, result)
}

func TestCatalogImagesLoader_EmptyResult(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	result, err := loaders.CatalogImagesByEntryID.Load(context.Background(), uuid.New())()
	require.NoError(t, err)
	assert.NotNil(t, result, "should return empty slice, not nil")
	assert.Empty(t, result)
}

func TestUserImagesLoader_EmptyResult(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	result, err := loaders.UserImagesByEntryID.Load(context.Background(), uuid.New())()
	require.NoError(t, err)
	assert.NotNil(t, result, "should return empty slice, not nil")
	assert.Empty(t, result)
}

func TestExamplesLoader_EmptyResult(t *testing.T) {
	repos := emptyRepos()
	loaders := dl.NewLoaders(repos)

	result, err := loaders.ExamplesBySenseID.Load(context.Background(), uuid.New())()
	require.NoError(t, err)
	assert.NotNil(t, result, "should return empty slice, not nil")
	assert.Empty(t, result)
}

func TestSensesLoader_PropagatesError(t *testing.T) {
	repos := emptyRepos()
	repos.Sense = &mockSenseRepo{err: domain.ErrNotFound}

	loaders := dl.NewLoaders(repos)

	_, err := loaders.SensesByEntryID.Load(context.Background(), uuid.New())()
	require.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}
