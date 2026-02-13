# Фаза 1: Скелет проекта и доменный слой


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (структура проекта), §2 (ошибки), §4 (контекст и user identity), §5 (логирование), §11 (конфигурация), §12 (naming conventions) |
| `data_model_v4.md` | Все секции — определяют поля доменных моделей |
| `infra_spec_v4.md` | §2 (конфигурация), §3 (логирование), §9 (зависимости) |
| `services/study_service_spec_v4_v1.1.md` | Секция SRS — определяет дополнительные поля для Card и ReviewLog |

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | Module path | `github.com/heartmarshall/myenglish-backend` |
| 2 | Go version | 1.23+ |
| 3 | Config library | `cleanenv` (YAML + ENV, тег-валидация) |
| 4 | Logger | `log/slog` (stdlib). Никаких zap, zerolog |
| 5 | Context helpers | В `pkg/ctxutil/` — чистые утилиты без зависимостей от `internal/` |
| 6 | Domain: nullable поля | Pointer types: `*string`, `*uuid.UUID`, `*time.Time` |
| 7 | Domain: enums | String-based constants (не iota) |
| 8 | Domain: tags | Никаких `db:""`, `json:""`, `gql:""` — чистые Go-структуры |
| 9 | Файлы | snake_case: `entry.go`, `srs_calculator.go` |
| 10 | Зависимости main.go | Только вызов bootstrap-функции из `internal/app/` |

---

## Задачи

### TASK-1.1: Структура проекта и Go-модуль

**Зависит от:** ничего

**Контекст:**
- `code_conventions_v4.md` — §1 (структура проекта), §12 (naming conventions)
- `infra_spec_v4.md` — §1 (принципы), §9 (зависимости go.mod)

**Что сделать:**

Создать полную структуру директорий проекта и инициализировать Go-модуль.

**Директории:**
```
backend_v4/
├── cmd/
│   └── server/
├── internal/
│   ├── app/
│   ├── config/
│   ├── domain/
│   ├── service/
│   │   ├── auth/
│   │   ├── user/
│   │   ├── dictionary/
│   │   ├── content/
│   │   ├── study/
│   │   ├── inbox/
│   │   ├── topic/
│   │   └── refcatalog/
│   ├── adapter/
│   │   ├── postgres/
│   │   └── provider/
│   └── transport/
│       ├── graphql/
│       │   ├── resolver/
│       │   ├── dataloader/
│       │   ├── middleware/
│       │   └── schema/
│       └── rest/
├── pkg/
├── migrations/
├── docs/
├── scripts/
├── .gitignore
├── .env.example
└── go.mod
```

**Go-модуль:**
- Инициализировать `go.mod` с module path `github.com/heartmarshall/myenglish-backend`
- Добавлять зависимости **по мере использования** в каждой фазе. В Фазе 1 реально используются:
  - `github.com/google/uuid` — UUID в domain models и context helpers
  - `github.com/ilyakaznacheev/cleanenv` — config loading
- Остальные зависимости (pgx, gqlgen, squirrel, scany, jwt, goose, testcontainers) добавляются в соответствующих фазах, когда появляется код, их импортирующий. Иначе `go mod tidy` их удалит.

**`tools.go` (dev-зависимости):**

Создать файл `tools.go` в корне проекта с build tag `//go:build tools` для фиксации версий CLI-инструментов. В Фазе 1 пока пустой шаблон, инструменты добавляются по мере необходимости:
- `github.com/matryer/moq` — добавится в Фазе 3 (repository layer)
- `github.com/sqlc-dev/sqlc/cmd/sqlc` — добавится в Фазе 2 (database infra)
- `github.com/99designs/gqlgen` — добавится в Фазе 9 (transport)
- `github.com/pressly/goose/v3/cmd/goose` — добавится в Фазе 2 (database infra)

Без `tools.go` разные агенты/разработчики будут использовать разные версии инструментов.

**`.golangci.yml` — конфигурация линтера:**

