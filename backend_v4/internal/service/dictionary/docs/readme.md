# Package `dictionary`

> Implements the user-facing dictionary service: creating, searching, importing, exporting, and deleting personal dictionary entries. Each user maintains their own dictionary of words/phrases with senses, translations, examples, pronunciations, images, and optional flashcards for spaced repetition learning.

## Business Rules

### Entry Lifecycle

- Entries are created either **from a reference catalog** (pre-existing linguistic data) or as **custom entries** (user-authored). The source is tracked in audit logs as `"catalog"` or `"user"` respectively (`create_from_catalog.go:146`, `create_custom.go:52`).
- A user **cannot have duplicate entries** for the same normalized text. Duplicate checks happen before creation; concurrent creates are handled by catching `ErrAlreadyExists` after the transaction (`create_from_catalog.go:48-54`, `create_custom.go:43-49`).
- Entries are **soft-deleted** (not permanently removed). Soft-deleted entries can be listed via `FindDeletedEntries` and restored via `RestoreEntry` (`delete_entry.go:31`, `delete_entry.go:63`).
- The repo exposes `HardDeleteOld` with a configurable retention threshold, but the service does not call it directly - it is expected to run as a background/cron job.
- When creating from catalog, the user can **cherry-pick specific senses** by passing `SenseIDs`. If omitted, all senses from the reference entry are included (`create_from_catalog.go:57-74`).
- Catalog-sourced entries link pronunciations and images from the reference data; custom entries do not (`create_from_catalog.go:119-131`).

### Entry Limit

- Each user has a **maximum number of entries** enforced at creation time. The limit is checked before both single creates and bulk imports (`create_from_catalog.go:39-45`, `create_custom.go:35-41`, `import_entries.go:30-36`).
- For imports, the check is **optimistic**: `current_count + total_items > limit` is rejected upfront, before any items are processed (`import_entries.go:34`).

### Import

- Import is processed in **chunks**, each in its own transaction. If a chunk's transaction fails, all items in that chunk are rolled back and reported as errors, but previously committed chunks remain (`import_entries.go:49-167`).
- **Deduplication** is two-level: within the import file (in-memory `seen` map) and against existing DB entries (`import_entries.go:79-101`).
- If an imported item has translations, a **single sense** is created with all translations attached. No definition or part of speech is set for imported senses (`import_entries.go:126-137`).
- Items whose text normalizes to empty string are skipped (`import_entries.go:68-76`).

### Export

- Export returns **all entries** (up to `ExportMaxEntries`), sorted by `created_at ASC` (`export_entries.go:25-29`).
- Export uses batch loading: entries -> senses -> translations + examples -> cards. This avoids N+1 queries (`export_entries.go:42-93`).
- Each exported item includes the flashcard learning status if a card exists (`export_entries.go:105-108`).

### Searching & Pagination

- `FindEntries` supports **dual pagination**: cursor-based (when `Cursor` is provided) or offset-based (default) (`find_entries.go:64-104`).
- Search text is normalized before querying (`find_entries.go:29-34`).
- Default sort: `created_at DESC` (`find_entries.go:41-47`).
- Entries can be filtered by: text search, has-card flag, part of speech, topic, and learning status (`input.go:146-156`).

### Flashcards

- When creating an entry (catalog or custom), the user can opt-in to also create a **flashcard** by setting `CreateCard: true`. The card is created with status `LearningStatusNew` and the configured `DefaultEaseFactor` (`create_from_catalog.go:134-138`, `create_custom.go:94-98`).

### Auditing

- All mutating operations (create, update notes, delete) write an **audit record** inside the same transaction as the data change (`create_from_catalog.go:141-150`, `update_notes.go:45-53`, `delete_entry.go:35-44`).
- Batch delete writes a **single** audit record listing all successfully deleted entry IDs. Audit failure for batch delete is logged but does not fail the operation (`delete_entry.go:119-129`).
- Restore does **not** create an audit record (potential gap) (`delete_entry.go:57-69`).

### Authorization

- Every public method extracts `userID` from context and returns `ErrUnauthorized` if missing. All repository calls are scoped to that user (`search_catalog.go:16-18`, etc.).

### Validation

