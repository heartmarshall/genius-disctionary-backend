# Фаза 5: RefCatalog и Dictionary сервисы


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (интерфейсы потребителем), §2 (обработка ошибок), §3 (валидация), §4 (контекст и user identity), §5 (логирование), §6 (аудит), §7 (тестирование, moq) |
| `services/service_layer_spec_v4.md` | §2 (структура пакетов), §3 (паттерны), §4 (аудит: ENTRY), §5 (application-level limits), §6 (карта сервисов: RefCatalogService, DictionaryService), §7 (тестирование), §8 (Hard Delete Job) |
| `services/refcatalog_service_spec_v4.md` | Все секции — полная спецификация RefCatalog Service: зависимости, операции, маппинг, corner cases, тесты |
| `services/dictionary_service_spec_v4.md` | Все секции — полная спецификация Dictionary Service: зависимости, 13 операций, валидация, error scenarios, тесты |
| `services/business_scenarios_v4.md` | D1–D13 (Dictionary), B1/B4/B5 (Batch), R1–R4 (RefCatalog), BG1 (Hard Delete) |
| `data_model_v4.md` | §1 (Reference Catalog pattern), §2 (ref-таблицы), §4 (user dictionary), §10 (soft delete) |
| `repo/repo_layer_spec_v4.md` | §4 (sqlc vs Squirrel), §5 (COALESCE), §8 (soft delete), §9 (конкурентность), §10 (Reference Catalog management), §16 (batch operations) |

---

## Пре-условия (из Фазы 1)

Перед началом Фазы 5 должны быть готовы:

- Domain-модели: `RefEntry`, `RefSense`, `RefTranslation`, `RefExample`, `RefPronunciation`, `RefImage` (`internal/domain/reference.go`)
- Domain-модели: `Entry`, `Sense`, `Translation`, `Example`, `UserImage` (`internal/domain/entry.go`)
- Domain-модели: `Card` (`internal/domain/card.go`)
- Domain-модели: `AuditRecord`, `Topic` (`internal/domain/organization.go`)
- `Entry.IsDeleted()` — helper для проверки soft delete (`internal/domain/entry.go`)
- `domain.NormalizeText()` — нормализация текста: trim, lower, compress spaces (`internal/domain/normalize.go`)
- Domain errors: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized` (`internal/domain/errors.go`)
- `ValidationError`, `FieldError`, `NewValidationError()` (`internal/domain/errors.go`)
- Enums: `PartOfSpeech` (NOUN, VERB, ..., OTHER), `EntityType` (ENTRY, ...), `AuditAction` (CREATE, UPDATE, DELETE), `LearningStatus` (`internal/domain/enums.go`)
- Context helpers: `ctxutil.UserIDFromCtx(ctx) → (uuid.UUID, bool)` (`pkg/ctxutil/`)
- Config: root `Config` struct с `SRSConfig.DefaultEaseFactor` (`internal/config/`)

> **Важно:** Фаза 5 **не зависит** от Фаз 2, 3, 4. Все зависимости на репозитории, TxManager и RefCatalogService мокаются в unit-тестах. Сервисы можно разрабатывать параллельно с инфраструктурой БД, репозиториями и Auth.

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | FreeDictionary API | `https://api.dictionaryapi.dev/api/v2/entries/en/{word}` — бесплатный, без API-ключа |
| 2 | Переводы на MVP | Stub: `translationProvider` interface определён, реализация возвращает `nil, nil`. Google Translate — post-MVP |
| 3 | source_slug FreeDictionary | `"freedict"` |
| 4 | source_slug translations (будущий) | `"translate"` |
| 5 | source_slug custom entries | `"user"` |
| 6 | source_slug import | `"import"` |
| 7 | Concurrent ref_entry creation | INSERT ON CONFLICT DO NOTHING + SELECT existing. Допускается дублирование HTTP-вызовов к FreeDictionary — цена за простоту (без advisory lock) |
| 8 | Shared provider types | `internal/provider/result.go` — `DictionaryResult`, `SenseResult`, `ExampleResult`, `PronunciationResult`. Импортируется и service, и adapter |
| 9 | DictionaryConfig | Новый struct в `config.go` с `MaxEntriesPerUser`, `DefaultEaseFactor`, `ImportChunkSize`, `ExportMaxEntries`, `HardDeleteRetentionDays` |
| 10 | Hard Delete | Отдельная CLI-команда `cmd/cleanup/`. Вызывается внешним cron. Не горутина в main server |
| 11 | ExportEntries batch-загрузка | Через batch repo-методы (не DataLoaders — это не GraphQL context) |
| 12 | FreeDictionary multiple entries | API может вернуть массив entries (разные этимологии). Берём все meanings из всех entries, pronunciations объединяем с дедупликацией |
| 13 | Translation привязка | Все переводы от translationProvider привязываются к **первому** sense. Упрощение для MVP |
| 14 | mapPartOfSpeech fallback | Неизвестная part_of_speech → `domain.PartOfSpeechOther` |
| 15 | ErrWordNotFound | Service-level ошибка в `service/refcatalog/errors.go`. Не в `domain/errors.go` — специфична для RefCatalog |
| 16 | Моки | `moq` (code generation) — моки генерируются из приватных интерфейсов в `_test.go` файлы |
| 17 | Mock TxManager | `RunInTx(ctx, fn)` просто вызывает `fn(ctx)` без реальной транзакции |
| 18 | TopicName в ImportItem | На MVP игнорируется (поле зарезервировано для будущей интеграции с TopicService) |

---

## Задачи

### TASK-5.1: DictionaryConfig

**Зависит от:** Фаза 1 (config уже создан)

**Контекст:**
- `services/dictionary_service_spec_v4.md` — §2.4 (DictionaryConfig)
- `services/service_layer_spec_v4.md` — §5 (Application-Level Limits)
- Текущий `internal/config/config.go` — DictionaryConfig отсутствует

**Что сделать:**

Добавить `DictionaryConfig` в `internal/config/config.go` и подключить к root `Config`.

**Новый struct:**

