package content

import (
	"context"
	"fmt"
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
)

// ---------------------------------------------------------------------------
// Consumer-defined interfaces (private)
// ---------------------------------------------------------------------------

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
}

type senseRepo interface {
	GetByID(ctx context.Context, senseID uuid.UUID) (*domain.Sense, error)
	GetByEntryID(ctx context.Context, entryID uuid.UUID) ([]domain.Sense, error)
	CountByEntry(ctx context.Context, entryID uuid.UUID) (int, error)
	CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error)
	Update(ctx context.Context, senseID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string) (*domain.Sense, error)
	Delete(ctx context.Context, senseID uuid.UUID) error
	Reorder(ctx context.Context, items []domain.ReorderItem) error
}

type translationRepo interface {
	GetByID(ctx context.Context, translationID uuid.UUID) (*domain.Translation, error)
	GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Translation, error)
	CountBySense(ctx context.Context, senseID uuid.UUID) (int, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
	Update(ctx context.Context, translationID uuid.UUID, text string) (*domain.Translation, error)
	Delete(ctx context.Context, translationID uuid.UUID) error
	Reorder(ctx context.Context, items []domain.ReorderItem) error
}

type exampleRepo interface {
	GetByID(ctx context.Context, exampleID uuid.UUID) (*domain.Example, error)
	GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error)
	CountBySense(ctx context.Context, senseID uuid.UUID) (int, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
	Update(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error)
	Delete(ctx context.Context, exampleID uuid.UUID) error
	Reorder(ctx context.Context, items []domain.ReorderItem) error
}

type imageRepo interface {
	GetUserByID(ctx context.Context, imageID uuid.UUID) (*domain.UserImage, error)
	CreateUser(ctx context.Context, entryID uuid.UUID, url string, caption *string) (*domain.UserImage, error)
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

// ---------------------------------------------------------------------------
// Ownership helpers (private)
// ---------------------------------------------------------------------------

// checkEntryOwnership verifies that the entry belongs to the user.
// Returns the entry for use in audit/logic.
func (s *Service) checkEntryOwnership(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error) {
	entry, err := s.entries.GetByID(ctx, userID, entryID)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// checkSenseOwnership loads the sense and verifies ownership of its parent entry.
// Returns the sense and entry.
func (s *Service) checkSenseOwnership(ctx context.Context, userID uuid.UUID, senseID uuid.UUID) (*domain.Sense, *domain.Entry, error) {
	sense, err := s.senses.GetByID(ctx, senseID)
	if err != nil {
		return nil, nil, err
	}

	entry, err := s.checkEntryOwnership(ctx, userID, sense.EntryID)
	if err != nil {
		// Sense exists but entry is not owned/deleted - return ErrNotFound
		if err == domain.ErrNotFound {
			return nil, nil, domain.ErrNotFound
		}
		return nil, nil, fmt.Errorf("check entry ownership: %w", err)
	}

	return sense, entry, nil
}

// checkTranslationOwnership loads the translation and verifies ownership chain.
// Returns the translation and entry.
func (s *Service) checkTranslationOwnership(ctx context.Context, userID uuid.UUID, translationID uuid.UUID) (*domain.Translation, *domain.Entry, error) {
	translation, err := s.translations.GetByID(ctx, translationID)
	if err != nil {
		return nil, nil, err
	}

	_, entry, err := s.checkSenseOwnership(ctx, userID, translation.SenseID)
	if err != nil {
		return nil, nil, err
	}

	return translation, entry, nil
}

// checkExampleOwnership loads the example and verifies ownership chain.
// Returns the example and entry.
func (s *Service) checkExampleOwnership(ctx context.Context, userID uuid.UUID, exampleID uuid.UUID) (*domain.Example, *domain.Entry, error) {
	example, err := s.examples.GetByID(ctx, exampleID)
	if err != nil {
		return nil, nil, err
	}

	_, entry, err := s.checkSenseOwnership(ctx, userID, example.SenseID)
	if err != nil {
		return nil, nil, err
	}

	return example, entry, nil
}

// checkUserImageOwnership loads the user image and verifies ownership chain.
// Returns the image and entry.
func (s *Service) checkUserImageOwnership(ctx context.Context, userID uuid.UUID, imageID uuid.UUID) (*domain.UserImage, *domain.Entry, error) {
	image, err := s.images.GetUserByID(ctx, imageID)
	if err != nil {
		return nil, nil, err
	}

	entry, err := s.checkEntryOwnership(ctx, userID, image.EntryID)
	if err != nil {
		return nil, nil, err
	}

	return image, entry, nil
}
