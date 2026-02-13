# MyEnglish Backend v4 — Auth Service Specification

> **Статус:** Draft v1.0
> **Дата:** 2026-02-12
> **Зависимости:** code_conventions_v4.md (секция 9), data_model_v4.md (секция 3), service_layer_spec_v4.md, business_scenarios_v4.md (A1–A6)

---

## 1. Ответственность

Auth Service отвечает за **идентификацию пользователя**: OAuth-аутентификация через внешних провайдеров, управление JWT-парой (access + refresh), token rotation, logout и валидация токенов для middleware.

Auth Service **не** отвечает за: авторизацию (проверку прав на конкретные ресурсы), управление профилем (UserService), настройки пользователя (UserService).

---

## 2. Зависимости

### 2.1. Интерфейсы репозиториев

```
userRepo interface {
    GetByID(ctx, userID uuid.UUID) → *domain.User, error
    GetByOAuth(ctx, provider string, oauthID string) → *domain.User, error
    CreateUser(ctx, user *domain.User) → *domain.User, error
    UpdateUser(ctx, userID uuid.UUID, name *string, avatarURL *string) → *domain.User, error
}

settingsRepo interface {
    CreateSettings(ctx, settings *domain.UserSettings) → error
}

tokenRepo interface {
    Create(ctx, token *domain.RefreshToken) → error
    GetByHash(ctx, tokenHash string) → *domain.RefreshToken, error
    RevokeByID(ctx, tokenID uuid.UUID) → error
    RevokeAllByUser(ctx, userID uuid.UUID) → error
    DeleteExpired(ctx) → (int, error)
}
```

### 2.2. Интерфейс TxManager

```
txManager interface {
    RunInTx(ctx, fn func(ctx) error) → error
}
```

### 2.3. Интерфейсы внешних компонентов

```
oauthVerifier interface {
    VerifyCode(ctx, provider string, code string) → *OAuthIdentity, error
}
```

OAuthIdentity — результат верификации OAuth code:
- Email (string) — email пользователя
- Name (*string) — отображаемое имя (может быть nil)
- AvatarURL (*string) — URL аватара (может быть nil)
- ProviderID (string) — уникальный ID пользователя у провайдера

```
jwtManager interface {
    GenerateAccessToken(userID uuid.UUID) → (string, error)
    ValidateAccessToken(token string) → (uuid.UUID, error)
    GenerateRefreshToken() → (rawToken string, hash string, error)
}
```

jwtManager — отдельный компонент (не сервис, а utility), живёт в `internal/auth/` или `internal/adapter/jwt/`. Отвечает за:
- Подпись и верификацию JWT access tokens (HS256 или RS256)
- Генерацию криптографически случайных refresh tokens
- Хеширование refresh tokens (SHA-256)

### 2.4. Конфигурация

```
AuthConfig {
    AccessTokenTTL    time.Duration   // default: 15m
    RefreshTokenTTL   time.Duration   // default: 30 days
    JWTSecret         string          // symmetric key for HS256
    JWTIssuer         string          // "myenglish"
    AllowedProviders  []string        // ["google", "apple"]
}
```

### 2.5. Конструктор

```
func NewService(
    logger      *slog.Logger,
    userRepo    userRepo,
    settingsRepo settingsRepo,
    tokenRepo   tokenRepo,
    txManager   txManager,
    oauth       oauthVerifier,
    jwt         jwtManager,
    config      AuthConfig,
) *Service
```

---

## 3. Domain Types

### 3.1. AuthResult

Результат Login и Refresh операций:
- AccessToken (string) — JWT access token
- RefreshToken (string) — raw refresh token (не hash)
- User (*domain.User) — данные пользователя

### 3.2. Используемые domain-модели

- `domain.User` — ID, Email, Name, AvatarURL, OAuthProvider, OAuthID, CreatedAt, UpdatedAt
- `domain.UserSettings` — UserID, NewCardsPerDay, ReviewsPerDay, MaxIntervalDays, Timezone, UpdatedAt
- `domain.RefreshToken` — ID, UserID, TokenHash, ExpiresAt, CreatedAt, RevokedAt

