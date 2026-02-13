# MyEnglish Backend v4 — Topic Service Specification

---

## 1. Ответственность

Topic Service — сервис категоризации слов по пользовательским темам. Темы — это лёгкие «папки» для организации словаря: «Еда», «IT-термины», «Phrasal verbs», «IELTS Writing» и т.д.

Отвечает за:

- CRUD тем (создание, редактирование, удаление)
- Просмотр списка тем пользователя
- Привязка / отвязка entries к темам (M2M)
- Массовая привязка нескольких entries к теме

Topic Service **не** отвечает за: фильтрацию словаря по теме (реализуется через `Dictionary.Find(topicID:...)` — dictionary сервис использует `entry_topics` через EXISTS-подзапрос), CRUD entries, отображение entries внутри темы.

**Ключевой принцип:** Темы — пользовательские метки, не иерархия. Один entry может принадлежать нескольким темам. Удаление темы не затрагивает entries — удаляются только M2M связи.

---

## 2. Структура пакета

```
internal/service/topic/
├── service.go          # Struct, конструктор, приватные интерфейсы
├── input.go            # Input-структуры с Validate()
└── service_test.go     # Unit-тесты с моками
```

---

## 3. Зависимости (приватные интерфейсы)

```go
// service/topic/service.go
package topic

import (
    "context"
    "log/slog"

    "github.com/google/uuid"
    "myenglish/internal/domain"
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

    // Чтение M2M
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

type Service struct {
    topics  topicRepo
    entries entryRepo
    audit   auditLogger
    tx      txManager
    log     *slog.Logger
}

func NewService(
    log *slog.Logger,
    topics topicRepo,
    entries entryRepo,
    audit auditLogger,
    tx txManager,
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

### 3.1. Зависимости — обоснование

- **entryRepo** — нужен для проверки существования entry перед link (ownership + soft-delete filter). Topic Service не создаёт и не модифицирует entries — только проверяет, что entry существует и принадлежит пользователю.
- **auditLogger** — темы аудитируются (create, update, delete). Это пользовательские данные с бизнес-значением, в отличие от inbox.
- **txManager** — нужен для create + audit и update + audit в одной транзакции.

---

## 4. Domain-модель

```go
// domain/topic.go

type Topic struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    Name        string
    Description *string
    CreatedAt   time.Time
    UpdatedAt   time.Time

    // Агрегат — заполняется при необходимости (List с подсчётом)
    EntryCount  int
}

