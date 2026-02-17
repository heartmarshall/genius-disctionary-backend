# Фаза 4: Сервисный слой — Auth & User


## Документы-источники

| Документ | Релевантные секции |
|----------|-------------------|
| `code_conventions_v4.md` | §1 (интерфейсы потребителем, правило зависимостей), §2 (обработка ошибок), §3 (валидация), §4 (контекст и user identity), §5 (логирование), §6 (аудит), §7 (тестирование, moq), §9 (аутентификация: OAuth flow, JWT claims, auth middleware) |
| `services/service_layer_spec_v4.md` | §2 (структура пакетов), §3 (паттерны: интерфейсы, input, валидация, транзакции, error handling, логирование), §4 (аудит: USER settings → update), §6 (карта сервисов: AuthService, UserService), §7 (тестирование: моки, категории, mock TxManager) |
| `services/auth_service_spec_v4.md` | Все секции — полная спецификация Auth Service: зависимости, операции, flows, corner cases, error scenarios, тесты |
| `data_model_v4.md` | §3 (users, user_settings, refresh_tokens) |
| `infra_spec_v4.md` | §2 (конфигурация: cleanenv), §4 (middleware-стек), §9 (зависимости: golang-jwt) |

---

## Пре-условия (из Фазы 1)

Перед началом Фазы 4 должны быть готовы:
- Domain-модели: `User`, `UserSettings`, `RefreshToken`, `OAuthProvider` (`internal/domain/user.go`)
- `DefaultUserSettings(userID)` — конструктор дефолтных настроек (`internal/domain/user.go`)
- `RefreshToken.IsRevoked()`, `RefreshToken.IsExpired(now)` — helper-методы (`internal/domain/user.go`)
- Domain errors: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized` (`internal/domain/errors.go`)
- `ValidationError`, `FieldError`, `NewValidationError()` (`internal/domain/errors.go`)
- Context helpers: `ctxutil.WithUserID`, `ctxutil.UserIDFromCtx` → `(uuid.UUID, bool)` (`pkg/ctxutil/`)
- Context helpers: `ctxutil.WithRequestID`, `ctxutil.RequestIDFromCtx` (`pkg/ctxutil/`)
- Config: `AuthConfig` с JWTSecret, AccessTokenTTL, RefreshTokenTTL, Google/Apple credentials (`internal/config/`)
- Enums: `OAuthProviderGoogle = "google"`, `OAuthProviderApple = "apple"` (`internal/domain/enums.go`)

> **Важно:** Фаза 4 **не зависит** от Фаз 2 и 3. Все зависимости на репозитории и TxManager мокаются в unit-тестах. Сервисы можно разрабатывать параллельно с инфраструктурой БД и слоем репозиториев.

---

## Зафиксированные решения

| # | Вопрос | Решение |
|---|--------|---------|
| 1 | JWT алгоритм | HS256 (symmetric, MVP). Переход на RS256 — post-MVP |
| 2 | JWT библиотека | `golang-jwt/jwt/v5` |
| 3 | Refresh token формат | 32 bytes `crypto/rand` → base64url encoding (raw). SHA-256 hex (hash для хранения) |
| 4 | OAuthIdentity расположение | `internal/auth/identity.go` — shared пакет, импортируется и service, и adapter |
| 5 | OAuth верификация (Google) | Exchange authorization code → access token → userinfo endpoint. MVP-подход |
| 6 | AllowedProviders | Вычисляются методом `AuthConfig.AllowedProviders()`: provider включён, если **все** его credentials заполнены (совместимо с `hasGoogleOAuth()`/`hasAppleOAuth()` из `validate.go`) |
| 7 | Token reuse detection | На MVP — reject + WARN лог "refresh token reuse attempted" (как в auth_service_spec §4.2). Revoke all — post-MVP |
| 8 | Email collision | На MVP — ошибка `ErrAlreadyExists`. Merge accounts — post-MVP |
| 9 | Auth endpoints | REST (не GraphQL) — реализуются в фазе transport layer (не в этой фазе) |
| 10 | Audit для auth | Login/Logout/Refresh **не** аудитируются в `audit_log`. Логируются через slog |
| 11 | Audit для user settings | Аудитируется в `audit_log` внутри транзакции. Требует `EntityTypeUser` в enum |
| 12 | Моки | `moq` (code generation) — моки генерируются в `_test.go` файлы |
| 13 | Mock TxManager | `RunInTx(ctx, fn)` просто вызывает `fn(ctx)` без реальной транзакции |
| 14 | Auth Middleware | Определяет свой узкий интерфейс `tokenValidator`. Живёт в `internal/transport/middleware/` |
| 15 | `UserIDFromCtx` в сервисах | Текущая сигнатура: `(uuid.UUID, bool)`. Сервисы проверяют `ok` и возвращают `domain.ErrUnauthorized` |
| 16 | Repo method naming | Auth service spec использует `CreateUser`/`UpdateUser`. В реализации (и Phase 3) используем `Create`/`Update` — идиоматичнее Go (контекст из имени пакета) |
| 17 | `GetByOAuth` parameter type | Auth service spec использует `provider string`. В реализации — `provider domain.OAuthProvider` для type safety. Каст в сервисе: `domain.OAuthProvider(input.Provider)` |
| 18 | `derefOrEmpty` utility | Приватная helper-функция в `service/auth/service.go` для конвертации `*string` → `string` (domain.User.Name — `string`, OAuthIdentity.Name — `*string`) |

---

## Задачи

### TASK-4.1: Расширение конфигурации для Auth

**Зависит от:** Фаза 1 (config уже создан)

**Контекст:**
- `services/auth_service_spec_v4.md` — §2.4 (AuthConfig: JWTIssuer, AllowedProviders)
- `infra_spec_v4.md` — §2 (конфигурация: cleanenv)
- Текущий `internal/config/config.go` — AuthConfig без JWTIssuer и AllowedProviders

**Что сделать:**

Расширить `internal/config/config.go` и `internal/config/validate.go`.

**Изменения в `AuthConfig`:**

Добавить поля:

| Поле | Тип | env | Default | Описание |
|------|-----|-----|---------|----------|
| `JWTIssuer` | `string` | `AUTH_JWT_ISSUER` | `"myenglish"` | Issuer claim в JWT access token |
| `GoogleRedirectURI` | `string` | `AUTH_GOOGLE_REDIRECT_URI` | — | Redirect URI для Google OAuth (должен совпадать с тем, что клиент передал Google) |

```go
type AuthConfig struct {
    // ... существующие поля ...
    JWTIssuer         string `yaml:"jwt_issuer"          env:"AUTH_JWT_ISSUER"          env-default:"myenglish"`
    GoogleRedirectURI string `yaml:"google_redirect_uri" env:"AUTH_GOOGLE_REDIRECT_URI"`
}
```

**Computed methods — AllowedProviders:**

Не хранятся в конфиге. Вычисляются из заполненных credentials. **Важно:** логика должна быть **идентична** существующим `hasGoogleOAuth()` и `hasAppleOAuth()` в `validate.go`:

```go
// AllowedProviders returns the list of configured OAuth providers.
// Logic must match hasGoogleOAuth()/hasAppleOAuth() in validate.go.
func (c AuthConfig) AllowedProviders() []string {
    var providers []string
    if c.GoogleClientID != "" && c.GoogleClientSecret != "" {
        providers = append(providers, string(domain.OAuthProviderGoogle))
    }
    if c.AppleKeyID != "" && c.AppleTeamID != "" && c.ApplePrivateKey != "" {
        providers = append(providers, string(domain.OAuthProviderApple))
    }
    return providers
}

