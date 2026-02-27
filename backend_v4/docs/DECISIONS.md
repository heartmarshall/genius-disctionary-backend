# Architecture Decisions

## ADR-001: GraphQL as Primary API, REST for Auth

**Status**: Active

**Context**: The app needs to serve rich, nested data (entries with senses, translations, examples, images, cards, topics). A mobile/web client would need many REST calls or bloated endpoints to fetch this in one go.

**Options considered**:
1. Pure REST -- simple, well-understood, but requires over-fetching or many round-trips for nested data
2. GraphQL -- flexible queries, single endpoint, but more complex server setup
3. Hybrid -- GraphQL for data-heavy queries, REST for simple auth flows

**Decision**: Hybrid. GraphQL (`POST /query`) for all dictionary, study, and user operations. REST (`POST /auth/*`) for authentication, because auth flows are simple request-response and don't benefit from GraphQL's flexibility.

**Trade-offs**: Two API styles to maintain. Auth errors use HTTP status codes while GraphQL errors use extension codes. DataLoaders add complexity but are necessary to prevent N+1 in nested resolvers.

---

## ADR-002: Layered Architecture with Interface-Based DI

**Status**: Active

**Context**: Need clear separation between business logic and infrastructure to enable unit testing and future adapter swaps.

**Options considered**:
1. Fat handlers (business logic in HTTP handlers) -- simple but untestable
2. Clean/hexagonal architecture with interfaces -- more boilerplate but testable
3. Framework-based (e.g., go-kit) -- structured but heavy

**Decision**: Layered hexagonal with unexported interfaces defined at the service level. Each service declares its own repo/provider interfaces (e.g., `userRepo`, `settingsRepo`), and adapters implement them. No framework.

**Trade-offs**: Each service has its own interface definitions rather than sharing -- some duplication, but services are fully decoupled. Mocks are generated per-service with `moq`. The `internal/app/app.go` file is a large wiring function (~300 lines) that manually connects all layers.

---

## ADR-003: Reference Catalog as Shared Immutable Data

**Status**: Active

**Context**: Users look up words and want standardized definitions. Fetching from FreeDictionary on every request is slow and rate-limited.

**Options considered**:
1. Fetch and embed data into each user's entry -- simple but duplicates data
2. Fetch once, store as immutable reference, link user entries to it -- normalized, shared
3. External cache (Redis) -- adds infrastructure complexity

**Decision**: Option 2. `ref_entry` + `ref_sense` + etc. tables hold immutable catalog data. User entries link via `ref_entry_id`. User senses can inherit from `ref_sense_id` or be fully custom.

**Trade-offs**: More complex data model (two parallel hierarchies: `ref_*` and user-owned). But storage is efficient, and catalog data is fetched once per word for all users. Translation provider is currently a stub, ready for future integration.

---

## ADR-004: Soft Delete with Cron-Based Hard Delete

**Status**: Active

**Context**: Users may accidentally delete entries and want to recover them. But old deleted data shouldn't accumulate forever.

**Options considered**:
1. Hard delete immediately -- simple, but no undo
2. Soft delete forever -- safe, but storage grows
3. Soft delete with time-boxed retention + cron cleanup -- best of both

**Decision**: Option 3. Entries get a `deleted_at` timestamp on delete. `cmd/cleanup` is a separate binary run via external cron that hard-deletes entries older than the configured retention period (default: 30 days). Same pattern for audit log pruning (default: 365 days).

**Trade-offs**: Cleanup is a separate process, not in-app. Queries must always filter `WHERE deleted_at IS NULL`. The retention period is configurable but lives in the main config file, not the cleanup binary.

---

## ADR-005: FSRS-5 for Spaced Repetition

**Status**: Active (updated from SM-2 to FSRS-5)

**Context**: Need a proven SRS algorithm for vocabulary study. Users expect Anki-like behavior.

**Options considered**:
1. SM-2 (original SuperMemo) -- proven, well-documented (originally chosen)
2. FSRS-5 (Free Spaced Repetition Scheduler) -- newer, more accurate, adopted by Anki
3. Simple fixed intervals -- easy but ineffective

**Decision**: Migrated from SM-2 to FSRS-5. Uses 19-parameter weight vector for stability/difficulty modeling, four grades (Again/Hard/Good/Easy), and configurable desired retention. Cards progress through NEW → LEARNING → REVIEW states. Implementation in `internal/service/study/fsrs/`.

**Trade-offs**: FSRS-5 provides better scheduling accuracy via retrievability modeling. Weights are configurable in `config.yaml` under `srs.fsrs_weights`. Learning/relearning steps remain configurable. Undo is supported by storing a `CardSnapshot` JSONB field in `review_log`.

---

## ADR-006: DataLoaders for N+1 Prevention

**Status**: Active

**Context**: GraphQL resolvers for nested types (entry → senses → translations) cause N+1 query problems when naively implemented.

**Options considered**:
1. Eager loading in repository (JOIN everything) -- fast but over-fetches when client only needs some fields
2. DataLoaders (batch + cache per request) -- lazy-loads only requested fields, batches by key
3. Query complexity limits only -- doesn't solve the problem, just limits damage

**Decision**: DataLoaders via `graph-gophers/dataloader/v7`. Eight loader types cover senses, translations, examples, pronunciations, images, cards, topics, and review logs. Injected as middleware, scoped per-request.

**Trade-offs**: Adds a middleware layer and per-request allocation. Batch functions must be implemented in repos alongside normal queries. Complexity limit (default: 300) is also set as a safety net.

---

## ADR-007: Audit Logging for Significant Mutations

**Status**: Active

**Context**: Need traceability for user data changes, especially settings and dictionary edits.

**Options considered**:
1. Application-level audit log -- flexible, captures business-meaningful diffs
2. Database triggers -- automatic but harder to include business context
3. Event sourcing -- full history but massive complexity

**Decision**: Application-level `audit_record` table. Services create audit records within the same transaction as the mutation. Changes stored as `map[string]any` with old/new values for changed fields.

**Trade-offs**: Audit logging is opt-in per service method -- some operations (like profile updates) are not audited. The `Changes` map is schemaless, which is flexible but means consumers must know the field names. Audit records are pruned by the same cleanup cron (configurable retention).
