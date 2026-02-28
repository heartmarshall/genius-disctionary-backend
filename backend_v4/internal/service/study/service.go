package study

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study/fsrs"
)

// ---------------------------------------------------------------------------
// Consumer-defined interfaces (private)
// ---------------------------------------------------------------------------

type cardRepo interface {
	GetByID(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
	GetByIDForUpdate(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
	GetByEntryID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error)
	Create(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error)
	UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error)
	Delete(ctx context.Context, userID, cardID uuid.UUID) error
	GetDueCards(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*domain.Card, error)
	GetNewCards(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Card, error)
	CountByStatus(ctx context.Context, userID uuid.UUID) (domain.CardStatusCounts, error)
	CountDue(ctx context.Context, userID uuid.UUID, now time.Time) (int, error)
	CountNew(ctx context.Context, userID uuid.UUID) (int, error)
	CountOverdue(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
	ExistsByEntryIDs(ctx context.Context, userID uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type reviewLogRepo interface {
	Create(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error)
	GetByCardID(ctx context.Context, cardID uuid.UUID, limit, offset int) ([]*domain.ReviewLog, int, error)
	GetLastByCardID(ctx context.Context, cardID uuid.UUID) (*domain.ReviewLog, error)
	Delete(ctx context.Context, id uuid.UUID) error
	CountToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
	CountNewToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
	GetStreakDays(ctx context.Context, userID uuid.UUID, dayStart time.Time, lastNDays int, timezone string) ([]domain.DayReviewCount, error)
	GetByPeriod(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.ReviewLog, error)
	GetStatsByCardID(ctx context.Context, cardID uuid.UUID) (domain.ReviewLogAggregation, error)
}

type sessionRepo interface {
	Create(ctx context.Context, session *domain.StudySession) (*domain.StudySession, error)
	GetByID(ctx context.Context, userID, sessionID uuid.UUID) (*domain.StudySession, error)
	GetActive(ctx context.Context, userID uuid.UUID) (*domain.StudySession, error)
	Finish(ctx context.Context, userID, sessionID uuid.UUID, result domain.SessionResult) (*domain.StudySession, error)
	Abandon(ctx context.Context, userID, sessionID uuid.UUID) error
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.StudySession, int, error)
}

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	GetByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error)
	ExistByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

type senseRepo interface {
	CountByEntryID(ctx context.Context, entryID uuid.UUID) (int, error)
	CountByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) (map[uuid.UUID]int, error)
}

type settingsRepo interface {
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
}

type auditLogger interface {
	Log(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the study business logic.
type Service struct {
	cards       cardRepo
	reviews     reviewLogRepo
	sessions    sessionRepo
	entries     entryRepo
	senses      senseRepo
	settings    settingsRepo
	audit       auditLogger
	tx          txManager
	log         *slog.Logger
	srsConfig   domain.SRSConfig
	fsrsWeights [19]float64
}

// NewService creates a new Study service.
func NewService(
	log *slog.Logger,
	cards cardRepo,
	reviews reviewLogRepo,
	sessions sessionRepo,
	entries entryRepo,
	senses senseRepo,
	settings settingsRepo,
	audit auditLogger,
	tx txManager,
	srsConfig domain.SRSConfig,
	fsrsWeights [19]float64,
) (*Service, error) {
	if err := fsrs.ValidateWeights(fsrsWeights); err != nil {
		return nil, fmt.Errorf("invalid FSRS weights: %w", err)
	}

	return &Service{
		cards:       cards,
		reviews:     reviews,
		sessions:    sessions,
		entries:     entries,
		senses:      senses,
		settings:    settings,
		audit:       audit,
		tx:          tx,
		log:         log.With("service", "study"),
		srsConfig:   srsConfig,
		fsrsWeights: fsrsWeights,
	}, nil
}
