# MyEnglish Backend v4 — Dictionary Service Specification

> **Статус:** Draft v1.0
> **Дата:** 2026-02-12
> **Зависимости:** code_conventions_v4.md (секции 2–4, 6–8), data_model_v4.md (секции 1, 4), repo_layer_spec_v4.md (секции 3–5, 8–9, 11, 13, 16, 19.2), service_layer_spec_v4.md, business_scenarios_v4.md (D1–D13, B1, B4, B5)

---

## 1. Ответственность

Dictionary Service — центральный сервис приложения. Отвечает за **управление словарём пользователя**: добавление слов (из каталога и вручную), поиск и фильтрация, редактирование заметок, soft delete / restore, импорт и экспорт.

Dictionary Service **не** отвечает за: CRUD дочернего контента (senses, translations, examples — это ContentService), review карточек (StudyService), управление топиками (TopicService). При добавлении слова DictionaryService создаёт начальный контент (senses/translations/examples из каталога), но дальнейшее редактирование — ContentService.

---

## 2. Зависимости

### 2.1. Интерфейсы репозиториев

```
entryRepo interface {
    GetByID(ctx, userID, entryID uuid.UUID) → (*domain.Entry, error)
    GetByText(ctx, userID uuid.UUID, textNormalized string) → (*domain.Entry, error)
    Find(ctx, userID uuid.UUID, filter entry.Filter) → ([]domain.Entry, int, error)
        // offset mode: entries, totalCount, error
    FindCursor(ctx, userID uuid.UUID, filter entry.Filter) → ([]domain.Entry, bool, error)
        // cursor mode: entries, hasNextPage, error
    GetByIDs(ctx, userID uuid.UUID, ids []uuid.UUID) → ([]domain.Entry, error)
    CountByUser(ctx, userID uuid.UUID) → (int, error)
    Create(ctx, entry *domain.Entry) → (*domain.Entry, error)
    UpdateNotes(ctx, userID, entryID uuid.UUID, notes *string) → (*domain.Entry, error)
    SoftDelete(ctx, userID, entryID uuid.UUID) → error
    Restore(ctx, userID, entryID uuid.UUID) → (*domain.Entry, error)
    FindDeleted(ctx, userID uuid.UUID, limit, offset int) → ([]domain.Entry, int, error)
    HardDeleteOld(ctx, threshold time.Time) → (int, error)
}

senseRepo interface {
    CreateFromRef(ctx, entryID, refSenseID uuid.UUID, sourceSlug string) → (*domain.Sense, error)
    CreateCustom(ctx, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) → (*domain.Sense, error)
}

translationRepo interface {
    CreateFromRef(ctx, senseID, refTranslationID uuid.UUID, sourceSlug string) → (*domain.Translation, error)
    CreateCustom(ctx, senseID uuid.UUID, text string, sourceSlug string) → (*domain.Translation, error)
}

exampleRepo interface {
    CreateFromRef(ctx, senseID, refExampleID uuid.UUID, sourceSlug string) → (*domain.Example, error)
    CreateCustom(ctx, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) → (*domain.Example, error)
}

pronunciationRepo interface {
    Link(ctx, entryID, refPronunciationID uuid.UUID) → error
}

imageRepo interface {
    LinkCatalog(ctx, entryID, refImageID uuid.UUID) → error
}

cardRepo interface {
    Create(ctx, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) → (*domain.Card, error)
}

auditRepo interface {
    Log(ctx, record domain.AuditRecord) → error
}
```

### 2.2. Интерфейс TxManager

```
txManager interface {
    RunInTx(ctx, fn func(ctx) error) → error
}
```

### 2.3. Интерфейс RefCatalogService

Единственная inter-service зависимость:

```
refCatalogService interface {
    GetOrFetchEntry(ctx, text string) → (*domain.RefEntry, error)
    GetRefEntry(ctx, refEntryID uuid.UUID) → (*domain.RefEntry, error)
    Search(ctx, query string, limit int) → ([]domain.RefEntry, error)
}
```

### 2.4. Конфигурация

```
DictionaryConfig {
    MaxEntriesPerUser   int             // default: 10_000
    DefaultEaseFactor   float64         // default: 2.5 (для создания карточки)
    ImportChunkSize     int             // default: 50
    ExportMaxEntries    int             // default: 10_000
}
```

### 2.5. Конструктор

```
func NewService(
    logger            *slog.Logger,
    entryRepo         entryRepo,
    senseRepo         senseRepo,
    translationRepo   translationRepo,
    exampleRepo       exampleRepo,
    pronunciationRepo pronunciationRepo,
    imageRepo         imageRepo,
    cardRepo          cardRepo,
    auditRepo         auditRepo,
    txManager         txManager,
    refCatalog        refCatalogService,
    config            DictionaryConfig,
) *Service
```

---

## 3. Бизнес-сценарии и операции

### 3.1. SearchCatalog (D1)

**Сценарий:** Пользователь вводит текст в поле поиска → видит список слов из Reference Catalog для autocomplete.

**Метод:** `SearchCatalog(ctx context.Context, query string, limit int) → ([]domain.RefEntry, error)`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
   └─ ошибка → return ErrUnauthorized

2. Clamp limit: min(max(limit, 1), 50), default 20

3. if query == "" → return [], nil  (пустой запрос — пустой результат)

