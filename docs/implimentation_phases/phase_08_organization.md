# Фаза 8: Сервисы организации (Topic + Inbox)


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (интерфейсы потребителем), §2 (обработка ошибок), §3 (валидация), §4 (контекст и user identity), §5 (логирование), §6 (аудит), §7 (тестирование, moq) |
| `services/service_layer_spec_v4.md` | §2 (структура пакетов: topic/, inbox/), §3 (паттерны), §4 (аудит: TOPIC), §5 (application-level limits: topics 100, inbox 500), §6 (карта сервисов: TopicService standalone, InboxService standalone), §7 (тестирование) |
| `services/topic_service_spec_v4.md` | Все секции — полная спецификация Topic Service: 8 операций (T1–T8), 42 теста, лимиты, аудит, M2M link/unlink/batch |
| `services/inbox_service_spec_v4.md` | Все секции — полная спецификация Inbox Service: 5 операций (I1–I5), 29 тестов, лимиты, без аудита |
| `services/business_scenarios_v4.md` | T1–T8 (Topics), I1–I5 (Inbox) |
| `data_model_v4.md` | §6 (topics, entry_topics, inbox_items) |

---

## Пре-условия (из Фазы 1)

Перед началом Фазы 8 должны быть готовы:

- Domain-модели: `Topic` с `EntryCount` computed field (`internal/domain/organization.go`)
- Domain-модели: `InboxItem` с `Text`, `Context` (`internal/domain/organization.go`)
- Domain-модели: `AuditRecord` (`internal/domain/organization.go`)
- Domain errors: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized` (`internal/domain/errors.go`)
- `ValidationError`, `FieldError`, `NewValidationError()` (`internal/domain/errors.go`)
- Enums: `EntityType` (включая `TOPIC`), `AuditAction` (CREATE, UPDATE, DELETE) (`internal/domain/enums.go`)
- Context helpers: `ctxutil.UserIDFromCtx(ctx) → (uuid.UUID, bool)` (`pkg/ctxutil/`)

> **Важно:** Фаза 8 **не зависит** от Фаз 2, 3, 4, 5, 6, 7. Все зависимости на репозитории и TxManager мокаются в unit-тестах. Topic Service и Inbox Service — standalone-сервисы без межсервисных зависимостей. Могут разрабатываться параллельно с любыми другими сервисами.

> **Важно:** Topic Service и Inbox Service **не зависят друг от друга** и могут разрабатываться полностью параллельно.

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | Темы — плоские или иерархические? | **Плоские**. Нет parent-child. Если нужна иерархия — пользователь именует: «IELTS», «IELTS / Writing». На 100 тем иерархия = overkill |
| 2 | Link/unlink аудитируются? | **Нет**. Это лёгкие M2M операции. Аудит строки в 32 байта раздувает audit_log без ценности |
| 3 | CRUD тем аудитируются? | **Да**. Create, Update, Delete — бизнес-значимые операции. EntityType = TOPIC |
| 4 | Inbox items аудитируются? | **Нет**. Временные заметки, создаются/удаляются часто. Нет update-операций |
| 5 | Inbox items — immutable? | **Да**. Нет Update. Если нужно исправить — удали и создай заново |
| 6 | Inbox — soft delete? | **Нет**. Hard delete. Удалённый item никому не нужен. Нет сценария восстановления |
| 7 | Имена тем case-sensitive? | **Да**. Unique constraint по raw name. «Food» и «food» — две темы. Проще в реализации |
| 8 | Дубли в inbox? | **Допускаются**. Одно слово с разным контекстом — два разных item'а |
| 9 | Нормализация текста в inbox? | **Нет**. В отличие от `entries.text_normalized`, inbox хранит as-is. Пользователь может записать фразу |
| 10 | EntryCount в ListTopics | В одном запросе через LEFT JOIN + COUNT. Не N+1 запросов |
| 11 | BatchLink — partial success? | **Да**. Несуществующие entries пропускаются. Уже привязанные — ON CONFLICT DO NOTHING. Возвращает linked/skipped counts |
| 12 | Моки | `moq` (code generation) — моки генерируются из приватных интерфейсов в `_test.go` файлы |
| 13 | Mock TxManager | `RunInTx(ctx, fn)` просто вызывает `fn(ctx)` без реальной транзакции |
| 14 | TopicUpdateParams | Domain-тип для partial update. `nil` = не менять. Для Description: `ptr("")` = очистить (NULL в БД) |
| 15 | Пустая строка в description при Create | Трактуется как nil (нет описания). `trimOrNil` helper |
| 16 | Inbox — конвертация в entry | **Не** ответственность Inbox Service. Клиент делает отдельные вызовы: GetItem → RefCatalog.Search → Dictionary.CreateEntry → DeleteItem |
| 17 | Link — без транзакции | Одна INSERT-операция, атомарна сама по себе. Нет audit для link |
| 18 | Unlink — без проверки entry | Unlink не требует проверки существования entry. Если entry удалён — CASCADE уже удалил связь |

---

## Задачи

### TASK-8.1: Domain Model Addition + Topic Service Foundation

**Зависит от:** Фаза 1 (domain models)

**Контекст:**
- `services/topic_service_spec_v4.md` — §2 (структура пакета), §3 (зависимости), §4 (domain model), §7 (валидация)
- `services/service_layer_spec_v4.md` — §3 (паттерны)
- Текущий `internal/domain/organization.go` — Topic и InboxItem уже существуют, но `TopicUpdateParams` отсутствует

**Что сделать:**

Добавить `TopicUpdateParams` в domain, создать пакет `internal/service/topic/` с foundation-компонентами: Service struct, приватные интерфейсы, input-структуры с валидацией, helper-функции.

**Файловая структура:**

```
internal/service/topic/
├── service.go          # Service struct, конструктор, приватные интерфейсы, helpers
├── input.go            # Все input-структуры + Validate()
└── service_test.go     # (TASK-8.2, TASK-8.3)
```

---

#### 1. Добавить `TopicUpdateParams`

**Файл:** `internal/domain/organization.go`

```go
// TopicUpdateParams holds fields for partial topic update.
// nil = don't change. For Description: ptr("") = clear (set NULL in DB).
type TopicUpdateParams struct {
    Name        *string // nil = don't change; must be non-empty if present
    Description *string // nil = don't change; ptr("") = clear
}
```

---

#### 2. `service.go` — приватные интерфейсы

```go
package topic

