# Phase 4: Auth & User Services — Design

**Date:** 2026-02-15
**Status:** Approved
**Author:** Claude + User collaboration

## Overview

Implement authentication and user management for the MyEnglish backend:
- OAuth 2.0 authentication (Google, with Apple structure)
- JWT-based access tokens (HS256, stateless)
- Refresh token rotation for security
- User profile and settings management
- Auth middleware for request authentication

## Context

### What's Already Implemented

**Phase 1 (Domain & Config):**
- ✅ Domain models: `User`, `UserSettings`, `RefreshToken` with helper methods
- ✅ Context utilities: `UserIDFromCtx`, `RequestIDFromCtx`
- ✅ Sentinel errors: `ErrNotFound`, `ErrUnauthorized`, `ErrValidation`, etc.
- ✅ OAuth enums: `OAuthProviderGoogle`, `OAuthProviderApple`
- ✅ Config structure with `AuthConfig`

**Phase 2-3 (Database & Repos):**
- ✅ User repository: `GetByID`, `GetByOAuth`, `GetByEmail`, `Create`, `Update`
- ✅ Token repository: `Create`, `GetByHash`, `RevokeByID`, `RevokeAllByUser`, `DeleteExpired`
- ✅ Settings repository: `GetSettings`, `CreateSettings`, `UpdateSettings`
- ✅ Audit repository: `Create`, `GetByEntity`, `GetByUser`
- ✅ Transaction manager: Context-based `RunInTx`

**Phases 5-8 (Other Services):**
- ✅ Dictionary, Inbox, RefCatalog, Content, Study, Topic services
- ✅ Established patterns: private interfaces, moq mocking, structured logging

**Dependencies:**
- ✅ JWT library already in go.mod: `github.com/golang-jwt/jwt/v5`

### What Needs Implementation

**New packages:**
- `internal/auth/` — JWT manager and OAuth identity type
- `internal/service/auth/` — Auth service (5 operations, ~30 tests)
- `internal/service/user/` — User service (4 operations, ~19 tests)
- `internal/adapter/provider/google/` — Google OAuth verifier
- `internal/transport/middleware/` — Auth and request ID middleware

**Config extensions:**
- Add `JWTIssuer`, `GoogleRedirectURI` to `AuthConfig`
- Add `AllowedProviders()` computed method

**Domain additions:**
- Add `EntityTypeUser` enum value

## Key Adjustments from Spec Review

The detailed Phase 4 specification in `docs/implimentation_phases/phase_04_auth_user_services.md` required these adjustments to match actual codebase patterns:

### 1. Repository Signatures

