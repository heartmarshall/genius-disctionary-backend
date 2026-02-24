package dictionary

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// Consumer-defined interfaces (private)
// ---------------------------------------------------------------------------

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	GetByText(ctx context.Context, userID uuid.UUID, textNormalized string) (*domain.Entry, error)
	GetByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error)
	Find(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, int, error)
	FindCursor(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, bool, error)
	FindDeleted(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Entry, int, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	Create(ctx context.Context, entry *domain.Entry) (*domain.Entry, error)
	UpdateNotes(ctx context.Context, userID, entryID uuid.UUID, notes *string) (*domain.Entry, error)
	SoftDelete(ctx context.Context, userID, entryID uuid.UUID) error
	Restore(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	HardDeleteOld(ctx context.Context, threshold time.Time) (int64, error)
}

type senseRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Sense, error)
	CreateFromRef(ctx context.Context, entryID, refSenseID uuid.UUID, sourceSlug string) (*domain.Sense, error)
	CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error)
}

type translationRepo interface {
	GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Translation, error)
	CreateFromRef(ctx context.Context, senseID, refTranslationID uuid.UUID, sourceSlug string) (*domain.Translation, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
}

type exampleRepo interface {
	GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Example, error)
	CreateFromRef(ctx context.Context, senseID, refExampleID uuid.UUID, sourceSlug string) (*domain.Example, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
}

type pronunciationRepo interface {
	Link(ctx context.Context, entryID, refPronunciationID uuid.UUID) error
}

type imageRepo interface {
	LinkCatalog(ctx context.Context, entryID, refImageID uuid.UUID) error
}

type cardRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Card, error)
	Create(ctx context.Context, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) (*domain.Card, error)
}

type auditRepo interface {
	Create(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error)
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type enrichmentEnqueuer interface {
	Enqueue(ctx context.Context, refEntryID uuid.UUID) error
}

type refCatalogService interface {
	GetOrFetchEntry(ctx context.Context, text string) (*domain.RefEntry, error)
	GetRefEntry(ctx context.Context, refEntryID uuid.UUID) (*domain.RefEntry, error)
	Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the dictionary business logic.
type Service struct {
	log            *slog.Logger
	entries        entryRepo
	senses         senseRepo
	translations   translationRepo
	examples       exampleRepo
	pronunciations pronunciationRepo
	images         imageRepo
	cards          cardRepo
	audit          auditRepo
	tx             txManager
	refCatalog     refCatalogService
	enrichment     enrichmentEnqueuer
	cfg            config.DictionaryConfig
}

// NewService creates a new Dictionary service.
func NewService(
	logger *slog.Logger,
	entries entryRepo,
	senses senseRepo,
	translations translationRepo,
	examples exampleRepo,
	pronunciations pronunciationRepo,
	images imageRepo,
	cards cardRepo,
	audit auditRepo,
	tx txManager,
	refCatalog refCatalogService,
	cfg config.DictionaryConfig,
) *Service {
	return &Service{
		log:            logger.With("service", "dictionary"),
		entries:        entries,
		senses:         senses,
		translations:   translations,
		examples:       examples,
		pronunciations: pronunciations,
		images:         images,
		cards:          cards,
		audit:          audit,
		tx:             tx,
		refCatalog:     refCatalog,
		cfg:            cfg,
	}
}

// SetEnrichment injects the optional enrichment enqueuer.
func (s *Service) SetEnrichment(e enrichmentEnqueuer) {
	s.enrichment = e
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// clampLimit ensures a limit is within [min, max], defaulting from 0 to defaultVal.
func clampLimit(limit, min, max, defaultVal int) int {
	if limit <= 0 {
		return defaultVal
	}
	if limit < min {
		return min
	}
	if limit > max {
		return max
	}
	return limit
}
