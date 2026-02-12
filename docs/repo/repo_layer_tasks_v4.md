# MyEnglish Backend v4 — Repository Layer: Tasks

> **Дата:** 2026-02-12
> **Документы-источники:**
> - `code_conventions_v4.md` — архитектура, паттерны, правила
> - `data_model_v4.md` — DDL, таблицы, индексы, triggers, ON DELETE
> - `repo_layer_spec_v4.md` — требования к repository layer, corner cases

---

## Как читать этот документ

Каждая задача содержит:

- **Зависит от** — какие задачи должны быть завершены перед началом этой
- **Контекст** — какие секции документов прочитать перед выполнением
- **Что сделать** — описание задачи
- **Acceptance criteria** — когда задача считается завершённой
- **Corner cases** — на что обратить внимание

Задачи сгруппированы по фазам. Внутри фазы задачи можно выполнять параллельно (если зависимости позволяют).

---

## Зафиксированные решения

Перед началом работы ознакомиться со всеми решениями — они влияют на каждую задачу.

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | Module path | `github.com/heartmarshall/myenglish-backend` |
| 2 | Domain models | Отдельная задача TASK-004 в фазе 0 |
| 3 | sqlc организация | **Вариант B — sqlc per repo.** Каждый пакет-репозиторий содержит свой `query/`, `sqlc/`, `sqlc.yaml`. Полная изоляция. |
| 4 | sqlc ENUM overrides | sqlc генерирует свои string-типы для ENUM. Маппинг sqlc enum → domain enum выполняется вручную в функциях маппинга каждого репозитория. |
| 5 | Goose формат | Sequential: `00001_enums.sql`, `00002_ref_catalog.sql`, ... |
| 6 | Test containers | **Один общий контейнер** на весь `adapter/postgres/` — через shared setup в `testhelper`. Не 15 контейнеров. |
| 7 | Filter struct | Тип `Filter` определяется **в пакете entry repo** (`entry.Filter`). Сервис импортирует и конструирует. |
| 8 | Position auto-increment | **Ответственность repo.** При создании — `MAX(position) + 1` автоматически. Сервис не передаёт position при создании, только при явном reorder. Race condition (два concurrent create → одинаковый position) допустим. |
| 9 | GetDueCards daily limit | **Repo не знает о daily limits.** Возвращает все due cards (overdue first, then new, с общим limit). Сервис фильтрует по бизнес-правилам (new cards per day). |
| 10 | GetStreak timezone | **Явное исключение** из правила "repo не знает timezone". GetStreak — единственный метод repo, принимающий timezone string. Причина: `date_trunc('day', reviewed_at AT TIME ZONE $tz)` нельзя заменить без потери корректности при DST. |
| 11 | DataLoaders | **Вызывают repo напрямую**, минуя service. Авторизация обеспечивается SQL (`WHERE user_id`). Бизнес-логики в batch-загрузке нет. |
| 12 | Wiring (main.go) | **Вне scope этого документа.** Будет описано в задачах на service/transport layer. |
| 13 | TxManager isolation level | **Read Committed** (PostgreSQL default). Достаточно для всех сценариев. Concurrent review — last-write-wins (допустимо). |
| 14 | SeedEntry | **Две функции:** `SeedEntry` (без card) и `SeedEntryWithCard`. Большинству тестов card не нужен. |
| 15 | CountByStatus | Repo возвращает **только non-zero** группы. Сервис дополняет недостающие статусы нулями. |
| 16 | Trigger tests (ref deleted) | Тестировать **один раз** в TASK-200 (Sense). Не дублировать в Translation и Example — паттерн идентичен. |
| 17 | Reorder API | Repo принимает `[]struct{ID uuid.UUID, Position int}`, обновляет батчем в одной транзакции. |
| 18 | Makefile / CI | Включено в TASK-001. |

---

## Фаза 0: Инфраструктура

### TASK-000: Миграции

**Зависит от:** ничего

**Контекст:**
- `data_model_v4.md` — полный DDL всех таблиц (секции 2–7), enums (секция 8), triggers (секция 9), ON DELETE сводка (секция 11)
- `repo_layer_spec_v4.md` — секция 2 (формат, порядок, правила миграций), секция 3 (нормализация текста)

**Что сделать:**

Создать 8 goose-миграций в директории `migrations/`. Формат: sequential numbering.

| # | Файл | Содержимое |
|---|------|-----------|
| 1 | `00001_enums.sql` | Все ENUM types: learning_status, review_grade, part_of_speech, entity_type, audit_action |
| 2 | `00002_ref_catalog.sql` | ref_entries, ref_senses, ref_translations, ref_examples, ref_pronunciations, ref_images + все индексы |
| 3 | `00003_users.sql` | users, user_settings, refresh_tokens + все индексы |
| 4 | `00004_user_dictionary.sql` | entries, senses, translations, examples, entry_pronunciations, entry_images, user_images + все индексы |
| 5 | `00005_cards.sql` | cards, review_logs + все индексы |
| 6 | `00006_organization.sql` | topics, entry_topics, inbox_items, audit_log + все индексы |
| 7 | `00007_triggers.sql` | 3 функции + 3 триггера для защиты данных при удалении ref-записей |
| 8 | `00008_extensions.sql` | pg_trgm extension + GIN-индексы для fuzzy search |