4. refCatalog.Search(ctx, query, limit)
   └─ Возвращает только ref_entries (text, text_normalized, id) без полного дерева — для autocomplete достаточно

5. return results, nil
```

**Corner cases:**
- Пустой query → пустой результат без обращения к БД.
- Query из одного символа → допустим, pg_trgm сам справится с релевантностью.
- Каталог пуст (свежая установка) → пустой результат, не ошибка.

---

### 3.2. PreviewRefEntry (D2)

**Сценарий:** Пользователь выбрал слово из autocomplete → загружается полное дерево ref_entry для preview в конструкторе.

**Метод:** `PreviewRefEntry(ctx context.Context, text string) → (*domain.RefEntry, error)`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
   └─ ошибка → return ErrUnauthorized

2. refCatalog.GetOrFetchEntry(ctx, text)
   ├─ Слово есть в каталоге → вернуть RefEntry с полным деревом
   ├─ Слова нет → fetch from external API → сохранить → вернуть
   └─ External API error → return error

3. return refEntry, nil
```

Метод возвращает полное дерево: RefEntry → []RefSense → []RefTranslation, []RefExample; []RefPronunciation; []RefImage.

Клиент использует это для отображения конструктора: пользователь видит все senses, выбирает нужные, затем вызывает CreateEntryFromCatalog.

**Corner cases:**
- Слово не найдено в external API → RefCatalogService возвращает ошибку "word not found". Клиент предлагает создать custom entry.
- External API timeout → ошибка пробрасывается. Клиент предлагает retry или создать custom.
- Concurrent preview → два пользователя preview одно слово → RefCatalogService.GetOrFetchEntry идемпотентен (INSERT ON CONFLICT DO NOTHING).

---

### 3.3. CreateEntryFromCatalog (D3)

**Сценарий:** Пользователь выбрал senses/translations/examples в конструкторе → нажал "Добавить" → слово создаётся в словаре со ссылками на каталог.

**Метод:** `CreateEntryFromCatalog(ctx context.Context, input CreateFromCatalogInput) → (*domain.Entry, error)`

**CreateFromCatalogInput:**
- RefEntryID (uuid.UUID) — ID ref_entry из каталога
- SenseIDs ([]uuid.UUID) — выбранные ref_senses (пустой = взять все)
- CreateCard (bool) — создать карточку для изучения
- Notes (*string) — заметки пользователя

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
   └─ ошибка → return ErrUnauthorized

2. input.Validate()
   └─ ошибка → return ValidationError

3. refEntry, err = refCatalog.GetRefEntry(ctx, input.RefEntryID)
   └─ ErrNotFound → return ValidationError("ref_entry_id", "reference entry not found")

4. normalized = domain.NormalizeText(refEntry.Text)

5. count, err = entryRepo.CountByUser(ctx, userID)
   └─ count >= config.MaxEntriesPerUser → return ValidationError("entries", "limit reached (10000)")

6. existing, err = entryRepo.GetByText(ctx, userID, normalized)
   └─ existing найден → return ErrAlreadyExists

--- До этого момента все операции — чтение. Внешних HTTP-вызовов нет (refEntry уже в БД). ---
--- Транзакция начинается ниже. ---

7. Определить senses для добавления:
   ├─ input.SenseIDs не пустой → отфильтровать refEntry.Senses по ID, валидировать что все найдены
   │   └─ какой-то ID не найден в refEntry.Senses → return ValidationError("sense_ids", "invalid sense ID: ...")
   └─ input.SenseIDs пустой → взять все refEntry.Senses

8. txManager.RunInTx(ctx, func(ctx) error {

   8a. entry, err = entryRepo.Create(ctx, &domain.Entry{
           UserID:         userID,
           RefEntryID:     &refEntry.ID,
           Text:           refEntry.Text,
           TextNormalized: normalized,
           Notes:          input.Notes,
       })
       └─ ErrAlreadyExists → return (concurrent create — unique constraint)

   8b. Для каждого выбранного refSense:
       sense, err = senseRepo.CreateFromRef(ctx, entry.ID, refSense.ID, refSense.SourceSlug)

       8b-i. Для каждого refTranslation в refSense.Translations:
             translationRepo.CreateFromRef(ctx, sense.ID, refTranslation.ID, refTranslation.SourceSlug)

       8b-ii. Для каждого refExample в refSense.Examples:
              exampleRepo.CreateFromRef(ctx, sense.ID, refExample.ID, refExample.SourceSlug)

   8c. Для каждого refPronunciation в refEntry.Pronunciations:
       pronunciationRepo.Link(ctx, entry.ID, refPronunciation.ID)

   8d. Для каждого refImage в refEntry.Images:
       imageRepo.LinkCatalog(ctx, entry.ID, refImage.ID)

   8e. if input.CreateCard:
       cardRepo.Create(ctx, userID, entry.ID, domain.StatusNew, config.DefaultEaseFactor)

   8f. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityEntry,
           EntityID:   &entry.ID,
           Action:     domain.ActionCreate,
           Changes: map[string]any{
               "text":        {"new": entry.Text},
               "source":      {"new": "catalog"},
               "senses_count": {"new": len(selectedSenses)},
               "card_created": {"new": input.CreateCard},
           },
       })

   return nil
   })