Создать конфигурацию golangci-lint с минимальным набором линтеров для production:
- `errcheck` — проверка обработки ошибок
- `govet` — vet checks
- `staticcheck` — расширенный анализ
- `unused` — неиспользуемый код
- `gosimple` — упрощение кода
- `ineffassign` — бессмысленные присваивания
- `revive` — замена golint

Настройки: Go 1.23+, исключить `_test.go` из некоторых проверок, исключить сгенерированный код (`sqlc/`, `gqlgen generated`).

**.gitignore:** стандартный Go .gitignore + `.env`, `tmp/`, `vendor/`, `*.exe`, coverage файлы.

**.env.example:** шаблон с ключами всех ENV-переменных (без значений для секретов).

**Acceptance criteria:**
- [ ] `go mod tidy` завершается без ошибок
- [ ] Все директории из дерева выше созданы (с `.gitkeep` файлами для пустых)
- [ ] Module path: `github.com/heartmarshall/myenglish-backend`
- [ ] В `go.mod` только реально используемые зависимости (uuid, cleanenv)
- [ ] `tools.go` с `//go:build tools` создан (шаблон)
- [ ] `.golangci.yml` создан с линтерами: errcheck, govet, staticcheck, unused, gosimple, ineffassign, revive
- [ ] `.gitignore` содержит `.env`, `tmp/`, бинарники, coverage
- [ ] `.env.example` содержит шаблон переменных окружения

---

### TASK-1.2: Система конфигурации

**Зависит от:** TASK-1.1

**Контекст:**
- `code_conventions_v4.md` — §11 (конфигурация, SRS конфигурация)
- `infra_spec_v4.md` — §2 (подход, секции, рекомендации)

**Что сделать:**

Создать пакет `internal/config/` с единой корневой config-структурой.

**Секции конфигурации:**

| Секция | Поля | Примечания |
|--------|------|-----------|
| Server | Host, Port, ReadTimeout, WriteTimeout, IdleTimeout, ShutdownTimeout | Таймауты как `time.Duration` |
| Database | DSN, MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime | DSN через ENV |
| Auth | JWTSecret, AccessTokenTTL, RefreshTokenTTL, GoogleClientID, GoogleClientSecret, AppleKeyID, AppleTeamID, ApplePrivateKey | Все секреты через ENV |
| GraphQL | PlaygroundEnabled, IntrospectionEnabled, ComplexityLimit | Playground off в production |
| Log | Level, Format | Format: "json" или "text" |
| SRS | DefaultEaseFactor, MinEaseFactor, MaxIntervalDays, GraduatingInterval, LearningSteps, NewCardsPerDay, ReviewsPerDay | Дефолты из code_conventions §11.3 |

**Путь к конфигу:**
- Определяется через ENV `CONFIG_PATH` (абсолютный или относительный путь к YAML-файлу)
- Если `CONFIG_PATH` не задан — fallback на `./config.yaml` (текущая директория)
- Если файл не найден и `CONFIG_PATH` не задан — допустимо работать только на ENV + defaults

**Требования:**
- Загрузка через `cleanenv`: YAML-файл + ENV-переменные с перекрытием
- Приоритет: ENV > YAML > defaults (теги `env-default`)
- Fail-fast: если конфиг невалиден — `log.Fatal` до запуска любых компонентов
- Для обязательных полей использовать тег `env-required:"true"` где применимо (DSN, JWT secret)
- Метод `func (c *Config) Validate() error` — отдельная бизнес-валидация поверх тегов:
  - JWT secret длина ≥ 32 символа
  - Хотя бы один OAuth-провайдер сконфигурирован (Google или Apple)
  - SRS constraints: MinEaseFactor > 0, MaxIntervalDays > 0, NewCardsPerDay ≥ 0
- `Validate()` вызывается после загрузки. Тестируется отдельно от загрузки.

**`LearningSteps []time.Duration` — особый случай:**

cleanenv не умеет парсить `"1m,10m"` в `[]time.Duration` из коробки. Варианты решения:
- Custom unmarshaler (реализовать интерфейс `encoding.TextUnmarshaler` на обёртке)
- Хранить как строку `LearningStepsRaw string`, парсить в `[]time.Duration` в `Validate()` или отдельном методе

