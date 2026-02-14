# MyEnglish Backend v4 — RefCatalog Service Specification

> **Статус:** Draft v1.0
> **Дата:** 2026-02-14
> **Зависимости:** code_conventions_v4.md (секции 1–4, 5, 7), data_model_v4.md (секция 2), repo_layer_spec_v4.md (секции 4, 9, 10, 19.1), service_layer_spec_v4.md (секции 3, 6, 7), business_scenarios_v4.md (R1–R4)

---

## 1. Ответственность

RefCatalog Service управляет **Reference Catalog** — shared immutable хранилищем словарных данных из внешних API. Отвечает за:

- Fuzzy search по каталогу (pg_trgm)
- Загрузку словарных данных из внешних API (FreeDictionary, будущие провайдеры)
- Сохранение загруженных данных в ref-таблицы (ref_entries, ref_senses, ref_translations, ref_examples, ref_pronunciations)
- Выдачу полного дерева ref_entry (entry → senses → translations, examples; pronunciations; images)
- Concurrent-safe upsert (два пользователя ищут одно слово одновременно → без дубликатов и ошибок)

RefCatalog Service **не** отвечает за:
- CRUD пользовательских entries (DictionaryService)
- Выбор senses/translations/examples при добавлении слова (DictionaryService)
- Управление дочерним контентом (ContentService)
- Отображение данных клиенту (Transport Layer)

---

## 2. Зависимости

### 2.1. Интерфейсы репозиториев

```
refEntryRepo interface {
    Search(ctx context.Context, query string, limit int) → ([]domain.RefEntry, error)
        // Fuzzy search по text_normalized (pg_trgm). Возвращает RefEntry БЕЗ дочернего дерева (только id, text, text_normalized).
    GetFullTreeByID(ctx context.Context, id uuid.UUID) → (*domain.RefEntry, error)
        // Загрузка ref_entry с полным деревом: senses → translations, examples; pronunciations; images.
    GetFullTreeByText(ctx context.Context, textNormalized string) → (*domain.RefEntry, error)
        // Аналогично GetFullTreeByID, но поиск по text_normalized.
    CreateWithTree(ctx context.Context, entry *domain.RefEntry) → (*domain.RefEntry, error)
        // Создание ref_entry со всем дочерним деревом. INSERT ON CONFLICT DO NOTHING для ref_entry.
        // Если entry уже существует (conflict) → return domain.ErrAlreadyExists.
        // Если создан успешно → return entry с сгенерированными ID.
}
```

### 2.2. Интерфейс TxManager

```
txManager interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) → error
}
```

### 2.3. Интерфейсы внешних провайдеров

```
dictionaryProvider interface {
    FetchEntry(ctx context.Context, word string) → (*provider.DictionaryResult, error)
        // Загрузка словарных данных из внешнего API.
        // Возвращает nil, nil если слово не найдено.
        // Возвращает nil, error при ошибке API.
}

translationProvider interface {
    FetchTranslations(ctx context.Context, word string) → ([]string, error)
        // Загрузка переводов слова.
        // Возвращает nil, nil если переводы недоступны (в т.ч. stub).
        // Возвращает nil, error при ошибке API.
}
```

### 2.4. Shared-типы провайдеров

Живут в `internal/provider/` — shared пакет, импортируется и сервисом, и адаптерами (аналогично `internal/auth/identity.go` для OAuthIdentity).

```
// DictionaryResult — результат от dictionary API провайдера.
type DictionaryResult struct {
    Word           string
    Senses         []SenseResult
    Pronunciations []PronunciationResult
}

type SenseResult struct {
    Definition   string
    PartOfSpeech *string   // raw строка от API, сервис маппит в domain.PartOfSpeech
    Examples     []ExampleResult
}

type ExampleResult struct {
    Sentence    string
    Translation *string
}

type PronunciationResult struct {
    Transcription *string
    AudioURL      *string
    Region        *string   // "US", "UK" и т.д.
}
```

### 2.5. Конструктор

```
func NewService(
    logger              *slog.Logger,
    refEntryRepo        refEntryRepo,
    txManager           txManager,
    dictionaryProvider  dictionaryProvider,
    translationProvider translationProvider,
) *Service
```

---

## 3. Бизнес-сценарии и операции

### 3.1. Search (R1)

**Сценарий:** Пользователь вводит текст в поле поиска → fuzzy search по ref_entries → список совпадений для autocomplete.