```go
// DictionaryConfig holds dictionary service settings.
type DictionaryConfig struct {
    MaxEntriesPerUser      int `yaml:"max_entries_per_user"       env:"DICT_MAX_ENTRIES_PER_USER"      env-default:"10000"`
    DefaultEaseFactor      float64 `yaml:"default_ease_factor"    env:"DICT_DEFAULT_EASE_FACTOR"       env-default:"2.5"`
    ImportChunkSize        int `yaml:"import_chunk_size"          env:"DICT_IMPORT_CHUNK_SIZE"         env-default:"50"`
    ExportMaxEntries       int `yaml:"export_max_entries"         env:"DICT_EXPORT_MAX_ENTRIES"        env-default:"10000"`
    HardDeleteRetentionDays int `yaml:"hard_delete_retention_days" env:"DICT_HARD_DELETE_RETENTION_DAYS" env-default:"30"`
}
```

**Подключение к root Config:**

```go
type Config struct {
    Server     ServerConfig     `yaml:"server"`
    Database   DatabaseConfig   `yaml:"database"`
    Auth       AuthConfig       `yaml:"auth"`
    Dictionary DictionaryConfig `yaml:"dictionary"`  // NEW
    GraphQL    GraphQLConfig    `yaml:"graphql"`
    Log        LogConfig        `yaml:"log"`
    SRS        SRSConfig        `yaml:"srs"`
}
```

**Валидация (`validate.go`):**

```go
// Dictionary validation
if c.Dictionary.MaxEntriesPerUser <= 0 {
    return fmt.Errorf("dictionary.max_entries_per_user must be positive")
}
if c.Dictionary.DefaultEaseFactor < 1.0 || c.Dictionary.DefaultEaseFactor > 5.0 {
    return fmt.Errorf("dictionary.default_ease_factor must be between 1.0 and 5.0")
}
if c.Dictionary.ImportChunkSize <= 0 || c.Dictionary.ImportChunkSize > 1000 {
    return fmt.Errorf("dictionary.import_chunk_size must be between 1 and 1000")
}
if c.Dictionary.ExportMaxEntries <= 0 {
    return fmt.Errorf("dictionary.export_max_entries must be positive")
}
if c.Dictionary.HardDeleteRetentionDays <= 0 {
    return fmt.Errorf("dictionary.hard_delete_retention_days must be positive")
}
```

**Acceptance criteria:**
- [ ] `DictionaryConfig` struct создан с полями и env-тегами
- [ ] Подключён к root `Config` как `Dictionary DictionaryConfig`
- [ ] `yaml` теги: `dictionary.*`
- [ ] `env` теги: `DICT_*`
- [ ] Defaults: 10000, 2.5, 50, 10000, 30
- [ ] Валидация: все поля positive, ease_factor 1.0–5.0, chunk_size 1–1000
- [ ] Unit-тесты: defaults корректны, невалидные значения → error
- [ ] `go build ./...` компилируется

---

### TASK-5.2: Shared Provider Types + TranslationProvider Stub

**Зависит от:** ничего (параллельно с остальными)

**Контекст:**
- `services/refcatalog_service_spec_v4.md` — §2.4 (shared-типы провайдеров)
- `services/service_layer_spec_v4.md` — §6.3 (External Providers)

**Что сделать:**

Создать два пакета:
1. `internal/provider/` — shared типы, импортируемые и сервисом, и адаптерами
2. `internal/adapter/provider/translate/` — stub-реализация TranslationProvider

**Файловая структура:**

```
internal/provider/
└── result.go              # DictionaryResult, SenseResult, ExampleResult, PronunciationResult

internal/adapter/provider/translate/
├── stub.go                # Stub struct + FetchTranslations → nil, nil
└── stub_test.go           # Тривиальный тест: stub returns nil
```

**`internal/provider/result.go`:**

```go
package provider

// DictionaryResult is the structured result from a dictionary API provider.
type DictionaryResult struct {
    Word           string
    Senses         []SenseResult
    Pronunciations []PronunciationResult
}

// SenseResult represents a single word sense from an external dictionary.
type SenseResult struct {
    Definition   string
    PartOfSpeech *string
    Examples     []ExampleResult
}

// ExampleResult represents a usage example from an external dictionary.
type ExampleResult struct {
    Sentence    string
    Translation *string
}

// PronunciationResult represents pronunciation data from an external dictionary.
type PronunciationResult struct {
    Transcription *string
    AudioURL      *string
    Region        *string
}
```

**`internal/adapter/provider/translate/stub.go`:**

```go
package translate

import "context"

// Stub is a no-op translation provider for MVP.
// Returns nil (no translations available).
type Stub struct{}

func NewStub() *Stub { return &Stub{} }

// FetchTranslations always returns nil — no translations on MVP.
func (s *Stub) FetchTranslations(ctx context.Context, word string) ([]string, error) {
    return nil, nil
}
```

**Acceptance criteria:**
- [ ] `internal/provider/result.go` создан с DictionaryResult, SenseResult, ExampleResult, PronunciationResult
- [ ] Пакет `internal/provider/` не импортирует ничего из `internal/` (чистый utility-пакет)
- [ ] Stub создан в `internal/adapter/provider/translate/stub.go`
- [ ] `NewStub()` возвращает `*Stub`
- [ ] `FetchTranslations` возвращает `nil, nil`
- [ ] Unit-тест: stub returns nil, nil
- [ ] `go build ./...` компилируется

---

### TASK-5.3: FreeDictionary API Provider

**Зависит от:** TASK-5.2 (тип `provider.DictionaryResult`)

**Контекст:**
- `services/refcatalog_service_spec_v4.md` — §2.3 (dictionaryProvider interface)
- `services/service_layer_spec_v4.md` — §6.3 (External Providers: timeout 10s, retry 1 при 5xx)
- FreeDictionary API docs: `https://dictionaryapi.dev/`

**Что сделать:**

Создать пакет `internal/adapter/provider/freedict/` с реализацией dictionary provider для FreeDictionary API.

**Файловая структура:**

```
internal/adapter/provider/freedict/
├── provider.go        # Provider struct, FetchEntry, HTTP-запрос, парсинг
├── response.go        # Приватные struct для JSON-десериализации API-ответа
└── provider_test.go   # Unit-тесты с httptest
```

**`provider.go`:**

