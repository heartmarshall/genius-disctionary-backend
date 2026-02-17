# Фаза 10: Сборка сервера и интеграция

> **Дата:** 2026-02-15
> **Назначение:** HTTP-сервер, middleware-стек, health endpoints, graceful shutdown, bootstrap wiring всех слоёв, audit cleanup (BG3), E2E smoke tests.

---

## Документы-источники

| Документ | Секции |
|----------|--------|
| `docs/infra/infra_spec_v4.md` | §4 (HTTP-сервер, middleware), §5 (health checks), §6 (graceful shutdown), §7 (Docker), §10 (bootstrap) |
| `docs/code_conventions_v4.md` | §1 (структура), §2 (ошибки), §4 (контекст), §5 (логирование), §9 (аутентификация) |
| `docs/services/service_layer_spec_v4.md` | §6 (карта сервисов, зависимости) |
| `docs/services/auth_service_spec_v4.md` | §4 (ValidateToken) — для auth middleware |
| `docs/services/business_scenarios_v4.md` | BG3 (audit cleanup) |
| `docs/data_model_v4.md` | §7 (audit_log) |

---

## Предусловия (от завершённых фаз)

**Фаза 1 (Skeleton + Domain):**
- Все доменные модели и enum'ы
- Sentinel errors в `domain/errors.go`
- `pkg/ctxutil/` — `WithUserID`, `UserIDFromCtx`, `WithRequestID`, `RequestIDFromCtx`
- `internal/config/` — `Config` со всеми секциями (Server, Database, Auth, Dictionary, GraphQL, Log, SRS)
- `internal/app/` — `NewLogger()`, `BuildVersion()`, базовый `Run(ctx)`
- `cmd/server/main.go` — вызывает `app.Run(ctx)`

**Фаза 2 (Database):**
- `postgres.NewPool(ctx, cfg)` — pgxpool с fail-fast
- `postgres.NewTxManager(pool)` — транзакции через context
- Миграции (8 файлов), `cmd/cleanup/main.go` (BG1)

**Фаза 3 (Repository Layer):**
- 14 репозиториев в `internal/adapter/postgres/`
- DataLoaders (9 загрузчиков) в `internal/transport/graphql/dataloader/`
- `dataloader.Middleware(repos)` — per-request middleware
- `dataloader.Repos` struct (Sense, Translation, Example, Pronunciation, Image, Card, Topic, ReviewLog)

**Фаза 4 (Auth + User):**
- `internal/service/auth/` — Login, Refresh, Logout, ValidateToken, CleanupExpiredTokens
- `internal/service/user/` — GetProfile, GetSettings, UpdateSettings
- OAuth verifier, JWT manager в `internal/adapter/`

**Фазы 5–8 (Service Layer):**
- `internal/service/dictionary/` — 13 операций, `NewService(logger, entries, senses, translations, examples, pronunciations, images, cards, audit, tx)`
- `internal/service/content/` — 14 операций, `NewService(logger, entries, senses, translations, examples, images, audit, tx)`
- `internal/service/study/` — 12 операций, `NewService(logger, cards, reviews, sessions, entries, senses, settings, audit, tx, srsConfig)`
- `internal/service/topic/` — 7 операций, `NewService(logger, topics, entries, audit, tx)`
- `internal/service/inbox/` — 5 операций, `NewService(logger, inbox)`
- `internal/service/refcatalog/` — `NewService(logger, refEntries, tx, dictProvider, transProvider)`

**Фаза 9 (Transport — GraphQL):**
- GraphQL-схема (8 `.graphql` файлов, 53 операции)
- `resolver.NewResolver(logger, dictionary, content, study, topic, inbox, user)` — Resolver struct
- `graphql.NewErrorPresenter(logger)` — error presenter
- Field resolvers с DataLoaders (9 resolvers)
- gqlgen handler setup

**Внешние провайдеры:**
- `provider/freedict.NewProvider(logger)` — FreeDictionary API
- `provider/translate.NewStub()` — Translation stub

---

## Зафиксированные решения

| # | Решение | Обоснование |
|---|---------|-------------|
| 1 | **Middleware в `internal/transport/middleware/`** — новый пакет | HTTP-middleware не привязаны к GraphQL. Чистое разделение. `graphql/middleware/` остаётся для GraphQL-specific (пока не используется) |
| 2 | **Middleware как `func(next http.Handler) http.Handler`** | Стандартный Go-паттерн из infra_spec §4 |
| 3 | **`Chain()` helper** для композиции middleware | Избегаем глубокой вложенности при вызове. `Chain(mw1, mw2, mw3)(handler)` |
| 4 | **Health endpoints вне middleware-стека** | infra_spec §4: "не должны требовать auth и не должны логировать каждый вызов" |
| 5 | **`http.ServeMux`** (Go 1.22+ pattern matching) | infra_spec §4. Без фреймворков |
| 6 | **GraphQL на `POST /query`** | infra_spec §4 |
| 7 | **Playground условно** — `cfg.GraphQL.PlaygroundEnabled` | infra_spec §2: "Для production отключать playground и introspection" |
| 8 | **Auth middleware: нет токена → pass through, невалидный → 401** | code_conventions §9: anonymous endpoints (searchCatalog, previewRefEntry) пропускаются. Protected ресурсы проверяют userID сами |
| 9 | **Graceful shutdown: HTTP → DB pool** | infra_spec §6: обратный порядку запуска |
| 10 | **CORSConfig** — отдельная секция конфигурации | Настраиваемые origins/methods/headers через ENV |
| 11 | **BG3 (audit cleanup) — расширение `cmd/cleanup/`** | Один бинарник для всех cleanup-задач. Флаг `--audit` |
| 12 | **E2E smoke tests** (5–10 тестов) | Проверка wiring: health, auth flow, один CRUD цикл, error codes |
| 13 | **`net/http.Server`** с таймаутами из `ServerConfig` | infra_spec §4: ReadTimeout, WriteTimeout, IdleTimeout |

