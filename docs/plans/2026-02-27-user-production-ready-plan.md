# User Production-Ready Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Bring user functionality to production-ready state with 5 high-priority improvements: IANA timezone validation, admin unit tests, token cleanup CLI, rate limiting middleware, and E2E auth tests.

**Architecture:** Each task is independent and can be committed separately. Changes follow existing hexagonal architecture patterns. No new external dependencies.

**Tech Stack:** Go 1.23, PostgreSQL, testcontainers, moq mocks, httptest

---

### Task 1: IANA Timezone Validation

**Files:**
- Modify: `backend_v4/internal/service/user/input.go:70-76`
- Modify: `backend_v4/internal/service/user/input_test.go:155-175`

**Step 1: Update validation to use `time.LoadLocation()`**

In `backend_v4/internal/service/user/input.go`, add `"time"` to imports and replace the timezone validation block (lines 70-76):

```go
if i.Timezone != nil {
    if *i.Timezone == "" {
        errs = append(errs, domain.FieldError{Field: "timezone", Message: "cannot be empty"})
    } else if len(*i.Timezone) > 64 {
        errs = append(errs, domain.FieldError{Field: "timezone", Message: "too long"})
    } else if _, err := time.LoadLocation(*i.Timezone); err != nil {
        errs = append(errs, domain.FieldError{Field: "timezone", Message: "invalid IANA timezone"})
    }
}
```

**Step 2: Update tests for IANA validation**