---

## 4. Бизнес-сценарии и операции

### 4.1. Login (A1 + A2)

**Сценарий A1 — Первый вход (регистрация):**
Пользователь впервые входит через Google → бэкенд создаёт аккаунт с дефолтными настройками → выдаёт JWT-пару.

**Сценарий A2 — Повторный вход:**
Пользователь входит через Google повторно → бэкенд находит существующий аккаунт → обновляет профиль если изменился → выдаёт JWT-пару.

**Метод:** `Login(ctx context.Context, input LoginInput) → (*AuthResult, error)`

**LoginInput:**
- Provider (string) — "google" или "apple"
- Code (string) — authorization code от OAuth-провайдера

**Пошаговый flow:**

```
1. input.Validate()
   └─ ошибка → return ValidationError

2. oauthVerifier.VerifyCode(ctx, provider, code)
   └─ ошибка → логировать ERROR "oauth verification failed", return error
   └─ успех → OAuthIdentity{Email, Name, AvatarURL, ProviderID}

3. userRepo.GetByOAuth(ctx, provider, identity.ProviderID)
   ├─ найден → existingUser
   │   ├─ 3a. Проверить, изменились ли name/avatar
   │   │   └─ если да → userRepo.UpdateUser(ctx, existingUser.ID, name, avatar)
   │   └─ 3b. user = existingUser (или обновлённый)
   │
   └─ ErrNotFound → новый пользователь
       └─ 4. txManager.RunInTx:
           ├─ 4a. userRepo.CreateUser(ctx, &domain.User{
           │       Email:         identity.Email,
           │       Name:          identity.Name,
           │       AvatarURL:     identity.AvatarURL,
           │       OAuthProvider: provider,
           │       OAuthID:       identity.ProviderID,
           │   })
           │   └─ ErrAlreadyExists → race condition (см. corner cases)
           │
           └─ 4b. settingsRepo.CreateSettings(ctx, domain.DefaultUserSettings(userID))

5. jwt.GenerateAccessToken(user.ID) → accessToken
6. jwt.GenerateRefreshToken() → rawRefresh, hashRefresh
7. tokenRepo.Create(ctx, &domain.RefreshToken{
       UserID:    user.ID,
       TokenHash: hashRefresh,
       ExpiresAt: now + config.RefreshTokenTTL,
   })

8. Логировать INFO:
   ├─ новый: "user registered" user_id=... provider=...
   └─ существующий: "user logged in" user_id=... provider=...

9. return &AuthResult{AccessToken: accessToken, RefreshToken: rawRefresh, User: user}
```

**Corner cases:**

- **OAuth provider unavailable:** VerifyCode возвращает ошибку (network timeout, invalid code, provider 5xx). Сервис возвращает ошибку как есть. Не retry — это ответственность oauthVerifier adapter.

- **Race condition при регистрации:** Два параллельных запроса с одним OAuth аккаунтом. Оба получают ErrNotFound от GetByOAuth, оба пытаются CreateUser. Первый успешно создаёт. Второй получает ErrAlreadyExists. Обработка: при ErrAlreadyExists от CreateUser внутри RunInTx — откатить транзакцию, повторить GetByOAuth, продолжить как login.

  ```
  4. txManager.RunInTx:
     └─ CreateUser → ErrAlreadyExists
        └─ return ErrAlreadyExists (tx откатится)
  4-fallback. userRepo.GetByOAuth(ctx, provider, identity.ProviderID) → user
  5. продолжить с шага 5 (issue tokens)
  ```

- **Email collision:** Пользователь зарегистрировался через Google, потом входит через Apple с тем же email. GetByOAuth(apple, ...) → не найден. CreateUser → ErrAlreadyExists (ux_users_email). На MVP: вернуть ошибку "account with this email already exists, use [google] to login". Merge аккаунтов — post-MVP.

- **Identity.Name = nil:** Некоторые OAuth-провайдеры не возвращают name. Допустимо: name остаётся nil в users.