---

## Файловая структура (результат фазы)

```
internal/transport/middleware/
├── chain.go                    # Chain() helper для композиции
├── recovery.go                 # Recovery middleware (panic → 500)
├── requestid.go                # Request ID middleware (UUID, X-Request-Id)
├── logger.go                   # Structured request logging
├── cors.go                     # CORS middleware
├── auth.go                     # Auth middleware (JWT → userID in ctx)
├── recovery_test.go
├── requestid_test.go
├── logger_test.go
├── cors_test.go
└── auth_test.go

internal/transport/rest/
├── health.go                   # /live, /ready, /health endpoints
└── health_test.go

internal/config/
└── config.go                   # EXTEND: добавить CORSConfig

internal/app/
└── app.go                      # EXTEND: полный bootstrap wiring

cmd/cleanup/
└── main.go                     # EXTEND: --audit флаг для BG3

tests/e2e/
├── helpers_test.go             # Test helper: testcontainers + полный bootstrap
└── e2e_test.go                 # Smoke tests
```

---

## Задачи

### TASK-10.1: CORSConfig + Infrastructure Middleware (Recovery, RequestID, Logger, CORS)

**Зависит от:** Фаза 1 (config, ctxutil)

**Контекст:**
- `infra_spec_v4.md` — §4 (middleware-стек, порядок)
- `code_conventions_v4.md` — §4 (контекст), §5 (логирование)
- `pkg/ctxutil/` — `WithRequestID`, `RequestIDFromCtx`

**Что сделать:**

Добавить `CORSConfig` в конфигурацию. Создать пакет `internal/transport/middleware/` с 4 middleware и `Chain()` helper. Написать unit-тесты.

---

#### 1. Добавить `CORSConfig` в конфигурацию

**Файл:** `internal/config/config.go`

Добавить секцию в `Config`:

```go
type Config struct {
    // ... существующие секции ...
    CORS CORSConfig `yaml:"cors"`
}
```

```go
type CORSConfig struct {
    AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"*"`
    AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,OPTIONS"`
    AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Authorization,Content-Type"`
    AllowCredentials bool  `yaml:"allow_credentials" env:"CORS_ALLOW_CREDENTIALS" env-default:"true"`
    MaxAge          int    `yaml:"max_age"          env:"CORS_MAX_AGE"          env-default:"86400"`
}
```

> **Примечание:** `AllowedOrigins`, `AllowedMethods`, `AllowedHeaders` — comma-separated строки. Парсинг в middleware через `strings.Split`.

---

#### 2. `chain.go` — Middleware chaining

```go
package middleware

import "net/http"

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware in order: first middleware is outermost (executes first).
// Chain(mw1, mw2, mw3)(handler) == mw1(mw2(mw3(handler)))
func Chain(mws ...Middleware) Middleware {
    return func(final http.Handler) http.Handler {
        for i := len(mws) - 1; i >= 0; i-- {
            final = mws[i](final)
        }
        return final
    }
}
```

---

#### 3. `recovery.go` — Recovery middleware

```go
package middleware

import (
    "log/slog"
    "net/http"
    "runtime/debug"
)

// Recovery перехватывает panic, логирует stack trace, возвращает 500.
func Recovery(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            defer func() {
                if err := recover(); err != nil {
                    stack := debug.Stack()
                    logger.ErrorContext(r.Context(), "panic recovered",
                        slog.Any("error", err),
                        slog.String("stack", string(stack)),
                        slog.String("method", r.Method),
                        slog.String("path", r.URL.Path),
                    )
                    http.Error(w, "internal server error", http.StatusInternalServerError)
                }
            }()
            next.ServeHTTP(w, r)
        })
    }
}
```

---

#### 4. `requestid.go` — Request ID middleware

```go
package middleware

