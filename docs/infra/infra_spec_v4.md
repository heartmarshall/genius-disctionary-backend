# Infrastructure Specification — Backend v4

> Паттерны и рекомендации по инфраструктурным аспектам проекта.
> Документ НЕ содержит готовой реализации — только принципы и решения,
> которых следует придерживаться при разработке.

---

### 1. Принципы

- **Одна точка входа.** `cmd/server/main.go` — только вызов bootstrap-функции из `internal/app`, никакой логики.
- **`internal/` для всего приложения.** Пакеты не экспортируются наружу.
- **`pkg/` — только утилиты без бизнес-зависимостей.** Контекстные хелперы, которые могут понадобиться в любом слое.
- **Плоская структура внутри пакетов.** Без глубокой вложенности — один уровень подпакетов максимум.

---

## 2. Конфигурация

### Подход

- **Библиотека:** `cleanenv` — загрузка из YAML + ENV, тег-валидация.
- **Приоритет:** ENV-переменные перекрывают YAML. Дефолты задаются тегами.
- **Единая структура.** Одна корневая config-структура, разбитая на секции через вложенные struct'ы (Server, Database, Auth, GraphQL, Log).
- **Валидация при старте.** Если конфиг невалиден — `log.Fatal` до запуска чего-либо. Обязательно проверять: наличие JWT-секрета достаточной длины, корректность database DSN, наличие credentials хотя бы одного OAuth-провайдера.

### Секции конфигурации

| Секция   | Что содержит                                                    |
|----------|-----------------------------------------------------------------|
| Server   | Host, port, таймауты (read, write, idle, shutdown)              |
| Database | DSN, параметры пула (min/max conns, lifetime)                   |
| Auth     | JWT (secret, access TTL, refresh TTL), Google OAuth, Apple Sign-In |
| GraphQL  | Playground on/off, introspection, complexity limit               |
| Log      | Level, format (json/text)                                        |

### Рекомендации

- Таймауты задавать через `time.Duration` (cleanenv поддерживает парсинг `"10s"`, `"5m"`).
- Для production отключать GraphQL playground и introspection.
- Не хранить секреты в YAML — только через ENV.

---

## 3. Логирование

### Подход

- **Библиотека:** `log/slog` (stdlib, Go 1.21+). Никаких внешних зависимостей.
- **Формат:** JSON в production, text в development — переключается через конфиг.
- **Уровни:** debug, info, warn, error. Уровень задаётся в конфиге.

### Паттерны

- **Context-aware логгер.** Логгер кладётся в `context.Context` при входе запроса (в middleware). Все слои достают его из контекста через хелпер из `pkg/context/`.
- **Structured fields.** Не строковые сообщения, а именованные поля: `request_id`, `user_id`, `method`, `path`, `status`, `duration`, `error`.
- **Никаких `log.Println`.** Только slog через контекст.
- **Ошибки логируются один раз** — на том уровне, где принимается решение (обычно handler/middleware), а не на каждом слое при пробросе.

---

## 4. HTTP-сервер и Middleware

### Сервер

- Стандартный `net/http.Server` — без фреймворков.
- Роутинг через `http.ServeMux` (Go 1.22+ pattern matching).
- GraphQL-хендлер монтируется на `/query`, health-эндпоинты — отдельно.

### Middleware-стек (порядок важен)

1. **Recovery** — перехват panic, логирование stack trace, ответ 500.
2. **Request ID** — генерация UUID, прокидывание в context и response header `X-Request-Id`.
3. **Logger** — structured-лог запроса: метод, path, статус, длительность. Кладёт логгер с `request_id` в context.
4. **CORS** — настраиваемые origins, methods, headers.
5. **Auth** — извлечение и валидация JWT из `Authorization: Bearer`. Если токен валиден — кладёт `AuthUser` в context. Если нет — пропускает (anonymous). Protected-ресурсы проверяют наличие пользователя сами.
6. **GraphQL Handler** — gqlgen.

### Рекомендации

- Middleware реализовывать как функции `func(next http.Handler) http.Handler`.
- Health-эндпоинты (`/health`, `/ready`, `/live`) регистрировать **вне** middleware-стека — они не должны требовать auth и не должны логировать каждый вызов.
- Таймаут на запросы задавать через `http.Server.ReadTimeout`/`WriteTimeout`, а не middleware.

---

## 5. Health Checks

### Три эндпоинта

| Эндпоинт  | Назначение              | Что проверяет               | Когда 503         |
|-----------|-------------------------|-----------------------------|-------------------|
| `/live`   | Liveness probe (k8s)    | Ничего — процесс жив       | Никогда           |
| `/ready`  | Readiness probe (k8s)   | Ping БД                    | БД недоступна     |
| `/health` | Full health check       | Ping БД + latency           | Любой компонент down |