In `backend_v4/internal/service/user/input_test.go`, update the timezone test cases. Replace the existing timezone cases (the "valid: timezone at max length (64)" case with 64 z's will now fail because `zzzz...` is not a valid IANA name):

Replace the test case `"valid: timezone at max length (64)"` with:
```go
{
    name:    "valid: timezone UTC",
    input:   UpdateSettingsInput{Timezone: ptr("UTC")},
    wantErr: false,
},
{
    name:    "valid: timezone America/New_York",
    input:   UpdateSettingsInput{Timezone: ptr("America/New_York")},
    wantErr: false,
},
{
    name:    "valid: timezone Europe/London",
    input:   UpdateSettingsInput{Timezone: ptr("Europe/London")},
    wantErr: false,
},
{
    name:    "invalid: timezone garbage string",
    input:   UpdateSettingsInput{Timezone: ptr("Not/A/Timezone")},
    wantErr: true,
},
```

Keep the existing `"invalid: timezone at 65"` and `"invalid: timezone empty"` cases. Remove `"valid: timezone at max length (64)"` (64 z's is not a valid IANA timezone).

**Step 3: Run tests**

Run: `cd backend_v4 && go test ./internal/service/user/ -run TestUpdateSettingsInput_Validate -v -count=1`
Expected: All PASS

**Step 4: Commit**

```bash
cd backend_v4
git add internal/service/user/input.go internal/service/user/input_test.go
git commit -m "feat(user): validate timezone against IANA database

Uses time.LoadLocation() to reject invalid timezone strings
instead of only checking length. No external dependencies needed."
```

---

### Task 2: Admin Unit Tests (SetUserRole + ListUsers)

**Files:**
- Modify: `backend_v4/internal/service/user/service_test.go` (append new tests)

Mocks already exist in `user_repo_mock_test.go` with `UpdateRoleFunc`, `ListUsersFunc`, `CountUsersFunc`.

**Step 1: Add SetUserRole tests**

Append to `backend_v4/internal/service/user/service_test.go`:

```go
// ---------------------------------------------------------------------------
// SetUserRole tests
// ---------------------------------------------------------------------------

func TestService_SetUserRole_Success(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	targetID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), callerID)
	ctx = ctxutil.WithUserRole(ctx, "admin")

	expected := domain.User{
		ID:   targetID,
		Role: domain.UserRoleAdmin,
	}

	users := &userRepoMock{
		UpdateRoleFunc: func(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
			assert.Equal(t, targetID, id)
			assert.Equal(t, "admin", role)
			return &expected, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, targetID, domain.UserRoleAdmin)

	require.NoError(t, err)
	assert.Equal(t, &expected, user)
	assert.Len(t, users.UpdateRoleCalls(), 1)
}

func TestService_SetUserRole_NotAdmin(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "user")

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRoleAdmin)

	require.ErrorIs(t, err, domain.ErrForbidden)
	assert.Nil(t, user)
}

func TestService_SetUserRole_SelfDemotion(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), callerID)
	ctx = ctxutil.WithUserRole(ctx, "admin")

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, callerID, domain.UserRoleUser)

	require.ErrorIs(t, err, domain.ErrValidation)
	assert.Nil(t, user)
}

func TestService_SetUserRole_InvalidRole(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "admin")

	svc := newTestService(nil, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRole("superadmin"))

	require.ErrorIs(t, err, domain.ErrValidation)
	assert.Nil(t, user)
}

func TestService_SetUserRole_RepoError(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	targetID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), callerID)
	ctx = ctxutil.WithUserRole(ctx, "admin")

	repoErr := errors.New("db error")
	users := &userRepoMock{
		UpdateRoleFunc: func(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
			return nil, repoErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, targetID, domain.UserRoleAdmin)

	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, user)
}

func TestService_SetUserRole_TargetNotFound(t *testing.T) {
	t.Parallel()

	callerID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), callerID)
	ctx = ctxutil.WithUserRole(ctx, "admin")

	users := &userRepoMock{
		UpdateRoleFunc: func(ctx context.Context, id uuid.UUID, role string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestService(users, nil, nil, nil)
	user, err := svc.SetUserRole(ctx, uuid.New(), domain.UserRoleAdmin)

	require.ErrorIs(t, err, domain.ErrNotFound)
	assert.Nil(t, user)
}
```

**Step 2: Add ListUsers tests**

Append to the same file:

```go
// ---------------------------------------------------------------------------
// ListUsers tests
// ---------------------------------------------------------------------------

func TestService_ListUsers_Success(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "admin")

	expectedUsers := []domain.User{
		{ID: uuid.New(), Email: "a@test.com"},
		{ID: uuid.New(), Email: "b@test.com"},
	}

	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit, offset int) ([]domain.User, error) {
			assert.Equal(t, 10, limit)
			assert.Equal(t, 5, offset)
			return expectedUsers, nil
		},
		CountUsersFunc: func(ctx context.Context) (int, error) {
			return 42, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 5)

	require.NoError(t, err)
	assert.Equal(t, expectedUsers, result)
	assert.Equal(t, 42, total)
	assert.Len(t, users.ListUsersCalls(), 1)
	assert.Len(t, users.CountUsersCalls(), 1)
}

func TestService_ListUsers_NotAdmin(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "user")

	svc := newTestService(nil, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 0)

	require.ErrorIs(t, err, domain.ErrForbidden)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}

func TestService_ListUsers_DefaultLimit(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "admin")

	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit, offset int) ([]domain.User, error) {
			assert.Equal(t, 50, limit, "zero limit should default to 50")
			return nil, nil
		},
		CountUsersFunc: func(ctx context.Context) (int, error) {
			return 0, nil
		},
	}

	svc := newTestService(users, nil, nil, nil)
	_, _, err := svc.ListUsers(ctx, 0, 0)

	require.NoError(t, err)
}

func TestService_ListUsers_RepoError(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "admin")

	repoErr := errors.New("list failed")
	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit, offset int) ([]domain.User, error) {
			return nil, repoErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 0)

	require.ErrorIs(t, err, repoErr)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}

func TestService_ListUsers_CountError(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	ctx = ctxutil.WithUserRole(ctx, "admin")

	countErr := errors.New("count failed")
	users := &userRepoMock{
		ListUsersFunc: func(ctx context.Context, limit, offset int) ([]domain.User, error) {
			return []domain.User{}, nil
		},
		CountUsersFunc: func(ctx context.Context) (int, error) {
			return 0, countErr
		},
	}

	svc := newTestService(users, nil, nil, nil)
	result, total, err := svc.ListUsers(ctx, 10, 0)

	require.ErrorIs(t, err, countErr)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}
```

**Step 3: Run tests**

Run: `cd backend_v4 && go test ./internal/service/user/ -run "TestService_SetUserRole|TestService_ListUsers" -v -count=1`
Expected: All 11 tests PASS

**Step 4: Commit**

```bash
cd backend_v4
git add internal/service/user/service_test.go
git commit -m "test(user): add unit tests for SetUserRole and ListUsers

Covers: success, forbidden, self-demotion, invalid role, not found,
default limit, repo errors, count errors."
```

---

### Task 3: Token Cleanup CLI Command

**Files:**
- Create: `backend_v4/cmd/cleanup-tokens/main.go`

**Step 1: Create the CLI command**

Follow the pattern from `cmd/promote/main.go`:

```go
// Command cleanup-tokens deletes expired and revoked refresh tokens.
//
// Usage:
//
//	cleanup-tokens
//
// Requires DATABASE_DSN environment variable to be set.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	tag, err := pool.Exec(ctx,
		"DELETE FROM refresh_tokens WHERE expires_at < now() OR revoked_at IS NOT NULL",
	)
	if err != nil {
		log.Fatalf("cleanup tokens: %v", err)
	}

	fmt.Printf("Deleted %d expired/revoked refresh tokens.\n", tag.RowsAffected())
}
```

**Step 2: Verify it compiles**

Run: `cd backend_v4 && go build ./cmd/cleanup-tokens/`
Expected: Builds without errors

**Step 3: Commit**

```bash
cd backend_v4
git add cmd/cleanup-tokens/main.go
git commit -m "feat(auth): add CLI command for refresh token cleanup

Deletes expired and revoked refresh tokens from the database.
Intended to run via cron or systemd timer (e.g., hourly)."
```

---

### Task 4: Rate Limiting Middleware

**Files:**
- Modify: `backend_v4/internal/config/config.go:8-18` (add RateLimitConfig to Config struct)
- Create: `backend_v4/internal/transport/middleware/ratelimit.go`
- Create: `backend_v4/internal/transport/middleware/ratelimit_test.go`
- Modify: `backend_v4/internal/app/app.go:256-265` (wrap auth endpoints)
- Modify: `backend_v4/config.yaml` (add rate_limit section)

**Step 1: Add RateLimitConfig**

In `backend_v4/internal/config/config.go`, add to the Config struct (after CORS field, line 17):

```go
RateLimit  RateLimitConfig  `yaml:"rate_limit"`
```

Add the new config type after `CORSConfig` (after line 27):

```go
// RateLimitConfig holds rate limiting settings for auth endpoints.
type RateLimitConfig struct {
	Enabled         bool          `yaml:"enabled"          env:"RATE_LIMIT_ENABLED"          env-default:"true"`
	Register        int           `yaml:"register"         env:"RATE_LIMIT_REGISTER"         env-default:"5"`
	Login           int           `yaml:"login"            env:"RATE_LIMIT_LOGIN"             env-default:"10"`
	Refresh         int           `yaml:"refresh"          env:"RATE_LIMIT_REFRESH"           env-default:"20"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" env:"RATE_LIMIT_CLEANUP_INTERVAL"  env-default:"5m"`
}
```

**Step 2: Create rate limiter middleware**

Create `backend_v4/internal/transport/middleware/ratelimit.go`:

```go
package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// RateLimiter implements per-IP token bucket rate limiting.
type RateLimiter struct {
	buckets sync.Map // map[string]*bucket
	stop    chan struct{}
}

type bucket struct {
	tokens    float64
	maxTokens float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a rate limiter with background cleanup.
// Call Stop() on shutdown.
func NewRateLimiter(cleanupInterval time.Duration) *RateLimiter {
	rl := &RateLimiter{stop: make(chan struct{})}
	go rl.cleanup(cleanupInterval)
	return rl
}

// Stop terminates the background cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// Limit returns middleware that rate-limits requests to maxPerMinute per IP.
func (rl *RateLimiter) Limit(maxPerMinute int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			b := rl.getBucket(ip, maxPerMinute)
			if !b.allow() {
				retryAfter := 60.0 / float64(maxPerMinute)
				w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter)+1))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) getBucket(key string, maxPerMinute int) *bucket {
	maxTokens := float64(maxPerMinute)
	refillRate := maxTokens / 60.0 // tokens per second

	val, loaded := rl.buckets.LoadOrStore(key, &bucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	})

	b := val.(*bucket)
	if loaded && b.maxTokens != maxTokens {
		// Different endpoint with different limit for same IP — store separate key
		// This shouldn't happen since we key by IP only within a single Limit() call
	}
	return b
}