**Метод:** `Search(ctx context.Context, query string, limit int) → ([]domain.RefEntry, error)`

**Flow:**

```
1. Clamp limit: min(max(limit, 1), 50), default 20

2. if query == "" → return [], nil  (пустой запрос — пустой результат)

3. refEntryRepo.Search(ctx, query, limit)
   └─ Возвращает RefEntry без дочернего дерева — только id, text, text_normalized

4. return results, nil
```

Метод **не требует userID** — каталог shared. Авторизация проверяется вызывающим сервисом (DictionaryService.SearchCatalog).

**Corner cases:**
- Пустой query → пустой результат без обращения к БД.
- Query из одного символа → допустим, pg_trgm сам определяет релевантность.
- Каталог пуст → пустой результат, не ошибка.
- Search возвращает RefEntry **без** дочерних деревьев — для autocomplete достаточно текста.

---

### 3.2. GetOrFetchEntry (R2 + R3)

**Сценарий:** Загрузка полного ref_entry для preview. Если слово есть в каталоге — вернуть из БД. Если нет — загрузить из внешних API, сохранить, вернуть.

**Метод:** `GetOrFetchEntry(ctx context.Context, text string) → (*domain.RefEntry, error)`

**Flow:**

```
1. normalized = domain.NormalizeText(text)
   └─ if normalized == "" → return nil, ValidationError("text", "required")

2. existing, err = refEntryRepo.GetFullTreeByText(ctx, normalized)
   ├─ existing != nil → return existing, nil  (есть в каталоге)
   └─ err != nil && !errors.Is(err, ErrNotFound) → return nil, err

--- С этого момента слова нет в каталоге. Внешние HTTP-вызовы ниже. ---

3. dictResult, err = dictionaryProvider.FetchEntry(ctx, text)
   ├─ err → log ERROR "dictionary provider failed" text=... , return nil, fmt.Errorf("refcatalog: fetch entry: %w", err)
   └─ dictResult == nil → return nil, fmt.Errorf("refcatalog: %w", ErrWordNotFound)

4. translations, err = translationProvider.FetchTranslations(ctx, text)
   ├─ err → log WARN "translation provider failed" text=... , translations = nil (graceful degradation)
   └─ translations == nil → ok, продолжаем без переводов

5. refEntry = mapToRefEntry(normalized, dictResult, translations)
   // Генерация UUID для всех сущностей, source_slug, positions

6. var saved *domain.RefEntry
   err = txManager.RunInTx(ctx, func(ctx) error {
       var txErr error
       saved, txErr = refEntryRepo.CreateWithTree(ctx, refEntry)
       return txErr
   })

   if errors.Is(err, ErrAlreadyExists) {
       // Concurrent create — другой запрос уже сохранил это слово
       saved, err = refEntryRepo.GetFullTreeByText(ctx, normalized)
       if err != nil {
           return nil, fmt.Errorf("refcatalog: get after conflict: %w", err)
       }
   } else if err != nil {
       return nil, fmt.Errorf("refcatalog: create ref entry: %w", err)
   }

7. log INFO "ref entry fetched and saved" text=... senses_count=N pronunciations_count=N

8. return saved, nil
```

**Corner cases:**

- **Concurrent fetch:** Два пользователя preview одно слово одновременно. Оба вызывают FetchEntry от FreeDictionary. Первый сохраняет CreateWithTree. Второй получает ErrAlreadyExists → загружает existing GetFullTreeByText. Оба возвращают корректный результат. HTTP-вызовы к FreeDictionary дублируются — допустимая цена за простоту (нет advisory lock).

- **Provider timeout:** dictionaryProvider.FetchEntry вернул ошибку → ошибка пробрасывается. Клиент может retry или создать custom entry.

- **Translation provider failure:** Graceful degradation — entry сохраняется без переводов. Лог WARN, не ERROR. Пользователь добавит переводы вручную через ContentService.

- **Word not found in provider:** dictResult == nil — слово не найдено в FreeDictionary. Возвращается ErrWordNotFound. Клиент предлагает создать custom entry.

- **Empty senses:** Provider вернул слово без definitions → RefEntry создаётся без senses. Допустимо — пользователь может добавить senses вручную.

- **Text нормализация:** "  Abandon  " → "abandon". Клиент передаёт raw text, сервис нормализует.

---

### 3.3. GetRefEntry (R2)

