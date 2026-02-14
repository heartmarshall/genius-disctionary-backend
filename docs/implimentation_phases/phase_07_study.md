# Фаза 7: Study сервис


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (интерфейсы потребителем), §2 (обработка ошибок), §3 (валидация), §4 (контекст и user identity), §5 (логирование), §6 (аудит), §7 (тестирование, moq), §11 (конфигурация SRS) |
| `services/service_layer_spec_v4.md` | §2 (структура пакетов: study/), §3 (паттерны), §4 (аудит: CARD), §5 (application-level limits: new cards/day, reviews/day), §6 (карта сервисов: StudyService), §7 (тестирование: SRS table-driven) |
| `services/study_service_spec_v4_v1.1.md` | Все секции — полная спецификация Study Service: SRS algorithm, 12 операций, undo, sessions, dashboard, ~64 теста, ~35 SRS кейсов |
| `services/business_scenarios_v4.md` | S1–S8 (Study), S9 (Sessions), S10 (UndoReview) |
| `data_model_v4.md` | §5 (cards, review_logs), §10 (soft delete: поведение для карточек) |
| `repo/repo_layer_spec_v4.md` | §6 (принципы), §8 (soft delete: JOIN entries), §9 (конкурентность) |

---

## Пре-условия (из Фазы 1)

Перед началом Фазы 7 должны быть готовы:

- Domain-модели: `Card`, `ReviewLog`, `CardSnapshot`, `SRSResult`, `StudySession` (`internal/domain/card.go`)
- Domain-модели: `Entry` с `DeletedAt` soft delete (`internal/domain/entry.go`)
- Domain-модели: `UserSettings` с `NewCardsPerDay`, `ReviewsPerDay`, `MaxIntervalDays`, `Timezone` (`internal/domain/user.go`)
- Domain-модели: `AuditRecord` (`internal/domain/organization.go`)
- Domain errors: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized` (`internal/domain/errors.go`)
- `ValidationError`, `FieldError`, `NewValidationError()` (`internal/domain/errors.go`)
- Enums: `LearningStatus` (NEW, LEARNING, REVIEW, MASTERED), `ReviewGrade` (AGAIN, HARD, GOOD, EASY), `EntityType` (CARD), `AuditAction` (CREATE, UPDATE, DELETE) (`internal/domain/enums.go`)
- Context helpers: `ctxutil.UserIDFromCtx(ctx) → (uuid.UUID, bool)` (`pkg/ctxutil/`)
- Config: `SRSConfig` с `DefaultEaseFactor`, `MinEaseFactor`, `MaxIntervalDays`, `GraduatingInterval`, `LearningSteps` (`internal/config/config.go`)

> **Важно:** Фаза 7 **не зависит** от Фаз 2, 3, 4, 5, 6. Все зависимости на репозитории, TxManager и settingsRepo мокаются в unit-тестах. Study Service может разрабатываться параллельно с другими сервисами.

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | SRS алгоритм | SM-2 + Anki-модификации. Не FSRS — для MVP проверенный алгоритм проще и достаточен. Данные совместимы с миграцией на FSRS |
| 2 | SRS чистая функция | `CalculateSRS(input SRSInput) SRSOutput` — в `service/study/srs.go`. Без side effects, без context, без БД. Тестируется изолированно |
| 3 | Overdue reviews | **Без лимита** — `reviews_per_day` рекомендательный. Overdue не блокируются. Пользователь может разгрести backlog за 1–2 сессии |
| 4 | New cards лимит | Строгий: `settings.new_cards_per_day`. Считается по `review_logs` за сегодня (timezone-aware) |
| 5 | MASTERED как метка | MASTERED — не отдельная фаза. Карточка продолжает повторяться. При AGAIN → откатывается через relearning |
| 6 | Undo механизм | `PrevState *CardSnapshot` в ReviewLog. Undo восстанавливает snapshot + удаляет review_log. Окно: 10 минут (конфигурируемо) |
| 7 | Study sessions | Lightweight: JSONB для результатов. Не более одной ACTIVE на пользователя (unique constraint). Идемпотентный StartSession |
| 8 | Fuzz factor | Детерминированный (card_id + review_count). Только для интервалов ≥ 3 дней. ±5% |
| 9 | Timezone | `next_review_at` в UTC. Daily limits и streak считаются по timezone пользователя из `user_settings` |
| 10 | Domain SRSConfig | Новый `domain.SRSConfig` — чистый domain-тип без env-тегов. `config.SRSConfig` расширяется недостающими полями. Bootstrap маппит config → domain |
| 11 | Card.IsDue() | Текущая реализация некорректна — MASTERED `return false`. Нужно исправить: MASTERED тоже due при `next_review_at <= now` |
| 12 | StudySession модель | Текущая модель упрощена (CardsStudied, AbandonedAt). Нужно обновить: Status enum, Result *SessionResult |
| 13 | PrevState в ReviewLog | Текущая доменная модель уже использует `PrevState *CardSnapshot`. Репозиторий маппит в отдельные колонки (миграция Phase 2) |
| 14 | Entry без senses → нельзя создать карточку | Проверка `senseCount > 0` в CreateCard и BatchCreateCards. Пустая карточка бесполезна |
| 15 | Конкурентность | Last-write-wins для параллельных review одной карточки. Допустимо для MVP |
| 16 | Моки | `moq` (code generation) — моки генерируются из приватных интерфейсов в `_test.go` файлы |
| 17 | Mock TxManager | `RunInTx(ctx, fn)` просто вызывает `fn(ctx)` без реальной транзакции |
| 18 | SessionResult агрегация | Через review_logs за период `[session.StartedAt, now]`. Сервис агрегирует, не repo |
| 19 | Streak расчёт | В сервисе, не в SQL. Загружаем DayReviewCount за последние 365 дней, считаем последовательные дни |
| 20 | GetCardStats | Все review logs загружаются, stats вычисляются в сервисе. Для MVP допустимо — количество reviews per card невелико |

---

## Задачи

### TASK-7.1: SRS Config Extension + Domain Model Updates

**Зависит от:** Фаза 1 (domain models, config)

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — §4 (domain models), §13 (SRS config)
- Текущий `internal/config/config.go` — SRSConfig неполный (нет EasyInterval, RelearningSteps, и др.)
- Текущий `internal/domain/card.go` — StudySession упрощён, Card.IsDue() некорректен для MASTERED

**Что сделать:**

Расширить SRSConfig, добавить недостающие domain-модели, обновить StudySession, исправить Card.IsDue().

---

#### 1. Расширить `config.SRSConfig`

**Файл:** `internal/config/config.go`

Добавить недостающие поля:

```go
type SRSConfig struct {
    DefaultEaseFactor    float64 `yaml:"default_ease_factor"    env:"SRS_DEFAULT_EASE"             env-default:"2.5"`
    MinEaseFactor        float64 `yaml:"min_ease_factor"        env:"SRS_MIN_EASE"                 env-default:"1.3"`
    MaxIntervalDays      int     `yaml:"max_interval_days"      env:"SRS_MAX_INTERVAL"             env-default:"365"`
    GraduatingInterval   int     `yaml:"graduating_interval"    env:"SRS_GRADUATING_INTERVAL"      env-default:"1"`
    EasyInterval         int     `yaml:"easy_interval"          env:"SRS_EASY_INTERVAL"            env-default:"4"`
    LearningStepsRaw     string  `yaml:"learning_steps"         env:"SRS_LEARNING_STEPS"           env-default:"1m,10m"`
    RelearningStepsRaw   string  `yaml:"relearning_steps"       env:"SRS_RELEARNING_STEPS"         env-default:"10m"`
    IntervalModifier     float64 `yaml:"interval_modifier"      env:"SRS_INTERVAL_MODIFIER"        env-default:"1.0"`
    HardIntervalModifier float64 `yaml:"hard_interval_modifier" env:"SRS_HARD_INTERVAL_MODIFIER"   env-default:"1.2"`
    EasyBonus            float64 `yaml:"easy_bonus"             env:"SRS_EASY_BONUS"               env-default:"1.3"`
    LapseNewInterval     float64 `yaml:"lapse_new_interval"     env:"SRS_LAPSE_NEW_INTERVAL"       env-default:"0.0"`
    UndoWindowMinutes    int     `yaml:"undo_window_minutes"    env:"SRS_UNDO_WINDOW_MINUTES"      env-default:"10"`
    NewCardsPerDay       int     `yaml:"new_cards_per_day"      env:"SRS_NEW_CARDS_DAY"            env-default:"20"`
    ReviewsPerDay        int     `yaml:"reviews_per_day"        env:"SRS_REVIEWS_DAY"              env-default:"200"`

    // Parsed from raw strings during validation.
    LearningSteps   []time.Duration `yaml:"-" env:"-"`
    RelearningSteps []time.Duration `yaml:"-" env:"-"`
}
```

**Новые поля:**
- `EasyInterval` (4) — интервал при EASY graduation
- `RelearningStepsRaw` / `RelearningSteps` (`"10m"` → `[10m]`) — шаги relearning после lapse
- `IntervalModifier` (1.0) — глобальный множитель интервалов
- `HardIntervalModifier` (1.2) — множитель для HARD в review
- `EasyBonus` (1.3) — множитель для EASY в review
- `LapseNewInterval` (0.0) — множитель интервала после lapse (0 = reset)
- `UndoWindowMinutes` (10) — окно для undo review

**Валидация (`validate.go`):**

```go
// SRS validation
if c.SRS.EasyInterval < 1 {
    return fmt.Errorf("srs.easy_interval must be >= 1")
}
if c.SRS.IntervalModifier <= 0 {
    return fmt.Errorf("srs.interval_modifier must be positive")
}
if c.SRS.HardIntervalModifier <= 0 {
    return fmt.Errorf("srs.hard_interval_modifier must be positive")
}
if c.SRS.EasyBonus <= 0 {
    return fmt.Errorf("srs.easy_bonus must be positive")
}
if c.SRS.LapseNewInterval < 0 || c.SRS.LapseNewInterval > 1 {
    return fmt.Errorf("srs.lapse_new_interval must be between 0.0 and 1.0")
}
if c.SRS.UndoWindowMinutes < 1 {
    return fmt.Errorf("srs.undo_window_minutes must be >= 1")
}
// Parse RelearningStepsRaw → RelearningSteps (аналогично LearningSteps)
```

---

#### 2. Добавить `domain.SRSConfig`

**Файл:** `internal/domain/card.go`

Чистый domain-тип без env-тегов. Используется чистой функцией `CalculateSRS` и Service struct:

```go
// SRSConfig holds SRS algorithm parameters. Clean domain type — no env/yaml tags.
// Populated from config.SRSConfig during bootstrap.
type SRSConfig struct {
    DefaultEaseFactor    float64
    MinEaseFactor        float64
    MaxIntervalDays      int
    GraduatingInterval   int
    EasyInterval         int
    LearningSteps        []time.Duration
    RelearningSteps      []time.Duration
    IntervalModifier     float64
    HardIntervalModifier float64
    EasyBonus            float64
    LapseNewInterval     float64
    UndoWindowMinutes    int
}
```

---

#### 3. Добавить `SessionStatus` enum

**Файл:** `internal/domain/enums.go`

```go
// SessionStatus represents the state of a study session.
type SessionStatus string