```go
type Provider struct {
    baseURL    string         // default: "https://api.dictionaryapi.dev/api/v2/entries/en"
    httpClient *http.Client
    log        *slog.Logger
}

func NewProvider(logger *slog.Logger) *Provider {
    return &Provider{
        baseURL:    "https://api.dictionaryapi.dev/api/v2/entries/en",
        httpClient: &http.Client{Timeout: 10 * time.Second},
        log:        logger.With("adapter", "freedict"),
    }
}

// NewProviderWithURL creates a provider with a custom base URL (for testing).
func NewProviderWithURL(baseURL string, logger *slog.Logger) *Provider
```

**Метод:**

```go
func (p *Provider) FetchEntry(ctx context.Context, word string) (*provider.DictionaryResult, error)
```

**Flow:**

```
1. url = p.baseURL + "/" + url.PathEscape(word)

2. req = http.NewRequestWithContext(ctx, "GET", url, nil)

3. resp, err = p.doWithRetry(ctx, req)
   ├─ err → return nil, fmt.Errorf("freedict: request failed: %w", err)
   └─ resp.StatusCode == 404 → return nil, nil  (слово не найдено)
       resp.StatusCode != 200 → return nil, fmt.Errorf("freedict: unexpected status %d", resp.StatusCode)

4. Десериализация JSON → []apiEntry

5. result = mapAPIResponse(apiEntries)
   // Объединить все entries (разные этимологии) в один DictionaryResult

6. return result, nil
```

**`response.go` — приватные JSON-структуры:**

```go
type apiEntry struct {
    Word      string        `json:"word"`
    Phonetics []apiPhonetic `json:"phonetics"`
    Meanings  []apiMeaning  `json:"meanings"`
}

type apiPhonetic struct {
    Text  string `json:"text"`
    Audio string `json:"audio"`
}

type apiMeaning struct {
    PartOfSpeech string          `json:"partOfSpeech"`
    Definitions  []apiDefinition `json:"definitions"`
}

type apiDefinition struct {
    Definition string `json:"definition"`
    Example    string `json:"example"`
}
```

**Маппинг FreeDictionary → provider.DictionaryResult:**

| FreeDictionary | DictionaryResult | Логика |
|----------------|-----------------|--------|
| `apiEntry.Word` | `Word` | Из первого entry |
| `apiMeaning.PartOfSpeech` + `apiDefinition` | `SenseResult` | Каждая definition → отдельный SenseResult. PartOfSpeech наследуется от parent meaning |
| `apiDefinition.Definition` | `SenseResult.Definition` | Как есть |
| `apiDefinition.Example` | `SenseResult.Examples[0]` | Если не пустой → один ExampleResult. Translation = nil |
| `apiPhonetic` | `PronunciationResult` | text → Transcription, audio → AudioURL |

**Объединение нескольких entries:**

FreeDictionary может вернуть массив entries (разные этимологии одного слова). Правила:
- Word: берётся из первого entry
- Senses: объединяются все meanings из всех entries (в порядке следования)
- Pronunciations: объединяются из всех entries, **дедупликация** по Transcription (если text совпадает — берём первый с audio)

**Region из audio URL:**

FreeDictionary часто включает регион в URL аудио (например, `...-us.mp3`, `...-uk.mp3`). Определение региона:

```go
func inferRegion(audioURL string) *string {
    lower := strings.ToLower(audioURL)
    if strings.Contains(lower, "-us.") || strings.Contains(lower, "-us-") {
        r := "US"
        return &r
    }
    if strings.Contains(lower, "-uk.") || strings.Contains(lower, "-uk-") {
        r := "UK"
        return &r
    }
    // AU, etc. — можно расширять
    return nil
}
```

**Retry-логика:**

```go
func (p *Provider) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
    resp, err := p.httpClient.Do(req)
    if err != nil || (resp != nil && resp.StatusCode >= 500) {
        // Retry один раз с backoff 500ms
        if ctx.Err() != nil {
            return resp, err  // context cancelled — не retry
        }
        if resp != nil && resp.Body != nil {
            resp.Body.Close()
        }
        time.Sleep(500 * time.Millisecond)
        resp, err = p.httpClient.Do(req)
    }
    return resp, err
}
```

Retry **только** при:
- Network error (timeout, connection refused)
- HTTP 5xx от FreeDictionary

**Не** retry при:
- HTTP 404 (слово не найдено) — ожидаемый результат
- HTTP 4xx — ошибка клиента

**Логирование:**
- DEBUG: "freedict request" word=...
- DEBUG: "freedict response" word=... status=... senses=N pronunciations=N
- WARN: "freedict retry" word=... reason=...
- ERROR: "freedict request failed" word=... error=...

**Corner cases:**

- **Пустой ответ (valid JSON, 0 entries):** API возвращает `[]` → mapAPIResponse возвращает DictionaryResult с пустыми Senses и Pronunciations. Сервис решает, что делать.
- **Word mismatch:** API вернул слово "abandon" когда запрошено "Abandon" → берём Word из ответа API (он каноничный).
- **Phonetics без text:** Запись phonetics есть, но `text` пустой → Transcription = nil (пропускаем пустые).
- **Phonetics без audio:** `audio` пустой → AudioURL = nil.
- **Definition без example:** `example` пустой → Examples = [] (пустой массив).
- **Encoding:** FreeDictionary возвращает UTF-8. Go's `json.Decoder` обрабатывает автоматически.
- **Large response:** Слово с множеством значений → может быть 20+ senses. Допустимо — никаких лимитов на provider level.