**Acceptance criteria:**
- [ ] Каждая миграция содержит `-- +goose Up` и `-- +goose Down`
- [ ] Down-миграции корректно откатывают всё (DROP в обратном порядке зависимостей)
- [ ] `goose up` + `goose down` + `goose up` выполняется без ошибок
- [ ] Все таблицы, индексы, constraints, triggers из data_model_v4.md присутствуют
- [ ] Unique constraints: `ux_ref_entries_text_norm`, `ux_entries_user_text` (partial — WHERE deleted_at IS NULL), `ux_users_email`, `ux_users_oauth`, `ux_topics_user_name`, `ux_cards_entry`
- [ ] Все FK ON DELETE поведение соответствует таблице из data_model_v4.md секция 11
- [ ] ENUM down-миграция: DROP TYPE в правильном порядке (нет зависимых таблиц к моменту дропа)

**Corner cases:**
- Миграция 007 (triggers) зависит от таблиц из миграций 002 и 004 — порядок критичен
- `ALTER TYPE ... ADD VALUE` нельзя откатить — если в будущем добавляется значение в enum, документировать в комментарии

---

### TASK-001: Базовые компоненты postgres-пакета + Makefile

**Зависит от:** ничего (может делаться параллельно с TASK-000)

**Контекст:**
- `code_conventions_v4.md` — секция 8 (работа с БД: Querier, транзакции, sqlc + Squirrel)
- `repo_layer_spec_v4.md` — секции 6 (принципы реализации), 7 (транзакции), 15 (timeouts)

**Что сделать:**

Создать в `internal/adapter/postgres/` общие компоненты, используемые всеми репозиториями:

**postgres.go:**
- Тип `Querier` — интерфейс с методами `Exec`, `Query`, `QueryRow` (реализуется и `*pgxpool.Pool`, и `pgx.Tx`)
- Функция `QuerierFromCtx(ctx, pool)` — извлекает tx из context или возвращает pool

**pool.go:**
- Функция создания `*pgxpool.Pool` из `config.DatabaseConfig`
- Конфигурация MaxConns, MinConns, MaxConnLifetime, MaxConnIdleTime
- Ping для проверки соединения при старте

**txmanager.go:**
- Тип `TxManager` с методом `RunInTx(ctx, fn)`
- Isolation level: **Read Committed** (PostgreSQL default), не конфигурируется
- При panic внутри fn — rollback + re-panic
- Nested RunInTx не поддерживается — документировать в комментарии

**errors.go:**
- Функция маппинга pgx/pgconn ошибок в domain-ошибки
- Покрытие: ErrNoRows → ErrNotFound, unique_violation (23505) → ErrAlreadyExists, fk_violation (23503) → ErrNotFound, check_violation (23514) → ErrValidation
- `context.DeadlineExceeded` и `context.Canceled` **не** маппятся — прокидываются как есть
- Ошибки оборачиваются с контекстом (entity name, ID)

**Makefile:**
- `make run` — запуск сервера
- `make build` — сборка бинарника
- `make test` — `go test ./... -race -count=1`
- `make test-cover` — coverage report
- `make generate` — `go generate ./...`
- `make migrate-up` / `make migrate-down` — goose миграции
- `make migrate-create name=...` — создание новой миграции
- `make docker-up` / `make docker-down` — docker compose для PostgreSQL
- `make lint` — golangci-lint

**Acceptance criteria:**
- [ ] `Querier` interface определён, реализуется pool и tx
- [ ] `QuerierFromCtx` возвращает tx из контекста если есть, иначе pool
- [ ] `TxManager.RunInTx` коммитит при успехе, откатывает при ошибке и при panic
- [ ] Error mapping покрывает все 4 кейса из таблицы
- [ ] context.DeadlineExceeded не маппится в domain-ошибку
- [ ] Все Makefile targets работают
- [ ] Unit-тесты для TxManager (success, error rollback, panic rollback)
- [ ] Unit-тесты для error mapping

---

### TASK-002: Конфигурация sqlc (шаблон)

**Зависит от:** TASK-000 (миграции — sqlc читает schema из них)

**Контекст:**
- `code_conventions_v4.md` — секция 8.2 (sqlc configuration)
- `repo_layer_spec_v4.md` — секция 4 (sqlc vs Squirrel)

**Что сделать:**

Создать **шаблонный** `sqlc.yaml`, который каждый репозиторий скопирует и адаптирует. Организация: **sqlc per repo** — каждый пакет содержит свой `query/`, `sqlc/`, `sqlc.yaml`.

Шаблон sqlc.yaml:

- engine: postgresql
- sql_package: pgx/v5
- `emit_empty_slices: true`
- `emit_json_tags: false`
- Overrides: UUID → `github.com/google/uuid.UUID`, TIMESTAMPTZ → `time.Time`, nullable TEXT → `*string`
- **ENUM overrides не добавляются** — sqlc генерирует свои string types, маппинг в domain enums выполняется вручную в repo

Пример структуры одного repo:
```
internal/adapter/postgres/entry/
├── query/
│   └── entries.sql          # sqlc queries
├── sqlc/                    # generated
│   ├── db.go
│   ├── models.go
│   └── entries.sql.go
├── sqlc.yaml                # config pointing to query/ and ../../migrations/
├── repo.go
├── filter.go
└── repo_test.go
```

Обновить `make generate`:
- Найти все `sqlc.yaml` в поддиректориях `adapter/postgres/` и запустить `sqlc generate` для каждого