- **Обновление профиля при login:** Если name/avatar у провайдера изменились (пользователь обновил Google-профиль), обновляем в БД. Сравниваем текущие значения с identity. Обновляем только если отличаются, чтобы не генерировать лишних UPDATE.

---

### 4.2. Refresh (A3)

**Сценарий:** Access token истёк → клиент отправляет refresh token → сервис проверяет, отзывает старый, выдаёт новую пару (token rotation).

**Метод:** `Refresh(ctx context.Context, input RefreshInput) → (*AuthResult, error)`

**RefreshInput:**
- RefreshToken (string) — raw refresh token

**Пошаговый flow:**

```
1. input.Validate()
   └─ ошибка → return ValidationError

2. hash = jwtManager.HashToken(input.RefreshToken)

3. tokenRepo.GetByHash(ctx, hash)
   └─ ErrNotFound → return domain.ErrUnauthorized
      (token не существует или уже revoked — GetByHash фильтрует revoked)

4. Проверки на полученном token:
   ├─ token.ExpiresAt < now → return domain.ErrUnauthorized
   └─ (revoked_at уже отфильтрован repo)

5. userRepo.GetByID(ctx, token.UserID)
   └─ ErrNotFound → return domain.ErrUnauthorized (пользователь удалён)

6. tokenRepo.RevokeByID(ctx, token.ID)
   (отзыв текущего refresh token — token rotation)

7. jwt.GenerateAccessToken(user.ID) → newAccessToken
8. jwt.GenerateRefreshToken() → newRawRefresh, newHash
9. tokenRepo.Create(ctx, &domain.RefreshToken{
       UserID:    user.ID,
       TokenHash: newHash,
       ExpiresAt: now + config.RefreshTokenTTL,
   })

10. return &AuthResult{AccessToken: newAccessToken, RefreshToken: newRawRefresh, User: user}
```

**Corner cases:**

- **Token rotation — replay detection:** Refresh token одноразовый. После использования — revoked. Если кто-то пытается использовать уже revoked token, GetByHash вернёт ErrNotFound (repo фильтрует revoked). Это сигнал потенциальной кражи. Логировать WARN "refresh token reuse attempted" с hash prefix (первые 8 символов). **Не** отзывать все tokens пользователя автоматически на MVP — слишком агрессивно (может быть race condition между двумя tabs клиента).

- **Concurrent refresh:** Два tab/устройства отправляют refresh одновременно с одним и тем же токеном. Первый успешно проходит, revoke-ит token. Второй получает ErrNotFound от GetByHash → ErrUnauthorized. Второй tab должен перенаправить на login. Это ожидаемое поведение token rotation.

- **Expired token:** Клиент может отправить expired refresh token (если не обращался 30+ дней). Проверка ExpiresAt → ErrUnauthorized. Клиент перенаправляется на login.

- **Шаги 6–9 не в транзакции:** Revoke старого и создание нового token — две отдельные операции. Если crash между ними — старый revoked, новый не создан → пользователь перенаправляется на login. Это безопаснее, чем транзакция (которая могла бы оставить невалидный старый token при откате).

---

### 4.3. Logout (A4)

**Сценарий:** Пользователь выходит → все его refresh tokens отзываются (logout everywhere).

**Метод:** `Logout(ctx context.Context) → error`

**Пошаговый flow:**

```
1. userID, err = UserIDFromCtx(ctx)
   └─ ошибка → return domain.ErrUnauthorized

2. tokenRepo.RevokeAllByUser(ctx, userID)

3. Логировать INFO "user logged out" user_id=...

4. return nil
```

**Corner cases:**

- **Нет активных tokens:** RevokeAllByUser на пользователе без active tokens — идемпотентно, 0 affected rows — не ошибка.
- **Access token остаётся валидным:** Logout отзывает только refresh tokens. Access token (JWT) остаётся валидным до истечения TTL (15 минут). Это архитектурное решение: JWT stateless, проверка revocation для каждого запроса требует обращения к БД и убивает преимущества JWT. Клиент должен удалить access token из памяти при logout.

---

### 4.4. ValidateToken (A5)

**Сценарий:** Middleware проверяет JWT access token при каждом входящем запросе.