type TopicUpdateParams struct {
    Name        *string    // nil = не менять
    Description *string    // nil = не менять; пустая строка = очистить
}
```

**Семантика nil vs пустая строка в TopicUpdateParams:**
- `Name: nil` → не менять имя. `Name: ptr("")` → невалидно, имя не может быть пустым.
- `Description: nil` → не менять описание. `Description: ptr("")` → очистить описание (установить NULL в БД).

Это partial update: клиент передаёт только изменённые поля.

---

## 5. Операции

### 5.1. CreateTopic (T1)

**Описание:** Создание новой темы.

**Input:**
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

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `input.Validate()`
3. Trim: `name = strings.TrimSpace(input.Name)`, `description = trimOrNil(input.Description)`
4. Проверить лимит: `count := topicRepo.Count(ctx, userID)` → если ≥ 100 → `ValidationError("topics", "limit reached (max 100)")`
5. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Создать тему: `topic := topicRepo.Create(ctx, userID, &domain.Topic{Name: name, Description: description})`
   b. Audit: `audit.Log(ctx, AuditRecord{EntityType: TOPIC, Action: CREATE, EntityID: topic.ID, Changes: {"name": {"new": name}}})`
6. Логировать INFO: `user_id`, `topic_id`, `name`
7. Вернуть `topic`

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Имя из пробелов | `ValidationError("name", "required")` после trim |
| Имя > 100 символов | `ValidationError` |
| Description > 500 символов | `ValidationError` |
| Тема с таким именем уже есть | `ErrAlreadyExists` (unique constraint `ux_topics_user_name`) |
| 100 тем уже создано | `ValidationError("topics", "limit reached")` |
| Description пустая строка | Трактуется как nil (нет описания) |

---

### 5.2. UpdateTopic (T2)

**Описание:** Редактирование имени и/или описания темы. Partial update — только переданные поля.

**Input:**
```go
type UpdateTopicInput struct {
    TopicID     uuid.UUID
    Name        *string
    Description *string   // nil = не менять; ptr("") = очистить
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

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Загрузить тему: `old := topicRepo.GetByID(ctx, userID, input.TopicID)` → `ErrNotFound`
4. Подготовить params: trim name/description, собрать `TopicUpdateParams`
5. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Обновить: `updated := topicRepo.Update(ctx, userID, input.TopicID, params)`
   b. Audit: `audit.Log(ctx, AuditRecord{EntityType: TOPIC, Action: UPDATE, Changes: diffFields(old, updated)})`
6. Логировать INFO: `user_id`, `topic_id`
7. Вернуть `updated`

**Audit changes — только изменённые поля:**
```json
{"name": {"old": "Food", "new": "Еда"}}
```
или
```json
{"description": {"old": "Слова про еду", "new": null}}
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Тема не найдена | `ErrNotFound` |
| Чужая тема | `ErrNotFound` |
| Ни одного поля не передано | `ValidationError("input", "at least one field")` |
| Новое имя совпадает с другой темой | `ErrAlreadyExists` (unique constraint) |
| Новое имя совпадает с текущим именем | Операция выполняется (idempotent), updated_at обновляется |
| Name = "" (пустая строка) | `ValidationError("name", "required")` |
| Description = "" (пустая строка) | Описание очищается (NULL в БД). Это **не** ошибка |

---

### 5.3. DeleteTopic (T3)

**Описание:** Удаление темы. Entries не затрагиваются — CASCADE удаляет только записи в `entry_topics`.

**Input:**
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

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Загрузить тему (для audit): `topic := topicRepo.GetByID(ctx, userID, input.TopicID)` → `ErrNotFound`
4. **Транзакция** `tx.RunInTx(ctx, fn)`:
   a. Удалить: `topicRepo.Delete(ctx, userID, input.TopicID)` (CASCADE удаляет entry_topics)
   b. Audit: `audit.Log(ctx, AuditRecord{EntityType: TOPIC, Action: DELETE, Changes: {"name": {"old": topic.Name}}})`
5. Логировать INFO: `user_id`, `topic_id`, `name`
6. Вернуть `nil`

**Что происходит с entries:** Ничего. `entry_topics` строки удаляются CASCADE, но entries остаются в словаре. Пользователь не теряет слова при удалении темы.

---

### 5.4. ListTopics (T4)

**Описание:** Список всех тем пользователя, отсортированный по имени.

**Input:** Нет (только userID из context).

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `topics := topicRepo.List(ctx, userID)` — возвращает все темы (max 100, без пагинации)
3. Вернуть `topics`

**Порядок:** `ORDER BY name ASC` (case-insensitive).

**Почему без пагинации:** Лимит — 100 тем. Это одна страница. Пагинация добавляет сложность без пользы. Если лимит увеличится — пагинация добавляется тривиально.

**EntryCount:** Каждая тема в ответе содержит `EntryCount` — количество привязанных entries. Repo вычисляет через `LEFT JOIN entry_topics ... GROUP BY` или subquery `COUNT(*)`. Это позволяет клиенту показать: «Еда (42)», «IT (17)».

---

### 5.5. LinkEntry (T5)

**Описание:** Привязка одного entry к теме.

**Input:**
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

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Проверить, что тема принадлежит пользователю: `topicRepo.GetByID(ctx, userID, input.TopicID)` → `ErrNotFound`
4. Проверить, что entry существует и принадлежит пользователю: `entryRepo.GetByID(ctx, userID, input.EntryID)` → `ErrNotFound`
5. `topicRepo.LinkEntry(ctx, input.EntryID, input.TopicID)` — `INSERT ... ON CONFLICT DO NOTHING`
6. Вернуть `nil`

**Идемпотентность:** `ON CONFLICT DO NOTHING` — повторный link не вызывает ошибку. Операция идемпотентна.

**Без аудита:** Link/unlink не аудитируются — это лёгкие операции категоризации, не критичные для бизнеса. Audit на каждый link раздувает лог без ценности.

**Без транзакции:** Одна INSERT-операция, атомарна сама по себе.

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

### 5.6. UnlinkEntry (T6)

**Описание:** Отвязка entry от темы.

**Input:**
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

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Проверить ownership темы: `topicRepo.GetByID(ctx, userID, input.TopicID)` → `ErrNotFound`
4. `topicRepo.UnlinkEntry(ctx, input.EntryID, input.TopicID)` — `DELETE FROM entry_topics WHERE entry_id = $1 AND topic_id = $2`
5. Вернуть `nil`

**Идемпотентность:** Если связь не существовала — DELETE affected 0 rows. **Не** ошибка — операция идемпотентна.

**Без проверки entry:** Unlink не требует проверки существования entry — если entry удалён, CASCADE уже удалил связь. Нет смысла проверять.

---

### 5.7. BatchLinkEntries (T7)

**Описание:** Массовая привязка нескольких entries к теме одним запросом.

**Input:**
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

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `input.Validate()`
3. Проверить ownership темы: `topicRepo.GetByID(ctx, userID, input.TopicID)` → `ErrNotFound`
4. Проверить существование entries: `existing := entryRepo.ExistByIDs(ctx, userID, input.EntryIDs)` → отфильтровать несуществующие и чужие
5. `linked := topicRepo.BatchLinkEntries(ctx, validEntryIDs, input.TopicID)` — multi-row INSERT с `ON CONFLICT DO NOTHING`
6. Логировать INFO: `user_id`, `topic_id`, `requested`, `linked`
7. Вернуть `{linked: N, skipped: M}` (skipped = уже были привязаны или не существуют)

**Реализация в repo:** Один multi-row INSERT, без транзакции (каждая строка идемпотентна):
```sql
INSERT INTO entry_topics (entry_id, topic_id)
VALUES ($1, $2), ($3, $4), ...
ON CONFLICT DO NOTHING
```

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Тема не найдена | `ErrNotFound` |
| Некоторые entries не существуют | Пропускаются, не ошибка |
| Все entries уже привязаны | linked = 0, skipped = all. Не ошибка |
| > 200 entries | `ValidationError` |
| Пустой массив | `ValidationError` |

---

### 5.8. Фильтрация словаря по теме (T8)

**Описание:** Пользователь выбирает тему и видит только entries этой темы.

**Реализация:** Это **не** операция Topic Service. Фильтрация реализуется через `Dictionary.Find(topicID: uuid)`, который добавляет EXISTS-подзапрос:

```sql
AND EXISTS (
    SELECT 1 FROM entry_topics et
    WHERE et.entry_id = e.id AND et.topic_id = $topicID
)
```

Topic Service предоставляет только данные (список тем, entry counts). Фильтрация — ответственность Dictionary Service.

---

## 6. Error Scenarios (сводная таблица)

| Операция | Ошибка | Тип | HTTP-аналог |
|----------|--------|-----|-------------|
| Все | Нет userID в ctx | `ErrUnauthorized` | 401 |
| CreateTopic | name пустой | `ValidationError` | 400 |
| CreateTopic | name > 100 символов | `ValidationError` | 400 |
| CreateTopic | description > 500 символов | `ValidationError` | 400 |
| CreateTopic | Тема с таким именем уже есть | `ErrAlreadyExists` | 409 |
| CreateTopic | Лимит 100 тем достигнут | `ValidationError` | 400 |
| UpdateTopic | topic_id = nil | `ValidationError` | 400 |
| UpdateTopic | Ни одного поля не передано | `ValidationError` | 400 |
| UpdateTopic | Тема не найдена | `ErrNotFound` | 404 |
| UpdateTopic | Новое имя дублирует существующее | `ErrAlreadyExists` | 409 |
| UpdateTopic | name = "" | `ValidationError` | 400 |
| DeleteTopic | topic_id = nil | `ValidationError` | 400 |
| DeleteTopic | Тема не найдена | `ErrNotFound` | 404 |
| LinkEntry | topic_id = nil | `ValidationError` | 400 |
| LinkEntry | entry_id = nil | `ValidationError` | 400 |
| LinkEntry | Тема не найдена | `ErrNotFound` | 404 |
| LinkEntry | Entry не найден | `ErrNotFound` | 404 |
| UnlinkEntry | topic_id = nil | `ValidationError` | 400 |
| UnlinkEntry | entry_id = nil | `ValidationError` | 400 |
| UnlinkEntry | Тема не найдена | `ErrNotFound` | 404 |
| BatchLinkEntries | topic_id = nil | `ValidationError` | 400 |
| BatchLinkEntries | Пустой массив | `ValidationError` | 400 |
| BatchLinkEntries | > 200 entries | `ValidationError` | 400 |
| BatchLinkEntries | Тема не найдена | `ErrNotFound` | 404 |

---

## 7. Валидация (сводная)

| Input | Поле | Правило |
|-------|------|---------|
| CreateTopicInput | name | required, trimmed, ≤ 100 символов |
| CreateTopicInput | description | optional, ≤ 500 символов, пустая строка → nil |
| UpdateTopicInput | topic_id | required, не Nil UUID |
| UpdateTopicInput | name | if present: trimmed, non-empty, ≤ 100 символов |
| UpdateTopicInput | description | if present: ≤ 500 символов. Пустая строка = очистить |
| UpdateTopicInput | (general) | Хотя бы одно поле должно быть передано |
| DeleteTopicInput | topic_id | required, не Nil UUID |
| LinkEntryInput | topic_id | required, не Nil UUID |
| LinkEntryInput | entry_id | required, не Nil UUID |
| UnlinkEntryInput | topic_id | required, не Nil UUID |
| UnlinkEntryInput | entry_id | required, не Nil UUID |
| BatchLinkEntriesInput | topic_id | required, не Nil UUID |
| BatchLinkEntriesInput | entry_ids | ≥ 1 элемент, ≤ 200 |

---

## 8. Аудит

| Операция | EntityType | Action | Changes |
|----------|------------|--------|---------|
| CreateTopic | TOPIC | CREATE | `{"name": {"new": "Food"}}` |
| UpdateTopic (name) | TOPIC | UPDATE | `{"name": {"old": "Food", "new": "Еда"}}` |
| UpdateTopic (description) | TOPIC | UPDATE | `{"description": {"old": "...", "new": null}}` |
| DeleteTopic | TOPIC | DELETE | `{"name": {"old": "Food"}}` |

**Link/Unlink/BatchLink не аудитируются.** Это лёгкие операции категоризации без бизнес-критичности.

**Примечание:** `EntityType: TOPIC` требует добавления значения `'TOPIC'` в enum `entity_type`. Текущий DDL содержит `('ENTRY', 'SENSE', 'EXAMPLE', 'IMAGE', 'PRONUNCIATION', 'CARD')`. Необходима миграция:

```sql
-- +goose Up
ALTER TYPE entity_type ADD VALUE 'TOPIC';
-- +goose Down
-- PostgreSQL не поддерживает удаление значений из ENUM
```

---

## 9. Лимиты

| Ресурс | Лимит | Где проверяется | Обоснование |
|--------|-------|----------------|-------------|
| Topics на пользователя | 100 | CreateTopic | Разумное количество категорий |
| Name length | 100 символов | Validate() | Короткое, читаемое название |
| Description length | 500 символов | Validate() | Краткое описание |
| BatchLink entries | 200 per request | Validate() | Ограничение размера batch |

**Нет лимита на entries в теме.** Одна тема может содержать все 10 000 entries пользователя. Это допустимо: тема — это фильтр, а не контейнер с ограничением.

---

## 10. Тестирование

### 10.1. Service Unit Tests (service_test.go)

| # | Тест | Категория | Что проверяем |
|---|------|-----------|---------------|
| 1 | `TestCreateTopic_Success` | Happy path | Тема создана с name и description |
| 2 | `TestCreateTopic_WithoutDescription` | Happy path | Тема создана без description |
| 3 | `TestCreateTopic_EmptyName` | Validation | Пустой name → ValidationError |
| 4 | `TestCreateTopic_WhitespaceOnlyName` | Validation | Name из пробелов → ValidationError |
| 5 | `TestCreateTopic_NameTooLong` | Validation | > 100 символов → ValidationError |
| 6 | `TestCreateTopic_DescriptionTooLong` | Validation | > 500 символов → ValidationError |
| 7 | `TestCreateTopic_DuplicateName` | Duplicate | Тема с таким именем → ErrAlreadyExists |
| 8 | `TestCreateTopic_LimitReached` | Limits | 100 тем → ValidationError |
| 9 | `TestCreateTopic_Audit` | Audit | Audit record создан |
| 10 | `TestCreateTopic_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 11 | `TestUpdateTopic_NameOnly` | Happy path | Обновлено только имя |
| 12 | `TestUpdateTopic_DescriptionOnly` | Happy path | Обновлено только описание |
| 13 | `TestUpdateTopic_ClearDescription` | Happy path | Description = "" → NULL |
| 14 | `TestUpdateTopic_BothFields` | Happy path | Оба поля обновлены |
| 15 | `TestUpdateTopic_NoFields` | Validation | Ни одного поля → ValidationError |
| 16 | `TestUpdateTopic_NotFound` | Not found | Тема не найдена → ErrNotFound |
| 17 | `TestUpdateTopic_DuplicateName` | Duplicate | Новое имя совпадает с другой темой → ErrAlreadyExists |
| 18 | `TestUpdateTopic_EmptyName` | Validation | Name = "" → ValidationError |
| 19 | `TestUpdateTopic_Audit` | Audit | Audit содержит только изменённые поля |
| 20 | `TestDeleteTopic_Success` | Happy path | Тема удалена |
| 21 | `TestDeleteTopic_NotFound` | Not found | Тема не найдена → ErrNotFound |
| 22 | `TestDeleteTopic_Audit` | Audit | Audit record с name |
| 23 | `TestDeleteTopic_EntriesPreserved` | Business | Entries не затронуты после удаления |
| 24 | `TestListTopics_Success` | Happy path | Все темы, sorted by name |
| 25 | `TestListTopics_Empty` | Edge case | Нет тем → пустой slice |
| 26 | `TestListTopics_WithEntryCounts` | Business | EntryCount корректен для каждой темы |
| 27 | `TestLinkEntry_Success` | Happy path | Entry привязан к теме |
| 28 | `TestLinkEntry_TopicNotFound` | Not found | Тема не найдена → ErrNotFound |
| 29 | `TestLinkEntry_EntryNotFound` | Not found | Entry не найден → ErrNotFound |
| 30 | `TestLinkEntry_EntryDeleted` | Soft delete | Entry soft-deleted → ErrNotFound |
| 31 | `TestLinkEntry_AlreadyLinked` | Idempotent | Повторный link → нет ошибки |
| 32 | `TestLinkEntry_WrongUserTopic` | Auth | Чужая тема → ErrNotFound |
| 33 | `TestLinkEntry_WrongUserEntry` | Auth | Чужой entry → ErrNotFound |
| 34 | `TestUnlinkEntry_Success` | Happy path | Связь удалена |
| 35 | `TestUnlinkEntry_TopicNotFound` | Not found | Тема не найдена → ErrNotFound |
| 36 | `TestUnlinkEntry_NotLinked` | Idempotent | Связи нет → нет ошибки |
| 37 | `TestBatchLinkEntries_Success` | Happy path | 5 entries привязаны |
| 38 | `TestBatchLinkEntries_SomeExist` | Partial | 2 уже привязаны → linked = 3 |
| 39 | `TestBatchLinkEntries_SomeNotFound` | Partial | Несуществующие entries пропускаются |
| 40 | `TestBatchLinkEntries_TopicNotFound` | Not found | Тема не найдена → ErrNotFound |
| 41 | `TestBatchLinkEntries_EmptyInput` | Validation | Пустой массив → ValidationError |
| 42 | `TestBatchLinkEntries_TooMany` | Validation | > 200 entries → ValidationError |

---

## 11. Обоснование решений

### 11.1. Почему плоские темы, а не иерархия?

Иерархия (parent → child) добавляет: рекурсивные запросы (CTE), UI для drag-and-drop дерева, сложность перемещения entry из подтемы в родительскую. На 100 тем это overkill. Если пользователю нужна иерархия — он именует темы: «IELTS», «IELTS / Writing», «IELTS / Speaking».

### 11.2. Почему нет M2M topic ↔ topic?

«Группы тем» или «мета-темы» — усложнение без ясного use case. Каждый entry может принадлежать нескольким темам одновременно — это покрывает потребность в cross-categorization.

### 11.3. Почему link/unlink не аудитируются?

Link — это добавление строки в M2M таблицу (32 байта). Аудит этой операции (128+ байт в audit_log) создаёт overhead больше самой операции. Пользователь с 2000 entries и 20 темами может сгенерировать ~5000 link-операций при реорганизации. Аудит каждой из них не имеет аналитической ценности.

### 11.4. Почему имена тем case-sensitive?

Unique constraint `ux_topics_user_name` — по raw name, без нормализации. «Food» и «food» — две разные темы. Это проще в реализации и соответствует ожиданиям: пользователь видит имя ровно так, как ввёл. Если потребуется case-insensitive uniqueness — можно добавить `name_normalized` (аналогично `entries.text_normalized`).

### 11.5. Почему EntryCount в ListTopics, а не отдельный запрос?

Один запрос с LEFT JOIN + COUNT эффективнее, чем N запросов (по одному на тему). При 100 темах — один query vs 100. Repo реализует это через:

```sql
SELECT t.*, COUNT(et.entry_id) AS entry_count
FROM topics t
LEFT JOIN entry_topics et ON et.topic_id = t.id
LEFT JOIN entries e ON e.id = et.entry_id AND e.deleted_at IS NULL
WHERE t.user_id = $1
GROUP BY t.id
ORDER BY t.name ASC;
```

Обратить внимание: JOIN с entries нужен для исключения soft-deleted entries из count.

---

## 12. Развитие post-MVP

| Фича | Описание | Приоритет |
|------|----------|-----------|
| Цвет / иконка темы | Визуальная идентификация темы в UI | Средний |
| Порядок тем (position) | Пользовательский порядок drag-and-drop | Низкий |
| Merge topics | Объединить две темы: перенести все entries из одной в другую, удалить исходную | Средний |
| Topic stats | Статистика по теме: % MASTERED, average ease, прогресс | Высокий |
| Auto-tagging | AI предлагает тему для нового entry на основе senses/definition | Низкий |
| Batch unlink | Массовая отвязка entries от темы | Средний |
