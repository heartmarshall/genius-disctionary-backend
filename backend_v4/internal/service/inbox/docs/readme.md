# Package `inbox`

> User-facing inbox for capturing quick text notes ("words to look up later"). Acts as a lightweight scratch pad where users jot down words or phrases with optional context, before processing them into proper dictionary entries.

## Business Rules

### Inbox Capacity

- Each user's inbox is capped at **500 items**. Creating a new item when the inbox is full returns a validation error (`create_item.go:33`).
- The count check happens before the insert, so the 500th item is allowed but the 501st is not.

### Item Creation

- The `text` field is trimmed before storage; leading/trailing whitespace is stripped (`create_item.go:26`).
- The optional `context` field is trimmed; if the trimmed result is empty, it is stored as `NULL` rather than an empty string (`service.go:44–52`, `trimOrNil`).
- Each created item gets a new UUID and a UTC timestamp assigned server-side (`create_item.go:38–42`).
- Creation is audit-logged with a text preview truncated to 50 characters (`create_item.go:47–56`).

### Item Access & Ownership

- Every operation extracts the user ID from the request context. If absent, `ErrUnauthorized` is returned immediately — there is no anonymous access.
- All repository calls scope queries by `userID`, so a user can only see and manage their own inbox items.

### Pagination

- If the client sends `limit=0`, the service defaults to **50** (`list_items.go:24–26`).
- The maximum allowed `limit` is **200** (`input.go:53–55`).

### Deletion

- Single-item delete requires a non-nil `ItemID` (`input.go:72`).
- `DeleteAll` removes every item for the user and returns the count of deleted rows (`delete_item.go:36–53`).
- Both delete operations are audit-logged with user ID and (for single delete) item ID.

### Validation

| Field / Rule | Constraint | Location |
|---|---|---|
| `text` | required, non-empty after trim | `input.go:20–23` |
| `text` length | max 500 characters (post-trim) | `input.go:24–26` |
| `context` length | max 2000 characters (post-trim) | `input.go:28–32` |
| `limit` | 0–200 (0 means "use default 50") | `input.go:50–55` |
| `offset` | must be non-negative | `input.go:56–58` |
| `item_id` | required (non-nil UUID) | `input.go:72–74` |

All validation collects every field error before returning, so clients receive all problems at once rather than one at a time.

## Configuration & Hardcoded Values

### Externally Configurable

None. The service receives only a logger and a repository — no config struct.

### Hardcoded / Internal Business Values

| Value | Location | Current Value | Meaning |
|---|---|---|---|
| `MaxInboxItems` | `service.go:13` | `500` | maximum items per user inbox |
| `DefaultLimit` | `service.go:14` | `50` | page size when client sends limit=0 |
| max `text` length | `input.go:24` | `500` chars | maximum length of the note text |
| max `context` length | `input.go:30` | `2000` chars | maximum length of the optional context |
| max `limit` | `input.go:53` | `200` | largest page the client can request |
| log text preview | `create_item.go:48` | `50` chars | truncation length for audit log preview |

## Public API

### Types

#### `Service`

Core service struct. Holds a repository reference and a tagged logger (`"service": "inbox"`).

#### `CreateItemInput`

| Field | Type | Description |
|---|---|---|
| `Text` | `string` | The word or phrase to save (required, max 500 chars) |
| `Context` | `*string` | Optional sentence or usage context (max 2000 chars) |

#### `ListItemsInput`

| Field | Type | Description |
|---|---|---|
| `Limit` | `int` | Page size (0 = default 50, max 200) |
| `Offset` | `int` | Number of items to skip (non-negative) |

#### `DeleteItemInput`

| Field | Type | Description |
|---|---|---|
| `ItemID` | `uuid.UUID` | ID of the item to delete (required) |

### Functions

**Constructor:**

| Function | Description | Errors |
|---|---|---|
| `NewService(log, inbox) *Service` | Creates the service. Logger tagged with `"service": "inbox"`. | — |

**Item operations:**

| Function | Description | Errors |
|---|---|---|
| `CreateItem(ctx, input) (*InboxItem, error)` | Creates an inbox item. Enforces auth, validates input, checks capacity limit (-> Inbox Capacity), trims fields, generates UUID + timestamp. Logs creation. | `ErrUnauthorized`, `ValidationError` (text/context/inbox full) |
| `ListItems(ctx, input) ([]*InboxItem, int, error)` | Returns a paginated list plus total count. Applies default limit if zero (-> Pagination). | `ErrUnauthorized`, `ValidationError` (limit/offset) |
| `GetItem(ctx, itemID) (*InboxItem, error)` | Fetches a single item by ID, scoped to the current user. | `ErrUnauthorized`, wrapped repo errors |
| `DeleteItem(ctx, input) error` | Deletes a single item. Validates input, scopes by user. Logs deletion. | `ErrUnauthorized`, `ValidationError` (item_id), wrapped repo errors |
| `DeleteAll(ctx) (int, error)` | Deletes all items for the user. Returns deleted count. Logs deletion. | `ErrUnauthorized`, wrapped repo errors |

## Error Handling

| Error | Condition | Handling |
|---|---|---|
| `ErrUnauthorized` | no user ID in context | returned directly, no wrapping |
| `ValidationError` | input fails validation or inbox full | returned with field-level details; unwraps to `ErrValidation` |
| repo errors | database failures | wrapped with context string (e.g., `"create inbox item: %w"`) |

Wrapping strategy: all repository errors are wrapped with `fmt.Errorf("{operation}: %w", err)` for debuggability. Validation and auth errors are returned unwrapped.

## Known Limitations & TODO

None identified in code comments.

Observations:
- `GetItem` does not use an input struct with validation — it takes a raw `uuid.UUID` directly, unlike the other operations which use typed input structs. This is a minor inconsistency but not a bug since the handler layer presumably validates the UUID before calling the service.
- No update/edit operation exists. Items are write-once: create, read, or delete. This is likely intentional (inbox items are meant to be quick captures moved elsewhere).
- No sorting options exposed — the list order depends entirely on the repository implementation.
