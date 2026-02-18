# Topic Service

Сервис категоризации словарных entries по пользовательским темам. Темы — плоские метки ("Еда", "IT-термины", "IELTS Writing"), без иерархии. Один entry может принадлежать нескольким темам. Удаление темы не затрагивает entries.

## Зависимости

```
topicRepo   — CRUD тем + M2M операции (link/unlink/batch)
entryRepo   — проверка существования entries (GetByID, ExistByIDs)
auditLogger — запись аудита для CRUD операций
txManager   — управление транзакциями
```

Все интерфейсы определены приватно в `service.go`. Wiring через duck typing в `main.go`.

## Операции

### CRUD (в транзакции с аудитом)

| Метод | Input | Возврат | Транзакция | Аудит |
|-------|-------|---------|------------|-------|
| `CreateTopic` | `CreateTopicInput{Name, Description}` | `*domain.Topic` | да | CREATE |
| `UpdateTopic` | `UpdateTopicInput{TopicID, Name?, Description?}` | `*domain.Topic` | да | UPDATE (только изменённые поля) |
| `DeleteTopic` | `DeleteTopicInput{TopicID}` | `error` | да | DELETE |
| `GetTopic` | `topicID uuid.UUID` | `*domain.Topic` | нет | нет |
| `ListTopics` | — (userID из ctx) | `[]*domain.Topic` | нет | нет |

### M2M: привязка entries к темам

| Метод | Input | Возврат | Идемпотентность |
|-------|-------|---------|-----------------|
| `LinkEntry` | `LinkEntryInput{TopicID, EntryID}` | `error` | да (ON CONFLICT DO NOTHING) |
| `UnlinkEntry` | `UnlinkEntryInput{TopicID, EntryID}` | `error` | да (0 affected rows — не ошибка) |
| `BatchLinkEntries` | `BatchLinkEntriesInput{TopicID, EntryIDs}` | `*BatchLinkResult{Linked, Skipped}` | да |

## Потоки выполнения

### CreateTopic

```
UserID из ctx → Validate → trim name/description → RunInTx {
    Count(userID) >= 100? → ValidationError
    topicRepo.Create(...)
    audit.Log(CREATE)
} → log INFO → return topic
```

Пустая description (`""`) преобразуется в `nil` через `trimOrNil`.

### UpdateTopic

```
UserID из ctx → Validate → trim поля → RunInTx {
    old = topicRepo.GetByID(...)          // внутри tx для консистентности
    updated = topicRepo.Update(...)
    diff = buildTopicChanges(old, updated) // только реально изменённые поля
    if diff не пуст → audit.Log(UPDATE)   // no-op update не аудитируется
} → log INFO → return updated
```

Partial update: `nil` = не менять, `ptr("")` для Description = очистить (NULL в БД).

### DeleteTopic

```
UserID из ctx → Validate → RunInTx {
    topic = topicRepo.GetByID(...)  // внутри tx, race-safe
    topicRepo.Delete(...)           // CASCADE удаляет entry_topics
    audit.Log(DELETE)
} → log INFO
```

### LinkEntry

```
UserID из ctx → Validate → RunInTx {
    topicRepo.GetByID(...)  // ownership check
    entryRepo.GetByID(...)  // ownership + soft-delete filter
    topicRepo.LinkEntry(...) // INSERT ON CONFLICT DO NOTHING
    audit.Log(UPDATE)
} → log INFO
```

### UnlinkEntry

```
UserID из ctx → Validate → RunInTx {
    topicRepo.GetByID(...)    // ownership check (entry не проверяется)
    topicRepo.UnlinkEntry(...)
    audit.Log(UPDATE)
} → log INFO
```

### BatchLinkEntries

```
UserID из ctx → Validate → deduplicate IDs → RunInTx {
    topicRepo.GetByID(...)                // ownership check
    existing = entryRepo.ExistByIDs(...)  // фильтр несуществующих
    если все отфильтрованы → {Linked: 0, Skipped: N}
    linked = topicRepo.BatchLinkEntries(validIDs, topicID)
    audit.Log(UPDATE)
} → log INFO → return {Linked, Skipped}
```

Несуществующие и дублированные entry ID пропускаются без ошибки.

## Валидация

Каждый Input имеет `Validate() error`, возвращающий `*domain.ValidationError` со всеми ошибками разом.

| Поле | Правило |
|------|---------|
| `name` | обязательное, trimmed, <= 100 символов (runes) |
| `description` | опциональное, <= 500 символов (runes) |
| `topic_id` | обязательное, не uuid.Nil |
| `entry_id` | обязательное, не uuid.Nil |
| `entry_ids` | 1..200 элементов |
| UpdateTopic | хотя бы одно поле (Name или Description) |

## Лимиты

| Ресурс | Лимит |
|--------|-------|
| Тем на пользователя | 100 (проверяется в CreateTopic внутри транзакции) |
| Длина имени | 100 символов |
| Длина описания | 500 символов |
| Batch link | 200 entries за запрос |
| Entries в теме | без лимита |

## Ошибки

Все операции: `ErrUnauthorized` если нет userID в контексте.

| Ситуация | Ошибка |
|----------|--------|
| Невалидный input | `*ValidationError` |
| Тема не найдена / чужая | `ErrNotFound` |
| Entry не найден / чужой / soft-deleted | `ErrNotFound` |
| Дублирующее имя темы | `ErrAlreadyExists` |
| Лимит 100 тем | `*ValidationError` |
| Повторный link/unlink | не ошибка (идемпотентно) |

## Файлы

```
service.go        — интерфейсы, конструктор, BatchLinkResult, helpers (trimOrNil, ptr)
input.go          — Input-структуры с Validate()
create_topic.go   — CreateTopic
update_topic.go   — UpdateTopic, buildTopicChanges
delete_topic.go   — DeleteTopic
get_topic.go      — GetTopic
list_topics.go    — ListTopics
link_entry.go     — LinkEntry, UnlinkEntry, BatchLinkEntries
generate.go       — //go:generate moq
service_test.go   — тесты CRUD + GetTopic (31 тест)
link_test.go      — тесты Link/Unlink/Batch (28 тестов)
mocks_test.go     — сгенерированные моки (moq)
```

## Тесты

59 unit-тестов, все с `t.Parallel()`. Покрытие: happy path, валидация, ошибки, аудит, edge cases (idempotent link, dedup, no-op update skip audit, soft-deleted entries).

```bash
go test -v -race ./internal/service/topic/...
```