Выбор реализации — на усмотрение агента, но парсинг должен быть покрыт тестом.

**Создать также:**
- YAML-файл `config.yaml` с дефолтными значениями для development
- Обновить `.env.example` с полным списком ENV-переменных

**Acceptance criteria:**
- [ ] Config загружается из YAML + ENV
- [ ] ENV-переменные перекрывают YAML
- [ ] Путь к YAML: `CONFIG_PATH` ENV → fallback `./config.yaml`
- [ ] Все секции из таблицы присутствуют
- [ ] Таймауты парсятся как `time.Duration` (форматы "10s", "5m")
- [ ] `LearningSteps` корректно парсится из строки `"1m,10m"` в `[]time.Duration`
- [ ] `Validate()` — отдельный метод на Config
- [ ] Валидация: отсутствие JWT secret → ошибка
- [ ] Валидация: короткий JWT secret (< 32) → ошибка
- [ ] Валидация: отсутствие DSN → ошибка
- [ ] Валидация: отсутствие OAuth credentials → ошибка
- [ ] SRS-дефолты: ease 2.5, min ease 1.3, max interval 365, graduating 1, learning steps [1m, 10m], new cards 20, reviews 200
- [ ] `config.yaml` для development создан
- [ ] Unit-тест: загрузка с валидным конфигом
- [ ] Unit-тест: загрузка с невалидным конфигом → ошибка
- [ ] Unit-тест: `Validate()` с невалидными SRS-параметрами → ошибка
- [ ] Unit-тест: `LearningSteps` парсинг (валидный, пустой, невалидный формат)

---

### TASK-1.3: Логирование

**Зависит от:** TASK-1.2 (нужен config с Log секцией)

**Контекст:**
- `code_conventions_v4.md` — §5 (логирование: стек, формат, уровни, правила, injection)
- `infra_spec_v4.md` — §3 (подход, паттерны)

**Что сделать:**

Создать функцию инициализации логгера и хелперы для работы с ним.

**Инициализация логгера (в `internal/app/` или отдельный файл):**
- На основе `config.Log.Format`: `"json"` → `slog.NewJSONHandler`, `"text"` → `slog.NewTextHandler`
- На основе `config.Log.Level`: `"debug"` / `"info"` / `"warn"` / `"error"` → `slog.Level`
- Output: всегда `os.Stderr` (стандарт для логов; stdout зарезервирован для данных)
- В development mode (`"text"` format): включить `AddSource: true` в `slog.HandlerOptions` — добавляет file:line в лог для отладки
- Возвращает `*slog.Logger`
- Устанавливает logger как default (`slog.SetDefault`) — это осознанное решение: меняет глобальный логгер, чтобы вызовы `slog.Info(...)` без явного логгера тоже работали корректно

**Logger injection:**
- Каждый сервис и адаптер получает `*slog.Logger` через конструктор
- Конструктор добавляет scope: `logger.With("service", "dictionary")` или `logger.With("adapter", "postgres.entry")`

**Правила (зафиксировать в конвенциях сервисов — НЕ реализовывать в этой задаче):**
- Ошибки логируются один раз (в transport/middleware)
- Structured fields: `slog.String(...)`, не `fmt.Sprintf`
- Никаких `log.Println`
- Контекстные поля: `request_id`, `user_id` добавляются через middleware (будет в Фазе 10)

**Acceptance criteria:**
- [ ] Функция `NewLogger(cfg config.Log) *slog.Logger` создана
- [ ] JSON handler для production, text handler для development
- [ ] Level конфигурируется
- [ ] Logger устанавливается как `slog.SetDefault`
- [ ] Unit-тест: создание logger с разными конфигами (json/text, debug/info/warn/error)

---

### TASK-1.4: Context Helpers

**Зависит от:** TASK-1.1

**Контекст:**
- `code_conventions_v4.md` — §4 (контекст и user identity, request ID)
- `infra_spec_v4.md` — §3 (context-aware логгер)

