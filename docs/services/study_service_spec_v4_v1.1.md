# MyEnglish Backend v4 — Study Service Specification

> **Статус:** Draft v1.1
> **Дата:** 2026-02-13
> **Зависимости:** code_conventions_v4.md, data_model_v4.md, repo_layer_spec_v4.md, service_layer_spec_v4.md
> **Покрываемые сценарии:** S1–S8 из business_scenarios_v4.md + S9 (Study Session), S10 (UndoReview)

## 1. Ответственность

Study Service — центральный сервис системы интервального повторения. Отвечает за:

- Формирование очереди изучения с учётом daily limits, приоритетов и timezone пользователя
- Обработку review карточки: оценка (grade), пересчёт SRS-параметров, запись в историю
- Отмену последнего review (undo) с полным восстановлением SRS-состояния
- CRUD карточек (создание, удаление, массовое создание)
- Управление study sessions (начало, завершение, агрегация результатов)
- Dashboard: статистика, streak, счётчики по статусам
- Историю и статистику отдельной карточки

Study Service **не** отвечает за: CRUD entries (Dictionary Service), CRUD senses/translations (Content Service), управление настройками пользователя (User Service).

---

## 2. Структура пакета

```
internal/service/study/
├── service.go          # Struct, конструктор, приватные интерфейсы
├── input.go            # Input-структуры с Validate()
├── srs.go              # SRS algorithm — чистая функция
├── srs_test.go         # Table-driven тесты SRS (≥30 кейсов)
├── session.go          # Логика study sessions
├── service_test.go     # Unit-тесты сервиса с моками
└── timezone.go         # Хелперы для работы с timezone
```

---

## 3. Зависимости (приватные интерфейсы)

```go
// service/study/service.go
package study

import (
    "context"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "myenglish/internal/domain"
)

type cardRepo interface {
    GetByID(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
    GetByEntryID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error)
    Create(ctx context.Context, userID uuid.UUID, card *domain.Card) (*domain.Card, error)
    UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error)
    Delete(ctx context.Context, userID, cardID uuid.UUID) error

    // Очередь: overdue + new, с исключением soft-deleted entries
    GetDueCards(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*domain.Card, error)
    GetNewCards(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Card, error)

    // Счётчики для dashboard и лимитов
    CountByStatus(ctx context.Context, userID uuid.UUID) (domain.CardStatusCounts, error)
    CountDue(ctx context.Context, userID uuid.UUID, now time.Time) (int, error)
    CountNew(ctx context.Context, userID uuid.UUID) (int, error)

    // Для массового создания
    ExistsByEntryIDs(ctx context.Context, userID uuid.UUID, entryIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type reviewLogRepo interface {
    Create(ctx context.Context, log *domain.ReviewLog) (*domain.ReviewLog, error)
    GetByCardID(ctx context.Context, cardID uuid.UUID, limit, offset int) ([]*domain.ReviewLog, int, error)
    GetLastByCardID(ctx context.Context, cardID uuid.UUID) (*domain.ReviewLog, error)
    Delete(ctx context.Context, id uuid.UUID) error
    CountToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
    CountNewToday(ctx context.Context, userID uuid.UUID, dayStart time.Time) (int, error)
    GetStreakDays(ctx context.Context, userID uuid.UUID, dayStart time.Time, lastNDays int) ([]domain.DayReviewCount, error)
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
    ExistByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

type senseRepo interface {
    CountByEntryID(ctx context.Context, entryID uuid.UUID) (int, error)
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

type Service struct {
    cards      cardRepo
    reviews    reviewLogRepo
    sessions   sessionRepo
    entries    entryRepo
    senses     senseRepo
    settings   settingsRepo
    audit      auditLogger
    tx         txManager
    log        *slog.Logger
    srsConfig  domain.SRSConfig
}

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
) *Service {
    return &Service{
        cards:     cards,
        reviews:   reviews,
        sessions:  sessions,
        entries:   entries,
        senses:    senses,
        settings:  settings,
        audit:     audit,
        tx:        tx,
        log:       log.With("service", "study"),
        srsConfig: srsConfig,
    }
}
```

---

## 4. Domain-модели (справочно, определены в `domain/`)

```go
// domain/card.go

type LearningStatus string

const (
    StatusNew      LearningStatus = "NEW"
    StatusLearning LearningStatus = "LEARNING"
    StatusReview   LearningStatus = "REVIEW"
    StatusMastered LearningStatus = "MASTERED"
)

type ReviewGrade string

const (
    GradeAgain ReviewGrade = "AGAIN"
    GradeHard  ReviewGrade = "HARD"
    GradeGood  ReviewGrade = "GOOD"
    GradeEasy  ReviewGrade = "EASY"
)

type Card struct {
    ID            uuid.UUID
    UserID        uuid.UUID
    EntryID       uuid.UUID
    Status        LearningStatus
    NextReviewAt  *time.Time       // nil для NEW карточек
    IntervalDays  int
    EaseFactor    float64
    LearningStep  int              // Текущий шаг в learning phase (0-based)
    CreatedAt     time.Time
    UpdatedAt     time.Time

    // Денормализованные поля для отображения в очереди (из entries)
    EntryText     string           // Заполняется при загрузке очереди
}

// ReviewLog хранит историю каждого review, включая prev_state для undo.
type ReviewLog struct {
    ID         uuid.UUID
    CardID     uuid.UUID
    Grade      ReviewGrade
    DurationMs *int               // Время на ответ (опционально)
    ReviewedAt time.Time

    // Snapshot состояния карточки ДО этого review — для undo
    PrevStatus       LearningStatus
    PrevIntervalDays int
    PrevEaseFactor   float64
    PrevLearningStep int
    PrevNextReviewAt *time.Time
}

type SRSUpdateParams struct {
    Status       LearningStatus
    NextReviewAt time.Time
    IntervalDays int
    EaseFactor   float64
    LearningStep int
}

type SRSConfig struct {
    DefaultEaseFactor    float64         // 2.5
    MinEaseFactor        float64         // 1.3
    MaxIntervalDays      int             // 365 (может быть overridden user_settings)
    GraduatingInterval   int             // 1 день
    EasyInterval         int             // 4 дня
    LearningSteps        []time.Duration // [1m, 10m]
    RelearningSteps      []time.Duration // [10m]
    IntervalModifier     float64         // 1.0 (множитель для интервалов)
    HardIntervalModifier float64         // 1.2 (множитель для HARD в review)
    EasyBonus            float64         // 1.3 (множитель для EASY в review)
    LapseNewInterval     float64         // 0.0 (множитель интервала после lapse: 0 = reset)
}

type CardStatusCounts struct {
    New      int
    Learning int
    Review   int
    Mastered int
    Total    int
}

type DayReviewCount struct {
    Date  time.Time
    Count int
}

// --- Study Session ---

type SessionStatus string

const (
    SessionActive    SessionStatus = "ACTIVE"
    SessionFinished  SessionStatus = "FINISHED"
    SessionAbandoned SessionStatus = "ABANDONED"
)

type StudySession struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Status      SessionStatus
    StartedAt   time.Time
    FinishedAt  *time.Time
    Result      *SessionResult     // nil для ACTIVE/ABANDONED
    CreatedAt   time.Time
}

type SessionResult struct {
    TotalReviewed int
    NewReviewed   int
    DueReviewed   int
    GradeCounts   GradeCounts
    DurationMs    int64              // FinishedAt - StartedAt
    AccuracyRate  float64            // % GOOD+EASY
}

type GradeCounts struct {
    Again int
    Hard  int
    Good  int
    Easy  int
}

// --- Dashboard ---

type Dashboard struct {
    DueCount       int
    NewCount       int
    ReviewedToday  int
    NewToday       int
    Streak         int
    StatusCounts   CardStatusCounts
    OverdueCount   int              // Карточки просроченные более чем на 1 день
    ActiveSession  *uuid.UUID       // ID активной сессии, если есть
}

type CardStats struct {
    TotalReviews  int
    AccuracyRate  float64   // % GOOD+EASY из всех reviews
    AverageTimeMs *int
    CurrentStatus LearningStatus
    IntervalDays  int
    EaseFactor    float64
}
```