// IsProviderAllowed checks if the given provider string is configured.
func (c AuthConfig) IsProviderAllowed(provider string) bool {
    for _, p := range c.AllowedProviders() {
        if p == provider {
            return true
        }
    }
    return false
}
```

**Обновление валидации (`validate.go`):**

Рефакторинг: заменить `hasGoogleOAuth()`/`hasAppleOAuth()` на вызов `AllowedProviders()`, чтобы логика не дублировалась:

```go
func (c *Config) Validate() error {
    // ...
    if len(c.Auth.AllowedProviders()) == 0 {
        return fmt.Errorf("at least one OAuth provider must be configured (Google or Apple)")
    }
    // ...
}
```

Удалить `hasGoogleOAuth()` и `hasAppleOAuth()` — их логика теперь в `AllowedProviders()`.

**Acceptance criteria:**
- [ ] `JWTIssuer` добавлен в `AuthConfig` с дефолтом `"myenglish"`
- [ ] `GoogleRedirectURI` добавлен в `AuthConfig`
- [ ] `env` теги: `AUTH_JWT_ISSUER`, `AUTH_GOOGLE_REDIRECT_URI`
- [ ] Метод `AllowedProviders()` возвращает `[]string` на основе **полностью** заполненных credentials
- [ ] `AllowedProviders()`: Google enabled только если `GoogleClientID != ""` **И** `GoogleClientSecret != ""`
- [ ] `AllowedProviders()`: Apple enabled только если `AppleKeyID != ""` **И** `AppleTeamID != ""` **И** `ApplePrivateKey != ""`
- [ ] `AllowedProviders()` при незаполненных credentials → пустой slice
- [ ] Метод `IsProviderAllowed(provider)` для проверки одного провайдера
- [ ] `hasGoogleOAuth()`/`hasAppleOAuth()` удалены из `validate.go` — логика перенесена в `AllowedProviders()`
- [ ] Валидация: `len(AllowedProviders()) == 0` → ошибка
- [ ] Unit-тесты: `AllowedProviders` с разными комбинациями (только Google, только Apple, оба, ни одного, partial credentials)
- [ ] Unit-тесты: `IsProviderAllowed` — positive и negative
- [ ] `go build ./...` компилируется

**Corner cases:**
- `GoogleClientID` заполнен, `GoogleClientSecret` пуст → Google **не считается** enabled (совместимо с существующей `hasGoogleOAuth()` в validate.go, которая требует оба поля)
- `JWTIssuer` пустая строка → допустимо, но бессмысленно. Можно добавить проверку в validate, но не критично для MVP
- `GoogleRedirectURI` пустой при наличии Google credentials → не ошибка на уровне config validation, но Google OAuth Verifier не сможет работать. Можно добавить проверку в validate: если Google enabled → `GoogleRedirectURI` обязателен

---

### TASK-4.2: Пакет internal/auth/ — JWT Manager + OAuthIdentity

**Зависит от:** ничего (параллельно с остальными)

**Контекст:**
- `services/auth_service_spec_v4.md` — §2.3 (jwtManager interface, OAuthIdentity)
- `code_conventions_v4.md` — §9 (JWT claims: sub, exp, iat, iss; HS256)
- `infra_spec_v4.md` — §9 (зависимость: `golang-jwt/jwt/v5`)

**Что сделать:**

Создать пакет `internal/auth/` с JWT-менеджером и типом OAuthIdentity.

**Файловая структура:**

```
internal/auth/
├── identity.go      # OAuthIdentity struct
├── jwt.go           # JWTManager struct и методы
└── jwt_test.go      # Unit-тесты JWT
```

**`identity.go` — OAuthIdentity:**

```go
// OAuthIdentity represents user information obtained from an OAuth provider.
type OAuthIdentity struct {
    Email      string
    Name       *string
    AvatarURL  *string
    ProviderID string
}
```

Тип живёт в `internal/auth/` потому что:
- Используется сервисом `service/auth/` — определяет interface с этим типом в return
- Реализуется адаптерами `adapter/provider/google/` (и в будущем `adapter/provider/apple/`)
- Не является core domain entity — это auth-инфраструктурный тип
- Оба слоя (service и adapter) могут импортировать `internal/auth/` без нарушения правила зависимостей

**`jwt.go` — JWTManager:**

```go
type JWTManager struct {
    secret    []byte
    issuer    string
    accessTTL time.Duration
}

func NewJWTManager(secret string, issuer string, accessTTL time.Duration) *JWTManager
```

**Методы:**

| Метод | Описание |
|-------|----------|
| `GenerateAccessToken(userID uuid.UUID) (string, error)` | JWT с claims: sub=userID, exp=now+TTL, iat=now, iss=issuer. Signing: HS256 |
| `ValidateAccessToken(tokenString string) (uuid.UUID, error)` | Верификация подписи, expiry, issuer. Возвращает userID из sub claim |
| `GenerateRefreshToken() (raw string, hash string, error)` | 32 bytes `crypto/rand` → base64url (raw). SHA-256 hex (hash) |
| `HashToken(raw string) string` | SHA-256 hex hash строки. Детерминистичен |

**JWT Claims:**

```go
type accessClaims struct {
    jwt.RegisteredClaims
}
```

Минимальные стандартные claims:
- `Subject` (sub) = `userID.String()`
- `ExpiresAt` (exp) = `now + accessTTL`
- `IssuedAt` (iat) = `now`
- `Issuer` (iss) = `issuer` из конфига

Без custom claims (ролей, email, permissions) — они могут измениться и потребуют revocation.

**GenerateAccessToken:**

```go
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID) (string, error) {
    now := time.Now()
    claims := accessClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   userID.String(),
            ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
            IssuedAt:  jwt.NewNumericDate(now),
            Issuer:    m.issuer,
        },
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(m.secret)
}
```

**ValidateAccessToken:**

```go
func (m *JWTManager) ValidateAccessToken(tokenString string) (uuid.UUID, error) {
    token, err := jwt.ParseWithClaims(tokenString, &accessClaims{},
        func(token *jwt.Token) (any, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return m.secret, nil
        },
        jwt.WithIssuer(m.issuer),
        jwt.WithValidMethods([]string{"HS256"}),
    )
    if err != nil {
        return uuid.Nil, fmt.Errorf("validate access token: %w", err)
    }
    claims, ok := token.Claims.(*accessClaims)
    if !ok {
        return uuid.Nil, fmt.Errorf("validate access token: unexpected claims type")
    }
    userID, err := uuid.Parse(claims.Subject)
    if err != nil {
        return uuid.Nil, fmt.Errorf("validate access token: invalid subject: %w", err)
    }
    return userID, nil
}
```

**GenerateRefreshToken и HashToken:**

```go
func (m *JWTManager) GenerateRefreshToken() (string, string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", "", fmt.Errorf("generate refresh token: %w", err)
    }
    raw := base64.RawURLEncoding.EncodeToString(b)
    hash := HashToken(raw)
    return raw, hash, nil
}

