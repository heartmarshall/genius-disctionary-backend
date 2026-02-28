# Architecture Decision Records

## ADR-001: Hexagonal Architecture with Interface-Based DI

**Status**: Active

**Context**: The backend needs clear separation between business logic and infrastructure (database, HTTP, external APIs) to enable testing, swapping implementations, and reasoning about the codebase.

**Options considered**:
1. **Layered architecture** (controller → service → repo) — simple but creates tight coupling between layers
2. **Hexagonal/ports & adapters** — domain at center, interfaces as ports, implementations as adapters

**Decision**: Hexagonal architecture. Services define small, private interfaces for their dependencies (e.g., `type cardRepo interface { ... }` inside the service package). Implementations live in `internal/adapter/`. Wiring happens in `internal/app/app.go`.

**Trade-offs**: More boilerplate (interface definitions per service), but each service documents exactly what it needs. Mocking is trivial — no mock generation tools needed, just manual structs in tests. Services are fully testable without a database.

---

## ADR-002: FSRS-5 over SM-2 for Spaced Repetition

**Status**: Active (migrated from SM-2)

**Context**: The original SM-2 algorithm had known issues — poor handling of forgotten cards, fixed difficulty increments, no stability concept. Users reported suboptimal review scheduling.

**Options considered**:
1. **SM-2** — simple, well-known, but outdated (1987) with no memory stability model
2. **FSRS-5** — modern, research-backed algorithm with stability/difficulty/retrievability model and 19 calibrated weights

**Decision**: Migrated from SM-2 to FSRS-5. Implemented full algorithm in `internal/service/study/fsrs/` with pre-trained default weights. Card schema extended with `stability`, `difficulty`, `step`, `lapses`, `elapsed_days`, `scheduled_days`.

**Trade-offs**: More complex implementation and harder to reason about scheduling decisions. But significantly better learning outcomes — FSRS models actual memory decay rather than using fixed intervals. Users can tune `desired_retention` (default 0.9) to trade off review frequency vs. recall rate.

---

## ADR-003: GraphQL as Primary API with REST for Auth

**Status**: Active

**Context**: The frontend needs flexible data fetching — dictionary entries have deeply nested data (senses → translations → examples), and different views need different subsets.

**Options considered**:
1. **REST only** — simple, well-understood, but leads to over-fetching or many endpoints
2. **GraphQL only** — flexible queries, but auth token exchange over GraphQL is awkward
3. **Hybrid** — GraphQL for data operations, REST for auth and health

**Decision**: GraphQL (`POST /query`) for all dictionary, study, content, and organization operations. REST for auth (`/auth/*`), health probes (`/live`, `/ready`, `/health`), and admin endpoints (`/admin/*`).

**Trade-offs**: Two API styles to maintain. But GraphQL eliminates N+1 concerns with DataLoaders, gives the frontend exactly the data it needs, and the schema serves as live documentation. REST auth is simpler — no need for unauthenticated GraphQL mutations.

---

## ADR-004: sqlc + Squirrel Hybrid for Data Access

**Status**: Active

**Context**: Need type-safe database access without a heavy ORM, but also need dynamic query construction for filtering and pagination.

**Options considered**:
1. **ORM (GORM/Ent)** — high-level, but magic behavior, hard to optimize, poor for complex queries
2. **Raw SQL everywhere** — full control, but error-prone and tedious
3. **sqlc for static queries + Squirrel for dynamic queries** — best of both worlds

**Decision**: sqlc generates type-safe Go code from `.sql` files for standard CRUD. Squirrel builds dynamic queries for filtering (`WHERE` with optional clauses), cursor pagination, and complex aggregations. Each repo package has its own `sqlc.yaml`.

**Trade-offs**: Two query patterns to understand. Developers need to decide which tool fits each query. But sqlc catches SQL errors at generation time, and Squirrel prevents SQL injection in dynamic queries — both safer than raw string concatenation.

---

## ADR-005: Context-Based Transaction Propagation

**Status**: Active

**Context**: Services need to run multiple repository operations atomically. The transaction must flow through the call stack without repos knowing about each other.

**Options considered**:
1. **Explicit tx parameter** — `repo.Create(ctx, tx, params)` — clutters every signature
2. **Unit of Work pattern** — central object tracks changes, flushes at end — complex
3. **Context-based tx** — store `pgx.Tx` in `context.Context`, repos extract it transparently

**Decision**: `TxManager.RunInTx(ctx, fn)` begins a transaction, stores it in context, and passes the enriched context to `fn`. Repos call `QuerierFromCtx(ctx, pool)` which returns the transaction if present, otherwise the pool. Services compose transactional operations naturally.

**Trade-offs**: Transaction is implicit — harder to trace at a glance. Nested `RunInTx` calls don't create savepoints (would silently use the outer tx or start a separate one). But the pattern keeps repo interfaces clean and makes transactional composition effortless.

---

## ADR-006: Cursor-Based Pagination for Dictionary Entries

**Status**: Active

**Context**: Users may have thousands of entries. Offset pagination breaks when entries are added/deleted between pages (rows shift, duplicates or gaps appear).

**Options considered**:
1. **Offset pagination** (`LIMIT/OFFSET`) — simple, but unstable under concurrent modifications
2. **Cursor/keyset pagination** — stable, efficient, but more complex to implement

**Decision**: Cursor pagination for the main dictionary listing. Cursor encodes `base64(sortValue|entryID)` for keyset queries: `WHERE (sort_col, id) > (decoded_value, decoded_id)`. Offset pagination kept for simpler lists (inbox, deleted entries, admin views).

**Trade-offs**: Can't jump to arbitrary pages (no "page 5 of 20"). But entries don't shift between pages, and performance is consistent regardless of offset depth. The frontend uses infinite scroll, which aligns naturally with cursor-based loading.

---

## ADR-007: Soft Deletes with Time-Based Hard Deletion

**Status**: Active

**Context**: Users accidentally delete entries. Immediate hard deletion loses data permanently. But keeping soft-deleted rows forever bloats the database.

**Options considered**:
1. **Hard delete immediately** — simple, but no undo
2. **Soft delete forever** — safe, but table grows indefinitely
3. **Soft delete with retention window** — balance between safety and cleanup

**Decision**: Entries use `deleted_at` timestamp for soft delete. Soft-deleted entries are excluded from all normal queries but visible in a "trash" view. Users can restore within the retention window (default 30 days). A background job hard-deletes entries past retention.

**Trade-offs**: Every entry query needs `WHERE deleted_at IS NULL`. But users get undo capability, and the retention window prevents unbounded growth. Related data (senses, cards) is not soft-deleted — it's cleaned up on hard delete only.