**Acceptance criteria:**
- [ ] Шаблон sqlc.yaml создан и задокументирован
- [ ] Пример одного repo (можно entry/) демонстрирует структуру
- [ ] `make generate` рекурсивно находит и обрабатывает все sqlc.yaml
- [ ] Сгенерированные модели: UUID, time.Time, *string маппятся корректно
- [ ] `emit_empty_slices: true` — пустые результаты `[]`, не `nil`

---

### TASK-003: Test Helpers

**Зависит от:** TASK-000, TASK-001

**Контекст:**
- `repo_layer_spec_v4.md` — секция 20 (тестирование: подход, helpers, правила)

**Что сделать:**

Создать пакет `internal/adapter/postgres/testhelper/`.

**db.go:**
- Функция `SetupTestDB(t)` → `*pgxpool.Pool`
- Один PostgreSQL контейнер (postgres:17-alpine) на весь `adapter/postgres/`
- Контейнер создаётся через `sync.Once` и переиспользуется всеми test-пакетами
- Применение всех goose-миграций при первом запуске
- `t.Cleanup` для закрытия pool (контейнер живёт до конца всех тестов)

**seed.go:**
- `SeedUser(t, pool)` → user + user_settings, возвращает domain.User
- `SeedRefEntry(t, pool, text)` → ref_entry + 2 ref_senses (каждый с 2 ref_translations + 2 ref_examples) + 2 ref_pronunciations, возвращает domain.RefEntry с заполненным деревом
- `SeedEntry(t, pool, userID, refEntryID)` → entry + senses (linked to ref) + translations + examples + pronunciations (M2M), **без card**, возвращает domain.Entry
- `SeedEntryWithCard(t, pool, userID, refEntryID)` → то же + card со статусом NEW, возвращает domain.Entry (с заполненным Card)
- `SeedEntryCustom(t, pool, userID)` → entry + custom senses (без ref links, с заполненными definition/pos), возвращает domain.Entry
- Все seed-функции используют `t.Helper()` и `require` для fail-fast

**Corner cases:**
- Один контейнер = общая БД. Тесты не должны конфликтовать: каждый создаёт уникальные данные (unique emails, unique texts). Seed-функции генерируют уникальные значения (UUID-based suffix в email/text).
- SeedRefEntry: конкретный набор данных важен — 2 senses с children достаточно для тестов COALESCE, partial customization, batch loading.

**Acceptance criteria:**
- [ ] `SetupTestDB` поднимает контейнер и применяет миграции
- [ ] Контейнер переиспользуется (не поднимается на каждый тест и каждый пакет)
- [ ] Seed-функции создают корректные данные с уникальными значениями
- [ ] SeedEntry и SeedEntryWithCard — две отдельные функции
- [ ] Seed-функции возвращают заполненные domain-модели для assertions
- [ ] Тесты из разных пакетов могут работать параллельно с общим контейнером

---

### TASK-004: Domain Models

**Зависит от:** ничего (может делаться параллельно с TASK-000, TASK-001)

**Контекст:**
- `data_model_v4.md` — вся схема, все таблицы и их поля
- `code_conventions_v4.md` — секция 2 (ошибки), секция 1 (структура domain пакета)

**Что сделать:**

Создать пакет `internal/domain/` со всеми типами, которые используются repository и service слоями.

**enums.go:**
- LearningStatus (NEW, LEARNING, REVIEW, MASTERED)
- ReviewGrade (AGAIN, HARD, GOOD, EASY)
- PartOfSpeech (NOUN, VERB, ADJECTIVE, ADVERB, PRONOUN, PREPOSITION, CONJUNCTION, INTERJECTION, PHRASE, IDIOM, OTHER)
- EntityType (ENTRY, SENSE, EXAMPLE, IMAGE, PRONUNCIATION, CARD)
- AuditAction (CREATE, UPDATE, DELETE)
- OAuthProvider (google, apple)

**errors.go:**
- Sentinel errors: ErrNotFound, ErrAlreadyExists, ErrValidation, ErrUnauthorized, ErrForbidden, ErrConflict
- ValidationError с FieldError slice, Unwrap → ErrValidation
- Конструкторы: NewValidationError(field, message), NewValidationErrors([]FieldError)

**user.go:**
- User (ID, Email, Name, AvatarURL, OAuthProvider, OAuthID, CreatedAt, UpdatedAt)
- UserSettings (UserID, NewCardsPerDay, ReviewsPerDay, MaxIntervalDays, Timezone, UpdatedAt)
- DefaultUserSettings(userID) → UserSettings с дефолтами
- RefreshToken (ID, UserID, TokenHash, ExpiresAt, CreatedAt, RevokedAt *time.Time)
- Методы: IsRevoked(), IsExpired(now)

**reference.go:**
- RefEntry (ID, Text, TextNormalized, CreatedAt + slices для Senses, Pronunciations, Images)
- RefSense (ID, RefEntryID, Definition, PartOfSpeech, CEFRLevel, SourceSlug, Position, CreatedAt + slices для Translations, Examples)
- RefTranslation (ID, RefSenseID, Text, SourceSlug, Position)
- RefExample (ID, RefSenseID, Sentence, Translation, SourceSlug, Position)
- RefPronunciation (ID, RefEntryID, Transcription, AudioURL, Region, SourceSlug)
- RefImage (ID, RefEntryID, URL, Caption, SourceSlug)

