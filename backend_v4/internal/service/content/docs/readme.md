# Package `content`

> Manages user-generated dictionary content — senses, translations, examples, and images — attached to dictionary entries. Every mutation enforces ownership, per-entity limits, input validation, and writes an audit trail, all within a database transaction.

## Business Rules

### Ownership

- Every operation extracts the user ID from the request context and verifies that the target entity belongs to that user **inside the transaction** (`sense.go:35`, `translation.go:34`, `example.go:40`, `userimage.go:40`).
- If the user ID is missing from context, all operations return `ErrUnauthorized` immediately.
- Ownership is checked via `GetByID`/`GetByIDForUser` calls that return `ErrNotFound` when the entity doesn't belong to the user, so unauthorized users receive the same error as for a missing resource (no information leakage).

### Entity Limits

Each parent can hold a bounded number of children:

| Child entity | Parent | Max count | Enforced in |
|---|---|---|---|
| Senses | Entry | 20 | `sense.go:45` |
| Translations | Sense | 20 | `translation.go:44` |
| Examples | Sense | 50 | `example.go:50` |
| User images | Entry | 20 | `userimage.go:50` |

Limits are checked by counting existing children **inside the transaction** before inserting. Exceeding the limit returns a validation error.

### Reordering

- Senses, translations, and examples support explicit position reordering.
- Reorder operations verify that every item in the request actually belongs to the specified parent entity; foreign IDs produce a validation error (`sense.go:228–229`, `translation.go:213`, `example.go:214`).
- Reorder items are capped at 50 per request, positions must be >= 0, and duplicate IDs are rejected (`input.go:432–457`).

### Inline Translations on Sense Creation

- `AddSense` accepts an optional `Translations` slice, creating the sense and its translations atomically in a single transaction (`sense.go:56–62`).

### Audit Logging

- **Creates** log the new entity fields.
- **Updates** compute a diff (old vs. new) and skip the audit record entirely if nothing changed (`sense.go:130–133`, `translation.go:110`, `example.go:120`, `userimage.go:119`).
- **Deletes** log the old values of the deleted entity.
- Translation/example audits are recorded against the **parent sense**, not the child entity itself (`translation.go:55`, `example.go:61`).
- User image audits are recorded against the **parent entry** (`userimage.go:61`).

### Source Slug

All created entities are tagged with `sourceSlug = "user"`, distinguishing user-created content from content imported from external sources (`sense.go:50`, `translation.go:49`, `example.go:55`).

### Validation

| Field / Input | Constraint | Location |
|---|---|---|
| `entry_id`, `sense_id`, `image_id`, `example_id`, `translation_id` | required (non-nil UUID) | all `Validate()` methods |
| Definition (sense) | optional; max 2000 runes | `input.go:37` |
| Part of speech | must pass `IsValid()` check (domain-defined enum) | `input.go:42` |
| CEFR level | must be one of: A1, A2, B1, B2, C1, C2 | `input.go:47` |
| Translation text | required, non-blank; max 500 runes | `input.go:157–161` |
| Inline translations (AddSense) | max 20 items; each non-blank, max 500 runes | `input.go:52–68` |
| Example sentence | required, non-blank; max 2000 runes | `input.go:239–243` |
| Example translation | optional; max 2000 runes | `input.go:247–249` |
| Image URL | required; max 2000 runes; must be valid HTTP(S) URL with host | `input.go:337–343` |
| Image caption | optional; max 500 runes | `input.go:346–349` |
| Reorder items | required, 1–50 items; no duplicate IDs; positions >= 0 | `input.go:428–457` |

All validation collects **all** field errors before returning (not fail-fast), using `domain.NewValidationErrors`. Text fields are whitespace-trimmed before persistence.

## Configuration & Hardcoded Values

### Externally Configurable

None. All dependencies are injected via the constructor; the package has no config struct or env variable reads.

### Hardcoded / Internal Business Values

| Value | Location | Current Value | Meaning |
|---|---|---|---|
| `MaxSensesPerEntry` | `service.go:16` | `20` | Max senses per dictionary entry |
| `MaxTranslationsPerSense` | `service.go:17` | `20` | Max translations per sense |
| `MaxExamplesPerSense` | `service.go:18` | `50` | Max examples per sense |
| `MaxUserImagesPerEntry` | `service.go:19` | `20` | Max user images per entry |
| Definition max length | `input.go:37` | `2000` runes | Inline literal, not a constant |
| Translation text max length | `input.go:161` | `500` runes | Inline literal |
| Example sentence max length | `input.go:243` | `2000` runes | Inline literal |
| Example translation max length | `input.go:249` | `2000` runes | Inline literal |
| Image URL max length | `input.go:339` | `2000` runes | Inline literal |
| Image caption max length | `input.go:349` | `500` runes | Inline literal |
| Reorder items max count | `input.go:433` | `50` | Inline literal |
| Source slug for user content | `sense.go:50` | `"user"` | Hardcoded string literal |

## Public API

### Types

#### `Service`

The main service struct. Holds injected dependencies (repos, audit logger, tx manager). All methods are on `*Service`.

#### Input Structs