**Сценарий:** Получение ref_entry по ID с полным деревом. Используется DictionaryService.CreateEntryFromCatalog для получения дерева ранее загруженного слова.

**Метод:** `GetRefEntry(ctx context.Context, refEntryID uuid.UUID) → (*domain.RefEntry, error)`

**Flow:**

```
1. refEntryRepo.GetFullTreeByID(ctx, refEntryID)
   └─ ErrNotFound → return nil, ErrNotFound

2. return refEntry, nil
```

Метод **не требует userID** — каталог shared.

---

## 4. Маппинг провайдерских данных

### 4.1. mapToRefEntry

Приватная функция в `mapper.go`. Конвертирует `provider.DictionaryResult` + `[]string` (translations) → `domain.RefEntry` с полным деревом.

**Правила маппинга:**

| Источник | Цель | Логика |
|----------|------|--------|
| dict.Word | RefEntry.Text | Как есть |
| normalized (параметр) | RefEntry.TextNormalized | Уже нормализован |
| dict.Senses[i].Definition | RefSense.Definition | Как есть |
| dict.Senses[i].PartOfSpeech | RefSense.PartOfSpeech | mapPartOfSpeech() — UPPER → domain.PartOfSpeech, unknown → OTHER |
| dict.Senses[i].Examples[j].Sentence | RefExample.Sentence | Как есть |
| dict.Senses[i].Examples[j].Translation | RefExample.Translation | Nullable |
| dict.Pronunciations[k] | RefPronunciation | Transcription, AudioURL, Region — nullable |
| translations[m] (если есть) | RefTranslation | Привязываются к первому sense. source_slug = "translate" |

**Генерация ID:**
- Все UUID генерируются через `uuid.New()` в момент маппинга.
- RefEntry.ID, RefSense.ID, RefTranslation.ID, RefExample.ID, RefPronunciation.ID — все уникальные.

**source_slug:**
- Senses, examples, pronunciations → source_slug зависит от провайдера (для FreeDictionary → `"freedict"`)
- Translations от translationProvider → `"translate"`

**Positions:**
- Senses: 0, 1, 2, ... по порядку из DictionaryResult
- Examples внутри sense: 0, 1, 2, ...
- Translations: 0, 1, 2, ...

**Translations привязка:**
- Если translationProvider вернул `[]string` и есть хотя бы один sense → все translations привязываются к **первому** sense (position=0). Это упрощение для MVP.
- Если senses пустой → translations игнорируются (нет parent для привязки).

### 4.2. mapPartOfSpeech

```
func mapPartOfSpeech(pos *string) → *domain.PartOfSpeech:
    if pos == nil → return nil
    upper = strings.ToUpper(*pos)
    mapped = domain.PartOfSpeech(upper)
    if mapped.IsValid() → return &mapped
    return &domain.PartOfSpeechOther
```

---

## 5. Ошибки

### 5.1. Sentinel errors

```
var ErrWordNotFound = errors.New("word not found in external provider")
```

Определяется в пакете `service/refcatalog/`. Не в `domain/errors.go` — это service-level ошибка, специфичная для RefCatalog.

### 5.2. Error Scenarios — полная таблица

| Операция | Условие | Ошибка | Логирование |
|----------|---------|--------|-------------|
| Search | Пустой query | `[]` (не ошибка) | — |
| Search | Repo error | error проброс | ERROR |
| GetOrFetchEntry | Пустой text | ValidationError("text", "required") | — |
| GetOrFetchEntry | Text из пробелов | ValidationError("text", "required") | — |
| GetOrFetchEntry | Слово в каталоге | — (success) | — |
| GetOrFetchEntry | Слово не найдено в provider | ErrWordNotFound | INFO "word not found in provider" |
| GetOrFetchEntry | Dictionary provider error | error проброс | ERROR "dictionary provider failed" |
| GetOrFetchEntry | Dictionary provider timeout | error проброс | ERROR "dictionary provider failed" |
| GetOrFetchEntry | Translation provider error | Graceful degradation | WARN "translation provider failed" |
| GetOrFetchEntry | Concurrent create | Загрузка existing | — |
| GetRefEntry | ID не найден | ErrNotFound | — |

---

## 6. Тесты

### 6.1. Моки

Все зависимости мокаются: refEntryRepo, txManager, dictionaryProvider, translationProvider.

Mock TxManager: `fn(ctx)` pass-through (как в auth service).

