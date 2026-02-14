# Фаза 2: Инфраструктура базы данных


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (структура `adapter/postgres/`), §8 (Querier, TxManager, sqlc + Squirrel, миграции, индексы) |
| `data_model_v4.md` | Все секции — полный DDL таблиц, индексов, triggers, enums, ON DELETE сводка |
| `infra_spec_v4.md` | §2 (конфигурация БД), §7 (Docker), §8 (Makefile), §10 (порядок инициализации) |
| `repo/repo_layer_spec_v4.md` | §1 (структура пакета), §2 (миграции), §6 (принципы реализации), §7 (транзакции), §15 (timeouts) |
| `repo/repo_layer_tasks_v4.md` | TASK-000 — TASK-003 (инфраструктурные задачи), зафиксированные решения |
| `services/study_service_spec_v4_v1.1.md` | §4 — дополнительные поля: `learning_step` в cards, `prev_state` JSONB в review_logs, таблица `study_sessions` |

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | СУБД | PostgreSQL 17 |
| 2 | Драйвер | `pgx/v5` + `pgxpool` для connection pooling |
| 3 | Миграции | `goose` с sequential numbering (`00001_`, `00002_`, ...) |
| 4 | SQL-инструменты | `sqlc` для статических запросов + `Squirrel` для динамических (фильтрация, поиск) |
| 5 | sqlc организация | **sqlc per repo** — каждый пакет-репозиторий содержит свой `query/`, `sqlc/`, `sqlc.yaml` |
| 6 | TxManager isolation | **Read Committed** (PostgreSQL default), не конфигурируется |
| 7 | Nested transactions | Не поддерживаются — документировать в комментарии |
| 8 | Docker (dev) | `docker-compose.yml` с тремя сервисами: `postgres`, `migrate`, `backend` |
| 9 | Test containers | Один общий PostgreSQL-контейнер на весь `adapter/postgres/` через `sync.Once` |
| 10 | Goose формат | Sequential: `00001_enums.sql`, `00002_ref_catalog.sql`, ... |
| 11 | Down-миграции | Обязательны для всех миграций |

---

## Задачи

### TASK-2.1: Docker и Docker Compose

**Зависит от:** ничего

**Контекст:**
- `infra_spec_v4.md` — §7 (Docker: Dockerfile, Docker Compose, рекомендации)

**Что сделать:**

Создать Docker-инфраструктуру для разработки и production.

**Dockerfile (multi-stage build):**
- **Build stage:** `golang:1.23` — копирует go.mod/go.sum, скачивает зависимости, собирает бинарник
- **Runtime stage:** `alpine` — только бинарник + `ca-certificates`
- Запуск от непривилегированного пользователя (`USER nonroot`)
- EXPOSE для порта из конфига (default 8080)

**docker-compose.yml (development):**

Три сервиса:

| Сервис | Образ | Назначение |
|--------|-------|-----------|
| `postgres` | `postgres:17-alpine` | БД с volume для данных, healthcheck через `pg_isready` |
| `migrate` | На основе `golang:1.23-alpine` или отдельный образ с goose | Одноразовый контейнер: запускает `goose up`, depends_on postgres (service_healthy) |
| `backend` | Текущий Dockerfile | Приложение, depends_on migrate (service_completed_successfully) |

**Требования:**
- `.env` файл для локальных переменных — credentials не хардкодятся в docker-compose
- Volume для данных PostgreSQL (`pgdata`)
- Healthcheck для postgres: `pg_isready -U ${POSTGRES_USER}`
- Сервис `migrate` завершается после применения миграций (`restart: "no"`)

**Acceptance criteria:**
- [ ] Dockerfile создан: multi-stage, alpine runtime, nonroot user
- [ ] `docker-compose.yml` создан с тремя сервисами
- [ ] `docker compose up` поднимает PostgreSQL, применяет миграции, запускает backend
- [ ] `docker compose down -v` корректно останавливает всё и очищает volumes
- [ ] Credentials берутся из `.env`, не хардкодятся
- [ ] Healthcheck для postgres работает
- [ ] Сервис migrate завершается после `goose up`

---

### TASK-2.2: Миграции (goose)

**Зависит от:** ничего (может выполняться параллельно с TASK-2.1)