**Метод:** `ValidateToken(ctx context.Context, token string) → (uuid.UUID, error)`

**Пошаговый flow:**

```
1. jwt.ValidateAccessToken(token)
   ├─ невалидная подпись → return uuid.Nil, domain.ErrUnauthorized
   ├─ expired → return uuid.Nil, domain.ErrUnauthorized
   ├─ malformed → return uuid.Nil, domain.ErrUnauthorized
   └─ ok → userID

2. return userID, nil
```

**Важные свойства:**

- **Без обращения к БД.** Это критически важно — ValidateToken вызывается при каждом запросе. JWT проверяется только по подписи и expiry. Stateless.
- **Не проверяет существование пользователя.** Пользователь мог быть удалён между выдачей и использованием токена. Это обрабатывается на уровне сервисов: repo вернёт ErrNotFound при попытке обращения к данным.
- **Не проверяет revocation.** JWT access token невозможно отозвать до истечения TTL. Это trade-off: производительность vs мгновенный revocation. При TTL 15 минут это допустимо.

---

### 4.5. CleanupExpiredTokens (A6)

**Сценарий:** Фоновая задача удаляет expired и revoked refresh tokens.

**Метод:** `CleanupExpiredTokens(ctx context.Context) → (int, error)`

**Пошаговый flow:**

```
1. count, err = tokenRepo.DeleteExpired(ctx)
   └─ ошибка → логировать ERROR "token cleanup failed", return

2. if count > 0 → логировать INFO "cleaned up expired tokens" count=...

3. return count, nil
```

**Свойства:**
- Не требует userID (background job).
- Вызывается scheduler-ом (горутина с ticker или external cron).
- DeleteExpired удаляет токены, где `expires_at < now()` ИЛИ `revoked_at IS NOT NULL`.
- Может удалять тысячи записей — без транзакции (каждый DELETE атомарен сам по себе).

---

## 5. Input Validation

### 5.1. LoginInput

```
type LoginInput struct {
    Provider string
    Code     string
}

func (i LoginInput) Validate() error:
    errors = []
    if Provider == ""         → append("provider", "required")
    if Provider not in AllowedProviders → append("provider", "unsupported provider")
    if Code == ""             → append("code", "required")
    if len(Code) > 4096       → append("code", "too long")
    if len(errors) > 0        → return ValidationError{errors}
    return nil
```

AllowedProviders берётся из конфигурации ("google", "apple"). Если в будущем добавится новый провайдер — достаточно обновить config.

### 5.2. RefreshInput

```
type RefreshInput struct {
    RefreshToken string
}

func (i RefreshInput) Validate() error:
    if RefreshToken == ""     → return ValidationError{("refresh_token", "required")}
    if len(RefreshToken) > 512 → return ValidationError{("refresh_token", "too long")}
    return nil
```

---

## 6. Error Scenarios — полная таблица

| Операция | Условие | Ошибка | HTTP/GQL | Логирование |
|----------|---------|--------|----------|-------------|
| Login | Невалидный input | ValidationError | 400 | — |
| Login | Provider не поддерживается | ValidationError ("unsupported provider") | 400 | — |
| Login | OAuth code невалиден / expired | Ошибка от oauthVerifier | 401 | ERROR "oauth verification failed" |
| Login | OAuth provider недоступен (timeout, 5xx) | Ошибка от oauthVerifier | 502 | ERROR "oauth provider unavailable" |
| Login | Email уже занят другим провайдером | ErrAlreadyExists | 409 | WARN "email collision" email=... providers=... |
| Login | Race condition (concurrent registration) | — (handled internally, retry) | — | WARN "concurrent registration, retrying" |
| Refresh | Невалидный input | ValidationError | 400 | — |
| Refresh | Token не найден / уже revoked | ErrUnauthorized | 401 | — |
| Refresh | Token expired | ErrUnauthorized | 401 | — |
| Refresh | Пользователь не существует | ErrUnauthorized | 401 | WARN "refresh for deleted user" |
| Refresh | Повторное использование revoked token | ErrUnauthorized | 401 | WARN "refresh token reuse" |
| Logout | Нет userID в ctx | ErrUnauthorized | 401 | — |
| ValidateToken | Невалидный/expired/malformed JWT | ErrUnauthorized | 401 | — |
| Cleanup | Ошибка БД | error (проброс) | — | ERROR "token cleanup failed" |