**Services define narrow interfaces** (what they need), **repos implement with value returns** (what's convenient). Go's duck typing handles the conversion.

**Token Repository Interface:**
```go
// Service needs:
type tokenRepo interface {
    Create(ctx, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*domain.RefreshToken, error)
    DeleteExpired(ctx) (int, error)  // returns count
}

// Actual repo implements:
func (r *Repo) Create(...) (domain.RefreshToken, error)  // value return
func (r *Repo) DeleteExpired(ctx) error  // ⚠️ NEEDS UPDATE to return (int, error)
```

**User Repository Update Signature:**
```go
// Service needs:
Update(ctx, id uuid.UUID, name string, avatarURL *string) (*domain.User, error)

// Note: name is string (not *string) - service must use derefOrEmpty(identity.Name)
```

### 2. Mock Generation Pattern

Use **single `mocks_test.go` file** per package (matches existing services):
```go
//go:generate moq -out mocks_test.go -pkg auth . userRepo settingsRepo tokenRepo txManager oauthVerifier jwtManager
```

### 3. Audit Interface

Services can define audit interface as:
```go
type auditRepo interface {
    Create(ctx, record domain.AuditRecord) error  // ignore returned record
}
```

Actual repo returns `(domain.AuditRecord, error)` but services don't use the return value.

## Architecture

### Component Structure

```
┌─────────────────────────────────────────────────────────────┐
│ Transport Layer                                              │
│  ├── REST: /auth/google/callback, /auth/refresh, /auth/logout│
│  ├── GraphQL: user queries/mutations (future)               │
│  └── Middleware: Auth (JWT validation) + RequestID          │
└────────────┬────────────────────────────────────────────────┘
             │ (calls)
┌────────────▼────────────────────────────────────────────────┐
│ Service Layer                                                │
│  ├── auth.Service (5 operations)                            │
│  │    - Login (OAuth code exchange + user creation/update)  │
│  │    - Refresh (token rotation)                            │
│  │    - Logout (revoke all tokens)                          │
│  │    - ValidateToken (stateless JWT check)                 │
│  │    - CleanupExpiredTokens (maintenance)                  │
│  └── user.Service (4 operations)                            │
│       - GetProfile, UpdateProfile                           │
│       - GetSettings, UpdateSettings (with audit)            │
└────────────┬────────────────────────────────────────────────┘
             │ (uses)
┌────────────▼────────────────────────────────────────────────┐
│ Auth Infrastructure (internal/auth/)                         │
│  ├── JWTManager: GenerateAccessToken, ValidateAccessToken   │
│  │               GenerateRefreshToken, HashToken            │
│  └── OAuthIdentity: shared type for provider responses      │
└──────────────────────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────────┐
│ OAuth Adapters (internal/adapter/provider/)                  │
│  └── google.Verifier: VerifyCode (code → identity)          │
│       - Token exchange (code → access_token)                │
│       - Userinfo fetch (access_token → user data)           │
│       - Retry logic (1 retry on 5xx, 500ms backoff)         │
└──────────────────────────────────────────────────────────────┘
             │
┌────────────▼────────────────────────────────────────────────┐
│ Repository Layer (already implemented)                       │
│  ├── user.Repo: CRUD for users and settings                 │
│  ├── token.Repo: Refresh token management                   │
│  ├── audit.Repo: Audit log writes                           │
│  └── postgres.TxManager: Context-based transactions         │
└──────────────────────────────────────────────────────────────┘
```

### Data Flow Examples

**Login Flow (New User):**
```
1. Client → /auth/google/callback?code=xyz
2. Transport → authService.Login(code)
3. Auth Service:
   a. oauthVerifier.VerifyCode(code) → OAuthIdentity
   b. users.GetByOAuth(google, providerID) → ErrNotFound
   c. tx.RunInTx:
      - users.Create(user) → domain.User
      - settings.CreateSettings(defaults) → success
   d. jwt.GenerateAccessToken(userID) → access_token
   e. jwt.GenerateRefreshToken() → (raw, hash)
   f. tokens.Create(userID, hash, expiresAt) → token
4. Return: {access_token, refresh_token, user}
```

**Refresh Flow (Token Rotation):**
```
1. Client → /auth/refresh {refresh_token}
2. Auth Service:
   a. auth.HashToken(raw) → hash
   b. tokens.GetByHash(hash) → token (if not found/revoked → 401)
   c. Check token.IsExpired(now) → if yes → 401
   d. users.GetByID(token.UserID) → user
   e. tokens.RevokeByID(token.ID) → revoked ✓
   f. Generate new token pair
   g. tokens.Create(new hash)
3. Return: {new_access_token, new_refresh_token, user}
```

**Authenticated Request:**
```
1. Client → GET /graphql {Authorization: Bearer <jwt>}
2. Middleware:
   a. Extract token from header
   b. authService.ValidateToken(token) → userID (or 401)
   c. ctxutil.WithUserID(ctx, userID)
3. GraphQL Resolver:
   a. userID := ctxutil.UserIDFromCtx(ctx)
   b. userService.GetProfile(ctx) → uses userID
```

## Key Design Decisions

### 1. JWT Strategy

**Choice:** HS256 (symmetric) for MVP
- **Pros:** Simple, fast, single secret
- **Cons:** Can't verify tokens without secret (fine for monolith)
- **Future:** RS256 (asymmetric) for microservices post-MVP

**Claims:** Minimal set to avoid revocation needs
- `sub` (subject): userID
- `exp` (expires): now + 15m
- `iat` (issued at): now
- `iss` (issuer): "myenglish"
- **No custom claims** (roles, email) — these can change, requiring revocation

### 2. Refresh Token Rotation

**Choice:** Automatic rotation on every refresh
- Old token revoked immediately
- New token issued
- **Trade-off:** Concurrent refreshes fail (expected behavior)
- **Security:** Mitigates token theft (stolen token becomes invalid quickly)

**Reuse Detection (MVP):**
- Revoked token usage → log WARN "refresh token reuse attempted"
- **Not implemented:** Revoke all user tokens (post-MVP enhancement)

### 3. Access Token Revocation

**Choice:** No revocation for JWT access tokens (stateless design)
- **Trade-off:** Logout doesn't invalidate active access tokens (15min window)
- **Mitigation:** Short TTL (15 minutes)
- **Benefit:** No database lookup on every request (scalability)

### 4. OAuth Code Exchange

**Choice:** Exchange code + fetch userinfo (2 HTTP requests)
- **Not using:** ID token verification (more complex, requires key management)
- **MVP approach:** Simple and reliable
- **Retry:** 1 retry on 5xx/timeout, 500ms backoff

### 5. Email Collision Handling

**Choice:** Return `ErrAlreadyExists` (block registration)
- **MVP:** Do not merge accounts automatically
- **Post-MVP:** Manual account linking flow

### 6. Audit Scope

**Auth operations:** Not audited (login/logout/refresh)
- Logged via slog (INFO level)
- High frequency, low value for audit trail

**User settings:** Audited in `audit_log`
- `EntityTypeUser`, `AuditActionUpdate`
- Changes field: diff of old vs new settings
- Written in same transaction as update

**User profile:** Not audited (name/avatar)
- OAuth-provided data, changes are minor

## Testing Strategy

### Unit Tests

**Auth Service (~30 tests):**
- Login: new user registration, existing user login, profile updates, race conditions, email collision
- Refresh: success, token rotation, expired token, deleted user, reuse detection
- Logout: success, no tokens, unauthorized
- ValidateToken: valid/expired/malformed/invalid signature
- CleanupExpiredTokens: count return, no tokens, errors

**User Service (~19 tests):**
- GetProfile: success, unauthorized, not found
- UpdateProfile: success, validation (empty name, too long, invalid URL), unauthorized
- GetSettings: success, unauthorized
- UpdateSettings: success, validation (range checks, timezone), audit record, partial updates, unauthorized

**JWT Manager (~8 tests):**
- Generate + validate roundtrip
- Expired token rejection
- Invalid signature, malformed token, wrong issuer, wrong algorithm
- Refresh token generation: uniqueness, hash determinism

**Google OAuth Verifier (~7 tests):**
- Success flow (token exchange + userinfo)
- Invalid code (400)
- Google 5xx → retry → success
- Google 5xx → retry → 5xx (both fail)
- Timeout
- Unverified email
- Missing name/picture → nil fields

**Auth Middleware (~5 tests):**
- Valid token → userID in context
- Invalid token → 401
- No header → anonymous (pass through)
- Non-Bearer scheme → anonymous
- Empty Bearer → anonymous

**Request ID Middleware (~2 tests):**
- Incoming X-Request-Id → reuse
- No header → generate UUID

### Integration Tests

Not in Phase 4 scope (repos already have integration tests in Phase 3).

### Mock Strategy

- **Tool:** `moq` (code generation)
- **Pattern:** Single `mocks_test.go` per service package
- **TxManager mock:** Pass-through `fn(ctx)` (no real transaction)
- **Audit mock:** Always succeeds

## Implementation Tasks

Following the spec's task breakdown with adjustments:

### TASK-4.1: Config Extensions
- Add `JWTIssuer` and `GoogleRedirectURI` to `AuthConfig`
- Implement `AllowedProviders()` method (checks both clientID and secret)
- Implement `IsProviderAllowed(provider)` helper
- Refactor `validate.go` to use `AllowedProviders()` (remove `hasGoogleOAuth()`/`hasAppleOAuth()`)
- Unit tests for computed methods

### TASK-4.2: JWT Manager + OAuthIdentity
- Create `internal/auth/identity.go` with `OAuthIdentity` struct
- Create `internal/auth/jwt.go` with `JWTManager`
- Methods: `GenerateAccessToken`, `ValidateAccessToken`, `GenerateRefreshToken`, `HashToken`
- Claims validation: signature, expiry, issuer, algorithm
- Unit tests: generate/validate, errors, hash determinism

### TASK-4.3: Google OAuth Verifier
- Create `internal/adapter/provider/google/verifier.go`
- Token exchange: POST to `oauth2.googleapis.com/token`
- Userinfo fetch: GET from `www.googleapis.com/oauth2/v2/userinfo`
- Retry logic: 1 retry on 5xx, 500ms backoff
- Error handling: 400 → invalid code, 5xx → unavailable, unverified email → error
- Unit tests with `httptest.NewServer`

### TASK-4.4: Auth Service
- Create `internal/service/auth/service.go` with private interfaces
- Implement 5 operations: Login, Refresh, Logout, ValidateToken, CleanupExpiredTokens
- Input types with `Validate()` methods
- Handle corner cases: race condition, email collision, profile updates, token rotation
- Helper functions: `derefOrEmpty`, `profileChanged`, `ptrStringNotEqual`
- ~30 unit tests covering all flows
- Generate mocks: `//go:generate moq -out mocks_test.go -pkg auth . userRepo settingsRepo tokenRepo txManager oauthVerifier jwtManager`

### TASK-4.5: User Service
- Create `internal/service/user/service.go` with private interfaces
- Implement 4 operations: GetProfile, UpdateProfile, GetSettings, UpdateSettings
- UpdateSettings: merge input + audit in transaction
- Helpers: `applySettingsChanges`, `buildSettingsChanges` (diff for audit)
- ~19 unit tests
- Generate mocks

### TASK-4.6: Add EntityTypeUser
- Add `EntityTypeUser = "USER"` to `internal/domain/enums.go`
- Update `IsValid()` method
- No migration needed (will use existing audit_log table)

### TASK-4.7: Auth Middleware
- Create `internal/transport/middleware/auth.go`
- Define narrow `tokenValidator` interface
- Extract Bearer token from Authorization header
- Anonymous requests pass through (no 401)
- Invalid tokens → 401
- Valid tokens → `ctxutil.WithUserID`
- Create `internal/transport/middleware/request_id.go`
- Reuse incoming X-Request-Id or generate UUID
- Set in context and response header
- ~7 unit tests total

### TASK-4.8: Update Token Repository
- Modify `internal/adapter/postgres/token/repo.go`
- Update `DeleteExpired()` to return `(int, error)`
- Use `CommandTag.RowsAffected()` to get count

## Parallelization Strategy

**Wave 1 (independent):**
- TASK-4.1: Config extensions
- TASK-4.2: JWT Manager + OAuthIdentity
- TASK-4.6: EntityTypeUser enum
- TASK-4.8: Token repo update

**Wave 2 (depends on Wave 1):**
- TASK-4.3: Google OAuth (needs OAuthIdentity from 4.2)
- TASK-4.4: Auth Service (needs 4.1, 4.2, 4.8)
- TASK-4.5: User Service (needs 4.6)
- TASK-4.7: Auth Middleware (needs 4.2 types for understanding)

**Total:** 2 sequential waves, up to 4 parallel tasks per wave

## Success Criteria

- [ ] All 8 tasks completed
- [ ] ~65 unit tests passing (30 auth + 19 user + 8 jwt + 7 oauth + ~5 middleware)
- [ ] `go build ./...` compiles without errors
- [ ] `go test ./...` passes
- [ ] `golangci-lint run` clean
- [ ] No secrets in logs (tokens, codes, secrets masked)
- [ ] Config validates: JWT secret ≥32 chars, at least one OAuth provider
- [ ] Token rotation works: old revoked, new issued
- [ ] Unauthorized requests handled correctly (401 vs anonymous)

## Out of Scope (Post-MVP)

- Apple OAuth implementation (structure ready, not implemented)
- RS256 JWT signing (asymmetric keys)
- Revoke-all-tokens on reuse detection
- Account merging on email collision
- GraphQL auth endpoints (REST only for MVP)
- Password-based authentication
- 2FA/MFA
- Rate limiting
- Session management (stateless JWT only)

## References

- Main spec: `docs/implimentation_phases/phase_04_auth_user_services.md`
- Code conventions: `docs/code_conventions_v4.md`
- Service layer patterns: `docs/services/service_layer_spec_v4.md`
- Auth service detailed spec: `docs/services/auth_service_spec_v4.md`
- Data model: `docs/data_model_v4.md`
- Infrastructure: `docs/infra/infra_spec_v4.md`
