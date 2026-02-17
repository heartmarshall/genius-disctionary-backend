# Фаза 3: Слой репозиториев


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (структура adapter/postgres/), §8 (Querier, sqlc + Squirrel, миграции, индексы) |
| `data_model_v4.md` | Все секции — DDL каждой таблицы, FK, индексы, triggers, enums |
| `repo/repo_layer_spec_v4.md` | Все секции — полная спецификация repository layer |
| `repo/repo_layer_tasks_v4.md` | TASK-100 — TASK-500 (детальные описания каждого репозитория) |
| `services/study_service_spec_v4_v1.1.md` | §4 — дополнительные поля: `learning_step` в cards, `prev_state` JSONB в review_logs, таблица `study_sessions` |

---

## Маппинг задач

| Фаза 3 | repo_layer_tasks | Пакет |
|--------|-----------------|-------|
| TASK-3.1 | TASK-100 | `adapter/postgres/user/` |
| TASK-3.2 | TASK-101 | `adapter/postgres/token/` |
| TASK-3.3 | TASK-102 | `adapter/postgres/refentry/` |
| TASK-3.4 | TASK-103 | `adapter/postgres/entry/` |
| TASK-3.5 | TASK-200 | `adapter/postgres/sense/` |
| TASK-3.6 | TASK-201 | `adapter/postgres/translation/` |
| TASK-3.7 | TASK-202 | `adapter/postgres/example/` |
| TASK-3.8 | TASK-203 | `adapter/postgres/pronunciation/` |
| TASK-3.9 | TASK-204 | `adapter/postgres/image/` |
| TASK-3.10 | TASK-300 | `adapter/postgres/card/` |
| TASK-3.11 | TASK-301 | `adapter/postgres/reviewlog/` |
| TASK-3.12 | TASK-400 | `adapter/postgres/topic/` |
| TASK-3.13 | TASK-401 | `adapter/postgres/inbox/` |
| TASK-3.14 | TASK-402 | `adapter/postgres/audit/` |
| TASK-3.15 | TASK-500 | `transport/graphql/dataloader/` |

---

## Пре-условия (из Фазы 2)

Перед началом Фазы 3 должны быть готовы:
- Миграции (`goose up` без ошибок)
- `pgxpool.Pool` + `NewPool` из config
- `Querier` interface + `QuerierFromCtx`
- `TxManager.RunInTx`
- Error mapping (pgx → domain)
- sqlc шаблон + `make generate`
- Test helpers (`SetupTestDB`, seed-функции)
- Makefile с targets для тестов и генерации

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | sqlc организация | **sqlc per repo** — каждый пакет содержит свой `query/`, `sqlc/`, `sqlc.yaml` |
| 2 | sqlc ENUM overrides | Не добавляются — sqlc генерирует свои string types, маппинг в domain enum вручную |
| 3 | Filter struct | Тип `Filter` определяется в пакете entry repo. Сервис импортирует и конструирует |
| 4 | Position auto-increment | Ответственность repo: `MAX(position) + 1`. Race condition допустим |
| 5 | GetDueCards daily limit | Repo не знает о daily limits. Возвращает все due cards с общим limit. Сервис фильтрует |
| 6 | GetStreak timezone | Явное исключение: единственный метод repo, принимающий timezone string |
| 7 | DataLoaders | Вызывают repo напрямую, минуя service. Авторизация через SQL (`WHERE user_id`) |
| 8 | CountByStatus | Repo возвращает только non-zero группы. Сервис дополняет нулями |
| 9 | Reorder API | Repo принимает `[]struct{ID, Position}`, обновляет батчем в одной транзакции |
| 10 | Trigger tests | Тестировать один раз в TASK-3.5 (Sense). Не дублировать в Translation и Example |
| 11 | SeedEntry | Две функции: `SeedEntry` (без card) и `SeedEntryWithCard` |
| 12 | CardSnapshot serialization | `prev_state JSONB` — сериализация/десериализация в repo (custom marshaling), не в domain |

---

## Общие правила для всех репозиториев

Каждый репозиторий:
- Имеет собственный пакет с sqlc (query/, sqlc/, sqlc.yaml) и repo.go
- Хранит `*pgxpool.Pool`, получает querier через `QuerierFromCtx(ctx, pool)`
- Преобразует pgx-ошибки в domain-ошибки через общий `mapError`
- Содержит приватные функции маппинга `sqlcModel → domain.Model`
- Все методы user-репозиториев принимают `userID` как параметр
- Все запросы к entries и дочерним таблицам фильтруют soft-deleted (`deleted_at IS NULL`)
- Integration-тесты используют `testhelper.SetupTestDB` и seed-функции

---

## Задачи

### TASK-3.1: User Repository

**Зависит от:** Фаза 2 (все задачи)