**entry.go:**
- Entry (ID, UserID, RefEntryID *uuid, Text, TextNormalized, Notes, CreatedAt, UpdatedAt, DeletedAt *time.Time + slices для Senses, Pronunciations, CatalogImages, UserImages, Card, Topics)
- IsDeleted() bool
- Sense (ID, EntryID, RefSenseID *uuid, Definition *string, PartOfSpeech *PartOfSpeech, CEFRLevel *string, SourceSlug, Position, CreatedAt + slices)
- Translation (ID, SenseID, RefTranslationID *uuid, Text *string, SourceSlug, Position)
- Example (ID, SenseID, RefExampleID *uuid, Sentence *string, Translation *string, SourceSlug, Position, CreatedAt)
- UserImage (ID, EntryID, URL, Caption, CreatedAt)

**card.go:**
- Card (ID, UserID, EntryID, Status, NextReviewAt *time.Time, IntervalDays, EaseFactor, CreatedAt, UpdatedAt)
- IsDue(now) bool
- ReviewLog (ID, CardID, Grade, DurationMs *int, ReviewedAt)
- SRSResult (Status, NextReviewAt, IntervalDays, EaseFactor) — value object для результата SRS-вычисления

**organization.go:**
- Topic (ID, UserID, Name, Description, CreatedAt, UpdatedAt)
- InboxItem (ID, UserID, Text, Context, CreatedAt)
- AuditRecord (ID, UserID, EntityType, EntityID *uuid, Action, Changes map[string]any, CreatedAt)

**normalize.go:**
- Функция `NormalizeText(text string) string` — правила из repo_layer_spec секция 3:
  - Trim пробелы по краям
  - Lower case
  - Сжать множественные пробелы в один
  - НЕ убирать диакритику, дефисы, апострофы
- Используется service-слоем перед передачей в repo

**Acceptance criteria:**
- [ ] Все типы из data_model_v4.md представлены в domain
- [ ] Nullable поля: pointer types (*string, *uuid.UUID, *time.Time)
- [ ] Enums: string-based constants
- [ ] Errors: sentinel errors + ValidationError с Unwrap
- [ ] NormalizeText: unit-тесты на все правила (trim, lower, spaces, diacritics, hyphens, apostrophes)
- [ ] Методы: IsDeleted(), IsDue(), IsRevoked(), IsExpired() — с unit-тестами

---

## Фаза 1: Core Repositories

### TASK-100: User Repository

**Зависит от:** TASK-001, TASK-002, TASK-003, TASK-004

**Контекст:**
- `data_model_v4.md` — секция 3 (users, user_settings)
- `repo_layer_spec_v4.md` — секции 6 (принципы), 19.7 (операции users)

**Что сделать:**

Создать пакет `internal/adapter/postgres/user/` с собственным sqlc (query/, sqlc/, sqlc.yaml) и repo.go.

**Операции:**
- GetByID, GetByOAuth (provider + oauth_id), GetByEmail
- CreateUser
- UpdateUser (name, avatar)
- GetSettings, CreateSettings, UpdateSettings

**Corner cases:**
- `CreateUser` при дубликате email → `domain.ErrAlreadyExists`
- `CreateUser` при дубликате (provider, oauth_id) → `domain.ErrAlreadyExists`
- `GetByOAuth` — основной метод для OAuth login flow, должен быть быстрым (indexed)
- Маппинг sqlc enum → domain.OAuthProvider выполняется в функции маппинга

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] sqlc per repo: собственный query/, sqlc/, sqlc.yaml
- [ ] Error mapping: not found → ErrNotFound, duplicate → ErrAlreadyExists
- [ ] Integration-тесты: CRUD happy path, not found, duplicate email, duplicate OAuth
- [ ] `make generate` обрабатывает sqlc.yaml этого пакета

---

### TASK-101: Token Repository

**Зависит от:** TASK-100 (нужен SeedUser)

**Контекст:**
- `data_model_v4.md` — секция 3 (refresh_tokens)
- `repo_layer_spec_v4.md` — секции 19.8 (операции), 17 (data retention)

**Что сделать:**

Создать пакет `internal/adapter/postgres/token/`.

**Операции:**
- Create (user_id, token_hash, expires_at)
- GetByHash (только активные: `WHERE revoked_at IS NULL`)
- RevokeByID
- RevokeAllByUser (logout everywhere)
- DeleteExpired (cleanup)

**Corner cases:**
- `GetByHash` возвращает ErrNotFound для revoked токенов
- `RevokeByID` для уже revoked — идемпотентно, не ошибка
- `DeleteExpired` — может удалять тысячи записей, без транзакции
- Cleanup job (кто вызывает DeleteExpired) — вне scope repo layer, будет в service/scheduler

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `GetByHash` не возвращает revoked
- [ ] `RevokeAllByUser` затрагивает только активные
- [ ] Integration-тесты: create + get, revoke + get (not found), revoke all, delete expired

---

### TASK-102: Reference Catalog Repository

**Зависит от:** TASK-001, TASK-002, TASK-003, TASK-004

