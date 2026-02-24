# Preloaded Datasets Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Migrate from on-demand FreeDictionary API to a preloaded reference catalog with lazy LLM enrichment.

**Architecture:** Python script filters merged word pool against Wiktionary dump, Go seeder populates ref_entries, new enrichment_queue table tracks user interest, enricher/importer pipelines enhanced with replace semantics, admin role gates management endpoints.

**Tech Stack:** Go 1.23, PostgreSQL, pgx v5, gqlgen, goose migrations, Python 3 (filtering script)

---

## Task 0: Commit directory reorganization

The move from `internal/seeder/`, `internal/enricher/`, `internal/llm_importer/` to `internal/app/` is done but uncommitted.

**Files:**
- Staged: All files listed in `git status` (deletions from old paths + additions in new paths)

**Step 1: Stage all changes**

```bash
cd backend_v4
git add internal/app/seeder/ internal/app/enricher/ internal/app/llm_importer/
git add internal/seeder/ internal/enricher/ internal/llm_importer/
git add cmd/seeder/main.go cmd/enrich/main.go cmd/llm-import/main.go
```

**Step 2: Verify build**

```bash
go build ./...
```

**Step 3: Run tests**

```bash
go test ./internal/app/seeder/... ./internal/app/enricher/... ./internal/app/llm_importer/... -race -count=1
```

**Step 4: Commit**

```bash
git commit -m "refactor: move seeder, enricher, llm_importer to internal/app/"
```

---

## Task 1: Filter merged pool against Wiktionary

Create a Python script that reads `datasets/merged/merged_pool.csv` and the 2.7GB Wiktionary Kaikki JSONL dump, outputs two files:
- `datasets/merged/seed_wordlist.txt` — words found in Wiktionary
- `datasets/merged/unmatched_words.txt` — words NOT found

**Files:**
- Create: `datasets/merged/filter_by_wiktionary.py`

**Step 1: Write the filter script**

