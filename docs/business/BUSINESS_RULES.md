# Business Rules

Rules are categorized by type and severity.

## Legend
- **Hard**: System enforces this — cannot be bypassed
- **Configurable**: Rule exists but thresholds/values are configurable
- **Informational**: Value exists for display/guidance but is NOT enforced
- ❓ **Inferred**: Not explicitly documented in code — derived from behavior

## Dictionary Rules

| # | Rule | Type | Code location |
|---|---|---|---|
| D1 | A user cannot have more than 10,000 dictionary entries | Configurable | `config.go:75` default, checked in `create_from_catalog.go:43` |
| D2 | No duplicate words per user (checked by normalized text) | Hard | `create_custom.go:44`, `create_from_catalog.go:48` |
| D3 | Entry text cannot exceed 500 characters | Hard | `dictionary/input.go` |
| D4 | Notes cannot exceed 5,000 characters | Hard | `dictionary/input.go` |
| D5 | Max 20 senses per entry | Hard | `dictionary/input.go` |
| D6 | Max 20 translations per sense | Hard | `dictionary/input.go` |
| D7 | Max 20 examples per sense | Hard | `dictionary/input.go` |
| D8 | Definitions cannot exceed 2,000 characters | Hard | `content/input.go` |
| D9 | Translation text: 1-500 characters | Hard | `content/input.go` |
| D10 | Example sentence: 1-2,000 characters | Hard | `content/input.go` |
| D11 | Entries are soft-deleted first, then hard-deleted after retention period | Configurable | `config.go:78`, default 30 days |
| D12 | Batch delete limited to 200 entries at once | Hard | `delete_entry.go:87` |
| D13 | Import limited to 5,000 items per batch | Hard | `dictionary/input.go` |
| D14 | Import chunk size (items per transaction) | Configurable | `config.go:76`, default 50 |
| D15 | User images must have valid HTTP(S) URL, 1-2,000 characters | Hard | `content/input.go` |
| D16 | Image captions limited to 500 characters | Hard | `content/input.go` |
| D17 | Reorder operations limited to 50 items, no duplicate IDs | Hard | `content/input.go` |

## Study / SRS Rules

| # | Rule | Type | Code location |
|---|---|---|---|
| S1 | New cards per day limit: default 20, user-configurable 1-999 | Configurable | `user/input.go`, `study_queue.go:45` |
| S2 | Due cards are ALWAYS shown, regardless of ReviewsPerDay setting | Hard | `study_queue.go:47-49` |
| S3 | ReviewsPerDay is informational only — NOT enforced in study queue | Informational | `domain/study.go:15`, `study_queue.go:48` |
| S4 | Only one active study session per user at a time | Hard | `session.go:38-46` |
| S5 | Starting a session is idempotent — returns existing if active | Hard | `session.go:31` |
| S6 | Cannot finish an already-finished session | Hard | `session.go:122-124` |
| S7 | Abandoning when no session exists is a silent no-op | Hard | `session.go:163-166` |
| S8 | Undo window: default 10 minutes after review | Configurable | `config.go:104`, `undo_review.go:52-55` |
| S9 | Only the last review can be undone (not arbitrary past reviews) | Hard | `undo_review.go:38` |
| S10 | A card cannot be created for an entry with zero senses | Hard | `card_crud.go:35-37` |
| S11 | One card per entry maximum (1:1 relationship) | Hard | Database constraint |
| S12 | Review duration capped at 600,000 ms (10 minutes) | Hard | `study/input.go` |
| S13 | Study queue default size: 50 cards, maximum: 200 | Hard | `study/input.go` |
| S14 | Batch card creation limited to 100 entries at once | Hard | `study/input.go` |
| S15 | Card difficulty clamped to range [1, 10] | Hard | `fsrs/algorithm.go:176-178` |
| S16 | Card stability has a floor of 0.1 (MinStability) | Hard | `fsrs/algorithm.go:11` |
| S17 | Review intervals clamped to [1 day, MaxIntervalDays] | Configurable | `fsrs/scheduler.go:338-346` |
| S18 | Interval ordering enforced: Hard ≤ Good < Easy | Hard | `fsrs/scheduler.go:253-262` |
| S19 | Lapse counter only increments on REVIEW → RELEARNING transition | Hard | `fsrs/scheduler.go:219` |
| S20 | Forget stability is capped to prevent exceeding pre-lapse stability | Hard | `fsrs/algorithm.go:148-152` |

## FSRS-5 Algorithm Parameters