**Контекст:**
- `data_model_v4.md` — секция 2 (все ref_ таблицы)
- `repo_layer_spec_v4.md` — секции 9.1 (конкурентное заполнение), 10 (управление каталогом), 19.1 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/refentry/`.

Этот репозиторий управляет **6 таблицами** (ref_entries + 5 дочерних) как единым агрегатом. Каталог заполняется атомарно — одна транзакция.

**Операции:**

Чтение:
- GetByID — ref_entry с полным деревом (senses → translations, examples; pronunciations; images). Реализация: отдельные запросы к каждой ref-таблице (не один гигантский JOIN). Это согласуется с подходом DataLoaders.
- GetByNormalizedText — для проверки существования перед вызовом внешнего API
- Search — fuzzy search по text_normalized (pg_trgm). Squirrel или raw SQL. При пустом query — пустой результат без запроса к БД.

Запись:
- Create — создать ref_entry + все дочерние в одной транзакции (через TxManager)
- GetOrCreate — upsert-семантика: `INSERT ... ON CONFLICT (text_normalized) DO NOTHING`, затем `SELECT`. Возвращает ref_entry (новый или существующий).

Batch:
- GetRefSensesByIDs, GetRefTranslationsByIDs, GetRefExamplesByIDs, GetRefPronunciationsByIDs, GetRefImagesByIDs — для отображения origin info в UI. На MVP могут быть не задействованы, но подготовить для DataLoaders.

**Corner cases:**
- **Конкурентное создание**: INSERT ON CONFLICT DO NOTHING + SELECT. Ни один из параллельных запросов не получает ошибку.
- **Транзакция Create**: если ошибка при создании ref_translation — откат всего, включая ref_entry и ref_senses
- **Search**: `ORDER BY similarity(text_normalized, $query) DESC LIMIT $limit`
- **Каталог immutable**: нет Update/Delete операций для ref_entries в repo API (trigger для защиты данных при удалении — отдельная история, handled by DB)

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] `GetOrCreate` работает без race conditions — тест с горутинами
- [ ] `Create` — атомарная транзакция для всего дерева
- [ ] При ошибке в Create — откат всего
- [ ] `Search` использует pg_trgm
- [ ] Batch-запросы корректны для массива UUID
- [ ] Integration-тесты: create full tree, get by text, GetOrCreate (concurrent), search, batch

---

### TASK-103: Entry Repository

**Зависит от:** TASK-001, TASK-002, TASK-003, TASK-004, TASK-102 (нужен SeedRefEntry)

**Контекст:**
- `data_model_v4.md` — секция 4 (entries)
- `repo_layer_spec_v4.md` — секции 3 (нормализация), 4.3 (Squirrel), 8 (soft delete), 9.2 (конкурентное добавление), 11 (limits), 13 (pagination), 19.2 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/entry/` с repo.go + filter.go.

**Операции:**

Чтение:
- GetByID (user_id + deleted_at фильтр)
- GetByText (user_id + text_normalized + deleted_at)
- Find (Squirrel — динамические фильтры)
- GetByIDs (batch)
- CountByUser (для лимитов)

Запись:
- Create
- UpdateNotes
- SoftDelete (set deleted_at = now())
- Restore (set deleted_at = NULL)

Batch:
- HardDeleteOld (deleted_at < $threshold)

**filter.go:**

Тип `Filter` определяется **в этом пакете** (не в сервисе). Содержит:
- Search *string
- HasCard *bool
- PartOfSpeech *domain.PartOfSpeech
- TopicID *uuid.UUID
- Status *domain.LearningStatus
- SortBy string (допустимые значения: "text", "created_at", "updated_at")
- SortOrder string ("ASC", "DESC")
- Limit int (default 50, max 200)
- Offset int (для offset-based)
- Cursor *string (для cursor-based)

Find возвращает:
- Offset mode: `[]domain.Entry`, `totalCount int`, `error`
- Cursor mode: `[]domain.Entry`, `hasNextPage bool`, `error`

**Cursor pagination:**
- Cursor = base64(sort_value + "|" + entry_id)
- sort_value кодирование: string as-is, time.Time как RFC3339
- Keyset: `WHERE (sort_field, id) > ($cursor_value, $cursor_id)`
- При невалидном cursor — вернуть ошибку domain.ErrValidation

**Corner cases:**
- Soft delete: все GET-запросы фильтруют `deleted_at IS NULL`
- Re-create after soft delete: partial unique constraint позволяет
- Конкурентное создание: duplicate → ErrAlreadyExists
- Find с нулём фильтров: все entries пользователя
- Find с пустым Search: игнорировать фильтр (не ILIKE '%%')
- CountByUser: только не-удалённые
- HardDeleteOld: батчами по 100 (LIMIT в DELETE)
- **UpdateText не реализуется на MVP** — редактирование текста слова слишком рискованно (unique constraint, нормализация, каскадные последствия). Если понадобится — отдельная задача.

**Acceptance criteria:**
- [ ] Все операции реализованы (кроме UpdateText)
- [ ] Soft delete: GetByID не возвращает soft-deleted
- [ ] SoftDelete идемпотентен
- [ ] Restore работает
- [ ] Find: каждый фильтр работает отдельно и в комбинации
- [ ] Find: offset и cursor pagination
- [ ] Find: totalCount не зависит от limit/offset
- [ ] Cursor: стабильная пагинация, невалидный cursor → ErrValidation
- [ ] HardDeleteOld: удаляет только старше threshold
- [ ] Integration-тесты: CRUD, soft delete/restore, re-create, Find (каждый фильтр, combined, pagination, cursor, empty result), concurrent create

---

## Фаза 2: Content Repositories

Все три COALESCE-репозитория следуют одному паттерну. Начинать с TASK-200, затем TASK-201/202 по аналогии.