**Контекст:**
- `data_model_v4.md` — §3 (users, user_settings)
- `repo_layer_spec_v4.md` — §6 (принципы), §19.7 (операции users)
- `repo_layer_tasks_v4.md` — TASK-100

**Что сделать:**

Создать пакет `internal/adapter/postgres/user/` с собственным sqlc и repo.go.

**Операции:**

| Метод | Описание |
|-------|----------|
| `GetByID(ctx, id)` | Получить пользователя по UUID |
| `GetByOAuth(ctx, provider, oauthID)` | Получить по OAuth credentials (основной метод для login flow) |
| `GetByEmail(ctx, email)` | Получить по email |
| `Create(ctx, user)` | Создать пользователя |
| `Update(ctx, id, name, avatarURL)` | Обновить профиль |
| `GetSettings(ctx, userID)` | Получить настройки |
| `CreateSettings(ctx, settings)` | Создать настройки (при регистрации) |
| `UpdateSettings(ctx, userID, settings)` | Обновить настройки |

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] sqlc per repo: собственный query/, sqlc/, sqlc.yaml
- [ ] Error mapping: not found → `ErrNotFound`, duplicate email → `ErrAlreadyExists`, duplicate OAuth → `ErrAlreadyExists`
- [ ] Маппинг sqlc enum → `domain.OAuthProvider` в функции маппинга
- [ ] Integration-тесты: CRUD happy path, not found, duplicate email, duplicate OAuth
- [ ] `make generate` обрабатывает sqlc.yaml этого пакета

---

### TASK-3.2: Token Repository

**Зависит от:** TASK-3.1 (нужен SeedUser для тестов)

**Контекст:**
- `data_model_v4.md` — §3 (refresh_tokens)
- `repo_layer_spec_v4.md` — §17 (data retention), §19.8 (операции)
- `repo_layer_tasks_v4.md` — TASK-101

**Что сделать:**

Создать пакет `internal/adapter/postgres/token/`.

**Операции:**

| Метод | Описание |
|-------|----------|
| `Create(ctx, userID, tokenHash, expiresAt)` | Создать refresh token |
| `GetByHash(ctx, tokenHash)` | Найти активный токен по хешу (`WHERE revoked_at IS NULL`) |
| `RevokeByID(ctx, id)` | Отозвать конкретный токен |
| `RevokeAllByUser(ctx, userID)` | Отозвать все токены пользователя (logout everywhere) |
| `DeleteExpired(ctx)` | Удалить expired/revoked токены (cleanup) |

**Corner cases:**
- `GetByHash` возвращает `ErrNotFound` для revoked токенов
- `RevokeByID` для уже revoked — идемпотентно, не ошибка
- `DeleteExpired` может удалять тысячи записей, без транзакции

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `GetByHash` не возвращает revoked/expired
- [ ] `RevokeAllByUser` затрагивает только активные
- [ ] Integration-тесты: create + get, revoke + get (not found), revoke all, delete expired

---

### TASK-3.3: Reference Catalog Repository

**Зависит от:** Фаза 2 (все задачи)

**Контекст:**
- `data_model_v4.md` — §2 (все ref_ таблицы)
- `repo_layer_spec_v4.md` — §9.1 (конкурентное заполнение), §10 (управление каталогом), §19.1 (операции)
- `repo_layer_tasks_v4.md` — TASK-102

**Что сделать:**

Создать пакет `internal/adapter/postgres/refentry/`. Управляет 6 таблицами (ref_entries + 5 дочерних) как единым агрегатом.

**Операции:**

Чтение:

| Метод | Описание |
|-------|----------|
| `GetByID(ctx, id)` | ref_entry с полным деревом (senses → translations, examples; pronunciations; images). Отдельные запросы к каждой ref-таблице |
| `GetByNormalizedText(ctx, text)` | Для проверки существования перед вызовом внешнего API |
| `Search(ctx, query, limit)` | Fuzzy search по `text_normalized` (pg_trgm). При пустом query — пустой результат без запроса к БД |

Запись:

| Метод | Описание |
|-------|----------|
| `Create(ctx, refEntry)` | Создать ref_entry + все дочерние в одной транзакции (через TxManager) |
| `GetOrCreate(ctx, text, ...)` | Upsert: `INSERT ON CONFLICT DO NOTHING`, затем `SELECT`. Возвращает ref_entry (новый или существующий) |

Batch (для DataLoaders):

| Метод | Описание |
|-------|----------|
| `GetRefSensesByIDs(ctx, ids)` | Batch по массиву UUID |
| `GetRefTranslationsByIDs(ctx, ids)` | |
| `GetRefExamplesByIDs(ctx, ids)` | |
| `GetRefPronunciationsByIDs(ctx, ids)` | |
| `GetRefImagesByIDs(ctx, ids)` | |