9. Логировать INFO "entry created from catalog" user_id=... entry_id=... text=...

10. return entry, nil  (entry без загруженного дерева — клиент запросит через DataLoaders)
```

**Corner cases:**

- **Дубликат text (оптимистичная проверка):** Шаг 6 проверяет GetByText перед транзакцией. Но concurrent request может создать entry между шагами 6 и 8a. Unique constraint в repo — последняя линия защиты. ErrAlreadyExists из repo пробрасывается наверх.

- **SenseIDs пустой:** Добавляются все senses из ref_entry со всеми их translations и examples. Это основной сценарий для "добавить быстро" (без выбора).

- **SenseIDs — частичный выбор:** Пользователь выбрал 2 из 5 senses. Translations и examples добавляются только для выбранных senses. Pronunciations и images добавляются всегда (они привязаны к entry, не к sense).

- **RefEntry без senses:** Возможно, если external API вернул слово без определений. Entry создаётся пустым (без senses). Пользователь может добавить senses вручную через ContentService.

- **Большое дерево:** RefEntry может иметь 10+ senses, каждый с 5+ translations/examples. Это десятки INSERT в одной транзакции. Допустимо для PostgreSQL, но стоит мониторить duration.

- **Уже soft-deleted entry с тем же text:** Partial unique constraint (WHERE deleted_at IS NULL) позволяет создать новый entry. Старый soft-deleted остаётся в БД.

---

### 3.4. CreateEntryCustom (D4)

**Сценарий:** Пользователь вводит слово и данные вручную (без каталога) → слово добавляется с user-данными.

**Метод:** `CreateEntryCustom(ctx context.Context, input CreateCustomInput) → (*domain.Entry, error)`

**CreateCustomInput:**
- Text (string) — текст слова/фразы
- Senses ([]SenseInput) — массив значений
- CreateCard (bool)
- Notes (*string)

**SenseInput:**
- Definition (*string) — определение (может быть пустым)
- PartOfSpeech (*domain.PartOfSpeech) — часть речи (может быть nil)
- Translations ([]string) — переводы
- Examples ([]ExampleInput) — примеры

**ExampleInput:**
- Sentence (string)
- Translation (*string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. normalized = domain.NormalizeText(input.Text)

4. count check: CountByUser < MaxEntriesPerUser
5. duplicate check: GetByText(userID, normalized)

6. txManager.RunInTx(ctx, func(ctx) error {

   6a. entry = entryRepo.Create(ctx, &domain.Entry{
           UserID:         userID,
           RefEntryID:     nil,  // нет ссылки на каталог
           Text:           input.Text,
           TextNormalized: normalized,
           Notes:          input.Notes,
       })

   6b. Для каждого SenseInput:
       sense = senseRepo.CreateCustom(ctx, entry.ID, si.Definition, si.PartOfSpeech, nil, "user")

       6b-i. Для каждого translation в si.Translations:
             translationRepo.CreateCustom(ctx, sense.ID, text, "user")

       6b-ii. Для каждого example в si.Examples:
              exampleRepo.CreateCustom(ctx, sense.ID, sentence, translation, "user")

   6c. if input.CreateCard:
       cardRepo.Create(ctx, userID, entry.ID, domain.StatusNew, config.DefaultEaseFactor)

   6d. auditRepo.Log(ctx, ...)

   return nil
   })

7. Логировать INFO "entry created custom" user_id=... entry_id=... text=...
8. return entry, nil
```

**Corner cases:**

- **Entry без senses:** Senses = [] — допустимо. Пользователь может создать пустое слово и добавить senses позже через ContentService.
- **Sense без definition:** Definition = nil — допустимо. Sense может иметь только translations.
- **Sense без translations:** Допустимо.
- **source_slug:** Все записи получают source_slug = "user".
- **Нормализация:** Применяется `domain.NormalizeText` — trim, lower, compress spaces. Текст "  Abandon  " → "abandon".

---

### 3.5. FindEntries (D5 + D6)

**Сценарий D5:** Пользователь ищет слово по тексту в своём словаре.
**Сценарий D6:** Пользователь фильтрует словарь по комбинации параметров.

Оба сценария реализуются **одной операцией** с динамическими фильтрами.

**Метод:** `FindEntries(ctx context.Context, input FindInput) → (*FindResult, error)`

**FindInput:**
- Search (*string) — поиск по тексту (ILIKE)
- HasCard (*bool) — фильтр по наличию карточки
- PartOfSpeech (*domain.PartOfSpeech)
- TopicID (*uuid.UUID)
- Status (*domain.LearningStatus)
- SortBy (string) — "text", "created_at", "updated_at" (default: "created_at")
- SortOrder (string) — "ASC", "DESC" (default: "DESC")
- Limit (int) — 1–200 (default: 50)
- Cursor (*string) — для cursor-based pagination (nil = первая страница)
- Offset (*int) — для offset-based pagination (nil = не используется)

**FindResult:**
- Entries ([]domain.Entry)
- TotalCount (int) — только для offset mode
- HasNextPage (bool) — только для cursor mode
- PageInfo (*PageInfo) — startCursor, endCursor для GraphQL

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. Нормализация:
   ├─ Search не nil → NormalizeText(*Search)
   │   └─ если после нормализации пустая строка → set Search = nil (игнорировать)
   └─ Clamp Limit: min(max(limit, 1), 200)