**Контекст:**
- `data_model_v4.md` — полный DDL всех таблиц (секции 2–8), triggers (секция 9), ON DELETE сводка (секция 11)
- `repo_layer_spec_v4.md` — §2 (формат, порядок, правила миграций)
- `repo_layer_tasks_v4.md` — TASK-000 (детали)
- `services/study_service_spec_v4_v1.1.md` — §4 (дополнительные поля: `learning_step`, `prev_state`, `study_sessions`)

**Что сделать:**

Создать 8 goose-миграций в директории `backend_v4/migrations/`. Каждая содержит `-- +goose Up` и `-- +goose Down`.

| # | Файл | Содержимое |
|---|------|-----------|
| 1 | `00001_enums.sql` | Все ENUM types: `learning_status`, `review_grade`, `part_of_speech`, `entity_type` (включая `TOPIC`), `audit_action` |
| 2 | `00002_ref_catalog.sql` | `ref_entries`, `ref_senses`, `ref_translations`, `ref_examples`, `ref_pronunciations`, `ref_images` + все индексы |
| 3 | `00003_users.sql` | `users`, `user_settings`, `refresh_tokens` + все индексы |
| 4 | `00004_user_dictionary.sql` | `entries`, `senses`, `translations`, `examples`, `entry_pronunciations`, `entry_images`, `user_images` + все индексы |
| 5 | `00005_cards.sql` | `cards` (с `learning_step`), `review_logs` (с `prev_state JSONB`), `study_sessions` + все индексы |
| 6 | `00006_organization.sql` | `topics`, `entry_topics`, `inbox_items`, `audit_log` + все индексы |
| 7 | `00007_triggers.sql` | 3 функции + 3 триггера для защиты данных при удалении ref-записей (`fn_preserve_sense_on_ref_delete`, `fn_preserve_translation_on_ref_delete`, `fn_preserve_example_on_ref_delete`) |
| 8 | `00008_extensions.sql` | `pg_trgm` extension + GIN-индексы для fuzzy search на `ref_entries.text_normalized` и `entries.text_normalized` |

**Дополнения к DDL из `data_model_v4.md` (из study_service_spec v1.1):**

Таблица `cards` — добавить колонку:
```sql
learning_step  INT NOT NULL DEFAULT 0   -- Текущий шаг в learning phase (0-based index в learning_steps)
```

Таблица `review_logs` — добавить колонку:
```sql
prev_state JSONB   -- Snapshot состояния карточки ДО review (для undo). NULL для первого review
```

Новая таблица `study_sessions`:
```sql
CREATE TABLE study_sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    started_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at   TIMESTAMPTZ,
    cards_studied INT NOT NULL DEFAULT 0,
    abandoned_at  TIMESTAMPTZ
);

CREATE INDEX ix_study_sessions_user ON study_sessions(user_id, started_at DESC);
```

Enum `entity_type` — значение `TOPIC` должно быть включено в первоначальное создание (миграция 00001).

**Правила:**
- `IF NOT EXISTS` / `IF EXISTS` где применимо
- Down-миграции: `DROP` в обратном порядке зависимостей
- Миграции не содержат DML (INSERT/UPDATE/DELETE) — только DDL
- Миграция 00007 (triggers) зависит от таблиц из миграций 00002 и 00004

**Acceptance criteria:**
- [ ] 8 миграций созданы в `backend_v4/migrations/`
- [ ] Каждая миграция содержит `-- +goose Up` и `-- +goose Down`
- [ ] `goose up` + `goose down` + `goose up` выполняется без ошибок
- [ ] Все таблицы, индексы, constraints, triggers из `data_model_v4.md` присутствуют
- [ ] `entity_type` включает значение `TOPIC`
- [ ] `cards.learning_step` INT NOT NULL DEFAULT 0
- [ ] `review_logs.prev_state` JSONB (nullable)
- [ ] Таблица `study_sessions` создана с индексом
- [ ] Unique constraints: `ux_ref_entries_text_norm`, `ux_entries_user_text` (partial — WHERE deleted_at IS NULL), `ux_users_email`, `ux_users_oauth`, `ux_topics_user_name`, `ux_cards_entry`
- [ ] Все FK ON DELETE поведение соответствует таблице из `data_model_v4.md` секция 11
- [ ] Trigger `trg_preserve_sense_on_ref_delete` корректно копирует данные при DELETE
- [ ] GIN-индекс для fuzzy search создан

