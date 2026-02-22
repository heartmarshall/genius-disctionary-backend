package refcatalog

import (
	"context"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// GetRelationsByEntryID returns word relations for a given reference entry.
func (s *Service) GetRelationsByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefWordRelation, error) {
	return s.refEntries.GetRelationsByEntryID(ctx, entryID)
}

// GetAllDataSources returns all registered data sources.
func (s *Service) GetAllDataSources(ctx context.Context) ([]domain.RefDataSource, error) {
	return s.refEntries.GetAllDataSources(ctx)
}

// GetDataSourceBySlug returns a single data source by slug.
func (s *Service) GetDataSourceBySlug(ctx context.Context, slug string) (*domain.RefDataSource, error) {
	return s.refEntries.GetDataSourceBySlug(ctx, slug)
}

// GetCoverageByEntryID returns source coverage records for a reference entry.
func (s *Service) GetCoverageByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefEntrySourceCoverage, error) {
	return s.refEntries.GetCoverageByEntryID(ctx, entryID)
}

// GetRefEntryByID returns a reference entry by ID (without full tree).
// This is a convenience alias used by GraphQL resolvers.
func (s *Service) GetRefEntryByID(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error) {
	return s.refEntries.GetFullTreeByID(ctx, id)
}
