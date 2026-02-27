# User Production-Ready Improvements Design

**Date:** 2026-02-27
**Status:** Approved

## Context

The user-related backend functionality (auth, profile, settings, admin) is architecturally sound but has gaps that prevent it from being production-ready. This design covers 5 high-priority improvements.

## 1. E2E Tests for Auth Flow

**Problem:** Zero E2E tests cover the auth REST endpoints (register, login, refresh, logout). All auth testing is unit-level with mocks.

**Solution:** New file `tests/e2e/auth_flow_test.go` using existing test infrastructure.

**Prerequisites:** Add auth handler registration to `setupTestServer()` in `helpers_test.go` — currently auth endpoints are not wired in the test mux.

**Test scenarios:**
- Register success → tokens returned → `me` GraphQL query works
- Register duplicate email → 409 Conflict
- Register invalid input (bad email, short password) → 400 with field errors
- Login with password → success, tokens returned
- Login wrong password → 401
- Login non-existent email → 401
- Token refresh → new tokens issued, old refresh token invalidated
- Token refresh with expired token → 401
- Logout → all refresh tokens revoked, refresh no longer works
- Full lifecycle: register → login → refresh → logout

**Pattern:** Follow `admin_test.go` REST testing pattern with `restRequest()` helper.

## 2. Unit Tests for Admin Functions

**Problem:** `SetUserRole` and `ListUsers` in user service have zero unit test coverage.

**Solution:** Add tests to existing `internal/service/user/service_test.go` following mock-based patterns already established there.

**Test scenarios for SetUserRole:**
- Admin changes another user's role → success
- Non-admin caller → ErrForbidden
- Admin demotes self → error (self-demotion prevention)
- Invalid role value → validation error
- Target user not found → ErrNotFound
- Repository error propagation

**Test scenarios for ListUsers:**
- Success with pagination (limit, offset, total count)
- Non-admin caller → ErrForbidden
- Empty result set
- Repository error propagation

## 3. IANA Timezone Validation

**Problem:** Settings accept any string up to 64 chars as timezone. Invalid values silently fall back to UTC in the study service.

**Solution:** Add `time.LoadLocation(tz)` validation in `internal/service/user/input.go` during settings input validation. Go's standard library includes an embedded IANA timezone database — no external dependencies needed.

**Changes:**
- `internal/service/user/input.go` — add `time.LoadLocation()` check in `UpdateSettingsInput.Validate()`
- `internal/service/user/input_test.go` — add tests for valid ("America/New_York", "UTC") and invalid ("Invalid/Zone", "garbage") timezones

## 4. Token Cleanup CLI Command

**Problem:** `CleanupExpiredTokens()` method exists in auth service but is never called. Expired/revoked tokens accumulate indefinitely.

**Solution:** New CLI command `cmd/cleanup-tokens/main.go` following the pattern of existing `cmd/promote/main.go`.

**Behavior:**
- Connects to database using `DATABASE_DSN` env var
- Deletes refresh tokens where `expires_at < now()` or `revoked_at IS NOT NULL`
- Logs count of deleted tokens
- Exit code 0 on success, 1 on error

**Deployment:** Run via cron, systemd timer, or Kubernetes CronJob (e.g., hourly).

## 5. Rate Limiting Middleware

**Problem:** Auth endpoints have no protection against brute-force attacks. No rate limiting exists anywhere in the application.

**Solution:** In-memory token bucket rate limiter as HTTP middleware.

### Configuration

New `RateLimitConfig` section in config system:

```yaml
rate_limit:
  enabled: true
  register: 5          # requests per minute per IP
  login: 10            # requests per minute per IP
  refresh: 20          # requests per minute per IP
  cleanup_interval: 5m # stale entry cleanup interval
```

Environment variables:
- `RATE_LIMIT_ENABLED` (default: `true`)
- `RATE_LIMIT_REGISTER` (default: `5`)
- `RATE_LIMIT_LOGIN` (default: `10`)
- `RATE_LIMIT_REFRESH` (default: `20`)
- `RATE_LIMIT_CLEANUP_INTERVAL` (default: `5m`)

### Implementation

**File:** `internal/transport/middleware/ratelimit.go`

**Algorithm:** Token bucket per IP using `sync.Map`. Each IP gets a bucket that refills at the configured rate. Background goroutine cleans up stale entries at `cleanup_interval`.

**Integration in app.go:** Each auth endpoint wrapped with its specific limit from config:
```go
limiter := middleware.NewRateLimiter(cfg.RateLimit)
mux.Handle("POST /auth/register", authCORS(limiter.Limit(cfg.RateLimit.Register, http.HandlerFunc(authHandler.Register))))
```

**Response on limit exceeded:** HTTP 429 Too Many Requests with `Retry-After` header (seconds until next allowed request).

**Graceful shutdown:** Limiter cleanup goroutine stops when server context is cancelled.

### Deploy model

Single instance, in-memory. No Redis dependency. If multi-replica deployment is needed later, the `RateLimiter` can be replaced with a Redis-backed implementation behind the same interface.

## Non-Goals

- Password reset/change flow (medium priority, separate design)
- Account deletion / GDPR (medium priority, separate design)
- Email verification (medium priority, separate design)
- Improved email validation with regex (low priority)
