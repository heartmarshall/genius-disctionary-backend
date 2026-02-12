# MyEnglish Backend v4 — Code Conventions

Этот документ фиксирует **все ключевые архитектурные решения и правила написания кода** для backend_v4 проекта MyEnglish. Каждый модуль, каждый PR должен следовать этим конвенциям. Документ является living document и обновляется по мере развития проекта.

---

## 1. Структура проекта

```
backend_v4/
├── cmd/
│   └── server/
│       └── main.go                  # Точка входа, wiring
├── internal/
│   ├── app/                         # Bootstrap: logger, health, graceful shutdown
│   ├── config/                      # Конфигурация (env + yaml)
│   ├── domain/                      # Доменные модели (entities, value objects, enums)
│   │   ├── entry.go
│   │   ├── card.go
│   │   ├── user.go
│   │   ├── srs.go                   # SRS value objects (ReviewGrade, SRSParams)
│   │   └── enums.go
│   ├── service/                     # Бизнес-логика
│   │   ├── dictionary/
│   │   │   ├── service.go           # Содержит интерфейсы зависимостей этого сервиса
│   │   │   ├── service_test.go
│   │   │   ├── input.go
│   │   │   └── ...
│   │   ├── study/
│   │   ├── card/
│   │   ├── inbox/
│   │   ├── auth/
│   │   └── ...
│   ├── adapter/                     # Реализации: БД, внешние API
│   │   ├── postgres/                # Реализации репозиториев
│   │   │   ├── query/               # SQL-файлы для sqlc
│   │   │   ├── sqlc/                # Сгенерированный код (go generate)
│   │   │   ├── sqlc.yaml            # Конфигурация sqlc
│   │   │   ├── entry_repo.go        # Тонкий слой: sqlc models → domain
│   │   │   ├── card_repo.go
│   │   │   ├── txmanager.go
│   │   │   └── ...
│   │   └── provider/                # Внешние API-клиенты
│   │       ├── freedict/
│   │       └── google_oauth/
│   └── transport/                   # HTTP / GraphQL слой
│       ├── graphql/
│       │   ├── resolver/
│       │   │   └── resolver.go      # Содержит интерфейсы сервисов, нужных resolvers
│       │   ├── dataloader/
│       │   ├── middleware/
│       │   └── schema/
│       └── rest/                    # Health, metrics, auth callback
├── migrations/
├── docs/
│   ├── code_conventions.md          # Этот документ
│   ├── architecture.md
│   └── api.md
├── scripts/
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

### Принципы организации

- **`domain/`** — чистые Go-структуры без зависимостей от БД, HTTP, фреймворков. Никаких тегов `db:""`, `json:""`. Это ядро приложения.
- **`service/`** — бизнес-логика. Каждый сервис **сам определяет интерфейсы** своих зависимостей (репозиториев, других сервисов, внешних провайдеров) — прямо в своём пакете. Зависит только от `domain/`.
- **`adapter/`** — реализации: PostgreSQL, внешние API. Зависит от `domain/`. Реализует интерфейсы, определённые в `service/`, но **не импортирует** `service/` — связывание происходит в `main.go` через duck typing.
- **`transport/`** — HTTP/GraphQL. Определяет **свои** интерфейсы сервисов (только нужные методы). Маппит DTO ↔ domain.

### Интерфейсы: определяются потребителем

Следуем идиоматическому Go: **"accept interfaces, define them where they are used"**.

Никакого централизованного пакета `port/` или `interfaces/`. Каждый пакет определяет минимальный интерфейс, описывающий только то, что ему нужно:

```go
// service/dictionary/service.go — сервис определяет свои зависимости
package dictionary

type entryRepo interface {
    GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.DictionaryEntry, error)
    FindByText(ctx context.Context, userID uuid.UUID, text string) (*domain.DictionaryEntry, error)
    Create(ctx context.Context, userID uuid.UUID, e *domain.DictionaryEntry) (*domain.DictionaryEntry, error)
    Delete(ctx context.Context, userID uuid.UUID, id uuid.UUID) error
}

type auditLogger interface {
    Log(ctx context.Context, record domain.AuditRecord) error
}

type Service struct {
    entries entryRepo
    audit   auditLogger
    // ...
}
```

```go
// transport/graphql/resolver/resolver.go — resolver определяет, что ему нужно от сервисов
package resolver

type dictionaryService interface {
    GetEntry(ctx context.Context, id uuid.UUID) (*domain.DictionaryEntry, error)
    CreateEntry(ctx context.Context, input ???) (*domain.DictionaryEntry, error)
}