The script must:
1. Stream the Kaikki JSONL (line by line, no full load — it's 2.7GB)
2. Collect all English word forms (normalized to lowercase) into a set
3. Read `merged_pool.csv` column `word`
4. Split into matched/unmatched
5. Write `seed_wordlist.txt` (one word per line) and `unmatched_words.txt`

```python
#!/usr/bin/env python3
"""Filter merged_pool.csv: keep only words found in Wiktionary Kaikki dump."""

import argparse
import csv
import json
import sys


def load_wiktionary_words(path: str) -> set[str]:
    """Stream Kaikki JSONL, collect normalized English words."""
    words = set()
    with open(path, "r", encoding="utf-8") as f:
        for i, line in enumerate(f):
            if i % 500_000 == 0 and i > 0:
                print(f"  ...scanned {i:,} lines, {len(words):,} unique words", file=sys.stderr)
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue
            if entry.get("lang") != "English":
                continue
            word = entry.get("word", "").strip().lower()
            if word:
                words.add(word)
    return words


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--wiktionary", required=True, help="Path to kaikki.org JSONL dump")
    parser.add_argument("--pool", default="merged_pool.csv", help="Path to merged_pool.csv")
    parser.add_argument("--matched", default="seed_wordlist.txt", help="Output: matched words")
    parser.add_argument("--unmatched", default="unmatched_words.txt", help="Output: unmatched words")
    args = parser.parse_args()

    print("Loading Wiktionary words...", file=sys.stderr)
    wikt_words = load_wiktionary_words(args.wiktionary)
    print(f"Wiktionary contains {len(wikt_words):,} unique English words", file=sys.stderr)

    matched = []
    unmatched = []
    with open(args.pool, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            word = row["word"].strip().lower()
            if word in wikt_words:
                matched.append(word)
            else:
                unmatched.append(word)

    matched.sort()
    unmatched.sort()

    with open(args.matched, "w", encoding="utf-8") as f:
        f.write("\n".join(matched) + "\n")

    with open(args.unmatched, "w", encoding="utf-8") as f:
        f.write("\n".join(unmatched) + "\n")

    print(f"Matched: {len(matched):,} | Unmatched: {len(unmatched):,}", file=sys.stderr)


if __name__ == "__main__":
    main()
```

**Step 2: Run the filter**

```bash
cd datasets/merged
python filter_by_wiktionary.py \
  --wiktionary ../kaikki.org-dictionary-English.jsonl \
  --pool merged_pool.csv \
  --matched seed_wordlist.txt \
  --unmatched unmatched_words.txt
```

Expected: seed_wordlist.txt with ~20-40k words, unmatched_words.txt with the remainder.

**Step 3: Commit**

```bash
git add datasets/merged/filter_by_wiktionary.py
git commit -m "feat(datasets): add Wiktionary filter script for merged pool"
```

---

## Task 2: Run seeder with filtered wordlist

This is an operational task — run the existing seeder pipeline with the new wordlist.

**Step 1: Update seeder config**

Edit `backend_v4/seeder.yaml` to point `wiktionary_path` to the Kaikki dump and verify `top_n` covers the wordlist size.

**Step 2: Start database**

```bash
cd backend_v4
make docker-up
make migrate-up
```

**Step 3: Run seeder**

```bash
cd backend_v4
go run ./cmd/seeder --seeder-config=seeder.yaml
```

**Step 4: Verify**

```sql
SELECT count(*) FROM ref_entries;
SELECT count(*) FROM ref_senses;
SELECT count(*) FROM ref_translations;
SELECT count(*) FROM ref_examples;
SELECT count(*) FROM ref_pronunciations;
```

---

## Task 3: Add admin role to domain

**Files:**
- Modify: `backend_v4/internal/domain/enums.go` — add `UserRole` enum
- Modify: `backend_v4/internal/domain/user.go` — add `Role` field to `User`

**Step 1: Add UserRole enum to enums.go**

After `SessionStatus` block, add:

```go
// UserRole represents the authorization level of a user.
type UserRole string

const (
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

func (r UserRole) String() string { return string(r) }

func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleUser, UserRoleAdmin:
		return true
	}
	return false
}

func (r UserRole) IsAdmin() bool {
	return r == UserRoleAdmin
}
```

**Step 2: Add Role field to User struct in user.go**

```go
type User struct {
	ID        uuid.UUID
	Email     string
	Username  string
	Name      string
	AvatarURL *string
	Role      UserRole  // <-- add this field
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

**Step 3: Run tests**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git commit -m "feat(domain): add UserRole enum and Role field to User"
```

---

## Task 4: Migration — add role column to users

**Files:**
- Create: `backend_v4/migrations/00016_user_role.sql`

**Step 1: Create migration**

```sql
-- +goose Up
ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS role;
```

**Step 2: Apply migration**

```bash
cd backend_v4
make migrate-up
```

**Step 3: Verify**

```bash
psql -c "SELECT column_name, data_type, column_default FROM information_schema.columns WHERE table_name='users' AND column_name='role';"
```

**Step 4: Commit**

```bash
git add migrations/00016_user_role.sql
git commit -m "feat(migration): add role column to users table"
```

---

## Task 5: Update user repo to read/write role

**Files:**
- Modify: User repo SQL queries and Go mapping to include `role` column
- Modify: Auth service registration to set default role

**Step 1: Find and update all user repo queries**

Every query that reads from `users` table and maps to `domain.User` must now include `role` column. Every insert must set it.

Scan files:
```bash
grep -rn "SELECT.*FROM users" internal/adapter/postgres/user/
```

Update all SELECT queries to include `role`, all INSERT queries to include `role` with default `'user'`.

Update the row scanner to populate `user.Role`.

**Step 2: Run tests**

```bash
go test ./internal/adapter/postgres/user/... -race -count=1
go build ./...
```

**Step 3: Commit**

```bash
git commit -m "feat(repo): include role column in user queries"
```

---

## Task 6: Add role to context + admin middleware

**Files:**
- Modify: `backend_v4/pkg/ctxutil/ctxutil.go` — add `WithUserRole` / `UserRoleFromCtx`
- Create: `backend_v4/internal/transport/middleware/admin.go` — admin check middleware
- Modify: `backend_v4/internal/transport/middleware/auth.go` — store role in context after validation

**Step 1: Add role context helpers to ctxutil.go**

```go
const userRoleKey ctxKey = "user_role"

func WithUserRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, userRoleKey, role)
}

func UserRoleFromCtx(ctx context.Context) string {
	role, _ := ctx.Value(userRoleKey).(string)
	return role
}

func IsAdminCtx(ctx context.Context) bool {
	return UserRoleFromCtx(ctx) == "admin"
}
```

**Step 2: Update auth middleware to load and store user role**

The auth middleware currently only validates the JWT and extracts userID. It needs to also load the user's role from DB and store in context.

This requires the auth middleware to have access to a user-role lookup. Add a `roleLoader` interface:

```go
type roleLoader interface {
	GetUserRole(ctx context.Context, userID uuid.UUID) (string, error)
}
```

Update `Auth` function signature to accept roleLoader. After validating token, call `roleLoader.GetUserRole(ctx, userID)` and store via `ctxutil.WithUserRole`.

**Step 3: Create admin guard helper**

File: `backend_v4/internal/transport/middleware/admin.go`

```go
package middleware

import "github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
import "context"

// RequireAdmin returns domain.ErrForbidden if the context user is not admin.
// Use in resolver methods, not as HTTP middleware.
func RequireAdmin(ctx context.Context) error {
	if !ctxutil.IsAdminCtx(ctx) {
		return domain.ErrForbidden
	}
	return nil
}
```

**Step 4: Run tests and build**

```bash
go build ./...
go test ./pkg/ctxutil/... -race -count=1
```

**Step 5: Commit**

```bash
git commit -m "feat(auth): add admin role to context and RequireAdmin guard"
```

---

## Task 7: Migration — enrichment_queue table

**Files:**
- Create: `backend_v4/migrations/00017_enrichment_queue.sql`

**Step 1: Create migration**

```sql
-- +goose Up
CREATE TABLE enrichment_queue (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id   UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    status         TEXT NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    priority       INT NOT NULL DEFAULT 0,
    error_message  TEXT,
    requested_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_enrichment_queue_ref_entry ON enrichment_queue(ref_entry_id);
CREATE INDEX ix_enrichment_queue_status ON enrichment_queue(status, priority DESC, requested_at);

-- +goose Down
DROP TABLE IF EXISTS enrichment_queue;
```

Key design: `UNIQUE(ref_entry_id)` — each word appears once in queue. Upsert bumps priority on re-request.

**Step 2: Apply and verify**

```bash
make migrate-up
```

**Step 3: Commit**

```bash
git add migrations/00017_enrichment_queue.sql
git commit -m "feat(migration): add enrichment_queue table"
```

---

## Task 8: Enrichment queue domain types

**Files:**
- Create: `backend_v4/internal/domain/enrichment.go`

**Step 1: Add domain types**

```go
package domain

import (
	"time"
	"github.com/google/uuid"
)

// EnrichmentStatus represents the processing state of a queued word.
type EnrichmentStatus string

const (
	EnrichmentStatusPending    EnrichmentStatus = "pending"
	EnrichmentStatusProcessing EnrichmentStatus = "processing"
	EnrichmentStatusDone       EnrichmentStatus = "done"
	EnrichmentStatusFailed     EnrichmentStatus = "failed"
)

func (s EnrichmentStatus) IsValid() bool {
	switch s {
	case EnrichmentStatusPending, EnrichmentStatusProcessing, EnrichmentStatusDone, EnrichmentStatusFailed:
		return true
	}
	return false
}

// EnrichmentQueueItem represents a word queued for LLM enrichment.
type EnrichmentQueueItem struct {
	ID           uuid.UUID
	RefEntryID   uuid.UUID
	Status       EnrichmentStatus
	Priority     int
	ErrorMessage *string
	RequestedAt  time.Time
	ProcessedAt  *time.Time
	CreatedAt    time.Time
}

// EnrichmentQueueStats holds aggregate counts by status.
type EnrichmentQueueStats struct {
	Pending    int
	Processing int
	Done       int
	Failed     int
	Total      int
}
```

**Step 2: Build**

```bash
go build ./...
```

**Step 3: Commit**

```bash
git add internal/domain/enrichment.go
git commit -m "feat(domain): add EnrichmentQueueItem and EnrichmentStatus types"
```

---

## Task 9: Enrichment queue repository

**Files:**
- Create: `backend_v4/internal/adapter/postgres/enrichment/repo.go`
- Create: `backend_v4/internal/adapter/postgres/enrichment/query/enrichment.sql`
- Create: `backend_v4/internal/adapter/postgres/enrichment/sqlc.yaml`

**Step 1: Write SQL queries**

File: `query/enrichment.sql`

```sql
-- name: Enqueue :exec
INSERT INTO enrichment_queue (ref_entry_id, priority)
VALUES ($1, $2)
ON CONFLICT (ref_entry_id)
DO UPDATE SET priority = enrichment_queue.priority + 1,
              requested_at = now()
WHERE enrichment_queue.status IN ('pending', 'failed');

-- name: ClaimBatch :many
UPDATE enrichment_queue
SET status = 'processing'
WHERE id IN (
    SELECT id FROM enrichment_queue
    WHERE status = 'pending'
    ORDER BY priority DESC, requested_at
    LIMIT $1
    FOR UPDATE SKIP LOCKED
)
RETURNING id, ref_entry_id, status, priority, error_message, requested_at, processed_at, created_at;

-- name: MarkDone :exec
UPDATE enrichment_queue SET status = 'done', processed_at = now(), error_message = NULL
WHERE ref_entry_id = $1;

-- name: MarkFailed :exec
UPDATE enrichment_queue SET status = 'failed', processed_at = now(), error_message = $2
WHERE ref_entry_id = $1;

-- name: GetStats :one
SELECT
    count(*) FILTER (WHERE status = 'pending')    AS pending,
    count(*) FILTER (WHERE status = 'processing') AS processing,
    count(*) FILTER (WHERE status = 'done')        AS done,
    count(*) FILTER (WHERE status = 'failed')      AS failed,
    count(*)                                        AS total
FROM enrichment_queue;

-- name: List :many
SELECT id, ref_entry_id, status, priority, error_message, requested_at, processed_at, created_at
FROM enrichment_queue
WHERE ($1::text = '' OR status = $1)
ORDER BY priority DESC, requested_at
LIMIT $2 OFFSET $3;
```

**Step 2: Create sqlc.yaml**

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query/"
    schema: "../../../../migrations"
    gen:
      go:
        package: "sqlc"
        out: "sqlc"
        sql_package: "pgx/v5"
```

**Step 3: Generate and implement repo**

```bash
make generate
```

Write `repo.go` wrapping sqlc methods, implementing an interface matching the queries above.

**Step 4: Run tests**

```bash
go build ./...
```

**Step 5: Commit**

```bash
git commit -m "feat(repo): add enrichment queue repository with sqlc"
```

---

## Task 10: Enrichment service

**Files:**
- Create: `backend_v4/internal/service/enrichment/service.go`

**Step 1: Implement service**

The service wraps the enrichment repo and provides business logic:

```go
package enrichment

type queueRepo interface {
    Enqueue(ctx context.Context, refEntryID uuid.UUID, priority int) error
    ClaimBatch(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error)
    MarkDone(ctx context.Context, refEntryID uuid.UUID) error
    MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error
    GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error)
    List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error)
}

