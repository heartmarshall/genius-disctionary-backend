# Фаза 6: Content сервис


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (интерфейсы потребителем), §2 (обработка ошибок), §3 (валидация), §4 (контекст и user identity), §5 (логирование), §6 (аудит), §7 (тестирование, moq) |
| `services/service_layer_spec_v4.md` | §2 (структура пакетов), §3 (паттерны), §4 (аудит: SENSE — translations/examples аудитируются как UPDATE на parent sense), §5 (application-level limits), §6 (карта сервисов: ContentService), §7 (тестирование) |
| `services/content_service_spec_v4.md` | Все секции — полная спецификация Content Service: зависимости, ownership chain, 14 операций, валидация, error scenarios, ~50 тестов |
| `services/business_scenarios_v4.md` | C1–C12 (Content) |
| `data_model_v4.md` | §4 (senses, translations, examples, user_images), §5 (COALESCE pattern) |
| `repo/repo_layer_spec_v4.md` | §5 (COALESCE), §12 (Reorder pattern), §19.3–19.4 (content repos) |

---

## Пре-условия (из Фазы 1)

Перед началом Фазы 6 должны быть готовы:

- Domain-модели: `Entry`, `Sense`, `Translation`, `Example`, `UserImage` (`internal/domain/entry.go`)
- Domain-модели: `AuditRecord` (`internal/domain/organization.go`)
- Domain errors: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized` (`internal/domain/errors.go`)
- `ValidationError`, `FieldError`, `NewValidationError()` (`internal/domain/errors.go`)
- Enums: `PartOfSpeech`, `EntityType` (ENTRY, SENSE), `AuditAction` (CREATE, UPDATE, DELETE) (`internal/domain/enums.go`)
- Context helpers: `ctxutil.UserIDFromCtx(ctx) → (uuid.UUID, bool)` (`pkg/ctxutil/`)

> **Важно:** Фаза 6 **не зависит** от Фаз 2, 3, 4, 5. Все зависимости на репозитории, TxManager мокаются в unit-тестах. Content Service не зависит от Dictionary Service и может разрабатываться параллельно.

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | Лимиты | Захардкожены как константы в пакете: `MaxSensesPerEntry=20`, `MaxTranslationsPerSense=20`, `MaxExamplesPerSense=50` |
| 2 | Аудит translations/examples | Аудитируются как UPDATE на parent Sense (EntityType=SENSE, EntityID=senseID). Не раздуваем audit log записями для каждого перевода/примера |
| 3 | Аудит user images | **Не аудитируются** — lightweight контент |
| 4 | Ownership error | Если элемент существует, но принадлежит чужому entry или entry soft-deleted → всегда `ErrNotFound` (не `ErrForbidden`). Не раскрываем существование чужих данных |
| 5 | Лимит user images | На MVP нет лимита на user images per entry |
| 6 | Reorder partial list | Клиент может отправить не все элементы. Repo обновляет только переданные. Остальные сохраняют свои позиции |
| 7 | source_slug | Все custom-операции Content Service используют `source_slug = "user"` |
| 8 | Моки | `moq` (code generation) — моки генерируются из приватных интерфейсов в `_test.go` файлы |
| 9 | Mock TxManager | `RunInTx(ctx, fn)` просто вызывает `fn(ctx)` без реальной транзакции |
| 10 | ReorderItem | Shared тип `struct { ID uuid.UUID; Position int }` — используется всеми Reorder-операциями. Определяется в `service.go` |
| 11 | UpdateSense partial update | Nil поле означает "не менять". Отличается от domain COALESCE-семантики (NULL = "наследовать из ref"). На уровне repo Update устанавливает только non-nil поля |
| 12 | DeleteSense CASCADE | Translations и examples удаляются автоматически FK ON DELETE CASCADE. Сервис не удаляет их явно |
| 13 | Position gaps | Удаление элемента оставляет gap в позициях. Это ожидаемое поведение — gap не влияет на ORDER BY |
| 14 | Транзакции для AddUserImage/DeleteUserImage | Не нужны — одна операция. Нет audit |

---

## Задачи

### TASK-6.1: Content Service — Foundation

**Зависит от:** Фаза 1 (domain models)

**Контекст:**
- `services/content_service_spec_v4.md` — §2 (зависимости), §3 (ownership helpers), §5 (input validation), §8 (файловая структура)
- `services/service_layer_spec_v4.md` — §3 (паттерны)

**Что сделать:**

Создать пакет `internal/service/content/` с foundation-компонентами: Service struct, приватные интерфейсы, константы, ownership helpers, input-структуры с валидацией.

**Файловая структура:**

```
internal/service/content/
├── service.go           # Service struct, конструктор, приватные интерфейсы,
│                        #   константы, ReorderItem, ownership helpers
└── input.go             # Все input-структуры + Validate()
```

**`service.go` — приватные интерфейсы:**

```go
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
    Reorder(ctx context.Context, items []ReorderItem) error
}