| Type | Purpose |
|---|---|
| `AddSenseInput` | Create a sense with optional definition, POS, CEFR, and inline translations |
| `UpdateSenseInput` | Patch sense definition, POS, and/or CEFR (nil = no change) |
| `ReorderSensesInput` | Reorder senses within an entry |
| `AddTranslationInput` | Add a translation to a sense |
| `UpdateTranslationInput` | Update a translation's text |
| `ReorderTranslationsInput` | Reorder translations within a sense |
| `AddExampleInput` | Add an example sentence (with optional translation) to a sense |
| `UpdateExampleInput` | Update example sentence and/or translation (nil translation = remove) |
| `ReorderExamplesInput` | Reorder examples within a sense |
| `AddUserImageInput` | Add a user image (URL + optional caption) to an entry |
| `UpdateUserImageInput` | Update an image's caption |

Each input struct has a `Validate() error` method.

### Functions

**Constructor:**

| Function | Description | Errors |
|---|---|---|
| `NewService(logger, entries, senses, translations, examples, images, audit, tx) *Service` | Creates a service. All dependencies injected; logger tagged with `"service": "content"`. | — |

**Sense operations:**

| Function | Description | Errors |
|---|---|---|
| `AddSense(ctx, input) (*Sense, error)` | Creates a sense under an entry with optional inline translations. Enforces ownership, sense limit (max 20), runs in tx. Audit-logs creation. | `ErrUnauthorized`, validation errors |
| `UpdateSense(ctx, input) (*Sense, error)` | Patches definition, POS, and/or CEFR level. Nil fields skipped. Skips audit if nothing changed. | `ErrUnauthorized`, validation errors |
| `DeleteSense(ctx, senseID) error` | Deletes a sense; translations/examples cascade via FK. Audit-logs deletion. | `ErrUnauthorized`, not-found errors |
| `ReorderSenses(ctx, input) error` | Reorders senses. Validates all IDs belong to the entry. | `ErrUnauthorized`, validation errors |

**Translation operations:**

| Function | Description | Errors |
|---|---|---|
| `AddTranslation(ctx, input) (*Translation, error)` | Adds a translation to a sense. Enforces ownership, translation limit (max 20). Audits on parent sense. | `ErrUnauthorized`, validation errors |
| `UpdateTranslation(ctx, input) (*Translation, error)` | Updates translation text. Skips audit if unchanged. | `ErrUnauthorized`, validation errors |
| `DeleteTranslation(ctx, translationID) error` | Deletes a translation. Audits on parent sense. | `ErrUnauthorized`, not-found errors |
| `ReorderTranslations(ctx, input) error` | Reorders translations. Validates all IDs belong to the sense. | `ErrUnauthorized`, validation errors |

**Example operations:**

| Function | Description | Errors |
|---|---|---|
| `AddExample(ctx, input) (*Example, error)` | Adds an example to a sense. Enforces ownership, example limit (max 50). Audits on parent sense. | `ErrUnauthorized`, validation errors |
| `UpdateExample(ctx, input) (*Example, error)` | Updates sentence and/or translation. `translation=nil` removes the translation. Skips audit if unchanged. | `ErrUnauthorized`, validation errors |
| `DeleteExample(ctx, exampleID) error` | Deletes an example. Audits on parent sense. | `ErrUnauthorized`, not-found errors |
| `ReorderExamples(ctx, input) error` | Reorders examples. Validates all IDs belong to the sense. | `ErrUnauthorized`, validation errors |

**User image operations:**

| Function | Description | Errors |
|---|---|---|
| `AddUserImage(ctx, input) (*UserImage, error)` | Adds a user image to an entry. Enforces ownership, image limit (max 20). URL must be valid HTTP(S). Audits on parent entry. | `ErrUnauthorized`, validation errors |
| `UpdateUserImage(ctx, input) (*UserImage, error)` | Updates image caption. Skips audit if unchanged. | `ErrUnauthorized`, validation errors |
| `DeleteUserImage(ctx, imageID) error` | Deletes a user image. Audits on parent entry. | `ErrUnauthorized`, not-found errors |

## Error Handling

| Error | Condition | Handling |
|---|---|---|
| `domain.ErrUnauthorized` | User ID missing from context | Returned immediately, no tx started |
| `domain.ValidationErrors` | Input fails validation | All field errors collected, returned before any DB work |
| Not-found (from repos) | Entity doesn't exist or doesn't belong to user | Propagated from repo layer inside tx |
| `fmt.Errorf("...: %w", err)` | Repo call failures | Wrapped with operation context for debugging |

The package does not define its own sentinel errors — it relies on `domain.ErrUnauthorized` and `domain.NewValidationError` / `domain.NewValidationErrors`.

## Known Limitations & TODO

None identified. No `TODO`, `FIXME`, `HACK`, or `XXX` comments found in the source.

**Observations:**
- Field length limits (2000, 500 runes) are inline literals rather than named constants, making them harder to adjust centrally.
- The `UpdateExample` semantics where `translation=nil` means "remove translation" is a design choice worth documenting for API consumers, as it differs from the nil-means-no-change pattern used for sense fields.
- The reorder max item count (50) is also an inline literal, separate from the named `Max*` constants.