**Corner cases:**
- **Конкурентное создание**: `INSERT ON CONFLICT DO NOTHING` + `SELECT`. Ни один из параллельных запросов не получает ошибку
- **Транзакция Create**: ошибка при создании ref_translation → откат всего, включая ref_entry и ref_senses
- **Search**: `ORDER BY similarity(text_normalized, $query) DESC LIMIT $limit`
- **Каталог immutable**: нет Update/Delete операций в repo API

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `GetOrCreate` работает без race conditions — тест с горутинами
- [ ] `Create` — атомарная транзакция для всего дерева
- [ ] При ошибке в Create — откат всего
- [ ] `Search` использует pg_trgm
- [ ] Batch-запросы корректны для массива UUID
- [ ] Integration-тесты: create full tree, get by text, GetOrCreate (concurrent), search, batch

---

### TASK-3.4: Entry Repository

**Зависит от:** TASK-3.3 (нужен SeedRefEntry для тестов)

**Контекст:**
- `data_model_v4.md` — §4 (entries)
- `repo_layer_spec_v4.md` — §3 (нормализация), §4.3 (Squirrel), §8 (soft delete), §9.2 (конкурентное добавление), §11 (limits), §13 (pagination), §19.2 (операции)
- `repo_layer_tasks_v4.md` — TASK-103

**Что сделать:**

Создать пакет `internal/adapter/postgres/entry/` с repo.go + filter.go.

**Операции:**

Чтение:

| Метод | Описание |
|-------|----------|
| `GetByID(ctx, userID, id)` | С фильтром `deleted_at IS NULL` |
| `GetByText(ctx, userID, textNormalized)` | Для проверки дубликатов |
| `Find(ctx, userID, filter)` | Squirrel — динамические фильтры |
| `GetByIDs(ctx, userID, ids)` | Batch |
| `CountByUser(ctx, userID)` | Для проверки лимитов |

Запись:

| Метод | Описание |
|-------|----------|
| `Create(ctx, userID, entry)` | Создать entry |
| `UpdateNotes(ctx, userID, id, notes)` | Обновить заметки |
| `SoftDelete(ctx, userID, id)` | `SET deleted_at = now()` |
| `Restore(ctx, userID, id)` | `SET deleted_at = NULL` |
| `HardDeleteOld(ctx, threshold)` | Удалить записи с `deleted_at < threshold`, батчами по 100 |

**filter.go — тип `Filter`:**

Определяется в этом пакете. Содержит:

| Поле | Тип | Описание |
|------|-----|----------|
| `Search` | `*string` | `text_normalized ILIKE '%...%'` (GIN trgm index) |
| `HasCard` | `*bool` | `EXISTS / NOT EXISTS` подзапрос к cards |
| `PartOfSpeech` | `*domain.PartOfSpeech` | `EXISTS` подзапрос к senses с COALESCE |
| `TopicID` | `*uuid.UUID` | `EXISTS` подзапрос к entry_topics |
| `Status` | `*domain.LearningStatus` | `EXISTS` подзапрос к cards |
| `SortBy` | `string` | "text", "created_at", "updated_at" |
| `SortOrder` | `string` | "ASC", "DESC" |
| `Limit` | `int` | Default 50, max 200 |
| `Offset` | `int` | Offset-based пагинация |
| `Cursor` | `*string` | Cursor-based: `base64(sort_value + "|" + entry_id)` |

Результат Find:
- Offset mode: `[]domain.Entry`, `totalCount int`, `error`
- Cursor mode: `[]domain.Entry`, `hasNextPage bool`, `error`

**Cursor pagination:**
- Cursor = `base64(sort_value + "|" + entry_id)`
- Keyset: `WHERE (sort_field, id) > ($cursor_value, $cursor_id)`
- Невалидный cursor → `domain.ErrValidation`

**Corner cases:**
- Soft delete: все GET-запросы фильтруют `deleted_at IS NULL`
- Re-create after soft delete: partial unique constraint позволяет
- Find с пустым `Search`: игнорировать фильтр (не `ILIKE '%%'`)
- Find с нулём фильтров: все entries пользователя
- `HardDeleteOld`: батчами по 100 (`LIMIT` в `DELETE`)
- **UpdateText не реализуется на MVP** — слишком рискованно (unique constraint, нормализация, каскады)

**Acceptance criteria:**
- [ ] Все операции реализованы (кроме UpdateText)
- [ ] Soft delete: `GetByID` не возвращает soft-deleted
- [ ] `SoftDelete` идемпотентен
- [ ] `Restore` работает
- [ ] Find: каждый фильтр работает отдельно и в комбинации
- [ ] Find: offset и cursor pagination
- [ ] Find: `totalCount` не зависит от limit/offset
- [ ] Cursor: стабильная пагинация, невалидный cursor → `ErrValidation`
- [ ] `HardDeleteOld`: удаляет только старше threshold
- [ ] Integration-тесты: CRUD, soft delete/restore, re-create, Find (каждый фильтр, combined, pagination, cursor, empty result), concurrent create