type Service struct {
    log   *slog.Logger
    queue queueRepo
}

func NewService(log *slog.Logger, queue queueRepo) *Service

// Enqueue adds a ref entry to the enrichment queue (idempotent, bumps priority if re-queued).
func (s *Service) Enqueue(ctx context.Context, refEntryID uuid.UUID) error

// ClaimBatch claims up to `limit` pending items for processing.
func (s *Service) ClaimBatch(ctx context.Context, limit int) ([]domain.EnrichmentQueueItem, error)

// MarkDone marks an item as successfully enriched.
func (s *Service) MarkDone(ctx context.Context, refEntryID uuid.UUID) error

// MarkFailed marks an item as failed with error message.
func (s *Service) MarkFailed(ctx context.Context, refEntryID uuid.UUID, errMsg string) error

// GetStats returns aggregate counts by status.
func (s *Service) GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error)

// List returns queue items filtered by status with pagination.
func (s *Service) List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error)
```

**Step 2: Write tests**

Test `Enqueue` idempotency, `ClaimBatch` with mocked repo, etc.

**Step 3: Build and test**

```bash
go test ./internal/service/enrichment/... -race -count=1
```

**Step 4: Commit**

```bash
git commit -m "feat(service): add enrichment queue service"
```

---

## Task 11: Wire enrichment queue triggers

When a user interacts with a ref entry (adds to dictionary or previews), enqueue the word.

**Files:**
- Modify: `backend_v4/internal/service/dictionary/create_from_catalog.go` — after creating entry, call `enrichment.Enqueue`
- Modify: `backend_v4/internal/service/dictionary/service.go` — add enrichment service dependency
- Modify: `backend_v4/internal/transport/graphql/resolver/dictionary.resolvers.go` — in `SearchCatalog` and `PreviewRefEntry`, enqueue viewed words
- Modify: `backend_v4/internal/app/app.go` — wire enrichment service

**Step 1: Add enrichment dependency to dictionary service**

In `dictionary/service.go`, add an optional enrichment interface:

```go
type enrichmentService interface {
    Enqueue(ctx context.Context, refEntryID uuid.UUID) error
}
```

Add field `enrichment enrichmentService` to `Service` struct. Update `NewService` to accept it.

**Step 2: Call Enqueue in CreateEntryFromCatalog**

In `create_from_catalog.go`, after the transaction succeeds (after line 161), fire-and-forget enqueue:

```go
// Best-effort: enqueue for enrichment (don't fail the request).
if s.enrichment != nil {
    go func() {
        if err := s.enrichment.Enqueue(context.Background(), input.RefEntryID); err != nil {
            s.log.Warn("enrichment enqueue failed", slog.String("error", err.Error()))
        }
    }()
}
```

**Step 3: Add enqueue trigger in resolvers for SearchCatalog and PreviewRefEntry**

In dictionary resolvers, after fetching ref entries, enqueue each returned entry ID (fire-and-forget, in goroutine).

**Step 4: Wire in app.go**

Create enrichment repo and service, inject into dictionary service and resolver.

**Step 5: Build and test**

```bash
go build ./...
go test ./internal/service/dictionary/... -race -count=1
```

**Step 6: Commit**

```bash
git commit -m "feat: wire enrichment queue triggers on catalog interaction"
```

---

## Task 12: Enhance LLM importer with replace semantics

Currently `BulkInsertSenses` uses `ON CONFLICT DO NOTHING`. For LLM enrichment, we need to **replace** existing senses/translations/examples for a ref_entry.

**Key architectural constraint:** User senses use `COALESCE(s.definition, rs.definition)` with `LEFT JOIN ref_senses`. User senses have `ref_sense_id` FK with `ON DELETE SET NULL`. So:
- If we DELETE ref_senses, user senses lose their link and show NULL definitions
- We need to UPDATE existing ref_senses in place where possible

**Files:**
- Modify: `backend_v4/internal/adapter/postgres/refentry/repo_bulk.go` — add `ReplaceEntryContent` method
- Modify: `backend_v4/internal/app/llm_importer/importer.go` — use replace instead of insert

**Step 1: Add ReplaceEntryContent to refentry repo**

```go
// ReplaceEntryContent replaces all senses, translations, and examples for a ref entry.
// Strategy: delete old child data, insert new. User senses with ref_sense_id pointing
// to deleted senses get SET NULL (by FK constraint), losing COALESCE link.
// This is acceptable because:
// 1. LLM data is strictly better quality
// 2. Users can re-add from updated catalog
func (r *Repo) ReplaceEntryContent(ctx context.Context, entryID uuid.UUID, senses []domain.RefSense, translations []domain.RefTranslation, examples []domain.RefExample) error
```

Implementation within a transaction:
1. `DELETE FROM ref_senses WHERE ref_entry_id = $1` (cascades to ref_translations, ref_examples)
2. Bulk insert new senses
3. Bulk insert new translations
4. Bulk insert new examples
5. Update `ref_entry_source_coverage` with source_slug='llm', status='fetched'

**Step 2: Update importer to use ReplaceEntryContent**

In `llm_importer/importer.go`, change the batch processing:
- For each `LLMWordEntry`, look up existing ref_entry by normalized text
- If exists: call `ReplaceEntryContent` with mapped senses/translations/examples
- If not exists: use existing `BulkInsert` flow
- Update `enrichment_queue` status to `done`

**Step 3: Add enrichment queue repo as dependency to importer**

Modify `llm_importer.Run` to accept an optional enrichment queue repo for updating status.

**Step 4: Build and test**

```bash
go build ./...
go test ./internal/app/llm_importer/... -race -count=1
```

**Step 5: Commit**

```bash
git commit -m "feat(importer): add replace semantics for LLM-enriched entries"
```

---

## Task 13: Enrichment CLI command enhancements

Enhance the enricher to accept words from the enrichment queue (via DB) in addition to a file.

**Files:**
- Modify: `backend_v4/internal/app/enricher/config.go` — add `Source` field ("file" or "queue")
- Modify: `backend_v4/internal/app/enricher/pipeline.go` — add queue-based word loading
- Modify: `backend_v4/cmd/enrich/main.go` — support queue mode

**Step 1: Add queue source mode**

When `Source=queue`, the enricher:
1. Connects to DB
2. Claims batch from enrichment_queue via `ClaimBatch(limit)`
3. Loads ref_entry texts for claimed items
4. Proceeds with existing enrichment pipeline
5. On completion, marks items done/failed

**Step 2: Build and test**

```bash
go build ./cmd/enrich/...
```

**Step 3: Commit**

```bash
git commit -m "feat(enricher): support queue-based word source"
```

---

## Task 14: Admin GraphQL schema + resolvers

**Files:**
- Create: `backend_v4/internal/transport/graphql/schema/admin.graphql`
- After `make generate`: modify generated resolver stubs

**Step 1: Add admin schema**

```graphql
# ============================================================
#  Admin — Enrichment Queue Management
# ============================================================