**Corner cases:**
- Down-миграция для triggers (00007): `DROP TRIGGER` + `DROP FUNCTION` — порядок не критичен, но функция не может быть удалена пока trigger ссылается на неё
- ENUM down-миграция (00001): `DROP TYPE` в обратном порядке — сначала таблицы должны быть удалены. При `goose down` миграция 00001 выполняется последней, к этому моменту все зависимые таблицы уже удалены
- `ALTER TYPE ... ADD VALUE` нельзя откатить — если в будущем добавляется значение в enum, документировать это в комментарии

---

### TASK-2.3: Подключение к БД и расширение app.Run

**Зависит от:** TASK-2.2 (миграции должны быть готовы для проверки подключения)

**Контекст:**
- `code_conventions_v4.md` — §8.1 (pgxpool)
- `infra_spec_v4.md` — §10 (порядок инициализации: конфиг → логгер → БД → ...)
- `repo_layer_tasks_v4.md` — TASK-001 (pool.go)

**Что сделать:**

**`internal/adapter/postgres/pool.go`:**
- Функция `NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error)`
- Конфигурация: `MaxConns`, `MinConns`, `MaxConnLifetime`, `MaxConnIdleTime` из config
- После создания pool — выполнить `pool.Ping(ctx)` для проверки соединения
- При ошибке подключения — возвращать ошибку (fail-fast, обработка в app.Run)

**Расширение `internal/app/app.go`:**

Текущая функция `Run` выполняет: конфиг → логгер → лог старта → return nil. Расширить:

1. Загрузить конфиг
2. Инициализировать логгер
3. Подключиться к БД (`postgres.NewPool`)
4. `defer pool.Close()`
5. Залогировать успешное подключение
6. Ожидать `ctx.Done()` (signal от main.go)
7. Залогировать начало graceful shutdown
8. Закрыть pool (через defer)

```go
// Псевдо-структура расширения:
func Run(ctx context.Context) error {
    cfg := ...
    logger := ...
    pool, err := postgres.NewPool(ctx, cfg.Database)
    if err != nil {
        return fmt.Errorf("connect to database: %w", err)
    }
    defer pool.Close()
    logger.Info("database connected", slog.Int("max_conns", int(cfg.Database.MaxConns)))

    // Ожидание сигнала завершения
    <-ctx.Done()
    logger.Info("shutting down")
    return nil
}
```

**Новые зависимости в go.mod:**
- `github.com/jackc/pgx/v5`

**Acceptance criteria:**
- [ ] `NewPool` создаёт `*pgxpool.Pool` из `config.DatabaseConfig`
- [ ] `NewPool` выполняет Ping при создании — fail-fast при недоступной БД
- [ ] Параметры пула (MaxConns, MinConns, lifetimes) конфигурируются
- [ ] `app.Run` подключается к БД и ожидает `ctx.Done()`
- [ ] При `SIGINT`/`SIGTERM` — graceful shutdown (pool закрывается)
- [ ] Лог при подключении + лог при shutdown
- [ ] `pgx/v5` добавлен в `go.mod`
- [ ] `go build ./cmd/server/` компилируется

**Corner cases:**
- `pool.Close()` вызывается через defer до `<-ctx.Done()` — при ошибке до ожидания pool всё равно закроется
- Если `DATABASE_DSN` не задан — config.Validate() уже ловит это в фазе 1

---

### TASK-2.4: Querier, TxManager и маппинг ошибок

**Зависит от:** TASK-2.3 (нужен pool)

**Контекст:**
- `code_conventions_v4.md` — §8.1 (Querier и транзакции)
- `repo_layer_spec_v4.md` — §6 (принципы: получение querier), §7 (транзакции: TxManager)
- `repo_layer_tasks_v4.md` — TASK-001 (детали реализации)

**Что сделать:**

Создать в `internal/adapter/postgres/` общие компоненты для всех репозиториев.