| Field / Rule | Constraint | Location |
|---|---|---|
| Entry text | required, max 500 chars | `input.go:65-69` |
| Entry notes | max 5000 chars | `input.go:134`, `input.go:207` |
| Senses per entry | max 20 | `input.go:71-73`, `input.go:25-27` |
| Definition per sense | max 2000 chars | `input.go:76-81` |
| Part of speech | must pass `IsValid()` | `input.go:82-87` |
| Translations per sense | max 20, each required & max 500 chars | `input.go:88-105` |
| Examples per sense | max 20, sentence required & max 2000 chars | `input.go:107-131` |
| Example translation | max 2000 chars | `input.go:125-130` |
| Import items | 1-5000 items, text required & max 500, translations max 20 per item | `input.go:234-258` |
| Batch delete IDs | 1-200 | `delete_entry.go:82-87` |
| Search catalog limit | clamped to 1-50, default 20 | `search_catalog.go:24` |
| Find entries limit | clamped to 1-200, default 20 | `find_entries.go:37` |
| Find deleted limit | clamped to 1-200, default 20 | `find_entries.go:134` |
| Sort by | allowed: `text`, `created_at`, `updated_at` | `input.go:163-168` |
| Sort order | allowed: `ASC`, `DESC` | `input.go:172-177` |
| Ref entry ID (catalog) | required (non-nil UUID) | `input.go:22-24` |
| Sense IDs (catalog) | max 20; each must exist in the ref entry | `input.go:25-27`, `create_from_catalog.go:67-73` |

## Configuration & Hardcoded Values

### Externally Configurable

| Parameter | Source | Default | Description |
|---|---|---|---|
| `MaxEntriesPerUser` | `DictionaryConfig` / `DICT_MAX_ENTRIES_PER_USER` | `10000` | Maximum dictionary entries a single user can have |
| `DefaultEaseFactor` | `DictionaryConfig` / `DICT_DEFAULT_EASE_FACTOR` | `2.5` | Initial ease factor for new flashcards (SM-2 algorithm) |
| `ImportChunkSize` | `DictionaryConfig` / `DICT_IMPORT_CHUNK_SIZE` | `50` | Number of items per transaction chunk during import |
| `ExportMaxEntries` | `DictionaryConfig` / `DICT_EXPORT_MAX_ENTRIES` | `10000` | Maximum entries returned in a single export |
| `HardDeleteRetentionDays` | `DictionaryConfig` / `DICT_HARD_DELETE_RETENTION_DAYS` | `30` | Days before soft-deleted entries are permanently purged |
| `AuditRetentionDays` | `DictionaryConfig` / `AUDIT_RETENTION_DAYS` | `365` | Days to retain audit records |

### Hardcoded / Internal Business Values

| Value | Location | Current Value | Meaning |
|---|---|---|---|
| Search catalog default limit | `search_catalog.go:24` | `20` | Default results when limit is 0 or negative |
| Search catalog max limit | `search_catalog.go:24` | `50` | Maximum catalog search results |
| Find entries default limit | `find_entries.go:37` | `20` | Default page size for entry listing |
| Find entries max limit | `find_entries.go:37` | `200` | Maximum page size for entry listing |
| Find deleted max limit | `find_entries.go:134` | `200` | Maximum page size for deleted entries |
| Batch delete max IDs | `delete_entry.go:85` | `200` | Maximum entries in a single batch delete |
| Import fallback chunk size | `import_entries.go:46` | `50` | Fallback if `ImportChunkSize` config is <= 0 |
| Custom entry source slug | `create_custom.go:52` | `"user"` | Source identifier for user-authored entries |
| Import source slug | `import_entries.go:38` | `"import"` | Source identifier for imported entries |
| Default sort field | `find_entries.go:42` | `"created_at"` | Default sort when none specified |
| Default sort order | `find_entries.go:45` | `"DESC"` | Default order when none specified |

## Public API

### Types

#### `Service`

The main dictionary service. All dependencies are injected via constructor. Not exported for direct field access.

#### `CreateFromCatalogInput`

Parameters for creating an entry from the reference catalog.

| Field | Type | Description |
|---|---|---|
| `RefEntryID` | `uuid.UUID` | Reference entry to copy from (required) |
| `SenseIDs` | `[]uuid.UUID` | Specific senses to include; empty = all |
| `CreateCard` | `bool` | Whether to also create a flashcard |
| `Notes` | `*string` | Optional user notes |