import (
    "net/http"

    "github.com/google/uuid"
    "github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

const RequestIDHeader = "X-Request-Id"

// RequestID генерирует UUID request ID (или берёт из заголовка),
// кладёт в context и response header.
func RequestID() Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            id := r.Header.Get(RequestIDHeader)
            if id == "" {
                id = uuid.New().String()
            }

            ctx := ctxutil.WithRequestID(r.Context(), id)
            w.Header().Set(RequestIDHeader, id)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

---

#### 5. `logger.go` — Request logging middleware

```go
package middleware

import (
    "log/slog"
    "net/http"
    "time"

    "github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// Logger логирует каждый HTTP-запрос: метод, path, статус, длительность.
// Обогащает логгер request_id и user_id из context.
func Logger(logger *slog.Logger) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
            next.ServeHTTP(sw, r)

            duration := time.Since(start)
            requestID := ctxutil.RequestIDFromCtx(r.Context())
            userID, _ := ctxutil.UserIDFromCtx(r.Context())

            attrs := []slog.Attr{
                slog.String("method", r.Method),
                slog.String("path", r.URL.Path),
                slog.Int("status", sw.status),
                slog.Duration("duration", duration),
                slog.String("request_id", requestID),
            }
            if userID.String() != "00000000-0000-0000-0000-000000000000" {
                attrs = append(attrs, slog.String("user_id", userID.String()))
            }

            level := slog.LevelInfo
            if sw.status >= 500 {
                level = slog.LevelError
            }

            logger.LogAttrs(r.Context(), level, "http.request", attrs...)
        })
    }
}

// statusWriter перехватывает WriteHeader для записи status code.
type statusWriter struct {
    http.ResponseWriter
    status      int
    wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
    if !w.wroteHeader {
        w.status = code
        w.wroteHeader = true
    }
    w.ResponseWriter.WriteHeader(code)
}
```

---

#### 6. `cors.go` — CORS middleware

```go
package middleware

import (
    "net/http"
    "strconv"
    "strings"

    "github.com/heartmarshall/myenglish-backend/internal/config"
)

// CORS обрабатывает preflight-запросы и устанавливает CORS-заголовки.
func CORS(cfg config.CORSConfig) Middleware {
    origins := strings.Split(cfg.AllowedOrigins, ",")
    methods := cfg.AllowedMethods
    headers := cfg.AllowedHeaders

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if origin != "" && isAllowedOrigin(origin, origins) {
                w.Header().Set("Access-Control-Allow-Origin", origin)
                if cfg.AllowCredentials {
                    w.Header().Set("Access-Control-Allow-Credentials", "true")
                }
            }

            // Preflight
            if r.Method == http.MethodOptions {
                w.Header().Set("Access-Control-Allow-Methods", methods)
                w.Header().Set("Access-Control-Allow-Headers", headers)
                w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func isAllowedOrigin(origin string, allowed []string) bool {
    for _, a := range allowed {
        a = strings.TrimSpace(a)
        if a == "*" || a == origin {
            return true
        }
    }
    return false
}
```

---

#### Unit-тесты

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestRecovery_NoPanic` | Handler без panic | Статус 200, handler вызван |
| 2 | `TestRecovery_Panic` | Handler паникует | Статус 500, response body "internal server error" |
| 3 | `TestRecovery_PanicStringError` | Panic с string | Статус 500, ошибка залогирована |
| 4 | `TestRequestID_Generated` | Запрос без X-Request-Id | UUID сгенерирован, в response header и context |
| 5 | `TestRequestID_Preserved` | Запрос с X-Request-Id | Входящий ID сохранён, в response header и context |
| 6 | `TestLogger_Success` | Статус 200 | Залогировано: method, path, status=200, duration |
| 7 | `TestLogger_ServerError` | Статус 500 | Уровень ERROR |
| 8 | `TestLogger_IncludesRequestID` | Запрос с request_id в ctx | request_id в лог-записи |
| 9 | `TestCORS_Preflight` | OPTIONS с Origin | Access-Control-Allow-* headers, статус 204 |
| 10 | `TestCORS_AllowedOrigin` | GET с разрешённым origin | Access-Control-Allow-Origin установлен |
| 11 | `TestCORS_DisallowedOrigin` | GET с неразрешённым origin | Заголовок не установлен |
| 12 | `TestCORS_Wildcard` | AllowedOrigins="*" | Любой origin разрешён |
| 13 | `TestChain_Order` | Chain(mw1, mw2)(handler) | mw1 выполняется первым (outermost) |
| 14 | `TestChain_Empty` | Chain()(handler) | Handler вызван напрямую |

**Всего: 14 тест-кейсов**

---

**Acceptance criteria:**
- [ ] `CORSConfig` добавлен в `internal/config/config.go`
- [ ] `internal/transport/middleware/` создан: chain.go, recovery.go, requestid.go, logger.go, cors.go
- [ ] Все middleware имеют сигнатуру `Middleware` (= `func(http.Handler) http.Handler`)
- [ ] `Chain()` применяет middleware в правильном порядке (первый = outermost)
- [ ] Recovery: panic → 500 + stack trace в логе
- [ ] RequestID: генерация UUID, приём из заголовка, context + response header
- [ ] Logger: structured log с method, path, status, duration, request_id, user_id
- [ ] Logger: status >= 500 → level ERROR
- [ ] CORS: preflight → 204, allowed origin → header, disallowed → нет header
- [ ] 14 unit-тестов проходят
- [ ] `go build ./...` компилируется

---

### TASK-10.2: Auth Middleware

**Зависит от:** TASK-10.1 (middleware пакет), Фаза 4 (auth service)

**Контекст:**
- `code_conventions_v4.md` — §9 (Auth Middleware)
- `services/auth_service_spec_v4.md` — §4 (ValidateToken)
- `pkg/ctxutil/` — `WithUserID`

**Что сделать:**

Реализовать auth middleware с consumer-defined `tokenValidator` интерфейсом. Написать unit-тесты.

---

#### `auth.go` — Auth middleware

```go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/google/uuid"
    "github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// tokenValidator определяет то, что middleware нужно от auth-сервиса.
type tokenValidator interface {
    ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

// Auth извлекает Bearer token из Authorization header.
// Валидный токен → userID в context.
// Нет токена → pass through (anonymous endpoints).
// Невалидный токен → 401.
func Auth(validator tokenValidator) Middleware {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                next.ServeHTTP(w, r)
                return
            }

            userID, err := validator.ValidateToken(r.Context(), token)
            if err != nil {
                http.Error(w, "unauthorized", http.StatusUnauthorized)
                return
            }

            ctx := ctxutil.WithUserID(r.Context(), userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if auth == "" {
        return ""
    }
    parts := strings.SplitN(auth, " ", 2)
    if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
        return ""
    }
    return parts[1]
}
```

---

#### Unit-тесты

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestAuth_NoToken_PassThrough` | Запрос без Authorization header | Handler вызван, userID нет в ctx |
| 2 | `TestAuth_ValidToken` | Bearer valid-token, validator → userID | Handler вызван, userID в ctx |
| 3 | `TestAuth_InvalidToken` | Bearer bad-token, validator → error | Статус 401, handler не вызван |
| 4 | `TestAuth_MalformedHeader` | "NotBearer token" | Pass through (не Bearer) |
| 5 | `TestAuth_EmptyBearer` | "Bearer " (пустой токен) | Pass through (token == "") |
| 6 | `TestAuth_ValidToken_UserIDInContext` | Валидный токен | `ctxutil.UserIDFromCtx(ctx)` возвращает правильный userID |
| 7 | `TestExtractBearerToken_Cases` | Table-driven: разные форматы Authorization | Корректный парсинг |

**Всего: 7 тест-кейсов**

**Паттерн тестирования:**

```go
func TestAuth_ValidToken(t *testing.T) {
    userID := uuid.New()
    validator := &tokenValidatorMock{
        ValidateTokenFunc: func(ctx context.Context, token string) (uuid.UUID, error) {
            if token == "valid-token" {
                return userID, nil
            }
            return uuid.Nil, errors.New("invalid")
        },
    }

    var gotUserID uuid.UUID
    inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotUserID, _ = ctxutil.UserIDFromCtx(r.Context())
        w.WriteHeader(http.StatusOK)
    })

    handler := Auth(validator)(inner)

    req := httptest.NewRequest(http.MethodPost, "/query", nil)
    req.Header.Set("Authorization", "Bearer valid-token")
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    assert.Equal(t, http.StatusOK, rec.Code)
    assert.Equal(t, userID, gotUserID)
}
```

> **Моки:** `//go:generate moq -out token_validator_mock_test.go -pkg middleware tokenValidator`

---

**Acceptance criteria:**
- [ ] `internal/transport/middleware/auth.go` создан
- [ ] `tokenValidator` интерфейс — приватный, consumer-defined
- [ ] Нет токена → pass through (anonymous)
- [ ] Невалидный токен → 401 Unauthorized
- [ ] Валидный токен → userID в context через `ctxutil.WithUserID`
- [ ] `extractBearerToken` парсит "Bearer <token>"
- [ ] 7 unit-тестов проходят
- [ ] `go build ./...` компилируется

---

### TASK-10.3: Health Endpoints (REST)

**Зависит от:** Фаза 2 (database pool)

**Контекст:**
- `infra_spec_v4.md` — §5 (Health Checks)
- `internal/app/version.go` — `BuildVersion()`

**Что сделать:**

Реализовать три REST-эндпоинта (`/live`, `/ready`, `/health`) в `internal/transport/rest/`. Написать unit-тесты.

---

#### `health.go` — Health handler

```go
package rest

import (
    "context"
    "encoding/json"
    "net/http"
    "time"
)

// dbPinger определяет минимальный интерфейс для проверки БД.
type dbPinger interface {
    Ping(ctx context.Context) error
}

// HealthHandler обрабатывает health check endpoints.
type HealthHandler struct {
    db      dbPinger
    version string
}

// NewHealthHandler создаёт HealthHandler.
func NewHealthHandler(db dbPinger, version string) *HealthHandler {
    return &HealthHandler{db: db, version: version}
}

// HealthResponse — JSON-ответ /health и /ready.
type HealthResponse struct {
    Status     string                `json:"status"`
    Version    string                `json:"version,omitempty"`
    Components map[string]CompStatus `json:"components,omitempty"`
    Timestamp  time.Time             `json:"timestamp"`
}

// CompStatus — статус отдельного компонента.
type CompStatus struct {
    Status  string `json:"status"`
    Latency string `json:"latency,omitempty"`
}
```

---

#### Три эндпоинта

**`/live`** — liveness probe. Всегда 200. Процесс жив.

```go
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, HealthResponse{
        Status:    "ok",
        Timestamp: time.Now(),
    })
}
```

**`/ready`** — readiness probe. Ping БД. 200 если доступна, 503 если нет.

```go
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    if err := h.db.Ping(ctx); err != nil {
        writeJSON(w, http.StatusServiceUnavailable, HealthResponse{
            Status:    "down",
            Timestamp: time.Now(),
        })
        return
    }

    writeJSON(w, http.StatusOK, HealthResponse{
        Status:    "ok",
        Timestamp: time.Now(),
    })
}
```

**`/health`** — полный health check. Ping БД + latency + версия.

```go
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
    defer cancel()

    components := make(map[string]CompStatus)
    overallStatus := "ok"

    start := time.Now()
    err := h.db.Ping(ctx)
    latency := time.Since(start)

    if err != nil {
        components["database"] = CompStatus{Status: "down"}
        overallStatus = "down"
    } else {
        components["database"] = CompStatus{
            Status:  "ok",
            Latency: latency.String(),
        }
    }

    status := http.StatusOK
    if overallStatus != "ok" {
        status = http.StatusServiceUnavailable
    }

    writeJSON(w, status, HealthResponse{
        Status:     overallStatus,
        Version:    h.version,
        Components: components,
        Timestamp:  time.Now(),
    })
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(v)
}
```

---

#### Unit-тесты

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestLive_Always200` | GET /live | Статус 200, `status: "ok"` |
| 2 | `TestReady_DBUp` | Ping → nil | Статус 200, `status: "ok"` |
| 3 | `TestReady_DBDown` | Ping → error | Статус 503, `status: "down"` |
| 4 | `TestHealth_AllOK` | Ping → nil | Статус 200, components.database.status = "ok", version присутствует |
| 5 | `TestHealth_DBDown` | Ping → error | Статус 503, components.database.status = "down" |
| 6 | `TestHealth_IncludesLatency` | Ping → nil | `components.database.latency` не пустой |

**Всего: 6 тест-кейсов**

> **Моки:** `//go:generate moq -out db_pinger_mock_test.go -pkg rest dbPinger`

---

**Acceptance criteria:**
- [ ] `internal/transport/rest/health.go` создан
- [ ] `dbPinger` интерфейс — приватный, consumer-defined
- [ ] `/live` → всегда 200, без проверок
- [ ] `/ready` → 200 если DB ping OK, 503 если нет
- [ ] `/health` → JSON с status, version, components (database + latency), timestamp
- [ ] Таймаут на DB ping: 3 секунды
- [ ] 6 unit-тестов проходят
- [ ] `go build ./...` компилируется

---

### TASK-10.4: Bootstrap Wiring + HTTP Server + Graceful Shutdown

**Зависит от:** TASK-10.1, TASK-10.2, TASK-10.3, все предыдущие фазы

**Контекст:**
- `infra_spec_v4.md` — §4 (HTTP-сервер), §6 (Graceful Shutdown), §10 (Bootstrap)
- `code_conventions_v4.md` — §1 (dependency injection)
- `services/service_layer_spec_v4.md` — §6 (карта сервисов и зависимости)

**Что сделать:**

Расширить `internal/app/app.go` — полный bootstrap: создание всех repos, services, GraphQL handler, middleware chain, HTTP server, graceful shutdown.

---

#### Порядок инициализации (в `app.Run`)

```
 1. Загрузить и провалидировать конфиг                  
 2. Инициализировать логгер                              
 3. Подключиться к БД (pool)                             
 4. Создать TxManager
 5. Создать репозитории (14 пакетов)
 6. Создать внешние провайдеры (FreeDictionary, Translation stub)
 7. Создать сервисы (8 пакетов)
 8. Создать GraphQL resolver + handler
 9. Создать DataLoader middleware
10. Создать Health handler
11. Собрать middleware chain
12. Создать ServeMux и зарегистрировать роуты
13. Создать и запустить HTTP server
14. Ожидать сигнал → graceful shutdown
```

---

#### Шаг 4: TxManager

```go
txm := postgres.NewTxManager(pool)
```

---

#### Шаг 5: Репозитории

```go
// Repos
entryRepo       := entry.New(pool)
senseRepo       := sense.New(pool, txm)
translationRepo := translation.New(pool, txm)
exampleRepo     := example.New(pool, txm)
pronunciationRepo := pronunciation.New(pool)
imageRepo       := image.New(pool)
cardRepo        := card.New(pool)
reviewLogRepo   := reviewlog.New(pool)
topicRepo       := topic.New(pool)
inboxRepo       := inbox.New(pool)
userRepo        := user.New(pool)
tokenRepo       := token.New(pool)
auditRepo       := audit.New(pool)
refEntryRepo    := refentry.New(pool, txm)
```

> **Примечание:** `sessionRepo` для Study Service. Если в текущей кодовой базе session repo реализован внутри другого пакета (например, reviewlog), использовать его. Если нет — потребуется реализация из Фазы 7.

---

#### Шаг 6: Внешние провайдеры

```go
freedictProvider := freedict.NewProvider(logger)
transProvider    := translate.NewStub()
```

---

#### Шаг 7: Сервисы

```go
// Auth + User (Фаза 4)
// Конкретные конструкторы зависят от реализации Фазы 4.
// authSvc := auth.NewService(logger, userRepo, settingsRepo, tokenRepo, txm, oauthVerifier, jwtManager, cfg.Auth)
// userSvc := user.NewService(logger, userRepo, settingsRepo)

// Business services
refCatalogSvc := refcatalog.NewService(logger, refEntryRepo, txm, freedictProvider, transProvider)
dictSvc       := dictionary.NewService(logger, entryRepo, senseRepo, translationRepo, exampleRepo, pronunciationRepo, imageRepo, cardRepo, auditRepo, txm)
contentSvc    := content.NewService(logger, entryRepo, senseRepo, translationRepo, exampleRepo, imageRepo, auditRepo, txm)
studySvc      := study.NewService(logger, cardRepo, reviewLogRepo, sessionRepo, entryRepo, senseRepo, settingsRepo, auditRepo, txm, domain.SRSConfigFromConfig(cfg.SRS))
topicSvc      := topic.NewService(logger, topicRepo, entryRepo, auditRepo, txm)
inboxSvc      := inbox.NewService(logger, inboxRepo)
```

> **Примечание:** `dictionary.NewService` может также принимать `refCatalogSvc` — единственная межсервисная зависимость (см. `service_layer_spec §6.2`). Точная сигнатура зависит от реализации Фазы 5. При необходимости добавить `refCatalogSvc` как параметр.

> **SRS конфиг:** Study Service принимает `domain.SRSConfig`. Если `domain.SRSConfigFromConfig(cfg.SRS)` не существует — создать маппинг `config.SRSConfig → domain.SRSConfig` здесь или в domain.

---

#### Шаг 8: GraphQL handler

```go
import (
    gqlhandler "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/playground"
    "github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
    "github.com/heartmarshall/myenglish-backend/internal/transport/graphql/resolver"
    graphqlpkg "github.com/heartmarshall/myenglish-backend/internal/transport/graphql"
)

res := resolver.NewResolver(logger, dictSvc, contentSvc, studySvc, topicSvc, inboxSvc, userSvc)

gqlSrv := gqlhandler.NewDefaultServer(
    generated.NewExecutableSchema(generated.Config{
        Resolvers: res,
    }),
)
gqlSrv.SetErrorPresenter(graphqlpkg.NewErrorPresenter(logger))
```

---

#### Шаг 9: DataLoader middleware

```go
dlRepos := &dataloader.Repos{
    Sense:         senseRepo,
    Translation:   translationRepo,
    Example:       exampleRepo,
    Pronunciation: pronunciationRepo,
    Image:         imageRepo,
    Card:          cardRepo,
    Topic:         topicRepo,
    ReviewLog:     reviewLogRepo,
}
```

---

#### Шаг 10: Health handler

```go
healthHandler := rest.NewHealthHandler(pool, app.BuildVersion())
```

> **`*pgxpool.Pool`** уже реализует `Ping(ctx) error` — удовлетворяет интерфейсу `dbPinger`.

---

#### Шаг 11: Middleware chain

```go
graphqlHandler := middleware.Chain(
    middleware.Recovery(logger),
    middleware.RequestID(),
    middleware.Logger(logger),
    middleware.CORS(cfg.CORS),
    middleware.Auth(authSvc), // authSvc реализует tokenValidator
    dataloader.Middleware(dlRepos),
)(gqlSrv)
```

**Порядок (outermost → innermost):**
1. Recovery — ловит panic из всех нижних middleware
2. RequestID — генерирует ID, доступен в Logger и далее
3. Logger — логирует запрос с request_id
4. CORS — обрабатывает preflight, устанавливает заголовки
5. Auth — валидирует JWT, кладёт userID в ctx
6. DataLoader — создаёт per-request DataLoaders (после Auth, т.к. DataLoaders могут использовать userID)

---

#### Шаг 12: ServeMux и routing

```go
mux := http.NewServeMux()

// Health endpoints — вне middleware-стека
mux.HandleFunc("GET /live", healthHandler.Live)
mux.HandleFunc("GET /ready", healthHandler.Ready)
mux.HandleFunc("GET /health", healthHandler.Health)

// GraphQL — полная middleware chain
mux.Handle("POST /query", graphqlHandler)

// Playground (conditional)
if cfg.GraphQL.PlaygroundEnabled {
    mux.Handle("GET /", playground.Handler("MyEnglish GraphQL", "/query"))
    logger.Info("GraphQL playground enabled", slog.String("path", "/"))
}
```

---

#### Шаг 13: HTTP server

```go
addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

srv := &http.Server{
    Addr:         addr,
    Handler:      mux,
    ReadTimeout:  cfg.Server.ReadTimeout,
    WriteTimeout: cfg.Server.WriteTimeout,
    IdleTimeout:  cfg.Server.IdleTimeout,
}

// Запуск в горутине
go func() {
    logger.Info("HTTP server started", slog.String("addr", addr))
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Error("HTTP server error", slog.String("error", err.Error()))
    }
}()
```

---

#### Шаг 14: Graceful shutdown

```go
// Ожидание сигнала (ctx создаётся в main.go через signal.NotifyContext)
<-ctx.Done()
logger.Info("shutdown signal received")

// 1. HTTP server shutdown (даём in-flight запросам завершиться)
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
defer shutdownCancel()

if err := srv.Shutdown(shutdownCtx); err != nil {
    logger.Error("HTTP server shutdown error", slog.String("error", err.Error()))
}
logger.Info("HTTP server stopped")

// 2. Закрытие БД pool (уже в defer pool.Close() выше, но логируем)
// pool.Close() вызывается через defer
logger.Info("shutdown complete")
```

**Порядок остановки (обратный запуску):**
1. HTTP server (Shutdown с таймаутом) → прекращает приём новых запросов, ждёт in-flight
2. DB pool (Close) → закрывает все соединения

---

#### Обновлённый `cmd/server/main.go`

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/heartmarshall/myenglish-backend/internal/app"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    if err := app.Run(ctx); err != nil {
        log.Fatalf("application error: %v", err)
    }

    os.Exit(0)
}
```

> **Примечание:** Текущий `main.go` уже создаёт signal context и вызывает `app.Run(ctx)`. Убедиться что используется `signal.NotifyContext` с `SIGINT` и `SIGTERM`.

---

#### Fail Fast

Любая ошибка на этапах 1–12 → `return fmt.Errorf(...)`. Сервер НЕ запускается с неполной инициализацией.

```go
func Run(ctx context.Context) error {
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("load config: %w", err)
    }
    // ... каждый шаг проверяет err ...
    // Если ошибка — return, сервер не стартует
}
```

---

**Acceptance criteria:**
- [ ] `internal/app/app.go` расширен — полный bootstrap от config до HTTP server
- [ ] Создаются все 14 репозиториев
- [ ] Создаются TxManager и все 8 сервисов
- [ ] GraphQL handler собран: resolver + error presenter + gqlgen server
- [ ] DataLoader middleware собран с Repos struct
- [ ] Middleware chain применяется в правильном порядке (6 middleware)
- [ ] Health endpoints зарегистрированы вне middleware-стека
- [ ] GraphQL на `POST /query`
- [ ] Playground условно (PlaygroundEnabled)
- [ ] HTTP server с таймаутами из ServerConfig
- [ ] Graceful shutdown: HTTP server.Shutdown → pool.Close
- [ ] Fail fast: ошибка инициализации → server не стартует
- [ ] `go build ./...` компилируется
- [ ] `docker-compose up` запускает полный стек (postgres → migrate → backend)

---

### TASK-10.5: Audit Log Cleanup (BG3)

**Зависит от:** Фаза 2 (audit repo, cleanup command)

**Контекст:**
- `services/business_scenarios_v4.md` — BG3: "audit_log старше 1 года → DELETE"
- `cmd/cleanup/main.go` — существующий cleanup command (BG1: hard delete entries)

**Что сделать:**

Расширить `cmd/cleanup/main.go` для поддержки audit cleanup. Добавить метод `DeleteOlderThan` в audit repo (если отсутствует).

---

#### 1. Добавить конфигурацию

**Файл:** `internal/config/config.go`

Добавить в `DictionaryConfig` (или создать отдельную `CleanupConfig`):

```go
// В существующую DictionaryConfig или отдельную секцию:
AuditRetentionDays int `yaml:"audit_retention_days" env:"AUDIT_RETENTION_DAYS" env-default:"365"`
```

---

#### 2. Метод audit repo

Если `audit.Repo` ещё не имеет метода для удаления старых записей — добавить:

```go
// adapter/postgres/audit/repo.go