**Acceptance criteria:**
- [ ] `Provider` struct создан с конструктором
- [ ] `FetchEntry` выполняет GET-запрос к FreeDictionary API
- [ ] URL: `{baseURL}/{word}` с `url.PathEscape`
- [ ] HTTP timeout: 10 секунд
- [ ] Retry: 1 раз при 5xx/network error, backoff 500ms
- [ ] Не retry при 404 (word not found)
- [ ] 404 → return `nil, nil` (не ошибка)
- [ ] Non-200/404 → return error
- [ ] JSON десериализация → `[]apiEntry`
- [ ] Маппинг: каждая `apiDefinition` → `SenseResult` с inherited `PartOfSpeech`
- [ ] Маппинг: `apiDefinition.Example` → `ExampleResult` (если не пустой)
- [ ] Маппинг: `apiPhonetic` → `PronunciationResult` с inferred Region
- [ ] Объединение multiple entries: senses concatenated, pronunciations deduplicated
- [ ] Пустые phonetics (без text и audio) пропускаются
- [ ] `NewProviderWithURL` для тестов с кастомным base URL
- [ ] Секреты не логируются (нет секретов у FreeDictionary, но word логируется только в DEBUG)
- [ ] Unit-тесты с `httptest.NewServer`:
  - [ ] Success: полный ответ → correct DictionaryResult
  - [ ] Word not found (404) → nil, nil
  - [ ] Server error (500) → retry → success
  - [ ] Server error (500) → retry → 500 (обе попытки fail)
  - [ ] Invalid JSON → error
  - [ ] Multiple entries → merged result
  - [ ] Phonetics deduplication
  - [ ] Empty definitions → empty senses
  - [ ] Definition with example → ExampleResult created
  - [ ] Definition without example → no ExampleResult
  - [ ] Region inference from audio URL
- [ ] `go build ./...` компилируется

---

### TASK-5.4: RefCatalog Service

**Зависит от:** TASK-5.2 (shared provider types)

> **Примечание:** TASK-5.4 **не зависит** от TASK-5.3 (FreeDictionary adapter). RefCatalog service определяет свой приватный `dictionaryProvider` interface и использует тип `provider.DictionaryResult` из TASK-5.2. В тестах dictionaryProvider и translationProvider мокаются. Реальные адаптеры (TASK-5.3, TASK-5.2) нужны только при wiring в main.go.

**Контекст:**
- `services/refcatalog_service_spec_v4.md` — все секции: §2 (зависимости), §3 (операции), §4 (маппинг), §5 (ошибки), §6 (тесты)
- `services/service_layer_spec_v4.md` — §3 (паттерны), §7 (тестирование)
- `services/business_scenarios_v4.md` — R1–R4

**Что сделать:**

Создать пакет `internal/service/refcatalog/` с полной реализацией RefCatalog Service.

**Файловая структура:**

```
internal/service/refcatalog/
├── service.go           # Service struct, конструктор, приватные интерфейсы
│                        # Search, GetOrFetchEntry, GetRefEntry
├── mapper.go            # mapToRefEntry, mapPartOfSpeech (приватные)
├── errors.go            # ErrWordNotFound
└── service_test.go      # ~22 unit-теста
```

**`service.go` — приватные интерфейсы:**

```go
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
```

**Конструктор:**

```go
type Service struct {
    log         *slog.Logger
    refEntries  refEntryRepo
    tx          txManager
    dictProvider dictionaryProvider
    transProvider translationProvider
}

func NewService(
    logger              *slog.Logger,
    refEntries          refEntryRepo,
    tx                  txManager,
    dictProvider        dictionaryProvider,
    transProvider       translationProvider,
) *Service {
    return &Service{
        log:           logger.With("service", "refcatalog"),
        refEntries:    refEntries,
        tx:            tx,
        dictProvider:  dictProvider,
        transProvider: transProvider,
    }
}
```

**`errors.go`:**

```go
package refcatalog

import "errors"

// ErrWordNotFound indicates the word was not found by any external provider.
var ErrWordNotFound = errors.New("word not found in external provider")
```

---

**Операция: Search(ctx, query, limit) → ([]domain.RefEntry, error)**

Полный flow из refcatalog_service_spec_v4.md §3.1:

```
1. Clamp limit: min(max(limit, 1), 50), default 20
2. if query == "" → return [], nil
3. return s.refEntries.Search(ctx, query, limit)
```

Не требует userID — каталог shared.

---

**Операция: GetOrFetchEntry(ctx, text) → (\*domain.RefEntry, error)**

Полный flow из refcatalog_service_spec_v4.md §3.2:

```
1. normalized = domain.NormalizeText(text)
   └─ normalized == "" → return ValidationError("text", "required")

2. existing = refEntryRepo.GetFullTreeByText(ctx, normalized)
   └─ found → return existing

--- Внешние HTTP-вызовы (вне транзакции) ---

3. dictResult = dictionaryProvider.FetchEntry(ctx, text)
   ├─ err → log ERROR, return err
   └─ nil → return ErrWordNotFound

4. translations = translationProvider.FetchTranslations(ctx, text)
   └─ err → log WARN, translations = nil (graceful degradation)

5. refEntry = mapToRefEntry(normalized, dictResult, translations)

6. txManager.RunInTx → refEntryRepo.CreateWithTree(ctx, refEntry)
   └─ ErrAlreadyExists → GetFullTreeByText (concurrent create)

7. log INFO "ref entry fetched and saved"
8. return saved
```

**Ключевые свойства:**
- **Никаких HTTP-вызовов внутри транзакции** — внешние API вызваны на шаге 3–4, до начала tx на шаге 6.
- **Translation provider failure = graceful degradation** — entry сохраняется без переводов, пользователь добавит их вручную.
- **Concurrent-safe** — ErrAlreadyExists из CreateWithTree обрабатывается загрузкой existing записи.

---

**Операция: GetRefEntry(ctx, refEntryID) → (\*domain.RefEntry, error)**

```
1. refEntryRepo.GetFullTreeByID(ctx, refEntryID)
   └─ ErrNotFound → return ErrNotFound
2. return refEntry
```

---

**`mapper.go` — mapToRefEntry:**

Конвертирует `provider.DictionaryResult` + `[]string` (translations) → `domain.RefEntry` с полным деревом.

**Правила:**
- Все UUID генерируются через `uuid.New()`
- source_slug: `"freedict"` для senses/examples/pronunciations, `"translate"` для translations
- Positions: последовательные (0, 1, 2...) внутри каждого parent
- PartOfSpeech: `mapPartOfSpeech(raw *string)` — ToUpper → domain.PartOfSpeech, unknown → OTHER, nil → nil
- Translations привязываются к первому sense (если есть)
- Пустые senses/examples — допустимы (пустые массивы)

Полное описание маппинга: refcatalog_service_spec_v4.md §4.

---

