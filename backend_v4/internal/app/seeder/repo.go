// Package seeder defines interfaces and orchestration for the dataset seeding pipeline.
package seeder

import (
	"context"

	"github.com/google/uuid"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// RefEntryBulkRepo defines the batch repository contract consumed by the seeder pipeline.
// All methods use only domain types — no adapter imports.
// Implemented by refentry.Repo.
type RefEntryBulkRepo interface {
	// Batch inserts — ON CONFLICT DO NOTHING (except BulkInsertCoverage).
	BulkInsertEntries(ctx context.Context, entries []domain.RefEntry) (int, error)
	BulkInsertSenses(ctx context.Context, senses []domain.RefSense) (int, error)
	BulkInsertTranslations(ctx context.Context, translations []domain.RefTranslation) (int, error)
	BulkInsertExamples(ctx context.Context, examples []domain.RefExample) (int, error)
	BulkInsertPronunciations(ctx context.Context, pronunciations []domain.RefPronunciation) (int, error)
	BulkInsertRelations(ctx context.Context, relations []domain.RefWordRelation) (int, error)
	BulkInsertCoverage(ctx context.Context, coverage []domain.RefEntrySourceCoverage) (int, error)

	// Replace — delete+insert for LLM enrichment.
	ReplaceEntryContent(ctx context.Context, entryID uuid.UUID, senses []domain.RefSense, translations []domain.RefTranslation, examples []domain.RefExample) error

	// Batch update — metadata enrichment via COALESCE.
	BulkUpdateEntryMetadata(ctx context.Context, updates []domain.EntryMetadataUpdate) (int, error)

	// Lookups — resolve words to UUIDs for cross-referencing.
	GetEntryIDsByNormalizedTexts(ctx context.Context, texts []string) (map[string]uuid.UUID, error)
	GetAllNormalizedTexts(ctx context.Context) (map[string]bool, error)
	GetFirstSenseIDsByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	GetPronunciationIPAsByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID]map[string]bool, error)

	// Registry — data source versioning.
	UpsertDataSources(ctx context.Context, sources []domain.RefDataSource) error
}