type studyService interface {
    GetStudyQueue(ctx context.Context, limit int) ([]domain.DictionaryEntry, error)
    ReviewCard(ctx context.Context, input ???) (*domain.ReviewResult, error)
}
```

Связывание происходит в `main.go` через Go duck typing — конкретные типы из `adapter/` передаются в конструкторы `service/`, которые принимают интерфейсы:

```go
// cmd/server/main.go
entryRepo := postgres.NewEntryRepo(pool)       // возвращает *postgres.EntryRepo
dictService := dictionary.NewService(entryRepo) // принимает entryRepo interface
```

### Правило зависимостей

```
transport → (свои интерфейсы) ← service → (свои интерфейсы) ← adapter
                                    ↓
                                  domain
```

Все стрелки импорта направлены внутрь. Связывание — только в `main.go`.

**Запрещено:**
- `service/` импортирует `adapter/` или `transport/`
- `domain/` импортирует что-либо кроме stdlib
- `transport/` импортирует `adapter/`
- Циклические зависимости между пакетами
- Централизованные пакеты с интерфейсами (`port/`, `interfaces/`, `contract/`)

---

## 2. Обработка ошибок

### 2.1. Единый набор ошибок

В v4 используется **один** пакет с бизнес-ошибками — `domain/errors.go`. Никаких дублирующих sentinel errors в разных слоях.

```go
// domain/errors.go
package domain

import "errors"

// Sentinel errors — бизнес-ошибки, понятные всем слоям.
var (
    ErrNotFound       = errors.New("not found")
    ErrAlreadyExists  = errors.New("already exists")
    ErrValidation     = errors.New("validation error")
    ErrUnauthorized   = errors.New("unauthorized")
    ErrForbidden      = errors.New("forbidden")
    ErrConflict       = errors.New("conflict")
)
```

### 2.2. Ошибки валидации

Для ошибок валидации используется структурированный тип, который позволяет возвращать ошибки по нескольким полям одновременно:

```go
// domain/errors.go

// FieldError описывает ошибку конкретного поля.
type FieldError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

// ValidationError содержит список ошибок валидации.
type ValidationError struct {
    Errors []FieldError
}

func (e *ValidationError) Error() string {
    if len(e.Errors) == 1 {
        return fmt.Sprintf("validation: %s — %s", e.Errors[0].Field, e.Errors[0].Message)
    }
    return fmt.Sprintf("validation: %d errors", len(e.Errors))
}

func (e *ValidationError) Unwrap() error { return ErrValidation }

func NewValidationError(field, message string) *ValidationError {
    return &ValidationError{
        Errors: []FieldError{{Field: field, Message: message}},
    }
}
```

### 2.3. Как оборачивать ошибки по слоям

**Adapter (repository):** Переводит infrastructure-ошибки в domain-ошибки. Добавляет контекст операции.

```go
// adapter/postgres/entry_repo.go
func (r *EntryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DictionaryEntry, error) {
    // ...
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, fmt.Errorf("entry %s: %w", id, domain.ErrNotFound)
    }
    return nil, fmt.Errorf("entry_repo.GetByID: %w", err)
}
```

**Service:** Не перетранслирует domain-ошибки, просто прокидывает. Оборачивает только неожиданные ошибки.

```go
// service/dictionary/service.go
func (s *Service) GetEntry(ctx context.Context, id uuid.UUID) (*domain.DictionaryEntry, error) {
    entry, err := s.entryRepo.GetByID(ctx, id)
    if err != nil {
        // domain.ErrNotFound пролетает как есть (через errors.Is)
        return nil, fmt.Errorf("dictionary.GetEntry: %w", err)
    }
    return entry, nil
}
```

**Transport:** Маппит domain-ошибки в HTTP/GraphQL коды.

```go
// transport/graphql/error_handler.go
func mapError(err error) *gqlerror.Error {
    switch {
    case errors.Is(err, domain.ErrNotFound):
        return gqlError(err, "NOT_FOUND")
    case errors.Is(err, domain.ErrValidation):
        return gqlValidationError(err)
    case errors.Is(err, domain.ErrUnauthorized):
        return gqlError(err, "UNAUTHORIZED")
    default:
        // Логируем неожиданную ошибку, пользователю — generic
        slog.ErrorContext(ctx, "unexpected error", "error", err)
        return gqlError(errors.New("internal error"), "INTERNAL")
    }
}
```

### 2.4. Правила

| Правило | Описание |
|---------|----------|
| **Всегда оборачивай** | `fmt.Errorf("operation_name: %w", err)` — для трассировки |
| **Не скрывай sentinel** | Оборачивай через `%w`, чтобы `errors.Is()` работал по всей цепочке |
| **Не логируй и прокидывай** | Ошибка логируется **один раз** — в transport-слое или в middleware. Логирование в service + прокидывание = дубль |
| **Panic только в main** | Panic допустим только при инициализации в `main.go`. В остальном коде — `return err` |
| **Нет голых errors.New в сервисах** | Сервисы используют `domain.ErrXxx` или `domain.NewValidationError()` |

---

## 3. Валидация

### 3.1. Где валидировать

Валидация происходит **ровно в двух местах**:

| Слой | Что валидирует | Пример |
|------|---------------|--------|
| **Transport** | Формат входных данных (парсинг, типы, required fields) | UUID парсится, enum корректен, строка не пустая |
| **Service** | Бизнес-правила | Слово с таким текстом уже существует, у пользователя превышен лимит |

**Repository не валидирует.** Репозиторий доверяет данным, пришедшим от сервиса. Constraint violations в БД обрабатываются как domain-ошибки (ErrAlreadyExists, ErrConflict).

### 3.2. Как валидировать в Service

Каждый сервисный метод начинается с вызова `validate*()`:

```go
func (s *Service) CreateEntry(ctx context.Context, input CreateEntryInput) (*domain.DictionaryEntry, error) {
    if err := input.Validate(); err != nil {
        return nil, err  // *domain.ValidationError, unwraps to domain.ErrValidation
    }
    // ... бизнес-логика
}
```

Метод `Validate()` определяется на input-структурах:

```go
// service/dictionary/input.go
type CreateEntryInput struct {
    Text           string
    Senses         []SenseInput
    CreateCard     bool
    // ...
}

