# Data Flow

## Entry Points

| Protocol | Path | Auth | Purpose |
|---|---|---|---|
| REST | `POST /auth/*` | Public | Registration, login, token refresh, logout |
| REST | `GET /live, /ready, /health` | Public | Health/readiness probes |
| GraphQL | `POST /query` | Bearer JWT | All dictionary, content, study, topic, inbox, user operations |
| GraphQL | `GET /` | Public | Playground (dev only) |
| CLI | `cmd/cleanup` | N/A (cron) | Hard-delete old entries + audit records |

---

## Flow 1: User Creates Entry from Catalog

The most common write path -- user looks up a word and adds it to their dictionary.

```mermaid
sequenceDiagram
    participant C as Client
    participant GQL as GraphQL
    participant Dict as Dictionary Svc
    participant RefCat as RefCatalog Svc
    participant FD as FreeDictionary API
    participant DB as PostgreSQL

    C->>GQL: mutation createEntryFromCatalog(text: "hello")
    GQL->>Dict: CreateEntryFromCatalog(ctx, input)
    Dict->>RefCat: GetOrFetchEntry(ctx, "hello")
    RefCat->>DB: SELECT ref_entry WHERE text_normalized = 'hello'
    alt Not cached
        RefCat->>FD: GET /api/v2/entries/en/hello
        FD-->>RefCat: definitions, phonetics, examples
        RefCat->>DB: INSERT ref_entry + senses + translations (tx)
    end
    RefCat-->>Dict: RefEntry
    Dict->>DB: INSERT entry (ref_entry_id = ...) (tx)
    Dict->>DB: INSERT senses (inherited from ref)
    Dict->>DB: INSERT card (NEW status, default ease)
    Dict->>DB: INSERT audit_record
    Dict-->>GQL: DictionaryEntry
    GQL-->>C: { entry, senses, card, ... }
```

---

## Flow 2: Study Session (Card Review)

User reviews flashcards with SRS algorithm deciding next review time.

```mermaid
sequenceDiagram
    participant C as Client
    participant GQL as GraphQL
    participant Study as Study Svc
    participant DB as PostgreSQL

    C->>GQL: mutation startStudySession
    GQL->>Study: StartSession(ctx)
    Study->>DB: INSERT study_session (ACTIVE)
    Study-->>C: session { id }

    C->>GQL: query studyQueue(limit: 10)
    GQL->>Study: GetStudyQueue(ctx, input)
    Study->>DB: SELECT due cards (next_review_at <= now)
    Study->>DB: SELECT new cards (status = NEW, up to daily limit)
    Study-->>C: [card1, card2, ...]

    loop Each card
        C->>GQL: mutation reviewCard(cardId, grade: GOOD)
        GQL->>Study: ReviewCard(ctx, input)
        Study->>Study: SRS algorithm (compute interval, ease, next_review)
        Study->>DB: UPDATE card (status, interval, ease, next_review_at) (tx)
        Study->>DB: INSERT review_log (grade, prev_state snapshot)
        Study-->>C: { card, reviewLog }
    end

    C->>GQL: mutation finishStudySession(input)
    GQL->>Study: FinishSession(ctx, input)
    Study->>DB: UPDATE study_session (FINISHED, result stats)
    Study-->>C: session { result { totalReviewed, accuracy } }
```

---

## Flow 3: Authentication (OAuth)

```mermaid
sequenceDiagram
    participant C as Client
    participant REST as Auth Handler
    participant Auth as Auth Svc
    participant Google as Google OAuth
    participant DB as PostgreSQL

    C->>REST: POST /auth/login { provider: "google", code: "..." }
    REST->>Auth: Login(ctx, input)
    Auth->>Google: Exchange code for user info
    Google-->>Auth: { email, name, avatarURL, providerID }
    Auth->>DB: SELECT auth_method WHERE provider_id = ...
    alt New user
        Auth->>DB: INSERT user (tx)
        Auth->>DB: INSERT auth_method
        Auth->>DB: INSERT user_settings (defaults)
    else Existing user
        Auth->>DB: UPDATE user (if profile changed)
    end
    Auth->>Auth: Generate JWT access token
    Auth->>Auth: Generate refresh token (random + SHA-256 hash)
    Auth->>DB: INSERT refresh_token (hash)
    Auth-->>REST: { accessToken, refreshToken, user }
    REST-->>C: 200 JSON response
```

---

## Processing Pipeline Summary

All write operations follow this pattern:

```
Request → Middleware (auth, logging) → Resolver → Service
    Service:
    1. Validate input (domain.ValidationError if bad)
    2. Check auth (ErrUnauthorized if missing)
    3. Check ownership (ErrForbidden if wrong user)
    4. Execute business logic (often in RunInTx)
    5. Create audit record (for significant mutations)
    6. Return result
```

## Error Flow

```mermaid
flowchart LR
    Service -->|domain error| Resolver
    Resolver -->|returns error| ErrPresenter[Error Presenter]
    ErrPresenter -->|maps to code| Client

    ErrPresenter -.->|ErrNotFound| NOT_FOUND
    ErrPresenter -.->|ErrValidation| VALIDATION
    ErrPresenter -.->|ErrUnauthorized| UNAUTHENTICATED
    ErrPresenter -.->|ErrForbidden| FORBIDDEN
    ErrPresenter -.->|unknown| INTERNAL
```

Validation errors include field-level details in the GraphQL error extensions.