const (
    SessionStatusActive    SessionStatus = "ACTIVE"
    SessionStatusFinished  SessionStatus = "FINISHED"
    SessionStatusAbandoned SessionStatus = "ABANDONED"
)

func (s SessionStatus) String() string { return string(s) }

func (s SessionStatus) IsValid() bool {
    switch s {
    case SessionStatusActive, SessionStatusFinished, SessionStatusAbandoned:
        return true
    }
    return false
}
```

---

#### 4. Обновить `StudySession`

**Файл:** `internal/domain/card.go`

Заменить текущую упрощённую модель:

```go
// StudySession tracks a user's study session from start to finish.
type StudySession struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    Status     SessionStatus
    StartedAt  time.Time
    FinishedAt *time.Time
    Result     *SessionResult
    CreatedAt  time.Time
}

// SessionResult holds aggregated results of a completed study session.
type SessionResult struct {
    TotalReviewed int
    NewReviewed   int
    DueReviewed   int
    GradeCounts   GradeCounts
    DurationMs    int64
    AccuracyRate  float64
}

// GradeCounts holds per-grade counters for a study session.
type GradeCounts struct {
    Again int
    Hard  int
    Good  int
    Easy  int
}
```

---

#### 5. Добавить вспомогательные domain-типы

**Файл:** `internal/domain/card.go`

```go
// SRSUpdateParams holds the fields to update on a card after SRS calculation.
type SRSUpdateParams struct {
    Status       LearningStatus
    NextReviewAt time.Time
    IntervalDays int
    EaseFactor   float64
    LearningStep int
}

// CardStatusCounts holds the count of cards per learning status.
type CardStatusCounts struct {
    New      int
    Learning int
    Review   int
    Mastered int
    Total    int
}

// DayReviewCount holds the review count for a specific date.
type DayReviewCount struct {
    Date  time.Time
    Count int
}

// Dashboard holds aggregated study statistics for the user.
type Dashboard struct {
    DueCount      int
    NewCount      int
    ReviewedToday int
    NewToday      int
    Streak        int
    StatusCounts  CardStatusCounts
    OverdueCount  int
    ActiveSession *uuid.UUID
}

// CardStats holds statistics for a single card.
type CardStats struct {
    TotalReviews  int
    AccuracyRate  float64
    AverageTimeMs *int
    CurrentStatus LearningStatus
    IntervalDays  int
    EaseFactor    float64
}
```

---

#### 6. Исправить `Card.IsDue()`

**Файл:** `internal/domain/card.go`

Текущая реализация некорректна — MASTERED карточки тоже должны повторяться по расписанию:

```go
// IsDue returns true if the card needs review at the given time.
//   - NEW cards with no NextReviewAt are always due.
//   - LEARNING / REVIEW / MASTERED cards are due when NextReviewAt <= now.
func (c *Card) IsDue(now time.Time) bool {
    if c.Status == LearningStatusNew && c.NextReviewAt == nil {
        return true
    }
    return c.NextReviewAt != nil && !c.NextReviewAt.After(now)
}
```

---

**Acceptance criteria:**
- [ ] `config.SRSConfig` расширен 7 новыми полями с env-тегами и defaults
- [ ] `RelearningStepsRaw` парсится аналогично `LearningStepsRaw`
- [ ] Валидация новых полей: EasyInterval ≥ 1, модификаторы > 0, LapseNewInterval 0.0–1.0, UndoWindowMinutes ≥ 1
- [ ] `domain.SRSConfig` создан — чистый тип без тегов
- [ ] `SessionStatus` enum (ACTIVE, FINISHED, ABANDONED) с `IsValid()`
- [ ] `StudySession` обновлён: Status, Result *SessionResult, CreatedAt
- [ ] `SessionResult`, `GradeCounts` структуры созданы
- [ ] `SRSUpdateParams`, `CardStatusCounts`, `DayReviewCount`, `Dashboard`, `CardStats` созданы
- [ ] `Card.IsDue()` исправлен — MASTERED cards due при `next_review_at <= now`
- [ ] Существующие тесты `Card.IsDue()` обновлены (MASTERED + NextReviewAt в прошлом → true)
- [ ] Unit-тесты: новые defaults в SRSConfig корректны, невалидные значения → error
- [ ] `go build ./...` компилируется
- [ ] `go test ./...` — все существующие тесты проходят

---

### TASK-7.2: SRS Algorithm + Timezone Helpers

**Зависит от:** TASK-7.1 (domain.SRSConfig, domain types)

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — §5 (SRS Algorithm: полное описание, переходы, формулы), §5.5 (Fuzz factor), §5.6 (Timezone), §12.1 (35 тест-кейсов)
- `services/service_layer_spec_v4.md` — §7.4 (SRS table-driven tests ≥ 30 кейсов)

**Что сделать:**

Создать файлы `srs.go`, `srs_test.go`, `timezone.go`, `timezone_test.go` в пакете `internal/service/study/`.

**Файловая структура:**

```
internal/service/study/
├── srs.go              # CalculateSRS, calculateNew, calculateLearning, calculateReview, applyFuzz
├── srs_test.go         # ≥35 table-driven тестов
├── timezone.go         # DayStart, NextDayStart
└── timezone_test.go    # Тесты timezone helpers
```

---

#### `srs.go` — типы

```go
package study