4. Построить entry.Filter из input (маппинг полей)

5. Определить mode:
   ├─ Cursor != nil → cursor mode:
   │   entries, hasNextPage, err = entryRepo.FindCursor(ctx, userID, filter)
   │   return FindResult{Entries: entries, HasNextPage: hasNextPage, PageInfo: buildPageInfo(entries)}
   │
   └─ иначе → offset mode:
       entries, totalCount, err = entryRepo.Find(ctx, userID, filter)
       return FindResult{Entries: entries, TotalCount: totalCount}
```

**Corner cases:**

- **Нулевые фильтры:** Все фильтры nil → возвращает все entries пользователя (paginated).
- **Search из пробелов:** "   " → после NormalizeText → "" → Search = nil → игнорируется.
- **Комбинация фильтров:** Все фильтры AND-ятся. `Search="cat" AND TopicID="animals" AND HasCard=true` — слова с "cat" в тексте, в теме "animals", с карточкой.
- **Cursor + Offset одновременно:** Cursor приоритетнее. Если оба заданы — используется cursor mode.
- **Невалидный cursor:** Repo вернёт ErrValidation → пробрасывается.
- **Пустой результат:** Entries = [], TotalCount = 0, HasNextPage = false. Не ошибка.

---

### 3.6. GetEntry (D7)

**Сценарий:** Пользователь открывает слово → видит полную информацию.

**Метод:** `GetEntry(ctx context.Context, entryID uuid.UUID) → (*domain.Entry, error)`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. entryRepo.GetByID(ctx, userID, entryID)
   └─ ErrNotFound → return ErrNotFound
3. return entry, nil
```

Метод возвращает entry без загруженного дерева (senses, translations, ...). Дочерний контент загружается клиентом через **DataLoaders** в GraphQL resolver. Это архитектурное решение: entry — aggregate root, дочерний контент подгружается лениво.

**Corner cases:**
- Soft-deleted entry → GetByID фильтрует `deleted_at IS NULL` → ErrNotFound.
- Entry другого пользователя → GetByID фильтрует `user_id` → ErrNotFound (не Forbidden — не раскрываем существование).

---

### 3.7. UpdateNotes (D8)

**Сценарий:** Пользователь добавляет/изменяет заметки к слову.

**Метод:** `UpdateNotes(ctx context.Context, input UpdateNotesInput) → (*domain.Entry, error)`

**UpdateNotesInput:**
- EntryID (uuid.UUID)
- Notes (*string) — nil = очистить заметки

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. oldEntry, err = entryRepo.GetByID(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {
   4a. entry, err = entryRepo.UpdateNotes(ctx, userID, input.EntryID, input.Notes)
   4b. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityEntry,
           EntityID:   &entry.ID,
           Action:     domain.ActionUpdate,
           Changes: map[string]any{
               "notes": {"old": oldEntry.Notes, "new": input.Notes},
           },
       })
   return nil
   })

5. return entry, nil
```

**Corner cases:**
- Notes = nil → очистить заметки (set notes = NULL в БД).
- Notes не изменились (old == new) → всё равно выполнить UPDATE + audit. Проще и предсказуемее, чем сравнивать.

---

### 3.8. DeleteEntry (D9)

**Сценарий:** Пользователь удаляет слово → soft delete.

**Метод:** `DeleteEntry(ctx context.Context, entryID uuid.UUID) → error`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. entry, err = entryRepo.GetByID(ctx, userID, entryID)
   └─ ErrNotFound → return ErrNotFound

3. txManager.RunInTx(ctx, func(ctx) error {
   3a. entryRepo.SoftDelete(ctx, userID, entryID)
   3b. auditRepo.Log(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityEntry,
           EntityID:   &entryID,
           Action:     domain.ActionDelete,
           Changes: map[string]any{
               "text": {"old": entry.Text},
           },
       })
   return nil
   })

4. Логировать INFO "entry soft-deleted" user_id=... entry_id=... text=...
5. return nil
```

**Corner cases:**
- **Уже deleted:** SoftDelete идемпотентен (set deleted_at = now() WHERE deleted_at IS NULL; affected 0 rows — не ошибка). Но GetByID на шаге 2 вернёт ErrNotFound для уже deleted entry, поэтому этот кейс не достижим через обычный flow.
- **Карточка слова:** Карточка остаётся в БД, но GetDueCards фильтрует soft-deleted entries через JOIN. Карточка фактически исключена из очереди.
- **Дочерний контент:** Senses, translations, examples остаются в БД. Они скрыты вместе с entry. Hard delete через 30 дней удалит всё CASCADE.

---

### 3.9. FindDeletedEntries (D10)

**Сценарий:** Пользователь просматривает корзину (список soft-deleted слов).

**Метод:** `FindDeletedEntries(ctx context.Context, limit, offset int) → ([]domain.Entry, int, error)`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. Clamp limit: min(max(limit, 1), 200), default 50
3. entryRepo.FindDeleted(ctx, userID, limit, offset)
   └─ Возвращает entries WHERE deleted_at IS NOT NULL, ORDER BY deleted_at DESC