---

### TASK-3.5: Sense Repository

**Зависит от:** TASK-3.4 (нужен SeedEntry для тестов)

**Контекст:**
- `data_model_v4.md` — §4 (senses)
- `repo_layer_spec_v4.md` — §5 (COALESCE), §12 (position management), §19.3 (операции)
- `repo_layer_tasks_v4.md` — TASK-200

**Что сделать:**

Создать пакет `internal/adapter/postgres/sense/`. Первый из трёх COALESCE-репозиториев — устанавливает паттерн для TASK-3.6 и TASK-3.7.

**Операции:**

Чтение:

| Метод | Описание |
|-------|----------|
| `GetByEntryID(ctx, entryID)` | С COALESCE (LEFT JOIN ref_senses), ORDER BY position |
| `GetByEntryIDs(ctx, entryIDs)` | Batch для DataLoader, с COALESCE. Результат содержит `entry_id` для группировки |
| `GetByID(ctx, senseID)` | Единичный, с COALESCE |
| `CountByEntry(ctx, entryID)` | Для проверки лимитов |

Запись:

| Метод | Описание |
|-------|----------|
| `CreateFromRef(ctx, entryID, refSenseID, sourceSlug)` | Position авто: `MAX(position)+1`. Поля definition/pos/cefr остаются NULL → COALESCE подхватит ref |
| `CreateCustom(ctx, entryID, definition, partOfSpeech, cefrLevel, sourceSlug)` | Position авто. Без ref_sense_id |
| `Update(ctx, senseID, definition, partOfSpeech, cefrLevel)` | **ref_sense_id НЕ трогается** (origin link) |
| `Delete(ctx, senseID)` | Удалить sense |
| `Reorder(ctx, items []struct{ID, Position})` | Батч в одной транзакции |

**COALESCE-поля:** `definition`, `part_of_speech`, `cefr_level`

Пример SQL-запроса с COALESCE:
```sql
SELECT
    s.id, s.entry_id,
    COALESCE(s.definition, rs.definition) AS definition,
    COALESCE(s.part_of_speech, rs.part_of_speech) AS part_of_speech,
    COALESCE(s.cefr_level, rs.cefr_level) AS cefr_level,
    s.source_slug, s.position, s.ref_sense_id, s.created_at
FROM senses s
LEFT JOIN ref_senses rs ON s.ref_sense_id = rs.id
WHERE s.entry_id = $1
ORDER BY s.position;
```

**Corner cases:**
- Partial customization: изменить definition, оставить part_of_speech из ref — корректное поведение
- Position auto-increment: race condition при concurrent create → допустим
- Reorder: транзакция, чтобы все позиции обновились атомарно
- **Trigger test** (только здесь, не дублировать в TASK-3.6/3.7): создать sense с ref_sense_id → удалить ref_sense → проверить что trigger скопировал данные в user-поля, sense сохранился, ref_sense_id стал NULL

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] COALESCE: user-поля NULL → ref-значения
- [ ] COALESCE: user-поля заполнены → user-значения
- [ ] Partial customization работает
- [ ] Update не обнуляет ref_sense_id
- [ ] Position: автоинкремент при создании
- [ ] Reorder: батч в транзакции
- [ ] **Trigger test**: ref_sense удалён → данные скопированы в user-поля
- [ ] Integration-тесты: create from ref, create custom, update (partial), COALESCE, trigger, reorder, batch

---

### TASK-3.6: Translation Repository

**Зависит от:** TASK-3.5 (аналогичный паттерн, копировать подход)

**Контекст:**
- `data_model_v4.md` — §4 (translations)
- `repo_layer_spec_v4.md` — §5 (COALESCE), §12 (position), §19.3 (операции)
- `repo_layer_tasks_v4.md` — TASK-201

**Что сделать:**

Создать пакет `internal/adapter/postgres/translation/`. Паттерн идентичен TASK-3.5, но:

- Parent: `sense_id` (не entry_id)
- COALESCE-поле: только `text`
- Ref-таблица: `ref_translations`
- Batch ключ: `sense_id`

**Операции:** `GetBySenseID`, `GetBySenseIDs` (batch), `GetByID`, `CountBySense`, `CreateFromRef`, `CreateCustom`, `Update`, `Delete`, `Reorder`.

**Corner cases:** аналогично TASK-3.5. Trigger тест **не дублировать** — покрыт в TASK-3.5.

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] COALESCE для поля `text`
- [ ] Position auto-increment, reorder
- [ ] Integration-тесты: create from ref, create custom, update, COALESCE, batch

---