import (
    "time"

    "github.com/heartmarshall/myenglish-backend/internal/domain"
)

// SRSInput holds all data needed for SRS calculation. Pure value — no side effects.
type SRSInput struct {
    CurrentStatus   domain.LearningStatus
    CurrentInterval int
    CurrentEase     float64
    LearningStep    int
    Grade           domain.ReviewGrade
    Now             time.Time
    Config          domain.SRSConfig
    MaxIntervalDays int  // min(config.MaxIntervalDays, user_settings.MaxIntervalDays)
}

// SRSOutput is the result of SRS calculation.
type SRSOutput struct {
    NewStatus       domain.LearningStatus
    NewInterval     int
    NewEase         float64
    NewLearningStep int
    NextReviewAt    time.Time
}
```

---

#### `srs.go` — CalculateSRS

```go
// CalculateSRS is a pure function. No DB, no context, no logger.
// All decisions are deterministic based on input parameters.
func CalculateSRS(input SRSInput) SRSOutput {
    switch input.CurrentStatus {
    case domain.LearningStatusNew:
        return calculateNew(input)
    case domain.LearningStatusLearning:
        return calculateLearning(input)
    case domain.LearningStatusReview, domain.LearningStatusMastered:
        return calculateReview(input)
    default:
        return calculateNew(input)
    }
}
```

**Приватные функции:**

**`calculateNew` — NEW → LEARNING/REVIEW:**

| Grade | Действие |
|-------|----------|
| AGAIN | LEARNING, шаг 0, next = now + steps[0] |
| HARD  | LEARNING, шаг 0, next = now + avg(steps[0], steps[1]) (если steps > 1, иначе steps[0]) |
| GOOD  | Если steps > 1: LEARNING шаг 1, next = now + steps[1]. Если steps ≤ 1: graduate |
| EASY  | Немедленный graduate → REVIEW, interval = easy_interval |

Если `len(steps) == 0`: любой grade кроме AGAIN → немедленный graduate.

**`calculateLearning` — LEARNING (learning/relearning):**

| Grade | Действие |
|-------|----------|
| AGAIN | Сброс на шаг 0, next = now + steps[0] |
| HARD  | Остаётся на текущем шаге, next = now + steps[current] |
| GOOD  | Следующий шаг. Если последний → **graduate** |
| EASY  | Немедленный graduate → REVIEW, interval = easy_interval |

**Graduate:**
- Status = REVIEW
- interval = graduating_interval (или easy_interval при EASY)
- ease = default_ease (при обычном graduate) или текущий ease (при relearning)
- next = now + interval days

**Определение learning vs relearning:** Если `input.CurrentInterval > 0` — это relearning (карточка уже была в REVIEW и сделала lapse). Steps берутся из `Config.RelearningSteps`. Иначе — из `Config.LearningSteps`.

**`calculateReview` — REVIEW/MASTERED:**

| Grade | Ease delta | New interval | Next status |
|-------|-----------|-------------|-------------|
| AGAIN | −0.20 (min 1.3) | max(1, old × lapse_new_interval) | → LEARNING (relearning), step=0 |
| HARD  | −0.15 (min 1.3) | old × hard_interval_modifier, min = old + 1 | REVIEW |
| GOOD  | 0 | old × ease × interval_modifier, min = old + 1 | REVIEW или MASTERED |
| EASY  | +0.15 | old × ease × easy_bonus × interval_modifier, min = old + 1 | REVIEW или MASTERED |

**Переход в MASTERED:** если `new_interval ≥ 21` AND `new_ease ≥ 2.5`.

**Cap:** `new_interval = min(new_interval, input.MaxIntervalDays)`.

**Fuzz:** применяется к интервалам ≥ 3 дней через `applyFuzz(interval, now)`.

---

#### `srs.go` — applyFuzz

```go
// applyFuzz adds deterministic jitter to prevent card clustering.
// Only applied to intervals >= 3 days. Range: ±5%.
func applyFuzz(interval int, now time.Time) int {
    if interval < 3 {
        return interval
    }
    fuzzRange := max(1, interval*5/100)
    // Deterministic seed from interval + timestamp day
    seed := int(now.UnixNano()/1e9) + interval
    fuzzDays := seed%(fuzzRange*2+1) - fuzzRange
    result := interval + fuzzDays
    if result < 1 {
        return 1
    }
    return result
}
```

---

#### `timezone.go`

```go
package study

import "time"

// DayStart returns the start of the current day in the user's timezone, converted to UTC.
func DayStart(now time.Time, tz *time.Location) time.Time {
    userNow := now.In(tz)
    dayStart := time.Date(userNow.Year(), userNow.Month(), userNow.Day(), 0, 0, 0, 0, tz)
    return dayStart.UTC()
}

// NextDayStart returns the start of the next day in the user's timezone, converted to UTC.
func NextDayStart(now time.Time, tz *time.Location) time.Time {
    return DayStart(now, tz).Add(24 * time.Hour)
}