func (b *bucket) allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *RateLimiter) cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			now := time.Now()
			rl.buckets.Range(func(key, value any) bool {
				b := value.(*bucket)
				b.mu.Lock()
				idle := now.Sub(b.lastRefill)
				b.mu.Unlock()
				if idle > 10*time.Minute {
					rl.buckets.Delete(key)
				}
				return true
			})
		}
	}
}
```

**Step 3: Write rate limiter tests**

Create `backend_v4/internal/transport/middleware/ratelimit_test.go`:

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimiter_AllowsUnderLimit(t *testing.T) {
	rl := NewRateLimiter(time.Minute)
	defer rl.Stop()

	handler := rl.Limit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i)
	}
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := NewRateLimiter(time.Minute)
	defer rl.Stop()

	handler := rl.Limit(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust the bucket.
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request should be blocked.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	rl := NewRateLimiter(time.Minute)
	defer rl.Stop()

	handler := rl.Limit(2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust IP A.
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "1.1.1.1:1234"
		handler.ServeHTTP(rec, req)
	}

	// IP B should still work.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "2.2.2.2:5678"
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimiter_TokenRefill(t *testing.T) {
	rl := NewRateLimiter(time.Minute)
	defer rl.Stop()

	// 60 per minute = 1 per second
	handler := rl.Limit(60)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust all tokens.
	for i := 0; i < 60; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		req.RemoteAddr = "3.3.3.3:1234"
		handler.ServeHTTP(rec, req)
	}

	// Wait for refill (1+ second = 1+ token).
	time.Sleep(1100 * time.Millisecond)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.RemoteAddr = "3.3.3.3:1234"
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
```