**Что сделать:**

Создать пакет утилит для работы с контекстом. Расположение: `pkg/ctxutil/` — чистый utility-пакет **без зависимостей от `internal/`**.

**Функции:**

| Функция | Описание |
|---------|----------|
| `WithUserID(ctx, uuid.UUID) context.Context` | Кладёт user ID в context |
| `UserIDFromCtx(ctx) (uuid.UUID, bool)` | Извлекает user ID; возвращает `false` если отсутствует или uuid.Nil |
| `WithRequestID(ctx, string) context.Context` | Кладёт request ID в context |
| `RequestIDFromCtx(ctx) string` | Извлекает request ID; возвращает пустую строку если отсутствует |

**Требования:**
- Ключи контекста: unexported типы (`type ctxKey string`) для предотвращения коллизий
- `UserIDFromCtx` возвращает `(uuid.UUID, bool)` — **не ошибку**. Вызывающий код (service layer) сам решает, какую ошибку бросить (обычно `domain.ErrUnauthorized`). Это сохраняет `pkg/` чистым от зависимостей на `internal/domain/`.
- `RequestIDFromCtx` возвращает `""` без ошибки — request ID не обязателен
- Пакет `pkg/ctxutil/` **НЕ импортирует** ничего из `internal/`. Единственная внешняя зависимость — `github.com/google/uuid`.

**Acceptance criteria:**
- [ ] Все 4 функции реализованы
- [ ] `UserIDFromCtx` без userID в контексте → `uuid.Nil, false`
- [ ] `UserIDFromCtx` с uuid.Nil в контексте → `uuid.Nil, false`
- [ ] `UserIDFromCtx` с валидным UUID → `uuid, true`
- [ ] `RequestIDFromCtx` без requestID → пустая строка
- [ ] Ключи контекста — unexported типы
- [ ] Пакет не импортирует ничего из `internal/`
- [ ] Unit-тесты: WithUserID + UserIDFromCtx (happy path), UserIDFromCtx (empty ctx), UserIDFromCtx (nil UUID), WithRequestID + RequestIDFromCtx

**Corner cases:**
- `UserIDFromCtx` должен корректно обрабатывать случай, когда в контексте лежит значение другого типа (не uuid.UUID) — не паниковать, вернуть `uuid.Nil, false`
- Context helpers не должны зависеть от `internal/` — ни от `domain/`, ни от `config/`, ни от других пакетов

---

### TASK-1.5: Доменный слой

**Зависит от:** TASK-1.1

**Контекст:**
- `data_model_v4.md` — все секции (определяют поля каждой модели)
- `code_conventions_v4.md` — §1 (domain/ — чистые структуры), §2 (ошибки), §12 (naming)
- `repo_layer_tasks_v4.md` — TASK-004 (детальное описание всех типов)
- `services/study_service_spec_v4_v1.1.md` — дополнительные поля: `LearningStep` в Card, `PrevState` в ReviewLog

**Что сделать:**

Создать пакет `internal/domain/` со всеми типами, которые используются repository и service слоями. Это самый крупный и важный TASK этой фазы.

> **Примечание по размеру задачи:** Это 7+ файлов с ~20 типами + тесты. Если агент испытывает трудности — допустимо разбить на подзадачи: (1) enums + errors → (2) models → (3) normalize + методы.

**Файлы и их содержимое:**

#### `enums.go` — все enum типы

| Enum | Значения |
|------|---------|
| `LearningStatus` | NEW, LEARNING, REVIEW, MASTERED |
| `ReviewGrade` | AGAIN, HARD, GOOD, EASY |
| `PartOfSpeech` | NOUN, VERB, ADJECTIVE, ADVERB, PRONOUN, PREPOSITION, CONJUNCTION, INTERJECTION, PHRASE, IDIOM, OTHER |
| `EntityType` | ENTRY, SENSE, EXAMPLE, IMAGE, PRONUNCIATION, CARD, TOPIC |
| `AuditAction` | CREATE, UPDATE, DELETE |
| `OAuthProvider` | google, apple |

