# Package `study`

> Implements the spaced repetition study system (SRS) for a language-learning dictionary app. Manages flashcard lifecycle, review scheduling using an SM-2-like algorithm, study sessions, undo capability, and dashboard statistics.

## Business Rules

### Card Lifecycle (SRS State Machine)

Cards progress through four statuses: `NEW` -> `LEARNING` -> `REVIEW` -> `MASTERED`.

- **NEW -> LEARNING**: Any review of a NEW card transitions it to LEARNING (`srs.go`, `calculateNew()`)
- **LEARNING -> REVIEW**: Card graduates when it completes all learning steps with GOOD, or immediately on EASY (`srs.go`, `calculateLearning()`)
- **REVIEW -> MASTERED**: A card is promoted to MASTERED when its interval reaches >= 21 days AND ease factor >= 2.5 (`srs.go:233`)
- **REVIEW -> LEARNING (lapse)**: Grading AGAIN on a REVIEW/MASTERED card sends it back to LEARNING (relearning) (`srs.go:185-202`)
- **Relearning preserves ease**: During relearning, the ease factor is not reset to default but kept from the review state (`srs.go:151`, `srs.go:169`)

### SRS Calculation (Anki SM-2 Variant)

Ease factor adjustments on REVIEW/MASTERED cards:

| Grade | Ease Change | Interval Formula |
|---|---|---|
| AGAIN | -0.20 (floored at `MinEaseFactor`) | `currentInterval * LapseNewInterval` (min 1 day) |
| HARD | -0.15 (floored at `MinEaseFactor`) | `currentInterval * HardIntervalModifier` (min currentInterval+1) |
| GOOD | no change | `currentInterval * ease * IntervalModifier` (min currentInterval+1) |
| EASY | +0.15 | `currentInterval * ease * EasyBonus * IntervalModifier` (min currentInterval+1) |

- Intervals are capped at `min(SRSConfig.MaxIntervalDays, UserSettings.MaxIntervalDays)` (`review_card.go:39`)
- **Fuzz**: Intervals >= 3 days get deterministic jitter of +/-5% to prevent card clustering (`srs.go:259-272`). The jitter is based on the interval value itself (not randomness).

### New Card Scheduling

- NEW cards graded AGAIN or HARD start at learning step 0 (`srs.go:49-78`)
- NEW cards graded GOOD advance to step 1 (or graduate if only 1 step) (`srs.go:81-93`)
- NEW cards graded EASY graduate immediately with `EasyInterval` (`srs.go:96-97`)
- HARD delay on NEW cards is the average of steps[0] and steps[1] (`srs.go:66-67`)

### Study Queue

- Due cards are always served first, then new cards fill remaining slots (`study_queue.go:55-63`)
- New cards per day are limited by `UserSettings.NewCardsPerDay` minus new cards already reviewed today (`study_queue.go:46`)
- Due (overdue) cards are NOT limited by any daily cap — they are always shown (`study_queue.go:48-49`)
- Default queue limit is 50 cards when caller passes 0 (`study_queue.go:25-27`)
- "Today" is calculated in the user's timezone (`study_queue.go:37-38`)

### Card Creation

- An entry must have at least one sense to create a card (`card_crud.go:35-37`)
- Duplicate card per entry is prevented at the repo level (batch create skips existing) (`card_crud.go:180-185`)
- New cards start with status `NEW` and `DefaultEaseFactor` from SRS config (`card_crud.go:44`)

### Undo Review

- Only the most recent review of a card can be undone (`undo_review.go:34`)
- Undo is time-limited by `SRSConfig.UndoWindowMinutes` — if the review is older than this window, undo is rejected (`undo_review.go:48-51`)
- The previous card state (snapshot) must exist in the review log (`undo_review.go:43-45`)
- Undo deletes the review log entry and restores the card to its pre-review state within a transaction (`undo_review.go:56-100`)

### Study Sessions

