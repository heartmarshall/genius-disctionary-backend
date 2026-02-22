package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock
// ---------------------------------------------------------------------------

type refCatalogServiceMock struct {
	GetRelationsByEntryIDFunc func(ctx context.Context, entryID uuid.UUID) ([]domain.RefWordRelation, error)
	GetAllDataSourcesFunc     func(ctx context.Context) ([]domain.RefDataSource, error)
	GetDataSourceBySlugFunc   func(ctx context.Context, slug string) (*domain.RefDataSource, error)
	GetCoverageByEntryIDFunc  func(ctx context.Context, entryID uuid.UUID) ([]domain.RefEntrySourceCoverage, error)
	GetRefEntryByIDFunc       func(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error)
}

func (m *refCatalogServiceMock) GetRelationsByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefWordRelation, error) {
	return m.GetRelationsByEntryIDFunc(ctx, entryID)
}

func (m *refCatalogServiceMock) GetAllDataSources(ctx context.Context) ([]domain.RefDataSource, error) {
	return m.GetAllDataSourcesFunc(ctx)
}

func (m *refCatalogServiceMock) GetDataSourceBySlug(ctx context.Context, slug string) (*domain.RefDataSource, error) {
	return m.GetDataSourceBySlugFunc(ctx, slug)
}

func (m *refCatalogServiceMock) GetCoverageByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefEntrySourceCoverage, error) {
	return m.GetCoverageByEntryIDFunc(ctx, entryID)
}

func (m *refCatalogServiceMock) GetRefEntryByID(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error) {
	return m.GetRefEntryByIDFunc(ctx, id)
}

// ---------------------------------------------------------------------------
// Query: refDataSources
// ---------------------------------------------------------------------------