Каждый enum — `type XxxType string` с константами. Не iota.

**Обязательные методы для каждого enum:**
- `String() string` — для логирования через slog и отладки
- `IsValid() bool` — для валидации при маппинге из строки (sqlc возвращает строку из БД, её нужно проверить). Без `IsValid()` невалидный enum тихо проскочит через repo в service.

> **Примечание:** `EntityType` включает значение `TOPIC`, которое требует добавления в enum в БД (миграция будет в Фазе 2). См. `topic_service_spec_v4.md`.

#### Общие типы полей

| Поле | Тип в domain | Примечание |
|------|-------------|-----------|
| `SourceSlug` | `string` | Идентификатор источника данных (например, `"freedict"`, `"google_translate"`, `"user"`). Простая строка, не enum — набор источников расширяем. Встречается в Sense, Translation, Example и их ref-аналогах. |
| `Position` | `int` | Порядок элемента в списке. `INT NOT NULL DEFAULT 0` в БД. В domain — `int`. |
| Все `ID` поля | `uuid.UUID` | Не pointer — ID всегда присутствует после создания. |
| `CreatedAt`, `UpdatedAt` | `time.Time` | Не pointer — всегда заполнены. |
| `DeletedAt`, `RevokedAt`, `FinishedAt`, `AbandonedAt` | `*time.Time` | Nullable — nil означает "не произошло". |

#### `errors.go` — ошибки

**Sentinel errors:**
- `ErrNotFound` — ресурс не найден
- `ErrAlreadyExists` — дубликат (unique violation)
- `ErrValidation` — ошибка валидации
- `ErrUnauthorized` — не аутентифицирован
- `ErrForbidden` — нет прав
- `ErrConflict` — конфликт состояния

**Structured validation error:**
- `FieldError` — структура с полями `Field` и `Message`
- `ValidationError` — содержит slice `[]FieldError`
- `ValidationError.Error()` — человекочитаемое сообщение
- `ValidationError.Unwrap()` — возвращает `ErrValidation` (для `errors.Is`)
- `NewValidationError(field, message)` — конструктор для одного поля
- `NewValidationErrors([]FieldError)` — конструктор для нескольких полей

#### `user.go` — пользовательские модели

| Модель | Поля |
|--------|------|
| `User` | ID (uuid), Email, Name, AvatarURL (*string), OAuthProvider, OAuthID, CreatedAt, UpdatedAt |
| `UserSettings` | UserID (uuid), NewCardsPerDay (int), ReviewsPerDay (int), MaxIntervalDays (int), Timezone (string), UpdatedAt |
| `RefreshToken` | ID (uuid), UserID (uuid), TokenHash (string), ExpiresAt (time.Time), CreatedAt, RevokedAt (*time.Time) |

- `DefaultUserSettings(userID uuid.UUID) UserSettings` — фабрика с дефолтами: new cards 20, reviews 200, max interval 365, timezone "UTC"
- `RefreshToken.IsRevoked() bool` — `RevokedAt != nil`
- `RefreshToken.IsExpired(now time.Time) bool` — `ExpiresAt.Before(now)`

#### `reference.go` — модели каталога (Reference Catalog)

| Модель | Поля |
|--------|------|
| `RefEntry` | ID, Text, TextNormalized, CreatedAt + slices: Senses, Pronunciations, Images |
| `RefSense` | ID, RefEntryID, Definition, PartOfSpeech (*PartOfSpeech), CEFRLevel (*string), SourceSlug, Position, CreatedAt + slices: Translations, Examples |
| `RefTranslation` | ID, RefSenseID, Text, SourceSlug, Position |
| `RefExample` | ID, RefSenseID, Sentence, Translation (*string), SourceSlug, Position |
| `RefPronunciation` | ID, RefEntryID, Transcription (*string), AudioURL (*string), Region (*string), SourceSlug |
| `RefImage` | ID, RefEntryID, URL, Caption (*string), SourceSlug |

#### `entry.go` — модели пользовательского словаря