// DeleteOlderThan удаляет audit_log записи старше указанной даты.
// Возвращает количество удалённых записей.
func (r *Repo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
    result, err := r.pool.Exec(ctx,
        "DELETE FROM audit_log WHERE created_at < $1", before)
    if err != nil {
        return 0, fmt.Errorf("audit.DeleteOlderThan: %w", err)
    }
    return result.RowsAffected(), nil
}
```

---

#### 3. Расширить cleanup command

**Файл:** `cmd/cleanup/main.go`

Добавить флаг `--audit` и обработку:

```go
func main() {
    auditFlag := flag.Bool("audit", false, "cleanup audit_log entries older than retention period")
    entriesFlag := flag.Bool("entries", true, "cleanup soft-deleted entries older than retention period")
    flag.Parse()

    cfg, err := config.Load()
    // ... config, logger, pool (существующий код) ...

    if *entriesFlag {
        // ... существующий код BG1 ...
    }

    if *auditFlag {
        auditRepo := audit.New(pool)
        threshold := time.Now().AddDate(0, 0, -cfg.Dictionary.AuditRetentionDays)

        deleted, err := auditRepo.DeleteOlderThan(ctx, threshold)
        if err != nil {
            logger.Error("audit cleanup failed",
                slog.String("error", err.Error()),
                slog.Time("threshold", threshold),
            )
            os.Exit(1)
        }
        logger.Info("audit cleanup completed",
            slog.Int64("deleted", deleted),
            slog.Time("threshold", threshold),
        )
    }
}
```

**Использование:**

```bash
# Только entries (по умолчанию)
./cleanup