- Only one ACTIVE session per user at a time (`session.go:40-48`)
- `StartSession` is idempotent: returns existing active session if one exists (`session.go:41-48`)
- Race condition on session creation is handled: if `ErrAlreadyExists`, it re-fetches (`session.go:64-75`)
- A session can only be finished if its status is ACTIVE (`session.go:105-107`)
- `AbandonSession` is idempotent: no error if no active session (`session.go:186-189`)
- **Accuracy rate** = `(GOOD + EASY) / totalReviewed * 100` (`session.go:146-147`)

### Dashboard & Streak

- Streak counts consecutive days with at least one review, starting from today or yesterday (`dashboard.go:228-254`)
- Streak looks back up to 365 days (`dashboard.go:59`)
- All "today" calculations respect user timezone (`dashboard.go:30-31`)
- Overdue count is currently hardcoded to 0 (not yet implemented) (`dashboard.go:82-83`)

### Validation

| Field / Rule | Constraint | Location |
|---|---|---|
| Queue limit | 0 - 200 | `input.go:17` |
| Card ID (review, undo, delete, history) | required (non-nil UUID) | `input.go:39`, `input.go:69`, `input.go:108`, `input.go:129` |
| Review grade | must be AGAIN, HARD, GOOD, or EASY | `input.go:42-43` |
| Review duration_ms | optional; if present: 0 - 600,000 (10 min) | `input.go:46-51` |
| Session ID (review) | optional | `input.go:52` |
| Entry ID (create card) | required (non-nil UUID) | `input.go:88` |
| Batch entry IDs | 1 - 100 entries | `input.go:153-157` |
| History limit | 0 - 200 | `input.go:131` |
| History offset | >= 0 | `input.go:134` |

## Configuration & Hardcoded Values

### Externally Configurable

| Parameter | Source | Description |
|---|---|---|
| `DefaultEaseFactor` | `SRSConfig` | Starting ease for new cards |
| `MinEaseFactor` | `SRSConfig` | Floor for ease after lapses |
| `MaxIntervalDays` | `SRSConfig` + `UserSettings` | Global + per-user cap; effective max = `min(both)` |
| `GraduatingInterval` | `SRSConfig` | Days after graduating from learning |
| `EasyInterval` | `SRSConfig` | Days when graduating via EASY |
| `LearningSteps` | `SRSConfig` | Duration steps for initial learning |
| `RelearningSteps` | `SRSConfig` | Duration steps for relearning after lapse |
| `IntervalModifier` | `SRSConfig` | Multiplier on GOOD reviews |
| `HardIntervalModifier` | `SRSConfig` | Multiplier on HARD reviews |
| `EasyBonus` | `SRSConfig` | Extra multiplier on EASY reviews |
| `LapseNewInterval` | `SRSConfig` | Interval multiplier after lapse |
| `UndoWindowMinutes` | `SRSConfig` | Time window to undo a review |
| `NewCardsPerDay` | `UserSettings` | Daily new card limit per user |
| `MaxIntervalDays` | `UserSettings` | Per-user max interval override |
| `Timezone` | `UserSettings` | User's timezone for day boundary |

### Hardcoded / Internal Business Values

| Value | Location | Current Value | Meaning |
|---|---|---|---|
| Default queue limit | `study_queue.go:26` | `50` | Queue size when caller passes 0 |
| Default history limit | `dashboard.go:124` | `50` | History page size when caller passes 0 |
| Mastered threshold (interval) | `srs.go:233` | `21` days | Min interval to qualify as MASTERED |
| Mastered threshold (ease) | `srs.go:233` | `2.5` | Min ease factor to qualify as MASTERED |
| Ease penalty for AGAIN | `srs.go:187` | `-0.20` | Ease reduction on lapse |
| Ease penalty for HARD | `srs.go:206` | `-0.15` | Ease reduction on hard review |
| Ease bonus for EASY | `srs.go:217` | `+0.15` | Ease increase on easy review |
| Fuzz threshold | `srs.go:261` | `3` days | Min interval before fuzz is applied |
| Fuzz range | `srs.go:263` | `5%` | Percentage of interval used for jitter |
| Fallback learning step | `srs.go:53` | `1 min` | Used when LearningSteps is empty |
| Fallback relearning step | `srs.go:193` | `10 min` | Used when RelearningSteps is empty |
| Max review duration | `input.go:49` | `600,000 ms` (10 min) | Cap on reported review time |
| Max queue/history limit | `input.go:17`, `input.go:131` | `200` | Max page size for queries |
| Max batch size | `input.go:155` | `100` | Max entries per batch card creation |
| Streak lookback | `dashboard.go:59` | `365` days | How far back to check for streaks |
| Overdue count | `dashboard.go:82` | `0` | Placeholder — not yet implemented |