| Модель | Поля |
|--------|------|
| `Entry` | ID, UserID, RefEntryID (*uuid), Text, TextNormalized, Notes (*string), CreatedAt, UpdatedAt, DeletedAt (*time.Time) + slices: Senses, Pronunciations ([]RefPronunciation), CatalogImages ([]RefImage), UserImages, Card (*Card), Topics |
| `Sense` | ID, EntryID, RefSenseID (*uuid), Definition (*string), PartOfSpeech (*PartOfSpeech), CEFRLevel (*string), SourceSlug, Position, CreatedAt + slices: Translations, Examples |
| `Translation` | ID, SenseID, RefTranslationID (*uuid), Text (*string), SourceSlug, Position |
| `Example` | ID, SenseID, RefExampleID (*uuid), Sentence (*string), Translation (*string), SourceSlug, Position, CreatedAt |
| `UserImage` | ID, EntryID, URL, Caption (*string), CreatedAt |

- `Entry.IsDeleted() bool` — `DeletedAt != nil`

> **Nullable поля в Sense/Translation/Example:** поля с `*` могут быть nil, когда значение берётся из ref-каталога через COALESCE в SQL. Это ядро паттерна "Reference Catalog + User Dictionary".

#### `card.go` — модели карточек и SRS

| Модель | Поля |
|--------|------|
| `Card` | ID, UserID, EntryID, Status (LearningStatus), LearningStep (int), NextReviewAt (*time.Time), IntervalDays (int), EaseFactor (float64), CreatedAt, UpdatedAt |
| `ReviewLog` | ID, CardID, Grade (ReviewGrade), PrevState (*CardSnapshot), DurationMs (*int), ReviewedAt |
| `CardSnapshot` | Status (LearningStatus), LearningStep (int), IntervalDays (int), EaseFactor (float64), NextReviewAt (*time.Time) |
| `SRSResult` | Status (LearningStatus), LearningStep (int), NextReviewAt (time.Time), IntervalDays (int), EaseFactor (float64) |
| `StudySession` | ID, UserID, StartedAt, FinishedAt (*time.Time), CardsStudied (int), AbandonedAt (*time.Time) |

- `Card.IsDue(now time.Time) bool` — карточка требует повторения. Логика (из `study_service_spec_v4_v1.1.md` и `repo_layer_spec_v4.md`):
  - `Status == MASTERED` → `false` (mastered карточки не повторяются)
  - `Status == NEW` и `NextReviewAt == nil` → `true` (новые карточки всегда due)
  - `Status == LEARNING` или `REVIEW` → `NextReviewAt != nil && !NextReviewAt.After(now)` (due если время пришло)
- `SRSResult` — value object, результат вычисления SRS-алгоритма (чистая функция)
- `CardSnapshot` — snapshot состояния карточки перед review (для undo)

> **Важно:** `LearningStep` в Card и `PrevState` в ReviewLog — дополнения из `study_service_spec_v4_v1.1.md`. `StudySession` — новая таблица из той же спецификации. Миграции для этих полей создаются в Фазе 2.

> **CardSnapshot и JSONB:** `PrevState *CardSnapshot` в ReviewLog хранится как JSONB в БД. Поскольку domain запрещает теги (`json:""`, `db:""`), сериализация CardSnapshot ↔ JSONB — **ответственность repo layer** (custom marshaling в функциях маппинга). В domain `CardSnapshot` — обычная Go-структура без тегов.

#### `organization.go` — темы, inbox, аудит

| Модель | Поля |
|--------|------|
| `Topic` | ID, UserID, Name, Description (*string), CreatedAt, UpdatedAt + EntryCount (int, computed) |
| `InboxItem` | ID, UserID, Text, Context (*string), CreatedAt |
| `AuditRecord` | ID, UserID, EntityType, EntityID (*uuid), Action (AuditAction), Changes (map[string]any), CreatedAt |

> `Topic.EntryCount` — вычисляемое поле (COUNT из entry_topics). Не хранится в БД, заполняется repo/service при загрузке списка тем.

#### `normalize.go` — нормализация текста