# Только audit
./cleanup --entries=false --audit

# Оба
./cleanup --audit
```

---

#### Тесты

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestDeleteOlderThan_Success` | 3 записи (2 старых, 1 новая) | deleted = 2, новая запись осталась |
| 2 | `TestDeleteOlderThan_NoOldRecords` | Все записи свежие | deleted = 0 |
| 3 | `TestDeleteOlderThan_EmptyTable` | Пустая таблица | deleted = 0, нет ошибок |

**Всего: 3 тест-кейса** (integration tests с testcontainers)

---

**Acceptance criteria:**
- [ ] `AuditRetentionDays` добавлен в конфигурацию (default: 365)
- [ ] `audit.Repo.DeleteOlderThan(ctx, before)` реализован
- [ ] `cmd/cleanup/main.go` поддерживает `--audit` и `--entries` флаги
- [ ] По умолчанию: entries cleanup (обратная совместимость)
- [ ] `--audit` можно запускать совместно с `--entries`
- [ ] 3 integration-теста проходят
- [ ] `go build ./cmd/cleanup/...` компилируется

---

### TASK-10.6: E2E Smoke Tests

**Зависит от:** TASK-10.4 (полный bootstrap)

**Контекст:**
- Все спецификации сервисов
- `code_conventions_v4.md` — §7 (тестирование, testcontainers)