4. return entries, totalCount, nil
```

**Corner cases:**
- Пустая корзина → [], 0, nil.
- Entries старше 30 дней могут быть уже hard-deleted (background job). Корзина показывает только то, что ещё есть.

---

### 3.10. RestoreEntry (D11)

**Сценарий:** Пользователь восстанавливает soft-deleted слово.

**Метод:** `RestoreEntry(ctx context.Context, entryID uuid.UUID) → (*domain.Entry, error)`

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. Проверить, нет ли active entry с тем же text:
   // Нужен специальный запрос: получить deleted entry, затем проверить active duplicate
   // Альтернатива: сделать Restore и поймать unique constraint violation

   Подход: optimistic.
   2a. entryRepo.Restore(ctx, userID, entryID)
       ├─ ErrNotFound → entry не существует или не deleted → return ErrNotFound
       └─ ErrAlreadyExists → active entry с тем же text уже есть
           → return ValidationError("text", "active entry with this text already exists")

3. Логировать INFO "entry restored" user_id=... entry_id=...
4. return entry, nil
```

**Corner cases:**
- **Text conflict:** Пользователь удалил "abandon", создал новый "abandon", пытается восстановить старый. Partial unique constraint не позволяет два active entry с одним text_normalized. Restore → unique violation → ErrAlreadyExists → понятная ошибка.
- **Hard-deleted entry:** Restore для entry, которого уже нет (hard delete прошёл). GetByID / Restore → ErrNotFound.
- **Карточка при восстановлении:** Карточка (если была) всё ещё в БД (FK ON DELETE CASCADE не сработал при soft delete). После restore карточка снова попадает в GetDueCards.

---

### 3.11. BatchDeleteEntries (B1)

**Сценарий:** Пользователь выбирает несколько слов → удаляет их разом.

**Метод:** `BatchDeleteEntries(ctx context.Context, entryIDs []uuid.UUID) → (*BatchResult, error)`

**BatchResult:**
- Deleted (int) — количество успешно удалённых
- Errors ([]BatchError) — ошибки по отдельным entries (если есть)