## Public API

### Types

#### `Service`

The main service struct. All dependencies are injected via constructor. Logger is tagged with `"service": "study"`.

#### `SRSInput` / `SRSOutput`

Pure value types for the SRS calculation function. No side effects, no context.

**SRSInput fields:**
| Field | Type | Description |
|---|---|---|
| `CurrentStatus` | `LearningStatus` | Card's current status (NEW, LEARNING, REVIEW, MASTERED) |
| `CurrentInterval` | `int` | Current interval in days |
| `CurrentEase` | `float64` | Current ease factor |
| `LearningStep` | `int` | Index in learning/relearning steps |
| `Grade` | `ReviewGrade` | AGAIN, HARD, GOOD, or EASY |
| `Now` | `time.Time` | Current timestamp |
| `Config` | `SRSConfig` | Global SRS configuration |
| `MaxIntervalDays` | `int` | Effective max interval (min of config and user setting) |

**SRSOutput fields:**
| Field | Type | Description |
|---|---|---|
| `NewStatus` | `LearningStatus` | Resulting status after review |
| `NewInterval` | `int` | New interval in days |
| `NewEase` | `float64` | Updated ease factor |
| `NewLearningStep` | `int` | New learning step index |
| `NextReviewAt` | `time.Time` | Scheduled next review time |

#### `BatchCreateResult`

| Field | Type | Description |
|---|---|---|
| `Created` | `int` | Cards successfully created |
| `SkippedExisting` | `int` | Entries already having a card |
| `SkippedNoSenses` | `int` | Entries with zero senses |
| `Errors` | `[]BatchCreateError` | Per-entry error details |

#### Input Types

`GetQueueInput`, `ReviewCardInput`, `UndoReviewInput`, `CreateCardInput`, `DeleteCardInput`, `GetCardHistoryInput`, `BatchCreateCardsInput`, `FinishSessionInput` — all have `Validate()` methods returning `domain.ValidationErrors`.

### Functions

**Constructor:**

| Function | Description | Errors |
|---|---|---|
| `NewService(log, cards, reviews, sessions, entries, senses, settings, audit, tx, srsConfig) *Service` | Creates a study service. All dependencies injected. Logger tagged with `"service": "study"`. | -- |

**SRS (pure):**

| Function | Description | Errors |
|---|---|---|
| `CalculateSRS(SRSInput) SRSOutput` | Pure deterministic function. Dispatches to `calculateNew`, `calculateLearning`, or `calculateReview` based on current status. Applies fuzz to intervals >= 3 days. See Business Rules > SRS Calculation. | -- |

**Card operations:**

| Function | Description | Errors |
|---|---|---|
| `CreateCard(ctx, CreateCardInput) (*Card, error)` | Creates a card for an entry. Entry must exist and have >= 1 sense. Runs in tx. Audit-logged. | `ErrUnauthorized`, `ErrNotFound`, validation errors |
| `DeleteCard(ctx, DeleteCardInput) error` | Soft-deletes a card. Checks ownership. CASCADE deletes review logs. Runs in tx. Audit-logged. | `ErrUnauthorized`, `ErrNotFound`, validation errors |
| `BatchCreateCards(ctx, BatchCreateCardsInput) (BatchCreateResult, error)` | Creates cards for up to 100 entries. Partial success: skips entries without senses, skips duplicates, reports errors per entry. Each card created in its own tx. | `ErrUnauthorized`, validation errors |

**Review operations:**

