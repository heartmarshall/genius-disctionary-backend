// Package cards содержит репозитории для работы с карточками и логами ревью.
package cards

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/repository/base"
	"github.com/heartmarshall/my-english/internal/database/schema"
	"github.com/heartmarshall/my-english/internal/model"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	// DefaultDueCardsLimit — лимит по умолчанию для получения карточек к повторению.
	DefaultDueCardsLimit = 10

	// MaxDueCardsLimit — максимальный лимит для получения карточек к повторению.
	MaxDueCardsLimit = 1000

	// DefaultEaseFactor — значение ease factor по умолчанию для новых карточек.
	// Стандартное значение из алгоритма SM-2.
	DefaultEaseFactor = 2.5

	// MinEaseFactor — минимальное значение ease factor.
	MinEaseFactor = 1.3

	// MaxReviewLogLimit — максимальный лимит для истории повторений.
	MaxReviewLogLimit = 1000
)

// ============================================================================
// DTO TYPES
// ============================================================================

// DashboardStats содержит агрегированную статистику для дашборда.
type DashboardStats struct {
	TotalWords    int `db:"total_words"`
	TotalCards    int `db:"total_cards"`
	NewCards      int `db:"new_cards"`
	LearningCards int `db:"learning_cards"`
	ReviewCards   int `db:"review_cards"`
	MasteredCards int `db:"mastered_cards"`
	DueToday      int `db:"due_today"`
}

// ============================================================================
// CARD REPOSITORY
// ============================================================================

// CardRepository предоставляет методы для работы с карточками.
type CardRepository struct {
	*base.Base[model.Card]
}

// NewCardRepository создаёт новый репозиторий карточек.
func NewCardRepository(q database.Querier) *CardRepository {
	return &CardRepository{
		Base: base.MustNewBase[model.Card](q, base.Config{
			Table:   schema.Cards.Name.String(),
			Columns: schema.Cards.Columns(),
		}),
	}
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

// GetByID получает карточку по ID.
func (r *CardRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Card, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}
	return r.Base.GetByID(ctx, schema.Cards.ID.Bare(), id)
}

// GetByEntryID получает карточку по ID записи словаря.
func (r *CardRepository) GetByEntryID(ctx context.Context, entryID uuid.UUID) (*model.Card, error) {
	if err := base.ValidateUUID(entryID, "entry_id"); err != nil {
		return nil, err
	}
	return r.FindOneBy(ctx, schema.Cards.EntryID.Bare(), entryID)
}

// GetByIDForUpdate получает карточку с блокировкой строки (SELECT FOR UPDATE).
// Используется в транзакциях для предотвращения race condition при обновлении.
func (r *CardRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*model.Card, error) {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}

	query := r.SelectBuilder().
		Where(squirrel.Eq{schema.Cards.ID.Bare(): id}).
		Suffix("FOR UPDATE")

	return r.GetOne(ctx, query)
}

// GetDueCards получает карточки, которые нужно повторить до указанного времени.
// Карточки сортируются по времени следующего повторения (самые просроченные первыми).
// Включает карточки со статусом NEW (новые карточки) и карточки с nextReviewAt <= now.
func (r *CardRepository) GetDueCards(ctx context.Context, now time.Time, limit int) ([]model.Card, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	// Нормализуем лимит
	if limit <= 0 {
		limit = DefaultDueCardsLimit
	}
	if limit > MaxDueCardsLimit {
		limit = MaxDueCardsLimit
	}

	// Включаем карточки со статусом NEW (новые карточки) или с nextReviewAt <= now или NULL
	// Используем squirrel.Or для OR условий
	query := r.SelectBuilder().
		Where(squirrel.Or{
			squirrel.Eq{schema.Cards.Status.Bare(): "NEW"},
			squirrel.LtOrEq{schema.Cards.NextReviewAt.Bare(): now},
			schema.Cards.NextReviewAt.IsNull(),
		}).
		OrderBy(
			// Сначала новые карточки (NEW), затем по времени следующего повторения
			"CASE WHEN "+schema.Cards.Status.Bare()+" = 'NEW' THEN 0 ELSE 1 END ASC",
			schema.Cards.NextReviewAt.Bare()+" ASC NULLS FIRST",
		).
		Limit(uint64(limit))

	return r.List(ctx, query)
}