// ParseTimezone parses a timezone string, returning UTC as fallback.
func ParseTimezone(tz string) *time.Location {
    loc, err := time.LoadLocation(tz)
    if err != nil {
        return time.UTC
    }
    return loc
}
```

---

#### `srs_test.go` — Table-driven tests (≥ 35 кейсов)

Полная таблица из study_service_spec §12.1:

| # | Категория | Input (Status/Step/Interval/Ease/Grade) | Expected (Status/Step/Interval/Ease) | Описание |
|---|-----------|---------------------------------------|--------------------------------------|----------|
| 1 | NEW→LEARNING | NEW/0/0/2.5/AGAIN | LEARNING/0/0/2.5, next=+1m | Забыл |
| 2 | NEW→LEARNING | NEW/0/0/2.5/HARD | LEARNING/0/0/2.5, next=+5.5m | Тяжело (avg шагов) |
| 3 | NEW→LEARNING | NEW/0/0/2.5/GOOD | LEARNING/1/0/2.5, next=+10m | Следующий шаг |
| 4 | NEW→REVIEW | NEW/0/0/2.5/EASY | REVIEW/0/4/2.5, next=+4d | Сразу graduate |
| 5 | LEARNING step 0 | LEARNING/0/0/2.5/AGAIN | LEARNING/0/0/2.5, next=+1m | Сброс |
| 6 | LEARNING step 0 | LEARNING/0/0/2.5/HARD | LEARNING/0/0/2.5, next=+1m | Повтор шага |
| 7 | LEARNING step 0 | LEARNING/0/0/2.5/GOOD | LEARNING/1/0/2.5, next=+10m | Шаг 1 |
| 8 | LEARNING step 0 | LEARNING/0/0/2.5/EASY | REVIEW/0/4/2.5, next=+4d | Graduate |
| 9 | LEARNING step 1 | LEARNING/1/0/2.5/AGAIN | LEARNING/0/0/2.5, next=+1m | Сброс |
| 10 | LEARNING step 1 | LEARNING/1/0/2.5/HARD | LEARNING/1/0/2.5, next=+10m | Повтор |
| 11 | LEARNING graduate | LEARNING/1/0/2.5/GOOD | REVIEW/0/1/2.5, next=+1d | Graduate |
| 12 | LEARNING EASY | LEARNING/1/0/2.5/EASY | REVIEW/0/4/2.5, next=+4d | Graduate easy |
| 13 | REVIEW AGAIN | REVIEW/0/1/2.5/AGAIN | LEARNING/0/1/2.3, next=+10m | Lapse → relearning |
| 14 | REVIEW HARD | REVIEW/0/1/2.5/HARD | REVIEW/0/2/2.35, next=+2d | ×1.2, ease −0.15 |
| 15 | REVIEW GOOD | REVIEW/0/1/2.5/GOOD | REVIEW/0/3/2.5, next=+3d | ×ease |
| 16 | REVIEW EASY | REVIEW/0/1/2.5/EASY | REVIEW/0/4/2.65, next=+4d | ×ease×1.3, ease +0.15 |
| 17 | REVIEW longer GOOD | REVIEW/0/10/2.5/GOOD | REVIEW/0/25/2.5, next=+25d | 10×2.5 |
| 18 | REVIEW longer HARD | REVIEW/0/10/2.5/HARD | REVIEW/0/12/2.35, next=+12d | 10×1.2 |
| 19 | REVIEW→MASTERED | REVIEW/0/21/2.5/GOOD | MASTERED/0/53/2.5 | interval≥21, ease≥2.5 |
| 20 | REVIEW not mastered | REVIEW/0/20/2.5/GOOD | REVIEW/0/50/2.5 | interval<21 |
| 21 | MASTERED GOOD | MASTERED/0/53/2.5/GOOD | MASTERED/0/133/2.5 | Остаётся |
| 22 | MASTERED AGAIN | MASTERED/0/53/2.5/AGAIN | LEARNING/0/1/2.3 | Lapse |
| 23 | Ease min | REVIEW/0/5/1.3/AGAIN | LEARNING/0/1/1.3 | Не ниже 1.3 |
| 24 | Ease at min+HARD | REVIEW/0/5/1.3/HARD | REVIEW/0/6/1.3 | Уже минимум |
| 25 | Max interval cap | REVIEW/0/200/2.5/GOOD (max=365) | REVIEW/0/365/2.5 | Global cap |
| 26 | User max interval | REVIEW/0/200/2.5/GOOD (max=180) | REVIEW/0/180/2.5 | User override |
| 27 | Min growth | REVIEW/0/10/2.5/GOOD | interval ≥ 11 | new ≥ old+1 |
| 28 | Relearning graduate | LEARNING/0/10/2.0/GOOD (relearn) | REVIEW/0/1/2.0 | After relearn |
| 29 | Relearning AGAIN | LEARNING/0/10/2.0/AGAIN (relearn) | LEARNING/0/10/2.0, next=+10m | Сброс relearning |
| 30 | Lapse reset (0.0) | REVIEW/0/30/2.5/AGAIN (lapse=0.0) | LEARNING/0/1/2.3 | interval=1 |
| 31 | Lapse 50% | REVIEW/0/30/2.5/AGAIN (lapse=0.5) | LEARNING/0/15/2.3 | 30×0.5 |
| 32 | Fuzz applied | REVIEW/0/10/2.5/GOOD | interval ~25 ±1 | ≤5% |
| 33 | Fuzz not applied | REVIEW/0/1/2.5/GOOD | exact interval | < 3 дней |
| 34 | Single step | config: steps=[10m], LEARNING/0/0/2.5/GOOD | REVIEW/0/1/2.5 | Graduate |
| 35 | Empty steps | config: steps=[], NEW/0/0/2.5/GOOD | REVIEW/0/1/2.5 | Немедленный graduate |

**timezone_test.go — тесты DayStart, NextDayStart, ParseTimezone:**

| # | Тест | Assert |
|---|------|--------|
| T1 | DayStart UTC midnight | 00:00 UTC |
| T2 | DayStart America/New_York | Корректный UTC offset |
| T3 | DayStart Asia/Tokyo | Корректный UTC offset |
| T4 | NextDayStart | DayStart + 24h |
| T5 | ParseTimezone valid | Correct Location |
| T6 | ParseTimezone invalid | Fallback UTC |

**Acceptance criteria:**
- [ ] `srs.go` создан с `CalculateSRS`, `calculateNew`, `calculateLearning`, `calculateReview`, `applyFuzz`
- [ ] `CalculateSRS` — чистая функция, без side effects
- [ ] **NEW:** AGAIN → step 0, HARD → step 0 (avg delay), GOOD → step 1 (или graduate), EASY → graduate
- [ ] **LEARNING:** AGAIN → reset step 0, HARD → repeat step, GOOD → next step / graduate, EASY → graduate
- [ ] **LEARNING→REVIEW (graduate):** interval = graduating_interval, ease preserved
- [ ] **REVIEW:** AGAIN → LEARNING (relearning), HARD/GOOD/EASY → interval grows, ease adjusted
- [ ] **REVIEW→MASTERED:** interval ≥ 21 AND ease ≥ 2.5
- [ ] **MASTERED AGAIN:** → LEARNING (relearning), ease −0.20
- [ ] **Relearning:** uses RelearningSteps, preserves current interval/ease
- [ ] **Boundaries:** ease ≥ 1.3, interval ≥ old+1 (GOOD/EASY), interval ≤ maxInterval
- [ ] **Fuzz:** only ≥ 3 days, ±5%, deterministic
- [ ] `timezone.go` с DayStart, NextDayStart, ParseTimezone
- [ ] ≥ 35 table-driven SRS тестов + 6 timezone тестов
- [ ] `go test ./internal/service/study/...` — все проходят
- [ ] `go vet ./internal/service/study/...` — без warnings

---

### TASK-7.3: Study Service — Foundation

**Зависит от:** TASK-7.1 (domain types)

> **Примечание:** TASK-7.3 **не зависит** от TASK-7.2 (SRS algorithm). Foundation определяет struct, интерфейсы и input-структуры. SRS algorithm подключается при реализации операций в TASK-7.4.

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — §3 (зависимости), §10 (валидация), §2 (файловая структура)
- `services/service_layer_spec_v4.md` — §3 (паттерны)

**Что сделать:**

Создать пакет `internal/service/study/` с foundation-компонентами: Service struct, приватные интерфейсы, input-структуры с валидацией, result-типы.

**Файловая структура (дополнение к TASK-7.2):**

```
internal/service/study/
├── service.go          # Service struct, конструктор, приватные интерфейсы
├── input.go            # Все input-структуры + Validate()
├── result.go           # BatchCreateResult и другие result-типы
├── srs.go              # (из TASK-7.2)
├── srs_test.go         # (из TASK-7.2)
├── timezone.go         # (из TASK-7.2)
├── timezone_test.go    # (из TASK-7.2)
└── service_test.go     # (TASK-7.4–7.6)
```

---

#### `service.go` — приватные интерфейсы

```go
package study

type cardRepo interface {
    GetByID(ctx context.Context, userID, cardID uuid.UUID) (*domain.Card, error)
    GetByEntryID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Card, error)
    Create(ctx context.Context, userID uuid.UUID, card *domain.Card) (*domain.Card, error)
    UpdateSRS(ctx context.Context, userID, cardID uuid.UUID, params domain.SRSUpdateParams) (*domain.Card, error)
    Delete(ctx context.Context, userID, cardID uuid.UUID) error
    GetDueCards(ctx context.Context, userID uuid.UUID, now time.Time, limit int) ([]*domain.Card, error)
    GetNewCards(ctx context.Context, userID uuid.UUID, limit int) ([]*domain.Card, error)
    CountByStatus(ctx context.Context, userID uuid.UUID) (domain.CardStatusCounts, error)
    CountDue(ctx context.Context, userID uuid.UUID, now time.Time) (int, error)
    CountNew(ctx context.Context, userID uuid.UUID) (int, error)
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
```

---

#### `service.go` — конструктор

```go
type Service struct {
    cards     cardRepo
    reviews   reviewLogRepo
    sessions  sessionRepo
    entries   entryRepo
    senses    senseRepo
    settings  settingsRepo
    audit     auditLogger
    tx        txManager
    log       *slog.Logger
    srsConfig domain.SRSConfig
}