**Что сделать:**

Создать test helper для bootstrap полного стека с testcontainers. Написать 5–10 smoke tests, проверяющих что wiring работает корректно.

---

#### Test Helper

**Файл:** `tests/e2e/helpers_test.go`

```go
package e2e_test

import (
    "context"
    "net/http"
    "testing"
    // imports...
)

// testServer содержит полный стек для E2E тестов.
type testServer struct {
    URL    string        // http://localhost:<port>
    Client *http.Client
    Pool   *pgxpool.Pool
    // cleanup func() — вызывать в defer
}

// setupTestServer поднимает полный стек:
// 1. Testcontainers PostgreSQL
// 2. Goose migrations
// 3. Полный bootstrap (repos → services → resolvers → middleware → server)
// 4. Запуск HTTP server на случайном порту
func setupTestServer(t *testing.T) *testServer {
    t.Helper()

    // 1. PostgreSQL container
    ctx := context.Background()
    pgContainer := startPostgres(t, ctx)

    // 2. Pool + Migrations
    pool := createPool(t, ctx, pgContainer.DSN())
    runMigrations(t, pgContainer.DSN())

    // 3. Bootstrap (аналогично app.Run, но без signal handling)
    // Создать repos, services, handler, middleware, mux

    // 4. httptest.NewServer(mux)
    srv := httptest.NewServer(mux)
    t.Cleanup(func() {
        srv.Close()
        pool.Close()
        pgContainer.Terminate(ctx)
    })

    return &testServer{
        URL:    srv.URL,
        Client: srv.Client(),
        Pool:   pool,
    }
}
```