---

## 7. Security Considerations

### 7.1. Refresh Token Storage

- Refresh token хранится в БД как SHA-256 hash. Raw token знает только клиент.
- Raw token передаётся клиенту один раз при Login/Refresh. Бэкенд не может восстановить raw token из hash.
- При компрометации БД атакующий получает hashes, но не может использовать их для refresh.

### 7.2. Token Rotation

- Каждый refresh token одноразовый. После использования — revoked, выдаётся новый.
- Это обеспечивает: обнаружение кражи (reuse detection), ограничение window of attack.
- Trade-off: два tab с одним refresh token → второй потеряет сессию. Клиент должен обрабатывать 401 на refresh и перенаправлять на login.

### 7.3. JWT Access Token

- Signing: HS256 (symmetric) на MVP. Переход на RS256 (asymmetric) если понадобится верификация без секрета (microservices).
- Claims: минимальные — sub (userID), exp, iat, iss. Без ролей, permissions, email (они могут измениться).
- TTL: 15 минут. Достаточно короткий для безопасности, достаточно длинный чтобы не refresh-ить слишком часто.

### 7.4. OAuth Code

- OAuth authorization code одноразовый (стандарт OAuth 2.0). Если VerifyCode удался — code использован. Повторная отправка того же code → ошибка от провайдера.
- Сервис не кеширует и не хранит authorization codes.

---

## 8. Тесты

### 8.1. Моки

Все зависимости мокаются: userRepo, settingsRepo, tokenRepo, txManager, oauthVerifier, jwtManager. Mock txManager просто вызывает `fn(ctx)`.

### 8.2. Конкретные тест-кейсы

**Login — happy path:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| L1 | Первый вход — регистрация | GetByOAuth → ErrNotFound, CreateUser → ok, CreateSettings → ok | AuthResult с новым user. CreateUser called once. CreateSettings called once. LOG INFO "user registered" |
| L2 | Повторный вход | GetByOAuth → existingUser | AuthResult с existing user. CreateUser NOT called |
| L3 | Повторный вход — профиль изменился | GetByOAuth → user(name="Old"), identity.Name="New" | UpdateUser called with name="New". AuthResult с updated user |
| L4 | Повторный вход — профиль не изменился | GetByOAuth → user(name="Same"), identity.Name="Same" | UpdateUser NOT called |

**Login — validation:**

| # | Тест | Input | Assert |
|---|------|-------|--------|
| L5 | Пустой provider | Provider="", Code="abc" | ValidationError, field="provider" |
| L6 | Неподдерживаемый provider | Provider="facebook" | ValidationError, field="provider", message="unsupported" |
| L7 | Пустой code | Provider="google", Code="" | ValidationError, field="code" |

**Login — error scenarios:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| L8 | OAuth verification failed | VerifyCode → error | Ошибка проброшена |
| L9 | Race condition — concurrent registration | GetByOAuth → ErrNotFound, CreateUser → ErrAlreadyExists, повторный GetByOAuth → user | AuthResult с existing user. Нет паники. |
| L10 | Email collision | GetByOAuth → ErrNotFound, CreateUser → ErrAlreadyExists (email), повторный GetByOAuth → ErrNotFound | ErrAlreadyExists |

**Login — tokens:**

| # | Тест | Assert |
|---|------|--------|
| L11 | Access token генерируется | GenerateAccessToken called with correct userID |
| L12 | Refresh token сохраняется | tokenRepo.Create called with hashed token, correct userID, correct ExpiresAt |
| L13 | Raw refresh token в ответе | AuthResult.RefreshToken == raw (not hash) |

**Refresh — happy path:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| R1 | Успешный refresh | GetByHash → valid token, GetByID → user | Новая JWT-пара. Старый token revoked. Новый token создан. |
| R2 | User data в ответе | GetByID → user | AuthResult.User содержит актуальные данные |