**Unit-тесты (из refcatalog_service_spec §6.2):**

Все зависимости мокаются через moq. Mock TxManager: `fn(ctx)` pass-through.

**Search:**

| # | Тест | Assert |
|---|------|--------|
| S1 | Пустой query | return [], Search NOT called |
| S2 | Нормальный query | Search called, results returned |
| S3 | Limit clamped to max | limit=999 → limit=50 |
| S4 | Limit clamped to min | limit=0 → limit=1 |

**GetOrFetchEntry:**

| # | Тест | Assert |
|---|------|--------|
| GF1 | Слово в каталоге | GetFullTreeByText → existing, providers NOT called |
| GF2 | Fetch success, без переводов | FetchEntry → result, FetchTranslations → nil, CreateWithTree called |
| GF3 | Fetch success, с переводами | Translations привязаны к первому sense |
| GF4 | Слово не найдено (nil result) | ErrWordNotFound |
| GF5 | Dictionary provider error | Error пробрасывается, ERROR logged |
| GF6 | Translation provider error | Entry saved без переводов, WARN logged |
| GF7 | Concurrent create (ErrAlreadyExists) | GetFullTreeByText called, existing returned |
| GF8 | Пустой text | ValidationError |
| GF9 | Text из пробелов | ValidationError |
| GF10 | Provider returns no senses | RefEntry created without senses |
| GF11 | Senses без examples | RefEntry created, senses без examples |
| GF12 | Переводы есть, senses пустой | Переводы игнорируются |

**GetRefEntry:**

| # | Тест | Assert |
|---|------|--------|
| GR1 | Найден | RefEntry with full tree returned |
| GR2 | Не найден | ErrNotFound |

**mapToRefEntry (table-driven):**

| # | Тест | Assert |
|---|------|--------|
| M1 | Full result | Все поля корректно замаплены |
| M2 | Без переводов | Senses без translations |
| M3 | С переводами | Translations привязаны к первому sense |
| M4 | Без pronunciations | Пустой массив |
| M5 | Множественные senses | Positions 0, 1, 2... |
| M6 | Уникальность UUID | Все ID в дереве уникальны |

**mapPartOfSpeech (table-driven):**

| # | Тест | Assert |
|---|------|--------|
| P1 | "noun" → NOUN | Correct |
| P2 | "verb" → VERB | Correct |
| P3 | "unknown" → OTHER | Fallback |
| P4 | nil → nil | Passthrough |

**Всего: ~22 тест-кейса**

