# Data Flow

## Main Request Lifecycle

Every request follows this path through the middleware chain before reaching business logic:

```mermaid
sequenceDiagram
    participant C as Client
    participant MW as Middleware Chain
    participant R as Resolver/Handler
    participant S as Service
    participant TX as TxManager
    participant DB as PostgreSQL

    C->>MW: HTTP Request
    MW->>MW: Recovery → RequestID → Logger → CORS
    MW->>MW: Auth: validate JWT → ctx(userID, role)
    MW->>MW: DataLoader: create per-request loaders
    MW->>R: Dispatch to resolver/handler

    R->>S: Call service method
    S->>TX: RunInTx(ctx, fn)
    TX->>DB: BEGIN
    S->>DB: Repo queries (via tx context)
    S->>DB: Audit log
    TX->>DB: COMMIT
    S-->>R: Return result or error

    R-->>C: JSON response
```

## Creating a Dictionary Entry from Catalog

The most complex write flow — creates an entry by linking to reference catalog data:

```mermaid
sequenceDiagram
    participant C as Client
    participant GQL as Resolver
    participant Dict as Dictionary Service
    participant TX as TxManager
    participant ER as EntryRepo
    participant SR as SenseRepo
    participant CR as CardRepo
    participant AR as AuditRepo

    C->>GQL: createEntryFromCatalog(refEntryID, senseIDs, createCard)
    GQL->>Dict: CreateEntryFromCatalog(ctx, input)

    Dict->>Dict: Validate input, normalize text
    Dict->>TX: RunInTx(ctx, fn)

    Note over TX,ER: Inside transaction
    TX->>ER: CountByUser(userID) — check limit
    TX->>ER: GetByText(normalized) — check duplicate
    TX->>ER: Create(entry) → Entry
    TX->>SR: CreateFromRef(senses) → []Sense
    alt createCard = true
        TX->>CR: Create(card, state=NEW) → Card
    end
    TX->>AR: Log(CREATE, entry)
    TX->>TX: COMMIT

    Dict-->>GQL: *Entry (with nested senses, card)
    GQL-->>C: CreateEntryPayload
```

## SRS Review Flow

How a card review updates FSRS-5 scheduling state:

```mermaid
sequenceDiagram
    participant C as Client
    participant GQL as Resolver
    participant Study as Study Service
    participant FSRS as FSRS-5 Scheduler
    participant TX as TxManager
    participant CR as CardRepo
    participant RR as ReviewLogRepo

    C->>GQL: reviewCard(cardID, grade: GOOD, durationMs)
    GQL->>Study: ReviewCard(ctx, input)

    Study->>TX: RunInTx(ctx, fn)
    TX->>CR: GetByIDForUpdate(cardID) — locks row
    Study->>Study: Load user SRS settings (retention, steps)
    Study->>FSRS: Schedule(card, grade, now)
    FSRS-->>Study: newState, nextDue, interval

    Study->>RR: Create(ReviewLog with CardSnapshot)
    Study->>CR: Update(card with new FSRS params)
    TX->>TX: COMMIT

    Study-->>GQL: Updated *Card
    GQL-->>C: ReviewCardPayload
```

## Authentication Flow

```mermaid
flowchart LR
    subgraph "Password Login"
        A1[POST /auth/login/password] --> A2[Validate email+password]
        A2 --> A3[bcrypt.Compare hash]
        A3 --> A4[Generate JWT + refresh token]
        A4 --> A5[Store refresh hash in DB]
        A5 --> A6[Return tokens + user]
    end

    subgraph "OAuth Login"
        B1[POST /auth/login] --> B2[Exchange code with Google]
        B2 --> B3[Fetch userinfo]
        B3 --> B4{User exists?}
        B4 -->|Yes| B5[Link OAuth method]
        B4 -->|No| B6[Create user + settings]
        B5 & B6 --> A4
    end

    subgraph "Token Refresh"
        C1[POST /auth/refresh] --> C2[Hash token, lookup in DB]
        C2 --> C3{Valid & not revoked?}
        C3 -->|Yes| C4[Revoke old, issue new pair]
        C3 -->|No| C5[401 Unauthorized]
    end
```

## GraphQL Field Resolution with DataLoaders

How nested data loads avoid N+1 queries:

```mermaid
sequenceDiagram
    participant GQL as gqlgen
    participant FR as Field Resolver
    participant DL as DataLoader
    participant Repo as SenseRepo

    Note over GQL: Query: { dictionary { entries { senses { translations } } } }

    GQL->>FR: Entry[0].Senses(entryID_1)
    FR->>DL: Load(entryID_1) — queued
    GQL->>FR: Entry[1].Senses(entryID_2)
    FR->>DL: Load(entryID_2) — queued
    GQL->>FR: Entry[2].Senses(entryID_3)
    FR->>DL: Load(entryID_3) — queued

    Note over DL: After 2ms wait window
    DL->>Repo: GetByEntryIDs([id_1, id_2, id_3])
    Repo-->>DL: []Sense (grouped by entryID)
    DL-->>FR: Distribute results to each caller
```

## Error Propagation

```
PostgreSQL error (e.g., unique constraint)
  ↓ Repository maps to domain error
domain.ErrAlreadyExists
  ↓ Service returns error
  ↓ Resolver receives error
  ↓ gqlgen ErrorPresenter maps to GraphQL error
GraphQL response: { errors: [{ message, extensions: { code: "ALREADY_EXISTS" } }] }

REST equivalent: { error: "already exists", code: "CONFLICT" } with HTTP 409
```

## Transaction Context Pattern

```
Service calls txm.RunInTx(ctx, fn)
  ↓ TxManager begins pgx.Tx
  ↓ Stores tx in context: ctx = withTx(ctx, tx)
  ↓ Passes ctx to callback fn(ctx)
    ↓ Repo calls QuerierFromCtx(ctx, pool)
    ↓ Finds tx in context → uses tx instead of pool
    ↓ All repo calls within fn share the same transaction
  ↓ fn returns nil → COMMIT
  ↓ fn returns error → ROLLBACK
```