Функция `NormalizeText(text string) string` с правилами:
1. Trim пробелы по краям
2. Lower case
3. Сжать множественные пробелы в один
4. **НЕ** убирать диакритику (café → café, не cafe)
5. **НЕ** убирать дефисы (well-known → well-known)
6. **НЕ** убирать апострофы (don't → don't)

Используется service-слоем перед передачей текста в repo.

**Acceptance criteria:**
- [ ] Все типы из `data_model_v4.md` представлены в domain
- [ ] Дополнительные типы из `study_service_spec_v4_v1.1.md` (LearningStep, PrevState/CardSnapshot, StudySession) включены
- [ ] Nullable поля: pointer types (`*string`, `*uuid.UUID`, `*time.Time`)
- [ ] Enums: string-based constants (не iota)
- [ ] Enum методы: `String()` и `IsValid()` для каждого enum типа
- [ ] `SourceSlug` — тип `string`, задокументирован
- [ ] `Position` — тип `int`
- [ ] Sentinel errors: все 6 определены
- [ ] `ValidationError` с `Unwrap() → ErrValidation`
- [ ] `NewValidationError`, `NewValidationErrors` — конструкторы
- [ ] `Entry.IsDeleted()` — unit-тест
- [ ] `Card.IsDue(now)` — unit-тесты:
  - MASTERED → false
  - NEW, NextReviewAt == nil → true
  - LEARNING, NextReviewAt в прошлом → true
  - REVIEW, NextReviewAt в будущем → false
- [ ] `RefreshToken.IsRevoked()` — unit-тест
- [ ] `RefreshToken.IsExpired(now)` — unit-тест
- [ ] `DefaultUserSettings(userID)` — unit-тест (проверка дефолтов)
- [ ] `NormalizeText` — unit-тесты на все правила:
  - trim пробелы
  - lower case
  - множественные пробелы → один
  - диакритика сохраняется
  - дефисы сохраняются
  - апострофы сохраняются
  - пустая строка → пустая строка
- [ ] Enum `IsValid()` — unit-тесты: валидное значение → true, невалидная строка → false
- [ ] `CardSnapshot` — без json/db тегов (сериализация в repo layer)
- [ ] Пакет `domain/` не импортирует ничего кроме stdlib (`errors`, `fmt`, `strings`, `time`, `unicode`)
- [ ] Пакет `domain/` не содержит тегов `db:""`, `json:""`, `gql:""`

**Corner cases:**
- `NormalizeText("")` → `""` (не паниковать)
- `NormalizeText("  ")` → `""` (только пробелы)
- `NormalizeText("Café")` → `"café"` (lower case, но диакритика сохранена)
- `Card.IsDue(now)` для `Status == MASTERED` → `false` (всегда, даже если NextReviewAt в прошлом)
- `Card.IsDue(now)` для `Status == NEW` и `NextReviewAt == nil` → `true`
- `Entry.IsDeleted()` для `DeletedAt == nil` → `false`
- `EntityType` должен включать `TOPIC` (добавление в enum в миграции — Фаза 2)
- `LearningStatus("INVALID").IsValid()` → `false`
- Все slice-поля в моделях (Senses, Translations и т.д.) — не nullable, инициализируются как `[]Type{}` при необходимости. Nil slice допустим в domain, но repo должен возвращать `[]` (это контролируется в repo layer)

---

### TASK-1.6: Точка входа приложения (stub)

**Зависит от:** TASK-1.2, TASK-1.3

**Контекст:**
- `infra_spec_v4.md` — §1 (одна точка входа), §10 (порядок инициализации, принципы)
- `code_conventions_v4.md` — §1 (cmd/server/main.go — только вызов bootstrap)

**Что сделать:**

Создать минимальный рабочий entry point, который:
- Настраивает signal-aware context (SIGINT, SIGTERM)
- Загружает конфиг
- Инициализирует логгер
- Логирует "starting application" + version + уровень лога
- Завершается (на этом этапе делать нечего — нет БД, нет сервера)

**Файлы:**

