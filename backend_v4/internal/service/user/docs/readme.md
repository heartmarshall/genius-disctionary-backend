# Package `user`

> Manages authenticated user profiles and SRS (spaced repetition system) study settings. Provides read and partial-update operations with input validation, audit logging, and transactional consistency.

## Business Rules

### Authentication

- Every operation requires a valid user ID extracted from the request context (`ctxutil.UserIDFromCtx`). If missing, all endpoints return `ErrUnauthorized`. There is no service-level authorization beyond "is the caller authenticated" -- users can only access their own profile and settings.

### Profile Updates

- Name is always required when updating a profile; you cannot clear it (`profile.go:30`, `input.go:17`).
- Avatar URL is optional (nil = don't change). When provided, it is passed through to the repository as-is -- no URL format validation is performed (`profile.go:43`).
- Profile updates are **not** wrapped in a transaction and **not** audit-logged, unlike settings updates (`profile.go:30-53`).

### Settings Updates

- Settings use partial-update semantics: only non-nil fields in the input are applied; omitted fields keep their current values (`settings.go:97-114`).
- Settings updates run inside a transaction that: reads current settings, applies changes, persists, and creates an audit record -- all atomically (`settings.go:48-83`).
- The audit record captures old and new values for every changed field as a `map[string]any` diff (`settings.go:117-146`).

### Validation

| Field / Rule | Constraint | Location |
|---|---|---|
| `name` (profile) | required, max 255 chars | `input.go:17-21` |
| `avatar_url` (profile) | optional, max 512 chars | `input.go:23-25` |
| `new_cards_per_day` | optional, 1 -- 999 | `input.go:46-51` |
| `reviews_per_day` | optional, 1 -- 9,999 | `input.go:54-59` |
| `max_interval_days` | optional, 1 -- 36,500 (~100 years) | `input.go:62-67` |
| `timezone` | optional, non-empty, max 64 chars | `input.go:70-75` |

Note: timezone is validated only for presence and length -- no IANA timezone format check is performed.

## Configuration & Hardcoded Values

### Externally Configurable

None. This service receives all dependencies via constructor injection and has no external configuration of its own.

### Hardcoded / Internal Business Values

| Value | Location | Current Value | Meaning |
|---|---|---|---|
| Name max length | `input.go:19` | `255` | max characters for user display name |
| Avatar URL max length | `input.go:23` | `512` | max characters for avatar URL |
| New cards per day range | `input.go:47-50` | `1 -- 999` | allowed range for daily new card limit |
| Reviews per day range | `input.go:55-58` | `1 -- 9999` | allowed range for daily review limit |
| Max interval days range | `input.go:63-66` | `1 -- 36500` | allowed range for max SRS interval |
| Timezone max length | `input.go:73` | `64` | max characters for timezone string |
| Logger tag | `service.go:51` | `"user"` | slog service identifier |
| Audit entity type | `settings.go:72` | `EntityTypeUser` | entity type written to audit records |
| Audit action | `settings.go:74` | `AuditActionUpdate` | action type written to audit records |

Default settings values live in `domain.DefaultUserSettings()`, not in this package: `NewCardsPerDay=20`, `ReviewsPerDay=200`, `MaxIntervalDays=365`, `Timezone="UTC"`.

## Public API

### Types

#### `Service`

The main service struct. Holds injected dependencies (logger, repos, tx manager). All fields are unexported.

#### `UpdateProfileInput`

Input for profile updates.

| Field | Type | Description |
|---|---|---|
| `Name` | `string` | Required. New display name. |
| `AvatarURL` | `*string` | Optional. New avatar URL; nil = don't change. |

#### `UpdateSettingsInput`

Input for settings updates. All fields optional (nil = don't change).

| Field | Type | Description |
|---|---|---|
| `NewCardsPerDay` | `*int` | Daily new card limit for SRS. |
| `ReviewsPerDay` | `*int` | Daily review limit for SRS. |
| `MaxIntervalDays` | `*int` | Maximum interval between reviews. |
| `Timezone` | `*string` | User's timezone string. |

### Functions

**Constructors:**

| Function | Description | Errors |
|---|---|---|
| `NewService(logger, users, settings, audit, tx) *Service` | Creates a service instance. All dependencies injected; logger tagged with `"service": "user"`. | -- |

**Profile operations:**

| Function | Description | Errors |
|---|---|---|
| `GetProfile(ctx) (*domain.User, error)` | Returns the authenticated user's profile. Reads userID from context. | `ErrUnauthorized` |
| `UpdateProfile(ctx, input) (*domain.User, error)` | Validates input, updates name and optionally avatar. Not transactional, not audit-logged. | `ValidationError`, `ErrUnauthorized` |

**Settings operations:**

| Function | Description | Errors |
|---|---|---|
| `GetSettings(ctx) (*domain.UserSettings, error)` | Returns the authenticated user's SRS settings. Reads userID from context. | `ErrUnauthorized` |
| `UpdateSettings(ctx, input) (*domain.UserSettings, error)` | Validates input, applies partial changes inside a transaction, creates an audit record with old/new diffs. | `ValidationError`, `ErrUnauthorized` |

## Error Handling

| Error | Condition | Handling |
|---|---|---|
| `domain.ErrUnauthorized` | No user ID in request context | Returned directly, no wrapping |
| `*domain.ValidationError` | Input fails validation rules | Returned directly with field-level details; unwraps to `domain.ErrValidation` |
| Wrapped repo/tx errors | Repository or transaction failure | Wrapped with `fmt.Errorf("user.<Method>: %w", err)` pattern |

## Known Limitations & TODO

- No TODOs or FIXMEs in the codebase.
- **Profile updates are not audit-logged** while settings updates are. This asymmetry may be intentional (profile changes are lower risk) or an oversight.
- **No timezone format validation**: the `Timezone` field accepts any string up to 64 characters. Invalid IANA timezone names (e.g., "Mars/Olympus") will be stored without error.
- **No URL format validation** on `AvatarURL`: any string up to 512 characters is accepted.
- **UpdateProfile is not transactional**: it calls the repo directly without `RunInTx`, so there's no atomicity guarantee if multiple writes were ever added to this path.