type EnrichmentQueueItem {
  id: UUID!
  refEntryId: UUID!
  refEntryText: String!
  status: String!
  priority: Int!
  errorMessage: String
  requestedAt: DateTime!
  processedAt: DateTime
}

type EnrichmentQueueStats {
  pending: Int!
  processing: Int!
  done: Int!
  failed: Int!
  total: Int!
}

type TriggerEnrichmentPayload {
  claimedCount: Int!
  outputDir: String!
}

type ImportEnrichedPayload {
  imported: Int!
  skipped: Int!
  errors: Int!
}

extend type Query {
  """Статистика очереди обогащения (admin only)."""
  enrichmentQueueStats: EnrichmentQueueStats!

  """Просмотр очереди обогащения (admin only)."""
  enrichmentQueue(status: String, limit: Int, offset: Int): [EnrichmentQueueItem!]!
}

extend type Mutation {
  """Запуск обогащения: забирает batchSize слов из очереди, генерирует prompt-файлы (admin only)."""
  triggerEnrichment(batchSize: Int!): TriggerEnrichmentPayload!

  """Импорт JSON результатов LLM из указанной директории (admin only)."""
  importEnrichedWords(directory: String!): ImportEnrichedPayload!
}
```

**Step 2: Generate resolvers**

```bash
make generate
```

**Step 3: Implement resolvers with admin guard**

Each admin resolver starts with:

```go
if err := middleware.RequireAdmin(ctx); err != nil {
    return nil, err
}
```

- `enrichmentQueueStats` → calls `enrichmentService.GetStats()`
- `enrichmentQueue` → calls `enrichmentService.List()`
- `triggerEnrichment` → calls `enrichmentService.ClaimBatch()`, then runs enricher pipeline
- `importEnrichedWords` → runs LLM importer with given directory

**Step 4: Wire in app.go**

Add `enrichmentService` to resolver, add enrichment repo to repo list.

**Step 5: Build and test**

```bash
make generate
go build ./...
```

**Step 6: Commit**

```bash
git commit -m "feat(graphql): add admin enrichment queue queries and mutations"
```

---

## Task 15: Admin REST endpoints for external cron

External cron needs HTTP endpoints (not GraphQL) for simpler integration.

**Files:**
- Create: `backend_v4/internal/transport/rest/admin.go`
- Modify: `backend_v4/internal/app/app.go` — register admin routes

**Step 1: Add admin REST handler**

```go
package rest