// HashToken returns the SHA-256 hex digest of the given token string.
func HashToken(raw string) string {
    h := sha256.Sum256([]byte(raw))
    return hex.EncodeToString(h[:])
}
```

`HashToken` — package-level function (не метод), т.к. используется и в сервисе (Refresh flow) без экземпляра JWTManager.

**Новые зависимости в go.mod:**
- `github.com/golang-jwt/jwt/v5`

**Acceptance criteria:**
- [ ] `OAuthIdentity` struct создан в `internal/auth/identity.go`
- [ ] `JWTManager` создан в `internal/auth/jwt.go` с конструктором
- [ ] `GenerateAccessToken` генерирует валидный JWT с HS256
- [ ] Claims: sub = userID, exp = now + TTL, iat = now, iss = issuer
- [ ] `ValidateAccessToken` верифицирует подпись, expiry, issuer, signing method
- [ ] `ValidateAccessToken` возвращает `uuid.UUID` из sub claim
- [ ] Невалидная подпись → error
- [ ] Expired token → error
- [ ] Malformed token (не JWT) → error
- [ ] Неверный issuer → error
- [ ] Неверный signing method (алгоритм) → error
- [ ] Невалидный UUID в subject → error
- [ ] `GenerateRefreshToken` возвращает crypto-random raw (base64url) + SHA-256 hex hash
- [ ] Каждый вызов `GenerateRefreshToken` возвращает уникальную пару
- [ ] `HashToken` детерминистичен: одинаковый input → одинаковый output
- [ ] `HashToken` — package-level function
- [ ] `golang-jwt/jwt/v5` добавлен в `go.mod`
- [ ] Unit-тесты: generate+validate roundtrip (success)
- [ ] Unit-тесты: expired token, invalid signature, malformed, wrong issuer, wrong method
- [ ] Unit-тесты: refresh token generation — unique per call, hash matches
- [ ] Unit-тесты: `HashToken` determinism
- [ ] `go build ./...` компилируется

**Corner cases:**
- `GenerateAccessToken` с `uuid.Nil` — технически допустимо на уровне JWT, но не должно происходить (сервис не передаёт Nil). Тестировать не нужно
- `ValidateAccessToken` с пустой строкой → error (malformed)
- Время: `jwt.RegisteredClaims` использует `jwt.NewNumericDate(time.Now())`. Для unit-тестов generate+validate roundtrip проблем нет (token будет валиден сразу). Для тестов expired — создать token с истёкшим exp напрямую
- `HashToken` не зависит от JWTManager — можно тестировать изолированно

---

### TASK-4.3: OAuth Verifier — Google

**Зависит от:** TASK-4.2 (тип `auth.OAuthIdentity`)

**Контекст:**
- `services/auth_service_spec_v4.md` — §2.3 (oauthVerifier interface), §7.4 (OAuth Code одноразовый)
- `services/service_layer_spec_v4.md` — §6.3 (External Providers: timeout 10s, retry 1 при 5xx)
- `code_conventions_v4.md` — §5 (логирование: ERROR для внешних API)

**Что сделать:**

Создать пакет `internal/adapter/provider/google/` с реализацией OAuth-верификации для Google.

**Файловая структура:**

```
internal/adapter/provider/google/
├── verifier.go       # Verifier struct и методы
└── verifier_test.go  # Unit-тесты с httptest
```

**`verifier.go`:**

```go
type Verifier struct {
    clientID     string
    clientSecret string
    redirectURI  string
    httpClient   *http.Client
    log          *slog.Logger
}

// NewVerifier creates a Google OAuth verifier.
// Parameters come from config.AuthConfig: GoogleClientID, GoogleClientSecret, GoogleRedirectURI.
func NewVerifier(clientID, clientSecret, redirectURI string, logger *slog.Logger) *Verifier {
    return &Verifier{
        clientID:     clientID,
        clientSecret: clientSecret,
        redirectURI:  redirectURI,
        httpClient:   &http.Client{Timeout: 10 * time.Second},
        log:          logger.With("adapter", "google_oauth"),
    }
}
```

**Метод:**

```go
func (v *Verifier) VerifyCode(ctx context.Context, provider, code string) (*auth.OAuthIdentity, error)
```

**Flow (2 HTTP-запроса):**

**Шаг 1 — Token Exchange:**
```
POST https://oauth2.googleapis.com/token
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code
code=<auth_code>
client_id=<client_id>
client_secret=<client_secret>
redirect_uri=<redirect_uri>

→ 200: { "access_token": "...", "id_token": "...", "token_type": "Bearer", ... }
→ 400: { "error": "invalid_grant", "error_description": "..." }
```

**Шаг 2 — Userinfo:**
```
GET https://www.googleapis.com/oauth2/v2/userinfo
Authorization: Bearer <access_token>

→ 200: { "id": "...", "email": "...", "name": "...", "picture": "...", "verified_email": true }
```

**Маппинг в OAuthIdentity:**

| Google field | OAuthIdentity field | Примечание |
|-------------|-------------------|-----------|
| `id` | `ProviderID` | Уникальный Google ID пользователя |
| `email` | `Email` | Email (всегда есть) |
| `name` | `Name` | Может отсутствовать → `nil` |
| `picture` | `AvatarURL` | Может отсутствовать → `nil` |

**Retry-логика:**

```go
func (v *Verifier) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
    resp, err := v.httpClient.Do(req.WithContext(ctx))
    if err != nil || resp.StatusCode >= 500 {
        // Retry один раз с backoff 500ms
        time.Sleep(500 * time.Millisecond)
        resp, err = v.httpClient.Do(req.WithContext(ctx))
    }
    return resp, err
}
```

Retry **только** при:
- Network error (timeout, connection refused)
- HTTP 5xx от Google

**Не** retry при:
- HTTP 400 (invalid code, expired code) — ошибка клиента
- HTTP 401/403 — невалидные credentials

**Error handling:**

| Ситуация | Действие |
|----------|----------|
| Token exchange → 400 (invalid_grant) | Вернуть ошибку: "oauth: invalid or expired code" |
| Token exchange → 5xx | Retry 1 раз, затем вернуть ошибку: "oauth: google unavailable" |
| Token exchange → timeout | Retry 1 раз, затем вернуть ошибку |
| Userinfo → non-200 | Вернуть ошибку: "oauth: failed to fetch user info" |
| Userinfo → JSON decode error | Вернуть ошибку |
| Google возвращает `verified_email=false` | Вернуть ошибку: "oauth: email not verified" |

**Логирование:**
- ERROR: "google oauth token exchange failed" с status code (без code/token в логах!)
- ERROR: "google oauth userinfo failed" с status code
- DEBUG: "google oauth success" с email

**Acceptance criteria:**
- [ ] `Verifier` struct создан с конструктором
- [ ] `VerifyCode` выполняет token exchange + userinfo
- [ ] HTTP timeout: 10 секунд
- [ ] Retry: 1 раз при 5xx/network error, backoff 500ms
- [ ] Не retry при 4xx
- [ ] Маппинг Google fields → `auth.OAuthIdentity` корректен
- [ ] `Name` и `AvatarURL` → `nil` если отсутствуют
- [ ] `verified_email=false` → ошибка
- [ ] Секреты (code, access_token) **не** логируются
- [ ] Unit-тесты с `httptest.NewServer`:
  - [ ] Success: token exchange + userinfo → correct OAuthIdentity
  - [ ] Invalid code (400 от Google)
  - [ ] Google 5xx → retry → success (на втором запросе)
  - [ ] Google 5xx → retry → 5xx (обе попытки fail)
  - [ ] Timeout
  - [ ] Unverified email
  - [ ] Name/picture отсутствуют → nil fields
- [ ] `go build ./...` компилируется

**Corner cases:**
- Google может не вернуть `name` (редко, но возможно — приватный профиль) → `OAuthIdentity.Name = nil`
- `redirect_uri` должен точно совпадать с тем, что клиент передал Google — расхождение → 400 от Google
- Authorization code одноразовый (OAuth 2.0 стандарт) — повторный вызов с тем же code → 400 от Google
- Retry при timeout: context может быть уже cancelled → проверять `ctx.Err()` перед retry

---

### TASK-4.4: Auth Service

**Зависит от:** TASK-4.1 (AllowedProviders), TASK-4.2 (JWT Manager + OAuthIdentity)

> **Примечание:** TASK-4.4 **не зависит** от TASK-4.3 (Google OAuth adapter). Auth service определяет свой приватный `oauthVerifier` interface и использует тип `auth.OAuthIdentity` из TASK-4.2. В тестах oauthVerifier мокается. Реальный Google adapter (TASK-4.3) нужен только при wiring в main.go.

**Контекст:**
- `services/auth_service_spec_v4.md` — все секции: §2 (зависимости), §3 (types), §4 (операции), §5 (валидация), §6 (errors), §7 (security), §8 (тесты)
- `services/service_layer_spec_v4.md` — §3 (паттерны: интерфейсы, input, транзакции, error handling, логирование), §7 (тестирование: моки)
- `code_conventions_v4.md` — §2 (ошибки: оборачивание, sentinel), §3 (валидация), §7 (моки через moq)

**Что сделать:**

Создать пакет `internal/service/auth/` с полной реализацией Auth Service.

**Файловая структура:**

```
internal/service/auth/
├── service.go        # Service struct, конструктор, приватные интерфейсы, все операции
├── input.go          # LoginInput, RefreshInput + Validate()
├── result.go         # AuthResult
└── service_test.go   # ~30 unit-тестов
```

**`service.go` — приватные интерфейсы:**

```go
type userRepo interface {
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    GetByOAuth(ctx context.Context, provider domain.OAuthProvider, oauthID string) (*domain.User, error)
    Create(ctx context.Context, user *domain.User) (*domain.User, error)
    Update(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error)
}