### TASK-3.7: Example Repository

**Зависит от:** TASK-3.5 (аналогичный паттерн)

**Контекст:**
- `data_model_v4.md` — §4 (examples)
- `repo_layer_spec_v4.md` — §5 (COALESCE), §12 (position), §19.3 (операции)
- `repo_layer_tasks_v4.md` — TASK-202

**Что сделать:**

Создать пакет `internal/adapter/postgres/example/`. Паттерн идентичен TASK-3.5, но:

- Parent: `sense_id`
- COALESCE-поля: `sentence` и `translation`
- Ref-таблица: `ref_examples`
- Batch ключ: `sense_id`

**Операции:** аналогичны TASK-3.6.

**Corner cases:** аналогично. Trigger тест **не дублировать**.

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] COALESCE для полей `sentence` и `translation`
- [ ] Integration-тесты: create from ref, create custom, update, COALESCE, batch

---

### TASK-3.8: Pronunciation Repository

**Зависит от:** TASK-3.4 (нужен SeedEntry)

**Контекст:**
- `data_model_v4.md` — §4 (entry_pronunciations M2M)
- `repo_layer_spec_v4.md` — §9.4 (идемпотентность), §19.4 (операции)
- `repo_layer_tasks_v4.md` — TASK-203

**Что сделать:**

Создать пакет `internal/adapter/postgres/pronunciation/`. M2M-only репозиторий — пользователь привязывает ref_pronunciations к entry, не создаёт своих.

**Операции:**

| Метод | Описание |
|-------|----------|
| `GetByEntryID(ctx, entryID)` | `[]RefPronunciation` (JOIN entry_pronunciations + ref_pronunciations) |
| `GetByEntryIDs(ctx, entryIDs)` | Batch для DataLoader (результат содержит `entry_id`) |
| `Link(ctx, entryID, refPronunciationID)` | `ON CONFLICT DO NOTHING` |
| `Unlink(ctx, entryID, refPronunciationID)` | Удалить связь |
| `UnlinkAll(ctx, entryID)` | Удалить все связи для entry |

**Corner cases:**
- Link дважды → не ошибка (ON CONFLICT DO NOTHING)
- Unlink несуществующей связи → не ошибка (affected 0 rows — ok)
- GetByEntryID без произношений → пустой slice `[]`

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] Link идемпотентен
- [ ] Batch: группировка по entry_id
- [ ] Integration-тесты: link, link duplicate, unlink, get, batch

---

### TASK-3.9: Image Repository

**Зависит от:** TASK-3.4 (нужен SeedEntry)

**Контекст:**
- `data_model_v4.md` — §4 (entry_images M2M, user_images)
- `repo_layer_spec_v4.md` — §19.4 (операции)
- `repo_layer_tasks_v4.md` — TASK-204

**Что сделать:**

Создать пакет `internal/adapter/postgres/image/`. Два типа изображений в одном репозитории:

**Каталожные (M2M через entry_images):**

| Метод | Описание |
|-------|----------|
| `GetCatalogByEntryID(ctx, entryID)` | `[]RefImage` через JOIN |
| `GetCatalogByEntryIDs(ctx, entryIDs)` | Batch для DataLoader |
| `LinkCatalog(ctx, entryID, refImageID)` | `ON CONFLICT DO NOTHING` |
| `UnlinkCatalog(ctx, entryID, refImageID)` | Удалить связь |

**Пользовательские (user_images — CRUD):**

| Метод | Описание |
|-------|----------|
| `GetUserByEntryID(ctx, entryID)` | `[]UserImage` |
| `GetUserByEntryIDs(ctx, entryIDs)` | Batch для DataLoader |
| `CreateUser(ctx, entryID, url, caption)` | Создать пользовательское изображение |
| `DeleteUser(ctx, imageID)` | Удалить |

**Corner cases:**
- `LinkCatalog` идемпотентно
- Batch: два набора запросов (catalog и user), оба с группировкой по `entry_id`

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `LinkCatalog` идемпотентен
- [ ] Batch для обоих типов
- [ ] Integration-тесты для catalog и user images

---

### TASK-3.10: Card Repository

**Зависит от:** TASK-3.4 (нужен SeedEntry/SeedEntryWithCard)

**Контекст:**
- `data_model_v4.md` — §5 (cards)
- `repo_layer_spec_v4.md` — §8 (soft delete + cards), §9.3 (конкурентный review), §14 (timezone), §19.5 (операции)
- `repo_layer_tasks_v4.md` — TASK-300
- `services/study_service_spec_v4_v1.1.md` — §4 (поле `learning_step`)

**Что сделать:**

Создать пакет `internal/adapter/postgres/card/`.

**Операции:**

Чтение:

| Метод | Описание |
|-------|----------|
| `GetByID(ctx, userID, cardID)` | Получить карточку |
| `GetByEntryID(ctx, userID, entryID)` | Получить карточку по entry |
| `GetByEntryIDs(ctx, userID, entryIDs)` | Batch для DataLoader |
| `GetDueCards(ctx, userID, now, limit)` | Очередь: JOIN entries WHERE deleted_at IS NULL, status != MASTERED, overdue first |
| `CountDue(ctx, userID, now)` | Количество due cards |
| `CountNew(ctx, userID)` | Количество NEW cards |
| `CountByStatus(ctx, userID)` | Возвращает **только non-zero** группы |

Запись:

| Метод | Описание |
|-------|----------|
| `Create(ctx, userID, entryID, status, easeFactor)` | Создать карточку |
| `UpdateSRS(ctx, userID, cardID, params)` | Обновить SRS-поля: status, next_review_at, interval_days, ease_factor, learning_step + updated_at |
| `Delete(ctx, userID, cardID)` | Удалить карточку |

**GetDueCards — критический запрос:**
- JOIN entries WHERE `deleted_at IS NULL` — обязательно
- `status != 'MASTERED'`
- `status = 'NEW' OR next_review_at <= $now`
- Сортировка: overdue first (`next_review_at ASC`), затем NEW
- Repo **не** учитывает daily limit — это ответственность сервиса
- `$now` передаётся сервисом (уже в UTC)

**Важно:** поле `learning_step INT NOT NULL DEFAULT 0` (из study_service v1.1) — включено в `UpdateSRS` параметры.

**Corner cases:**
- GetDueCards **не** возвращает карточки soft-deleted entries
- `UNIQUE(entry_id)` — Create при дубликате → `ErrAlreadyExists`
- CountDue и CountNew тоже фильтруют через JOIN entries
- `user_id` в cards — денормализация; корректность (`user_id == entry.user_id`) проверяется сервисом

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `learning_step` включён в `UpdateSRS`
- [ ] GetDueCards не возвращает soft-deleted
- [ ] GetDueCards: overdue first, then new
- [ ] GetDueCards: respects limit
- [ ] CountByStatus: только non-zero группы
- [ ] Create: duplicate entry_id → `ErrAlreadyExists`
- [ ] UpdateSRS: обновляет все SRS-поля + `updated_at`
- [ ] Integration-тесты: create, get due (with soft-deleted), count, update SRS, ordering

---

### TASK-3.11: Review Log Repository

**Зависит от:** TASK-3.10 (нужен card для тестов)

**Контекст:**
- `data_model_v4.md` — §5 (review_logs)
- `repo_layer_spec_v4.md` — §14 (timezone), §19.6 (операции)
- `repo_layer_tasks_v4.md` — TASK-301
- `services/study_service_spec_v4_v1.1.md` — §4 (поле `prev_state JSONB`)

**Что сделать:**

Создать пакет `internal/adapter/postgres/reviewlog/`.

**Операции:**

Чтение:

| Метод | Описание |
|-------|----------|
| `GetByCardID(ctx, cardID, limit, offset)` | Ordered by `reviewed_at DESC` |
| `GetLastByCardID(ctx, cardID)` | Последний review (для undo) |
| `GetByCardIDs(ctx, cardIDs)` | Batch для DataLoader, группировка по `card_id` |
| `CountToday(ctx, userID, dayStart)` | `dayStart` в UTC, от сервиса |
| `GetStreakDays(ctx, userID, timezone, days)` | **Исключение**: единственный метод, принимающий timezone string. `date_trunc('day', reviewed_at AT TIME ZONE $tz)` |

Запись:

| Метод | Описание |
|-------|----------|
| `Create(ctx, reviewLog)` | Создать лог (включая `prev_state JSONB`) |
| `Delete(ctx, id)` | Удалить лог (для undo) |

**prev_state JSONB:**
- `domain.CardSnapshot` → JSONB сериализация/десериализация в repo layer
- Domain-модель `CardSnapshot` не содержит json-тегов
- Repo использует custom marshaling (промежуточная структура с json-тегами или `json.Marshal` с map)
- При `prev_state = NULL` (первый review) → domain `PrevState == nil`

**Corner cases:**
- CountToday: `WHERE reviewed_at >= $dayStart` — dayStart уже в UTC
- GetStreakDays: timezone как строка (e.g. `"Europe/Moscow"`). Документировать как исключение из правила "repo не знает timezone"
- Create: **не идемпотентен** — каждый вызов создаёт новую запись. Защита от дублей — ответственность сервиса

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `prev_state JSONB`: корректная сериализация/десериализация `CardSnapshot`
- [ ] `prev_state = NULL` → `PrevState == nil` в domain
- [ ] CountToday: корректно с boundary (23:59 vs 00:01 по timezone)
- [ ] GetStreakDays: группировка по дням с timezone
- [ ] Batch: группировка по card_id
- [ ] Integration-тесты: create (with prev_state), count today (boundary), streak, batch, delete