type AdminHandler struct {
    enrichment enrichmentService
    log        *slog.Logger
}

// POST /admin/enrichment/trigger  { "batch_size": 50 }
func (h *AdminHandler) TriggerEnrichment(w http.ResponseWriter, r *http.Request)

// POST /admin/enrichment/import   { "directory": "/path/to/llm-output" }
func (h *AdminHandler) ImportEnriched(w http.ResponseWriter, r *http.Request)

// GET /admin/enrichment/stats
func (h *AdminHandler) QueueStats(w http.ResponseWriter, r *http.Request)
```

**Step 2: Register routes in app.go**

Admin routes use full middleware chain (auth required) + admin role check inside handler.

```go
adminHandler := rest.NewAdminHandler(enrichmentService, logger)
adminChain := middleware.Chain(
    middleware.Recovery(logger),
    middleware.RequestID(),
    middleware.Logger(logger),
    middleware.CORS(cfg.CORS),
    middleware.Auth(authService),
)
mux.Handle("POST /admin/enrichment/trigger", adminChain(http.HandlerFunc(adminHandler.TriggerEnrichment)))
mux.Handle("POST /admin/enrichment/import", adminChain(http.HandlerFunc(adminHandler.ImportEnriched)))
mux.Handle("GET /admin/enrichment/stats", adminChain(http.HandlerFunc(adminHandler.QueueStats)))
```

**Step 3: Build**

```bash
go build ./...
```

**Step 4: Commit**

```bash
git commit -m "feat(rest): add admin REST endpoints for enrichment management"
```

---

## Task 16: Propagation verification

Verify that when ref_senses are replaced, user entries correctly reflect changes.

**Files:**
- Create: `backend_v4/tests/e2e/enrichment_propagation_test.go` (build tag: e2e)

**Step 1: Write E2E test**

Test scenario:
1. Seed a ref_entry with Wiktionary senses
2. Create a user entry from catalog (CreateEntryFromCatalog)
3. Verify user sees Wiktionary definitions via COALESCE
4. Replace ref_senses with LLM data (ReplaceEntryContent)
5. Query user's entry again
6. Verify: user senses now have `ref_sense_id = NULL` (FK SET NULL after delete)
7. Verify: user sees NULL definitions (since own fields are NULL and ref link broken)
8. Document this as known behavior — users who added words BEFORE enrichment need to re-sync

**Alternatively**, if we want seamless propagation, implement Task 16b.

**Step 2: Run E2E test**

```bash
make test-e2e
```

**Step 3: Commit**

```bash
git commit -m "test(e2e): add enrichment propagation verification"
```

---

## Task 16b (Optional): Re-link user senses after replacement

If seamless propagation is required (users automatically see LLM data without re-adding):

**Strategy:** After inserting new ref_senses, update user senses that were linked to old senses of the same ref_entry.

```sql
-- After inserting new ref_senses for ref_entry X:
-- Match user senses to new ref_senses by position
UPDATE senses s
SET ref_sense_id = new_ref.id
FROM (
    SELECT id, position FROM ref_senses WHERE ref_entry_id = $1
) new_ref
WHERE s.entry_id IN (SELECT id FROM entries WHERE ref_entry_id = $1)
  AND s.ref_sense_id IS NULL
  AND s.position = new_ref.position;