### 6.2. Тест-кейсы

**Search:**

| # | Тест | Assert |
|---|------|--------|
| S1 | Пустой query | return [], refEntryRepo.Search NOT called |
| S2 | Нормальный query | refEntryRepo.Search called, results returned |
| S3 | Limit clamped to max | limit=999 → Search called with limit=50 |
| S4 | Limit clamped to min | limit=0 → Search called with limit=1 |

**GetOrFetchEntry:**

| # | Тест | Assert |
|---|------|--------|
| GF1 | Слово в каталоге | GetFullTreeByText returns existing, providers NOT called |
| GF2 | Слова нет, fetch success, без переводов | FetchEntry called, FetchTranslations returns nil, CreateWithTree called, entry returned |
| GF3 | Слова нет, fetch success, с переводами | FetchTranslations returns ["перевод"], translations привязаны к первому sense |
| GF4 | Слово не найдено в provider (nil result) | ErrWordNotFound |
| GF5 | Dictionary provider error | Error пробрасывается, ERROR logged |
| GF6 | Translation provider error, graceful degradation | Entry saved без переводов, WARN logged |
| GF7 | Concurrent create (ErrAlreadyExists от CreateWithTree) | GetFullTreeByText called после conflict, existing returned |
| GF8 | Пустой text | ValidationError |
| GF9 | Text из пробелов | ValidationError (после нормализации пустой) |
| GF10 | Provider returns no senses | RefEntry created without senses |
| GF11 | Provider returns senses without examples | RefEntry created, senses без examples |
| GF12 | Переводы есть, но senses пустой | Переводы игнорируются |

**GetRefEntry:**

| # | Тест | Assert |
|---|------|--------|
| GR1 | Найден | RefEntry with full tree returned |
| GR2 | Не найден | ErrNotFound |

**mapToRefEntry (unit, table-driven):**

| # | Тест | Assert |
|---|------|--------|
| M1 | Full result с senses, examples, pronunciations | Все поля корректно замаплены |
| M2 | Без переводов (nil) | Senses без translations |
| M3 | С переводами | Translations привязаны к первому sense, source_slug="translate" |
| M4 | Без pronunciations | Пустой массив pronunciations |
| M5 | Множественные senses | Корректные positions (0, 1, 2...) |
| M6 | Все UUID уникальны | Нет дубликатов ID в дереве |

**mapPartOfSpeech (unit, table-driven):**

| # | Тест | Assert |
|---|------|--------|
| P1 | "noun" → NOUN | Correct mapping |
| P2 | "verb" → VERB | Correct mapping |
| P3 | "unknown_pos" → OTHER | Fallback |
| P4 | nil → nil | Nil passthrough |

**Всего: ~22 тест-кейса**

---

## 7. Файловая структура

```
internal/service/refcatalog/
├── service.go           # Service struct, конструктор, приватные интерфейсы
│                        # Search, GetOrFetchEntry, GetRefEntry
├── mapper.go            # mapToRefEntry, mapPartOfSpeech (приватные)
├── errors.go            # ErrWordNotFound
└── service_test.go      # Все тесты из секции 6 (~22 теста)
```

---

## 8. Взаимодействие с другими сервисами

### 8.1. DictionaryService (потребитель)

DictionaryService определяет свой узкий интерфейс refCatalogService (consumer-defined):

```
refCatalogService interface {
    Search(ctx context.Context, query string, limit int) → ([]domain.RefEntry, error)
    GetOrFetchEntry(ctx context.Context, text string) → (*domain.RefEntry, error)
    GetRefEntry(ctx context.Context, refEntryID uuid.UUID) → (*domain.RefEntry, error)
}
```

DictionaryService **не** вызывает external providers напрямую — всё через RefCatalogService.

### 8.2. External Providers (зависимости)

Два типа провайдеров:

| Тип | Реализация (MVP) | Реализация (post-MVP) |
|-----|------------------|----------------------|
| dictionaryProvider | FreeDictionary API | + OpenAI, Wiktionary |
| translationProvider | Stub (returns nil) | Google Translate, DeepL |

Провайдеры реализуют интерфейсы, определённые в RefCatalog Service. Wiring — в `main.go`.

### 8.3. Transport Layer

RefCatalog Service **не** имеет прямых transport endpoints. Доступ к каталогу — исключительно через DictionaryService (SearchCatalog, PreviewRefEntry, CreateEntryFromCatalog).