// GetDashboardStats возвращает агрегированную статистику для дашборда.
// Выполняет один оптимизированный запрос вместо множества.
func (r *CardRepository) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	// Используем PostgreSQL FILTER для эффективного подсчёта
	// FILTER быстрее CASE WHEN для агрегации
	sql := `
		SELECT
			(SELECT COUNT(*) FROM dictionary_entries)::int as total_words,
			COUNT(*)::int as total_cards,
			COUNT(*) FILTER (WHERE status = 'NEW')::int as new_cards,
			COUNT(*) FILTER (WHERE status = 'LEARNING')::int as learning_cards,
			COUNT(*) FILTER (WHERE status = 'REVIEW')::int as review_cards,
			COUNT(*) FILTER (WHERE status = 'MASTERED')::int as mastered_cards,
			COUNT(*) FILTER (WHERE next_review_at <= NOW())::int as due_today
		FROM cards
	`

	var stats DashboardStats
	if err := r.QueryRowRaw(ctx, &stats, sql); err != nil {
		return nil, err
	}
	return &stats, nil
}

// ============================================================================
// WRITE OPERATIONS
// ============================================================================

// Create создает новую карточку с дефолтными значениями.
//
// Дефолтные значения:
//   - Status: StatusNew (если не указан)
//   - EaseFactor: DefaultEaseFactor (если равен 0). EraseFactor это коэффициент легкости, который используется для расчета интервала повторения.
func (r *CardRepository) Create(ctx context.Context, card *model.Card) (*model.Card, error) {
	if card == nil {
		return nil, fmt.Errorf("%w: card is required", database.ErrInvalidInput)
	}
	if err := base.ValidateUUID(card.EntryID, "entry_id"); err != nil {
		return nil, err
	}

	// Применяем дефолтные значения (не мутируем входной объект)
	status := card.Status
	if status == "" {
		status = model.StatusNew
	}

	easeFactor := card.EaseFactor
	if easeFactor == 0 {
		easeFactor = DefaultEaseFactor
	}

	insert := r.InsertBuilder().
		Columns(schema.Cards.InsertColumns()...).
		Values(
			card.EntryID,
			status,
			card.NextReviewAt,
			card.IntervalDays,
			easeFactor,
		)

	return r.InsertReturning(ctx, insert)
}

// Update обновляет карточку полностью.
func (r *CardRepository) Update(ctx context.Context, id uuid.UUID, card *model.Card) (*model.Card, error) {
	if card == nil {
		return nil, fmt.Errorf("%w: card is required", database.ErrInvalidInput)
	}
	if err := base.ValidateUUID(id, "id"); err != nil {
		return nil, err
	}

	update := r.UpdateBuilder().
		Set("status", card.Status).
		Set("next_review_at", card.NextReviewAt).
		Set("interval_days", card.IntervalDays).
		Set("ease_factor", card.EaseFactor).
		Where(squirrel.Eq{schema.Cards.ID.Bare(): id})

	return r.Base.Update(ctx, update)
}