**BatchError:**
- EntryID (uuid.UUID)
- Error (string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. Validate: len(entryIDs) > 0, len(entryIDs) <= 200

3. Для каждого entryID:
   err = entryRepo.SoftDelete(ctx, userID, entryID)
   ├─ nil → deleted++
   └─ error → append to errors (entry не найден, не принадлежит пользователю, и т.д.)

4. if deleted > 0:
   auditRepo.Log(ctx, domain.AuditRecord{
       UserID:     userID,
       EntityType: domain.EntityEntry,
       Action:     domain.ActionDelete,
       Changes:    {"batch_deleted": {"new": deleted}},
   })

5. return &BatchResult{Deleted: deleted, Errors: errors}, nil
```

**Важно:** Batch delete **не** в транзакции. Каждый SoftDelete — отдельная операция. Partial failure допустим: 8 из 10 удалены, 2 не найдены → не ошибка, а отчёт.

---

### 3.12. ImportEntries (D12 + B4)

**Сценарий:** Пользователь загружает файл (CSV/JSON) → массовое создание entries.

**Метод:** `ImportEntries(ctx context.Context, input ImportInput) → (*ImportResult, error)`

**ImportInput:**
- Items ([]ImportItem) — до 5000 записей

**ImportItem:**
- Text (string)
- Translations ([]string)
- Notes (*string)
- TopicName (*string) — имя топика для автоматической привязки

**ImportResult:**
- Imported (int) — успешно создано
- Skipped (int) — пропущено (дубликат, пустой текст)
- Errors ([]ImportError) — ошибки

**ImportError:**
- LineNumber (int)
- Text (string)
- Reason (string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)
2. input.Validate()

3. Проверить лимит: entryRepo.CountByUser(ctx, userID) + len(input.Items) <= MaxEntriesPerUser
   └─ Превышает → return ValidationError("entries", "import would exceed limit")
      Проверка приблизительная (concurrent creates могут добавить между проверкой и импортом).

4. Разбить Items на chunks по config.ImportChunkSize (50):

   Для каждого chunk:
     txManager.RunInTx(ctx, func(ctx) error {

       Для каждого item в chunk:
         normalized = domain.NormalizeText(item.Text)
         if normalized == "" → skip (append Skipped)

         existing, err = entryRepo.GetByText(ctx, userID, normalized)
         if existing != nil → skip (append Skipped, reason: "already exists")

         entry, err = entryRepo.Create(ctx, ...)
         if ErrAlreadyExists → skip (concurrent duplicate)

         // Для каждого translation → create sense + translation
         if len(item.Translations) > 0:
           sense = senseRepo.CreateCustom(ctx, entry.ID, nil, nil, nil, "import")
           for _, tr := range item.Translations:
             translationRepo.CreateCustom(ctx, sense.ID, tr, "import")

         imported++

       return nil
     })
     └─ Ошибка chunk → записать в errors для всех items chunk, продолжить со следующего chunk

5. Логировать INFO "import completed" user_id=... imported=... skipped=... errors=...
6. return &ImportResult{Imported, Skipped, Errors}, nil
```

**Corner cases:**

- **Partial failure:** Ошибка в chunk откатывает весь chunk (транзакция), но не всю операцию. Следующие chunks продолжают обрабатываться.
- **Дубликаты внутри файла:** Два одинаковых слова в одном файле. Первое создаётся, второе skipped ("already exists").
- **Пустой text:** После нормализации text пустой → skip.
- **TopicName:** На MVP не реализуется (требует TopicService). Поле зарезервировано в ImportItem для будущей реализации. Если передано — игнорируется.
- **5000 записей * 50 chunk = 100 транзакций.** Каждая транзакция содержит ~50 entries * ~3 INSERT = ~150 операций. Это ощутимая нагрузка. Импорт должен выполняться с увеличенным timeout (transport layer).
- **source_slug:** Все записи получают source_slug = "import".

---

### 3.13. ExportEntries (D13 + B5)

**Сценарий:** Пользователь запрашивает экспорт словаря → получает файл со всеми данными.

**Метод:** `ExportEntries(ctx context.Context) → (*ExportResult, error)`

**ExportResult:**
- Items ([]ExportItem)
- ExportedAt (time.Time)

**ExportItem:**
- Text (string)
- Notes (*string)
- Senses ([]ExportSense)
- CardStatus (*domain.LearningStatus)
- CreatedAt (time.Time)

**ExportSense:**
- Definition (*string) — resolved (COALESCE applied)
- PartOfSpeech (*domain.PartOfSpeech)
- Translations ([]string)
- Examples ([]ExportExample)

**ExportExample:**
- Sentence (string)
- Translation (*string)

**Flow:**

```
1. userID, err = UserIDFromCtx(ctx)

2. entries, totalCount, err = entryRepo.Find(ctx, userID, entry.Filter{
       SortBy: "created_at", SortOrder: "ASC", Limit: config.ExportMaxEntries,
   })

3. Для каждого entry загрузить дочерний контент:
   // Batch-загрузка через repo batch-методы (не DataLoaders — это не GraphQL context)
   entryIDs = extractIDs(entries)
   sensesMap = senseRepo.GetByEntryIDs(ctx, entryIDs)
   // ... аналогично для translations, examples, cards

4. Собрать ExportItems из загруженных данных

5. return &ExportResult{Items: exportItems, ExportedAt: now}, nil
```

**Corner cases:**

- **10 000 entries:** Максимальный объём словаря. Загрузка batch-методами в несколько запросов (entries → senses → translations → examples → cards). Memory: ~10 000 * ~1KB = ~10MB — допустимо.
- **COALESCE:** Senses/translations/examples возвращаются repo с applied COALESCE. Экспорт содержит resolved-значения, не raw user-поля.
- **Формат файла:** Сервис возвращает структурированные данные. Сериализация в CSV/JSON — ответственность transport layer.
- **Soft-deleted:** Не включаются (entryRepo.Find фильтрует deleted_at IS NULL).

---

## 4. Input Validation

### 4.1. CreateFromCatalogInput

```
Validate():
    if RefEntryID == uuid.Nil      → ("ref_entry_id", "required")
    if len(SenseIDs) > 20          → ("sense_ids", "too many (max 20)")
    if Notes != nil && len > 5000  → ("notes", "too long (max 5000)")
```

### 4.2. CreateCustomInput

```
Validate():
    if Text == "" after trim         → ("text", "required")
    if len(Text) > 500              → ("text", "too long (max 500)")
    if len(Senses) > 20             → ("senses", "too many (max 20)")
    for each sense:
      if Definition != nil && len > 2000 → ("senses[i].definition", "too long")
      if len(Translations) > 20     → ("senses[i].translations", "too many")
      for each translation:
        if text == "" after trim     → ("senses[i].translations[j]", "required")
        if len > 500                → ("senses[i].translations[j]", "too long")
      if len(Examples) > 50         → ("senses[i].examples", "too many")
      for each example:
        if Sentence == "" after trim → ("senses[i].examples[j].sentence", "required")
        if len(Sentence) > 2000     → ("senses[i].examples[j].sentence", "too long")
        if Translation != nil && len > 2000 → ("senses[i].examples[j].translation", "too long")
    if Notes != nil && len > 5000   → ("notes", "too long")
```

### 4.3. FindInput

```
Validate():
    if SortBy not in ["text", "created_at", "updated_at"] → ("sort_by", "invalid")
    if SortOrder not in ["ASC", "DESC"]                   → ("sort_order", "invalid")
    // Limit clamped, not validated — convenience
```

### 4.4. UpdateNotesInput

```
Validate():
    if EntryID == uuid.Nil          → ("entry_id", "required")
    if Notes != nil && len > 5000   → ("notes", "too long")
```

### 4.5. ImportInput

```
Validate():
    if len(Items) == 0              → ("items", "required")
    if len(Items) > 5000            → ("items", "too many (max 5000)")
    for each item:
      if Text == "" after trim      → ("items[i].text", "required")
      if len(Text) > 500            → ("items[i].text", "too long")
      if len(Translations) > 20    → ("items[i].translations", "too many")
```

---

## 5. Error Scenarios — полная таблица

| Операция | Условие | Ошибка | Логирование |
|----------|---------|--------|-------------|
| SearchCatalog | Нет userID | ErrUnauthorized | — |
| SearchCatalog | Пустой query | [] (не ошибка) | — |
| PreviewRefEntry | External API timeout | error (проброс) | ERROR "external provider timeout" |
| PreviewRefEntry | Слово не найдено в API | error "word not found" | INFO "word not found in provider" |
| CreateFromCatalog | Невалидный input | ValidationError | — |
| CreateFromCatalog | RefEntry не найден | ValidationError("ref_entry_id") | — |
| CreateFromCatalog | Лимит entries превышен | ValidationError("entries", "limit reached") | WARN "entry limit reached" |
| CreateFromCatalog | Дубликат text | ErrAlreadyExists | — |
| CreateFromCatalog | Concurrent create (unique constraint) | ErrAlreadyExists | — |
| CreateFromCatalog | Невалидный SenseID | ValidationError("sense_ids") | — |
| CreateCustom | Невалидный input | ValidationError | — |
| CreateCustom | Лимит / дубликат | аналогично CreateFromCatalog | — |
| FindEntries | Невалидный cursor | ErrValidation | — |
| GetEntry | Entry не найден / deleted / чужой | ErrNotFound | — |
| UpdateNotes | Entry не найден | ErrNotFound | — |
| DeleteEntry | Entry не найден | ErrNotFound | — |
| RestoreEntry | Entry не найден / не deleted | ErrNotFound | — |
| RestoreEntry | Active entry с тем же text | ValidationError("text") | — |
| ImportEntries | Лимит превышен | ValidationError("entries") | — |
| ImportEntries | Ошибка в chunk | Частичный откат chunk | WARN "import chunk failed" |

---

## 6. Тесты

### 6.1. Моки

Все зависимости мокаются: entryRepo, senseRepo, translationRepo, exampleRepo, pronunciationRepo, imageRepo, cardRepo, auditRepo, txManager, refCatalogService.

### 6.2. Тест-кейсы

**SearchCatalog:**

| # | Тест | Assert |
|---|------|--------|
| SC1 | Пустой query | return [], refCatalog.Search NOT called |
| SC2 | Нормальный query | refCatalog.Search called, results returned |
| SC3 | Limit clamped | limit=999 → search called with limit=50 |
| SC4 | Нет userID | ErrUnauthorized |

**PreviewRefEntry:**

| # | Тест | Assert |
|---|------|--------|
| PR1 | Слово в каталоге | GetOrFetchEntry returns RefEntry with tree |
| PR2 | Слово не в каталоге, fetch ok | GetOrFetchEntry fetches and returns |
| PR3 | External API error | Error пробрасывается |

**CreateFromCatalog:**

| # | Тест | Assert |
|---|------|--------|
| CF1 | Happy path — все senses | Entry создан. Senses = все из ref. Translations, examples, pronunciations, images linked. Audit logged. |
| CF2 | Выбранные senses | Entry создан. Только выбранные senses + их translations/examples |
| CF3 | Пустой SenseIDs | Все senses из ref добавлены |
| CF4 | С карточкой | cardRepo.Create called |
| CF5 | Без карточки | cardRepo.Create NOT called |
| CF6 | С заметками | entry.Notes set |
| CF7 | Дубликат text — проверка GetByText | ErrAlreadyExists |
| CF8 | Дубликат text — unique constraint | ErrAlreadyExists from repo |
| CF9 | Лимит entries | ValidationError "limit reached" |
| CF10 | RefEntry не найден | ValidationError "reference entry not found" |
| CF11 | Невалидный SenseID | ValidationError "invalid sense ID" |
| CF12 | Невалидный input — пустой RefEntryID | ValidationError |
| CF13 | Нет userID | ErrUnauthorized |
| CF14 | Audit record | AuditRecord содержит правильные changes |
| CF15 | RefEntry без senses | Entry создан пустой, без senses |

**CreateCustom:**

| # | Тест | Assert |
|---|------|--------|
| CC1 | Happy path — с senses и translations | Entry создан, senses с source_slug="user" |
| CC2 | Пустые senses | Entry создан без senses |
| CC3 | Sense без definition | Sense создан с definition=nil |
| CC4 | С карточкой | cardRepo.Create called |
| CC5 | Нормализация текста | Text="  Abandon  " → normalized="abandon" |
| CC6 | Дубликат | ErrAlreadyExists |
| CC7 | Невалидный input — пустой text | ValidationError |
| CC8 | Невалидный input — слишком длинный text | ValidationError |
| CC9 | RefEntryID == nil | Entry.RefEntryID is nil |

**FindEntries:**

| # | Тест | Assert |
|---|------|--------|
| FE1 | Без фильтров — offset mode | entryRepo.Find called, entries returned |
| FE2 | С Search | Filter.Search set, normalized |
| FE3 | Search из пробелов | Filter.Search = nil (ignored) |
| FE4 | С HasCard | Filter.HasCard set |
| FE5 | С TopicID | Filter.TopicID set |
| FE6 | Комбинация фильтров | Все фильтры в Filter |
| FE7 | Cursor mode | entryRepo.FindCursor called, hasNextPage returned |
| FE8 | Cursor + Offset → cursor wins | FindCursor called (not Find) |
| FE9 | Limit clamped | limit=999 → filter.Limit=200 |
| FE10 | Default sort | SortBy="created_at", SortOrder="DESC" |
| FE11 | Невалидный SortBy | ValidationError |

**GetEntry:**

| # | Тест | Assert |
|---|------|--------|
| GE1 | Найден | Entry returned |
| GE2 | Не найден | ErrNotFound |
| GE3 | Нет userID | ErrUnauthorized |

**UpdateNotes:**

| # | Тест | Assert |
|---|------|--------|
| UN1 | Set notes | entry.Notes updated. Audit logged with old/new. |
| UN2 | Clear notes (nil) | entry.Notes = nil. Audit logged. |
| UN3 | Entry не найден | ErrNotFound |
| UN4 | Невалидный input — too long | ValidationError |

**DeleteEntry:**

| # | Тест | Assert |
|---|------|--------|
| DE1 | Happy path | SoftDelete called. Audit logged with text. |
| DE2 | Entry не найден | ErrNotFound |
| DE3 | Нет userID | ErrUnauthorized |

**FindDeletedEntries:**

| # | Тест | Assert |
|---|------|--------|
| FD1 | Есть deleted entries | Entries returned, sorted by deleted_at DESC |
| FD2 | Пусто | [], 0 |
| FD3 | Limit clamped | limit=999 → clamped to 200 |

**RestoreEntry:**

| # | Тест | Assert |
|---|------|--------|
| RE1 | Happy path | Entry restored, returned |
| RE2 | Entry не найден / не deleted | ErrNotFound |
| RE3 | Active entry с тем же text | ValidationError |

**BatchDeleteEntries:**

| # | Тест | Assert |
|---|------|--------|
| BD1 | Все найдены | BatchResult.Deleted = N, Errors empty |
| BD2 | Частичный успех | Deleted = N-M, Errors contains M items |
| BD3 | Пустой массив | ValidationError |
| BD4 | Слишком большой массив | ValidationError |
| BD5 | Audit logged | Один audit record для batch |

**ImportEntries:**

| # | Тест | Assert |
|---|------|--------|
| IM1 | Happy path | ImportResult.Imported = N |
| IM2 | Дубликат в файле | Первый imported, второй skipped |
| IM3 | Дубликат с existing entry | Skipped |
| IM4 | Пустой text после нормализации | Skipped |
| IM5 | Chunk failure | Chunk errors recorded, next chunks processed |
| IM6 | Лимит превышен | ValidationError |
| IM7 | Пустой Items | ValidationError |

**ExportEntries:**

| # | Тест | Assert |
|---|------|--------|
| EX1 | Happy path | ExportResult.Items contains all entries with senses/translations |
| EX2 | Пустой словарь | Items = [] |
| EX3 | COALESCE applied | Resolved values in export (not raw NULLs) |

---

## 7. Файловая структура

```
internal/service/dictionary/
├── service.go           # Service struct, конструктор, приватные интерфейсы
│                        # SearchCatalog, PreviewRefEntry, CreateEntryFromCatalog,
│                        # CreateEntryCustom, FindEntries, GetEntry, UpdateNotes,
│                        # DeleteEntry, FindDeletedEntries, RestoreEntry,
│                        # BatchDeleteEntries, ImportEntries, ExportEntries
├── input.go             # CreateFromCatalogInput, CreateCustomInput, FindInput,
│                        # UpdateNotesInput, ImportInput + Validate()
├── result.go            # FindResult, BatchResult, ImportResult, ExportResult,
│                        # ExportItem, ExportSense, ExportExample
└── service_test.go      # Все тесты из секции 6 (~60 тестов)
```

---

## 8. Взаимодействие с другими сервисами

### 8.1. RefCatalogService (зависимость)

DictionaryService вызывает RefCatalogService для:
- SearchCatalog → `refCatalog.Search`
- PreviewRefEntry → `refCatalog.GetOrFetchEntry`
- CreateFromCatalog → `refCatalog.GetRefEntry` (получить дерево для ранее загруженного ref_entry)

RefCatalogService отвечает за взаимодействие с external API. DictionaryService **не** вызывает external providers напрямую.

### 8.2. ContentService (нет зависимости)

DictionaryService создаёт начальный контент (senses, translations, examples) при добавлении слова. Дальнейшее редактирование контента — через ContentService. Между ними нет прямой зависимости — оба работают с repo напрямую.

### 8.3. StudyService (нет зависимости)

DictionaryService может создавать карточку при добавлении слова (CreateCard=true). Для этого вызывает cardRepo напрямую, не StudyService. SRS-логика не задействована — карточка создаётся со статусом NEW и default ease.

### 8.4. Transport Layer (GraphQL)

Все операции доступны через GraphQL mutations и queries. Resolver определяет свой узкий интерфейс:

```
type dictionaryService interface {
    SearchCatalog(ctx, query, limit) → ([]RefEntry, error)
    PreviewRefEntry(ctx, text) → (*RefEntry, error)
    CreateEntryFromCatalog(ctx, input) → (*Entry, error)
    CreateEntryCustom(ctx, input) → (*Entry, error)
    FindEntries(ctx, input) → (*FindResult, error)
    GetEntry(ctx, entryID) → (*Entry, error)
    UpdateNotes(ctx, input) → (*Entry, error)
    DeleteEntry(ctx, entryID) → error
    FindDeletedEntries(ctx, limit, offset) → ([]Entry, int, error)
    RestoreEntry(ctx, entryID) → (*Entry, error)
    BatchDeleteEntries(ctx, entryIDs) → (*BatchResult, error)
    ImportEntries(ctx, input) → (*ImportResult, error)
    ExportEntries(ctx) → (*ExportResult, error)
}
```