---

### TASK-3.12: Topic Repository

**Зависит от:** TASK-3.4 (нужен SeedEntry для тестов M2M)

**Контекст:**
- `data_model_v4.md` — §6 (topics, entry_topics)
- `repo_layer_spec_v4.md` — §9.4 (идемпотентность), §19.9 (операции)
- `repo_layer_tasks_v4.md` — TASK-400

**Что сделать:**

Создать пакет `internal/adapter/postgres/topic/`.

**Операции:**

| Метод | Описание |
|-------|----------|
| `GetByID(ctx, userID, topicID)` | Получить тему |
| `ListByUser(ctx, userID)` | Все темы пользователя, ORDER BY name |
| `GetByEntryID(ctx, entryID)` | Темы конкретного entry (M2M join) |
| `GetByEntryIDs(ctx, entryIDs)` | Batch для DataLoader |
| `GetEntryIDsByTopicID(ctx, topicID)` | Entry IDs в теме |
| `Create(ctx, userID, name, description)` | Создать тему |
| `Update(ctx, userID, topicID, name, description)` | Обновить |
| `Delete(ctx, userID, topicID)` | Удалить |
| `LinkEntry(ctx, entryID, topicID)` | `ON CONFLICT DO NOTHING` |
| `UnlinkEntry(ctx, entryID, topicID)` | Удалить связь |

**Corner cases:**
- Create: duplicate name per user → `ErrAlreadyExists`
- LinkEntry: идемпотентно
- Delete topic: CASCADE удаляет entry_topics, entries **не** затрагиваются

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] Duplicate name → `ErrAlreadyExists`
- [ ] Link идемпотентен
- [ ] Delete не удаляет entries
- [ ] Integration-тесты: CRUD, link/unlink, batch, delete cascade

---

### TASK-3.13: Inbox Repository

**Зависит от:** TASK-3.1 (нужен SeedUser)

**Контекст:**
- `data_model_v4.md` — §6 (inbox_items)
- `repo_layer_spec_v4.md` — §11 (limits), §19.10 (операции)
- `repo_layer_tasks_v4.md` — TASK-401

**Что сделать:**

Создать пакет `internal/adapter/postgres/inbox/`.

**Операции:**

| Метод | Описание |
|-------|----------|
| `GetByID(ctx, userID, itemID)` | Получить элемент |
| `ListByUser(ctx, userID, limit, offset)` | ORDER BY `created_at DESC`, с `totalCount` |
| `CountByUser(ctx, userID)` | Для проверки лимитов |
| `Create(ctx, userID, text, context)` | Создать элемент |
| `Delete(ctx, userID, itemID)` | Удалить элемент |
| `DeleteAll(ctx, userID)` | Очистить весь inbox |

**Corner cases:**
- `ListByUser`: пустой inbox → `[]`, `totalCount = 0`
- `DeleteAll`: идемпотентно

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] Pagination: limit + offset + totalCount
- [ ] CountByUser для лимитов
- [ ] DeleteAll: очищает все items
- [ ] Integration-тесты: CRUD, list pagination, count, delete all

---

### TASK-3.14: Audit Repository

**Зависит от:** TASK-3.1 (нужен SeedUser)

**Контекст:**
- `data_model_v4.md` — §7 (audit_log)
- `repo_layer_spec_v4.md` — §17 (retention), §19.11 (операции)
- `repo_layer_tasks_v4.md` — TASK-402

**Что сделать:**

Создать пакет `internal/adapter/postgres/audit/`.

**Операции:**

| Метод | Описание |
|-------|----------|
| `Create(ctx, record)` | Создать запись аудита |
| `GetByEntity(ctx, entityType, entityID, limit)` | История изменений сущности |
| `GetByUser(ctx, userID, limit, offset)` | Аудит-лог пользователя |

**Corner cases:**
- `changes`: `map[string]any` → JSONB. JSON маршалинг/демаршалинг в repo
- `entity_id` может быть NULL
- Append-only: нет Update/Delete (retention cleanup — отдельный job, не в repo API)

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] JSONB `changes` корректно сериализуются/десериализуются
- [ ] `entity_id = NULL` обрабатывается корректно
- [ ] Integration-тесты: create, get by entity, get by user with pagination

---

### TASK-3.15: DataLoaders

**Зависит от:** все репозитории (TASK-3.1 — TASK-3.14)

**Контекст:**
- `repo_layer_spec_v4.md` — §18 (DataLoaders)
- `code_conventions_v4.md` — §10.3 (DataLoaders)
- `repo_layer_tasks_v4.md` — TASK-500

**Что сделать:**