**`querier.go`:**
- Тип `Querier` — интерфейс с методами, которые реализуются и `*pgxpool.Pool`, и `pgx.Tx`:
  ```go
  type Querier interface {
      Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
      Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
      QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
  }
  ```
- Unexported context key для хранения tx
- `withTx(ctx, tx)` — помещает tx в context
- `QuerierFromCtx(ctx, pool)` — возвращает tx из context если есть, иначе pool

**`txmanager.go`:**
- Тип `TxManager` с полем `pool *pgxpool.Pool`
- Конструктор `NewTxManager(pool *pgxpool.Pool) *TxManager`
- Метод `RunInTx(ctx context.Context, fn func(ctx context.Context) error) error`:
  - Начинает tx через `pool.Begin(ctx)`
  - Помещает tx в context через `withTx(ctx, tx)`
  - Вызывает `fn(txCtx)`
  - При panic внутри fn: `tx.Rollback(ctx)` + `re-panic`
  - При ошибке: `tx.Rollback(ctx)` + return error
  - При успехе: `tx.Commit(ctx)`
- Isolation level: Read Committed (PostgreSQL default) — не конфигурируется
- Nested RunInTx не поддерживается — документировать в комментарии

**`errors.go`:**
- Функция маппинга pgx/pgconn ошибок в domain-ошибки:

| pgx/pgconn ошибка | domain ошибка | Как определять |
|---|---|---|
| `pgx.ErrNoRows` | `domain.ErrNotFound` | `errors.Is(err, pgx.ErrNoRows)` |
| PgError code `23505` (unique_violation) | `domain.ErrAlreadyExists` | `pgconn.PgError.Code` |
| PgError code `23503` (foreign_key_violation) | `domain.ErrNotFound` | `pgconn.PgError.Code` |
| PgError code `23514` (check_violation) | `domain.ErrValidation` | `pgconn.PgError.Code` |
| `context.DeadlineExceeded` | **не маппится** | Прокидывается как есть |
| `context.Canceled` | **не маппится** | Прокидывается как есть |
| Всё остальное | Оборачивается с контекстом | `fmt.Errorf("entity op: %w", err)` |

- Ошибки оборачиваются с контекстом (entity name, ID) для трассировки
- Рекомендуемая сигнатура: `func mapError(err error, entity string, id uuid.UUID) error`

**Acceptance criteria:**
- [ ] `Querier` interface определён, реализуется pool и tx
- [ ] `QuerierFromCtx` возвращает tx из контекста если есть, иначе pool
- [ ] Context key — unexported тип
- [ ] `TxManager.RunInTx` коммитит при успехе
- [ ] `TxManager.RunInTx` откатывает при ошибке из fn
- [ ] `TxManager.RunInTx` откатывает при panic + re-panic
- [ ] Error mapping: `pgx.ErrNoRows` → `domain.ErrNotFound`
- [ ] Error mapping: unique_violation (23505) → `domain.ErrAlreadyExists`
- [ ] Error mapping: foreign_key_violation (23503) → `domain.ErrNotFound`
- [ ] Error mapping: check_violation (23514) → `domain.ErrValidation`
- [ ] `context.DeadlineExceeded` не маппится в domain-ошибку
- [ ] Unit-тесты: TxManager (success commit, error rollback, panic rollback)
- [ ] Unit-тесты: error mapping (все 4 кейса + context errors)

**Corner cases:**
- `RunInTx` не должен глотать ошибку от `tx.Rollback()` если fn вернул ошибку — но ошибка fn приоритетнее
- Если `pool.Begin()` возвращает ошибку (например, context cancelled) — возвращаем эту ошибку без маппинга
- Panic recovery: defer должен быть ДО вызова fn, чтобы перехватить panic из любого места внутри транзакции

---

### TASK-2.5: Конфигурация sqlc (шаблон)

**Зависит от:** TASK-2.2 (sqlc читает schema из миграций)

**Контекст:**
- `code_conventions_v4.md` — §8.2 (sqlc configuration, sqlc.yaml)
- `repo_layer_spec_v4.md` — §4 (sqlc vs Squirrel)
- `repo_layer_tasks_v4.md` — TASK-002, зафиксированное решение #3 (sqlc per repo)

**Что сделать:**

