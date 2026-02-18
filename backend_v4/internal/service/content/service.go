package content

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	MaxSensesPerEntry       = 20
	MaxTranslationsPerSense = 20
	MaxExamplesPerSense     = 50
	MaxUserImagesPerEntry   = 20
)

// ValidCEFRLevels is the set of valid CEFR language proficiency levels.
var ValidCEFRLevels = map[string]bool{
	"A1": true, "A2": true,
	"B1": true, "B2": true,
	"C1": true, "C2": true,
}

// ---------------------------------------------------------------------------
// Consumer-defined interfaces (private)
// ---------------------------------------------------------------------------

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
}

type senseRepo interface {
	GetByIDForUser(ctx context.Context, userID, senseID uuid.UUID) (*domain.Sense, error)
	GetByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.Sense, error)
	CountByEntry(ctx context.Context, entryID uuid.UUID) (int, error)
	CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error)
	Update(ctx context.Context, senseID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error)
	Delete(ctx context.Context, senseID uuid.UUID) error
	Reorder(ctx context.Context, items []domain.ReorderItem) error
}

type translationRepo interface {
	GetByIDForUser(ctx context.Context, userID, translationID uuid.UUID) (*domain.Translation, error)
	GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Translation, error)
	CountBySense(ctx context.Context, senseID uuid.UUID) (int, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
	Update(ctx context.Context, translationID uuid.UUID, text string) (*domain.Translation, error)
	Delete(ctx context.Context, translationID uuid.UUID) error
	Reorder(ctx context.Context, items []domain.ReorderItem) error
}

type exampleRepo interface {
	GetByIDForUser(ctx context.Context, userID, exampleID uuid.UUID) (*domain.Example, error)
	GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error)
	CountBySense(ctx context.Context, senseID uuid.UUID) (int, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
	Update(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error)
	Delete(ctx context.Context, exampleID uuid.UUID) error
	Reorder(ctx context.Context, items []domain.ReorderItem) error
}

type imageRepo interface {
	GetUserByIDForUser(ctx context.Context, userID, imageID uuid.UUID) (*domain.UserImage, error)
	CountUserByEntry(ctx context.Context, entryID uuid.UUID) (int, error)
	CreateUser(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error)
	UpdateUser(ctx context.Context, imageID uuid.UUID, caption *string) (*domain.UserImage, error)
	DeleteUser(ctx context.Context, imageID uuid.UUID) error
}

type auditRepo interface {
	Log(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the content business logic.
type Service struct {
	log          *slog.Logger
	entries      entryRepo
	senses       senseRepo
	translations translationRepo
	examples     exampleRepo
	images       imageRepo
	audit        auditRepo
	tx           txManager
}

// NewService creates a new Content service.
func NewService(
	logger *slog.Logger,
	entries entryRepo,
	senses senseRepo,
	translations translationRepo,
	examples exampleRepo,
	images imageRepo,
	audit auditRepo,
	tx txManager,
) *Service {
	return &Service{
		log:          logger.With("service", "content"),
		entries:      entries,
		senses:       senses,
		translations: translations,
		examples:     examples,
		images:       images,
		audit:        audit,
		tx:           tx,
	}
}