import (
    "context"
    "log/slog"

    "github.com/google/uuid"
    "github.com/heartmarshall/myenglish-backend/internal/domain"
)

type topicRepo interface {
    Create(ctx context.Context, userID uuid.UUID, topic *domain.Topic) (*domain.Topic, error)
    GetByID(ctx context.Context, userID, topicID uuid.UUID) (*domain.Topic, error)
    Update(ctx context.Context, userID, topicID uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error)
    Delete(ctx context.Context, userID, topicID uuid.UUID) error
    List(ctx context.Context, userID uuid.UUID) ([]*domain.Topic, error)
    Count(ctx context.Context, userID uuid.UUID) (int, error)

    // M2M: entry ↔ topic
    LinkEntry(ctx context.Context, entryID, topicID uuid.UUID) error
    UnlinkEntry(ctx context.Context, entryID, topicID uuid.UUID) error
    BatchLinkEntries(ctx context.Context, entryIDs []uuid.UUID, topicID uuid.UUID) (int, error)

    // M2M read
    GetTopicsByEntryID(ctx context.Context, entryID uuid.UUID) ([]*domain.Topic, error)
    GetEntryIDsByTopicID(ctx context.Context, topicID uuid.UUID) ([]uuid.UUID, error)
    CountEntriesByTopicID(ctx context.Context, topicID uuid.UUID) (int, error)
}

type entryRepo interface {
    GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
    ExistByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}