Создать **шаблонный** `sqlc.yaml` и демонстрационную структуру одного репозитория.

**Организация: sqlc per repo.** Каждый пакет-репозиторий содержит свой `query/`, `sqlc/`, `sqlc.yaml`:

```
internal/adapter/postgres/entry/
├── query/
│   └── entries.sql          # sqlc queries
├── sqlc/                    # generated
│   ├── db.go
│   ├── models.go
│   └── entries.sql.go
├── sqlc.yaml                # config: schema → ../../migrations/, queries → query/
├── repo.go                  # Тонкий слой: sqlc models → domain
├── filter.go                # Squirrel-запросы (если нужны)
└── repo_test.go
```

**Шаблон sqlc.yaml:**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query/"
    schema: "../../../../migrations/"
    gen:
      go:
        package: "sqlc"
        out: "sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: false
        emit_empty_slices: true
        overrides:
          - db_type: "uuid"
            go_type:
              import: "github.com/google/uuid"
              type: "UUID"
          - db_type: "timestamptz"
            go_type: "time.Time"
          - db_type: "timestamptz"
            nullable: true
            go_type:
              type: "*time.Time"
```

**Ключевые настройки:**
- `emit_empty_slices: true` — пустые результаты `[]`, не `nil`
- `emit_json_tags: false` — domain models не содержат json тегов
- **ENUM overrides не добавляются** — sqlc генерирует свои string types, маппинг в domain enums выполняется вручную в repo-функциях маппинга
- `schema` указывает на директорию миграций относительно расположения конкретного `sqlc.yaml`

**Демонстрационный файл `query/entries.sql`:**

Создать минимальный SQL-файл с 1-2 запросами для проверки работоспособности sqlc generate. Полные запросы будут добавлены в Фазе 3. Пример:

```sql
-- name: GetEntryByID :one
SELECT id, user_id, ref_entry_id, text, text_normalized, notes,
       created_at, updated_at, deleted_at