### TASK-200: Sense Repository

**Зависит от:** TASK-103 (нужен SeedEntry)

**Контекст:**
- `data_model_v4.md` — секция 4 (senses)
- `repo_layer_spec_v4.md` — секции 5 (COALESCE), 12 (position management), 19.3 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/sense/`.

**Операции:**

Чтение:
- GetByEntryID — с COALESCE (LEFT JOIN ref_senses), ORDER BY position
- GetByEntryIDs — batch для DataLoader, с COALESCE. Результат содержит entry_id для группировки.
- GetByID — единичный, с COALESCE
- CountByEntry — для лимитов

Запись:
- CreateFromRef (entry_id, ref_sense_id, source_slug) — position авто: MAX(position)+1. Поля definition/pos/cefr остаются NULL → COALESCE подхватит ref.
- CreateCustom (entry_id, definition, part_of_speech, cefr_level, source_slug) — position авто. Без ref_sense_id.
- Update (sense_id, definition, part_of_speech, cefr_level) — **ref_sense_id НЕ трогается** (origin link)
- Delete
- Reorder — принимает `[]struct{ID, Position}`, обновляет батчем в одной транзакции

**COALESCE-поля:** definition, part_of_speech, cefr_level

**Corner cases:**
- Partial customization: изменить definition, оставить pos из ref — корректное поведение
- Position auto-increment: race condition при concurrent create → допустим (дубликат позиции не ломает ORDER BY)
- Reorder: транзакция, чтобы все позиции обновились атомарно
- Batch (GetByEntryIDs): результат нужен для DataLoader — каждая строка содержит entry_id
- **Trigger test**: создать sense с ref_sense_id → удалить ref_sense → проверить что trigger скопировал данные в user-поля, sense сохранился, ref_sense_id стал NULL. Этот тест покрывает trigger-паттерн для всех трёх COALESCE-таблиц (не дублировать в TASK-201, TASK-202).

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] COALESCE: user-поля NULL → ref-значения
- [ ] COALESCE: user-поля заполнены → user-значения
- [ ] Partial customization работает
- [ ] Update не обнуляет ref_sense_id
- [ ] Position: автоинкремент при создании
- [ ] Reorder: батч в транзакции
- [ ] Trigger: ref_sense удалён → данные скопированы в user-поля
- [ ] Integration-тесты: create from ref, create custom, update (partial), COALESCE, trigger, reorder, batch

---

### TASK-201: Translation Repository

**Зависит от:** TASK-200 (аналогичный паттерн)

**Контекст:**
- `data_model_v4.md` — секция 4 (translations)
- `repo_layer_spec_v4.md` — секции 5 (COALESCE), 12 (position), 19.3 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/translation/`. Паттерн идентичен TASK-200, но:

- Parent: sense_id (не entry_id)
- COALESCE-поле: только `text`
- Ref-таблица: ref_translations
- Batch ключ: sense_id

**Операции:** GetBySenseID, GetBySenseIDs (batch), GetByID, CountBySense, CreateFromRef, CreateCustom, Update, Delete, Reorder.

**Corner cases:** аналогично TASK-200. Trigger тест **не дублировать** — покрыт в TASK-200.

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] COALESCE для поля text
- [ ] Position auto-increment, reorder
- [ ] Integration-тесты: create from ref, create custom, update, COALESCE, batch

---

### TASK-202: Example Repository

**Зависит от:** TASK-200 (аналогичный паттерн)

**Контекст:**
- `data_model_v4.md` — секция 4 (examples)
- `repo_layer_spec_v4.md` — секции 5 (COALESCE), 12 (position), 19.3 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/example/`. Паттерн идентичен TASK-200, но:

- Parent: sense_id
- COALESCE-поля: `sentence` и `translation`
- Ref-таблица: ref_examples
- Batch ключ: sense_id

**Операции:** аналогичны TASK-201.

**Corner cases:** аналогично. Trigger тест не дублировать.

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] COALESCE для полей sentence и translation
- [ ] Integration-тесты: create from ref, create custom, update, COALESCE, batch

---

### TASK-203: Pronunciation Repository

**Зависит от:** TASK-103

**Контекст:**
- `data_model_v4.md` — секция 4 (entry_pronunciations M2M)
- `repo_layer_spec_v4.md` — секции 9.4 (идемпотентность), 19.4 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/pronunciation/`.

M2M-only репозиторий — пользователь привязывает ref_pronunciations к entry, не создаёт своих.

**Операции:**
- GetByEntryID → []RefPronunciation (JOIN entry_pronunciations + ref_pronunciations)
- GetByEntryIDs → batch для DataLoader (результат содержит entry_id)
- Link (entry_id, ref_pronunciation_id) — ON CONFLICT DO NOTHING
- Unlink (entry_id, ref_pronunciation_id)
- UnlinkAll (entry_id)

**Corner cases:**
- Link дважды → не ошибка
- Unlink несуществующей связи → не ошибка (affected 0 rows — ok)
- GetByEntryID без произношений → пустой slice

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] Link идемпотентен
- [ ] Batch: группировка по entry_id
- [ ] Integration-тесты: link, link duplicate, unlink, get, batch

---

### TASK-204: Image Repository

**Зависит от:** TASK-103