func TestRefDataSources_Success(t *testing.T) {
	t.Parallel()

	mock := &refCatalogServiceMock{
		GetAllDataSourcesFunc: func(_ context.Context) ([]domain.RefDataSource, error) {
			return []domain.RefDataSource{
				{Slug: "freedict", Name: "Free Dictionary", SourceType: "definitions", IsActive: true},
				{Slug: "wordnet", Name: "WordNet", SourceType: "relations", IsActive: true},
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.RefDataSources(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "freedict", result[0].Slug)
	assert.Equal(t, "wordnet", result[1].Slug)
}

func TestRefDataSources_Empty(t *testing.T) {
	t.Parallel()

	mock := &refCatalogServiceMock{
		GetAllDataSourcesFunc: func(_ context.Context) ([]domain.RefDataSource, error) {
			return []domain.RefDataSource{}, nil
		},
	}

	resolver := &queryResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.RefDataSources(context.Background())

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestRefDataSources_Error(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	mock := &refCatalogServiceMock{
		GetAllDataSourcesFunc: func(_ context.Context) ([]domain.RefDataSource, error) {
			return nil, dbErr
		},
	}

	resolver := &queryResolver{&Resolver{refCatalog: mock}}
	_, err := resolver.RefDataSources(context.Background())

	require.ErrorIs(t, err, dbErr)
}

// ---------------------------------------------------------------------------
// Query: refEntryRelations
// ---------------------------------------------------------------------------

func TestRefEntryRelations_Success(t *testing.T) {
	t.Parallel()

	entryID := uuid.New()
	targetID := uuid.New()
	mock := &refCatalogServiceMock{
		GetRelationsByEntryIDFunc: func(_ context.Context, id uuid.UUID) ([]domain.RefWordRelation, error) {
			assert.Equal(t, entryID, id)
			return []domain.RefWordRelation{
				{ID: uuid.New(), SourceEntryID: entryID, TargetEntryID: targetID, RelationType: "synonym", SourceSlug: "wordnet"},
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.RefEntryRelations(context.Background(), entryID)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "synonym", result[0].RelationType)
}

func TestRefEntryRelations_Error(t *testing.T) {
	t.Parallel()

	dbErr := errors.New("db error")
	mock := &refCatalogServiceMock{
		GetRelationsByEntryIDFunc: func(_ context.Context, _ uuid.UUID) ([]domain.RefWordRelation, error) {
			return nil, dbErr
		},
	}

	resolver := &queryResolver{&Resolver{refCatalog: mock}}
	_, err := resolver.RefEntryRelations(context.Background(), uuid.New())

	require.ErrorIs(t, err, dbErr)
}

// ---------------------------------------------------------------------------
// Field: RefEntry.relations
// ---------------------------------------------------------------------------

func TestRefEntry_Relations(t *testing.T) {
	t.Parallel()

	entryID := uuid.New()
	mock := &refCatalogServiceMock{
		GetRelationsByEntryIDFunc: func(_ context.Context, id uuid.UUID) ([]domain.RefWordRelation, error) {
			assert.Equal(t, entryID, id)
			return []domain.RefWordRelation{
				{ID: uuid.New(), RelationType: "antonym"},
			}, nil
		},
	}

	resolver := &refEntryResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.Relations(context.Background(), &domain.RefEntry{ID: entryID})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "antonym", result[0].RelationType)
}

// ---------------------------------------------------------------------------
// Field: RefEntry.sourceCoverage
// ---------------------------------------------------------------------------

func TestRefEntry_SourceCoverage(t *testing.T) {
	t.Parallel()

	entryID := uuid.New()
	mock := &refCatalogServiceMock{
		GetCoverageByEntryIDFunc: func(_ context.Context, id uuid.UUID) ([]domain.RefEntrySourceCoverage, error) {
			assert.Equal(t, entryID, id)
			return []domain.RefEntrySourceCoverage{
				{RefEntryID: entryID, SourceSlug: "freedict", Status: "fetched"},
			}, nil
		},
	}

	resolver := &refEntryResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.SourceCoverage(context.Background(), &domain.RefEntry{ID: entryID})

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "fetched", result[0].Status)
}

// ---------------------------------------------------------------------------
// Field: RefEntrySourceCoverage.source
// ---------------------------------------------------------------------------

func TestRefEntrySourceCoverage_Source(t *testing.T) {
	t.Parallel()

	expected := &domain.RefDataSource{Slug: "freedict", Name: "Free Dictionary"}
	mock := &refCatalogServiceMock{
		GetDataSourceBySlugFunc: func(_ context.Context, slug string) (*domain.RefDataSource, error) {
			assert.Equal(t, "freedict", slug)
			return expected, nil
		},
	}

	resolver := &refEntrySourceCoverageResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.Source(context.Background(), &domain.RefEntrySourceCoverage{SourceSlug: "freedict"})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

// ---------------------------------------------------------------------------
// Field: RefWordRelation.sourceEntry / targetEntry
// ---------------------------------------------------------------------------

func TestRefWordRelation_SourceEntry(t *testing.T) {
	t.Parallel()

	sourceID := uuid.New()
	expected := &domain.RefEntry{ID: sourceID, Text: "hello"}
	mock := &refCatalogServiceMock{
		GetRefEntryByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.RefEntry, error) {
			assert.Equal(t, sourceID, id)
			return expected, nil
		},
	}

	resolver := &refWordRelationResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.SourceEntry(context.Background(), &domain.RefWordRelation{SourceEntryID: sourceID})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestRefWordRelation_TargetEntry(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	expected := &domain.RefEntry{ID: targetID, Text: "world"}
	mock := &refCatalogServiceMock{
		GetRefEntryByIDFunc: func(_ context.Context, id uuid.UUID) (*domain.RefEntry, error) {
			assert.Equal(t, targetID, id)
			return expected, nil
		},
	}

	resolver := &refWordRelationResolver{&Resolver{refCatalog: mock}}
	result, err := resolver.TargetEntry(context.Background(), &domain.RefWordRelation{TargetEntryID: targetID})

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}