**`cmd/server/main.go`:**
- Создать signal-aware context: `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)` — это фундамент graceful shutdown. Даже если в Фазе 1 shutdown делать нечего, контракт функции должен быть правильным с самого начала.
- Вызвать `app.Run(ctx)`
- При ошибке → `log.Fatal`
- Никакой логики, никаких `init()`

**`internal/app/app.go`:**
- Функция `Run(ctx context.Context) error`
- Порядок: загрузить конфиг → создать логгер → залогировать старт (version, log level) → return nil
- На последующих фазах эта функция будет расширяться (подключение к БД, создание сервисов, запуск HTTP-сервера, ожидание ctx.Done())

**`internal/app/version.go`:**
- Переменные `Version`, `Commit`, `BuildTime` — заполняются через ldflags при сборке
- Дефолтные значения: `"dev"`, `"unknown"`, `"unknown"`
- Функция `BuildVersion() string` — для стартового лога и health endpoints

Для production-бинарника нужно уметь ответить на вопрос «какая версия запущена». Сборка через: `go build -ldflags "-X internal/app.Version=1.0.0 -X internal/app.Commit=$(git rev-parse HEAD)"`.

**Требования:**
- `main.go` содержит не более 15-20 строк (signal context + вызов app.Run)
- Вся инициализация — в `internal/app/`
- Signal handling настроен в main — ctx отменяется по SIGINT/SIGTERM
- Fail fast: ошибка конфига → не запускаемся
- Никаких глобальных переменных (кроме ldflags version vars)

**Acceptance criteria:**
- [ ] `go build ./cmd/server/` компилируется
- [ ] `go run ./cmd/server/` с валидным конфигом → логирует старт (version, log level) и завершается без ошибки
- [ ] `go run ./cmd/server/` без конфига → ошибка с понятным сообщением
- [ ] `main.go` создаёт signal-aware context (SIGINT, SIGTERM)
- [ ] `app.Run` принимает `context.Context` и использует `internal/config/` и logger
- [ ] Version/Commit/BuildTime — переменные для ldflags с дефолтами
- [ ] Нет глобальных переменных (кроме version ldflags), нет `init()`

---

## Сводка зависимостей задач

```
TASK-1.1 (Structure + Go module) ──┬──→ TASK-1.2 (Config)
                                   │         │
                                   │         └──→ TASK-1.3 (Logger) ──┐
                                   │                                   │
                                   ├──→ TASK-1.4 (Context Helpers)    │
                                   │                                   │
                                   ├──→ TASK-1.5 (Domain Layer)       │
                                   │                                   │
                                   └───────────────────────────────────┴──→ TASK-1.6 (Entry Point)
```

## Параллелизация

| Волна | Задачи |
|-------|--------|
| 1 | TASK-1.1 |
| 2 | TASK-1.2, TASK-1.4, TASK-1.5 (параллельно) |
| 3 | TASK-1.3 |
| 4 | TASK-1.6 |

> TASK-1.4 и TASK-1.5 зависят только от TASK-1.1, поэтому могут выполняться параллельно с TASK-1.2.
> TASK-1.6 требует TASK-1.2 и TASK-1.3.

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок (с `.golangci.yml`)
- [ ] Все acceptance criteria всех задач выполнены
- [ ] Пакет `domain/` содержит все модели из `data_model_v4.md` + дополнения из study_service v1.1
- [ ] Пакет `domain/` не имеет внешних зависимостей (только stdlib)
- [ ] Все enum типы имеют методы `String()` и `IsValid()`
- [ ] Пакет `pkg/ctxutil/` не импортирует ничего из `internal/`
- [ ] Конфигурация загружается из YAML + ENV, `Validate()` покрыт тестами
- [ ] Logger инициализируется из конфига
- [ ] `cmd/server/main.go` создаёт signal-aware context и вызывает `app.Run`
- [ ] Version info инъецируется через ldflags
- [ ] `tools.go` и `.golangci.yml` созданы
- [ ] Структура директорий соответствует `code_conventions_v4.md` §1