FROM entries
WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;
```

**Обновление `make generate`:**
- Найти все `sqlc.yaml` в поддиректориях `internal/adapter/postgres/` и запустить `sqlc generate` для каждого

**Обновление `tools.go`:**
- Добавить `github.com/sqlc-dev/sqlc/cmd/sqlc`
- Добавить `github.com/pressly/goose/v3/cmd/goose`

**Acceptance criteria:**
- [ ] Шаблон `sqlc.yaml` создан с корректными overrides
- [ ] Демонстрационная структура одного repo (entry/) создана
- [ ] `sqlc generate` успешно генерирует код из демонстрационного запроса
- [ ] Сгенерированные модели: UUID → `uuid.UUID`, TIMESTAMPTZ → `time.Time`, nullable → `*time.Time`
- [ ] `emit_empty_slices: true` — пустые результаты `[]`, не `nil`
- [ ] `make generate` рекурсивно находит и обрабатывает все `sqlc.yaml`
- [ ] `tools.go` обновлён: sqlc и goose добавлены
- [ ] `go build ./...` компилируется после генерации

**Corner cases:**
- Путь `schema` в sqlc.yaml зависит от глубины вложенности пакета — при копировании шаблона для другого repo скорректировать относительный путь
- sqlc генерирует свои типы для PostgreSQL enums (например, `LearningStatus string`) — **не** путать их с `domain.LearningStatus`. Маппинг выполняется в repo layer

---

### TASK-2.6: Test Helpers (testcontainers)

**Зависит от:** TASK-2.2 (миграции), TASK-2.3 (pool)

**Контекст:**
- `repo_layer_spec_v4.md` — §20 (тестирование: подход, helpers, правила)
- `repo_layer_tasks_v4.md` — TASK-003 (детали реализации test helpers)

**Что сделать:**

Создать пакет `internal/adapter/postgres/testhelper/` с инфраструктурой для integration-тестов.

**`db.go` — SetupTestDB:**
- Функция `SetupTestDB(t *testing.T) *pgxpool.Pool`
- Один PostgreSQL контейнер (`postgres:17-alpine`) на весь `adapter/postgres/`
- Контейнер создаётся через `sync.Once` и переиспользуется всеми test-пакетами
- При первом запуске: поднять контейнер → применить все goose-миграции
- `t.Cleanup` для закрытия pool (контейнер живёт до конца всех тестов)
- Контейнер конфигурируется через testcontainers-go

**`seed.go` — функции создания тестовых данных:**

| Функция | Что создаёт | Возвращает |
|---------|------------|-----------|
| `SeedUser(t, pool)` | user + user_settings с дефолтами | `domain.User` |
| `SeedRefEntry(t, pool, text)` | ref_entry + 2 ref_senses (каждый с 2 ref_translations + 2 ref_examples) + 2 ref_pronunciations | `domain.RefEntry` с заполненным деревом |
| `SeedEntry(t, pool, userID, refEntryID)` | entry + senses (linked to ref) + translations + examples + pronunciations (M2M), **без card** | `domain.Entry` |
| `SeedEntryWithCard(t, pool, userID, refEntryID)` | то же что SeedEntry + card со статусом NEW | `domain.Entry` (с заполненным Card) |
| `SeedEntryCustom(t, pool, userID)` | entry + custom senses (без ref links, с заполненными definition/pos) | `domain.Entry` |

**Требования к seed-функциям:**
- Все используют `t.Helper()` и `require` для fail-fast
- Генерируют уникальные значения (UUID-based suffix в email/text) — тесты из разных пакетов не конфликтуют
- Возвращают заполненные domain-модели — для использования в assertions
- Вставка данных через прямые SQL (`pool.Exec`), не через repo-методы (repos ещё не реализованы в этой фазе)

**Новые зависимости в go.mod:**
- `github.com/testcontainers/testcontainers-go`
- `github.com/pressly/goose/v3`

**Acceptance criteria:**
- [ ] `SetupTestDB` поднимает контейнер PostgreSQL 17 и применяет миграции
- [ ] Контейнер переиспользуется через `sync.Once` (не поднимается на каждый тест/пакет)
- [ ] Все seed-функции создают корректные данные с уникальными значениями
- [ ] `SeedEntry` и `SeedEntryWithCard` — две отдельные функции
- [ ] `SeedRefEntry` создаёт полное дерево: 2 senses × (2 translations + 2 examples) + 2 pronunciations
- [ ] Seed-функции возвращают заполненные domain-модели
- [ ] Тесты из разных пакетов могут работать параллельно с общим контейнером
- [ ] Минимальный smoke-тест: `SetupTestDB` + `SeedUser` + проверка что user существует в БД
- [ ] `testcontainers-go` и `goose/v3` добавлены в `go.mod`

**Corner cases:**
- Один контейнер = общая БД. Тесты не должны конфликтовать: seed-функции генерируют уникальные данные
- `SeedRefEntry`: конкретный набор данных важен — 2 senses с children достаточно для тестов COALESCE, partial customization, batch loading в Фазе 3
- Миграции применяются через goose programmatic API, не через CLI
- Goose миграции в тестах: путь к файлам миграций определяется относительно пакета testhelper

---

### TASK-2.7: Makefile

**Зависит от:** TASK-2.1 (Docker), TASK-2.2 (миграции)

**Контекст:**
- `infra_spec_v4.md` — §8 (Makefile: категории целей, рекомендации)
- `repo_layer_tasks_v4.md` — TASK-001 (список targets)

**Что сделать:**

Создать `backend_v4/Makefile` со всеми необходимыми целями.

**Цели:**

| Категория | Target | Команда | Описание |
|-----------|--------|---------|----------|
| Сборка | `build` | `go build -o bin/server ./cmd/server/` | Собрать бинарник |
| Сборка | `run` | `go run ./cmd/server/` | Запустить сервер |
| Тесты | `test` | `go test ./... -race -count=1` | Все unit-тесты |
| Тесты | `test-cover` | `go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out` | Coverage report |
| Тесты | `test-integration` | `go test ./... -race -count=1 -tags=integration` | Integration-тесты (требуют Docker) |
| Генерация | `generate` | Рекурсивный sqlc generate + `go generate ./...` | Генерация кода |
| Миграции | `migrate-up` | `goose -dir migrations/ postgres "$$DATABASE_DSN" up` | Применить миграции |
| Миграции | `migrate-down` | `goose -dir migrations/ postgres "$$DATABASE_DSN" down` | Откатить последнюю миграцию |
| Миграции | `migrate-status` | `goose -dir migrations/ postgres "$$DATABASE_DSN" status` | Статус миграций |
| Миграции | `migrate-create` | `goose -dir migrations/ create $(name) sql` | Создать новую миграцию |
| Docker | `docker-up` | `docker compose up -d` | Поднять dev-окружение |
| Docker | `docker-down` | `docker compose down` | Остановить dev-окружение |
| Docker | `docker-logs` | `docker compose logs -f` | Логи dev-окружения |
| Качество | `lint` | `golangci-lint run` | Линтер |
| Помощь | `help` | Автогенерация из комментариев | Список целей (default target) |

**Требования:**
- `help` — default target (при `make` без аргументов)
- Переменные (`DATABASE_DSN`, порт) через переменные окружения с дефолтами в Makefile
- `test-integration` использует build tag `integration`
- `generate` находит все `sqlc.yaml` в поддиректориях `adapter/postgres/` и запускает sqlc для каждого

**Acceptance criteria:**
- [ ] Makefile создан в `backend_v4/`
- [ ] `make` (без аргументов) показывает help
- [ ] `make build` собирает бинарник в `bin/server`
- [ ] `make test` запускает unit-тесты с `-race`
- [ ] `make lint` запускает golangci-lint
- [ ] `make generate` рекурсивно обрабатывает sqlc.yaml
- [ ] `make migrate-up` / `make migrate-down` работают с `DATABASE_DSN`
- [ ] `make docker-up` / `make docker-down` управляют docker compose
- [ ] Все targets имеют описания (для `make help`)

---

## Сводка зависимостей задач

```
TASK-2.1 (Docker) ─────────────────────────────────────────┐
                                                            │