type translationRepo interface {
    GetByID(ctx context.Context, translationID uuid.UUID) (*domain.Translation, error)
    GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Translation, error)
    CountBySense(ctx context.Context, senseID uuid.UUID) (int, error)
    CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
    Update(ctx context.Context, translationID uuid.UUID, text string) (*domain.Translation, error)
    Delete(ctx context.Context, translationID uuid.UUID) error
    Reorder(ctx context.Context, items []ReorderItem) error
}

type exampleRepo interface {
    GetByID(ctx context.Context, exampleID uuid.UUID) (*domain.Example, error)
    GetBySenseID(ctx context.Context, senseID uuid.UUID) ([]domain.Example, error)
    CountBySense(ctx context.Context, senseID uuid.UUID) (int, error)
    CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
    Update(ctx context.Context, exampleID uuid.UUID, sentence string, translation *string) (*domain.Example, error)
    Delete(ctx context.Context, exampleID uuid.UUID) error
    Reorder(ctx context.Context, items []ReorderItem) error
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
```

**Константы:**

```go
const (
    MaxSensesPerEntry       = 20
    MaxTranslationsPerSense = 20
    MaxExamplesPerSense     = 50
)
```

**ReorderItem:**

```go
type ReorderItem struct {
    ID       uuid.UUID
    Position int
}
```

**Конструктор:**

```go
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

func NewService(
    logger       *slog.Logger,
    entries      entryRepo,
    senses       senseRepo,
    translations translationRepo,
    examples     exampleRepo,
    images       imageRepo,
    audit        auditRepo,
    tx           txManager,
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
```

**Ownership helpers:**

Пять приватных методов для проверки ownership chain. Каждый загружает элемент и проверяет цепочку до user через parent:

```go
// checkEntryOwnership: проверить что entry принадлежит пользователю.
func (s *Service) checkEntryOwnership(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
    entry, err = entryRepo.GetByID(ctx, userID, entryID)
    └─ ErrNotFound → return ErrNotFound
    return entry, nil

// checkSenseOwnership: загрузить sense, затем проверить ownership его entry.
func (s *Service) checkSenseOwnership(ctx context.Context, userID uuid.UUID, senseID uuid.UUID) (*domain.Sense, *domain.Entry, error)
    sense, err = senseRepo.GetByID(ctx, senseID)
    └─ ErrNotFound → return ErrNotFound
    entry, err = s.checkEntryOwnership(ctx, userID, sense.EntryID)
    └─ ErrNotFound → return ErrNotFound (sense существует, но entry чужой/deleted)
    return sense, entry, nil

// checkTranslationOwnership: загрузить translation → sense → entry → user.
func (s *Service) checkTranslationOwnership(ctx context.Context, userID uuid.UUID, translationID uuid.UUID) (*domain.Translation, *domain.Entry, error)
    translation, err = translationRepo.GetByID(ctx, translationID)
    └─ ErrNotFound → return ErrNotFound
    _, entry, err = s.checkSenseOwnership(ctx, userID, translation.SenseID)
    return translation, entry, err

// checkExampleOwnership: загрузить example → sense → entry → user.
func (s *Service) checkExampleOwnership(ctx context.Context, userID uuid.UUID, exampleID uuid.UUID) (*domain.Example, *domain.Entry, error)
    // Аналогично checkTranslationOwnership

// checkUserImageOwnership: загрузить image → entry → user.
func (s *Service) checkUserImageOwnership(ctx context.Context, userID uuid.UUID, imageID uuid.UUID) (*domain.UserImage, *domain.Entry, error)
    image, err = imageRepo.GetUserByID(ctx, imageID)
    └─ ErrNotFound → return ErrNotFound
    entry, err = s.checkEntryOwnership(ctx, userID, image.EntryID)
    return image, entry, err
```

Это 2–3 SQL-запроса на каждую мутацию. Для MVP допустимо.

---

**`input.go` — Input-структуры и валидация:**

Все input-структуры и правила валидации описаны в `content_service_spec_v4.md` §5.

| Input | Validate() | Ключевые правила |
|-------|-----------|-----------------|
| `AddSenseInput` | §5.1 | EntryID required, Definition max 2000, CEFRLevel max 10, Translations max 20, each translation required & max 500 |
| `UpdateSenseInput` | §5.2 | SenseID required, Definition max 2000, CEFRLevel max 10 |
| `ReorderSensesInput` | §5.3 | EntryID required, Items non-empty & max 50, each item ID required & position >= 0 |
| `AddTranslationInput` | §5.4 | SenseID required, Text required (after trim) & max 500 |
| `UpdateTranslationInput` | §5.5 | TranslationID required, Text required (after trim) & max 500 |
| `ReorderTranslationsInput` | §5.3 | SenseID required, Items non-empty & max 50, each item ID required & position >= 0 |
| `AddExampleInput` | §5.6 | SenseID required, Sentence required (after trim) & max 2000, Translation max 2000 |
| `UpdateExampleInput` | §5.7 | ExampleID required, Sentence required (after trim) & max 2000, Translation max 2000 |
| `ReorderExamplesInput` | §5.3 | SenseID required, Items non-empty & max 50, each item ID required & position >= 0 |
| `AddUserImageInput` | §5.8 | EntryID required, URL required & valid HTTP(S) & max 2000, Caption max 500 |

Каждый `Validate()` собирает **все** ошибки (не fail-fast). Возвращает `*domain.ValidationError`.

**AddSenseInput:**

```go
type AddSenseInput struct {
    EntryID      uuid.UUID
    Definition   *string
    PartOfSpeech *domain.PartOfSpeech
    CEFRLevel    *string
    Translations []string
}

func (i AddSenseInput) Validate() error {
    var errs []domain.FieldError
    if i.EntryID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
    }
    if i.Definition != nil && len(*i.Definition) > 2000 {
        errs = append(errs, domain.FieldError{Field: "definition", Message: "too long (max 2000)"})
    }
    if i.CEFRLevel != nil && len(*i.CEFRLevel) > 10 {
        errs = append(errs, domain.FieldError{Field: "cefr_level", Message: "too long"})
    }
    if len(i.Translations) > 20 {
        errs = append(errs, domain.FieldError{Field: "translations", Message: "too many (max 20)"})
    }
    for idx, tr := range i.Translations {
        trimmed := strings.TrimSpace(tr)
        if trimmed == "" {
            errs = append(errs, domain.FieldError{
                Field: fmt.Sprintf("translations[%d]", idx), Message: "required",
            })
        }
        if len(tr) > 500 {
            errs = append(errs, domain.FieldError{
                Field: fmt.Sprintf("translations[%d]", idx), Message: "too long (max 500)",
            })
        }
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**UpdateSenseInput:**

```go
type UpdateSenseInput struct {
    SenseID      uuid.UUID
    Definition   *string
    PartOfSpeech *domain.PartOfSpeech
    CEFRLevel    *string
}
```

**ReorderSensesInput / ReorderTranslationsInput / ReorderExamplesInput:**

```go
type ReorderSensesInput struct {
    EntryID uuid.UUID
    Items   []ReorderItem
}

type ReorderTranslationsInput struct {
    SenseID uuid.UUID
    Items   []ReorderItem
}

type ReorderExamplesInput struct {
    SenseID uuid.UUID
    Items   []ReorderItem
}
```

Все три Reorder input валидируются одинаково:
- ParentID required
- len(Items) > 0 и <= 50
- Каждый item: ID required, Position >= 0

**AddTranslationInput / UpdateTranslationInput:**

```go
type AddTranslationInput struct {
    SenseID uuid.UUID
    Text    string
}

type UpdateTranslationInput struct {
    TranslationID uuid.UUID
    Text          string
}
```

**AddExampleInput / UpdateExampleInput:**

```go
type AddExampleInput struct {
    SenseID     uuid.UUID
    Sentence    string
    Translation *string
}

type UpdateExampleInput struct {
    ExampleID   uuid.UUID
    Sentence    string
    Translation *string   // nil = убрать перевод примера
}
```

**AddUserImageInput:**

```go
type AddUserImageInput struct {
    EntryID uuid.UUID
    URL     string
    Caption *string
}
```

URL валидация: `isValidHTTPURL(url)` — проверяет scheme http/https.

**Acceptance criteria:**
- [ ] `internal/service/content/service.go` создан
- [ ] 7 приватных интерфейсов: entryRepo, senseRepo, translationRepo, exampleRepo, imageRepo, auditRepo, txManager
- [ ] Константы: `MaxSensesPerEntry=20`, `MaxTranslationsPerSense=20`, `MaxExamplesPerSense=50`
- [ ] `ReorderItem` struct определён
- [ ] Конструктор `NewService` с 8 параметрами, логгер `"service", "content"`
- [ ] 5 ownership helpers: checkEntryOwnership, checkSenseOwnership, checkTranslationOwnership, checkExampleOwnership, checkUserImageOwnership
- [ ] `internal/service/content/input.go` создан
- [ ] 10 input-структур с `Validate()` — собирают все ошибки (не fail-fast)
- [ ] `isValidHTTPURL` helper для валидации URL
- [ ] `go build ./...` компилируется
- [ ] `go vet ./internal/service/content/...` — без warnings

---

### TASK-6.2: Content Service — Sense Operations + Tests

**Зависит от:** TASK-6.1 (Service struct, interfaces, ownership helpers, input structs)

**Контекст:**
- `services/content_service_spec_v4.md` — §4.1 (AddSense), §4.2 (UpdateSense), §4.3 (DeleteSense), §4.4 (ReorderSenses)
- `services/service_layer_spec_v4.md` — §4 (аудит: audit SENSE entity)

**Что сделать:**

Реализовать 4 операции для senses: AddSense, UpdateSense, DeleteSense, ReorderSenses. Написать unit-тесты.

---

**Операция: AddSense(ctx, input AddSenseInput) → (*domain.Sense, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate() → ValidationError

3. entry, err = s.checkEntryOwnership(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. count = senseRepo.CountByEntry(ctx, input.EntryID)
   └─ count >= MaxSensesPerEntry → ValidationError("senses", "limit reached (20)")

5. txManager.RunInTx(ctx, func(ctx) error {
   5a. sense = senseRepo.CreateCustom(ctx, input.EntryID, input.Definition, input.PartOfSpeech, input.CEFRLevel, "user")
       // position auto-increment в repo: MAX(position)+1

   5b. Для каждого text в input.Translations:
       translationRepo.CreateCustom(ctx, sense.ID, text, "user")

   5c. auditRepo.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &sense.ID,
           Action:     domain.ActionCreate,
           Changes: map[string]any{
               "entry_id":           {"new": input.EntryID},
               "definition":         {"new": input.Definition},
               "translations_count": {"new": len(input.Translations)},
           },
       })
   })

6. return sense, nil
```

**Corner cases:**
- Sense без definition — допустимо (definition=nil)
- Sense без translations — допустимо (Translations=[])
- PartOfSpeech = nil — допустимо
- Entry soft-deleted → checkEntryOwnership вернёт ErrNotFound
- Position — repo автоматически назначает MAX(position)+1

---

**Операция: UpdateSense(ctx, input UpdateSenseInput) → (*domain.Sense, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate() → ValidationError

3. oldSense, entry, err = s.checkSenseOwnership(ctx, userID, input.SenseID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {
   4a. sense = senseRepo.Update(ctx, input.SenseID, input.Definition, input.PartOfSpeech, input.CEFRLevel)
       // ref_sense_id НЕ трогается — инвариант repo

   4b. auditRepo.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &input.SenseID,
           Action:     domain.ActionUpdate,
           Changes:    buildSenseChanges(oldSense, input),
       })
   })

5. return sense, nil
```

**Семантика nil:** Nil поле означает "не менять" (partial update). Если пользователь хочет вернуть наследование из каталога — это отдельная операция (не на MVP).

**Corner cases:**
- Все поля nil — UPDATE + audit выполняются (no-op, но допустимо)
- Sense без ref — работает аналогично

---

**Операция: DeleteSense(ctx, senseID uuid.UUID) → error**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. sense, entry, err = s.checkSenseOwnership(ctx, userID, senseID)
   └─ ErrNotFound → return ErrNotFound

3. txManager.RunInTx(ctx, func(ctx) error {
   3a. senseRepo.Delete(ctx, senseID)
       // CASCADE: translations, examples удаляются автоматически

   3b. auditRepo.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &senseID,
           Action:     domain.ActionDelete,
           Changes: map[string]any{
               "entry_id":   {"old": sense.EntryID},
               "definition": {"old": sense.Definition},
           },
       })
   })

4. return nil
```

**Corner cases:**
- Удаление последнего sense — допустимо
- CASCADE удаляет translations и examples автоматически
- Position gap — ожидаемое поведение

---

**Операция: ReorderSenses(ctx, input ReorderSensesInput) → error**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate() → ValidationError

3. entry, err = s.checkEntryOwnership(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. Валидация items:
   existingSenses = senseRepo.GetByEntryID(ctx, input.EntryID)
   existingIDs = set(existingSenses.IDs)

   Для каждого item в input.Items:
     if item.ID not in existingIDs → ValidationError("items", "sense does not belong to this entry: " + item.ID)

5. senseRepo.Reorder(ctx, input.Items)

6. return nil
```

**Corner cases:**
- Partial list (не все senses) — допустимо
- Duplicate positions в input — допустимо (ORDER BY по (position, id))
- ID из другого entry — ValidationError на шаге 4
- Пустой Items — ValidationError в Validate()

---

**Unit-тесты:**

Моки через moq. Mock TxManager: `fn(ctx)` pass-through.

**Ownership (тестируются через sense операции):**

| # | Тест | Assert |
|---|------|--------|
| OW1 | Entry принадлежит пользователю | Operation succeeds |
| OW2 | Entry не найден | ErrNotFound |
| OW3 | Entry soft-deleted | ErrNotFound |
| OW4 | Sense → entry чужой | ErrNotFound |
| OW5 | Translation → sense → entry чужой | ErrNotFound |

**AddSense:**

| # | Тест | Assert |
|---|------|--------|
| AS1 | Happy path — с definition и translations | Sense создан. Translations создаются. Audit logged |
| AS2 | Без definition | Sense создан с definition=nil |
| AS3 | Без translations | Sense создан. translationRepo.CreateCustom NOT called |
| AS4 | Лимит senses (20) | ValidationError "limit reached" |
| AS5 | Entry не найден | ErrNotFound |
| AS6 | Entry deleted | ErrNotFound |
| AS7 | Невалидный input | ValidationError |
| AS8 | Нет userID | ErrUnauthorized |
| AS9 | source_slug = "user" | CreateCustom called with sourceSlug="user" |

**UpdateSense:**

| # | Тест | Assert |
|---|------|--------|
| US1 | Изменить definition | senseRepo.Update called. Audit logged with old/new |
| US2 | Изменить pos | senseRepo.Update called with pos |
| US3 | Partial: только definition, pos не трогать | Update called. ref_sense_id NOT changed |
| US4 | Все поля nil | Update called (no-op but allowed) |
| US5 | Sense не найден | ErrNotFound |
| US6 | Sense чужого entry | ErrNotFound |

**DeleteSense:**

| # | Тест | Assert |
|---|------|--------|
| DS1 | Happy path | senseRepo.Delete called. Audit logged |
| DS2 | Последний sense entry | Delete succeeds (entry без senses допустим) |
| DS3 | Sense не найден | ErrNotFound |
| DS4 | Audit содержит definition | Changes include old definition |

**ReorderSenses:**

| # | Тест | Assert |
|---|------|--------|
| RS1 | Happy path | senseRepo.Reorder called with items |
| RS2 | Entry не найден | ErrNotFound |
| RS3 | Sense ID из другого entry | ValidationError "does not belong" |
| RS4 | Partial list (не все senses) | Reorder called only with provided items |
| RS5 | Пустой Items | ValidationError |

**Всего: ~24 тест-кейса (включая ownership)**

**Acceptance criteria:**
- [ ] `AddSense`: полный flow — validate → ownership check → limit check → tx(create sense + translations + audit)
- [ ] `AddSense`: source_slug = "user"
- [ ] `AddSense`: без definition — допустимо
- [ ] `AddSense`: без translations — translationRepo.CreateCustom не вызывается
- [ ] `UpdateSense`: partial update — nil поля не меняются
- [ ] `UpdateSense`: ref_sense_id не трогается
- [ ] `UpdateSense`: audit с buildSenseChanges (old/new)
- [ ] `DeleteSense`: CASCADE удаляет translations/examples
- [ ] `DeleteSense`: удаление последнего sense — допустимо
- [ ] `ReorderSenses`: валидация что sense IDs принадлежат entry
- [ ] `ReorderSenses`: partial list — допустимо
- [ ] Ownership helpers корректно работают для sense → entry → user chain
- [ ] Все мутации: ErrUnauthorized при отсутствии userID
- [ ] ~24 unit-теста покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/content/...` — все проходят
- [ ] `go vet ./internal/service/content/...` — без warnings

---

### TASK-6.3: Content Service — Translation & Example Operations + Tests

**Зависит от:** TASK-6.1 (Service struct, interfaces, ownership helpers, input structs)

> **Примечание:** TASK-6.3 **не зависит** от TASK-6.2 (Sense operations). Translation и Example операции используют те же ownership helpers из TASK-6.1 и могут разрабатываться параллельно с TASK-6.2.

**Контекст:**
- `services/content_service_spec_v4.md` — §4.5–§4.12 (Translation и Example операции)
- `services/service_layer_spec_v4.md` — §4 (аудит: translations/examples аудитируются как UPDATE на parent SENSE)

**Что сделать:**

Реализовать 8 операций: AddTranslation, UpdateTranslation, DeleteTranslation, ReorderTranslations, AddExample, UpdateExample, DeleteExample, ReorderExamples. Написать unit-тесты.

---

**Операция: AddTranslation(ctx, input AddTranslationInput) → (*domain.Translation, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. _, entry, err = s.checkSenseOwnership(ctx, userID, input.SenseID)
   └─ ErrNotFound → return ErrNotFound

4. count = translationRepo.CountBySense(ctx, input.SenseID)
   └─ count >= MaxTranslationsPerSense → ValidationError("translations", "limit reached (20)")

5. txManager.RunInTx(ctx, func(ctx) error {
   5a. translation = translationRepo.CreateCustom(ctx, input.SenseID, input.Text, "user")
   5b. auditRepo.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &input.SenseID,
           Action:     domain.ActionUpdate,
           Changes: map[string]any{
               "translation_added": {"new": input.Text},
           },
       })
   })

6. return translation, nil
```

**Audit note:** EntityType = SENSE, EntityID = senseID (не translationID).

---

**Операция: UpdateTranslation(ctx, input UpdateTranslationInput) → (*domain.Translation, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. oldTranslation, entry, err = s.checkTranslationOwnership(ctx, userID, input.TranslationID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {
   4a. translation = translationRepo.Update(ctx, input.TranslationID, input.Text)
       // ref_translation_id НЕ трогается
   4b. auditRepo.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntitySense,
           EntityID:   &oldTranslation.SenseID,
           Action:     domain.ActionUpdate,
           Changes: map[string]any{
               "translation_text": {"old": oldTranslation.Text, "new": input.Text},
           },
       })
   })

5. return translation, nil
```

**Corner cases:**
- Текст не изменился — UPDATE + audit всё равно выполняются
- Translation из каталога — ref_translation_id сохраняется

---

**Операция: DeleteTranslation(ctx, translationID uuid.UUID) → error**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. translation, entry, err = s.checkTranslationOwnership(ctx, userID, translationID)
   └─ ErrNotFound → return ErrNotFound

3. txManager.RunInTx(ctx, func(ctx) error {
   3a. translationRepo.Delete(ctx, translationID)
   3b. auditRepo.Log(ctx, ...) // EntitySense, ActionUpdate
   })

4. return nil
```

**Corner case:** Удаление последнего перевода — допустимо.

---

**Операция: ReorderTranslations(ctx, input ReorderTranslationsInput) → error**

Flow аналогичен ReorderSenses, но:
- Ownership check через checkSenseOwnership
- Валидация items через translationRepo.GetBySenseID
- Repo call: translationRepo.Reorder

---

**Операция: AddExample(ctx, input AddExampleInput) → (*domain.Example, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. _, entry, err = s.checkSenseOwnership(ctx, userID, input.SenseID)
   └─ ErrNotFound → return ErrNotFound

4. count = exampleRepo.CountBySense(ctx, input.SenseID)
   └─ count >= MaxExamplesPerSense → ValidationError("examples", "limit reached (50)")

5. txManager.RunInTx(ctx, func(ctx) error {
   5a. example = exampleRepo.CreateCustom(ctx, input.SenseID, input.Sentence, input.Translation, "user")
   5b. auditRepo.Log(ctx, ...) // EntitySense, ActionUpdate
   })

6. return example, nil
```

---

**Операция: UpdateExample(ctx, input UpdateExampleInput) → (*domain.Example, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. oldExample, entry, err = s.checkExampleOwnership(ctx, userID, input.ExampleID)
   └─ ErrNotFound → return ErrNotFound

4. txManager.RunInTx(ctx, func(ctx) error {
   4a. example = exampleRepo.Update(ctx, input.ExampleID, input.Sentence, input.Translation)
   4b. auditRepo.Log(ctx, ...) // EntitySense, ActionUpdate
   })

5. return example, nil
```

**Corner case:** Translation = nil → перевод примера убирается (set translation = NULL в БД). Для examples.translation NULL означает "нет перевода" (не COALESCE-семантика).

---

**Операция: DeleteExample(ctx, exampleID uuid.UUID) → error**

Flow аналогичен DeleteTranslation: ownership check → tx(delete + audit).

---

**Операция: ReorderExamples(ctx, input ReorderExamplesInput) → error**

Flow аналогичен ReorderTranslations: ownership через checkSenseOwnership, валидация items через exampleRepo.GetBySenseID.

---

**Unit-тесты:**

**AddTranslation:**

| # | Тест | Assert |
|---|------|--------|
| AT1 | Happy path | Translation создан. Audit на parent sense |
| AT2 | Лимит translations (20) | ValidationError "limit reached" |
| AT3 | Sense не найден | ErrNotFound |
| AT4 | Sense чужого entry | ErrNotFound |

**UpdateTranslation:**

| # | Тест | Assert |
|---|------|--------|
| UT1 | Happy path | translationRepo.Update called. ref_translation_id не трогается |
| UT2 | Translation не найден | ErrNotFound |
| UT3 | Audit содержит old/new text | Changes: {"translation_text": {"old": ..., "new": ...}} |

**DeleteTranslation:**

| # | Тест | Assert |
|---|------|--------|
| DT1 | Happy path | Deleted. Audit logged |
| DT2 | Последний translation sense | Delete succeeds |
| DT3 | Translation не найден | ErrNotFound |

**ReorderTranslations:**

| # | Тест | Assert |
|---|------|--------|
| RT1 | Happy path | translationRepo.Reorder called |
| RT2 | Translation ID из другого sense | ValidationError |

**AddExample:**

| # | Тест | Assert |
|---|------|--------|
| AE1 | Happy path — sentence + translation | Example создан |
| AE2 | Без translation | Example создан с translation=nil |
| AE3 | Лимит examples (50) | ValidationError |
| AE4 | Sense не найден | ErrNotFound |

**UpdateExample:**

| # | Тест | Assert |
|---|------|--------|
| UE1 | Изменить sentence | exampleRepo.Update called |
| UE2 | Translation = nil (убрать перевод) | Update called with translation=nil |
| UE3 | Example не найден | ErrNotFound |

**DeleteExample:**

| # | Тест | Assert |
|---|------|--------|
| DEX1 | Happy path | Deleted. Audit logged |
| DEX2 | Example не найден | ErrNotFound |

**ReorderExamples:**

| # | Тест | Assert |
|---|------|--------|
| REX1 | Happy path | exampleRepo.Reorder called |
| REX2 | Example ID из другого sense | ValidationError |

**Всего: ~23 тест-кейса**

**Acceptance criteria:**
- [ ] **AddTranslation**: ownership через checkSenseOwnership, лимит 20, audit на parent SENSE
- [ ] **UpdateTranslation**: ref_translation_id не трогается, audit с old/new text
- [ ] **DeleteTranslation**: ownership check через full chain, audit
- [ ] **ReorderTranslations**: валидация что translation IDs принадлежат sense
- [ ] **AddExample**: ownership через checkSenseOwnership, лимит 50, source_slug="user"
- [ ] **UpdateExample**: translation=nil → убрать перевод (не COALESCE-семантика)
- [ ] **DeleteExample**: ownership check, audit
- [ ] **ReorderExamples**: валидация items через exampleRepo.GetBySenseID
- [ ] Все audit записи: EntityType=SENSE, EntityID=senseID (не child ID)
- [ ] ~23 unit-теста покрывают все сценарии
- [ ] `go test ./internal/service/content/...` — все проходят

---

### TASK-6.4: Content Service — UserImage Operations + Tests

**Зависит от:** TASK-6.1 (Service struct, interfaces, ownership helpers, input structs)

> **Примечание:** TASK-6.4 **не зависит** от TASK-6.2 и TASK-6.3. UserImage операции используют только checkEntryOwnership и checkUserImageOwnership из TASK-6.1.

**Контекст:**
- `services/content_service_spec_v4.md` — §4.13 (AddUserImage), §4.14 (DeleteUserImage)

**Что сделать:**

Реализовать 2 операции: AddUserImage, DeleteUserImage. Написать unit-тесты.

---

**Операция: AddUserImage(ctx, input AddUserImageInput) → (*domain.UserImage, error)**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. entry, err = s.checkEntryOwnership(ctx, userID, input.EntryID)
   └─ ErrNotFound → return ErrNotFound

4. image = imageRepo.CreateUser(ctx, input.EntryID, input.URL, input.Caption)

5. return image, nil
```

**Нет транзакции** (одна операция). **Нет audit** (user images — lightweight контент).

**Corner cases:**
- URL validation — проверяется в Validate() (HTTP(S) URL)
- Кто хостит изображения — вне scope (URL от клиента)
- Нет лимита на images per entry (MVP)

---

**Операция: DeleteUserImage(ctx, imageID uuid.UUID) → error**

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. image, entry, err = s.checkUserImageOwnership(ctx, userID, imageID)
   └─ ErrNotFound → return ErrNotFound

3. imageRepo.DeleteUser(ctx, imageID)

4. return nil
```

**Нет audit.** Нет транзакции.

---

**Unit-тесты:**

**AddUserImage:**

| # | Тест | Assert |
|---|------|--------|
| AI1 | Happy path | Image created. No audit |
| AI2 | Entry не найден | ErrNotFound |
| AI3 | Невалидный URL | ValidationError |
| AI4 | С caption | Image.Caption set |
| AI5 | Без caption | Image.Caption = nil |

**DeleteUserImage:**

| # | Тест | Assert |
|---|------|--------|
| DI1 | Happy path | Deleted. No audit |
| DI2 | Image не найден | ErrNotFound |
| DI3 | Image чужого entry | ErrNotFound |

**Всего: ~8 тест-кейсов**

**Acceptance criteria:**
- [ ] **AddUserImage**: ownership check → create. Без транзакции, без audit
- [ ] **AddUserImage**: URL validation в Validate() — HTTP(S), max 2000
- [ ] **DeleteUserImage**: ownership через checkUserImageOwnership → delete. Без audit
- [ ] Image чужого entry → ErrNotFound (не ErrForbidden)
- [ ] ~8 unit-тестов покрывают все сценарии
- [ ] `go test ./internal/service/content/...` — все проходят

---

## Сводка зависимостей задач

```
TASK-6.1 (Foundation) ──┬──→ TASK-6.2 (Sense Operations)
                        ├──→ TASK-6.3 (Translation & Example Operations)
                        └──→ TASK-6.4 (UserImage Operations)
```

Детализация:
- **TASK-6.2** (Sense Operations) зависит от: TASK-6.1 (Service struct, interfaces, ownership helpers, input structs)
- **TASK-6.3** (Translation & Example Operations) зависит от: TASK-6.1. **Не зависит** от TASK-6.2
- **TASK-6.4** (UserImage Operations) зависит от: TASK-6.1. **Не зависит** от TASK-6.2 и TASK-6.3
- TASK-6.2, TASK-6.3, TASK-6.4 не имеют взаимных зависимостей

---

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-6.1 (Foundation) |
| 2 | TASK-6.2 (Sense Operations), TASK-6.3 (Translation & Example Operations), TASK-6.4 (UserImage Operations) |

> При полной параллелизации — **2 sequential волны**. Волна 2 — до 3 задач параллельно.

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] **Foundation:**
  - [ ] Service struct с 7 приватными интерфейсами
  - [ ] Конструктор с 8 параметрами, логгер "service", "content"
  - [ ] Константы: MaxSensesPerEntry=20, MaxTranslationsPerSense=20, MaxExamplesPerSense=50
  - [ ] ReorderItem shared тип
  - [ ] 5 ownership helpers корректно проверяют chain
  - [ ] 10 input-структур с Validate() — собирают все ошибки
- [ ] **Sense Operations** — все 4 операции реализованы:
  - [ ] AddSense: создание с translations, лимит, source_slug="user"
  - [ ] UpdateSense: partial update, ref_sense_id сохраняется
  - [ ] DeleteSense: CASCADE удаляет children, audit logged
  - [ ] ReorderSenses: валидация ownership sense IDs
- [ ] **Translation Operations** — все 4 операции реализованы:
  - [ ] AddTranslation: лимит 20, audit на parent SENSE
  - [ ] UpdateTranslation: ref_translation_id сохраняется
  - [ ] DeleteTranslation: ownership full chain
  - [ ] ReorderTranslations: валидация ownership
- [ ] **Example Operations** — все 4 операции реализованы:
  - [ ] AddExample: лимит 50, audit на parent SENSE
  - [ ] UpdateExample: translation=nil → убрать перевод
  - [ ] DeleteExample: ownership full chain
  - [ ] ReorderExamples: валидация ownership
- [ ] **UserImage Operations** — обе операции реализованы:
  - [ ] AddUserImage: без транзакции, без audit
  - [ ] DeleteUserImage: без audit
- [ ] Все ownership errors → ErrNotFound (не ErrForbidden)
- [ ] Audit: SENSE entity для sense/translation/example мутаций. Нет audit для images
- [ ] ~55 unit-тестов покрывают все сценарии из spec §7.2
- [ ] Моки сгенерированы через `moq` из приватных интерфейсов
- [ ] Все acceptance criteria всех 4 задач выполнены