> **Примечание:** Для auth flow в E2E тестах нужен мок OAuth verifier (не вызывать реальный Google). JWT manager использует тестовый секрет. Создать тестового пользователя напрямую в БД или через мокнутый auth flow.

---

#### Helper: GraphQL запрос

```go
// graphqlQuery выполняет GraphQL запрос и возвращает response body.
func (ts *testServer) graphqlQuery(t *testing.T, query string, variables map[string]any, token string) map[string]any {
    t.Helper()

    body := map[string]any{
        "query":     query,
        "variables": variables,
    }
    jsonBody, _ := json.Marshal(body)

    req, _ := http.NewRequest(http.MethodPost, ts.URL+"/query", bytes.NewReader(jsonBody))
    req.Header.Set("Content-Type", "application/json")
    if token != "" {
        req.Header.Set("Authorization", "Bearer "+token)
    }

    resp, err := ts.Client.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()

    var result map[string]any
    require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
    return result
}
```

---

#### E2E Test Scenarios

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestE2E_LiveEndpoint` | GET /live | Статус 200, `{"status":"ok"}` |
| 2 | `TestE2E_ReadyEndpoint` | GET /ready (DB up) | Статус 200, `{"status":"ok"}` |
| 3 | `TestE2E_HealthEndpoint` | GET /health | Статус 200, version не пустой, database.status = "ok" |
| 4 | `TestE2E_GraphQL_Unauthenticated` | Query `searchCatalog` без auth | Работает (anonymous endpoint) |
| 5 | `TestE2E_GraphQL_AuthRequired` | Mutation `createTopic` без auth | Error с кодом `UNAUTHENTICATED` |
| 6 | `TestE2E_GraphQL_CRUD_Topic` | Auth → createTopic → topics → deleteTopic | Полный CRUD цикл проходит |
| 7 | `TestE2E_RequestID_InResponse` | Любой запрос | X-Request-Id в response header |
| 8 | `TestE2E_CORS_Preflight` | OPTIONS /query с Origin | Access-Control-Allow-* headers |
| 9 | `TestE2E_GraphQL_NotFound` | Query `dictionaryEntry(id: random)` с auth | Error с кодом `NOT_FOUND` |
| 10 | `TestE2E_GraphQL_ValidationError` | Mutation `createTopic(input: {name: ""})` с auth | Error с кодом `VALIDATION`, fields в extensions |

**Всего: 10 тест-кейсов**

---

#### Пример: CRUD цикл

```go
func TestE2E_GraphQL_CRUD_Topic(t *testing.T) {
    ts := setupTestServer(t)
    token := createTestUserAndGetToken(t, ts) // helper: insert user + generate JWT

    // Create
    result := ts.graphqlQuery(t, `
        mutation($input: CreateTopicInput!) {
            createTopic(input: $input) {
                topic { id name description }
            }
        }
    `, map[string]any{
        "input": map[string]any{"name": "E2E Test Topic", "description": "test"},
    }, token)

    require.Nil(t, result["errors"])
    data := result["data"].(map[string]any)
    topicData := data["createTopic"].(map[string]any)["topic"].(map[string]any)
    topicID := topicData["id"].(string)
    assert.Equal(t, "E2E Test Topic", topicData["name"])

    // List
    result = ts.graphqlQuery(t, `{ topics { id name } }`, nil, token)
    require.Nil(t, result["errors"])
    topics := result["data"].(map[string]any)["topics"].([]any)
    assert.Len(t, topics, 1)

    // Delete
    result = ts.graphqlQuery(t, `
        mutation($id: UUID!) {
            deleteTopic(id: $id) { topicId }
        }
    `, map[string]any{"id": topicID}, token)
    require.Nil(t, result["errors"])

    // Verify deleted
    result = ts.graphqlQuery(t, `{ topics { id } }`, nil, token)
    topics = result["data"].(map[string]any)["topics"].([]any)
    assert.Empty(t, topics)
}
```

---

#### Build tag

E2E тесты запускаются только с build tag `e2e` или `integration`:

```go
//go:build e2e