### 4.1. Необходимая миграция: learning_step в cards

Поле `learning_step` отсутствует в текущем DDL (data_model_v4.md). Требуется миграция:

```sql
-- +goose Up
ALTER TABLE cards ADD COLUMN learning_step INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE cards DROP COLUMN learning_step;
```

### 4.2. Необходимая миграция: prev_state в review_logs

```sql
-- +goose Up
ALTER TABLE review_logs
    ADD COLUMN prev_status       learning_status,
    ADD COLUMN prev_interval_days INT,
    ADD COLUMN prev_ease_factor   FLOAT,
    ADD COLUMN prev_learning_step INT,
    ADD COLUMN prev_next_review_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE review_logs
    DROP COLUMN prev_status,
    DROP COLUMN prev_interval_days,
    DROP COLUMN prev_ease_factor,
    DROP COLUMN prev_learning_step,
    DROP COLUMN prev_next_review_at;
```

### 4.3. Необходимая миграция: study_sessions

```sql
-- +goose Up
CREATE TYPE session_status AS ENUM ('ACTIVE', 'FINISHED', 'ABANDONED');

CREATE TABLE study_sessions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      session_status NOT NULL DEFAULT 'ACTIVE',
    started_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    result      JSONB,             -- SessionResult, null для ACTIVE/ABANDONED
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_study_sessions_user ON study_sessions(user_id, started_at DESC);
-- Не более одной ACTIVE сессии на пользователя
CREATE UNIQUE INDEX ux_study_sessions_active ON study_sessions(user_id) WHERE status = 'ACTIVE';

-- +goose Down
DROP TABLE study_sessions;
DROP TYPE session_status;
```

---

## 5. SRS Algorithm (Spaced Repetition System)

### 5.1. Обзор

Алгоритм основан на SM-2 (SuperMemo 2) с модификациями из Anki. Карточка проходит через фазы: **NEW → LEARNING → REVIEW ↔ MASTERED**. При ошибке в REVIEW/MASTERED карточка возвращается в **RELEARNING** (подмножество LEARNING).

### 5.2. Диаграмма переходов состояний

```
                    ┌─────────────────────────┐
                    │          NEW             │
                    │  (ожидает первый review)  │
                    └────────────┬─────────────┘
                                 │ Любой grade
                                 ▼
                    ┌─────────────────────────┐
            ┌──────│       LEARNING            │◄─────────┐
            │      │  (learning steps: 1m, 10m)│          │
            │      └────────────┬──────────────┘          │
            │                   │                          │
            │      Все шаги пройдены                       │
            │      (GOOD/EASY на последнем шаге)          │
            │                   │                          │
            │                   ▼                          │
            │      ┌─────────────────────────┐            │
            │      │        REVIEW            │            │
            │      │  (интервальные повторения)│            │
            │      └──┬──────────┬────────────┘            │
            │         │          │                          │
            │    GOOD/HARD    AGAIN                        │
            │    EASY         (lapse)                      │
            │         │          │                          │
            │         ▼          ▼                          │
            │    ┌─────────┐  ┌──────────────┐            │
            │    │ REVIEW  │  │  LEARNING     │────────────┘
            │    │(new int)│  │ (relearning)  │
            │    └────┬────┘  └──────────────┘
            │         │
            │    interval ≥ 21 AND ease ≥ 2.5
            │         │
            │         ▼
            │    ┌─────────────────────────┐
            │    │       MASTERED           │
            │    └──┬──────────┬────────────┘
            │       │          │
            │  GOOD/HARD    AGAIN
            │  EASY         (lapse)
            │       │          │
            │       ▼          │
            │    REVIEW ◄──────┘ (через LEARNING/relearning)
            │
            │  AGAIN на любом шаге LEARNING
            └──────► Сброс на шаг 0
```

### 5.3. Фазы и правила

#### NEW → LEARNING (первый review)

Когда карточка впервые появляется в очереди и пользователь оценивает её:

| Grade | Действие |
|-------|----------|
| AGAIN | Остаётся в LEARNING, шаг 0, next_review = now + learning_steps[0] |
| HARD  | Остаётся в LEARNING, шаг 0, next_review = now + avg(learning_steps[0], learning_steps[1]) (если есть) |
| GOOD  | Переход на шаг 1 (или graduate если шагов ≤ 1), next_review = now + learning_steps[1] |
| EASY  | Немедленный graduate → REVIEW, interval = easy_interval (4 дня) |

#### LEARNING (learning steps)

Карточка проходит через learning_steps (по умолчанию: 1 минута, 10 минут).

| Grade | Действие |
|-------|----------|
| AGAIN | Сброс на шаг 0, next_review = now + learning_steps[0] |
| HARD  | Остаётся на текущем шаге, next_review = now + текущий step (повтор) |
| GOOD  | Переход на следующий шаг. Если это был последний шаг → **graduate** |
| EASY  | Немедленный graduate → REVIEW, interval = easy_interval |

**Graduation** (переход LEARNING → REVIEW):
- Status = REVIEW
- interval = graduating_interval (1 день)
- ease_factor = default_ease (2.5)
- next_review = now + interval

#### REVIEW (интервальные повторения)

Основная фаза. Интервалы растут по формуле SM-2.

| Grade | EaseFactor change | New interval | Действие |
|-------|-------------------|--------------|----------|
| AGAIN | ease − 0.20 (min: 1.3) | max(1, old_interval × lapse_new_interval) | → LEARNING (relearning) |
| HARD  | ease − 0.15 (min: 1.3) | old_interval × hard_interval_modifier (1.2) | Остаётся REVIEW, min interval = old + 1 день |
| GOOD  | ease (без изменений) | old_interval × ease_factor | Остаётся REVIEW |
| EASY  | ease + 0.15 | old_interval × ease_factor × easy_bonus (1.3) | Остаётся REVIEW |

**Формула нового интервала для GOOD:**
```
new_interval = old_interval × ease_factor × interval_modifier
```

**Ограничения:**
- Минимальный interval после review: `old_interval + 1` (для GOOD/EASY — гарантирует рост)
- Максимальный interval: `min(max_interval_days, user_settings.max_interval_days)`
- Минимальный ease_factor: 1.3

#### Relearning (AGAIN в REVIEW/MASTERED → LEARNING)

Когда карточка «забыта» (lapse):
- Status = LEARNING (relearning)
- learning_step = 0
- Проходит relearning_steps (по умолчанию: [10m])
- После прохождения relearning → возврат в REVIEW с новым интервалом
- Новый интервал после relearning: `max(1, old_interval × lapse_new_interval)`
- ease_factor уменьшается на 0.20

#### REVIEW → MASTERED

Карточка получает статус MASTERED когда **одновременно**:
- interval_days ≥ 21
- ease_factor ≥ 2.5

MASTERED — это метка, а не отдельная фаза. Карточка продолжает повторяться по расписанию. При AGAIN → откатывается в LEARNING (relearning) → ease и interval сбрасываются → теряет статус MASTERED.

### 5.4. Реализация — чистая функция

```go
// service/study/srs.go

// SRSInput — входные данные для расчёта нового состояния карточки.
type SRSInput struct {
    CurrentStatus   domain.LearningStatus
    CurrentInterval int              // в днях
    CurrentEase     float64
    LearningStep    int              // текущий шаг в learning/relearning
    Grade           domain.ReviewGrade
    Now             time.Time
    Config          domain.SRSConfig
    MaxIntervalDays int              // из user_settings, может быть < config
}

// SRSOutput — результат расчёта.
type SRSOutput struct {
    NewStatus       domain.LearningStatus
    NewInterval     int
    NewEase         float64
    NewLearningStep int
    NextReviewAt    time.Time
}

// CalculateSRS — чистая функция. Не зависит от БД, контекста, логгера.
// Все решения детерминированы входными параметрами.
func CalculateSRS(input SRSInput) SRSOutput {
    switch input.CurrentStatus {
    case domain.StatusNew:
        return calculateNew(input)
    case domain.StatusLearning:
        return calculateLearning(input)
    case domain.StatusReview, domain.StatusMastered:
        return calculateReview(input)
    default:
        // Defensive: трактуем unknown status как NEW
        return calculateNew(input)
    }
}
```