| Parameter | Default | User adjustable? | Code location |
|---|---|---|---|
| Desired Retention | 0.9 (90%) | Yes, range 0.70-0.99 | `config.go:97`, `user/input.go` |
| Max Interval Days | 365 | Yes, range 1-36,500 | `config.go:98`, `user/input.go` |
| Enable Fuzz | true | No (server config only) | `config.go:99` |
| Learning Steps | 1min, 10min | No (server config only) | `config.go:100` |
| Relearning Steps | 10min | No (server config only) | `config.go:101` |
| Model Weights | FSRS-5 defaults (19 weights) | No (hardcoded) | `fsrs/algorithm.go:14-34` |

## Authentication Rules

| # | Rule | Type | Code location |
|---|---|---|---|
| A1 | Email must be unique across all users | Hard | Database constraint |
| A2 | Username must be unique across all users | Hard | Database constraint |
| A3 | Password: 8-72 characters | Hard | `auth/input.go` |
| A4 | Email stored lowercase and trimmed | Hard | `register.go:21`, `login_password.go:19` |
| A5 | Access token TTL: 15 minutes (default) | Configurable | `config.go:62` |
| A6 | Refresh token TTL: 30 days (default) | Configurable | `config.go:63` |
| A7 | Refresh token rotation: old token revoked on each refresh | Hard | `refresh.go:55` |
| A8 | Logout revokes ALL refresh tokens for the user | Hard | `logout.go:21` |
| A9 | OAuth account linking: if OAuth email matches existing account, link methods | Hard | `login.go:73-98` |
| A10 | JWT secret must be at least 32 characters | Hard | `config/validate.go` |
| A11 | Password hash cost: 4-31 (bcrypt range) | Configurable | `config.go:64` |
| A12 | OAuth code max length: 4,096 characters | Hard | `auth/input.go` |
| A13 | Refresh token max length: 512 characters | Hard | `auth/input.go` |
| A14 | Failed login returns generic "unauthorized" (prevents enumeration) | Hard | `login_password.go:29-48` |

## Rate Limiting Rules

| # | Rule | Type | Code location |
|---|---|---|---|
| R1 | Registration: 5 requests per minute per IP | Configurable | `config.go:33` |
| R2 | Login: 10 requests per minute per IP | Configurable | `config.go:34` |
| R3 | Token refresh: 20 requests per minute per IP | Configurable | `config.go:35` |
| R4 | Token bucket algorithm, per-IP enforcement | Hard | `middleware/ratelimit.go` |
| R5 | Returns HTTP 429 with Retry-After header when exceeded | Hard | `middleware/ratelimit.go` |

## Topic & Inbox Rules

| # | Rule | Type | Code location |
|---|---|---|---|
| T1 | Topic name: required, max 100 characters | Hard | `topic/input.go` |
| T2 | Topic description: optional, max 500 characters | Hard | `topic/input.go` |
| T3 | Batch link entries to topic: 1-200 entries at once | Hard | `topic/input.go` |
| T4 | Inbox item text: 1-500 characters | Hard | `inbox/input.go` |
| T5 | Inbox item context: optional, max 2,000 characters | Hard | `inbox/input.go` |
| T6 | Inbox list pagination: max 200 items per page | Hard | `inbox/input.go` |

## User Settings Rules

| # | Rule | Type | Code location |
|---|---|---|---|
| U1 | New cards per day: 1-999 | Hard | `user/input.go` |
| U2 | Reviews per day: 1-9,999 | Hard | `user/input.go` |
| U3 | Max interval days: 1-36,500 (100 years) | Hard | `user/input.go` |
| U4 | Desired retention: 0.70-0.99 | Hard | `user/input.go` |
| U5 | Timezone: must be valid IANA timezone | Hard | `user/input.go` |
| U6 | Profile name: required, max 255 characters | Hard | `user/input.go` |
| U7 | Avatar URL: optional, max 512 characters | Hard | `user/input.go` |

## Business Formulas

### Study Accuracy Rate
```
accuracy_rate = (good_count + easy_count) / total_reviews * 100
```
"Accuracy" counts only Good and Easy as successful recalls; Again and Hard are considered failures.

**Code**: `internal/service/study/dashboard.go:189`

### FSRS-5 Retrievability (Probability of Recall)
```
R(t, S) = (1 + t / (9 * S))^(-1)

where:
  t = elapsed days since last review
  S = stability (expected days until 90% recall probability)
```

### FSRS-5 Next Interval
```
I(S, r) = round(9 * S * (1/r - 1))

where:
  S = stability
  r = desired retention (e.g., 0.9)
```
Result clamped to [1 day, max_interval_days].

**Code**: `internal/service/study/fsrs/algorithm.go:57-65`

### Streak Calculation
```
Starting from today (or yesterday if today has no reviews):
  Count consecutive days backward where review_count > 0
  Break on the first gap day
```

**Code**: `internal/service/study/dashboard.go:215-242`
