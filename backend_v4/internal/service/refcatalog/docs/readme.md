# Package `refcatalog`

> Manages the shared reference dictionary catalog — a system-wide, immutable store of word entries fetched from external dictionary and translation providers. Provides search, get-by-ID, and a "get or fetch" operation that transparently populates the catalog on first access.

## Business Rules

### Catalog Immutability
- Reference entries are shared across all users and are never modified after creation. The catalog is append-only: once a word is fetched and saved, it stays as-is (`get_or_fetch.go`, `GetOrFetchEntry()`).

### Get-or-Fetch Lifecycle
- When a word is requested, the service first checks the local database by normalized text (`get_or_fetch.go:23`).
- If not found locally, it calls the **dictionary provider** for definitions, then the **translation provider** for translations (`get_or_fetch.go:32-52`).
- External HTTP calls happen **outside** the database transaction to avoid holding locks during network I/O (`get_or_fetch.go:59`).
- If the dictionary provider returns no result (`nil`), the word is considered not found and `ErrWordNotFound` is returned — no entry is created (`get_or_fetch.go:40-42`).

### Translation Graceful Degradation
- If the translation provider fails, the entry is still created **without translations**. A warning is logged but the operation succeeds (`get_or_fetch.go:45-52`). This means translations are best-effort, not mandatory.

### Translation Attachment Rule
- Translations are attached **only to the first sense** of an entry, regardless of how many senses exist (`mapper.go:49-58`). If the entry has zero senses, translations are silently discarded.

### Concurrent Insert Handling
- If two concurrent requests try to create the same word, the second one detects `ErrAlreadyExists` from the transaction and falls back to reading the existing entry (`get_or_fetch.go:67-73`). This makes the operation safe under concurrent access without distributed locks.

### Part-of-Speech Mapping
- Raw part-of-speech strings from the external provider are uppercased and validated against the domain enum (`mapper.go:78-89`).
- If the value is not recognized, it maps to `OTHER` rather than being rejected.
- If the provider sends `nil`, it stays `nil` (no part-of-speech assigned).

### Search Behavior
- An empty search query returns an empty result immediately — no database call is made (`search.go:12-14`).

### Validation

| Field / Rule | Constraint | Location |
|---|---|---|
| `text` in GetOrFetchEntry | must be non-empty after normalization (trim + lowercase + compress spaces) | `get_or_fetch.go:17-20` |
| Search query | empty string returns `[]` without DB call | `search.go:12-14` |
| Search limit | clamped to `[1, 50]`, `0` defaults to `20` | `service.go:58-66` |

## Configuration & Hardcoded Values

### Externally Configurable

None. All configuration is injected via constructor dependencies (logger, repos, providers, tx manager). The service itself exposes no config knobs.

### Hardcoded / Internal Business Values

| Value | Location | Current Value | Meaning |
|---|---|---|---|
| Default search limit | `service.go:60` | `20` | result count when caller passes `0` or negative |
| Max search limit | `service.go:62` | `50` | upper cap on search results per request |
| Dictionary source slug | `mapper.go:28` | `"freedict"` | source attribution for senses, examples, and pronunciations |
| Translation source slug | `mapper.go:55` | `"translate"` | source attribution for translations |
| Logger service tag | `service.go:49` | `"refcatalog"` | structured log field identifying this service |

## Public API

### Types

#### `Service`

The main service struct. Holds a logger and four injected dependencies: reference entry repository, transaction manager, dictionary provider, and translation provider. All fields are unexported; construction goes through `NewService`.

### Functions

**Constructor:**

| Function | Description | Errors |
|---|---|---|
| `NewService(logger, refEntries, tx, dictProvider, transProvider) *Service` | Creates a service instance. Logger is tagged with `"service": "refcatalog"`. All dependencies are injected. | — |

**Core operations:**

| Function | Description | Errors |
|---|---|---|
| `GetOrFetchEntry(ctx, text) (*RefEntry, error)` | Returns an existing entry or fetches it from external providers, saves it, and returns it. Text is normalized before lookup. Handles concurrent inserts via fallback read (see Concurrent Insert Handling). Translations degrade gracefully (see Translation Graceful Degradation). | `ValidationError` (empty text), `ErrWordNotFound` (provider returned nothing), wrapped repo/provider errors |
| `GetRefEntry(ctx, refEntryID) (*RefEntry, error)` | Returns a reference entry by UUID. Delegates directly to the repository's full-tree fetch. | Passes through repo errors (typically `ErrNotFound`) |
| `Search(ctx, query, limit) ([]RefEntry, error)` | Searches reference entries by text query. Empty query short-circuits to empty result. Limit is clamped (see Validation). | Passes through repo errors |

## Error Handling

| Error | Condition | Handling |
|---|---|---|
| `ErrWordNotFound` | Dictionary provider returned `nil` for the requested word | Returned to caller; no entry is created |
| `domain.ValidationError` | `text` is empty after normalization in `GetOrFetchEntry` | Returned to caller with field name `"text"` |
| `domain.ErrNotFound` | Entry not in DB (from repo, passed through by `GetRefEntry`) | Passed through to caller |
| `domain.ErrAlreadyExists` | Concurrent insert race in `GetOrFetchEntry` | Handled internally — falls back to reading existing entry |
| Provider errors | Dictionary or translation provider HTTP failures | Dictionary errors are wrapped and returned; translation errors are logged and swallowed (graceful degradation) |

Error wrapping strategy: `fmt.Errorf("context: %w", err)` with descriptive prefixes like `"get ref entry by text"`, `"fetch entry"`, `"create ref entry"`.

## Known Limitations & TODO

- **Translations only on first sense**: all translations from the provider are attached to sense #0. Multi-sense translation mapping is not implemented.
- **No cache invalidation or re-fetch**: once an entry is saved, there is no mechanism to refresh it if the external provider data improves or changes.
- **No rate limiting on external calls**: concurrent cache-miss requests for different words will each independently call external providers with no throttling.
- **Source slugs are hardcoded strings**: `"freedict"` and `"translate"` are embedded in the mapper, not derived from the actual provider configuration. If providers change, these strings must be manually updated.
- No `TODO`, `FIXME`, `HACK`, or `XXX` comments found in the package.