type settingsRepo interface {
    CreateSettings(ctx context.Context, settings *domain.UserSettings) error
}

type tokenRepo interface {
    Create(ctx context.Context, token *domain.RefreshToken) error
    GetByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
    RevokeByID(ctx context.Context, id uuid.UUID) error
    RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
    DeleteExpired(ctx context.Context) (int, error)
}

type txManager interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type oauthVerifier interface {
    VerifyCode(ctx context.Context, provider, code string) (*auth.OAuthIdentity, error)
}

type jwtManager interface {
    GenerateAccessToken(userID uuid.UUID) (string, error)
    ValidateAccessToken(token string) (uuid.UUID, error)
    GenerateRefreshToken() (raw string, hash string, err error)
}
```

**Конструктор:**

```go
type Service struct {
    log      *slog.Logger
    users    userRepo
    settings settingsRepo
    tokens   tokenRepo
    tx       txManager
    oauth    oauthVerifier
    jwt      jwtManager
    cfg      config.AuthConfig
}

func NewService(
    logger   *slog.Logger,
    users    userRepo,
    settings settingsRepo,
    tokens   tokenRepo,
    tx       txManager,
    oauth    oauthVerifier,
    jwt      jwtManager,
    cfg      config.AuthConfig,
) *Service {
    return &Service{
        log:      logger.With("service", "auth"),
        users:    users,
        settings: settings,
        tokens:   tokens,
        tx:       tx,
        oauth:    oauth,
        jwt:      jwt,
        cfg:      cfg,
    }
}
```

**`result.go` — AuthResult:**

```go
// AuthResult is returned by Login and Refresh operations.
type AuthResult struct {
    AccessToken  string
    RefreshToken string       // raw token, NOT hash
    User         *domain.User
}
```

---

**Операция: Login(ctx, input LoginInput) → (\*AuthResult, error)**

Полный flow из `auth_service_spec_v4.md` §4.1:

```
1. input.Validate(s.cfg.AllowedProviders())
   └─ ошибка → return ValidationError

2. s.oauth.VerifyCode(ctx, input.Provider, input.Code)
   └─ ошибка → log ERROR "oauth verification failed", return error
   └─ успех → identity (OAuthIdentity)

3. s.users.GetByOAuth(ctx, OAuthProvider(input.Provider), identity.ProviderID)
   ├─ найден → existingUser
   │   ├─ 3a. Сравнить name/avatar: изменились?
   │   │   └─ если да → s.users.Update(ctx, existingUser.ID, identity.Name, identity.AvatarURL)
   │   └─ 3b. user = existingUser (или обновлённый)
   │
   └─ ErrNotFound → новый пользователь
       └─ 4. s.tx.RunInTx:
           ├─ 4a. s.users.Create(ctx, &domain.User{
           │       Email:         identity.Email,
           │       Name:          derefOrEmpty(identity.Name),
           │       AvatarURL:     identity.AvatarURL,
           │       OAuthProvider: domain.OAuthProvider(input.Provider),
           │       OAuthID:       identity.ProviderID,
           │   })
           │   └─ ErrAlreadyExists → return ErrAlreadyExists (tx откатится)
           │
           └─ 4b. s.settings.CreateSettings(ctx, domain.DefaultUserSettings(user.ID))

   4-fallback. Если шаг 4 вернул ErrAlreadyExists:
       └─ s.users.GetByOAuth(ctx, provider, providerID)
           ├─ найден → user (race condition, продолжаем как login)
           └─ ErrNotFound → email collision → return ErrAlreadyExists

5. accessToken, err = s.jwt.GenerateAccessToken(user.ID)
6. rawRefresh, hashRefresh, err = s.jwt.GenerateRefreshToken()
7. s.tokens.Create(ctx, &domain.RefreshToken{
       UserID:    user.ID,
       TokenHash: hashRefresh,
       ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
   })

8. Log INFO:
   ├─ новый: "user registered" user_id=... provider=...
   └─ существующий: "user logged in" user_id=... provider=...

9. return &AuthResult{AccessToken: accessToken, RefreshToken: rawRefresh, User: user}
```

---

**Операция: Refresh(ctx, input RefreshInput) → (\*AuthResult, error)**

Flow из `auth_service_spec_v4.md` §4.2:

```
1. input.Validate()
   └─ ошибка → return ValidationError

2. hash = auth.HashToken(input.RefreshToken)

3. token, err = s.tokens.GetByHash(ctx, hash)
   └─ ErrNotFound → return domain.ErrUnauthorized
      (token не существует или уже revoked — repo фильтрует revoked)

4. if token.IsExpired(time.Now()) → return domain.ErrUnauthorized

5. user, err = s.users.GetByID(ctx, token.UserID)
   └─ ErrNotFound → log WARN "refresh for deleted user", return domain.ErrUnauthorized

6. s.tokens.RevokeByID(ctx, token.ID)

7. newAccessToken = s.jwt.GenerateAccessToken(user.ID)
8. newRaw, newHash = s.jwt.GenerateRefreshToken()
9. s.tokens.Create(ctx, &domain.RefreshToken{
       UserID:    user.ID,
       TokenHash: newHash,
       ExpiresAt: time.Now().Add(s.cfg.RefreshTokenTTL),
   })

10. return &AuthResult{AccessToken: newAccessToken, RefreshToken: newRaw, User: user}
```

**Важно:** Шаги 6–9 **не** в транзакции. Если crash между Revoke и Create — старый revoked, новый не создан → пользователь перенаправляется на login. Это безопаснее, чем транзакция (которая могла бы оставить невалидный старый token при откате).

---

**Операция: Logout(ctx) → error**

Flow из `auth_service_spec_v4.md` §4.3:

```
1. userID, ok = ctxutil.UserIDFromCtx(ctx)
   └─ !ok → return domain.ErrUnauthorized

2. s.tokens.RevokeAllByUser(ctx, userID)

3. s.log.InfoContext(ctx, "user logged out", slog.String("user_id", userID.String()))

4. return nil
```

---

**Операция: ValidateToken(ctx, token string) → (uuid.UUID, error)**

Flow из `auth_service_spec_v4.md` §4.4:

```
1. userID, err = s.jwt.ValidateAccessToken(token)
   └─ любая ошибка → return uuid.Nil, domain.ErrUnauthorized

2. return userID, nil
```

**Свойства:**
- **Без обращения к БД** — вызывается при каждом запросе, JWT stateless
- **Не проверяет существование пользователя** — обрабатывается сервисами при обращении к данным
- **Не проверяет revocation** — JWT access token невозможно отозвать до истечения TTL (trade-off)

---

**Операция: CleanupExpiredTokens(ctx) → (int, error)**

Flow из `auth_service_spec_v4.md` §4.5:

```
1. count, err = s.tokens.DeleteExpired(ctx)
   └─ ошибка → log ERROR "token cleanup failed", return 0, err

2. if count > 0 → log INFO "cleaned up expired tokens" count=...

3. return count, nil
```

Свойства: не требует userID, вызывается scheduler-ом.

---

**`input.go` — валидация:**

**LoginInput:**

```go
type LoginInput struct {
    Provider string
    Code     string
}

