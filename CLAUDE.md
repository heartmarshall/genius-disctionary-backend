# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MyEnglish Backend v4 — a Go backend for an English vocabulary learning app with spaced repetition (SRS). The Go module lives in `backend_v4/` (module path: `github.com/heartmarshall/myenglish-backend`). Documentation is in Russian in `docs/`.

## Commands

All commands run from `backend_v4/`:

```bash
# Build & run (requires env vars for DB and Auth)
DATABASE_DSN="postgres://..." AUTH_JWT_SECRET="..." AUTH_GOOGLE_CLIENT_ID="..." AUTH_GOOGLE_CLIENT_SECRET="..." go run ./cmd/server

# Tests
go test ./...                          # all tests
go test ./internal/domain/...          # single package
go test -run TestCard_IsDue ./internal/domain  # single test
go test -v -count=1 ./...             # verbose, no cache
go test -race ./...                    # with race detector
go test -coverprofile=coverage.out ./...  # with coverage

# Lint
golangci-lint run ./...

# Generate (mocks, sqlc — when tools are added)
go generate ./...
```

Required env vars for running: `DATABASE_DSN`, `AUTH_JWT_SECRET`, and at least one OAuth provider (Google: `AUTH_GOOGLE_CLIENT_ID` + `AUTH_GOOGLE_CLIENT_SECRET`, or Apple: `AUTH_APPLE_KEY_ID` + `AUTH_APPLE_TEAM_ID` + `AUTH_APPLE_PRIVATE_KEY`). See `backend_v4/.env.example` for all vars. Config loads from `config.yaml` + env override (env wins).

## Architecture

Clean architecture with consumer-defined interfaces and dependency flow:

```
transport → (own interfaces) ← service → (own interfaces) ← adapter
                                    ↓
                                  domain
```

- **`domain/`** — Pure Go structs, no external deps (no `db:""`, `json:""` tags). Sentinel errors in `errors.go`, enums with `IsValid()` method, text normalization.
- **`service/`** — Business logic. Each service defines its own dependency interfaces in its own package (not centralized). Input structs have `Validate() error` methods.
- **`adapter/postgres/`** — Repository implementations. Uses sqlc for static queries, Squirrel for dynamic. Maps sqlc models → domain. Transaction via context (`TxManager.RunInTx` puts tx in context, repos extract it).
- **`adapter/provider/`** — External API clients (FreeDictionary, Google OAuth).
- **`transport/graphql/`** — gqlgen resolvers. Defines its own service interfaces. Maps domain errors → GraphQL error codes.
- **`transport/rest/`** — Health, auth callbacks.
- **`internal/app/`** — Bootstrap: config loading, logger setup, graceful shutdown. Version via ldflags.
- **`internal/config/`** — cleanenv-based config (YAML + ENV). Validation of JWT secret length, OAuth, SRS params.
- **`pkg/ctxutil/`** — Context helpers for `userID` and `requestID`.
- **`cmd/server/main.go`** — Entry point. Wiring of all layers via Go duck typing.

**No centralized interface packages** (`port/`, `interfaces/`, `contract/`). Each consumer defines the minimal interface it needs.

## Key Conventions

### Dependencies & Imports
- `domain/` imports only stdlib
- `service/` never imports `adapter/` or `transport/`
- `transport/` never imports `adapter/`
- Wiring happens only in `main.go`

### Error Handling
- Sentinel errors from `domain/` only: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`
- Wrap errors: `fmt.Errorf("operation: %w", err)` — preserve sentinel unwrapping
- Validation uses `*domain.ValidationError` (collects all field errors, unwraps to `ErrValidation`)
- Adapter maps infra errors (e.g., `pgx.ErrNoRows` → `domain.ErrNotFound`)
- Service passes domain errors through, wraps only unexpected errors
- Errors logged once (in transport/middleware), never log-and-propagate

### Testing
- Naming: `Test<Target>_<Method>_<Scenario>` (e.g., `TestDictionary_CreateEntry_DuplicateText`)
- Table-driven tests for pure functions (SRS, validation)
- Mocks via `moq` from local interfaces into `_test.go` files: `//go:generate moq -out entry_repo_mock_test.go -pkg dictionary entryRepo`
- Integration tests: testcontainers-go with real PostgreSQL
- Use `t.Parallel()` where possible; no `time.Sleep`
- Coverage target: >=80% for service layer

### Validation
- Transport: format validation (parsing, types, required fields)
- Service: business rules (duplicates, limits, states). Input structs have `Validate() error`
- Repository: does not validate (trusts service)

### Logging
- `log/slog` (stdlib only). Injected via constructor with `.With("service", "name")`
- Always include `request_id` and `user_id` from context
- Structured fields (`slog.String(...)`)

### Commits
- Format: `<type>(<scope>): <description>`

### Database
- All queries filter by `user_id` — repos always take `userID` as parameter
- Soft delete pattern (`deleted_at` field) for entries
- Audit records written in same transaction as the operation

## Tech Stack

Go 1.23, PostgreSQL 17, pgx/v5, sqlc + Squirrel, goose migrations, gqlgen (GraphQL), cleanenv, net/http + ServeMux, google/uuid, golang-jwt/jwt/v5, moq, testcontainers-go, scany/v2.

## Implementation Status

Phase 1 (skeleton + domain) is complete. Phases 2-5 are documented in `docs/implimentation_phases/`. Phases 6-10 are planned. See `docs/implimentation_phases/00_overview.md` for the full roadmap.

## Documentation Map

- `docs/code_conventions_v4.md` — Architecture, code style, all patterns and rules
- `docs/data_model_v4.md` — Full DB schema (22 tables), ER diagram
- `docs/repo/` — Repository layer spec and task breakdown
- `docs/infra/infra_spec_v4.md` — Infrastructure: config, logging, Docker, Makefile
- `docs/services/` — Service specs (auth, dictionary, content, study, inbox, topic, refcatalog)
- `docs/implimentation_phases/` — Phase-by-phase implementation plans