#### `CreateCustomInput`

Parameters for creating a user-authored entry.

| Field | Type | Description |
|---|---|---|
| `Text` | `string` | The word or phrase (required, max 500) |
| `Senses` | `[]SenseInput` | Senses with translations and examples |
| `CreateCard` | `bool` | Whether to also create a flashcard |
| `Notes` | `*string` | Optional user notes |
| `TopicID` | `*uuid.UUID` | Optional topic to associate with |

#### `SenseInput`

A single sense within a custom entry creation.

| Field | Type | Description |
|---|---|---|
| `Definition` | `*string` | Optional definition text |
| `PartOfSpeech` | `*domain.PartOfSpeech` | Optional part of speech |
| `Translations` | `[]string` | Translation texts |
| `Examples` | `[]ExampleInput` | Usage examples |

#### `ExampleInput`

| Field | Type | Description |
|---|---|---|
| `Sentence` | `string` | Example sentence (required) |
| `Translation` | `*string` | Optional translated sentence |

#### `FindInput`

Parameters for searching and paginating entries.

| Field | Type | Description |
|---|---|---|
| `Search` | `*string` | Free-text search (normalized before query) |
| `HasCard` | `*bool` | Filter: has/doesn't have a flashcard |
| `PartOfSpeech` | `*domain.PartOfSpeech` | Filter by part of speech |
| `TopicID` | `*uuid.UUID` | Filter by topic |
| `Status` | `*domain.LearningStatus` | Filter by flashcard learning status |
| `SortBy` | `string` | Sort field: `text`, `created_at`, `updated_at` |
| `SortOrder` | `string` | `ASC` or `DESC` |
| `Limit` | `int` | Page size (clamped 1-200, default 20) |
| `Cursor` | `*string` | Cursor for cursor-based pagination |
| `Offset` | `*int` | Offset for offset-based pagination |

#### `UpdateNotesInput`

| Field | Type | Description |
|---|---|---|
| `EntryID` | `uuid.UUID` | Entry to update (required) |
| `Notes` | `*string` | New notes value (nil = clear) |

#### `ImportInput` / `ImportItem`

| Field | Type | Description |
|---|---|---|
| `Items` | `[]ImportItem` | 1-5000 items to import |
| `ImportItem.Text` | `string` | Word/phrase text (required) |
| `ImportItem.Translations` | `[]string` | Optional translations |
| `ImportItem.Notes` | `*string` | Optional notes |
| `ImportItem.TopicName` | `*string` | Ignored in MVP |

#### `FindResult`

| Field | Type | Description |
|---|---|---|
| `Entries` | `[]domain.Entry` | Matching entries |
| `TotalCount` | `int` | Total matches (offset mode only) |
| `HasNextPage` | `bool` | Whether more results exist |
| `PageInfo` | `*PageInfo` | Start/end cursors |

#### `BatchResult` / `BatchError`

| Field | Type | Description |
|---|---|---|
| `Deleted` | `int` | Number successfully deleted |
| `Errors` | `[]BatchError` | Per-entry failures |

#### `ImportResult` / `ImportError`

| Field | Type | Description |
|---|---|---|
| `Imported` | `int` | Successfully imported count |
| `Skipped` | `int` | Skipped (duplicates, empty text, chunk failures) |
| `Errors` | `[]ImportError` | Per-item failure details with line numbers |

#### `ExportResult` / `ExportItem` / `ExportSense` / `ExportExample`

Fully denormalized export representation of the user's dictionary, including card status, senses, translations, and examples.

### Functions

**Constructor:**

| Function | Description | Errors |
|---|---|---|
| `NewService(logger, entries, senses, translations, examples, pronunciations, images, cards, audit, tx, refCatalog, cfg) *Service` | Creates a service instance. All dependencies injected; logger tagged with `"service": "dictionary"`. | -- |

**Catalog operations:**

| Function | Description | Errors |
|---|---|---|
| `SearchCatalog(ctx, query, limit) ([]RefEntry, error)` | Searches the reference catalog. Returns empty slice for empty query. Limit clamped to 1-50, default 20. | `ErrUnauthorized` |
| `PreviewRefEntry(ctx, text) (*RefEntry, error)` | Fetches or retrieves a reference entry by text. Delegates to `refCatalogService.GetOrFetchEntry`. | `ErrUnauthorized` |