**Acceptance criteria:**
- [ ] Service struct с приватными интерфейсами (refEntryRepo, txManager, dictionaryProvider, translationProvider)
- [ ] Конструктор `NewService` с логгером `"service", "refcatalog"`
- [ ] **Search:** clamp limit, пустой query → [], делегация в repo
- [ ] **GetOrFetchEntry:** полный flow — check DB → fetch → save → handle conflict
- [ ] **GetOrFetchEntry:** никаких HTTP-вызовов внутри транзакции
- [ ] **GetOrFetchEntry:** translation provider failure → graceful degradation (WARN, не ERROR)
- [ ] **GetOrFetchEntry:** concurrent create → ErrAlreadyExists → загрузка existing
- [ ] **GetRefEntry:** делегация в repo с пробросом ErrNotFound
- [ ] `ErrWordNotFound` определён в `errors.go`
- [ ] `mapToRefEntry`: корректный маппинг всех полей, UUID генерация, positions
- [ ] `mapPartOfSpeech`: ToUpper → valid → return, unknown → OTHER, nil → nil
- [ ] Translations привязываются к первому sense
- [ ] ~22 unit-теста покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/refcatalog/...` — все проходят
- [ ] `go vet ./internal/service/refcatalog/...` — без warnings

---

### TASK-5.5: Dictionary Service

**Зависит от:** TASK-5.1 (DictionaryConfig)

> **Примечание:** TASK-5.5 **не зависит** от TASK-5.4 (RefCatalog Service implementation). Dictionary Service определяет свой приватный `refCatalogService` interface и мокает его в тестах. Реальный RefCatalog Service подключается только при wiring в main.go.

**Контекст:**
- `services/dictionary_service_spec_v4.md` — все секции: §2 (зависимости), §3 (13 операций), §4 (валидация), §5 (error scenarios), §6 (тесты), §7 (файловая структура), §8 (взаимодействие)
- `services/service_layer_spec_v4.md` — §3 (паттерны), §4 (аудит: ENTRY), §5 (limits: 10000 entries)
- `data_model_v4.md` — §4 (entries, senses, translations, examples)

**Что сделать:**

Создать пакет `internal/service/dictionary/` с полной реализацией Dictionary Service. Это **самый большой сервис** приложения — 13 операций.

**Файловая структура:**

```
internal/service/dictionary/
├── service.go           # Service struct, конструктор, приватные интерфейсы
│                        # Все 13 операций
├── input.go             # CreateFromCatalogInput, CreateCustomInput, FindInput,
│                        # UpdateNotesInput, ImportInput + Validate()
├── result.go            # FindResult, BatchResult, ImportResult, ExportResult,
│                        # ExportItem, ExportSense, ExportExample, PageInfo
└── service_test.go      # Все тесты (~60 тестов)
```

**`service.go` — приватные интерфейсы:**

```go
type entryRepo interface {
    GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
    GetByText(ctx context.Context, userID uuid.UUID, textNormalized string) (*domain.Entry, error)
    GetByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error)
    Find(ctx context.Context, userID uuid.UUID, filter EntryFilter) ([]domain.Entry, int, error)
    FindCursor(ctx context.Context, userID uuid.UUID, filter EntryFilter) ([]domain.Entry, bool, error)
    FindDeleted(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Entry, int, error)
    CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
    Create(ctx context.Context, entry *domain.Entry) (*domain.Entry, error)
    UpdateNotes(ctx context.Context, userID, entryID uuid.UUID, notes *string) (*domain.Entry, error)
    SoftDelete(ctx context.Context, userID, entryID uuid.UUID) error
    Restore(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
    HardDeleteOld(ctx context.Context, threshold time.Time) (int, error)
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
    Create(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type refCatalogService interface {
    GetOrFetchEntry(ctx context.Context, text string) (*domain.RefEntry, error)
    GetRefEntry(ctx context.Context, refEntryID uuid.UUID) (*domain.RefEntry, error)
    Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
}
```

**Конструктор:**

```go
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
    cfg            config.DictionaryConfig
}

func NewService(
    logger            *slog.Logger,
    entries           entryRepo,
    senses            senseRepo,
    translations      translationRepo,
    examples          exampleRepo,
    pronunciations    pronunciationRepo,
    images            imageRepo,
    cards             cardRepo,
    audit             auditRepo,
    tx                txManager,
    refCatalog        refCatalogService,
    cfg               config.DictionaryConfig,
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
```

---

**Операции:**

Все 13 операций описаны детально в `dictionary_service_spec_v4.md` §3. Ниже — краткая сводка с ключевыми implementation notes.

| # | Метод | Spec §3.X | Описание | Транзакция | Аудит |
|---|-------|-----------|----------|------------|-------|
| 1 | `SearchCatalog` | §3.1 | Fuzzy search в каталоге → autocomplete | Нет | Нет |
| 2 | `PreviewRefEntry` | §3.2 | Загрузка полного дерева ref_entry | Нет | Нет |
| 3 | `CreateEntryFromCatalog` | §3.3 | Добавление слова из каталога с выбранными senses | Да | CREATE |
| 4 | `CreateEntryCustom` | §3.4 | Добавление слова вручную | Да | CREATE |
| 5 | `FindEntries` | §3.5 | Поиск/фильтрация с offset и cursor pagination | Нет | Нет |
| 6 | `GetEntry` | §3.6 | Получение entry по ID | Нет | Нет |
| 7 | `UpdateNotes` | §3.7 | Обновление заметок | Да | UPDATE |
| 8 | `DeleteEntry` | §3.8 | Soft delete | Да | DELETE |
| 9 | `FindDeletedEntries` | §3.9 | Просмотр корзины | Нет | Нет |
| 10 | `RestoreEntry` | §3.10 | Восстановление из корзины | Нет | Нет |
| 11 | `BatchDeleteEntries` | §3.11 | Массовый soft delete | Нет* | Да (batch) |
| 12 | `ImportEntries` | §3.12 | Массовый импорт из файла | Да (per chunk) | Нет |
| 13 | `ExportEntries` | §3.13 | Экспорт словаря | Нет | Нет |

*BatchDelete — каждый SoftDelete отдельно, partial failure допустим.

---

**Ключевые implementation notes (дополнения к spec):**

**1. SearchCatalog и PreviewRefEntry** — делегация в refCatalogService:

```
SearchCatalog:
1. userID check
2. refCatalog.Search(ctx, query, limit)

PreviewRefEntry:
1. userID check
2. refCatalog.GetOrFetchEntry(ctx, text)
```

**2. CreateEntryFromCatalog (§3.3)** — самая сложная операция:

Все чтения (шаги 2–7) — вне транзакции. Транзакция начинается на шаге 8.
Внутри транзакции: Create entry → CreateFromRef senses → CreateFromRef translations/examples → Link pronunciations → Link images → Create card (optional) → Audit.

Важно: `refCatalog.GetRefEntry` вызывается **до** транзакции (шаг 3). HTTP-вызовов нет — ref_entry уже в БД.

**3. CreateEntryCustom (§3.4):**

Аналогичная структура, но без ref-ссылок. Все записи с `source_slug = "user"`.

**4. FindEntries (§3.5)** — два режима пагинации:

```go
// EntryFilter — маппинг FindInput → repo filter (приватный тип)
type EntryFilter struct {
    Search      *string
    HasCard     *bool
    PartOfSpeech *domain.PartOfSpeech
    TopicID     *uuid.UUID
    Status      *domain.LearningStatus
    SortBy      string
    SortOrder   string
    Limit       int
    Cursor      *string
    Offset      *int
}
```

Cursor mode приоритетнее Offset mode (если заданы оба).

**5. BatchDeleteEntries (§3.11):**

**Не** в транзакции. Каждый SoftDelete — отдельная операция. Partial failure допустим.

**6. ImportEntries (§3.12):**

Chunks по `config.ImportChunkSize` (default 50). Каждый chunk — отдельная транзакция.
Ошибка в chunk → rollback chunk, продолжить со следующим.

**7. ExportEntries (§3.13):**

Batch-загрузка через repo batch-методы (не DataLoaders — не GraphQL context):
```
entryIDs = extractIDs(entries)
sensesMap = senseRepo.GetByEntryIDs(ctx, entryIDs)
senseIDs = extractSenseIDs(sensesMap)
translationsMap = translationRepo.GetBySenseIDs(ctx, senseIDs)
examplesMap = exampleRepo.GetBySenseIDs(ctx, senseIDs)
cardsMap = cardRepo.GetByEntryIDs(ctx, entryIDs)
```

---

**`input.go` — Input-структуры и валидация:**

Все input-структуры и правила валидации описаны в `dictionary_service_spec_v4.md` §4.

| Input | Validate() | Ключевые правила |
|-------|-----------|-----------------|
| `CreateFromCatalogInput` | §4.1 | RefEntryID required, SenseIDs max 20, Notes max 5000 |
| `CreateCustomInput` | §4.2 | Text required (max 500), Senses max 20, nested validation |
| `FindInput` | §4.3 | SortBy in ["text", "created_at", "updated_at"], SortOrder in ["ASC", "DESC"] |
| `UpdateNotesInput` | §4.4 | EntryID required, Notes max 5000 |
| `ImportInput` | §4.5 | Items required (1–5000), text required/max 500, translations max 20 |

Каждый `Validate()` собирает **все** ошибки (не fail-fast). Возвращает `*domain.ValidationError`.

**`result.go` — Result-структуры:**

```go
type FindResult struct {
    Entries     []domain.Entry
    TotalCount  int        // только для offset mode
    HasNextPage bool       // только для cursor mode
    PageInfo    *PageInfo  // для GraphQL
}

type PageInfo struct {
    StartCursor *string
    EndCursor   *string
}

type BatchResult struct {
    Deleted int
    Errors  []BatchError
}

type BatchError struct {
    EntryID uuid.UUID
    Error   string
}

type ImportResult struct {
    Imported int
    Skipped  int
    Errors   []ImportError
}

type ImportError struct {
    LineNumber int
    Text       string
    Reason     string
}

type ExportResult struct {
    Items      []ExportItem
    ExportedAt time.Time
}

type ExportItem struct {
    Text       string
    Notes      *string
    Senses     []ExportSense
    CardStatus *domain.LearningStatus
    CreatedAt  time.Time
}

type ExportSense struct {
    Definition   *string
    PartOfSpeech *domain.PartOfSpeech
    Translations []string
    Examples     []ExportExample
}

type ExportExample struct {
    Sentence    string
    Translation *string
}
```

---

**Unit-тесты (из dictionary_service_spec §6.2):**

Все зависимости мокаются. Mock TxManager: `fn(ctx)` pass-through. Полная таблица тестов — в spec §6.2.

**Сводка тест-кейсов:**

| Группа | Кол-во | Покрытие |
|--------|--------|----------|
| SearchCatalog | 4 | SC1–SC4: пустой query, нормальный, limit clamp, no auth |
| PreviewRefEntry | 3 | PR1–PR3: в каталоге, fetch ok, API error |
| CreateFromCatalog | 15 | CF1–CF15: happy path, selected senses, card, notes, duplicates, limit, invalid sense ID, audit |
| CreateCustom | 9 | CC1–CC9: happy path, empty senses, normalize, duplicate, validation |
| FindEntries | 11 | FE1–FE11: no filters, search, hasCard, topic, cursor, limit clamp, default sort |
| GetEntry | 3 | GE1–GE3: found, not found, no auth |
| UpdateNotes | 4 | UN1–UN4: set, clear, not found, validation |
| DeleteEntry | 3 | DE1–DE3: happy path, not found, no auth |
| FindDeletedEntries | 3 | FD1–FD3: entries, empty, limit clamp |
| RestoreEntry | 3 | RE1–RE3: happy path, not found, text conflict |
| BatchDeleteEntries | 5 | BD1–BD5: all ok, partial, empty, too many, audit |
| ImportEntries | 7 | IM1–IM7: happy, duplicate in file, existing, empty text, chunk fail, limit, empty items |
| ExportEntries | 3 | EX1–EX3: happy path, empty, COALESCE applied |

**Всего: ~73 тест-кейса** (spec указывает ~60, с учётом edge cases — ~73)

**Acceptance criteria:**
- [ ] Service struct с 10 приватными интерфейсами зависимостей + refCatalogService
- [ ] Конструктор `NewService` с 12 параметрами, логгер `"service", "dictionary"`
- [ ] **SearchCatalog:** делегация в refCatalog.Search, userID check, limit clamp
- [ ] **PreviewRefEntry:** делегация в refCatalog.GetOrFetchEntry, userID check
- [ ] **CreateEntryFromCatalog:** полный flow — validate → check ref_entry → check limit → check duplicate → tx(create + senses + translations + examples + pronunciations + images + card + audit)
- [ ] **CreateEntryFromCatalog:** SenseIDs пустой → все senses из ref
- [ ] **CreateEntryFromCatalog:** невалидный SenseID → ValidationError
- [ ] **CreateEntryFromCatalog:** concurrent create → ErrAlreadyExists
- [ ] **CreateEntryCustom:** полный flow — validate → check limit → check duplicate → tx(create + senses + translations + examples + card + audit)
- [ ] **CreateEntryCustom:** source_slug = "user"
- [ ] **FindEntries:** offset mode (Find) и cursor mode (FindCursor)
- [ ] **FindEntries:** cursor приоритетнее offset
- [ ] **FindEntries:** search нормализация → пустой после нормализации → nil
- [ ] **GetEntry:** делегация в entryRepo.GetByID с userID
- [ ] **UpdateNotes:** в транзакции с audit (old/new changes)
- [ ] **DeleteEntry:** soft delete в транзакции с audit
- [ ] **FindDeletedEntries:** deleted_at IS NOT NULL, sorted by deleted_at DESC
- [ ] **RestoreEntry:** optimistic restore, ErrAlreadyExists → ValidationError "active entry exists"
- [ ] **BatchDeleteEntries:** без транзакции, partial failure допустим, BatchResult
- [ ] **ImportEntries:** chunks по config.ImportChunkSize, per-chunk transactions, partial failure
- [ ] **ImportEntries:** дубликаты внутри файла → skip, source_slug = "import"
- [ ] **ExportEntries:** batch-загрузка через repo batch-методы, COALESCE resolved
- [ ] Все input-структуры с `Validate()` — собирают все ошибки
- [ ] Все операции с userID: `ErrUnauthorized` при отсутствии
- [ ] Аудит: CREATE для создания, UPDATE для notes, DELETE для soft delete
- [ ] Логирование: INFO для create/delete/restore/import, WARN для limit reached
- [ ] ~60–73 unit-теста покрывают все сценарии из spec §6.2
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/dictionary/...` — все проходят
- [ ] `go vet ./internal/service/dictionary/...` — без warnings

---

### TASK-5.6: Hard Delete CLI-команда

**Зависит от:** ничего (параллельно с остальными, использует repo interface)

**Контекст:**
- `services/service_layer_spec_v4.md` — §8 (Hard Delete Job)
- `services/business_scenarios_v4.md` — BG1 (Hard delete entries)
- `repo/repo_layer_spec_v4.md` — §8.3 (Hard delete), §16.3 (batch cleanup)
- `data_model_v4.md` — §10 (Soft Delete, Hard Delete)

**Что сделать:**

Создать CLI-команду `cmd/cleanup/` для физического удаления soft-deleted entries старше N дней. Запускается внешним cron (не горутина в основном сервере).

**Файловая структура:**

```
cmd/cleanup/
└── main.go              # Entry point: config → DB → cleanup → exit
```

**`main.go` — Flow:**

```
1. Загрузить конфигурацию (config.MustLoad())
2. Инициализировать логгер (slog)
3. Подключиться к БД (postgres.NewPool)
4. Создать экземпляр entryRepo

5. threshold = time.Now().AddDate(0, 0, -cfg.Dictionary.HardDeleteRetentionDays)

6. deleted, err = entryRepo.HardDeleteOld(ctx, threshold)
   ├─ err → log ERROR "hard delete failed", os.Exit(1)
   └─ nil → log INFO "hard delete completed" deleted=N threshold=...

7. Закрыть pool
8. os.Exit(0)
```

**Свойства:**
- **Не является сервисом** — не проходит через service layer. Вызывает repo напрямую (как описано в service_layer_spec §8).
- **Idempotent** — повторный запуск удаляет только записи, ещё не удалённые.
- **CASCADE** — физическое удаление entries каскадно удаляет все дочерние записи (senses, translations, examples, cards, review_logs и т.д.) через FK constraints.
- **Timeout:** Context с timeout 5 минут (для больших объёмов).
- **Exit codes:** 0 = success, 1 = error.

**Зависимости runtime:**
- `internal/config/` — загрузка конфигурации
- `internal/adapter/postgres/` — pool, entryRepo (из Phase 2 и 3)
- `pkg/ctxutil/` — не нужен (нет userID)

> **Примечание:** TASK-5.6 определяет структуру CLI-команды и её flow. Фактическая компиляция и запуск возможны только после завершения Phase 2 (pool) и Phase 3 (entryRepo). До этого — code skeleton без возможности запуска.

**Acceptance criteria:**
- [ ] `cmd/cleanup/main.go` создан
- [ ] Загрузка конфигурации через `config.MustLoad()`
- [ ] Инициализация логгера
- [ ] Подключение к БД через `postgres.NewPool`
- [ ] Вычисление threshold: `now - HardDeleteRetentionDays`
- [ ] Вызов `entryRepo.HardDeleteOld(ctx, threshold)`
- [ ] Логирование: INFO при успехе с количеством удалённых, ERROR при ошибке
- [ ] Graceful: pool.Close() в defer
- [ ] Context с timeout 5 минут
- [ ] Exit code: 0 success, 1 error
- [ ] `go build ./cmd/cleanup/...` компилируется (после Phase 2–3)

---

## Сводка зависимостей задач

```
TASK-5.1 (DictionaryConfig) ──────────────→ TASK-5.5 (Dictionary Service)

TASK-5.2 (Provider Types + Stub) ──┬──→ TASK-5.3 (FreeDictionary)
                                    └──→ TASK-5.4 (RefCatalog Service)

TASK-5.6 (Hard Delete CLI) ────────────── (standalone)
```

Детализация:
- **TASK-5.5** (Dictionary Service) зависит от: TASK-5.1 (DictionaryConfig). **Не зависит** от TASK-5.4 — refCatalogService мокается в тестах
- **TASK-5.4** (RefCatalog Service) зависит от: TASK-5.2 (provider types). **Не зависит** от TASK-5.3 — dictionaryProvider мокается
- **TASK-5.3** (FreeDictionary) зависит от: TASK-5.2 (provider.DictionaryResult type)
- **TASK-5.6** (Hard Delete CLI) не зависит от других задач фазы (использует repo interfaces из Phase 3)
- TASK-5.1, TASK-5.2, TASK-5.6 не имеют взаимных зависимостей

---

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-5.1 (DictionaryConfig), TASK-5.2 (Provider Types + Stub), TASK-5.6 (Hard Delete CLI) |
| 2 | TASK-5.3 (FreeDictionary), TASK-5.4 (RefCatalog Service), TASK-5.5 (Dictionary Service) |

> При полной параллелизации — **2 sequential волны**. Волна 2 — до 3 задач параллельно.
> TASK-5.5 не зависит от TASK-5.3 и TASK-5.4 (все зависимости мокаются) и может начинаться сразу после TASK-5.1.

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] **DictionaryConfig:** добавлен в config с валидацией и defaults
- [ ] **Provider Types:** `internal/provider/result.go` создан с DictionaryResult и связанными типами
- [ ] **Translation Stub:** `internal/adapter/provider/translate/stub.go` — FetchTranslations → nil, nil
- [ ] **FreeDictionary Provider:**
  - [ ] HTTP GET к `api.dictionaryapi.dev`
  - [ ] Парсинг JSON → DictionaryResult
  - [ ] 404 → nil, nil (слово не найдено)
  - [ ] Retry при 5xx, timeout 10s
  - [ ] Multiple entries → merged senses + deduplicated pronunciations
  - [ ] Region inference из audio URL
- [ ] **RefCatalog Service** — все 3 операции реализованы:
  - [ ] Search: fuzzy search, limit clamp
  - [ ] GetOrFetchEntry: DB check → provider fetch → save → concurrent-safe
  - [ ] GetRefEntry: load full tree by ID
- [ ] **RefCatalog Service** — маппинг:
  - [ ] mapToRefEntry: UUID generation, positions, source_slug
  - [ ] mapPartOfSpeech: valid → mapped, unknown → OTHER, nil → nil
  - [ ] Translations привязаны к первому sense
- [ ] **RefCatalog Service** — ~22 unit-теста проходят
- [ ] **Dictionary Service** — все 13 операций реализованы:
  - [ ] SearchCatalog, PreviewRefEntry — делегация в RefCatalog
  - [ ] CreateEntryFromCatalog — из каталога с выбором senses
  - [ ] CreateEntryCustom — ручное создание
  - [ ] FindEntries — offset и cursor pagination
  - [ ] GetEntry, UpdateNotes, DeleteEntry, RestoreEntry
  - [ ] FindDeletedEntries — корзина
  - [ ] BatchDeleteEntries — массовый soft delete
  - [ ] ImportEntries — chunks, partial failure
  - [ ] ExportEntries — batch-загрузка
- [ ] **Dictionary Service** — ~60–73 unit-теста проходят
- [ ] **Hard Delete CLI** — `cmd/cleanup/main.go` создан
- [ ] Все input-структуры с `Validate()` — собирают все ошибки
- [ ] Аудит: ENTRY CREATE/UPDATE/DELETE в транзакциях
- [ ] Логирование соответствует спецификации (INFO/WARN/ERROR)
- [ ] Моки сгенерированы через `moq` из приватных интерфейсов
- [ ] Все acceptance criteria всех 6 задач выполнены