TASK-2.2 (Миграции) ──┬──→ TASK-2.3 (Pool + app.Run) ─────┤
                       │         │                          │
                       │         └──→ TASK-2.4 (Querier,   │
                       │              TxManager, Errors)    │
                       │                                    │
                       ├──→ TASK-2.5 (sqlc шаблон)         │
                       │                                    │
                       └──→ TASK-2.6 (Test Helpers) ←──────┤
                                                            │
TASK-2.7 (Makefile) ←──────────────────────────────────────┘
```

## Параллелизация

| Волна | Задачи |
|-------|--------|
| 1 | TASK-2.1, TASK-2.2 (параллельно — нет взаимных зависимостей) |
| 2 | TASK-2.3, TASK-2.5 (параллельно — обе зависят только от TASK-2.2) |
| 3 | TASK-2.4 (зависит от TASK-2.3) |
| 4 | TASK-2.6 (зависит от TASK-2.2, TASK-2.3) |
| 5 | TASK-2.7 (зависит от TASK-2.1, TASK-2.2; может делаться раньше, но полная проверка возможна после всех задач) |

> TASK-2.7 (Makefile) технически может создаваться параллельно с волной 1, но полноценно тестировать все targets можно только после завершения зависимых задач.

---

## Чеклист завершения фазы

- [ ] `docker compose up` поднимает PostgreSQL, применяет миграции, запускает backend
- [ ] `goose up` + `goose down` + `goose up` — без ошибок
- [ ] Все 22 таблицы + `study_sessions` созданы с корректными индексами, constraints, triggers
- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] `app.Run` подключается к PostgreSQL, ожидает signal, graceful shutdown
- [ ] `TxManager.RunInTx` корректно обрабатывает commit, rollback, panic
- [ ] Error mapping покрывает все случаи (ErrNoRows, unique, fk, check violations)
- [ ] `QuerierFromCtx` корректно определяет tx в контексте
- [ ] sqlc генерирует код из шаблонного запроса
- [ ] `SetupTestDB` поднимает контейнер и применяет миграции
- [ ] Seed-функции создают валидные тестовые данные
- [ ] Все Makefile targets работают
- [ ] Новые зависимости в `go.mod`: `pgx/v5`, `testcontainers-go`, `goose/v3`
- [ ] `tools.go` обновлён: sqlc, goose
- [ ] Все acceptance criteria всех задач выполнены
