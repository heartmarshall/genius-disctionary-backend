package refcatalog

import (
	"context"
	"errors"
	"fmt"
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

// Search finds reference entries matching the query. The catalog is shared (no userID required).
// An empty query returns an empty result. Limit is clamped to [1, 50], defaulting to 20.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	if query == "" {
		return []domain.RefEntry{}, nil
	}

	limit = clampLimit(limit)

	return s.refEntries.Search(ctx, query, limit)
}

// GetOrFetchEntry returns an existing reference entry or fetches it from external providers.
// External HTTP calls are made outside the transaction. If a concurrent insert races,
// the existing entry is returned.
func (s *Service) GetOrFetchEntry(ctx context.Context, text string) (*domain.RefEntry, error) {
	normalized := domain.NormalizeText(text)
	if normalized == "" {
		return nil, domain.NewValidationError("text", "required")
	}

	// 1. Check if the entry already exists in the catalog.
	existing, err := s.refEntries.GetFullTreeByText(ctx, normalized)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("get ref entry by text: %w", err)
	}

	// 2. Fetch from dictionary provider (outside transaction).
	dictResult, err := s.dictProvider.FetchEntry(ctx, text)
	if err != nil {
		s.log.ErrorContext(ctx, "dictionary provider error",
			slog.String("word", text),
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("fetch entry: %w", err)
	}
	if dictResult == nil {
		return nil, ErrWordNotFound
	}

	// 3. Fetch translations (graceful degradation on error).
	translations, err := s.transProvider.FetchTranslations(ctx, text)
	if err != nil {
		s.log.WarnContext(ctx, "translation provider error, proceeding without translations",
			slog.String("word", text),
			slog.String("error", err.Error()),
		)
		translations = nil
	}

	// 4. Map to domain model.
	refEntry := mapToRefEntry(normalized, dictResult, translations)

	// 5. Save within a transaction.
	var saved *domain.RefEntry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var createErr error
		saved, createErr = s.refEntries.CreateWithTree(txCtx, refEntry)
		return createErr
	})

	if txErr != nil {
		// Handle concurrent create: another goroutine inserted the same entry.
		if errors.Is(txErr, domain.ErrAlreadyExists) {
			existing, err := s.refEntries.GetFullTreeByText(ctx, normalized)
			if err != nil {
				return nil, fmt.Errorf("get ref entry after conflict: %w", err)
			}
			return existing, nil
		}
		return nil, fmt.Errorf("create ref entry: %w", txErr)
	}

	s.log.InfoContext(ctx, "ref entry fetched and saved",
		slog.String("word", text),
		slog.String("entry_id", saved.ID.String()),
	)

	return saved, nil
}

// GetRefEntry returns a reference entry by its ID.
func (s *Service) GetRefEntry(ctx context.Context, refEntryID uuid.UUID) (*domain.RefEntry, error) {
	entry, err := s.refEntries.GetFullTreeByID(ctx, refEntryID)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// clampLimit ensures the limit is within [1, 50], defaulting 0 to 20.
func clampLimit(limit int) int {
	if limit <= 0 {
		limit = 1
	}
	if limit > 50 {
		limit = 50
	}
	return limit
}