**Контекст:**
- `data_model_v4.md` — секция 4 (entry_images M2M, user_images)
- `repo_layer_spec_v4.md` — секция 19.4 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/image/`.

Два типа изображений в одном репозитории:
- Каталожные (M2M через entry_images) — аналогично pronunciation
- Пользовательские (user_images) — CRUD

**Операции:**

Каталожные: GetCatalogByEntryID, GetCatalogByEntryIDs (batch), LinkCatalog, UnlinkCatalog

Пользовательские: GetUserByEntryID, GetUserByEntryIDs (batch), CreateUser, DeleteUser

**Corner cases:**
- LinkCatalog: идемпотентно
- Batch: два набора запросов (catalog и user), оба с группировкой по entry_id
- DeleteUser: FK cascade на entry обеспечивает ownership (если entry deleted — user_images тоже)

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] LinkCatalog идемпотентен
- [ ] Batch для обоих типов
- [ ] Integration-тесты для catalog и user images

---

## Фаза 3: Study Repositories

### TASK-300: Card Repository

**Зависит от:** TASK-103

**Контекст:**
- `data_model_v4.md` — секция 5 (cards)
- `repo_layer_spec_v4.md` — секции 8 (soft delete + cards), 9.3 (конкурентный review), 14 (timezone), 19.5 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/card/`.

**Операции:**

Чтение:
- GetByID (user_id)
- GetByEntryID (user_id)
- GetByEntryIDs (batch для DataLoader)
- GetDueCards (user_id, now, limit) — **обязательно** JOIN entries WHERE deleted_at IS NULL
- CountDue (user_id, now)
- CountNew (user_id)
- CountByStatus (user_id) — возвращает **только non-zero** группы; сервис дополняет нулями

Запись:
- Create (user_id, entry_id, status, ease_factor)
- UpdateSRS (card_id, user_id, status, next_review_at, interval_days, ease_factor)
- Delete (card_id, user_id)

**GetDueCards — критический запрос:**

Фильтры:
- JOIN entries WHERE deleted_at IS NULL — обязательно
- status != 'MASTERED'
- status = 'NEW' OR next_review_at <= $now

Сортировка: overdue first (next_review_at ASC), затем NEW.

Repo **не** учитывает daily limit на new cards — это ответственность сервиса.

`$now` передаётся сервисом (уже в UTC, с учётом timezone).

**Corner cases:**
- GetDueCards **не** возвращает карточки soft-deleted entries
- UNIQUE(entry_id) — Create при дубликате → ErrAlreadyExists
- CountDue и CountNew тоже фильтруют через JOIN entries
- user_id в cards — денормализация; корректность (user_id == entry.user_id) проверяется сервисом

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] GetDueCards не возвращает soft-deleted
- [ ] GetDueCards: overdue first, then new
- [ ] GetDueCards: respects limit
- [ ] CountByStatus: только non-zero группы
- [ ] Create: duplicate entry_id → ErrAlreadyExists
- [ ] UpdateSRS: обновляет все SRS-поля + updated_at
- [ ] Integration-тесты: create, get due (with soft-deleted), count, update SRS, ordering

---

### TASK-301: Review Log Repository

**Зависит от:** TASK-300

**Контекст:**
- `data_model_v4.md` — секция 5 (review_logs)
- `repo_layer_spec_v4.md` — секции 14 (timezone), 19.6 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/reviewlog/`.

**Операции:**

Чтение:
- GetByCardID (ordered by reviewed_at DESC)
- GetByCardIDs (batch для DataLoader, группировка по card_id)
- CountToday (user_id, dayStart) — dayStart в UTC, от сервиса
- GetStreak (user_id, timezone string, days int) — **исключение**: единственный метод repo, принимающий timezone напрямую. Причина: `date_trunc('day', reviewed_at AT TIME ZONE $tz)` нельзя заменить pre-computed ranges из-за DST.

Запись:
- Create (card_id, grade, duration_ms)

**Corner cases:**
- CountToday: `WHERE reviewed_at >= $dayStart` — dayStart уже в UTC
- GetStreak: timezone как строка (e.g. "Europe/Moscow"). Использование: `date_trunc('day', reviewed_at AT TIME ZONE $tz)`. Документировать как исключение.
- Create: **не идемпотентен** — каждый вызов создаёт новую запись. Защита от дублей — ответственность сервиса.

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] CountToday: корректно с boundary (23:59 vs 00:01 по timezone)
- [ ] GetStreak: группировка по дням с timezone
- [ ] Batch: группировка по card_id
- [ ] Integration-тесты: create, count today (boundary), streak, batch

---

## Фаза 4: Organization Repositories

### TASK-400: Topic Repository

**Зависит от:** TASK-103

**Контекст:**
- `data_model_v4.md` — секция 6 (topics, entry_topics)
- `repo_layer_spec_v4.md` — секции 9.4 (идемпотентность), 19.9 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/topic/`.

**Операции:**

Чтение: GetByID (user_id), ListByUser (ORDER BY name), GetByEntryID (M2M join), GetByEntryIDs (batch), GetEntryIDsByTopicID.

Запись: Create, Update (name, description), Delete, LinkEntry (ON CONFLICT DO NOTHING), UnlinkEntry.

**Corner cases:**
- Create: duplicate name per user → ErrAlreadyExists
- LinkEntry: идемпотентно
- Delete topic: CASCADE удаляет entry_topics, entries **не** затрагиваются

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] Duplicate name → ErrAlreadyExists
- [ ] Link идемпотентен
- [ ] Delete не удаляет entries
- [ ] Integration-тесты: CRUD, link/unlink, batch, delete cascade