**Entry creation:**

| Function | Description | Errors |
|---|---|---|
| `CreateEntryFromCatalog(ctx, input) (*Entry, error)` | Creates an entry from catalog data. Enforces entry limit, duplicate check, sense selection, links pronunciations/images, optionally creates card. Runs in transaction. Audit-logged. | `ErrUnauthorized`, `ErrAlreadyExists`, validation errors |
| `CreateEntryCustom(ctx, input) (*Entry, error)` | Creates a user-authored entry. Text is normalized; enforces entry limit and duplicate check. Source slug = `"user"`. Runs in transaction. Audit-logged. | `ErrUnauthorized`, `ErrAlreadyExists`, validation errors |

**Query operations:**

| Function | Description | Errors |
|---|---|---|
| `FindEntries(ctx, input) (*FindResult, error)` | Searches entries with filtering and dual pagination (cursor or offset). Normalizes search text, clamps limit. | `ErrUnauthorized`, validation errors |
| `GetEntry(ctx, entryID) (*Entry, error)` | Returns a single entry by ID, scoped to the authenticated user. | `ErrUnauthorized`, `ErrNotFound` |
| `FindDeletedEntries(ctx, limit, offset) ([]Entry, int, error)` | Lists soft-deleted entries. Limit clamped to 1-200. | `ErrUnauthorized` |

**Mutation operations:**

| Function | Description | Errors |
|---|---|---|
| `UpdateNotes(ctx, input) (*Entry, error)` | Updates entry notes. Captures old value for audit diff. Runs in transaction. Audit-logged. | `ErrUnauthorized`, `ErrNotFound`, validation errors |
| `DeleteEntry(ctx, entryID) error` | Soft-deletes an entry. Fetches entry text for audit. Runs in transaction. Audit-logged. | `ErrUnauthorized`, `ErrNotFound` |
| `RestoreEntry(ctx, entryID) (*Entry, error)` | Restores a soft-deleted entry. No audit record created. | `ErrUnauthorized`, `ErrNotFound` |
| `BatchDeleteEntries(ctx, entryIDs) (*BatchResult, error)` | Soft-deletes up to 200 entries. NOT transactional (partial failure OK). Single audit record for all successes. | `ErrUnauthorized`, validation errors |

**Bulk operations:**

| Function | Description | Errors |
|---|---|---|
| `ImportEntries(ctx, input) (*ImportResult, error)` | Imports 1-5000 items in chunks. Per-chunk transactions. Deduplicates within file and against DB. Checks entry limit upfront. | `ErrUnauthorized`, validation errors |
| `ExportEntries(ctx) (*ExportResult, error)` | Exports all entries with senses, translations, examples, and card status. Batch-loaded to avoid N+1. | `ErrUnauthorized` |

## Error Handling

| Error | Condition | Handling |
|---|---|---|
| `domain.ErrUnauthorized` | No user ID in context | Returned immediately, no operation performed |
| `domain.ErrAlreadyExists` | Duplicate entry (same normalized text for user) | Returned from pre-check or caught from tx unique constraint |
| `domain.ErrNotFound` | Entry or reference entry not found | Returned to caller |
| `domain.NewValidationError(...)` | Single field validation failure | Returned with field name and message |
| `domain.NewValidationErrors(...)` | Multiple validation failures | Returned with all field errors collected |
| `fmt.Errorf("...: %w", err)` | Internal/repo errors | Wrapped with operation context for stack tracing |

Batch delete audit errors are **logged but swallowed** -- they do not fail the batch operation (`delete_entry.go:126-129`).

## Known Limitations & TODO

- `ImportItem.TopicName` is declared but **ignored in MVP** (`input.go:227`).
- `RestoreEntry` does **not write an audit record**, unlike all other mutations -- potential audit gap.
- `BatchDeleteEntries` is **not transactional** -- partial deletes can occur with no rollback.
- `ExportEntries` is bounded by `ExportMaxEntries` but has no streaming/pagination -- large exports load everything into memory at once.
- `CreateEntryCustom` accepts `TopicID` in input but **never uses it** during creation (`input.go:44`).