// UpdateSRSFields обновляет только SRS поля карточки после ревью.
//
// Параметры:
//   - id: ID карточки
//   - status: новый статус обучения
//   - nextReviewAt: время следующего повторения (может быть nil для MASTERED)
//   - intervalDays: интервал в днях (≥0)
//   - easeFactor: фактор легкости (≥1.3)
func (r *CardRepository) UpdateSRSFields(
	ctx context.Context,
	id uuid.UUID,
	status model.LearningStatus,
	nextReviewAt *time.Time,
	intervalDays int,
	easeFactor float64,
) error {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return database.WrapDBError(err)
	}

	// Валидация входных данных
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	if status == "" {
		return fmt.Errorf("%w: status is required", database.ErrInvalidInput)
	}
	if !status.IsValid() {
		return fmt.Errorf("%w: invalid status: %s", database.ErrInvalidInput, status)
	}
	if easeFactor < MinEaseFactor {
		return fmt.Errorf("%w: ease_factor must be >= %.1f, got %.2f", database.ErrInvalidInput, MinEaseFactor, easeFactor)
	}
	if intervalDays < 0 {
		return fmt.Errorf("%w: interval_days must be >= 0, got %d", database.ErrInvalidInput, intervalDays)
	}

	update := r.UpdateBuilder().
		Set("status", status).
		Set("next_review_at", nextReviewAt).
		Set("interval_days", intervalDays).
		Set("ease_factor", easeFactor).
		Where(squirrel.Eq{schema.Cards.ID.Bare(): id})

	_, err := r.Base.Update(ctx, update)
	return err
}

// Delete удаляет карточку.
func (r *CardRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := base.ValidateUUID(id, "id"); err != nil {
		return err
	}
	return r.Base.Delete(ctx, schema.Cards.ID.Bare(), id)
}

// ============================================================================
// REVIEW LOG REPOSITORY
// ============================================================================

// ReviewLogRepository предоставляет методы для работы с логами ревью.
type ReviewLogRepository struct {
	*base.Base[model.ReviewLog]
}

// NewReviewLogRepository создаёт новый репозиторий логов ревью.
func NewReviewLogRepository(q database.Querier) *ReviewLogRepository {
	return &ReviewLogRepository{
		Base: base.MustNewBase[model.ReviewLog](q, base.Config{
			Table:   schema.ReviewLogs.Name.String(),
			Columns: schema.ReviewLogs.Columns(),
		}),
	}
}

// Create создает запись о ревью.
func (r *ReviewLogRepository) Create(ctx context.Context, log *model.ReviewLog) (*model.ReviewLog, error) {
	if log == nil {
		return nil, fmt.Errorf("%w: log is required", database.ErrInvalidInput)
	}
	if err := base.ValidateUUID(log.CardID, "card_id"); err != nil {
		return nil, err
	}
	if log.Grade == "" {
		return nil, fmt.Errorf("%w: grade is required", database.ErrInvalidInput)
	}

	insert := r.InsertBuilder().
		Columns(schema.ReviewLogs.InsertColumns()...).
		Values(log.CardID, log.Grade, log.DurationMs)

	return r.InsertReturning(ctx, insert)
}

// ListByCardID возвращает историю повторений для карточки.
// Записи сортируются по дате повторения (новые первыми).
func (r *ReviewLogRepository) ListByCardID(ctx context.Context, cardID uuid.UUID, limit int) ([]model.ReviewLog, error) {
	if err := base.ValidateUUID(cardID, "card_id"); err != nil {
		return nil, err
	}

	query := r.SelectBuilder().
		Where(squirrel.Eq{schema.ReviewLogs.CardID.Bare(): cardID}).
		OrderBy(schema.ReviewLogs.ReviewedAt.Bare() + " DESC")

	// Применяем лимит если указан
	if limit > 0 {
		if limit > MaxReviewLogLimit {
			limit = MaxReviewLogLimit
		}
		query = query.Limit(uint64(limit))
	}

	return r.List(ctx, query)
}

// ListByEntryIDs возвращает список карточек для указанных entryIDs.
// Используется для DataLoaders.
func (r *CardRepository) ListByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]model.Card, error) {
	if len(entryIDs) == 0 {
		return []model.Card{}, nil
	}
	// Используем базовый метод, указывая колонку entry_id
	return r.ListByUUIDs(ctx, schema.Cards.EntryID.Bare(), entryIDs)
}