**Refresh — error scenarios:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| R3 | Token не найден | GetByHash → ErrNotFound | ErrUnauthorized |
| R4 | Token expired | GetByHash → token(ExpiresAt=past) | ErrUnauthorized |
| R5 | User deleted | GetByHash → ok, GetByID → ErrNotFound | ErrUnauthorized |
| R6 | Пустой input | RefreshToken="" | ValidationError |

**Refresh — rotation:**

| # | Тест | Assert |
|---|------|--------|
| R7 | Старый token revoked | RevokeByID called with old token ID |
| R8 | Новый token создан | tokenRepo.Create called with new hash |
| R9 | Новый token в ответе | AuthResult.RefreshToken != input.RefreshToken |

**Logout:**

| # | Тест | Setup | Assert |
|---|------|-------|--------|
| LO1 | Успешный logout | userID в ctx | RevokeAllByUser called with correct userID |
| LO2 | Нет userID в ctx | ctx без userID | ErrUnauthorized |
| LO3 | Нет active tokens | RevokeAllByUser → 0 affected | Нет ошибки |

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
| CL1 | Есть expired tokens | DeleteExpired → 42 | Returns 42, logged |
| CL2 | Нет expired tokens | DeleteExpired → 0 | Returns 0, no log |
| CL3 | DB error | DeleteExpired → error | Error проброшен, logged ERROR |

---

## 9. Файловая структура

```
internal/service/auth/
├── service.go           # Service struct, конструктор, приватные интерфейсы,
│                        #   Login, Refresh, Logout, ValidateToken, CleanupExpiredTokens
├── input.go             # LoginInput, RefreshInput + Validate()
└── service_test.go      # Все тесты из секции 8

internal/auth/           # (или internal/adapter/jwt/)
├── jwt.go               # JWT manager: generate/validate access, generate/hash refresh
└── jwt_test.go          # Unit-тесты JWT operations
```

jwtManager — **не** часть auth service. Это adapter/utility, который инжектится в сервис через интерфейс. Сервис не знает про HS256, signing keys, crypto — он знает только "дай мне токен" и "проверь токен".

---

## 10. Взаимодействие с другими компонентами

### 10.1. Transport Layer (REST)

Auth endpoints реализуются как **REST**, не GraphQL:

| Endpoint | Method | Описание |
|----------|--------|----------|
| `/auth/login` | POST | Принимает {provider, code}, возвращает {access_token, refresh_token, user} |
| `/auth/refresh` | POST | Принимает {refresh_token}, возвращает {access_token, refresh_token, user} |
| `/auth/logout` | POST | Требует Authorization header. Отзывает все refresh tokens |

Причина REST вместо GraphQL: auth endpoints вызываются до установления GraphQL-сессии. Refresh может вызываться когда access token уже expired (GraphQL middleware вернёт 401).

### 10.2. Auth Middleware

Middleware вызывает `authService.ValidateToken(ctx, token)` для каждого запроса с `Authorization: Bearer` header. Middleware определяет свой узкий интерфейс:

```
type tokenValidator interface {
    ValidateToken(ctx, token string) → (uuid.UUID, error)
}
```

### 10.3. Scheduler

Scheduler вызывает `authService.CleanupExpiredTokens(ctx)` периодически (каждые 24 часа). Не требует authentication context.

---

## 11. Нерешённые вопросы (для обсуждения)

| # | Вопрос | Варианты | Рекомендация |
|---|--------|----------|--------------|
| 1 | Refresh token — cookie или body | HTTPOnly cookie (безопаснее) vs JSON body (проще для mobile) | HTTPOnly cookie для web, body для mobile. Определить при реализации transport layer. |
| 2 | Token reuse → revoke all | При обнаружении reuse revoke все tokens пользователя (Дополнительная security) vs просто reject (проще) | Просто reject на MVP. Revoke all — post-MVP после анализа false positive rate. |
| 3 | Rate limiting на login/refresh | Middleware или in-service | Middleware (transport layer). Не в сервисе. |
