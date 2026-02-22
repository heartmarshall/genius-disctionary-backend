package refcatalog

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/provider"
)

type refEntryRepo interface {
	Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
	GetFullTreeByID(ctx context.Context, id uuid.UUID) (*domain.RefEntry, error)
	GetFullTreeByText(ctx context.Context, textNormalized string) (*domain.RefEntry, error)
	CreateWithTree(ctx context.Context, entry *domain.RefEntry) (*domain.RefEntry, error)
	GetRelationsByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefWordRelation, error)
	GetAllDataSources(ctx context.Context) ([]domain.RefDataSource, error)
	GetDataSourceBySlug(ctx context.Context, slug string) (*domain.RefDataSource, error)
	GetCoverageByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.RefEntrySourceCoverage, error)
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type dictionaryProvider interface {
	FetchEntry(ctx context.Context, word string) (*provider.DictionaryResult, error)
}

type translationProvider interface {
	FetchTranslations(ctx context.Context, word string) ([]string, error)
}

// Service implements reference catalog operations: search, fetch-or-create, and get.
type Service struct {
	log           *slog.Logger
	refEntries    refEntryRepo
	tx            txManager
	dictProvider  dictionaryProvider
	transProvider translationProvider
}

// NewService creates a new RefCatalog service.
func NewService(
	logger *slog.Logger,
	refEntries refEntryRepo,
	tx txManager,
	dictProvider dictionaryProvider,
	transProvider translationProvider,
) *Service {
	return &Service{
		log:           logger.With("service", "refcatalog"),
		refEntries:    refEntries,
		tx:            tx,
		dictProvider:  dictProvider,
		transProvider: transProvider,
	}
}

// clampLimit ensures the limit is within [1, 50], defaulting 0 to 20.
func clampLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 50 {
		return 50
	}
	return limit
}