func (i LoginInput) Validate(allowedProviders []string) error {
    var errs []domain.FieldError

    if i.Provider == "" {
        errs = append(errs, domain.FieldError{Field: "provider", Message: "required"})
    } else if !contains(allowedProviders, i.Provider) {
        errs = append(errs, domain.FieldError{Field: "provider", Message: "unsupported provider"})
    }

    if i.Code == "" {
        errs = append(errs, domain.FieldError{Field: "code", Message: "required"})
    } else if len(i.Code) > 4096 {
        errs = append(errs, domain.FieldError{Field: "code", Message: "too long"})
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

`allowedProviders` передаётся из `s.cfg.AllowedProviders()` при вызове в сервисе.

**RefreshInput:**

```go
type RefreshInput struct {
    RefreshToken string
}

func (i RefreshInput) Validate() error {
    var errs []domain.FieldError

    if i.RefreshToken == "" {
        errs = append(errs, domain.FieldError{Field: "refresh_token", Message: "required"})
    } else if len(i.RefreshToken) > 512 {
        errs = append(errs, domain.FieldError{Field: "refresh_token", Message: "too long"})
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

---

**Corner cases (из auth_service_spec §4, все обработки описаны в flow):**

1. **Race condition при регистрации:** Два параллельных запроса с одним OAuth аккаунтом. Оба получают ErrNotFound от GetByOAuth, оба пытаются CreateUser. Первый успешно. Второй получает ErrAlreadyExists от CreateUser. Обработка: при ErrAlreadyExists из RunInTx — повторить GetByOAuth, продолжить как login

2. **Email collision:** GetByOAuth → ErrNotFound, CreateUser → ErrAlreadyExists (ux_users_email, не ux_users_oauth), повторный GetByOAuth → ErrNotFound (другой provider). Сервис возвращает ErrAlreadyExists — "account with this email already exists". MVP: не merge

3. **Обновление профиля при login:** Если identity.Name != existingUser.Name или identity.AvatarURL != existingUser.AvatarURL → вызвать Update. Сравнение pointer-значений с nil-safety:
   ```go
   func profileChanged(user *domain.User, identity *auth.OAuthIdentity) bool {
       if identity.Name != nil && *identity.Name != user.Name {
           return true
       }
       if identity.AvatarURL != nil && ptrStringNotEqual(identity.AvatarURL, user.AvatarURL) {
           return true
       }
       return false
   }
   ```

   **Helper-функции** (приватные, в `service.go`):
   ```go
   // derefOrEmpty returns the dereferenced value or empty string if nil.
   // Used because domain.User.Name is string, but OAuthIdentity.Name is *string.
   func derefOrEmpty(s *string) string {
       if s == nil {
           return ""
       }
       return *s
   }

   // ptrStringNotEqual compares *string with *string, treating nil as distinct from "".
   func ptrStringNotEqual(a, b *string) bool {
       if a == nil && b == nil {
           return false
       }
       if a == nil || b == nil {
           return true
       }
       return *a != *b
   }
   ```

4. **Token rotation — replay detection:** Revoked token → GetByHash возвращает ErrNotFound (repo фильтрует revoked). Невозможно отличить "token не существует" от "token reused" на уровне сервиса. WARN лог "refresh token reuse attempted" записывается при любом ErrNotFound от GetByHash (как требует auth_service_spec §4.2). False positive допустим — лучше перестраховаться

5. **Concurrent refresh:** Два tab/устройства отправляют refresh одновременно с одним токеном. Первый проходит. Второй → ErrUnauthorized. Ожидаемое поведение token rotation

6. **Access token после logout:** Logout отзывает только refresh tokens. Access token (JWT) остаётся валидным до истечения TTL (15 минут). Архитектурный trade-off: JWT stateless

---

**Unit-тесты (из auth_service_spec §8.2):**

Все зависимости мокаются через moq. Mock TxManager: `fn(ctx)` без реальной транзакции.

**Login — happy path:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| L1 | Первый вход — регистрация | GetByOAuth → ErrNotFound, Create → ok, CreateSettings → ok | AuthResult с новым user. Create вызван. CreateSettings вызван. LOG "user registered" |
| L2 | Повторный вход | GetByOAuth → existingUser | AuthResult с existing user. Create НЕ вызван |
| L3 | Повторный вход — профиль изменился | GetByOAuth → user(Name="Old"), identity.Name="New" | Update вызван с Name="New". AuthResult с updated user |
| L4 | Повторный вход — профиль не изменился | GetByOAuth → user(Name="Same"), identity.Name="Same" | Update НЕ вызван |

**Login — validation:**

| # | Тест | Input | Assert |
|---|------|-------|--------|
| L5 | Пустой provider | Provider="", Code="abc" | ValidationError, field="provider", message="required" |
| L6 | Неподдерживаемый provider | Provider="facebook" | ValidationError, field="provider", message="unsupported provider" |
| L7 | Пустой code | Provider="google", Code="" | ValidationError, field="code", message="required" |

**Login — error scenarios:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| L8 | OAuth verification failed | VerifyCode → error | Ошибка проброшена (не wrapped в domain error) |
| L9 | Race condition | GetByOAuth → ErrNotFound, Create → ErrAlreadyExists, повторный GetByOAuth → user | AuthResult с existing user. Без паники |
| L10 | Email collision | GetByOAuth → ErrNotFound, Create → ErrAlreadyExists, повторный GetByOAuth → ErrNotFound | ErrAlreadyExists |

**Login — tokens:**

| # | Тест | Assert |
|---|------|--------|
| L11 | Access token генерируется | GenerateAccessToken вызван с correct userID |
| L12 | Refresh token сохраняется | tokens.Create вызван: TokenHash == hash (не raw), UserID correct, ExpiresAt ≈ now + RefreshTokenTTL |
| L13 | Raw refresh token в ответе | AuthResult.RefreshToken == raw (не hash) |

**Refresh — happy path:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| R1 | Успешный refresh | GetByHash → valid token, GetByID → user | Новая JWT-пара. Старый token revoked. Новый создан |
| R2 | User data в ответе | GetByID → user(Name="John") | AuthResult.User.Name == "John" |

**Refresh — error scenarios:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| R3 | Token не найден | GetByHash → ErrNotFound | ErrUnauthorized |
| R4 | Token expired | GetByHash → token(ExpiresAt=time.Now().Add(-1h)) | ErrUnauthorized |
| R5 | User deleted | GetByHash → ok, GetByID → ErrNotFound | ErrUnauthorized |
| R6 | Пустой input | RefreshToken="" | ValidationError |

**Refresh — rotation:**

| # | Тест | Assert |
|---|------|--------|
| R7 | Старый token revoked | RevokeByID вызван с old token.ID |
| R8 | Новый token создан | tokens.Create вызван с new hash ≠ old hash |
| R9 | Новый token в ответе | AuthResult.RefreshToken ≠ input.RefreshToken |

**Logout:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| LO1 | Успешный logout | userID в ctx | RevokeAllByUser вызван с correct userID |
| LO2 | Нет userID в ctx | ctx без userID | ErrUnauthorized |
| LO3 | Нет active tokens | RevokeAllByUser → nil (0 affected) | Нет ошибки |

**ValidateToken:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| V1 | Валидный token | ValidateAccessToken → userID | Correct userID returned |
| V2 | Expired token | ValidateAccessToken → error | ErrUnauthorized |
| V3 | Invalid signature | ValidateAccessToken → error | ErrUnauthorized |
| V4 | Malformed token | ValidateAccessToken → error | ErrUnauthorized |

**CleanupExpiredTokens:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| CL1 | Есть expired tokens | DeleteExpired → 42 | Returns 42 |
| CL2 | Нет expired tokens | DeleteExpired → 0 | Returns 0 |
| CL3 | DB error | DeleteExpired → error | Error проброшен |

**Всего: ~30 тест-кейсов**

**Acceptance criteria:**
- [ ] Service struct с приватными интерфейсами зависимостей
- [ ] Конструктор `NewService` принимает все зависимости, устанавливает логгер с `"service", "auth"`
- [ ] **Login:** полный flow с registration и login ветками
- [ ] **Login:** Registration в транзакции (CreateUser + CreateSettings)
- [ ] **Login:** race condition обрабатывается (retry GetByOAuth после ErrAlreadyExists от CreateUser)
- [ ] **Login:** email collision → ErrAlreadyExists (retry GetByOAuth возвращает ErrNotFound)
- [ ] **Login:** профиль обновляется при изменении name/avatar
- [ ] **Login:** профиль НЕ обновляется если не изменился
- [ ] **Refresh:** token rotation — revoke old + create new
- [ ] **Refresh:** expired token → ErrUnauthorized
- [ ] **Refresh:** deleted user → ErrUnauthorized
- [ ] **Refresh:** шаги 6-9 НЕ в транзакции
- [ ] **Logout:** revoke all tokens по userID из context
- [ ] **Logout:** без userID → ErrUnauthorized
- [ ] **ValidateToken:** stateless JWT validation, без БД
- [ ] **ValidateToken:** любая JWT-ошибка → domain.ErrUnauthorized
- [ ] **CleanupExpiredTokens:** делегирует в tokenRepo, не требует userID
- [ ] `LoginInput.Validate(allowedProviders)` — все поля, unsupported provider
- [ ] `RefreshInput.Validate()` — пустой и слишком длинный token
- [ ] Логирование: INFO для login/register/logout, WARN для deleted user refresh, ERROR для provider errors
- [ ] ~30 unit-тестов покрывают все сценарии из auth_service_spec §8
- [ ] Моки сгенерированы через `moq` из приватных интерфейсов
- [ ] Mock TxManager: `fn(ctx)` pass-through
- [ ] `go test ./internal/service/auth/...` — все проходят
- [ ] `go vet ./internal/service/auth/...` — без warnings

---

### TASK-4.5: User Service

**Зависит от:** TASK-4.6 (EntityTypeUser для аудита)

**Контекст:**
- `services/service_layer_spec_v4.md` — §3 (паттерны), §4 (аудит: USER settings → update), §6.1 (UserService: профиль, настройки), §7 (тестирование)
- `code_conventions_v4.md` — §3 (валидация), §6 (аудит в транзакции), §7 (тестирование, moq)
- `data_model_v4.md` — §3 (users, user_settings)

**Что сделать:**

Создать пакет `internal/service/user/` с User Service.

**Файловая структура:**

```
internal/service/user/
├── service.go        # Service struct, конструктор, приватные интерфейсы, методы
├── input.go          # UpdateProfileInput, UpdateSettingsInput + Validate()
└── service_test.go   # ~15 unit-тестов
```

**`service.go` — приватные интерфейсы:**

```go
type userRepo interface {
    GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
    Update(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error)
}

type settingsRepo interface {
    GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error)
    UpdateSettings(ctx context.Context, userID uuid.UUID, settings *domain.UserSettings) (*domain.UserSettings, error)
}

type auditRepo interface {
    Create(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
    RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
```

**Конструктор:**

```go
type Service struct {
    log      *slog.Logger
    users    userRepo
    settings settingsRepo
    audit    auditRepo
    tx       txManager
}

func NewService(
    logger   *slog.Logger,
    users    userRepo,
    settings settingsRepo,
    audit    auditRepo,
    tx       txManager,
) *Service {
    return &Service{
        log:      logger.With("service", "user"),
        users:    users,
        settings: settings,
        audit:    audit,
        tx:       tx,
    }
}
```

---

**Операция: GetProfile(ctx) → (\*domain.User, error)**

```
1. userID, ok = ctxutil.UserIDFromCtx(ctx)
   └─ !ok → return nil, domain.ErrUnauthorized

2. user, err = s.users.GetByID(ctx, userID)
   └─ err → return nil, fmt.Errorf("user.GetProfile: %w", err)

3. return user, nil
```

---

**Операция: UpdateProfile(ctx, input UpdateProfileInput) → (\*domain.User, error)**

```
1. userID, ok = ctxutil.UserIDFromCtx(ctx)
   └─ !ok → return nil, domain.ErrUnauthorized

2. if err := input.Validate(); err != nil → return nil, err

3. user, err = s.users.Update(ctx, userID, input.Name, input.AvatarURL)
   └─ err → return nil, fmt.Errorf("user.UpdateProfile: %w", err)

4. s.log.InfoContext(ctx, "profile updated",
       slog.String("user_id", userID.String()))

5. return user, nil
```

Профиль (name, avatar) **не аудитируется** — данные приходят из OAuth-провайдера, изменения незначительны.

---

**Операция: GetSettings(ctx) → (\*domain.UserSettings, error)**

```
1. userID, ok = ctxutil.UserIDFromCtx(ctx)
   └─ !ok → return nil, domain.ErrUnauthorized

2. settings, err = s.settings.GetSettings(ctx, userID)
   └─ err → return nil, fmt.Errorf("user.GetSettings: %w", err)

3. return settings, nil
```

---

**Операция: UpdateSettings(ctx, input UpdateSettingsInput) → (\*domain.UserSettings, error)**

```
1. userID, ok = ctxutil.UserIDFromCtx(ctx)
   └─ !ok → return nil, domain.ErrUnauthorized

2. if err := input.Validate(); err != nil → return nil, err

3. oldSettings, err = s.settings.GetSettings(ctx, userID)
   └─ err → return nil, fmt.Errorf("user.UpdateSettings get current: %w", err)

4. merged = applySettingsChanges(oldSettings, input)

5. var newSettings *domain.UserSettings
   err = s.tx.RunInTx(ctx, func(ctx context.Context) error {
       var err error
       newSettings, err = s.settings.UpdateSettings(ctx, userID, merged)
       if err != nil {
           return err
       }
       return s.audit.Create(ctx, domain.AuditRecord{
           UserID:     userID,
           EntityType: domain.EntityTypeUser,
           Action:     domain.AuditActionUpdate,
           Changes:    buildSettingsChanges(oldSettings, newSettings),
       })
   })
   └─ err → return nil, fmt.Errorf("user.UpdateSettings: %w", err)

6. s.log.InfoContext(ctx, "settings updated",
       slog.String("user_id", userID.String()))

7. return newSettings, nil
```

**`applySettingsChanges`** — merge input поверх текущих settings:

```go
func applySettingsChanges(current *domain.UserSettings, input UpdateSettingsInput) *domain.UserSettings {
    result := *current
    if input.NewCardsPerDay != nil {
        result.NewCardsPerDay = *input.NewCardsPerDay
    }
    if input.ReviewsPerDay != nil {
        result.ReviewsPerDay = *input.ReviewsPerDay
    }
    if input.MaxIntervalDays != nil {
        result.MaxIntervalDays = *input.MaxIntervalDays
    }
    if input.Timezone != nil {
        result.Timezone = *input.Timezone
    }
    return &result
}
```

**`buildSettingsChanges`** — diff для аудита (только изменённые поля):

```go
func buildSettingsChanges(old, new *domain.UserSettings) map[string]any {
    changes := make(map[string]any)
    if old.NewCardsPerDay != new.NewCardsPerDay {
        changes["new_cards_per_day"] = map[string]any{"old": old.NewCardsPerDay, "new": new.NewCardsPerDay}
    }
    if old.ReviewsPerDay != new.ReviewsPerDay {
        changes["reviews_per_day"] = map[string]any{"old": old.ReviewsPerDay, "new": new.ReviewsPerDay}
    }
    // ... аналогично для MaxIntervalDays, Timezone
    return changes
}
```

---

**`input.go` — валидация:**

**UpdateProfileInput:**

```go
type UpdateProfileInput struct {
    Name      *string
    AvatarURL *string
}

func (i UpdateProfileInput) Validate() error {
    var errs []domain.FieldError

    if i.Name != nil {
        name := strings.TrimSpace(*i.Name)
        if name == "" {
            errs = append(errs, domain.FieldError{Field: "name", Message: "must not be empty"})
        } else if len(name) > 100 {
            errs = append(errs, domain.FieldError{Field: "name", Message: "too long (max 100)"})
        }
    }

    if i.AvatarURL != nil && *i.AvatarURL != "" {
        if _, err := url.Parse(*i.AvatarURL); err != nil {
            errs = append(errs, domain.FieldError{Field: "avatar_url", Message: "invalid URL"})
        }
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**UpdateSettingsInput:**

```go
type UpdateSettingsInput struct {
    NewCardsPerDay  *int
    ReviewsPerDay   *int
    MaxIntervalDays *int
    Timezone        *string
}

func (i UpdateSettingsInput) Validate() error {
    var errs []domain.FieldError

    if i.NewCardsPerDay != nil && (*i.NewCardsPerDay < 0 || *i.NewCardsPerDay > 100) {
        errs = append(errs, domain.FieldError{Field: "new_cards_per_day", Message: "must be 0-100"})
    }
    if i.ReviewsPerDay != nil && (*i.ReviewsPerDay < 0 || *i.ReviewsPerDay > 1000) {
        errs = append(errs, domain.FieldError{Field: "reviews_per_day", Message: "must be 0-1000"})
    }
    if i.MaxIntervalDays != nil && (*i.MaxIntervalDays < 1 || *i.MaxIntervalDays > 3650) {
        errs = append(errs, domain.FieldError{Field: "max_interval_days", Message: "must be 1-3650"})
    }
    if i.Timezone != nil {
        if _, err := time.LoadLocation(*i.Timezone); err != nil {
            errs = append(errs, domain.FieldError{Field: "timezone", Message: "invalid IANA timezone"})
        }
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

---

**Corner cases:**
- `UpdateProfile` с обоими полями nil → допустимо. Repo вызывается с nil/nil — возвращает текущий user без изменений. Альтернатива: проверить и вернуть без вызова repo (оптимизация, не критично)
- `UpdateSettings` с обоими полями nil → допустимо. Audit changes будет пустой map — audit record всё равно создаётся (действие зафиксировано). Или: если changes пуст → не создавать audit record и не вызывать UpdateSettings
- Timezone validation: `time.LoadLocation("UTC")` → ok. `time.LoadLocation("")` → ok (возвращает UTC). `time.LoadLocation("Invalid/Zone")` → error
- `UpdateSettings` — audit changes включают **только** изменённые поля. Если input.NewCardsPerDay == old.NewCardsPerDay → не включать в changes
- `GetProfile`/`GetSettings` для несуществующего пользователя → repo вернёт `ErrNotFound`. Сервис прокидывает как есть

---

**Unit-тесты:**

| # | Тест | Операция | Assert |
|---|------|----------|--------|
| U1 | GetProfile success | GetProfile | Correct user returned |
| U2 | GetProfile — no auth | GetProfile без userID в ctx | ErrUnauthorized |
| U3 | GetProfile — not found | GetByID → ErrNotFound | ErrNotFound проброшен |
| U4 | UpdateProfile success | UpdateProfile(Name="New") | Update called with correct params, correct result |
| U5 | UpdateProfile — name validation | UpdateProfile(Name="") | ValidationError, field="name" |
| U6 | UpdateProfile — name too long | UpdateProfile(Name=string(101 chars)) | ValidationError, field="name" |
| U7 | UpdateProfile — no auth | UpdateProfile без userID | ErrUnauthorized |
| U8 | GetSettings success | GetSettings | Correct settings returned |
| U9 | GetSettings — no auth | GetSettings без userID | ErrUnauthorized |
| U10 | UpdateSettings success | UpdateSettings(NewCardsPerDay=30) | UpdateSettings called, audit created |
| U11 | UpdateSettings — cards out of range | UpdateSettings(NewCardsPerDay=-1) | ValidationError |
| U12 | UpdateSettings — reviews out of range | UpdateSettings(ReviewsPerDay=1001) | ValidationError |
| U13 | UpdateSettings — interval out of range | UpdateSettings(MaxIntervalDays=0) | ValidationError |
| U14 | UpdateSettings — timezone invalid | UpdateSettings(Timezone="Invalid/Zone") | ValidationError |
| U15 | UpdateSettings — timezone valid | UpdateSettings(Timezone="Europe/Moscow") | Success, settings updated |
| U16 | UpdateSettings — no auth | UpdateSettings без userID | ErrUnauthorized |
| U17 | UpdateSettings — audit diff | UpdateSettings(ReviewsPerDay: 100→200) | AuditRecord.Changes содержит `{"reviews_per_day": {"old": 100, "new": 200}}` |
| U18 | UpdateSettings — partial update | UpdateSettings(только Timezone) | Остальные поля не изменены в merged settings |
| U19 | UpdateSettings — audit в транзакции | UpdateSettings | RunInTx вызван, audit.Create вызван внутри |

**Всего: ~19 тест-кейсов**

**Acceptance criteria:**
- [ ] Service struct с приватными интерфейсами (userRepo, settingsRepo, auditRepo, txManager)
- [ ] Конструктор `NewService` с логгером `"service", "user"`
- [ ] `GetProfile`: возвращает user по userID из context
- [ ] `UpdateProfile`: обновляет name и/или avatar через repo
- [ ] `GetSettings`: возвращает settings по userID
- [ ] `UpdateSettings`: partial merge + update + audit в транзакции
- [ ] `applySettingsChanges`: корректный merge nullable полей
- [ ] `buildSettingsChanges`: diff включает только изменённые поля
- [ ] Все четыре операции: ErrUnauthorized при отсутствии userID в ctx
- [ ] `UpdateProfileInput.Validate()`: name не пустой (1-100), URL валидный
- [ ] `UpdateSettingsInput.Validate()`: ranges — cards 0-100, reviews 0-1000, interval 1-3650
- [ ] `UpdateSettingsInput.Validate()`: timezone через `time.LoadLocation`
- [ ] Audit record для UpdateSettings: `EntityTypeUser`, `AuditActionUpdate`, changes diff
- [ ] Audit пишется в транзакции с UpdateSettings (через RunInTx)
- [ ] Логирование: INFO для UpdateProfile и UpdateSettings
- [ ] ~19 unit-тестов
- [ ] Моки через `moq`
- [ ] `go test ./internal/service/user/...` — все проходят

---

### TASK-4.6: Добавление EntityTypeUser в domain

**Зависит от:** ничего (параллельно с остальными задачами)

**Контекст:**
- `services/service_layer_spec_v4.md` — §4 (аудит: USER settings → update)
- `data_model_v4.md` — §7 (audit_log, entity_type enum)
- Текущий `internal/domain/enums.go` — EntityType без USER

**Что сделать:**

Добавить значение `USER` в `domain.EntityType` enum.

**Изменения в `internal/domain/enums.go`:**

```go
const (
    EntityTypeEntry         EntityType = "ENTRY"
    EntityTypeSense         EntityType = "SENSE"
    EntityTypeExample       EntityType = "EXAMPLE"
    EntityTypeImage         EntityType = "IMAGE"
    EntityTypePronunciation EntityType = "PRONUNCIATION"
    EntityTypeCard          EntityType = "CARD"
    EntityTypeTopic         EntityType = "TOPIC"
    EntityTypeUser          EntityType = "USER"   // NEW
)
```

Обновить `IsValid()`:

```go
func (e EntityType) IsValid() bool {
    switch e {
    case EntityTypeEntry, EntityTypeSense, EntityTypeExample, EntityTypeImage,
        EntityTypePronunciation, EntityTypeCard, EntityTypeTopic, EntityTypeUser:
        return true
    }
    return false
}
```

**Миграция (если Фаза 2 завершена):**

Создать дополнительную миграцию `00009_add_user_entity_type.sql`:

```sql
-- +goose Up
ALTER TYPE entity_type ADD VALUE IF NOT EXISTS 'USER';

-- +goose Down
-- ALTER TYPE ... DROP VALUE не поддерживается в PostgreSQL.
-- Значение останется в enum после downgrade. Это задокументированное
-- ограничение PostgreSQL: нельзя удалить значение из enum.
```

Если Фаза 2 ещё не завершена — добавить `USER` в начальную миграцию `00001_enums.sql`.

**Acceptance criteria:**
- [ ] `EntityTypeUser = "USER"` добавлен в enum
- [ ] `IsValid()` обновлён — включает `EntityTypeUser`
- [ ] Существующие тесты обновлены и проходят
- [ ] Миграция подготовлена (или `00001_enums.sql` обновлён)
- [ ] `go build ./...` компилируется

---

### TASK-4.7: Auth Middleware

**Зависит от:** TASK-4.2 (JWT types, для понимания ошибок)

**Контекст:**
- `code_conventions_v4.md` — §9.4 (Auth Middleware: tokenValidator interface, flow)
- `infra_spec_v4.md` — §4 (middleware-стек: порядок)
- `services/auth_service_spec_v4.md` — §10.2 (Auth Middleware: ValidateToken)

**Что сделать:**

Создать пакет `internal/transport/middleware/` с auth middleware и request ID middleware.

**Файловая структура:**

```
internal/transport/middleware/
├── auth.go           # Auth middleware
├── auth_test.go      # Unit-тесты auth middleware
├── request_id.go     # Request ID middleware
└── request_id_test.go
```

**`auth.go` — Auth Middleware:**

Middleware определяет **свой** узкий интерфейс (consumer-defined):

```go
// tokenValidator defines what auth middleware needs from the auth service.
type tokenValidator interface {
    ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

// Auth returns middleware that validates JWT access tokens.
func Auth(validator tokenValidator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token := extractBearerToken(r)
            if token == "" {
                // Нет токена — пропускаем (anonymous request).
                // Protected endpoints проверяют наличие userID сами.
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

// extractBearerToken extracts token from "Authorization: Bearer <token>" header.
func extractBearerToken(r *http.Request) string {
    auth := r.Header.Get("Authorization")
    if !strings.HasPrefix(auth, "Bearer ") {
        return ""
    }
    return strings.TrimPrefix(auth, "Bearer ")
}
```

**Поведение:**
- Нет `Authorization` header → пропустить запрос (anonymous). Не возвращать 401
- `Authorization: Bearer <token>` → validate → success → `ctxutil.WithUserID(ctx, userID)`
- `Authorization: Bearer <token>` → validate → error → 401 Unauthorized
- `Authorization: NotBearer ...` → пропустить как anonymous (не Bearer scheme)

**`request_id.go` — Request ID Middleware:**

```go
// RequestID injects a unique request ID into the context and response header.
func RequestID(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        id := r.Header.Get("X-Request-Id")
        if id == "" {
            id = uuid.New().String()
        }

        ctx := ctxutil.WithRequestID(r.Context(), id)
        w.Header().Set("X-Request-Id", id)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

**Поведение:**
- Входящий `X-Request-Id` header → переиспользовать
- Нет header → сгенерировать UUID
- Установить в context через `ctxutil.WithRequestID`
- Добавить в response header

**Corner cases:**
- Пустой Bearer token (`Authorization: Bearer `) → `extractBearerToken` вернёт `""` → пропустить как anonymous
- Несколько пробелов после "Bearer" → `strings.TrimPrefix` корректно обработает
- Token с пробелами → передаётся в ValidateToken как есть (JWT не содержит пробелов → ошибка валидации)
- Request ID: клиент может отправить произвольный ID. Для безопасности можно валидировать длину (max 128 chars), но на MVP — принимать как есть

**Unit-тесты (auth middleware):**

| # | Тест | Request | Assert |
|---|------|---------|--------|
| M1 | Валидный token | `Authorization: Bearer valid-token` | userID в ctx, next called, status 200 |
| M2 | Невалидный token | `Authorization: Bearer invalid` | 401, next NOT called |
| M3 | Нет header | Без Authorization | next called (anonymous), userID НЕ в ctx |
| M4 | Не Bearer scheme | `Authorization: Basic abc123` | next called (anonymous), userID НЕ в ctx |
| M5 | Пустой Bearer | `Authorization: Bearer ` | next called (anonymous) |

**Unit-тесты (request ID middleware):**

| # | Тест | Request | Assert |
|---|------|---------|--------|
| RI1 | С X-Request-Id | `X-Request-Id: abc-123` | ctx содержит "abc-123", response header = "abc-123" |
| RI2 | Без X-Request-Id | Нет header | ctx содержит сгенерированный UUID, response header установлен |

**Всего: ~7 тест-кейсов**

**Acceptance criteria:**
- [ ] `tokenValidator` interface определён в middleware пакете (consumer-defined)
- [ ] Auth middleware извлекает Bearer token из Authorization header
- [ ] Нет token → anonymous request (пропустить, не 401)
- [ ] Валидный token → `ctxutil.WithUserID` в context
- [ ] Невалидный token → 401 Unauthorized
- [ ] `extractBearerToken` корректно парсит header
- [ ] Request ID middleware: переиспользует входящий или генерирует UUID
- [ ] Request ID: сохраняет в context через `ctxutil.WithRequestID`
- [ ] Request ID: устанавливает `X-Request-Id` response header
- [ ] ~7 unit-тестов с `httptest.NewRecorder`
- [ ] Middleware реализованы как `func(http.Handler) http.Handler`
- [ ] `go test ./internal/transport/middleware/...` — все проходят

---

## Сводка зависимостей задач

```
TASK-4.1 (Config) ──────────────────┐
                                     ├──→ TASK-4.4 (Auth Service)
TASK-4.2 (JWT + OAuthIdentity) ──┬──┘
                                  │
                                  ├──→ TASK-4.3 (OAuth Google)
                                  │
                                  └──→ TASK-4.7 (Auth Middleware)

TASK-4.6 (EntityType USER) ──→ TASK-4.5 (User Service)
```

Детализация:
- **TASK-4.4** (Auth Service) зависит от: TASK-4.1 (AllowedProviders) и TASK-4.2 (jwt/identity types). **Не зависит** от TASK-4.3 — oauthVerifier мокается в тестах
- **TASK-4.3** (OAuth Google) зависит от: TASK-4.2 (OAuthIdentity тип)
- **TASK-4.7** (Auth Middleware) зависит от: TASK-4.2 — слабая зависимость, middleware определяет свой interface
- **TASK-4.5** (User Service) зависит от: TASK-4.6 (EntityTypeUser для audit)
- TASK-4.1, TASK-4.2, TASK-4.6 не имеют взаимных зависимостей

---

## Параллелизация

| Волна | Задачи (параллельно) |
|-------|---------------------|
| 1 | TASK-4.1 (Config), TASK-4.2 (JWT + OAuthIdentity), TASK-4.6 (EntityType USER) |
| 2 | TASK-4.3 (OAuth Google), TASK-4.4 (Auth Service), TASK-4.5 (User Service), TASK-4.7 (Auth Middleware) |

> При полной параллелизации — **2 sequential волны**. Волна 2 самая широкая: до 4 задач параллельно.
> TASK-4.4 не зависит от TASK-4.3 (oauthVerifier мокается) и может начинаться сразу после TASK-4.1 и TASK-4.2.

---

## Чеклист завершения фазы

- [ ] `go build ./...` компилируется без ошибок
- [ ] `go test ./...` — все тесты проходят
- [ ] `go vet ./...` — без warnings
- [ ] `golangci-lint run` — без ошибок
- [ ] **Config:** `JWTIssuer` добавлен, `AllowedProviders()` и `IsProviderAllowed()` работают
- [ ] **JWT Manager:** генерирует и валидирует access tokens (HS256, claims: sub/exp/iat/iss)
- [ ] **JWT Manager:** генерирует crypto-random refresh tokens с SHA-256 hash
- [ ] **OAuthIdentity:** тип определён в `internal/auth/identity.go`
- [ ] **Google OAuth Verifier:** token exchange + userinfo → OAuthIdentity
- [ ] **Google OAuth Verifier:** retry при 5xx, timeout 10s
- [ ] **Auth Service** — все 5 операций реализованы:
  - [ ] Login: registration и login ветки, race condition, email collision
  - [ ] Refresh: token rotation (revoke old + issue new)
  - [ ] Logout: revoke all tokens
  - [ ] ValidateToken: stateless JWT validation
  - [ ] CleanupExpiredTokens: delegate to repo
- [ ] **Auth Service** — ~30 unit-тестов проходят
- [ ] **User Service** — все 4 операции реализованы:
  - [ ] GetProfile, UpdateProfile
  - [ ] GetSettings, UpdateSettings (с audit в транзакции)
- [ ] **User Service** — ~19 unit-тестов проходят
- [ ] **Auth Middleware:** Bearer token → ValidateToken → userID в ctx
- [ ] **Auth Middleware:** anonymous requests пропускаются
- [ ] **Request ID Middleware:** генерация/переиспользование request ID
- [ ] **EntityTypeUser** добавлен в domain enum
- [ ] Все input-структуры с `Validate()` — собирают все ошибки
- [ ] Логирование соответствует спецификации (INFO/WARN/ERROR)
- [ ] Секреты (tokens, codes) **не** попадают в логи
- [ ] Моки сгенерированы через `moq` из приватных интерфейсов
- [ ] `golang-jwt/jwt/v5` добавлен в `go.mod`
- [ ] Все acceptance criteria всех 7 задач выполнены