func NewService(
    log       *slog.Logger,
    cards     cardRepo,
    reviews   reviewLogRepo,
    sessions  sessionRepo,
    entries   entryRepo,
    senses    senseRepo,
    settings  settingsRepo,
    audit     auditLogger,
    tx        txManager,
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

#### `input.go` — Input-структуры с валидацией

Все input-структуры из study_service_spec §10:

```go
type GetQueueInput struct {
    Limit int
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

```go
type ReviewCardInput struct {
    CardID     uuid.UUID
    Grade      domain.ReviewGrade
    DurationMs *int
    SessionID  *uuid.UUID
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

```go
type GetCardHistoryInput struct {
    CardID uuid.UUID
    Limit  int
    Offset int
}

func (i GetCardHistoryInput) Validate() error {
    var errs []domain.FieldError
    if i.CardID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "card_id", Message: "required"})
    }
    if i.Limit < 0 {
        errs = append(errs, domain.FieldError{Field: "limit", Message: "must be non-negative"})
    }
    if i.Limit > 200 {
        errs = append(errs, domain.FieldError{Field: "limit", Message: "max 200"})
    }
    if i.Offset < 0 {
        errs = append(errs, domain.FieldError{Field: "offset", Message: "must be non-negative"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

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

---

#### `result.go` — Result-типы

```go
package study

import "github.com/google/uuid"

// BatchCreateResult holds the outcome of a batch card creation.
type BatchCreateResult struct {
    Created         int
    SkippedExisting int
    SkippedNoSenses int
    Errors          []BatchCreateError
}

// BatchCreateError describes an error for a specific entry during batch creation.
type BatchCreateError struct {
    EntryID uuid.UUID
    Reason  string
}
```

---

**Acceptance criteria:**
- [ ] `service.go` создан с 8 приватными интерфейсами: cardRepo, reviewLogRepo, sessionRepo, entryRepo, senseRepo, settingsRepo, auditLogger, txManager
- [ ] Конструктор `NewService` с 10 параметрами, логгер `"service", "study"`
- [ ] `input.go` с 8 input-структурами, каждая с `Validate()`
- [ ] Все `Validate()` собирают все ошибки (не fail-fast)
- [ ] `result.go` с `BatchCreateResult`, `BatchCreateError`
- [ ] Defaults: limit=0 → 50 для GetQueueInput и GetCardHistoryInput (clamp в сервисе, не в Validate)
- [ ] `go build ./...` компилируется
- [ ] `go vet ./internal/service/study/...` — без warnings

---

### TASK-7.4: Study Service — Queue, Review & Undo

**Зависит от:** TASK-7.2 (SRS algorithm), TASK-7.3 (Service struct, interfaces, input structs)

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — §6.1 (GetStudyQueue), §6.2 (ReviewCard), §6.3 (UndoReview), §12.2 (tests #1–#28)
- `services/service_layer_spec_v4.md` — §4 (аудит: CARD UPDATE для review)

**Что сделать:**

Реализовать 3 ключевые операции: GetStudyQueue, ReviewCard, UndoReview. Написать unit-тесты.

---

#### Операция: GetStudyQueue(ctx, input GetQueueInput) → ([]*domain.Card, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()
3. limit = input.Limit; if limit == 0 { limit = 50 }

4. settings = settingsRepo.GetByUserID(ctx, userID)
5. tz = ParseTimezone(settings.Timezone)
   └─ невалидный → UTC, log WARN
6. dayStart = DayStart(now, tz)

7. newToday = reviewLogRepo.CountNewToday(ctx, userID, dayStart)
8. newRemaining = max(0, settings.NewCardsPerDay - newToday)

9. dueCards = cardRepo.GetDueCards(ctx, userID, now, limit)
   // Overdue НЕ ограничены reviews_per_day

10. if len(dueCards) < limit && newRemaining > 0:
    newLimit = min(limit - len(dueCards), newRemaining)
    newCards = cardRepo.GetNewCards(ctx, userID, newLimit)

11. queue = append(dueCards, newCards...)
12. log INFO: user_id, due_count, new_count, total
13. return queue
```

**Порядок:** LEARNING first → REVIEW/MASTERED overdue (by next_review_at ASC) → NEW (by created_at ASC). Порядок обеспечивается repo (SQL ORDER BY).

---

#### Операция: ReviewCard(ctx, input ReviewCardInput) → (*domain.Card, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. card = cardRepo.GetByID(ctx, userID, input.CardID) → ErrNotFound

4. settings = settingsRepo.GetByUserID(ctx, userID)
5. maxInterval = min(s.srsConfig.MaxIntervalDays, settings.MaxIntervalDays)

6. snapshot = CardSnapshot{
       Status: card.Status, LearningStep: card.LearningStep,
       IntervalDays: card.IntervalDays, EaseFactor: card.EaseFactor,
       NextReviewAt: card.NextReviewAt,
   }

7. srsResult = CalculateSRS(SRSInput{
       CurrentStatus: card.Status, CurrentInterval: card.IntervalDays,
       CurrentEase: card.EaseFactor, LearningStep: card.LearningStep,
       Grade: input.Grade, Now: now, Config: s.srsConfig,
       MaxIntervalDays: maxInterval,
   })

8. tx.RunInTx(ctx, func(ctx) error {
   8a. updatedCard = cardRepo.UpdateSRS(ctx, userID, card.ID, domain.SRSUpdateParams{
           Status: srsResult.NewStatus, NextReviewAt: srsResult.NextReviewAt,
           IntervalDays: srsResult.NewInterval, EaseFactor: srsResult.NewEase,
           LearningStep: srsResult.NewLearningStep,
       })
   8b. reviewLogRepo.Create(ctx, &domain.ReviewLog{
           CardID: card.ID, Grade: input.Grade,
           PrevState: &snapshot,
           DurationMs: input.DurationMs, ReviewedAt: now,
       })
   8c. audit.Log(ctx, AuditRecord{
           UserID: userID, EntityType: CARD, EntityID: &card.ID,
           Action: UPDATE, Changes: buildReviewChanges(card, srsResult, input.Grade),
       })
   })

9. log INFO: user_id, card_id, grade, old_status, new_status, new_interval
10. return updatedCard
```

---

#### Операция: UndoReview(ctx, input UndoReviewInput) → (*domain.Card, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. card = cardRepo.GetByID(ctx, userID, input.CardID) → ErrNotFound

4. lastLog = reviewLogRepo.GetLastByCardID(ctx, input.CardID)
   └─ ErrNotFound → ValidationError("card has no reviews to undo")

5. if lastLog.PrevState == nil → ValidationError("review cannot be undone")
6. if time.Since(lastLog.ReviewedAt) > time.Duration(s.srsConfig.UndoWindowMinutes) * time.Minute
   → ValidationError("undo window expired")

7. tx.RunInTx(ctx, func(ctx) error {
   7a. cardRepo.UpdateSRS(ctx, userID, card.ID, domain.SRSUpdateParams{
           Status: lastLog.PrevState.Status,
           NextReviewAt: *lastLog.PrevState.NextReviewAt, // handle nil
           IntervalDays: lastLog.PrevState.IntervalDays,
           EaseFactor: lastLog.PrevState.EaseFactor,
           LearningStep: lastLog.PrevState.LearningStep,
       })
   7b. reviewLogRepo.Delete(ctx, lastLog.ID)
   7c. audit.Log(ctx, AuditRecord{
           UserID: userID, EntityType: CARD, EntityID: &card.ID,
           Action: UPDATE, Changes: map[string]any{
               "undo": map[string]any{"old": lastLog.Grade},
               "status": map[string]any{"old": card.Status, "new": lastLog.PrevState.Status},
           },
       })
   })

8. log INFO: user_id, card_id, undone_grade, restored_status
9. return restored card (re-fetch or construct from PrevState)
```

**Обработка `PrevState.NextReviewAt == nil`:** Для NEW карточек `NextReviewAt` был nil. При undo UpdateSRS должен принять optional NextReviewAt. Вариант: repo принимает `*time.Time` и устанавливает NULL при nil. Либо SRSUpdateParams.NextReviewAt делается `*time.Time`.

---

#### Unit-тесты (из spec §12.2, #1–#28)

**GetStudyQueue:**

| # | Тест | Assert |
|---|------|--------|
| 1 | Success | Due cards + new cards в правильном порядке |
| 2 | NewLimitReached | newCardsPerDay исчерпан → только due cards |
| 3 | OverdueNotLimited | 500 overdue → все доступны |
| 4 | EmptyQueue | Нет карточек → пустой slice |
| 5 | Unauthorized | Нет userID → ErrUnauthorized |
| 6 | InvalidLimit | limit = -1 → ValidationError |
| 7 | DefaultLimit | limit = 0 → 50 |
| 8 | LearningCardsFirst | LEARNING в начале очереди |

**ReviewCard:**

| # | Тест | Assert |
|---|------|--------|
| 9 | NewToLearning | NEW + GOOD → LEARNING step 1 |
| 10 | GraduateToReview | LEARNING last step + GOOD → REVIEW |
| 11 | ReviewGood | REVIEW + GOOD → interval grows |
| 12 | ReviewAgainLapse | REVIEW + AGAIN → LEARNING (relearning) |
| 13 | ToMastered | REVIEW + GOOD, interval≥21, ease≥2.5 → MASTERED |
| 14 | PrevStateStored | ReviewLog содержит корректный prev_state snapshot |
| 15 | NotFound | Несуществующий card_id → ErrNotFound |
| 16 | WrongUser | Чужая карточка → ErrNotFound |
| 17 | InvalidGrade | Grade="INVALID" → ValidationError |
| 18 | NilCardID | CardID=nil → ValidationError |
| 19 | DurationTooLong | DurationMs=700000 → ValidationError |
| 20 | Audit | Audit record с корректными old/new |
| 21 | TransactionRollback | Ошибка в reviewLogRepo.Create → tx rollback |
| 22 | Unauthorized | Нет userID → ErrUnauthorized |

**UndoReview:**

| # | Тест | Assert |
|---|------|--------|
| 23 | Success | Карточка восстановлена, review_log удалён |
| 24 | RestoredState | Все SRS-поля восстановлены из prev_state |
| 25 | NoReviews | Нет логов → ValidationError |
| 26 | WindowExpired | > 10 минут → ValidationError |
| 27 | NoPrevState | Legacy лог без prev_state → ValidationError |
| 28 | Audit | Audit содержит undone grade |

**Всего: 28 тест-кейсов**

**Acceptance criteria:**
- [ ] **GetStudyQueue:** полный flow — settings → timezone → dayStart → countNewToday → due cards → new cards (limited) → merge
- [ ] **GetStudyQueue:** overdue **не ограничены** reviews_per_day
- [ ] **GetStudyQueue:** new cards ограничены `settings.NewCardsPerDay - countNewToday`
- [ ] **GetStudyQueue:** невалидный timezone → fallback UTC, WARN
- [ ] **GetStudyQueue:** limit=0 → default 50
- [ ] **ReviewCard:** полный flow — validate → load card → load settings → snapshot → CalculateSRS → tx(update + log + audit)
- [ ] **ReviewCard:** `PrevState` корректно сохраняется в ReviewLog (snapshot ДО review)
- [ ] **ReviewCard:** `maxInterval = min(srsConfig, userSettings)`
- [ ] **ReviewCard:** SessionID — optional, игнорируется если сессия не ACTIVE
- [ ] **UndoReview:** полный flow — validate → load card → load last log → check prev_state → check window → tx(restore + delete log + audit)
- [ ] **UndoReview:** окно 10 минут (из `srsConfig.UndoWindowMinutes`)
- [ ] **UndoReview:** PrevState=nil → ValidationError
- [ ] Все операции: ErrUnauthorized при отсутствии userID
- [ ] 28 unit-тестов покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/study/...` — все проходят

---

### TASK-7.5: Study Service — Session Operations

**Зависит от:** TASK-7.3 (Service struct, interfaces, input structs)

> **Примечание:** TASK-7.5 **не зависит** от TASK-7.4 (Queue & Review). Session операции используют только sessionRepo и reviewLogRepo. Могут разрабатываться параллельно с TASK-7.4.

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — §6.5 (StartSession), §6.6 (FinishSession), §6.7 (AbandonSession)
- `services/study_service_spec_v4_v1.1.md` — §12.2 (tests #46–#53)

**Что сделать:**

Реализовать 3 операции для study sessions: StartSession, FinishSession, AbandonSession. Написать unit-тесты.

---

#### Операция: StartSession(ctx) → (*domain.StudySession, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. existing = sessionRepo.GetActive(ctx, userID)
   └─ found → return existing (идемпотентность)

3. session = sessionRepo.Create(ctx, &domain.StudySession{
       UserID: userID, Status: ACTIVE, StartedAt: now,
   })
   └─ ErrAlreadyExists (race condition, unique constraint)
       → sessionRepo.GetActive(ctx, userID) → return existing

4. log INFO: user_id, session_id
5. return session
```

**Corner case:** Unique constraint `ux_study_sessions_active` гарантирует одну ACTIVE. Race condition: два запроса одновременно, один получит ErrAlreadyExists → загрузить существующую.

---

#### Операция: FinishSession(ctx, input FinishSessionInput) → (*domain.StudySession, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. session = sessionRepo.GetByID(ctx, userID, input.SessionID) → ErrNotFound

4. if session.Status != ACTIVE → ValidationError("session already finished")

5. Агрегировать результаты из review_logs за период [session.StartedAt, now]:
   // Загружаем все review logs за этот период
   // Вычисляем: totalReviewed, newReviewed (prev_status=NEW), gradeCounts
   // durationMs = now.Sub(session.StartedAt).Milliseconds()
   // accuracyRate = (good + easy) / total * 100 (0 если total = 0)

6. result = domain.SessionResult{
       TotalReviewed: total, NewReviewed: newReviewed,
       DueReviewed: total - newReviewed,
       GradeCounts: gradeCounts,
       DurationMs: durationMs, AccuracyRate: accuracy,
   }

7. finished = sessionRepo.Finish(ctx, userID, session.ID, result)
8. log INFO: user_id, session_id, total_reviewed, accuracy, duration
9. return finished
```

**Агрегация review logs:**

Сервис (не repo) агрегирует review_logs. Для этого нужен метод reviewLogRepo:

```go
// В reviewLogRepo interface добавить:
GetByPeriod(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.ReviewLog, error)
```

> **Обновление интерфейса:** В TASK-7.3 необходимо добавить `GetByPeriod` в `reviewLogRepo`. Если TASK-7.3 уже выполнен — добавить при реализации TASK-7.5.

**Вычисление newReviewed:** Review считается "new" если `log.PrevState != nil && log.PrevState.Status == NEW`. Это корректно определяет карточки, которые были NEW на момент review.

---

#### Операция: AbandonSession(ctx) → error

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. session = sessionRepo.GetActive(ctx, userID)
   └─ nil → return nil (noop, идемпотентность)

3. sessionRepo.Abandon(ctx, userID, session.ID)
4. log INFO: user_id, session_id
5. return nil
```

**Идемпотентность:** Если нет ACTIVE сессии — noop. Клиент может вызывать при старте приложения для cleanup.

---

#### Unit-тесты (из spec §12.2, #46–#53)

| # | Тест | Assert |
|---|------|--------|
| 46 | StartSession_Success | Сессия создана, status=ACTIVE |
| 47 | StartSession_AlreadyActive | Возвращает существующую ACTIVE сессию |
| 48 | FinishSession_Success | Сессия завершена, результаты агрегированы |
| 49 | FinishSession_AlreadyFinished | Сессия FINISHED → ValidationError |
| 50 | FinishSession_EmptySession | Нет reviews → totalReviewed=0, accuracy=0, сессия завершается |
| 51 | FinishSession_NotFound | Несуществующая → ErrNotFound |
| 52 | AbandonSession_Success | Сессия ABANDONED |
| 53 | AbandonSession_NoActive | Нет ACTIVE → noop (nil error) |

**Всего: 8 тест-кейсов**

**Acceptance criteria:**
- [ ] **StartSession:** идемпотентность — возвращает существующую ACTIVE сессию
- [ ] **StartSession:** race condition → ErrAlreadyExists → GetActive
- [ ] **FinishSession:** агрегация review_logs за период сессии
- [ ] **FinishSession:** корректный расчёт accuracyRate, newReviewed, durationMs
- [ ] **FinishSession:** пустая сессия (0 reviews) → завершается нормально
- [ ] **FinishSession:** уже FINISHED/ABANDONED → ValidationError
- [ ] **AbandonSession:** идемпотентность — нет ACTIVE → noop
- [ ] `reviewLogRepo` interface расширен `GetByPeriod` (если не включён в TASK-7.3)
- [ ] 8 unit-тестов покрывают все сценарии
- [ ] `go test ./internal/service/study/...` — все проходят

---

### TASK-7.6: Study Service — Card CRUD, Dashboard & Statistics

**Зависит от:** TASK-7.3 (Service struct, interfaces, input structs)

> **Примечание:** TASK-7.6 **не зависит** от TASK-7.4 и TASK-7.5. Может разрабатываться параллельно.

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — §6.4 (Dashboard), §6.8 (CreateCard), §6.9 (DeleteCard), §6.10 (GetCardHistory), §6.11 (GetCardStats), §6.12 (BatchCreateCards)
- `services/study_service_spec_v4_v1.1.md` — §12.2 (tests #29–#45, #54–#64)

**Что сделать:**

Реализовать 6 операций: CreateCard, DeleteCard, BatchCreateCards, GetDashboard, GetCardHistory, GetCardStats. Написать unit-тесты.

---

#### Операция: CreateCard(ctx, input CreateCardInput) → (*domain.Card, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. entryRepo.GetByID(ctx, userID, input.EntryID) → ErrNotFound

4. senseCount = senseRepo.CountByEntryID(ctx, input.EntryID)
   └─ senseCount == 0 → ValidationError("entry_id", "entry must have at least one sense to create a card")

5. tx.RunInTx(ctx, func(ctx) error {
   5a. card = cardRepo.Create(ctx, userID, &domain.Card{
           EntryID: input.EntryID, Status: NEW,
           EaseFactor: s.srsConfig.DefaultEaseFactor,
       })
       └─ ErrAlreadyExists → return ErrAlreadyExists
   5b. audit.Log(ctx, AuditRecord{
           UserID: userID, EntityType: CARD, EntityID: &card.ID,
           Action: CREATE, Changes: map[string]any{"entry_id": map[string]any{"new": input.EntryID}},
       })
   })

6. log INFO: user_id, card_id, entry_id
7. return card
```

---

#### Операция: DeleteCard(ctx, input DeleteCardInput) → error

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. card = cardRepo.GetByID(ctx, userID, input.CardID) → ErrNotFound

4. tx.RunInTx(ctx, func(ctx) error {
   4a. cardRepo.Delete(ctx, userID, input.CardID)
       // CASCADE удаляет review_logs
   4b. audit.Log(ctx, AuditRecord{
           EntityType: CARD, Action: DELETE,
           Changes: map[string]any{"entry_id": map[string]any{"old": card.EntryID}},
       })
   })

5. log INFO: user_id, card_id
6. return nil
```

---

#### Операция: BatchCreateCards(ctx, input BatchCreateCardsInput) → (*BatchCreateResult, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. existingEntries = entryRepo.ExistByIDs(ctx, userID, input.EntryIDs)
4. validEntryIDs = filter(input.EntryIDs, existingEntries)

5. Для каждого validEntryID:
   senseCount = senseRepo.CountByEntryID(ctx, entryID)
   └─ senseCount == 0 → skippedNoSenses++, skip

6. existingCards = cardRepo.ExistsByEntryIDs(ctx, userID, entriesWithContent)
7. toCreate = filter(entriesWithContent, !existingCards)

8. Для каждого batch из toCreate (по 50):
   tx.RunInTx(ctx, func(ctx) error {
       for each entryID in batch:
           card = cardRepo.Create(ctx, userID, &domain.Card{...})
           audit.Log(ctx, ...)
           created++
   })
   // Ошибка batch → log WARN, continue with next batch

9. return &BatchCreateResult{
       Created: created,
       SkippedExisting: skippedExisting,
       SkippedNoSenses: skippedNoSenses,
       Errors: errors,
   }
```

---

#### Операция: GetDashboard(ctx) → (*domain.Dashboard, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. settings = settingsRepo.GetByUserID(ctx, userID)
3. tz = ParseTimezone(settings.Timezone)
4. dayStart = DayStart(now, tz)

5. Последовательно (без транзакции):
   dueCount      = cardRepo.CountDue(ctx, userID, now)
   newCount      = cardRepo.CountNew(ctx, userID)
   reviewedToday = reviewLogRepo.CountToday(ctx, userID, dayStart)
   newToday      = reviewLogRepo.CountNewToday(ctx, userID, dayStart)
   statusCounts  = cardRepo.CountByStatus(ctx, userID)
   streakDays    = reviewLogRepo.GetStreakDays(ctx, userID, dayStart, 365)
   activeSession = sessionRepo.GetActive(ctx, userID)

6. streak = calculateStreak(streakDays, dayStart)

7. return &domain.Dashboard{
       DueCount:      dueCount,
       NewCount:      newCount,
       ReviewedToday: reviewedToday,
       NewToday:      newToday,
       Streak:        streak,
       StatusCounts:  statusCounts,
       OverdueCount:  overdueFromDueCount,  // dueCount включает overdue
       ActiveSession: activeSessionID,
   }
```

**calculateStreak:**

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

---

#### Операция: GetCardHistory(ctx, input GetCardHistoryInput) → ([]*domain.ReviewLog, int, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()
3. limit = input.Limit; if limit == 0 { limit = 50 }

4. cardRepo.GetByID(ctx, userID, input.CardID) → ErrNotFound

5. logs, total = reviewLogRepo.GetByCardID(ctx, input.CardID, limit, input.Offset)
6. return logs, total, nil
```

---

#### Операция: GetCardStats(ctx, cardID uuid.UUID) → (*domain.CardStats, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. card = cardRepo.GetByID(ctx, userID, cardID) → ErrNotFound

3. logs, total = reviewLogRepo.GetByCardID(ctx, cardID, 0, 0)
   // limit=0 означает "все" — repo обрабатывает

4. Вычислить:
   totalReviews = total
   goodEasy = count(logs where grade in [GOOD, EASY])
   accuracyRate = if total > 0 { goodEasy * 100 / total } else { 0 }
   averageTimeMs = avg(logs.DurationMs where DurationMs != nil)

5. return &domain.CardStats{
       TotalReviews:  totalReviews,
       AccuracyRate:  accuracyRate,
       AverageTimeMs: averageTimeMs,
       CurrentStatus: card.Status,
       IntervalDays:  card.IntervalDays,
       EaseFactor:    card.EaseFactor,
   }
```

---

#### Unit-тесты (из spec §12.2, #29–#45, #54–#64)

**CreateCard:**

| # | Тест | Assert |
|---|------|--------|
| 29 | Success | Карточка создана, status=NEW, ease=default |
| 30 | EntryNotFound | ErrNotFound |
| 31 | AlreadyExists | ErrAlreadyExists |
| 32 | SoftDeletedEntry | ErrNotFound |
| 33 | EntryNoSenses | ValidationError "at least one sense" |
| 34 | Audit | Audit record создан |
| 35 | NilEntryID | ValidationError |

**DeleteCard:**

| # | Тест | Assert |
|---|------|--------|
| 36 | Success | Карточка удалена |
| 37 | NotFound | ErrNotFound |
| 38 | Audit | Audit с entry_id |

**GetDashboard:**

| # | Тест | Assert |
|---|------|--------|
| 39 | Success | Все счётчики корректны |
| 40 | NoCards | Всё = 0, streak = 0 |
| 41 | StreakCalculation | 5 дней подряд → streak=5 |
| 42 | StreakBroken | Пропуск дня → streak обрывается |
| 43 | StreakTodayNotReviewed | Сегодня нет reviews → streak от вчера |
| 44 | OverdueCount | Корректно отражает backlog |
| 45 | ActiveSession | ActiveSession ID присутствует |

**GetCardHistory:**

| # | Тест | Assert |
|---|------|--------|
| 54 | Success | Список review logs |
| 55 | CardNotFound | ErrNotFound |

**GetCardStats:**

| # | Тест | Assert |
|---|------|--------|
| 56 | Success | Accuracy=75% (3 GOOD+EASY из 4) |
| 57 | NoReviews | totalReviews=0, accuracy=0, averageTimeMs=nil |

**BatchCreateCards:**

| # | Тест | Assert |
|---|------|--------|
| 58 | Success | 5 entries → 5 карточек |
| 59 | SomeExist | 2 уже есть → 3 создано, skippedExisting=2 |
| 60 | SkipsNoSenses | Entries без senses пропускаются |
| 61 | AllExist | created=0 |
| 62 | EmptyInput | ValidationError |
| 63 | TooMany | 101 entry → ValidationError |

**SoftDelete/Restore:**

| # | Тест | Assert |
|---|------|--------|
| 64 | CardPreserved | После restore карточка с прогрессом снова в очереди |

**Всего: 28 тест-кейсов**

**Acceptance criteria:**
- [ ] **CreateCard:** validate → check entry → check senses > 0 → tx(create + audit)
- [ ] **CreateCard:** entry без senses → ValidationError
- [ ] **CreateCard:** duplicate → ErrAlreadyExists (unique constraint ux_cards_entry)
- [ ] **CreateCard:** status=NEW, ease=srsConfig.DefaultEaseFactor
- [ ] **DeleteCard:** CASCADE удаляет review_logs. Audit logged
- [ ] **BatchCreateCards:** проверка entries exist → check senses → filter existing cards → batch create (по 50)
- [ ] **BatchCreateCards:** partial success — skippedExisting, skippedNoSenses, errors отдельно
- [ ] **GetDashboard:** полный flow — settings → timezone → 7 repo calls → calculateStreak → Dashboard
- [ ] **GetDashboard:** streak корректен (последовательные дни, сегодняшний считается если есть reviews)
- [ ] **GetDashboard:** ActiveSession ID из sessionRepo.GetActive
- [ ] **GetCardHistory:** ownership check через cardRepo.GetByID, пагинация
- [ ] **GetCardStats:** accuracyRate, averageTimeMs (nil если нет duration), totalReviews
- [ ] **calculateStreak:** приватная функция, тестируется через GetDashboard + отдельные table-driven tests
- [ ] Все мутации: ErrUnauthorized при отсутствии userID
- [ ] 28 unit-тестов покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/study/...` — все проходят

---

## Сводка зависимостей задач

```
TASK-7.1 (Config + Domain) ──┬──→ TASK-7.2 (SRS Algorithm)
                              └──→ TASK-7.3 (Foundation)

TASK-7.2 (SRS Algorithm)   ──────→ TASK-7.4 (Queue & Review)

TASK-7.3 (Foundation)       ──┬──→ TASK-7.4 (Queue & Review)
                              ├──→ TASK-7.5 (Sessions)
                              └──→ TASK-7.6 (Card CRUD, Dashboard, Stats)
```

Детализация:
- **TASK-7.2** (SRS Algorithm) зависит от: TASK-7.1 (domain.SRSConfig)
- **TASK-7.3** (Foundation) зависит от: TASK-7.1 (domain types)
- **TASK-7.4** (Queue & Review) зависит от: TASK-7.2 (CalculateSRS) + TASK-7.3 (Service struct, inputs)
- **TASK-7.5** (Sessions) зависит от: TASK-7.3. **Не зависит** от TASK-7.2 и TASK-7.4
- **TASK-7.6** (Card CRUD, Dashboard, Stats) зависит от: TASK-7.3. **Не зависит** от TASK-7.2, TASK-7.4, TASK-7.5
- TASK-7.2 и TASK-7.3 не имеют взаимных зависимостей

---

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-7.1 (Config + Domain Updates) |
| 2 | TASK-7.2 (SRS Algorithm), TASK-7.3 (Foundation) |
| 3 | TASK-7.4 (Queue & Review), TASK-7.5 (Sessions), TASK-7.6 (Card CRUD, Dashboard, Stats) |

> При полной параллелизации — **3 sequential волны**. Волна 2 — 2 задачи параллельно. Волна 3 — до 3 задач параллельно.
> TASK-7.5 и TASK-7.6 не зависят от TASK-7.2 и TASK-7.4, поэтому могут начинаться сразу после TASK-7.3.

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] **Config:** SRSConfig расширен 7 полями (EasyInterval, RelearningSteps, IntervalModifier, HardIntervalModifier, EasyBonus, LapseNewInterval, UndoWindowMinutes)
- [ ] **Domain:** domain.SRSConfig создан как чистый тип
- [ ] **Domain:** SessionStatus enum с IsValid()
- [ ] **Domain:** StudySession обновлён (Status, Result, CreatedAt)
- [ ] **Domain:** Добавлены CardStatusCounts, DayReviewCount, Dashboard, CardStats, SRSUpdateParams, SessionResult, GradeCounts
- [ ] **Domain:** Card.IsDue() исправлен для MASTERED карточек
- [ ] **SRS Algorithm** — чистая функция CalculateSRS:
  - [ ] NEW: AGAIN/HARD/GOOD/EASY с learning steps
  - [ ] LEARNING: step progression, graduate, relearning
  - [ ] REVIEW: interval growth, ease adjustment, lapse → relearning
  - [ ] MASTERED: continues reviewing, lapse downgrades
  - [ ] Boundaries: ease ≥ 1.3, min growth, max interval cap
  - [ ] Fuzz: ±5% для intervals ≥ 3 дней
- [ ] **SRS Algorithm** — ≥ 35 table-driven тестов
- [ ] **Timezone** — DayStart, NextDayStart, ParseTimezone с fallback UTC
- [ ] **Study Service** — все 12 операций реализованы:
  - [ ] GetStudyQueue: overdue без лимита, new cards limited, timezone-aware
  - [ ] ReviewCard: CalculateSRS, PrevState snapshot, audit
  - [ ] UndoReview: restore from PrevState, delete log, 10-min window
  - [ ] StartSession: идемпотентная (existing ACTIVE → return)
  - [ ] FinishSession: aggregate review_logs → SessionResult
  - [ ] AbandonSession: идемпотентная (no ACTIVE → noop)
  - [ ] CreateCard: entry exists + has senses, status=NEW
  - [ ] DeleteCard: CASCADE review_logs, audit
  - [ ] BatchCreateCards: filter exists/senses/cards, batch by 50
  - [ ] GetDashboard: 7 repo calls, calculateStreak
  - [ ] GetCardHistory: ownership check, pagination
  - [ ] GetCardStats: accuracy, averageTime, current state
- [ ] **Study Service** — ~64 unit-теста покрывают все сценарии из spec §12.2
- [ ] Аудит: CARD CREATE/UPDATE/DELETE в транзакциях
- [ ] Логирование соответствует спецификации (INFO/WARN/ERROR)
- [ ] Моки сгенерированы через `moq` из приватных интерфейсов
- [ ] Все input-структуры с `Validate()` — собирают все ошибки
- [ ] Все acceptance criteria всех 6 задач выполнены