Создать пакет `internal/transport/graphql/dataloader/`. DataLoaders **вызывают repo напрямую** (не через service). Авторизация обеспечивается SQL (`WHERE user_id`).

**9 DataLoaders:**

| DataLoader | Ключ | Результат | Repo метод |
|------------|------|-----------|------------|
| SensesByEntryID | entry_id | `[]Sense` | sense.GetByEntryIDs |
| TranslationsBySenseID | sense_id | `[]Translation` | translation.GetBySenseIDs |
| ExamplesBySenseID | sense_id | `[]Example` | example.GetBySenseIDs |
| PronunciationsByEntryID | entry_id | `[]RefPronunciation` | pronunciation.GetByEntryIDs |
| CatalogImagesByEntryID | entry_id | `[]RefImage` | image.GetCatalogByEntryIDs |
| UserImagesByEntryID | entry_id | `[]UserImage` | image.GetUserByEntryIDs |
| CardByEntryID | entry_id | `*Card` (nullable) | card.GetByEntryIDs |
| TopicsByEntryID | entry_id | `[]Topic` | topic.GetByEntryIDs |
| ReviewLogsByCardID | card_id | `[]ReviewLog` | reviewlog.GetByCardIDs |

**Middleware:**
- DataLoaders создаются **per-request** в middleware
- Middleware помещает DataLoaders в context
- Helper-функции для извлечения из context

**Параметры:** `maxBatch = 100`, `wait = 2ms`

**Пустые результаты:** `[]` (не nil). `CardByEntryID`: `nil` если нет card.

**Acceptance criteria:**
- [ ] Все 9 DataLoaders реализованы
- [ ] Middleware создаёт per-request и помещает в context
- [ ] Batch: один SQL-запрос на batch (проверить через логи)
- [ ] Пустые результаты: `[]`, не nil
- [ ] `CardByEntryID`: nil для entries без card
- [ ] Helper-функции для извлечения из context
- [ ] Unit-тест middleware: DataLoaders доступны из context

---

## Сводка зависимостей задач

```
Фаза 2 (полностью) ──┬──→ TASK-3.1 (User) ──→ TASK-3.2 (Token)
                       │                     ──→ TASK-3.13 (Inbox)
                       │                     ──→ TASK-3.14 (Audit)
                       │
                       ├──→ TASK-3.3 (RefEntry)
                       │
                       └──→ TASK-3.4 (Entry) ──┬→ TASK-3.5 (Sense) ──→ TASK-3.6 (Translation)
                                                │                    ──→ TASK-3.7 (Example)
                                                ├→ TASK-3.8 (Pronunciation)
                                                ├→ TASK-3.9 (Image)
                                                ├→ TASK-3.10 (Card) ──→ TASK-3.11 (ReviewLog)
                                                └→ TASK-3.12 (Topic)

ВСЕ РЕПОЗИТОРИИ ──→ TASK-3.15 (DataLoaders)
```

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-3.1 (User), TASK-3.3 (RefEntry) |
| 2 | TASK-3.2 (Token), TASK-3.4 (Entry), TASK-3.13 (Inbox), TASK-3.14 (Audit) |
| 3 | TASK-3.5 (Sense), TASK-3.8 (Pronunciation), TASK-3.9 (Image), TASK-3.10 (Card), TASK-3.12 (Topic) |
| 4 | TASK-3.6 (Translation), TASK-3.7 (Example), TASK-3.11 (ReviewLog) |
| 5 | TASK-3.15 (DataLoaders) |

> При полной параллелизации — 5 sequential волн. Волна 3 самая широкая: до 5 задач параллельно.

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят (unit + integration)
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] `make generate` генерирует код для всех sqlc.yaml без ошибок
- [ ] **14 репозиториев** созданы, каждый в своём пакете с sqlc per repo
- [ ] Все операции из `repo_layer_spec_v4.md` §19 реализованы
- [ ] **COALESCE-паттерн** работает для senses, translations, examples
- [ ] **Trigger test** пройден (ref_sense deleted → данные скопированы)
- [ ] **Soft delete** фильтрация во всех запросах к entries и дочерним таблицам
- [ ] **GetDueCards** не возвращает карточки soft-deleted entries
- [ ] **Cursor pagination** работает стабильно для entries
- [ ] **Конкурентные тесты**: GetOrCreate (RefEntry), concurrent entry create
- [ ] **Error mapping**: все cases (not found, duplicate, FK violation)
- [ ] **Position management**: auto-increment при создании, reorder батчем
- [ ] **Batch запросы** для всех DataLoader-compatible методов
- [ ] **9 DataLoaders** реализованы с per-request middleware
- [ ] **prev_state JSONB** в review_log: сериализация/десериализация CardSnapshot
- [ ] **learning_step** включён в Card UpdateSRS
- [ ] Все acceptance criteria всех 15 задач выполнены