type auditLogger interface {
    Log(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

---

#### 3. `service.go` — конструктор и constants

```go
const (
    MaxTopicsPerUser = 100
)

type Service struct {
    topics  topicRepo
    entries entryRepo
    audit   auditLogger
    tx      txManager
    log     *slog.Logger
}

func NewService(
    log     *slog.Logger,
    topics  topicRepo,
    entries entryRepo,
    audit   auditLogger,
    tx      txManager,
) *Service {
    return &Service{
        topics:  topics,
        entries: entries,
        audit:   audit,
        tx:      tx,
        log:     log.With("service", "topic"),
    }
}
```

---

#### 4. `service.go` — helper-функции

```go
// trimOrNil trims whitespace. Returns nil if result is empty.
func trimOrNil(s *string) *string {
    if s == nil {
        return nil
    }
    trimmed := strings.TrimSpace(*s)
    if trimmed == "" {
        return nil
    }
    return &trimmed
}

// ptr returns a pointer to the given string.
func ptr(s string) *string {
    return &s
}
```

---

#### 5. `input.go` — Input-структуры с валидацией

**CreateTopicInput:**

```go
type CreateTopicInput struct {
    Name        string
    Description *string
}

func (i CreateTopicInput) Validate() error {
    var errs []domain.FieldError

    name := strings.TrimSpace(i.Name)
    if name == "" {
        errs = append(errs, domain.FieldError{Field: "name", Message: "required"})
    }
    if len(name) > 100 {
        errs = append(errs, domain.FieldError{Field: "name", Message: "max 100 characters"})
    }

    if i.Description != nil && len(strings.TrimSpace(*i.Description)) > 500 {
        errs = append(errs, domain.FieldError{Field: "description", Message: "max 500 characters"})
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**UpdateTopicInput:**

```go
type UpdateTopicInput struct {
    TopicID     uuid.UUID
    Name        *string
    Description *string // nil = don't change; ptr("") = clear
}

func (i UpdateTopicInput) Validate() error {
    var errs []domain.FieldError

    if i.TopicID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
    }
    if i.Name == nil && i.Description == nil {
        errs = append(errs, domain.FieldError{Field: "input", Message: "at least one field must be provided"})
    }
    if i.Name != nil {
        name := strings.TrimSpace(*i.Name)
        if name == "" {
            errs = append(errs, domain.FieldError{Field: "name", Message: "required"})
        }
        if len(name) > 100 {
            errs = append(errs, domain.FieldError{Field: "name", Message: "max 100 characters"})
        }
    }
    if i.Description != nil && len(*i.Description) > 500 {
        errs = append(errs, domain.FieldError{Field: "description", Message: "max 500 characters"})
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**DeleteTopicInput:**

```go
type DeleteTopicInput struct {
    TopicID uuid.UUID
}

func (i DeleteTopicInput) Validate() error {
    if i.TopicID == uuid.Nil {
        return domain.NewValidationError("topic_id", "required")
    }
    return nil
}
```

**LinkEntryInput:**

```go
type LinkEntryInput struct {
    TopicID uuid.UUID
    EntryID uuid.UUID
}

func (i LinkEntryInput) Validate() error {
    var errs []domain.FieldError
    if i.TopicID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
    }
    if i.EntryID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**UnlinkEntryInput:**

```go
type UnlinkEntryInput struct {
    TopicID uuid.UUID
    EntryID uuid.UUID
}

func (i UnlinkEntryInput) Validate() error {
    var errs []domain.FieldError
    if i.TopicID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
    }
    if i.EntryID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "entry_id", Message: "required"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**BatchLinkEntriesInput:**

```go
type BatchLinkEntriesInput struct {
    TopicID  uuid.UUID
    EntryIDs []uuid.UUID
}

func (i BatchLinkEntriesInput) Validate() error {
    var errs []domain.FieldError
    if i.TopicID == uuid.Nil {
        errs = append(errs, domain.FieldError{Field: "topic_id", Message: "required"})
    }
    if len(i.EntryIDs) == 0 {
        errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "at least one entry required"})
    }
    if len(i.EntryIDs) > 200 {
        errs = append(errs, domain.FieldError{Field: "entry_ids", Message: "max 200 entries per batch"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

---

#### 6. `service.go` — Result-типы

```go
// BatchLinkResult holds the outcome of a batch link operation.
type BatchLinkResult struct {
    Linked  int
    Skipped int
}
```

---

**Acceptance criteria:**
- [ ] `domain.TopicUpdateParams` добавлен в `internal/domain/organization.go`
- [ ] `internal/service/topic/service.go` создан с 4 приватными интерфейсами: topicRepo, entryRepo, auditLogger, txManager
- [ ] Конструктор `NewService` с 5 параметрами, логгер `"service", "topic"`
- [ ] Константа `MaxTopicsPerUser = 100`
- [ ] Helper-функции: `trimOrNil`, `ptr`
- [ ] `BatchLinkResult` struct определён
- [ ] `internal/service/topic/input.go` создан
- [ ] 6 input-структур с `Validate()` — собирают все ошибки (не fail-fast)
- [ ] Все `Validate()` из topic_service_spec §7 реализованы корректно
- [ ] `UpdateTopicInput`: валидация "at least one field" при Name=nil и Description=nil
- [ ] `UpdateTopicInput`: Description > 500 проверяется по raw длине (не trimmed), допускается пустая строка (это "очистить")
- [ ] `go build ./...` компилируется
- [ ] `go vet ./internal/service/topic/...` — без warnings

---

### TASK-8.2: Topic Service — CRUD Operations + Tests

**Зависит от:** TASK-8.1 (Service struct, interfaces, input structs)

**Контекст:**
- `services/topic_service_spec_v4.md` — §5.1 (CreateTopic), §5.2 (UpdateTopic), §5.3 (DeleteTopic), §5.4 (ListTopics), §8 (аудит)
- `services/service_layer_spec_v4.md` — §4 (аудит: EntityType=TOPIC)

**Что сделать:**

Реализовать 4 операции: CreateTopic, UpdateTopic, DeleteTopic, ListTopics. Написать unit-тесты.

---

#### Операция: CreateTopic(ctx, input CreateTopicInput) → (*domain.Topic, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. Trim input:
   name = strings.TrimSpace(input.Name)
   description = trimOrNil(input.Description)

4. count = topicRepo.Count(ctx, userID)
   └─ count >= MaxTopicsPerUser → ValidationError("topics", "limit reached (max 100)")

5. tx.RunInTx(ctx, func(ctx) error {
   5a. topic = topicRepo.Create(ctx, userID, &domain.Topic{
           Name: name, Description: description,
       })
       └─ ErrAlreadyExists → return ErrAlreadyExists (unique constraint ux_topics_user_name)

   5b. audit.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityTypeTopic,
           EntityID:   &topic.ID,
           Action:     domain.AuditActionCreate,
           Changes: map[string]any{
               "name": map[string]any{"new": name},
           },
       })
   })

6. log INFO: user_id, topic_id, name
7. return topic
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Имя из пробелов | `ValidationError("name", "required")` после trim |
| Имя > 100 символов | `ValidationError` |
| Description > 500 символов | `ValidationError` |
| Тема с таким именем уже есть | `ErrAlreadyExists` (unique constraint) |
| 100 тем уже создано | `ValidationError("topics", "limit reached")` |
| Description пустая строка | Трактуется как nil (нет описания) через `trimOrNil` |

---

#### Операция: UpdateTopic(ctx, input UpdateTopicInput) → (*domain.Topic, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. old = topicRepo.GetByID(ctx, userID, input.TopicID) → ErrNotFound

4. Подготовить params:
   params = domain.TopicUpdateParams{}
   if input.Name != nil:
       trimmed = strings.TrimSpace(*input.Name)
       params.Name = &trimmed
   if input.Description != nil:
       if strings.TrimSpace(*input.Description) == "":
           params.Description = ptr("")  // clear description → NULL in DB
       else:
           trimmed = strings.TrimSpace(*input.Description)
           params.Description = &trimmed

5. tx.RunInTx(ctx, func(ctx) error {
   5a. updated = topicRepo.Update(ctx, userID, input.TopicID, params)
       └─ ErrAlreadyExists → return ErrAlreadyExists (name collision)

   5b. audit.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityTypeTopic,
           EntityID:   &input.TopicID,
           Action:     domain.AuditActionUpdate,
           Changes:    buildTopicChanges(old, updated),
       })
   })

6. log INFO: user_id, topic_id
7. return updated
```

**buildTopicChanges — только изменённые поля:**

```go
func buildTopicChanges(old, updated *domain.Topic) map[string]any {
    changes := make(map[string]any)
    if old.Name != updated.Name {
        changes["name"] = map[string]any{"old": old.Name, "new": updated.Name}
    }
    // Compare descriptions: handle nil vs non-nil
    oldDesc := ""
    if old.Description != nil {
        oldDesc = *old.Description
    }
    newDesc := ""
    if updated.Description != nil {
        newDesc = *updated.Description
    }
    if oldDesc != newDesc || (old.Description == nil) != (updated.Description == nil) {
        changes["description"] = map[string]any{"old": old.Description, "new": updated.Description}
    }
    return changes
}
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Тема не найдена | `ErrNotFound` |
| Чужая тема | `ErrNotFound` |
| Ни одного поля не передано | `ValidationError("input", "at least one field")` |
| Новое имя совпадает с другой темой | `ErrAlreadyExists` (unique constraint) |
| Новое имя = текущее имя | Операция выполняется (idempotent), updated_at обновляется |
| Name = "" | `ValidationError("name", "required")` |
| Description = "" | Описание очищается (NULL в БД). **Не** ошибка |

---

#### Операция: DeleteTopic(ctx, input DeleteTopicInput) → error

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. topic = topicRepo.GetByID(ctx, userID, input.TopicID) → ErrNotFound

4. tx.RunInTx(ctx, func(ctx) error {
   4a. topicRepo.Delete(ctx, userID, input.TopicID)
       // CASCADE удаляет записи в entry_topics
       // Entries НЕ затрагиваются

   4b. audit.Log(ctx, AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityTypeTopic,
           EntityID:   &input.TopicID,
           Action:     domain.AuditActionDelete,
           Changes: map[string]any{
               "name": map[string]any{"old": topic.Name},
           },
       })
   })

5. log INFO: user_id, topic_id, name
6. return nil
```

**Что происходит с entries:** Ничего. `entry_topics` строки удаляются CASCADE, но entries остаются в словаре.

---

#### Операция: ListTopics(ctx) → ([]*domain.Topic, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. topics = topicRepo.List(ctx, userID)
   // Возвращает все темы (max 100), sorted by name ASC (case-insensitive)
   // Каждая тема содержит EntryCount — количество привязанных entries (без soft-deleted)

3. return topics
```

**Без пагинации:** Лимит 100 тем — одна страница.

**EntryCount:** Repo вычисляет через `LEFT JOIN entry_topics + entries (WHERE deleted_at IS NULL) ... GROUP BY`. Это позволяет клиенту показать: «Еда (42)», «IT (17)».

---

#### Unit-тесты (из topic_service_spec §10.1, #1–#26)

**CreateTopic:**

| # | Тест | Assert |
|---|------|--------|
| 1 | `TestCreateTopic_Success` | Тема создана с name и description. Audit logged |
| 2 | `TestCreateTopic_WithoutDescription` | Тема создана без description |
| 3 | `TestCreateTopic_EmptyName` | Пустой name → ValidationError |
| 4 | `TestCreateTopic_WhitespaceOnlyName` | Name из пробелов → ValidationError после trim |
| 5 | `TestCreateTopic_NameTooLong` | > 100 символов → ValidationError |
| 6 | `TestCreateTopic_DescriptionTooLong` | > 500 символов → ValidationError |
| 7 | `TestCreateTopic_DuplicateName` | Тема с таким именем → ErrAlreadyExists |
| 8 | `TestCreateTopic_LimitReached` | 100 тем → ValidationError "limit reached" |
| 9 | `TestCreateTopic_Audit` | Audit record создан с EntityType=TOPIC, Action=CREATE, name в changes |
| 10 | `TestCreateTopic_Unauthorized` | Нет userID → ErrUnauthorized |
| 11 | `TestCreateTopic_EmptyDescription` | Description = "" → trimOrNil → nil |

**UpdateTopic:**

| # | Тест | Assert |
|---|------|--------|
| 12 | `TestUpdateTopic_NameOnly` | Обновлено только имя. Audit содержит old/new name |
| 13 | `TestUpdateTopic_DescriptionOnly` | Обновлено только описание |
| 14 | `TestUpdateTopic_ClearDescription` | Description = "" → NULL в БД |
| 15 | `TestUpdateTopic_BothFields` | Оба поля обновлены |
| 16 | `TestUpdateTopic_NoFields` | Ни одного поля → ValidationError |
| 17 | `TestUpdateTopic_NotFound` | Тема не найдена → ErrNotFound |
| 18 | `TestUpdateTopic_DuplicateName` | Новое имя совпадает с другой темой → ErrAlreadyExists |
| 19 | `TestUpdateTopic_EmptyName` | Name = "" → ValidationError |
| 20 | `TestUpdateTopic_Audit` | Audit содержит только изменённые поля |
| 21 | `TestUpdateTopic_Unauthorized` | Нет userID → ErrUnauthorized |

**DeleteTopic:**

| # | Тест | Assert |
|---|------|--------|
| 22 | `TestDeleteTopic_Success` | Тема удалена. Audit logged с name |
| 23 | `TestDeleteTopic_NotFound` | Тема не найдена → ErrNotFound |
| 24 | `TestDeleteTopic_Audit` | Audit record с Action=DELETE, name в changes |
| 25 | `TestDeleteTopic_Unauthorized` | Нет userID → ErrUnauthorized |

**ListTopics:**

| # | Тест | Assert |
|---|------|--------|
| 26 | `TestListTopics_Success` | Все темы, sorted by name |
| 27 | `TestListTopics_Empty` | Нет тем → пустой slice (не nil) |
| 28 | `TestListTopics_WithEntryCounts` | EntryCount корректен для каждой темы |
| 29 | `TestListTopics_Unauthorized` | Нет userID → ErrUnauthorized |

**Всего: 29 тест-кейсов**

**Acceptance criteria:**
- [ ] **CreateTopic:** полный flow — validate → trim → limit check → tx(create + audit)
- [ ] **CreateTopic:** description пустая строка → nil через trimOrNil
- [ ] **CreateTopic:** лимит 100 → ValidationError("topics", "limit reached (max 100)")
- [ ] **CreateTopic:** duplicate name → ErrAlreadyExists (unique constraint ux_topics_user_name)
- [ ] **UpdateTopic:** partial update — nil поля не меняются
- [ ] **UpdateTopic:** Description = "" → очищается (NULL в БД). Не ошибка
- [ ] **UpdateTopic:** "at least one field" валидация
- [ ] **UpdateTopic:** duplicate name → ErrAlreadyExists
- [ ] **UpdateTopic:** buildTopicChanges — только изменённые поля в audit
- [ ] **DeleteTopic:** CASCADE удаляет entry_topics. Entries не затрагиваются
- [ ] **ListTopics:** все темы с EntryCount, sorted by name
- [ ] **ListTopics:** пустой slice (не nil) при отсутствии тем
- [ ] Все мутации: ErrUnauthorized при отсутствии userID
- [ ] 29 unit-тестов покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/topic/...` — все проходят

---

### TASK-8.3: Topic Service — Link Operations + Tests

**Зависит от:** TASK-8.1 (Service struct, interfaces, input structs)

> **Примечание:** TASK-8.3 **не зависит** от TASK-8.2 (CRUD Operations). Link операции используют только topicRepo и entryRepo из TASK-8.1 и могут разрабатываться параллельно с TASK-8.2.

**Контекст:**
- `services/topic_service_spec_v4.md` — §5.5 (LinkEntry), §5.6 (UnlinkEntry), §5.7 (BatchLinkEntries)
- `services/topic_service_spec_v4.md` — §10.1 (тесты #27–#42)

**Что сделать:**

Реализовать 3 операции: LinkEntry, UnlinkEntry, BatchLinkEntries. Написать unit-тесты.

---

#### Операция: LinkEntry(ctx, input LinkEntryInput) → error

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. topicRepo.GetByID(ctx, userID, input.TopicID) → ErrNotFound
   // Проверяет ownership темы

4. entryRepo.GetByID(ctx, userID, input.EntryID) → ErrNotFound
   // Проверяет ownership entry + фильтрует soft-deleted

5. topicRepo.LinkEntry(ctx, input.EntryID, input.TopicID)
   // INSERT INTO entry_topics ... ON CONFLICT DO NOTHING
   // Идемпотентно — повторный link не ошибка

6. return nil
```

**Без транзакции:** Одна INSERT-операция, атомарна сама по себе.

**Без аудита:** Link — лёгкая операция категоризации.

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Тема не найдена | `ErrNotFound` |
| Entry не найден | `ErrNotFound` |
| Entry soft-deleted | entryRepo.GetByID фильтрует → `ErrNotFound` |
| Entry уже привязан к теме | Noop (ON CONFLICT DO NOTHING), не ошибка |
| Чужая тема | `ErrNotFound` |
| Чужой entry | `ErrNotFound` |

---

#### Операция: UnlinkEntry(ctx, input UnlinkEntryInput) → error

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. topicRepo.GetByID(ctx, userID, input.TopicID) → ErrNotFound
   // Проверяет ownership темы

4. topicRepo.UnlinkEntry(ctx, input.EntryID, input.TopicID)
   // DELETE FROM entry_topics WHERE entry_id = $1 AND topic_id = $2
   // Если связи не было — 0 affected rows, НЕ ошибка (идемпотентно)

5. return nil
```

**Без проверки entry:** Unlink не требует проверки существования entry. Если entry удалён — CASCADE уже удалил связь. Нет смысла проверять.

**Идемпотентность:** Если связь не существовала — DELETE affected 0 rows. **Не** ошибка.

---

#### Операция: BatchLinkEntries(ctx, input BatchLinkEntriesInput) → (*BatchLinkResult, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. topicRepo.GetByID(ctx, userID, input.TopicID) → ErrNotFound
   // Проверяет ownership темы

4. existing = entryRepo.ExistByIDs(ctx, userID, input.EntryIDs)
   // Возвращает map[uuid.UUID]bool — какие entries существуют и принадлежат пользователю

5. validEntryIDs = filter(input.EntryIDs, existing)
   // Фильтруем: только существующие и не soft-deleted

6. if len(validEntryIDs) == 0:
   return &BatchLinkResult{Linked: 0, Skipped: len(input.EntryIDs)}, nil

7. linked = topicRepo.BatchLinkEntries(ctx, validEntryIDs, input.TopicID)
   // Multi-row INSERT ... ON CONFLICT DO NOTHING
   // Returns count of actually inserted rows (not counting ON CONFLICT skips)

8. skipped = len(input.EntryIDs) - linked

9. log INFO: user_id, topic_id, requested=len(input.EntryIDs), linked, skipped
10. return &BatchLinkResult{Linked: linked, Skipped: skipped}, nil
```

**Без транзакции:** Multi-row INSERT с ON CONFLICT DO NOTHING — каждая строка идемпотентна.

**Partial success:** Несуществующие entries пропускаются. Уже привязанные — ON CONFLICT DO NOTHING. Возвращает linked/skipped counts.

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Тема не найдена | `ErrNotFound` |
| Некоторые entries не существуют | Пропускаются, не ошибка. skipped++ |
| Все entries уже привязаны | linked = 0, skipped = all. Не ошибка |
| > 200 entries | `ValidationError` |
| Пустой массив | `ValidationError` |
| Все entries не существуют | linked = 0, skipped = all. Не ошибка |

---

#### Unit-тесты (из topic_service_spec §10.1, #27–#42)

**LinkEntry:**

| # | Тест | Assert |
|---|------|--------|
| 1 | `TestLinkEntry_Success` | Entry привязан к теме. topicRepo.LinkEntry called |
| 2 | `TestLinkEntry_TopicNotFound` | Тема не найдена → ErrNotFound |
| 3 | `TestLinkEntry_EntryNotFound` | Entry не найден → ErrNotFound |
| 4 | `TestLinkEntry_EntryDeleted` | Entry soft-deleted → ErrNotFound |
| 5 | `TestLinkEntry_AlreadyLinked` | Повторный link → нет ошибки (idempotent) |
| 6 | `TestLinkEntry_WrongUserTopic` | Чужая тема → ErrNotFound |
| 7 | `TestLinkEntry_WrongUserEntry` | Чужой entry → ErrNotFound |
| 8 | `TestLinkEntry_Unauthorized` | Нет userID → ErrUnauthorized |
| 9 | `TestLinkEntry_NilTopicID` | TopicID = nil → ValidationError |
| 10 | `TestLinkEntry_NilEntryID` | EntryID = nil → ValidationError |

**UnlinkEntry:**

| # | Тест | Assert |
|---|------|--------|
| 11 | `TestUnlinkEntry_Success` | Связь удалена |
| 12 | `TestUnlinkEntry_TopicNotFound` | Тема не найдена → ErrNotFound |
| 13 | `TestUnlinkEntry_NotLinked` | Связи нет → нет ошибки (idempotent) |
| 14 | `TestUnlinkEntry_Unauthorized` | Нет userID → ErrUnauthorized |

**BatchLinkEntries:**

| # | Тест | Assert |
|---|------|--------|
| 15 | `TestBatchLinkEntries_Success` | 5 entries привязаны. linked=5, skipped=0 |
| 16 | `TestBatchLinkEntries_SomeAlreadyLinked` | 2 уже привязаны → linked=3, skipped=2 |
| 17 | `TestBatchLinkEntries_SomeNotFound` | Несуществующие entries пропускаются. skipped includes not found |
| 18 | `TestBatchLinkEntries_AllAlreadyLinked` | linked=0, skipped=all |
| 19 | `TestBatchLinkEntries_TopicNotFound` | Тема не найдена → ErrNotFound |
| 20 | `TestBatchLinkEntries_EmptyInput` | Пустой массив → ValidationError |
| 21 | `TestBatchLinkEntries_TooMany` | > 200 entries → ValidationError |
| 22 | `TestBatchLinkEntries_Unauthorized` | Нет userID → ErrUnauthorized |
| 23 | `TestBatchLinkEntries_AllEntriesNotFound` | Ни один entry не найден → linked=0 |

**Всего: 23 тест-кейса**

**Acceptance criteria:**
- [ ] **LinkEntry:** ownership check для topic и entry. ON CONFLICT DO NOTHING (idempotent)
- [ ] **LinkEntry:** entry soft-deleted → ErrNotFound
- [ ] **LinkEntry:** повторный link → нет ошибки
- [ ] **LinkEntry:** без транзакции, без аудита
- [ ] **UnlinkEntry:** ownership check только для topic. Без проверки entry
- [ ] **UnlinkEntry:** связи нет → нет ошибки (idempotent)
- [ ] **UnlinkEntry:** без транзакции, без аудита
- [ ] **BatchLinkEntries:** filter existing → batch INSERT ON CONFLICT DO NOTHING → return linked/skipped
- [ ] **BatchLinkEntries:** несуществующие entries пропускаются (не ошибка)
- [ ] **BatchLinkEntries:** все уже привязаны → linked=0, skipped=all (не ошибка)
- [ ] **BatchLinkEntries:** без транзакции, без аудита
- [ ] Все операции: ErrUnauthorized при отсутствии userID
- [ ] 23 unit-теста покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/topic/...` — все проходят

---

### TASK-8.4: Inbox Service — Complete Implementation + Tests

**Зависит от:** Фаза 1 (domain models)

> **Примечание:** TASK-8.4 **полностью независима** от TASK-8.1, TASK-8.2, TASK-8.3. Inbox Service — standalone-сервис без зависимостей на Topic Service. Может разрабатываться параллельно с любыми другими задачами Фазы 8.

**Контекст:**
- `services/inbox_service_spec_v4.md` — все секции: §3 (зависимости), §5 (операции), §8 (валидация), §10 (лимиты), §11 (тестирование)
- `services/service_layer_spec_v4.md` — §5 (limits: inbox 500), §6 (InboxService standalone)

**Что сделать:**

Создать полный пакет `internal/service/inbox/` с Service struct, приватными интерфейсами, input-структурами, всеми 5 операциями и unit-тестами.

**Файловая структура:**

```
internal/service/inbox/
├── service.go          # Service struct, конструктор, приватные интерфейсы, constants, helpers
├── input.go            # Input-структуры + Validate()
└── service_test.go     # Все unit-тесты
```

---

#### `service.go` — приватные интерфейсы и конструктор

```go
package inbox

import (
    "context"
    "log/slog"

    "github.com/google/uuid"
    "github.com/heartmarshall/myenglish-backend/internal/domain"
)

const (
    MaxInboxItems  = 500
    DefaultLimit   = 50
)

type inboxRepo interface {
    Create(ctx context.Context, userID uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error)
    GetByID(ctx context.Context, userID, itemID uuid.UUID) (*domain.InboxItem, error)
    List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error)
    Delete(ctx context.Context, userID, itemID uuid.UUID) error
    DeleteAll(ctx context.Context, userID uuid.UUID) (int, error)
    Count(ctx context.Context, userID uuid.UUID) (int, error)
}

type Service struct {
    inbox inboxRepo
    log   *slog.Logger
}

func NewService(
    log   *slog.Logger,
    inbox inboxRepo,
) *Service {
    return &Service{
        inbox: inbox,
        log:   log.With("service", "inbox"),
    }
}
```

**Обоснование минимальных зависимостей:**
- **Нет auditLogger** — inbox items не аудитируются (временные заметки)
- **Нет txManager** — каждая операция затрагивает одну таблицу
- **Нет entryRepo** — конвертация item → entry выполняется клиентом отдельно

---

#### `service.go` — helper-функция

```go
// trimOrNil trims whitespace. Returns nil if result is empty.
func trimOrNil(s *string) *string {
    if s == nil {
        return nil
    }
    trimmed := strings.TrimSpace(*s)
    if trimmed == "" {
        return nil
    }
    return &trimmed
}
```

---

#### `input.go` — Input-структуры

**CreateItemInput:**

```go
type CreateItemInput struct {
    Text    string
    Context *string
}

func (i CreateItemInput) Validate() error {
    var errs []domain.FieldError

    text := strings.TrimSpace(i.Text)
    if text == "" {
        errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
    }
    if len(text) > 500 {
        errs = append(errs, domain.FieldError{Field: "text", Message: "max 500 characters"})
    }

    if i.Context != nil {
        ctx := strings.TrimSpace(*i.Context)
        if len(ctx) > 2000 {
            errs = append(errs, domain.FieldError{Field: "context", Message: "max 2000 characters"})
        }
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**ListItemsInput:**

```go
type ListItemsInput struct {
    Limit  int
    Offset int
}

func (i ListItemsInput) Validate() error {
    var errs []domain.FieldError
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

**DeleteItemInput:**

```go
type DeleteItemInput struct {
    ItemID uuid.UUID
}

func (i DeleteItemInput) Validate() error {
    if i.ItemID == uuid.Nil {
        return domain.NewValidationError("item_id", "required")
    }
    return nil
}
```

---

#### Операция: CreateItem(ctx, input CreateItemInput) → (*domain.InboxItem, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. Trim input:
   text = strings.TrimSpace(input.Text)
   context = trimOrNil(input.Context)

4. count = inboxRepo.Count(ctx, userID)
   └─ count >= MaxInboxItems → ValidationError("inbox", "inbox is full (max 500 items)")

5. item = inboxRepo.Create(ctx, userID, &domain.InboxItem{
       Text: text, Context: context,
   })

6. log INFO: user_id, item_id, text (первые 50 символов)
7. return item
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Текст из пробелов | `ValidationError("text", "required")` после trim |
| Текст > 500 символов | `ValidationError` |
| Context > 2000 символов | `ValidationError` |
| Context пустая строка | Трактуется как nil (trimOrNil) |
| Inbox полон (500 items) | `ValidationError("inbox", "inbox is full")` |
| Дублирующий текст | Допускается — не ошибка |
| Unicode, emoji | Допускаются — хранятся as-is |

---

#### Операция: ListItems(ctx, input ListItemsInput) → ([]*domain.InboxItem, int, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()
3. limit = input.Limit; if limit == 0 { limit = DefaultLimit }

4. items, totalCount = inboxRepo.List(ctx, userID, limit, input.Offset)
   // ORDER BY created_at DESC — newest first

5. return items, totalCount, nil
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Пустой inbox | Пустой slice (не nil), totalCount = 0 |
| offset > totalCount | Пустой slice, totalCount = N |

---

#### Операция: GetItem(ctx, itemID uuid.UUID) → (*domain.InboxItem, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. item = inboxRepo.GetByID(ctx, userID, itemID) → ErrNotFound

3. return item
```

---

#### Операция: DeleteItem(ctx, input DeleteItemInput) → error

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized
2. input.Validate()

3. inboxRepo.Delete(ctx, userID, input.ItemID) → ErrNotFound
   // Repo проверяет по affected rows. 0 rows → ErrNotFound
   // Один запрос, не два (без предварительного GetByID)

4. log INFO: user_id, item_id
5. return nil
```

---

#### Операция: DeleteAll(ctx) → (int, error)

```
1. userID = UserIDFromCtx(ctx) → ErrUnauthorized

2. deletedCount = inboxRepo.DeleteAll(ctx, userID)

3. log INFO: user_id, deleted_count
4. return deletedCount, nil
```

**Идемпотентность:** Пустой inbox → deletedCount = 0, не ошибка.

---

#### Unit-тесты (из inbox_service_spec §11.1, #1–#29)

**CreateItem:**

| # | Тест | Assert |
|---|------|--------|
| 1 | `TestCreateItem_Success` | Item создан с text и context |
| 2 | `TestCreateItem_WithoutContext` | Item создан без context (nil) |
| 3 | `TestCreateItem_EmptyText` | Пустой text → ValidationError |
| 4 | `TestCreateItem_WhitespaceOnlyText` | Text из пробелов → ValidationError |
| 5 | `TestCreateItem_TextTooLong` | > 500 символов → ValidationError |
| 6 | `TestCreateItem_ContextTooLong` | > 2000 символов → ValidationError |
| 7 | `TestCreateItem_EmptyContextToNil` | Context = "" → nil |
| 8 | `TestCreateItem_TextTrimmed` | "  hello  " → "hello" |
| 9 | `TestCreateItem_InboxFull` | 500 items → ValidationError "inbox is full" |
| 10 | `TestCreateItem_DuplicateAllowed` | Одинаковый text допускается |
| 11 | `TestCreateItem_Unauthorized` | Нет userID → ErrUnauthorized |

**ListItems:**

| # | Тест | Assert |
|---|------|--------|
| 12 | `TestListItems_Success` | Список items, newest first |
| 13 | `TestListItems_Empty` | Пустой inbox → пустой slice, totalCount = 0 |
| 14 | `TestListItems_Pagination` | offset + limit корректно передаются в repo |
| 15 | `TestListItems_DefaultLimit` | limit = 0 → 50 |
| 16 | `TestListItems_InvalidLimit` | limit = -1 → ValidationError |
| 17 | `TestListItems_LimitTooLarge` | limit = 201 → ValidationError |
| 18 | `TestListItems_Unauthorized` | Нет userID → ErrUnauthorized |

**GetItem:**

| # | Тест | Assert |
|---|------|--------|
| 19 | `TestGetItem_Success` | Item загружен |
| 20 | `TestGetItem_NotFound` | Несуществующий ID → ErrNotFound |
| 21 | `TestGetItem_WrongUser` | Чужой item → ErrNotFound |
| 22 | `TestGetItem_Unauthorized` | Нет userID → ErrUnauthorized |

**DeleteItem:**

| # | Тест | Assert |
|---|------|--------|
| 23 | `TestDeleteItem_Success` | Item удалён |
| 24 | `TestDeleteItem_NotFound` | Несуществующий ID → ErrNotFound |
| 25 | `TestDeleteItem_WrongUser` | Чужой item → ErrNotFound |
| 26 | `TestDeleteItem_NilID` | ItemID = nil → ValidationError |
| 27 | `TestDeleteItem_Unauthorized` | Нет userID → ErrUnauthorized |

**DeleteAll:**

| # | Тест | Assert |
|---|------|--------|
| 28 | `TestDeleteAll_Success` | Все items удалены, returns count |
| 29 | `TestDeleteAll_EmptyInbox` | Пустой inbox → deletedCount = 0, не ошибка |
| 30 | `TestDeleteAll_Unauthorized` | Нет userID → ErrUnauthorized |

**Всего: 30 тест-кейсов**

**Acceptance criteria:**
- [ ] `internal/service/inbox/service.go` создан с 1 приватным интерфейсом inboxRepo
- [ ] Конструктор `NewService` с 2 параметрами, логгер `"service", "inbox"`
- [ ] Константы: `MaxInboxItems=500`, `DefaultLimit=50`
- [ ] `internal/service/inbox/input.go` с 3 input-структурами, каждая с `Validate()`
- [ ] **CreateItem:** validate → trim → limit check → create. Без транзакции, без аудита
- [ ] **CreateItem:** дубли допускаются. Текст хранится as-is (без нормализации). Context пустой → nil
- [ ] **CreateItem:** лимит 500 → ValidationError("inbox", "inbox is full (max 500 items)")
- [ ] **ListItems:** limit=0 → default 50. ORDER BY created_at DESC
- [ ] **ListItems:** пустой inbox → пустой slice (не nil), totalCount=0
- [ ] **GetItem:** ownership через repo (фильтр по userID). ErrNotFound для чужого/несуществующего
- [ ] **DeleteItem:** repo проверяет по affected rows (один запрос, не два)
- [ ] **DeleteAll:** идемпотентно. Пустой inbox → 0, не ошибка
- [ ] Нет auditLogger, нет txManager — Inbox standalone и простой
- [ ] Все операции: ErrUnauthorized при отсутствии userID
- [ ] 30 unit-тестов покрывают все сценарии
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/inbox/...` — все проходят
- [ ] `go vet ./internal/service/inbox/...` — без warnings

---

## Сводка зависимостей задач

```
TASK-8.1 (Topic Foundation) ──┬──→ TASK-8.2 (Topic CRUD)
                               └──→ TASK-8.3 (Topic Link Operations)

TASK-8.4 (Inbox Complete)  ─────── (независима от всех остальных)
```

Детализация:
- **TASK-8.1** (Topic Foundation) зависит от: Фаза 1 (domain models)
- **TASK-8.2** (Topic CRUD) зависит от: TASK-8.1 (Service struct, interfaces, input structs)
- **TASK-8.3** (Topic Link Operations) зависит от: TASK-8.1. **Не зависит** от TASK-8.2
- **TASK-8.4** (Inbox Complete) зависит от: Фаза 1 (domain models). **Не зависит** от TASK-8.1, TASK-8.2, TASK-8.3
- TASK-8.2 и TASK-8.3 не имеют взаимных зависимостей

---

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-8.1 (Topic Foundation), TASK-8.4 (Inbox Complete) |
| 2 | TASK-8.2 (Topic CRUD), TASK-8.3 (Topic Link Operations) |

> При полной параллелизации — **2 sequential волны**. Волна 1 — до 2 задач параллельно. Волна 2 — до 2 задач параллельно.
> TASK-8.4 (Inbox) полностью независима и может начинаться одновременно с TASK-8.1 (Topic Foundation).

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] **Domain:** `TopicUpdateParams` добавлен в `domain/organization.go`
- [ ] **Topic Service — Foundation:**
  - [ ] Service struct с 4 приватными интерфейсами: topicRepo, entryRepo, auditLogger, txManager
  - [ ] Конструктор с 5 параметрами, логгер "service", "topic"
  - [ ] Константа MaxTopicsPerUser=100
  - [ ] BatchLinkResult struct
  - [ ] 6 input-структур с Validate() — собирают все ошибки
  - [ ] Helper-функции: trimOrNil, ptr
- [ ] **Topic Service — CRUD** — все 4 операции реализованы:
  - [ ] CreateTopic: trim, limit 100, tx(create + audit), source ErrAlreadyExists
  - [ ] UpdateTopic: partial update, Description="" → clear, buildTopicChanges
  - [ ] DeleteTopic: CASCADE entry_topics, entries preserved, audit
  - [ ] ListTopics: all topics, sorted by name, EntryCount computed
- [ ] **Topic Service — Link Operations** — все 3 операции реализованы:
  - [ ] LinkEntry: ownership check topic + entry, ON CONFLICT DO NOTHING, idempotent
  - [ ] UnlinkEntry: ownership check topic only, idempotent (0 rows = OK)
  - [ ] BatchLinkEntries: filter existing entries, batch INSERT, return linked/skipped
- [ ] **Topic Service** — link/unlink/batch: без транзакций, без аудита
- [ ] **Topic Service** — CRUD: аудит в транзакциях (EntityType=TOPIC)
- [ ] **Inbox Service** — все 5 операций реализованы:
  - [ ] CreateItem: trim, limit 500, дубли допускаются, без нормализации
  - [ ] ListItems: paginated, newest first, default limit 50
  - [ ] GetItem: ownership через repo
  - [ ] DeleteItem: hard delete, affected rows check
  - [ ] DeleteAll: idempotent, returns count
- [ ] **Inbox Service** — standalone: без аудита, без транзакций, без txManager
- [ ] Логирование соответствует спецификации (INFO для мутаций)
- [ ] Все ownership errors → ErrNotFound (не ErrForbidden)
- [ ] Моки сгенерированы через `moq` из приватных интерфейсов
- [ ] ~82 unit-тестов покрывают все сценарии (29 Topic CRUD + 23 Topic Link + 30 Inbox)
- [ ] Все acceptance criteria всех 4 задач выполнены