| Function | Description | Errors |
|---|---|---|
| `ReviewCard(ctx, ReviewCardInput) (*Card, error)` | Records a review and updates SRS state. Snapshots pre-review state. MaxInterval = `min(config, userSettings)`. Runs in tx (card update + review log + audit). | `ErrUnauthorized`, `ErrNotFound`, validation errors |
| `UndoReview(ctx, UndoReviewInput) (*Card, error)` | Reverts the last review within the undo window. Restores card snapshot and deletes the review log. Runs in tx + audit. | `ErrUnauthorized`, `ErrNotFound`, validation errors (no reviews, no snapshot, window expired) |

**Study queue:**

| Function | Description | Errors |
|---|---|---|
| `GetStudyQueue(ctx, GetQueueInput) ([]*Card, error)` | Returns due cards first, then new cards up to user's daily limit. Respects user timezone for "today". Default limit 50. | `ErrUnauthorized`, validation errors |

**Session operations:**

| Function | Description | Errors |
|---|---|---|
| `StartSession(ctx) (*StudySession, error)` | Starts a session or returns existing active one (idempotent). Handles race conditions. | `ErrUnauthorized` |
| `FinishSession(ctx, FinishSessionInput) (*StudySession, error)` | Finishes an ACTIVE session. Aggregates review logs from session period into stats (accuracy, grade counts, duration). | `ErrUnauthorized`, `ErrNotFound`, validation errors (already finished) |
| `AbandonSession(ctx) error` | Abandons the active session. Idempotent noop if none active. | `ErrUnauthorized` |
| `GetActiveSession(ctx) (*StudySession, error)` | Returns current active session or nil if none. | `ErrUnauthorized` |

**Dashboard & stats:**

| Function | Description | Errors |
|---|---|---|
| `GetDashboard(ctx) (Dashboard, error)` | Aggregates: due count, new count, reviewed/new today, streak, status breakdown, active session ID. 7 repo calls. Timezone-aware. | `ErrUnauthorized` |
| `GetCardHistory(ctx, GetCardHistoryInput) ([]*ReviewLog, int, error)` | Paginated review history for a card. Checks ownership. Default limit 50. | `ErrUnauthorized`, `ErrNotFound`, validation errors |
| `GetCardStats(ctx, GetCardHistoryInput) (CardStats, error)` | Aggregated stats for a card: total reviews, accuracy rate, average time, grade distribution. Loads all review logs. | `ErrUnauthorized`, `ErrNotFound`, validation errors |

**Timezone helpers (exported):**

| Function | Description | Errors |
|---|---|---|
| `DayStart(now, tz) time.Time` | Returns midnight of `now` in given timezone, converted to UTC. | -- |
| `NextDayStart(now, tz) time.Time` | Returns midnight of next day in timezone (DST-safe via `AddDate`). | -- |
| `ParseTimezone(tz string) *time.Location` | Parses IANA timezone string; falls back to UTC on error. | -- |

## Error Handling

| Error | Condition | Handling |
|---|---|---|
| `domain.ErrUnauthorized` | No user ID in context | Returned immediately, no retry |
| `domain.ErrNotFound` | Card, session, or entry not found / not owned by user | Returned to caller |
| `domain.ErrAlreadyExists` | Race condition on session creation | Retried: re-fetches active session |
| `domain.ValidationErrors` | Input validation failed | Returned with per-field details |
| `domain.NewValidationError(field, msg)` | Business rule violation (no senses, session finished, undo expired) | Returned as single-field validation error |

Wrapping strategy: all repo/internal errors are wrapped with `fmt.Errorf("operation: %w", err)` for context.

## Known Limitations & TODO

- `TODO: Performance optimization - add CountByEntryIDs batch method to senseRepo interface` — `card_crud.go:194`. Batch card creation checks sense counts with N sequential queries instead of a single batch query.
- `TODO: Implement proper overdue calculation via cardRepo.CountOverdue(ctx, userID, dayStart)` — `dashboard.go:80`. Dashboard always returns `overdueCount = 0`.
- `GetDashboard` makes 7 sequential repo calls. No parallelization or caching.
- `GetCardStats` loads ALL review logs for a card into memory to compute stats. Could be expensive for heavily-reviewed cards.
- `applyFuzz` uses a deterministic formula based on interval value alone (`seed = interval`), meaning every card with the same interval gets the same fuzz offset. This limits the anti-clustering benefit.