```

Add this to `ReplaceEntryContent` in `refentry/repo_bulk.go`.

---

## Task 17: Update GraphQL schema for User role

**Files:**
- Modify: `backend_v4/internal/transport/graphql/schema/user.graphql` — add `role` field to `User` type

**Step 1: Add role to User type**

```graphql
type User {
  id: UUID!
  email: String!
  username: String!
  name: String
  avatarUrl: String
  role: String!        # <-- add
  createdAt: DateTime!
  settings: UserSettings!
}
```

**Step 2: Regenerate**

```bash
make generate
```

**Step 3: Commit**

```bash
git commit -m "feat(graphql): expose user role in schema"
```

---

## Dependency Graph

```
Task 0 (commit reorg)
  └─→ Task 1 (filter wordlist) → Task 2 (run seeder)
  └─→ Task 3 (domain role) → Task 4 (migration role) → Task 5 (repo role) → Task 6 (middleware)
  └─→ Task 7 (migration queue) → Task 8 (domain queue) → Task 9 (repo queue) → Task 10 (service queue)
       └─→ Task 11 (wire triggers)
       └─→ Task 13 (enricher queue mode)
       └─→ Task 14 (admin GraphQL) → Task 15 (admin REST)
  └─→ Task 12 (importer replace) → Task 16 (propagation test) → Task 16b (re-link)
  └─→ Task 17 (user role in schema)
```

Three parallel tracks after Task 0:
1. **Data track**: Tasks 1 → 2 (Python filtering + seeder run)
2. **Admin track**: Tasks 3 → 4 → 5 → 6 → 17 (role infra)
3. **Queue track**: Tasks 7 → 8 → 9 → 10 → 11, 13, 14, 15 (enrichment pipeline)

Task 12 (importer replace) can start after Task 9 (queue repo).
Task 16 (propagation) requires Tasks 2 + 12.