Каждая ветка (`calculateNew`, `calculateLearning`, `calculateReview`) — приватная функция в том же файле.

### 5.5. Fuzz factor

Для предотвращения скопления карточек на одну дату, к интервалу добавляется детерминированный fuzz:

```
fuzz_range = max(1, interval * 0.05)  // ±5% от интервала
fuzz_days  = deterministicFuzz(card_id, review_count) % (fuzz_range * 2 + 1) - fuzz_range
final_interval = interval + fuzz_days
```

Fuzz применяется **только** к интервалам ≥ 3 дней (для коротких интервалов смысла нет).

Fuzz детерминирован (основан на card_id + review_count), чтобы повторные вычисления давали тот же результат. Это важно для тестируемости и идемпотентности.

### 5.6. Timezone-aware scheduling

`next_review_at` всегда хранится в UTC, но расчёт «сегодня» и daily limits привязан к timezone пользователя.

```go
// service/study/timezone.go

func DayStart(now time.Time, tz *time.Location) time.Time {
    userNow := now.In(tz)
    dayStart := time.Date(userNow.Year(), userNow.Month(), userNow.Day(), 0, 0, 0, 0, tz)
    return dayStart.UTC()
}

func NextDayStart(now time.Time, tz *time.Location) time.Time {
    return DayStart(now, tz).Add(24 * time.Hour)
}
```

---

## 6. Операции

### 6.1. GetStudyQueue (S1)

**Описание:** Формирует очередь карточек для study session. Overdue first, затем new. **Overdue reviews не ограничены `reviews_per_day`** — лимит применяется только к new cards.

**Input:**
```go
type GetQueueInput struct {
    Limit int  // max карточек в ответе, default 50, max 200
}

func (i GetQueueInput) Validate() error {
    var errs []domain.FieldError
    if i.Limit < 0 {
        errs = append(errs, domain.FieldError{Field: "limit", Message: "must be non-negative"})
    }
    if i.Limit > 200 {
        errs = append(errs, domain.FieldError{Field: "limit", Message: "max 200"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `input.Validate()` → если ошибка → `ValidationError`. Если limit = 0 → default 50.
3. Загрузить `settings` через `settingsRepo.GetByUserID(ctx, userID)`
4. Вычислить `dayStart` по `settings.Timezone`
5. Подсчитать `newToday` = `reviewLogRepo.CountNewToday(ctx, userID, dayStart)`
6. Рассчитать лимит на new cards: `newRemaining = max(0, settings.NewCardsPerDay - newToday)`
7. Загрузить due cards (**без лимита по reviews_per_day** — overdue не ограничены):
   - `dueCards = cardRepo.GetDueCards(ctx, userID, now, limit)`
8. Если `len(dueCards) < limit` и `newRemaining > 0`:
   - `newLimit = min(limit - len(dueCards), newRemaining)`
   - `newCards = cardRepo.GetNewCards(ctx, userID, newLimit)`
9. Объединить: `queue = append(dueCards, newCards...)`
10. Логировать INFO: `user_id`, `due_count`, `new_count`, `total`
11. Вернуть `queue`

**Порядок внутри очереди:**
- LEARNING (relearning) карточки — первые (intraday reviews)
- REVIEW/MASTERED overdue — по `next_review_at ASC` (самые просроченные первые)
- NEW — в порядке создания (FIFO)

**Daily limits — вариант B (overdue без лимита):**

| Что ограничивается | Лимит | Обоснование |
|-------------------|-------|-------------|
| New cards в день | `settings.new_cards_per_day` | Контроль нагрузки — не добавлять слишком много нового |
| Overdue reviews | **Без лимита** | Позволяет пользователю быстро разгрести backlog после перерыва |

Обоснование: `reviews_per_day` теперь **рекомендательный** параметр для dashboard (показать «рекомендуемая дневная нагрузка: ~200»), но **не блокирует** overdue reviews. Пользователь, пропустивший 2 недели, должен иметь возможность вернуться в режим за 1–2 дня, а не за неделю.

**Ответ включает `next_review_at` для каждой карточки** — клиент использует это для:
- Таймеров обратного отсчёта learning-карточек («вернётся через 8 мин»)
- Индикации просроченности («просрочена 3 дня»)

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Нет карточек вообще | Вернуть пустой slice, лог WARN |
| Все new лимиты исчерпаны, нет overdue | Вернуть пустой slice |
| Timezone невалидный | Fallback на UTC, лог WARN |
| limit = 0 | Применить default (50) |
| Soft-deleted entries | cardRepo исключает через JOIN |
| 500 overdue после 2-недельного перерыва | Все доступны (ограничены только `limit` per request) |
| Learning-карточка с next_review через 5 мин | Не попадает в текущий запрос (next_review_at > now), клиент делает refetch через 5 мин |

---

### 6.2. ReviewCard (S2)

**Описание:** Пользователь оценивает карточку. SRS пересчитывает параметры. Создаётся review_log **со snapshot предыдущего состояния** (для undo). **Самая критичная операция сервиса.**

**Input:**
```go
type ReviewCardInput struct {
    CardID     uuid.UUID
    Grade      domain.ReviewGrade
    DurationMs *int          // время на ответ (опционально, client-side)
    SessionID  *uuid.UUID    // ID активной study session (опционально)
}