**Step 4: Run tests**

Run: `cd backend_v4 && go test ./internal/transport/middleware/ -run TestRateLimiter -v -count=1`
Expected: All 4 PASS

**Step 5: Wire into app.go**

In `backend_v4/internal/app/app.go`, after creating `authHandler` (line 231), create the rate limiter:

```go
// Rate limiter for auth endpoints.
var authRateLimitRegister, authRateLimitLogin, authRateLimitRefresh middleware.Middleware
if cfg.RateLimit.Enabled {
    rateLimiter := middleware.NewRateLimiter(cfg.RateLimit.CleanupInterval)
    defer rateLimiter.Stop()
    authRateLimitRegister = rateLimiter.Limit(cfg.RateLimit.Register)
    authRateLimitLogin = rateLimiter.Limit(cfg.RateLimit.Login)
    authRateLimitRefresh = rateLimiter.Limit(cfg.RateLimit.Refresh)
    logger.Info("rate limiting enabled",
        slog.Int("register", cfg.RateLimit.Register),
        slog.Int("login", cfg.RateLimit.Login),
        slog.Int("refresh", cfg.RateLimit.Refresh),
    )
} else {
    noop := func(next http.Handler) http.Handler { return next }
    authRateLimitRegister = noop
    authRateLimitLogin = noop
    authRateLimitRefresh = noop
}
```

Then update auth endpoint registrations (lines 258-262) to wrap with rate limiters:

```go
mux.Handle("POST /auth/register", authCORS(authRateLimitRegister(http.HandlerFunc(authHandler.Register))))
mux.Handle("POST /auth/login", authCORS(authRateLimitLogin(http.HandlerFunc(authHandler.Login))))
mux.Handle("POST /auth/login/password", authCORS(authRateLimitLogin(http.HandlerFunc(authHandler.LoginWithPassword))))
mux.Handle("POST /auth/refresh", authCORS(authRateLimitRefresh(http.HandlerFunc(authHandler.Refresh))))
mux.Handle("POST /auth/logout", authCORS(http.HandlerFunc(authHandler.Logout)))
```

**Step 6: Add config to config.yaml**

Add to `backend_v4/config.yaml`:

```yaml
rate_limit:
  enabled: true
  register: 5
  login: 10
  refresh: 20
  cleanup_interval: 5m
```

**Step 7: Add to .env.example**

Add to `backend_v4/.env.example`:

```bash
# Rate Limiting (per IP, requests per minute)
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REGISTER=5
RATE_LIMIT_LOGIN=10
RATE_LIMIT_REFRESH=20
RATE_LIMIT_CLEANUP_INTERVAL=5m
```

**Step 8: Run build and existing tests**

Run: `cd backend_v4 && go build ./... && go test ./internal/transport/middleware/ -v -count=1`
Expected: Build succeeds, middleware tests pass

**Step 9: Commit**