package e2e_test
```

Запуск:

```bash
go test -tags=e2e -count=1 -v ./tests/e2e/...
```

Добавить цель в Makefile:

```makefile
test-e2e:
	go test -tags=e2e -count=1 -v ./tests/e2e/...
```

---

**Acceptance criteria:**
- [ ] `tests/e2e/` создан с build tag `e2e`
- [ ] `setupTestServer` поднимает PostgreSQL в testcontainer, применяет миграции, bootstrap полный стек
- [ ] Helper `graphqlQuery` отправляет GraphQL запросы
- [ ] Helper `createTestUserAndGetToken` создаёт тестового пользователя и JWT
- [ ] 10 smoke tests проходят: health, auth, CRUD, error codes, middleware
- [ ] `make test-e2e` работает
- [ ] Тесты изолированы друг от друга (каждый тест — свой пользователь или cleanup)
- [ ] Тесты не запускаются без build tag (не ломают `go test ./...`)

---

## Граф зависимостей задач

```
TASK-10.1 (Infra Middleware) ──┐
                               ├──→ TASK-10.4 (Bootstrap Wiring) ──→ TASK-10.6 (E2E Tests)
TASK-10.2 (Auth Middleware)  ──┤
                               │
TASK-10.3 (Health Endpoints) ──┘

TASK-10.5 (Audit Cleanup) — независимая, может выполняться параллельно
```

**Рекомендуемый порядок:**
1. TASK-10.1 + TASK-10.3 + TASK-10.5 — параллельно
2. TASK-10.2 — после TASK-10.1
3. TASK-10.4 — после TASK-10.1, 10.2, 10.3
4. TASK-10.6 — после TASK-10.4