---

### TASK-401: Inbox Repository

**Зависит от:** TASK-100 (нужен SeedUser)

**Контекст:**
- `data_model_v4.md` — секция 6 (inbox_items)
- `repo_layer_spec_v4.md` — секции 11 (limits), 19.10 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/inbox/`.

**Операции:** GetByID (user_id), ListByUser (limit, offset, ORDER BY created_at DESC), CountByUser, Create, Delete, DeleteAll.

**Corner cases:**
- ListByUser: пустой inbox → `[]`, totalCount = 0
- DeleteAll: идемпотентно

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] Pagination: limit + offset
- [ ] CountByUser для лимитов
- [ ] DeleteAll: очищает все items
- [ ] Integration-тесты: CRUD, list pagination, count, delete all

---

### TASK-402: Audit Repository

**Зависит от:** TASK-100

**Контекст:**
- `data_model_v4.md` — секция 7 (audit_log)
- `repo_layer_spec_v4.md` — секции 17 (retention), 19.11 (операции)

**Что сделать:**

Создать пакет `internal/adapter/postgres/audit/`.

**Операции:** Create, GetByEntity (entity_type, entity_id, limit), GetByUser (user_id, limit, offset).

**Corner cases:**
- changes: `map[string]any` → JSONB. JSON маршалинг/демаршалинг.
- entity_id может быть NULL
- Append-only: нет Update/Delete (кроме retention cleanup, не в repo API)

**Acceptance criteria:**
- [ ] Все операции реализованы
- [ ] JSONB changes корректно сериализуются/десериализуются
- [ ] Integration-тесты: create, get by entity, get by user with pagination

---

## Фаза 5: DataLoaders

### TASK-500: DataLoaders

**Зависит от:** все репозитории фаз 1–4

**Контекст:**
- `repo_layer_spec_v4.md` — секция 18 (DataLoaders)
- `code_conventions_v4.md` — секция 10.3 (DataLoaders)

**Что сделать:**

Создать пакет `internal/transport/graphql/dataloader/`.

DataLoaders **вызывают repo напрямую** (не через service). Авторизация обеспечивается SQL.

**9 DataLoaders:**

| DataLoader | Ключ | Результат | Repo метод |
|------------|------|-----------|------------|
| SensesByEntryID | entry_id | []Sense | sense.Repo.GetByEntryIDs |
| TranslationsBySenseID | sense_id | []Translation | translation.Repo.GetBySenseIDs |
| ExamplesBySenseID | sense_id | []Example | example.Repo.GetBySenseIDs |
| PronunciationsByEntryID | entry_id | []RefPronunciation | pronunciation.Repo.GetByEntryIDs |
| CatalogImagesByEntryID | entry_id | []RefImage | image.Repo.GetCatalogByEntryIDs |
| UserImagesByEntryID | entry_id | []UserImage | image.Repo.GetUserByEntryIDs |
| CardByEntryID | entry_id | *Card (nullable) | card.Repo.GetByEntryIDs |
| TopicsByEntryID | entry_id | []Topic | topic.Repo.GetByEntryIDs |
| ReviewLogsByCardID | card_id | []ReviewLog | reviewlog.Repo.GetByCardIDs |

**Требования:**
- Per-request middleware: создаёт DataLoaders и помещает в context
- Параметры: maxBatch = 100, wait = 2ms
- Пустой результат: `[]` (не nil). CardByEntryID: nil если нет card.

**Acceptance criteria:**
- [ ] Все 9 DataLoaders реализованы
- [ ] Middleware создаёт per-request
- [ ] Batch: один SQL-запрос на batch
- [ ] Пустые результаты: `[]`, не nil
- [ ] CardByEntryID: nil для entries без card

---

## Сводка зависимостей

```
TASK-000 (Migrations) ─────────────────────┐
TASK-001 (Base + Makefile) ────────────────┤
TASK-004 (Domain Models) ─────────────────┤
                                           ├──→ TASK-002 (sqlc Template)
                                           ├──→ TASK-003 (Test Helpers)
                                           │
                                           ├──→ TASK-100 (User) ──→ TASK-101 (Token)
                                           │                     ──→ TASK-401 (Inbox)
                                           │                     ──→ TASK-402 (Audit)
                                           │
                                           ├──→ TASK-102 (RefEntry)
                                           │
                                           └──→ TASK-103 (Entry) ──┬→ TASK-200 (Sense)
                                                                   │  ├→ TASK-201 (Translation)
                                                                   │  └→ TASK-202 (Example)
                                                                   ├→ TASK-203 (Pronunciation)
                                                                   ├→ TASK-204 (Image)
                                                                   ├→ TASK-300 (Card) ──→ TASK-301 (ReviewLog)
                                                                   └→ TASK-400 (Topic)

ВСЕ РЕПОЗИТОРИИ ──→ TASK-500 (DataLoaders)
```

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-000, TASK-001, TASK-004 |
| 2 | TASK-002, TASK-003 |
| 3 | TASK-100, TASK-102 |
| 4 | TASK-101, TASK-103, TASK-401, TASK-402 |
| 5 | TASK-200, TASK-203, TASK-204, TASK-300, TASK-400 |
| 6 | TASK-201, TASK-202, TASK-301 |
| 7 | TASK-500 |

Итого: **19 задач**, 7 волн. При полной параллелизации — 7 sequential шагов.