func (i CreateEntryInput) Validate() error {
    var errs []domain.FieldError

    if strings.TrimSpace(i.Text) == "" {
        errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
    }
    if len(i.Senses) == 0 {
        errs = append(errs, domain.FieldError{Field: "senses", Message: "at least one sense required"})
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

### 3.3. Правила

- Все input-структуры имеют метод `Validate() error`
- `Validate()` собирает **все** ошибки (не возвращает при первой)
- Transport-слой вызывает `Validate()` только для format-валидации (парсинг). Бизнес-валидация — в сервисе
- Никогда не валидируем `user_id` в сервисе — он приходит из middleware через context и считается доверенным

---

## 4. Контекст и User Identity

### 4.1. User ID в контексте

После аутентификации middleware помещает `userID` в context. Все слои ниже достают его оттуда:

```go
// internal/app/context.go
type ctxKey string
const userIDKey ctxKey = "user_id"

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
    return context.WithValue(ctx, userIDKey, id)
}

func UserIDFromCtx(ctx context.Context) (uuid.UUID, error) {
    id, ok := ctx.Value(userIDKey).(uuid.UUID)
    if !ok || id == uuid.Nil {
        return uuid.Nil, domain.ErrUnauthorized
    }
    return id, nil
}
```

### 4.2. Где доставать User ID

- **Transport (resolvers):** достаёт userID и передаёт в сервис явным параметром или через context
- **Service:** получает userID из context с помощью `UserIDFromCtx(ctx)` в начале каждого метода
- **Repository:** получает userID как параметр метода. Репозиторий **всегда** фильтрует данные по userID

```go
// Репозиторий ВСЕГДА принимает userID
func (r *EntryRepo) GetByID(ctx context.Context, userID uuid.UUID, id uuid.UUID) (*domain.DictionaryEntry, error)
func (r *EntryRepo) Find(ctx context.Context, userID uuid.UUID, filter EntryFilter) ([]domain.DictionaryEntry, error)
```

### 4.3. Request ID

Request ID прокидывается через context для трассировки. Middleware генерирует его или берёт из заголовка `X-Request-ID`.

```go
func RequestIDFromCtx(ctx context.Context) string {
    id, _ := ctx.Value(requestIDKey).(string)
    return id
}
```

---

## 5. Логирование

### 5.1. Стек

Используем `log/slog` из stdlib. Никаких сторонних библиотек (zap, zerolog). `slog` достаточно производителен и является стандартом Go.

### 5.2. Формат

- **Development:** `text` (human-readable)
- **Production:** `json` (structured, для Loki/ELK/CloudWatch)

### 5.3. Уровни

| Уровень | Когда использовать |
|---------|-------------------|
| `Debug` | Детали для отладки: SQL-запросы, параметры вызовов, промежуточные значения |
| `Info` | Значимые бизнес-события: пользователь создал слово, завершена сессия повторения |
| `Warn` | Нештатные, но обработанные ситуации: retry внешнего API, degraded mode |
| `Error` | Ошибки, требующие внимания: не удалось записать в БД, внешний сервис недоступен |

### 5.4. Что логировать по слоям

**Transport (middleware):** Каждый HTTP-запрос. Это единственное место, где логируется ВСЯ информация о запросе.

```
INFO  http.request  method=POST path=/graphql status=200 duration=45ms request_id=abc-123 user_id=def-456
ERROR http.request  method=POST path=/graphql status=500 duration=12ms request_id=abc-123 error="..."
```

**Service:** Значимые бизнес-события и неожиданные ошибки.

```
INFO  dictionary.CreateEntry  user_id=def-456 entry_id=ghi-789 text="abandon"
WARN  study.GetQueue          user_id=def-456 msg="no due cards"
ERROR auth.ValidateToken      msg="token verification failed" error="..."
```

**Adapter:** Debug-level для SQL и внешних вызовов. Error-level для инфраструктурных проблем.

```
DEBUG postgres.entry.GetByID  query_ms=2 entry_id=ghi-789
ERROR postgres.entry.Create   msg="unique violation" entry_text="abandon"
DEBUG freedict.Fetch          word="abandon" status=200 duration=230ms
```

### 5.5. Правила

| Правило | Описание |
|---------|----------|
| **Всегда добавляй request_id** | Через `slog.With("request_id", RequestIDFromCtx(ctx))` |
| **Всегда добавляй user_id** | Если доступен в контексте |
| **Не логируй sensitive data** | Никаких паролей, токенов, PII в логах |
| **Не дублируй** | Ошибка логируется один раз. Если прокидываешь выше — не логируй |
| **Используй slog.ErrorContext** | Для ошибок всегда используй `slog.ErrorContext(ctx, ...)` — это позволяет прикрепить контекст |
| **Structured fields** | `slog.String("entry_id", id.String())`, не `fmt.Sprintf` |

### 5.6. Logger injection

Logger прокидывается через конструктор в каждый сервис и адаптер:

```go
func NewService(logger *slog.Logger, entries entryRepo, ...) *Service {
    return &Service{
        log: logger.With("service", "dictionary"),
        // ...
    }
}
```

---

## 6. Аудит

### 6.1. Что аудитируем

Все **мутирующие** операции над основными сущностями:

- Dictionary Entries: create, update, delete
- Cards: create, review (SRS changes)
- Content (senses, examples, translations): create, update, delete
- User settings: changes

### 6.2. Как аудитируем

Аудит пишется **в рамках той же транзакции**, что и основная операция. Это гарантирует консистентность.

```go
// service/dictionary/create.go
func (s *Service) CreateEntry(ctx context.Context, input CreateEntryInput) (*domain.DictionaryEntry, error) {
    // ...
    var entry *domain.DictionaryEntry
    err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
        var err error
        entry, err = s.entryRepo.Create(ctx, userID, domainEntry)
        if err != nil {
            return err
        }
        return s.auditRepo.Log(ctx, domain.AuditRecord{
            UserID:     userID,
            EntityType: domain.EntityEntry,
            EntityID:   entry.ID,
            Action:     domain.ActionCreate,
            Changes:    auditChangesForCreate(entry),
        })
    })
    // ...
}
```

### 6.3. Формат changes

```json
{
  "text": {"new": "abandon"},
  "senses_count": {"new": 3},
  "card_created": {"new": true}
}
```

Для update — diff:

```json
{
  "text": {"old": "abandone", "new": "abandon"},
  "notes": {"old": null, "new": "Часто путаю с 'abandoned'"}
}
```

### 6.4. Правила

- Audit record **всегда** содержит `user_id`
- Audit **всегда** в транзакции с основной операцией
- Audit не должен ломать основную операцию — если audit-запись не записалась, вся транзакция откатывается. Это сознательный выбор: консистентность важнее availability аудита
- В production рассмотреть вынос audit в async (outbox pattern) если станет bottleneck

---

## 7. Тестирование

### 7.1. Стратегия

```
                  ┌─────────────┐
                  │   E2E Tests │  ← GraphQL integration (testcontainers)
                  ├─────────────┤
              ┌───┤ Integration │  ← Repository + real DB (testcontainers)
              │   ├─────────────┤
              │   │ Unit Tests  │  ← Service logic (mocked repos)
              │   └─────────────┘
              │        ▲
              │        │ Основной фокус
              └────────┘
```

**Основной фокус — unit-тесты сервисов.** Это самый ценный слой тестов: они быстрые, стабильные, тестируют бизнес-логику.

### 7.2. Unit-тесты сервисов

Сервисы зависят от локальных интерфейсов → легко мокать. Моки генерируются из интерфейсов, определённых в том же пакете.

```go
// service/dictionary/service_test.go
func TestCreateEntry_DuplicateText(t *testing.T) {
    // Arrange
    entryRepo := &entryRepoMock{
        FindByTextFunc: func(ctx context.Context, userID uuid.UUID, text string) (*domain.DictionaryEntry, error) {
            return &domain.DictionaryEntry{ID: uuid.New(), Text: "abandon"}, nil
        },
    }
    svc := dictionary.NewService(slog.Default(), entryRepo, ...)

    // Act
    _, err := svc.CreateEntry(ctx, dictionary.CreateEntryInput{Text: "abandon"})

    // Assert
    assert.ErrorIs(t, err, domain.ErrAlreadyExists)
}
```

### 7.3. TDD-подход

Для каждого нового модуля придерживаемся порядка:

1. **Определить интерфейсы зависимостей** в пакете сервиса
2. **Написать тесты** для сервиса (используя моки)
3. **Реализовать** сервис (добиться прохождения тестов)
4. **Написать integration-тесты** для репозитория
5. **Реализовать** репозиторий
6. **Написать E2E-тесты** для transport-слоя

### 7.4. Именование тестов

```go
func TestServiceName_MethodName_Scenario(t *testing.T)

// Примеры:
func TestDictionary_CreateEntry_Success(t *testing.T)
func TestDictionary_CreateEntry_DuplicateText(t *testing.T)
func TestDictionary_CreateEntry_EmptyText_ReturnsValidationError(t *testing.T)
func TestStudy_ReviewCard_AgainGrade_ResetsInterval(t *testing.T)
```

### 7.5. Table-Driven Tests для алгоритмов

Чистые функции (SRS, валидация) тестируем через table-driven tests:

```go
func TestCalculateSRS(t *testing.T) {
    tests := []struct {
        name     string
        card     domain.Card
        grade    domain.ReviewGrade
        wantInterval int
        wantStatus   domain.LearningStatus
    }{
        {
            name:  "new card, grade Good, interval becomes 1",
            card:  domain.Card{Status: domain.StatusNew, IntervalDays: 0, EaseFactor: 2.5},
            grade: domain.GradeGood,
            wantInterval: 1,
            wantStatus:   domain.StatusLearning,
        },
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := study.CalculateSRS(&tt.card, tt.grade, time.Now())
            assert.Equal(t, tt.wantInterval, result.IntervalDays)
            assert.Equal(t, tt.wantStatus, result.Status)
        })
    }
}
```

### 7.6. Моки

Используем **moq** (code generation) для генерации моков из локальных интерфейсов:

```bash
//go:generate moq -out entry_repo_mock_test.go -pkg dictionary entryRepo
```

Моки генерируются в `_test.go` файлы того же пакета (не экспортируются). Запуск: `go generate ./...`

Если один и тот же интерфейс нужен в тестах нескольких пакетов — это нормально. Каждый пакет генерирует свой мок под свой узкий интерфейс, а не шарит "общий" мок на 30 методов.

### 7.7. Testcontainers (Integration + E2E)

Для тестов с реальной БД используем testcontainers-go (уже есть в v3):

```go
func SetupTestDB(t *testing.T) *pgxpool.Pool {
    t.Helper()
    // ... запуск PostgreSQL в контейнере
    // ... применение миграций
    // ... возврат pool
}
```

### 7.8. Правила

- Каждый PR содержит тесты для нового/изменённого кода
- Coverage цель: **≥80%** для service layer
- Тесты не зависят друг от друга и могут запускаться параллельно
- Используем `t.Parallel()` где возможно
- Никаких `time.Sleep` — используем каналы, context, или clock injection
- Тест-файлы рядом с тестируемым кодом: `service.go` → `service_test.go`

---

## 8. Работа с базой данных

### 8.1. Querier и транзакции

Используем паттерн **context-aware transactions**. Querier (pool или tx) прокидывается через context:

```go
// adapter/postgres/txmanager.go
type TxManager struct {
    pool *pgxpool.Pool
}

func (m *TxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
    tx, err := m.pool.Begin(ctx)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback(ctx)
            panic(p)
        }
    }()

    // Помещаем tx в context — репозитории автоматически используют его
    txCtx := withTx(ctx, tx)
    if err := fn(txCtx); err != nil {
        _ = tx.Rollback(ctx)
        return err
    }
    return tx.Commit(ctx)
}
```

Репозитории достают querier из context:

```go
func (r *EntryRepo) querier(ctx context.Context) Querier {
    if tx := txFromCtx(ctx); tx != nil {
        return tx
    }
    return r.pool
}
```

### 8.2. SQL: sqlc + Squirrel

Используем **два инструмента**, каждый для своего класса запросов:

| Инструмент | Когда использовать | Примеры |
|------------|-------------------|---------|
| **sqlc** | Статические запросы с фиксированной структурой | `GetByID`, `Create`, `Delete`, `GetDueCards`, `GetDashboardStats` |
| **Squirrel** | Динамические запросы, где набор условий определяется в runtime | `Find(filter)`, `Dictionary(search, hasCard, partOfSpeech, ...)` |

#### sqlc — основной инструмент

Запросы пишутся в `.sql` файлах, sqlc генерирует type-safe Go-код:

```
adapter/postgres/
├── query/                      # SQL-файлы для sqlc
│   ├── entries.sql
│   ├── cards.sql
│   ├── senses.sql
│   └── ...
├── sqlc/                       # Сгенерированный код (go generate)
│   ├── db.go
│   ├── models.go
│   ├── entries.sql.go
│   └── ...
├── sqlc.yaml                   # Конфигурация sqlc
├── entry_repo.go               # Репозиторий: тонкий слой маппинга sqlc models → domain
├── card_repo.go
└── ...
```

Пример SQL-файла:

```sql
-- query/cards.sql

-- name: GetCardByID :one
SELECT id, user_id, entry_id, status, next_review_at,
       interval_days, ease_factor, created_at, updated_at
FROM cards
WHERE id = $1 AND user_id = $2;

-- name: GetDueCards :many
SELECT id, user_id, entry_id, status, next_review_at,
       interval_days, ease_factor, created_at, updated_at
FROM cards
WHERE user_id = $1
  AND (status = 'NEW' OR next_review_at <= $2)
ORDER BY
    CASE WHEN status = 'NEW' THEN 0 ELSE 1 END,
    next_review_at ASC
LIMIT $3;

-- name: CreateCard :one
INSERT INTO cards (user_id, entry_id, status, ease_factor)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSRSFields :exec
UPDATE cards
SET status = $3, next_review_at = $4, interval_days = $5,
    ease_factor = $6, updated_at = now()
WHERE id = $1 AND user_id = $2;
```

Репозиторий маппит sqlc-модели в domain:

```go
// adapter/postgres/card_repo.go
func (r *CardRepo) GetByID(ctx context.Context, userID, id uuid.UUID) (*domain.Card, error) {
    row, err := r.q(ctx).GetCardByID(ctx, sqlc.GetCardByIDParams{
        ID:     id,
        UserID: userID,
    })
    if err != nil {
        return nil, mapError(err, "card", id)
    }
    return cardToDomain(row), nil
}
```

#### Squirrel — для динамических запросов

Используется **только** когда набор `WHERE`-условий формируется динамически:

```go
// adapter/postgres/entry_repo.go
func (r *EntryRepo) Find(ctx context.Context, userID uuid.UUID, filter dictionary.Filter) ([]domain.DictionaryEntry, error) {
    qb := squirrel.Select("id", "text", "text_normalized", "notes", "created_at", "updated_at").
        From("dictionary_entries").
        Where(squirrel.Eq{"user_id": userID}).
        PlaceholderFormat(squirrel.Dollar)

    if filter.Search != "" {
        qb = qb.Where("text_normalized ILIKE ?", "%"+filter.Search+"%")
    }
    if filter.HasCard != nil {
        if *filter.HasCard {
            qb = qb.Where("EXISTS (SELECT 1 FROM cards c WHERE c.entry_id = dictionary_entries.id)")
        } else {
            qb = qb.Where("NOT EXISTS (SELECT 1 FROM cards c WHERE c.entry_id = dictionary_entries.id)")
        }
    }
    if filter.PartOfSpeech != nil {
        qb = qb.Where("EXISTS (SELECT 1 FROM senses s WHERE s.entry_id = dictionary_entries.id AND s.part_of_speech = ?)", *filter.PartOfSpeech)
    }

    // Сортировка и пагинация
    qb = qb.OrderBy(filter.OrderClause()).Limit(uint64(filter.Limit)).Offset(uint64(filter.Offset))

    query, args, err := qb.ToSql()
    if err != nil {
        return nil, fmt.Errorf("entry_repo.Find build query: %w", err)
    }
    // ...
}
```

#### sqlc.yaml

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query/"
    schema: "../../migrations/"
    gen:
      go:
        package: "sqlc"
        out: "sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: false
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
```

#### Правило выбора инструмента

**Простой тест:** если все `WHERE`-условия известны при написании кода → sqlc. Если набор условий зависит от пользовательского ввода (фильтры, поиск) → Squirrel.

Генерация: `sqlc generate` запускается через `go generate ./...` или `make generate`.

### 8.3. Миграции

- Инструмент: **goose**
- Формат имени: `YYYYMMDDHHMMSS_description.sql`
- Каждая миграция содержит `-- +goose Up` и `-- +goose Down`
- Down-миграция **обязательна** для всех миграций
- Миграции **идемпотентны** где возможно (`IF NOT EXISTS`)
- Все таблицы содержат `user_id` (multi-tenant by design)

### 8.4. Индексы

- Все foreign keys индексируются
- Все поля, по которым фильтруем — индексируются
- Composite index для multi-tenant: `(user_id, ...)` — user_id всегда первый
- Объяснение каждого индекса в комментарии

### 8.5. Правила

- Репозиторий **всегда** фильтрует по `user_id`. Нет ни одного запроса без `WHERE user_id = $1`
- Timeout на каждый запрос через context: `context.WithTimeout(ctx, 5*time.Second)`
- Batch-операции используют `COPY` или batch inserts (не один INSERT за раз в цикле)
- Soft delete для основных сущностей (dictionary_entries, cards). `deleted_at TIMESTAMPTZ`

---

## 9. Аутентификация и авторизация

### 9.1. OAuth Flow

```
Client → Google/Apple → Authorization Code
Client → Backend /auth/callback → Validate code → Issue JWT pair
Client → GraphQL (Authorization: Bearer <access_token>)
```

### 9.2. Токены

| Токен | TTL | Хранение |
|-------|-----|----------|
| Access Token (JWT) | 15 минут | Client (memory) |
| Refresh Token | 30 дней | БД (hashed), HTTPOnly cookie |

### 9.3. JWT Claims

```json
{
  "sub": "user-uuid",
  "exp": 1234567890,
  "iat": 1234567890,
  "iss": "myenglish"
}
```

### 9.4. Auth Middleware

```go
// transport/graphql/middleware/auth.go

// tokenValidator определяет то, что middleware нужно от auth-сервиса.
type tokenValidator interface {
    ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

func AuthMiddleware(auth tokenValidator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                // Для unauthenticated endpoints (health, auth callback)
                next.ServeHTTP(w, r)
                return
            }

            userID, err := auth.ValidateToken(r.Context(), token)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            ctx := WithUserID(r.Context(), userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

## 10. GraphQL конвенции

### 10.1. Schema design

- **Все мутации** требуют авторизации
- **Query** может иметь публичные эндпоинты (health)
- Input types всегда с суффиксом `Input`
- Payload types для мутаций: `CreateWordPayload`, `ReviewCardPayload`
- Enum names в UPPER_SNAKE_CASE

### 10.2. Pagination

Cursor-based pagination для списков:

```graphql
type DictionaryConnection {
  edges: [DictionaryEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type DictionaryEdge {
  node: DictionaryEntry!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

### 10.3. DataLoaders

DataLoaders используются для всех 1:N и N:N связей. Каждый DataLoader:
- Создаётся per-request (в middleware)
- Имеет timeout (100ms default)
- Batch size ≤ 100
- Кеширует в рамках одного запроса

---

## 11. Конфигурация

### 11.1. Приоритет

```
Environment variables > .env file > config.yaml > defaults
```

### 11.2. Правила

- Все секреты — через env variables (никогда в yaml/коде)
- Config struct с тегами `env:""` и `env-default:""`
- Валидация конфига при старте (fail fast)
- Разделение: `ServerConfig`, `DatabaseConfig`, `AuthConfig`, `SRSConfig`

### 11.3. SRS конфигурация

```go
type SRSConfig struct {
    DefaultEaseFactor  float64       `env:"SRS_DEFAULT_EASE" env-default:"2.5"`
    MinEaseFactor      float64       `env:"SRS_MIN_EASE" env-default:"1.3"`
    MaxIntervalDays    int           `env:"SRS_MAX_INTERVAL" env-default:"365"`
    GraduatingInterval int           `env:"SRS_GRADUATING_INTERVAL" env-default:"1"`
    LearningSteps      []time.Duration `env:"SRS_LEARNING_STEPS" env-default:"1m,10m"`
    NewCardsPerDay     int           `env:"SRS_NEW_CARDS_DAY" env-default:"20"`
    ReviewsPerDay      int           `env:"SRS_REVIEWS_DAY" env-default:"200"`
}
```

---

## 12. Naming Conventions

### 12.1. Go code

| Элемент | Стиль | Пример |
|---------|-------|--------|
| Package | lowercase, short | `dictionary`, `study`, `postgres` |
| Interface | с суффиксом -er или описательное | `EntryRepository`, `TokenValidator` |
| Struct | CamelCase | `DictionaryEntry`, `ReviewResult` |
| Method | CamelCase, глагол | `CreateEntry`, `GetByID` |
| Constant | CamelCase (exported), camelCase (unexported) | `MaxInterval`, `defaultEase` |
| Variable | camelCase | `entryRepo`, `userID` |
| Error var | `Err` prefix | `ErrNotFound`, `ErrValidation` |
| Test func | `Test<Target>_<Method>_<Scenario>` | `TestDictionary_Create_Success` |
| Mock | `<interface>Mock` (unexported, in `_test.go`) | `entryRepoMock` |
| File | snake_case | `entry_repo.go`, `srs_calculator.go` |

### 12.2. Database

| Элемент | Стиль | Пример |
|---------|-------|--------|
| Table | snake_case, plural | `dictionary_entries`, `review_logs` |
| Column | snake_case | `user_id`, `created_at`, `ease_factor` |
| Index | `ix_<table>_<columns>` | `ix_cards_user_status` |
| Unique index | `ux_<table>_<columns>` | `ux_entries_user_text` |
| Foreign key | `fk_<table>_<ref_table>` | `fk_cards_entries` |
| Enum type | snake_case | `learning_status`, `review_grade` |

### 12.3. GraphQL

| Элемент | Стиль | Пример |
|---------|-------|--------|
| Type | PascalCase | `DictionaryEntry`, `ReviewResult` |
| Field | camelCase | `textNormalized`, `nextReviewAt` |
| Enum value | UPPER_SNAKE | `NEW`, `LEARNING`, `GRADE_GOOD` |
| Input | PascalCase + `Input` | `CreateWordInput` |
| Mutation | camelCase, verb-first | `createWord`, `reviewCard` |
| Query | camelCase | `dictionary`, `studyQueue` |

---

## 13. Git & PR Conventions

### 13.1. Commit messages

Формат: `<type>(<scope>): <description>`

```
feat(study): add learning steps to SRS algorithm
fix(dictionary): handle duplicate text normalization
refactor(auth): extract token validation to separate service
test(study): add unit tests for calculateSRS
docs: update code conventions
chore: upgrade pgx to v5.9
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`

### 13.2. Branch naming

```
feature/study-learning-steps
fix/duplicate-entry-handling
refactor/auth-middleware
```

### 13.3. PR checklist

- [ ] Код соответствует Code Conventions
- [ ] Unit-тесты написаны и проходят
- [ ] Нет нарушений правил зависимостей между слоями
- [ ] Миграции содержат down-миграцию
- [ ] Логирование добавлено для значимых операций
- [ ] Аудит добавлен для мутирующих операций
- [ ] Все запросы фильтруются по user_id
- [ ] Нет sensitive data в логах

---

## 14. Порядок реализации модулей

Рекомендуемый порядок реализации для backend_v4:

1. **Skeleton** — структура проекта, domain models, ports, config, main.go
2. **Database** — миграции, txmanager, base repo
3. **Auth** — OAuth flow, JWT, middleware, user management
4. **Dictionary** — CRUD, поиск, фильтрация (core функционал)
5. **Content** — Senses, translations, examples, images, pronunciations
6. **Cards & SRS** — Карточки, алгоритм повторения, study sessions
7. **Study** — Очередь повторения, dashboard, daily limits
8. **Inbox** — Входящие, конвертация в слова
9. **Topics** — Категоризация, фильтрация по топикам
10. **Import/Export** — Импорт/экспорт словаря
11. **Suggestions** — Внешние провайдеры (FreeDict, OpenAI)

Каждый модуль реализуется в TDD-порядке: интерфейсы зависимостей → тесты → service → adapter → transport.