func (i ReviewCardInput) Validate() error {
    var errs []domain.FieldError
    if i.CardID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "card_id", Message: "required"})
    }
    if !i.Grade.IsValid() {
        errs = append(errs, domain.FieldError{Field: "grade", Message: "must be AGAIN, HARD, GOOD, or EASY"})
    }
    if i.DurationMs != nil && *i.DurationMs < 0 {
        errs = append(errs, domain.FieldError{Field: "duration_ms", Message: "must be non-negative"})
    }
    if i.DurationMs != nil && *i.DurationMs > 600_000 {
        errs = append(errs, domain.FieldError{Field: "duration_ms", Message: "max 10 minutes"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `input.Validate()` → если ошибка → `ValidationError`
3. Загрузить карточку: `card := cardRepo.GetByID(ctx, userID, input.CardID)` → если нет → `ErrNotFound`
4. Загрузить настройки: `settings := settingsRepo.GetByUserID(ctx, userID)`
5. Определить maxInterval: `min(srsConfig.MaxIntervalDays, settings.MaxIntervalDays)`
6. **Сохранить snapshot** текущего состояния для undo
7. Вызвать SRS: `result := CalculateSRS(SRSInput{...})`
8. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Обновить карточку: `cardRepo.UpdateSRS(ctx, userID, card.ID, SRSUpdateParams{...})`
   b. Создать review log **с prev_state**: `reviewLogRepo.Create(ctx, &domain.ReviewLog{..., PrevStatus: card.Status, PrevIntervalDays: card.IntervalDays, PrevEaseFactor: card.EaseFactor, PrevLearningStep: card.LearningStep, PrevNextReviewAt: card.NextReviewAt})`
   c. Записать audit: `audit.Log(ctx, AuditRecord{EntityType: CARD, Action: UPDATE, ...})`
9. Логировать INFO: `user_id`, `card_id`, `grade`, `old_status`, `new_status`, `new_interval`
10. Вернуть обновлённую карточку

**Audit changes format:**
```json
{
    "grade": {"new": "GOOD"},
    "status": {"old": "LEARNING", "new": "REVIEW"},
    "interval_days": {"old": 0, "new": 1},
    "ease_factor": {"old": 2.5, "new": 2.5},
    "next_review_at": {"new": "2026-02-14T10:00:00Z"}
}
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Карточка не найдена | `ErrNotFound` |
| Карточка чужого пользователя | `ErrNotFound` (не раскрываем существование) |
| Entry soft-deleted | cardRepo.GetByID проверяет через JOIN → `ErrNotFound` |
| Повторный review той же карточки за секунду | Допустимо — каждый review создаёт отдельный log. Last-write-wins для SRS-состояния (см. секцию 15.1) |
| DurationMs > 10 минут | Validation error |
| Grade невалидный | Validation error |
| Ошибка при создании review_log | Транзакция откатывает и update карточки |
| SessionID указан, но сессия не ACTIVE | Игнорировать sessionID, review выполняется (сессия — soft tracking) |

---

### 6.3. UndoReview (S10) — NEW

**Описание:** Отмена последнего review карточки. Восстанавливает SRS-состояние из snapshot в ReviewLog. Критичный сценарий: пользователь случайно нажал AGAIN на карточку с interval=60 дней.

**Input:**
```go
type UndoReviewInput struct {
    CardID uuid.UUID
}

func (i UndoReviewInput) Validate() error {
    if i.CardID == uuid.Nil {
        return domain.NewValidationError("card_id", "required")
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Загрузить карточку: `card := cardRepo.GetByID(ctx, userID, input.CardID)` → `ErrNotFound`
4. Загрузить последний review log: `lastLog := reviewLogRepo.GetLastByCardID(ctx, input.CardID)`
   - Если нет логов → `ValidationError("card has no reviews to undo")`
5. Проверить, что undo возможен:
   - `lastLog.PrevStatus` не nil (лог содержит snapshot) → если nil → `ValidationError("review cannot be undone")`
   - `time.Since(lastLog.ReviewedAt) < 10 * time.Minute` → если нет → `ValidationError("undo window expired (10 minutes)")`
6. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Восстановить карточку: `cardRepo.UpdateSRS(ctx, userID, card.ID, SRSUpdateParams{Status: lastLog.PrevStatus, IntervalDays: lastLog.PrevIntervalDays, EaseFactor: lastLog.PrevEaseFactor, LearningStep: lastLog.PrevLearningStep, NextReviewAt: lastLog.PrevNextReviewAt})`
   b. Удалить review log: `reviewLogRepo.Delete(ctx, lastLog.ID)`
   c. Audit: `audit.Log(ctx, AuditRecord{EntityType: CARD, Action: UPDATE, Changes: {"undo": {"old": lastLog.Grade}}})`
7. Логировать INFO: `user_id`, `card_id`, `undone_grade`, `restored_status`
8. Вернуть восстановленную карточку

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Нет review logs | `ValidationError("no reviews to undo")` |
| Последний лог без prev_state (legacy) | `ValidationError("review cannot be undone")` |
| Прошло > 10 минут | `ValidationError("undo window expired")` |
| Двойной undo | После первого undo последний лог меняется — второй undo отменит предпоследний review (это корректно) |
| Undo в другой сессии | Допустимо — undo привязан к карточке, не к сессии |

**Ограничение: только один уровень undo.** Для MVP отмена отменяет только последний review. Цепочка undo (undo-undo-undo) технически работает (каждый раз берётся последний лог), но это edge case, и временное окно 10 минут естественно его ограничивает.

---

### 6.4. GetDashboard (S3)

**Описание:** Агрегированная статистика для dashboard.

**Input:** Нет (только userID из context).

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. Загрузить settings → вычислить `dayStart`
3. Параллельно (или последовательно, без транзакции):
   - `dueCount = cardRepo.CountDue(ctx, userID, now)`
   - `newCount = cardRepo.CountNew(ctx, userID)`
   - `reviewedToday = reviewLogRepo.CountToday(ctx, userID, dayStart)`
   - `newToday = reviewLogRepo.CountNewToday(ctx, userID, dayStart)`
   - `statusCounts = cardRepo.CountByStatus(ctx, userID)`
   - `streakDays = reviewLogRepo.GetStreakDays(ctx, userID, dayStart, 365)`
   - `activeSession = sessionRepo.GetActive(ctx, userID)` (может быть nil)
4. Вычислить streak из `streakDays`
5. Вычислить `overdueCount` — карточки с `next_review_at < dayStart` (просрочены хотя бы день)
6. Вернуть `Dashboard{..., OverdueCount: overdueCount, ActiveSession: activeSession?.ID}`

**Расчёт streak:**

Streak — количество **последовательных дней** (до сегодня), в которые пользователь сделал хотя бы один review. Сегодняшний день считается, если пользователь уже сделал review. Если сегодня reviews нет — проверяем начиная со вчера.

```go
func calculateStreak(days []domain.DayReviewCount, today time.Time) int {
    if len(days) == 0 {
        return 0
    }

    streak := 0
    expectedDate := today
    if len(days) > 0 && !days[0].Date.Equal(today) {
        expectedDate = today.AddDate(0, 0, -1)
    }

    for _, d := range days {
        if d.Date.Equal(expectedDate) {
            streak++
            expectedDate = expectedDate.AddDate(0, 0, -1)
        } else if d.Date.Before(expectedDate) {
            break
        }
    }
    return streak
}
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Нет карточек | Все счётчики = 0, streak = 0 |
| Нет reviews сегодня | reviewedToday = 0, streak считается от вчера |
| Timezone "America/New_York" и UTC midnight | dayStart корректно вычисляется как 05:00 UTC |
| 500 overdue после перерыва | overdueCount = 500, dashboard показывает «500 просроченных карточек» |

---

### 6.5. StartSession (S9) — NEW

**Описание:** Начало study session. Создаёт запись с ACTIVE статусом.

**Input:** Нет (только userID из context).

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. Проверить, нет ли уже ACTIVE сессии: `existing := sessionRepo.GetActive(ctx, userID)`
   - Если есть → вернуть существующую (идемпотентность, не ошибка)
3. Создать сессию: `session := sessionRepo.Create(ctx, &domain.StudySession{UserID: userID, Status: ACTIVE, StartedAt: now})`
4. Логировать INFO: `user_id`, `session_id`
5. Вернуть `session`

**Corner case:** Unique constraint `ux_study_sessions_active` гарантирует не более одной ACTIVE на пользователя. При race condition — один из запросов получит `ErrAlreadyExists`, сервис выполнит `GetActive` и вернёт существующую.

---

### 6.6. FinishSession (S9) — NEW

**Описание:** Завершение study session. Агрегирует результаты из review_logs за время сессии.

**Input:**
```go
type FinishSessionInput struct {
    SessionID uuid.UUID
}

func (i FinishSessionInput) Validate() error {
    if i.SessionID == uuid.Nil {
        return domain.NewValidationError("session_id", "required")
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Загрузить сессию: `session := sessionRepo.GetByID(ctx, userID, input.SessionID)` → `ErrNotFound`
4. Проверить статус: `session.Status == ACTIVE` → если нет → `ValidationError("session already finished")`
5. Агрегировать результаты из review_logs за период `[session.StartedAt, now]`:
   - `totalReviewed` — общее количество reviews
   - `newReviewed` — reviews карточек, которые были NEW на момент review (определяется по `prev_status = NEW` в review_log)
   - `dueReviewed = totalReviewed - newReviewed`
   - `gradeCounts` — количество по каждому grade
   - `durationMs = now - session.StartedAt`
   - `accuracyRate = (gradeCounts.Good + gradeCounts.Easy) / totalReviewed * 100`
6. Завершить сессию: `sessionRepo.Finish(ctx, userID, session.ID, SessionResult{...})`
7. Логировать INFO: `user_id`, `session_id`, `total_reviewed`, `accuracy`, `duration`
8. Вернуть обновлённую сессию

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Сессия не найдена | `ErrNotFound` |
| Сессия уже FINISHED | `ValidationError("session already finished")` |
| Сессия уже ABANDONED | `ValidationError("session already finished")` |
| Нет reviews за время сессии | totalReviewed = 0, accuracy = 0, сессия завершается (пустая сессия допустима) |
| Сессия длилась > 24 часов | Завершается нормально. Клиент может показать предупреждение |

---

### 6.7. AbandonSession — NEW

**Описание:** Отмена незавершённой сессии (пользователь закрыл приложение). Не удаляет review_logs.

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. Загрузить активную сессию: `session := sessionRepo.GetActive(ctx, userID)` → если нет → noop (нечего abandon)
3. `sessionRepo.Abandon(ctx, userID, session.ID)`
4. Логировать INFO

**Corner case:** Клиент может вызвать AbandonSession при открытии приложения, чтобы подчистить брошенные сессии. Операция идемпотентна — если нет ACTIVE сессии, ничего не происходит.

---

### 6.8. CreateCard (S4)

**Описание:** Создание карточки для entry. **Entry должен иметь хотя бы один sense** — карточка без содержимого бесполезна.

**Input:**
```go
type CreateCardInput struct {
    EntryID uuid.UUID
}

func (i CreateCardInput) Validate() error {
    if i.EntryID == uuid.Nil {
        return domain.NewValidationError("entry_id", "required")
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Проверить, что entry существует: `entryRepo.GetByID(ctx, userID, input.EntryID)` → `ErrNotFound`
4. **Проверить, что entry имеет senses:** `senseCount := senseRepo.CountByEntryID(ctx, input.EntryID)` → если 0 → `ValidationError("entry_id", "entry must have at least one sense to create a card")`
5. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Создать карточку: `card := cardRepo.Create(ctx, userID, &domain.Card{EntryID: input.EntryID, Status: NEW, EaseFactor: srsConfig.DefaultEaseFactor})`
   b. Audit: `audit.Log(ctx, AuditRecord{EntityType: CARD, Action: CREATE, ...})`
6. Логировать INFO: `user_id`, `card_id`, `entry_id`
7. Вернуть `card`

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Entry не существует | `ErrNotFound` |
| Карточка для entry уже есть | `ErrAlreadyExists` (unique constraint ux_cards_entry) |
| Entry soft-deleted | entryRepo.GetByID фильтрует → `ErrNotFound` |
| Entry без senses | `ValidationError("entry must have at least one sense")` |

---

### 6.9. DeleteCard (S5)

**Описание:** Удаление карточки. Entry остаётся в словаре.

**Input:**
```go
type DeleteCardInput struct {
    CardID uuid.UUID
}

func (i DeleteCardInput) Validate() error {
    if i.CardID == uuid.Nil {
        return domain.NewValidationError("card_id", "required")
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Загрузить карточку: `card := cardRepo.GetByID(ctx, userID, input.CardID)` → `ErrNotFound`
4. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Удалить карточку: `cardRepo.Delete(ctx, userID, input.CardID)` (CASCADE удаляет review_logs)
   b. Audit: `audit.Log(ctx, AuditRecord{EntityType: CARD, Action: DELETE, Changes: {"entry_id": {"old": card.EntryID}}})`
5. Логировать INFO
6. Вернуть `nil`

---

### 6.10. GetCardHistory (S6)

**Описание:** Полная история reviews карточки.

**Input:**
```go
type GetCardHistoryInput struct {
    CardID uuid.UUID
    Limit  int   // default 50, max 200
    Offset int
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. Проверить, что карточка принадлежит пользователю: `cardRepo.GetByID(ctx, userID, input.CardID)` → `ErrNotFound`
3. `logs, total := reviewLogRepo.GetByCardID(ctx, input.CardID, limit, offset)`
4. Вернуть `logs, total`

---

### 6.11. GetCardStats (S7)

**Описание:** Статистика карточки: total reviews, accuracy rate, average duration.

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. Загрузить карточку: `card := cardRepo.GetByID(ctx, userID, input.CardID)` → `ErrNotFound`
3. Загрузить все review logs: `logs, total := reviewLogRepo.GetByCardID(ctx, input.CardID, 0, 0)`
4. Вычислить stats:
   - `totalReviews = total`
   - `accuracyRate = count(GOOD + EASY) / total * 100`
   - `averageTimeMs = avg(duration_ms)` (только non-nil)
5. Вернуть `CardStats{...}`

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Карточка без reviews | totalReviews = 0, accuracyRate = 0, averageTimeMs = nil |
| Все reviews без duration | averageTimeMs = nil |

---

### 6.12. BatchCreateCards (S8)

**Описание:** Массовое создание карточек. **Entries без senses пропускаются.**

**Input:**
```go
type BatchCreateCardsInput struct {
    EntryIDs []uuid.UUID
}

func (i BatchCreateCardsInput) Validate() error {
    var errs []domain.FieldError
    if len(i.EntryIDs) == 0 {
        errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "at least one entry required"})
    }
    if len(i.EntryIDs) > 100 {
        errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "max 100 entries per batch"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Проверить существование entries: `existingEntries := entryRepo.ExistByIDs(ctx, userID, input.EntryIDs)`
4. Отфильтровать несуществующие → `validEntryIDs`
5. **Проверить, какие entries имеют senses:** для каждого `senseRepo.CountByEntryID` → отфильтровать entries без senses → `entriesWithContent`
6. Проверить, какие entries уже имеют карточки: `existingCards := cardRepo.ExistsByEntryIDs(ctx, userID, entriesWithContent)`
7. Отфильтровать entries с карточками → `toCreate`
8. **Транзакция** для каждого batch по 50:
   - Создать карточки
   - Audit для каждой
9. Вернуть результат: `{created: N, skippedExisting: M, skippedNoSenses: K, errors: [...]}`

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Все entries уже имеют карточки | created = 0, skippedExisting = len |
| Некоторые entries без senses | Пропускаются, skippedNoSenses++ |
| Некоторые entryIDs не существуют | Пропускаются |
| Пустой массив | Validation error |
| > 100 entries | Validation error |

---

## 7. Data Requirements для отображения карточки

Study Service возвращает `Card` с `EntryID` и `EntryText`. Для **полного отображения карточки** в study mode клиенту нужны дополнительные данные, загружаемые через GraphQL DataLoaders:

| Данные | DataLoader | Сторона карточки |
|--------|-----------|------------------|
| Entry text, notes | Card.EntryText (денормализовано) | Front (вопрос) |
| Senses (definition, POS) | SensesByEntryID | Back (ответ) |
| Translations | TranslationsBySenseID | Back (ответ) |
| Examples (sentence + translation) | ExamplesBySenseID | Back (ответ) |
| Pronunciations (transcription, audio) | PronunciationsByEntryID | Back (ответ) / Front (подсказка) |
| Images | CatalogImagesByEntryID + UserImagesByEntryID | Back (ответ) |

**Типичный flow отображения:**

1. Клиент показывает **front**: entry text (слово)
2. Пользователь мысленно вспоминает значение
3. Нажимает «показать ответ»
4. Клиент показывает **back**: definition, translations, examples, images, audio
5. Пользователь оценивает: AGAIN / HARD / GOOD / EASY

GraphQL query для study session оптимизируется DataLoaders — все карточки в очереди загружают senses/translations/etc за 1 batch query на тип данных, а не N+1.

---

## 8. Soft Delete и Restore — поведение для карточек

### 8.1. Soft delete entry

Когда entry получает `deleted_at` (soft delete):
- Карточка **не удаляется** физически (CASCADE не срабатывает на soft delete)
- Карточка **исключается из очереди** — `GetDueCards` и `GetNewCards` фильтруют через JOIN `entries.deleted_at IS NULL`
- SRS-состояние карточки **сохраняется** как есть

### 8.2. Restore entry

Когда entry восстанавливается (`deleted_at = NULL`):
- Карточка **снова видна** в очереди с сохранённым прогрессом
- Если `next_review_at` уже прошёл — карточка считается overdue и попадёт в начало очереди
- Пользователь не теряет прогресс

### 8.3. Hard delete entry (30 дней)

Когда entry удаляется физически (hard delete job):
- `ON DELETE CASCADE` удаляет карточку и все review_logs
- Прогресс теряется необратимо

---

## 9. Error Scenarios (сводная таблица)

| Операция | Ошибка | Тип | HTTP-аналог |
|----------|--------|-----|-------------|
| Все | Нет userID в ctx | `ErrUnauthorized` | 401 |
| GetStudyQueue | Невалидный limit | `ValidationError` | 400 |
| GetStudyQueue | Невалидный timezone в settings | Fallback UTC, WARN лог | — |
| ReviewCard | card_id = nil | `ValidationError` | 400 |
| ReviewCard | Невалидный grade | `ValidationError` | 400 |
| ReviewCard | duration_ms < 0 или > 600000 | `ValidationError` | 400 |
| ReviewCard | Карточка не найдена | `ErrNotFound` | 404 |
| ReviewCard | Карточка чужого пользователя | `ErrNotFound` | 404 |
| ReviewCard | Entry soft-deleted | `ErrNotFound` | 404 |
| ReviewCard | Ошибка записи review_log | Tx rollback, internal error | 500 |
| UndoReview | card_id = nil | `ValidationError` | 400 |
| UndoReview | Карточка не найдена | `ErrNotFound` | 404 |
| UndoReview | Нет review logs | `ValidationError` | 400 |
| UndoReview | Нет prev_state (legacy log) | `ValidationError` | 400 |
| UndoReview | Прошло > 10 минут | `ValidationError` | 400 |
| CreateCard | entry_id = nil | `ValidationError` | 400 |
| CreateCard | Entry не найден | `ErrNotFound` | 404 |
| CreateCard | Entry без senses | `ValidationError` | 400 |
| CreateCard | Карточка уже существует | `ErrAlreadyExists` | 409 |
| CreateCard | Entry soft-deleted | `ErrNotFound` | 404 |
| DeleteCard | card_id = nil | `ValidationError` | 400 |
| DeleteCard | Карточка не найдена | `ErrNotFound` | 404 |
| GetCardHistory | Карточка не найдена | `ErrNotFound` | 404 |
| GetCardHistory | Чужая карточка | `ErrNotFound` | 404 |
| GetCardStats | Карточка не найдена | `ErrNotFound` | 404 |
| BatchCreateCards | Пустой массив | `ValidationError` | 400 |
| BatchCreateCards | > 100 entries | `ValidationError` | 400 |
| GetDashboard | Нет settings (не должно быть) | Internal error | 500 |
| StartSession | — | Всегда OK (идемпотентна) | — |
| FinishSession | Сессия не найдена | `ErrNotFound` | 404 |
| FinishSession | Сессия не ACTIVE | `ValidationError` | 400 |

---

## 10. Валидация (сводная)

| Input | Поле | Правило |
|-------|------|---------|
| GetQueueInput | limit | ≥ 0, ≤ 200. Default: 50 |
| ReviewCardInput | card_id | required, не Nil UUID |
| ReviewCardInput | grade | must be AGAIN/HARD/GOOD/EASY |
| ReviewCardInput | duration_ms | if present: ≥ 0, ≤ 600000 (10 min) |
| ReviewCardInput | session_id | optional, если указан — не Nil UUID |
| UndoReviewInput | card_id | required, не Nil UUID |
| CreateCardInput | entry_id | required, не Nil UUID |
| DeleteCardInput | card_id | required, не Nil UUID |
| GetCardHistoryInput | card_id | required, не Nil UUID |
| GetCardHistoryInput | limit | ≥ 1, ≤ 200. Default: 50 |
| GetCardHistoryInput | offset | ≥ 0 |
| BatchCreateCardsInput | entry_ids | ≥ 1 элемент, ≤ 100 |
| FinishSessionInput | session_id | required, не Nil UUID |

---

## 11. Аудит

| Операция | EntityType | Action | Changes |
|----------|------------|--------|---------|
| CreateCard | CARD | CREATE | `{"entry_id": {"new": "..."}}` |
| ReviewCard | CARD | UPDATE | `{"grade": {"new": "GOOD"}, "status": {"old": "NEW", "new": "LEARNING"}, "interval_days": {"old": 0, "new": 0}, "ease_factor": {"old": 2.5, "new": 2.5}}` |
| UndoReview | CARD | UPDATE | `{"undo": {"old": "GOOD"}, "status": {"old": "REVIEW", "new": "LEARNING"}}` |
| DeleteCard | CARD | DELETE | `{"entry_id": {"old": "..."}}` |
| BatchCreateCards | CARD | CREATE | Отдельный audit record для каждой созданной карточки |

Study sessions не аудитируются — они несут аналитическую, а не бизнес-ценность.

---

## 12. Тестирование

### 12.1. SRS Algorithm — Table-Driven Tests (srs_test.go)

**Минимум 35 тестовых кейсов.** Чистая функция `CalculateSRS` тестируется изолированно.

| # | Категория | Input (Status/Step/Interval/Ease/Grade) | Expected Output (Status/Step/Interval/Ease) | Описание |
|---|-----------|---------------------------------------|---------------------------------------------|----------|
| 1 | NEW→LEARNING | NEW/0/0/2.5/AGAIN | LEARNING/0/0/2.5, next=+1m | Первый review, забыл |
| 2 | NEW→LEARNING | NEW/0/0/2.5/HARD | LEARNING/0/0/2.5, next=+5.5m | Первый review, тяжело (avg шагов) |
| 3 | NEW→LEARNING | NEW/0/0/2.5/GOOD | LEARNING/1/0/2.5, next=+10m | Первый review, следующий шаг |
| 4 | NEW→REVIEW | NEW/0/0/2.5/EASY | REVIEW/0/4/2.5, next=+4d | Первый review, сразу graduate |
| 5 | LEARNING step 0 | LEARNING/0/0/2.5/AGAIN | LEARNING/0/0/2.5, next=+1m | Сброс на шаг 0 |
| 6 | LEARNING step 0 | LEARNING/0/0/2.5/HARD | LEARNING/0/0/2.5, next=+1m | Повтор текущего шага |
| 7 | LEARNING step 0 | LEARNING/0/0/2.5/GOOD | LEARNING/1/0/2.5, next=+10m | Переход на шаг 1 |
| 8 | LEARNING step 0 | LEARNING/0/0/2.5/EASY | REVIEW/0/4/2.5, next=+4d | Немедленный graduate |
| 9 | LEARNING step 1 (last) | LEARNING/1/0/2.5/AGAIN | LEARNING/0/0/2.5, next=+1m | Сброс на шаг 0 |
| 10 | LEARNING step 1 (last) | LEARNING/1/0/2.5/HARD | LEARNING/1/0/2.5, next=+10m | Повтор последнего шага |
| 11 | LEARNING graduate | LEARNING/1/0/2.5/GOOD | REVIEW/0/1/2.5, next=+1d | Graduate → REVIEW |
| 12 | LEARNING graduate EASY | LEARNING/1/0/2.5/EASY | REVIEW/0/4/2.5, next=+4d | Graduate → REVIEW с easy interval |
| 13 | REVIEW | REVIEW/0/1/2.5/AGAIN | LEARNING/0/1/2.3, next=+10m | Lapse → relearning |
| 14 | REVIEW | REVIEW/0/1/2.5/HARD | REVIEW/0/2/2.35, next=+2d | Interval × 1.2, ease −0.15 |
| 15 | REVIEW | REVIEW/0/1/2.5/GOOD | REVIEW/0/3/2.5, next=+3d | Interval × ease |
| 16 | REVIEW | REVIEW/0/1/2.5/EASY | REVIEW/0/4/2.65, next=+4d | Interval × ease × 1.3, ease +0.15 |
| 17 | REVIEW longer | REVIEW/0/10/2.5/GOOD | REVIEW/0/25/2.5, next=+25d | 10 × 2.5 = 25 |
| 18 | REVIEW longer | REVIEW/0/10/2.5/HARD | REVIEW/0/12/2.35, next=+12d | 10 × 1.2 = 12, ease −0.15 |
| 19 | REVIEW → MASTERED | REVIEW/0/21/2.5/GOOD | MASTERED/0/53/2.5, next=+53d | interval ≥ 21, ease ≥ 2.5 |
| 20 | REVIEW not mastered | REVIEW/0/20/2.5/GOOD | REVIEW/0/50/2.5 | interval < 21 перед review |
| 21 | MASTERED GOOD | MASTERED/0/53/2.5/GOOD | MASTERED/0/133/2.5 | Остаётся MASTERED |
| 22 | MASTERED AGAIN | MASTERED/0/53/2.5/AGAIN | LEARNING/0/1/2.3 | Lapse, теряет MASTERED |
| 23 | Ease min boundary | REVIEW/0/5/1.3/AGAIN | LEARNING/0/1/1.3 | Ease не падает ниже 1.3 |
| 24 | Ease at min + HARD | REVIEW/0/5/1.3/HARD | REVIEW/0/6/1.3 | Ease уже на минимуме |
| 25 | Max interval cap | REVIEW/0/200/2.5/GOOD | REVIEW/0/365/2.5 | Ограничение max_interval |
| 26 | User max interval | REVIEW/0/200/2.5/GOOD (max=180) | REVIEW/0/180/2.5 | User setting перекрывает |
| 27 | REVIEW GOOD min growth | REVIEW/0/10/2.5/GOOD | REVIEW/0/25/2.5 | new ≥ old + 1 |
| 28 | Relearning graduate | LEARNING(relearn)/0/10/2.0/GOOD | REVIEW/0/1/2.0 | After relearn, interval=max(1, old×lapse) |
| 29 | Relearning AGAIN | LEARNING(relearn)/0/10/2.0/AGAIN | LEARNING/0/10/2.0, next=+10m | Сброс relearning |
| 30 | Lapse reset | REVIEW/0/30/2.5/AGAIN (lapse=0.0) | LEARNING/0/1/2.3 | lapse_new_interval=0 → interval=1 |
| 31 | Lapse 50% | REVIEW/0/30/2.5/AGAIN (lapse=0.5) | LEARNING/0/15/2.3 | 30 × 0.5 = 15 |
| 32 | Fuzz applied | REVIEW/0/10/2.5/GOOD | interval ~25 ±1 | Fuzz ≤ 5% |
| 33 | Fuzz not applied short | REVIEW/0/1/2.5/GOOD | interval = 3, exact | Fuzz не применяется при interval < 3 |
| 34 | Single learning step | config: steps=[10m], LEARNING/0/0/2.5/GOOD | REVIEW/0/1/2.5 | Graduate с одним шагом |
| 35 | Empty learning steps | config: steps=[], NEW/0/0/2.5/GOOD | REVIEW/0/1/2.5 | Немедленный graduate |

### 12.2. Service Unit Tests (service_test.go)

| # | Тест | Категория | Что проверяем |
|---|------|-----------|---------------|
| 1 | `TestGetStudyQueue_Success` | Happy path | Due cards + new cards в правильном порядке |
| 2 | `TestGetStudyQueue_NewLimitReached` | Limits | newCardsPerDay исчерпан → только due cards |
| 3 | `TestGetStudyQueue_OverdueNotLimited` | **Limits** | **500 overdue → все доступны (не ограничены reviews_per_day)** |
| 4 | `TestGetStudyQueue_EmptyQueue` | Edge case | Нет карточек → пустой slice |
| 5 | `TestGetStudyQueue_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 6 | `TestGetStudyQueue_InvalidLimit` | Validation | limit = -1 → ValidationError |
| 7 | `TestGetStudyQueue_DefaultLimit` | Defaults | limit = 0 → используется 50 |
| 8 | `TestGetStudyQueue_LearningCardsFirst` | Order | LEARNING карточки в начале очереди |
| 9 | `TestReviewCard_Success_NewToLearning` | Happy path | NEW + GOOD → LEARNING step 1 |
| 10 | `TestReviewCard_Success_GraduateToReview` | Happy path | LEARNING last step + GOOD → REVIEW |
| 11 | `TestReviewCard_Success_ReviewGood` | Happy path | REVIEW + GOOD → interval grows |
| 12 | `TestReviewCard_Success_ReviewAgain_Lapse` | Happy path | REVIEW + AGAIN → LEARNING (relearning) |
| 13 | `TestReviewCard_Success_ToMastered` | Happy path | REVIEW + GOOD, interval ≥ 21, ease ≥ 2.5 → MASTERED |
| 14 | `TestReviewCard_PrevStateStored` | **Undo prep** | **ReviewLog содержит корректный prev_state snapshot** |
| 15 | `TestReviewCard_NotFound` | Not found | Несуществующий card_id → ErrNotFound |
| 16 | `TestReviewCard_WrongUser` | Auth | Чужая карточка → ErrNotFound |
| 17 | `TestReviewCard_InvalidGrade` | Validation | Grade = "INVALID" → ValidationError |
| 18 | `TestReviewCard_NilCardID` | Validation | CardID = nil → ValidationError |
| 19 | `TestReviewCard_DurationTooLong` | Validation | DurationMs = 700000 → ValidationError |
| 20 | `TestReviewCard_Audit` | Audit | Audit record содержит корректные old/new значения |
| 21 | `TestReviewCard_Transaction_Rollback` | Transaction | Ошибка в reviewLogRepo.Create → tx rollback |
| 22 | `TestReviewCard_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 23 | `TestUndoReview_Success` | **Happy path** | **Карточка восстановлена, review_log удалён** |
| 24 | `TestUndoReview_RestoredState` | **Undo** | **Все SRS-поля восстановлены из prev_state** |
| 25 | `TestUndoReview_NoReviews` | **Validation** | **Нет логов → ValidationError** |
| 26 | `TestUndoReview_WindowExpired` | **Validation** | **Прошло > 10 минут → ValidationError** |
| 27 | `TestUndoReview_NoPrevState` | **Validation** | **Legacy лог без prev_state → ValidationError** |
| 28 | `TestUndoReview_Audit` | **Audit** | **Audit содержит undone grade** |
| 29 | `TestCreateCard_Success` | Happy path | Карточка создана, status = NEW |
| 30 | `TestCreateCard_EntryNotFound` | Not found | Entry не существует → ErrNotFound |
| 31 | `TestCreateCard_AlreadyExists` | Duplicate | Карточка уже есть → ErrAlreadyExists |
| 32 | `TestCreateCard_SoftDeletedEntry` | Soft delete | Entry soft-deleted → ErrNotFound |
| 33 | `TestCreateCard_EntryNoSenses` | **Validation** | **Entry без senses → ValidationError** |
| 34 | `TestCreateCard_Audit` | Audit | Audit record создан |
| 35 | `TestCreateCard_NilEntryID` | Validation | EntryID = nil → ValidationError |
| 36 | `TestDeleteCard_Success` | Happy path | Карточка удалена |
| 37 | `TestDeleteCard_NotFound` | Not found | Карточка не найдена → ErrNotFound |
| 38 | `TestDeleteCard_Audit` | Audit | Audit record с entry_id |
| 39 | `TestGetDashboard_Success` | Happy path | Все счётчики корректны |
| 40 | `TestGetDashboard_NoCards` | Edge case | Все счётчики = 0, streak = 0 |
| 41 | `TestGetDashboard_StreakCalculation` | Business | 5 дней подряд → streak = 5 |
| 42 | `TestGetDashboard_StreakBroken` | Business | Пропуск дня → streak обрывается |
| 43 | `TestGetDashboard_StreakTodayNotReviewed` | Business | Сегодня не было reviews → streak от вчера |
| 44 | `TestGetDashboard_OverdueCount` | **Dashboard** | **overdueCount корректно отражает backlog** |
| 45 | `TestGetDashboard_ActiveSession` | **Session** | **ActiveSession ID присутствует в dashboard** |
| 46 | `TestStartSession_Success` | **Happy path** | **Сессия создана, status = ACTIVE** |
| 47 | `TestStartSession_AlreadyActive` | **Idempotent** | **Возвращает существующую ACTIVE сессию** |
| 48 | `TestFinishSession_Success` | **Happy path** | **Сессия завершена, результаты агрегированы** |
| 49 | `TestFinishSession_AlreadyFinished` | **Validation** | **Сессия FINISHED → ValidationError** |
| 50 | `TestFinishSession_EmptySession` | **Edge case** | **Нет reviews → totalReviewed=0, сессия завершается** |
| 51 | `TestFinishSession_NotFound` | **Not found** | **Несуществующая сессия → ErrNotFound** |
| 52 | `TestAbandonSession_Success` | **Happy path** | **Сессия ABANDONED** |
| 53 | `TestAbandonSession_NoActive` | **Idempotent** | **Нет активной сессии → noop** |
| 54 | `TestGetCardHistory_Success` | Happy path | Список review logs по card_id |
| 55 | `TestGetCardHistory_CardNotFound` | Not found | ErrNotFound |
| 56 | `TestGetCardStats_Success` | Happy path | Accuracy rate = 75% |
| 57 | `TestGetCardStats_NoReviews` | Edge case | totalReviews=0, accuracy=0 |
| 58 | `TestBatchCreateCards_Success` | Happy path | 5 entries → 5 карточек |
| 59 | `TestBatchCreateCards_SomeExist` | Partial | 2 уже имеют карточки → 3 создано |
| 60 | `TestBatchCreateCards_SkipsNoSenses` | **Content** | **Entries без senses пропускаются** |
| 61 | `TestBatchCreateCards_AllExist` | Edge case | 0 создано |
| 62 | `TestBatchCreateCards_EmptyInput` | Validation | ValidationError |
| 63 | `TestBatchCreateCards_TooMany` | Validation | 101 entry → ValidationError |
| 64 | `TestSoftDeleteRestore_CardPreserved` | **Soft delete** | **После restore карточка с прогрессом снова в очереди** |

---

## 13. Конфигурация SRS

### 13.1. Глобальная конфигурация (из config)

| Параметр | Default | Описание |
|----------|---------|----------|
| `DefaultEaseFactor` | 2.5 | Начальный ease_factor для новых карточек |
| `MinEaseFactor` | 1.3 | Минимальное значение ease_factor |
| `MaxIntervalDays` | 365 | Глобальный максимум интервала (может быть overridden user settings) |
| `GraduatingInterval` | 1 | Интервал в днях при graduation из LEARNING |
| `EasyInterval` | 4 | Интервал в днях при EASY graduation |
| `LearningSteps` | [1m, 10m] | Шаги в фазе learning |
| `RelearningSteps` | [10m] | Шаги в фазе relearning (после lapse) |
| `IntervalModifier` | 1.0 | Глобальный множитель интервалов |
| `HardIntervalModifier` | 1.2 | Множитель для HARD в review |
| `EasyBonus` | 1.3 | Множитель для EASY в review |
| `LapseNewInterval` | 0.0 | Множитель интервала после lapse (0 = reset to minimum) |
| `UndoWindowMinutes` | 10 | Окно для undo review |

### 13.2. Per-user настройки (из user_settings)

| Параметр | Default | Описание |
|----------|---------|----------|
| `new_cards_per_day` | 20 | Лимит новых карточек в день |
| `reviews_per_day` | 200 | Рекомендательный лимит (не блокирует overdue) |
| `max_interval_days` | 365 | Максимальный интервал (берётся min с глобальным) |
| `timezone` | "UTC" | Timezone для расчёта «сегодня» |

---

## 14. Порядок карточек в очереди (детализация)

1. **Intraday reviews (LEARNING/relearning, next_review_at ≤ now)** — по `next_review_at ASC`.

2. **Overdue reviews (REVIEW/MASTERED, next_review_at ≤ now)** — по `next_review_at ASC` (самые просроченные первые).

3. **New cards (status = NEW, next_review_at IS NULL)** — по `created_at ASC` (FIFO).

```sql
-- Due cards (LEARNING + overdue REVIEW/MASTERED) — без лимита по reviews_per_day
SELECT c.*, e.text AS entry_text
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1
  AND e.deleted_at IS NULL
  AND c.status != 'NEW'
  AND c.next_review_at <= $2
ORDER BY
  CASE WHEN c.status = 'LEARNING' THEN 0 ELSE 1 END,
  c.next_review_at ASC
LIMIT $3;

-- New cards (ограничены new_cards_per_day)
SELECT c.*, e.text AS entry_text
FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1
  AND e.deleted_at IS NULL
  AND c.status = 'NEW'
ORDER BY c.created_at ASC
LIMIT $3;
```

---

## 15. Конкурентность и edge cases

### 15.1. Одновременные сессии с разных устройств

Пользователь открывает study session на телефоне и планшете. Оба загружают одну очередь. Оба reviewят одни и те же карточки.

**Поведение: last-write-wins.** Второй `UpdateSRS` перезапишет первый. Оба review_log сохраняются. SRS-состояние соответствует последнему review. Для MVP это допустимо — строгая защита через `SELECT ... FOR UPDATE` добавляет сложность без реальной пользы (два одновременных review одной карточки — экстремально редкий сценарий).

**Study sessions:** Unique constraint `ux_study_sessions_active` не позволяет двум ACTIVE сессиям. Второе устройство получит существующую сессию через `StartSession` (идемпотентность).

### 15.2. Смена timezone между reviews

Ранее вычисленный `dayStart` может сместиться при смене timezone. Часть reviews «перетечёт» в другой день. Для MVP это допустимо — смена timezone mid-day крайне редка. Streak может сброситься некорректно в этом edge case.

### 15.3. Изменение max_interval_days между reviews

Текущий interval не пересчитывается. Новое значение влияет только на будущие reviews (при вычислении нового интервала). Карточка с interval=200 при max=90 будет показана через 200 дней, но после review получит ≤ 90.

---

## 16. Обоснование ключевых решений

### 16.1. Почему SM-2 + Anki, а не FSRS?

FSRS — более современный алгоритм, но для MVP SM-2+Anki лучше: проверен десятилетиями, прост в реализации, не требует ML. Структура данных полностью совместима с миграцией на FSRS в будущем — review_logs содержат всю историю для обучения модели.

### 16.2. Почему prev_state в ReviewLog, а не отдельная таблица?

~20 байт на запись (5 полей). При 20M review_logs — ~400MB дополнительно. Это допустимо, а undo становится тривиальной операцией без сложных join/вычислений.

### 16.3. Почему overdue без лимита (вариант B)?

Пользователь, пропустивший 2 недели, должен иметь возможность вернуться в режим за 1–2 сессии. Ограничение overdue через `reviews_per_day` означало бы разгребание backlog неделями — это демотивирует. Лимит на new cards остаётся — это контролирует нагрузку от нового материала.

### 16.4. Почему entry без senses → нельзя создать карточку?

Карточка без содержимого бесполезна: клиент покажет front (слово), но back (ответ) будет пустым. Это frustrирует пользователя. Проверка `senseCount > 0` — дешёвый guard, который предотвращает создание бесполезных карточек.

### 16.5. Почему StudySession — отдельная сущность?

Агрегация на клиенте ненадёжна (закрыл вкладку — данные потеряны). Серверная сессия позволяет: показать «вы прошли 25 карточек за 12 минут, accuracy 76%» даже после перезагрузки. Это ключевой элемент мотивации. Сессия lightweight — JSONB для результатов, без отдельных таблиц для grade counts.

### 16.6. Почему undo ограничен 10 минутами?

Баланс между безопасностью и простотой. 10 минут достаточно, чтобы заметить ошибку (обычно замечают сразу). Без ограничения — риск злоупотребления (откат review недельной давности ломает SRS-модель). Значение конфигурируемое через `UndoWindowMinutes`.

---

## 17. Граничные значения и защитные проверки

| Параметр | Ограничение | Защита |
|----------|-------------|--------|
| ease_factor | ≥ 1.3 | `max(minEase, ease + delta)` в CalculateSRS |
| interval_days | ≥ 0 | Всегда `max(0, ...)` |
| interval_days (review) | ≥ old_interval + 1 (для GOOD/EASY) | Гарантирует рост |
| interval_days | ≤ min(config.max, user.max) | Cap в конце расчёта |
| learning_step | 0 ≤ step < len(steps) | Bounds check перед доступом |
| duration_ms | 0 ≤ ms ≤ 600000 | Validation в input |
| next_review_at | Всегда в будущем (относительно now) | SRS гарантирует now + duration |
| status transitions | Только допустимые | switch/case в CalculateSRS |
| undo window | ≤ 10 минут (конфигурируемо) | Проверка в UndoReview |
| senses count для card | ≥ 1 | Проверка в CreateCard и BatchCreateCards |