### Паттерны

- Ответ в JSON: `status` (ok/degraded/down), `components` (map компонентов с их статусами), `timestamp`.
- `/health` и `/ready` — реальные проверки с коротким таймаутом (2-3 сек).
- `/live` — всегда 200, без проверок. Нужен чтобы k8s/docker не рестартовал контейнер из-за временной проблемы с БД.

---

## 6. Graceful Shutdown

### Паттерн

- Слушать `SIGINT` и `SIGTERM` через `signal.NotifyContext`.
- При получении сигнала:
  1. Вызвать `http.Server.Shutdown(ctx)` с таймаутом (10 сек) — дать in-flight запросам завершиться.
  2. Закрыть пул соединений БД.
  3. Выполнить `os.Exit(0)`.
- Если таймаут shutdown истёк — принудительное завершение.

### Порядок остановки

**Обратный порядку запуска:** HTTP-сервер → сервисы (если есть background workers) → БД.

---

## 7. Docker

### Dockerfile

- **Multi-stage build:** первый stage — `golang:1.23` (сборка), второй — `alpine` (runtime).
- В runtime-образе только бинарник + ca-certificates. Никаких исходников, go toolchain.
- Запускать от непривилегированного пользователя (`USER nonroot`).

### Docker Compose (dev)

- **Три сервиса:** `postgres`, `migrate`, `backend`.
- **Зависимости:** postgres → healthcheck → migrate → backend. Использовать `depends_on` с `condition: service_healthy` / `service_completed_successfully`.
- **Postgres:** образ `postgres:16-alpine`, volume для данных, healthcheck через `pg_isready`.
- **Migrate:** одноразовый контейнер, запускает `goose up` и завершается.
- **Backend:** зависит от migrate, свой healthcheck через `/ready`.

### Рекомендации

- `.env` файл для локальных переменных, в `.gitignore`.
- Не хардкодить credentials в docker-compose — ссылаться на `.env`.

---

## 8. Makefile

### Категории целей

| Категория    | Примеры целей                                      |
|-------------|---------------------------------------------------|
| Генерация   | `generate` (gqlgen)                                |
| Сборка      | `build`, `run`                                     |
| Тесты       | `test`, `test-coverage`, `test-integration`        |
| Миграции    | `migrate-up`, `migrate-down`, `migrate-status`, `migrate-create` |
| Docker      | `docker-up`, `docker-down`, `docker-logs`          |
| Качество    | `lint` (golangci-lint)                             |

### Рекомендации

- Цель `help` — по умолчанию, выводит список всех целей с описаниями.
- Переменные (DSN, порт) — через переменные окружения с дефолтами в Makefile.
- `test-integration` — запускает с build tag `integration`, требует Docker.

---

## 9. Зависимости (go.mod)

### Критерий выбора

- Предпочитать stdlib где возможно (`log/slog`, `net/http`, `crypto`).
- Внешние библиотеки — только когда stdlib значимо уступает.

### Внешние зависимости

| Библиотека               | Зачем                                  |
|--------------------------|----------------------------------------|
| `pgx/v5`                 | PostgreSQL-драйвер + connection pool   |
| `gqlgen`                 | Генерация GraphQL                      |
| `squirrel`               | SQL query builder                      |
| `scany/v2`               | Маппинг rows → structs                 |
| `google/uuid`            | UUID-генерация                         |
| `cleanenv`               | Загрузка конфигурации                  |
| `golang-jwt/jwt/v5`      | JWT-токены                             |
| `pressly/goose/v3`       | Миграции                               |
| `testcontainers-go`      | Integration-тесты с реальной БД        |

### Чего НЕ тянуть

- HTTP-фреймворки (chi, gin, fiber) — хватает stdlib.
- Внешние логгеры (zap, zerolog) — хватает slog.
- ORM (gorm, ent) — используем squirrel + scany.

---

## 10. Точка входа (bootstrap)

### Порядок инициализации

1. Загрузить и провалидировать конфиг.
2. Инициализировать логгер.
3. Подключиться к БД (pool).
4. Создать репозитории и TxManager.
5. Создать auth-сервисы (JWT, OAuth).
6. Создать бизнес-сервисы.
7. Собрать middleware-цепочку и HTTP-хендлер.
8. Зарегистрировать роуты (health + main handler).
9. Запустить HTTP-сервер.
10. Ожидать сигнал → graceful shutdown.

### Принципы

- **Fail fast.** Любая ошибка на этапах 1–6 — `log.Fatal`. Не запускать сервер с неполной инициализацией.
- **Dependency injection через конструкторы.** Никаких глобальных переменных или init(). Все зависимости передаются явно.
- **Bootstrap-функция в `internal/app/`**, а не в `main.go`. Main только вызывает её.