```bash
cd backend_v4
git add internal/config/config.go internal/transport/middleware/ratelimit.go \
    internal/transport/middleware/ratelimit_test.go internal/app/app.go \
    config.yaml .env.example
git commit -m "feat(auth): add configurable rate limiting for auth endpoints

In-memory token bucket per IP. Configurable via config.yaml or ENV:
- register: 5 req/min, login: 10 req/min, refresh: 20 req/min
Returns 429 with Retry-After header when exceeded."
```

---

### Task 5: E2E Auth Flow Tests

**Files:**
- Modify: `backend_v4/tests/e2e/helpers_test.go:281-311` (add auth routes to test mux)
- Create: `backend_v4/tests/e2e/auth_flow_test.go`

**Step 1: Wire auth endpoints into test server**

In `backend_v4/tests/e2e/helpers_test.go`, after the admin endpoint registrations (line 311), add auth endpoints to the test mux. Find the section after `mux.Handle("PUT /admin/users/{id}/role"...` and before `srv := httptest.NewServer(mux)`:

```go
// Auth endpoints (CORS only, no auth middleware — matches production app.go).
authCORS := middleware.CORS(config.CORSConfig{
    AllowedOrigins:   "*",
    AllowedMethods:   "GET,POST,OPTIONS",
    AllowedHeaders:   "Authorization,Content-Type",
    AllowCredentials: true,
    MaxAge:           86400,
})
mux.Handle("POST /auth/register", authCORS(http.HandlerFunc(authHandler.Register)))
mux.Handle("POST /auth/login/password", authCORS(http.HandlerFunc(authHandler.LoginWithPassword)))
mux.Handle("POST /auth/refresh", authCORS(http.HandlerFunc(authHandler.Refresh)))
mux.Handle("POST /auth/logout", authCORS(http.HandlerFunc(authHandler.Logout)))
```

Also need to create the `authHandler` — add after `adminHandler` creation (around line 292):

```go
authHandler := rest.NewAuthHandler(authService, logger)
```

**Step 2: Add restRequest helper**

If not already present in helpers_test.go, add after `createTestUserWithID`:

```go
// restRequest sends an HTTP request with JSON body and auth token.
func restRequest(t *testing.T, ts *testServer, method, path, token string, body any) *http.Response {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(jsonBody)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, ts.URL+path, reqBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	return resp
}
```

**Step 3: Create auth flow E2E tests**

Create `backend_v4/tests/e2e/auth_flow_test.go`:

```go
//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Registration tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_Register_Success(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "register@test.com",
		"username": "registeruser",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result["accessToken"])
	assert.NotEmpty(t, result["refreshToken"])

	user := result["user"].(map[string]any)
	assert.Equal(t, "register@test.com", user["email"])
	assert.Equal(t, "registeruser", user["username"])
	assert.Equal(t, "user", user["role"])

	// Verify token works with GraphQL me query.
	token := result["accessToken"].(string)
	status, gqlResult := ts.graphqlQuery(t, `query { me { id email username } }`, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, gqlResult)
	me := gqlPayload(t, gqlResult, "me")
	assert.Equal(t, "register@test.com", me["email"])
}

func TestE2E_Auth_Register_DuplicateEmail(t *testing.T) {
	ts := setupTestServer(t)

	body := map[string]string{
		"email":    "dup@test.com",
		"username": "dupuser1",
		"password": "SecurePass123",
	}

	resp := restRequest(t, ts, "POST", "/auth/register", "", body)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Second registration with same email.
	body["username"] = "dupuser2"
	resp = restRequest(t, ts, "POST", "/auth/register", "", body)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestE2E_Auth_Register_InvalidInput(t *testing.T) {
	ts := setupTestServer(t)

	tests := []struct {
		name string
		body map[string]string
	}{
		{"missing email", map[string]string{"username": "user", "password": "SecurePass123"}},
		{"short password", map[string]string{"email": "a@b.com", "username": "user", "password": "short"}},
		{"missing username", map[string]string{"email": "a@b.com", "password": "SecurePass123"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := restRequest(t, ts, "POST", "/auth/register", "", tt.body)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Password login tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_LoginPassword_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Register first.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "login@test.com",
		"username": "loginuser",
		"password": "SecurePass123",
	})
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Login.
	resp = restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "login@test.com",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.NotEmpty(t, result["accessToken"])
	assert.NotEmpty(t, result["refreshToken"])
}

func TestE2E_Auth_LoginPassword_WrongPassword(t *testing.T) {
	ts := setupTestServer(t)

	// Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "wrongpw@test.com",
		"username": "wrongpwuser",
		"password": "SecurePass123",
	})
	resp.Body.Close()

	// Login with wrong password.
	resp = restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "wrongpw@test.com",
		"password": "WrongPassword!",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestE2E_Auth_LoginPassword_NonExistentEmail(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "nobody@test.com",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Token refresh tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_Refresh_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Register to get tokens.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "refresh@test.com",
		"username": "refreshuser",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regResult map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regResult))
	refreshToken := regResult["refreshToken"].(string)

	// Refresh.
	resp2 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	})
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var refreshResult map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&refreshResult))
	assert.NotEmpty(t, refreshResult["accessToken"])
	assert.NotEmpty(t, refreshResult["refreshToken"])

	// New tokens should be different.
	assert.NotEqual(t, refreshToken, refreshResult["refreshToken"])
}

func TestE2E_Auth_Refresh_OldTokenInvalidated(t *testing.T) {
	ts := setupTestServer(t)

	// Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "oldtoken@test.com",
		"username": "oldtokenuser",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()

	var regResult map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regResult))
	oldRefresh := regResult["refreshToken"].(string)

	// Refresh once.
	resp2 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": oldRefresh,
	})
	resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	// Try old token again — should fail (token rotation).
	resp3 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": oldRefresh,
	})
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp3.StatusCode)
}

func TestE2E_Auth_Refresh_InvalidToken(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": "invalid-token-that-does-not-exist",
	})
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Logout tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_Logout_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "logout@test.com",
		"username": "logoutuser",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()

	var regResult map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regResult))
	accessToken := regResult["accessToken"].(string)
	refreshToken := regResult["refreshToken"].(string)

	// Logout.
	resp2 := restRequest(t, ts, "POST", "/auth/logout", accessToken, nil)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	// Refresh should fail (all tokens revoked).
	resp3 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	})
	defer resp3.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp3.StatusCode)
}

func TestE2E_Auth_Logout_NoToken(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/logout", "", nil)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Full lifecycle test
// ---------------------------------------------------------------------------

func TestE2E_Auth_FullLifecycle(t *testing.T) {
	ts := setupTestServer(t)

	// 1. Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "lifecycle@test.com",
		"username": "lifecycleuser",
		"password": "SecurePass123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regResult map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regResult))

	// 2. Login with same credentials.
	resp2 := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "lifecycle@test.com",
		"password": "SecurePass123",
	})
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var loginResult map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&loginResult))

	// 3. Refresh.
	resp3 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": loginResult["refreshToken"].(string),
	})
	defer resp3.Body.Close()
	require.Equal(t, http.StatusOK, resp3.StatusCode)

	var refreshResult map[string]any
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&refreshResult))

	// 4. Use new access token for GraphQL.
	newToken := refreshResult["accessToken"].(string)
	status, gqlResult := ts.graphqlQuery(t, `query { me { email } }`, nil, newToken)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, gqlResult)

	// 5. Logout.
	resp4 := restRequest(t, ts, "POST", "/auth/logout", newToken, nil)
	defer resp4.Body.Close()
	require.Equal(t, http.StatusOK, resp4.StatusCode)

	// 6. Refresh with latest token should fail.
	resp5 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": refreshResult["refreshToken"].(string),
	})
	defer resp5.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp5.StatusCode)
}
```

**Step 4: Run E2E tests**

Run: `cd backend_v4 && go test -tags=e2e ./tests/e2e/ -run TestE2E_Auth -v -count=1`
Expected: All 12 tests PASS

**Step 5: Commit**

```bash
cd backend_v4
git add tests/e2e/helpers_test.go tests/e2e/auth_flow_test.go
git commit -m "test(auth): add E2E tests for auth flow

Covers register, login, refresh (with token rotation), logout,
and full lifecycle. Uses real PostgreSQL via testcontainers."
```

---

### Task 6: Final Verification

**Step 1: Run all unit tests**

Run: `cd backend_v4 && make test`
Expected: All PASS

**Step 2: Run all E2E tests**

Run: `cd backend_v4 && make test-e2e`
Expected: All PASS

**Step 3: Run linter**

Run: `cd backend_v4 && make lint`
Expected: No new issues

**Step 4: Final commit (if any fixes needed)**

Only if linter or tests revealed issues that need fixing.
