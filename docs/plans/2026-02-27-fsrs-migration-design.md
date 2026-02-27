# FSRS-5 Migration Design

**Date:** 2026-02-27
**Status:** Approved
**Approach:** Own FSRS-5 implementation (no external dependencies), complete SM-2 replacement

## Context

The MyEnglish app currently uses an SM-2 variant for spaced repetition with ease factor, learning steps, and 4 states (NEW/LEARNING/REVIEW/MASTERED). This design replaces it entirely with FSRS-5 (Free Spaced Repetition Scheduler), which uses a more accurate memory model based on Stability, Difficulty, and Retrievability.

No real users exist yet — this is a clean replacement with no backward compatibility needed.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| FSRS version | FSRS-5 (19 parameters) | Stable, well-documented, upgrade path to v6 later |
| Implementation | Own code, no external library | Full control, no dependency risk |
| Migration strategy | Full replacement | No users, no backward compat needed |
| Card states | Standard 4 FSRS states | New, Learning, Review, Relearning (no MASTERED) |
| Scheduler | Short-term + long-term | Learning steps for New/Relearning, FSRS formulas for Review |
| User settings | Minimal | DesiredRetention + NewCardsPerDay + ReviewsPerDay + MaxInterval + Timezone |
| Architecture | Separate `fsrs/` package | Isolated algorithm, easy to test and upgrade |

## Algorithm: FSRS-5

### Memory Model

- **Stability (S):** Time interval (in days) for recall probability to drop from 100% to 90%
- **Difficulty (D):** Inherent complexity of material, range [1, 10]
- **Retrievability (R):** Probability of successful recall at time t

### Parameters

19 weights (w0–w18) with default values:
```
[0.40255, 1.18385, 3.173, 15.69105, 7.1949, 0.5345, 1.4604, 0.0046,
 1.54575, 0.1192, 1.01925, 1.9395, 0.11, 0.29605, 2.2698, 0.2315,
 2.9898, 0.51655, 0.6621]
```

### Core Formulas

**Retrievability (forgetting curve):**
```
R(t, S) = (1 + factor * t/S) ^ decay
where decay = -0.5, factor = 19/81
```

**Next interval from desired retention r:**
```
I(r, S) = (S / factor) * (r^(1/decay) - 1)
       = (S * 81/19) * (r^(-2) - 1)
```

**Initial stability (first rating):**
```
S0(G) = w[G-1]   (G=1..4 maps to w0..w3)
```

**Initial difficulty:**
```
D0(G) = w4 - e^(w5 * (G-1)) + 1
```

**Difficulty update after review:**
```
ΔD = -w6 * (G - 3)
D' = D + ΔD * (10 - D) / 9        (linear damping)
D'' = w7 * D0(4) + (1-w7) * D'     (mean reversion)
D_final = clamp(D'', 1, 10)
```

**Stability after successful recall:**
```
S'r = S * (e^w8 * (11-D) * S^(-w9) * (e^(w10*(1-R)) - 1) * hardPenalty * easyBonus + 1)
where:
  hardPenalty = w15 if G == Hard, else 1
  easyBonus = w16 if G == Easy, else 1
```

**Stability after forgetting (lapse):**
```
S'f = w11 * D^(-w12) * ((S+1)^w13 - 1) * e^(w14*(1-R))
```

**Same-day stability (short-term):**
```
S'_short = S * e^(w17 * (G - 3 + w18))
```

### State Transitions

**Short-term scheduler (Learning / Relearning):**
- Uses fixed learning steps (default: [1m, 10m]) and relearning steps (default: [10m])
- Again → reset to step 0, due = now + steps[0]
- Hard → repeat current step, due = now + steps[step]
- Good → advance step; if at end, graduate to Review
- Easy → graduate to Review immediately
- On graduation: calculate S and D using FSRS formulas, set interval via I(r, S)

**Long-term scheduler (Review):**
- Again (lapse) → Relearning, lapses++, S = S'f, update D
- Hard → Review, S = S'r (with hardPenalty), update D
- Good → Review, S = S'r, update D
- Easy → Review, S = S'r (with easyBonus), update D
- Interval = I(desired_retention, S), capped at maxInterval, fuzzed ±5% for intervals ≥ 3 days

**New cards:**
- First review calculates S0(G) and D0(G)
- If short-term enabled: enter Learning with steps
- If long-term only: go directly to Review with calculated interval

## Domain Models

### Card (replaces current)

```go
type Card struct {
    ID            uuid.UUID
    UserID        uuid.UUID
    EntryID       uuid.UUID
    State         CardState     // New(0), Learning(1), Review(2), Relearning(3)
    Step          int           // current learning/relearning step index
    Stability     float64       // S
    Difficulty    float64       // D [1, 10]
    Due           time.Time     // when card should be reviewed
    LastReview    *time.Time    // last review timestamp
    Reps          int           // total successful reviews
    Lapses        int           // total lapses (Again in Review)
    ScheduledDays int           // planned interval in days
    ElapsedDays   int           // days since last review
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

### CardSnapshot (for undo)

```go
type CardSnapshot struct {
    State         CardState  `json:"state"`
    Step          int        `json:"step"`
    Stability     float64    `json:"stability"`
    Difficulty    float64    `json:"difficulty"`
    Due           time.Time  `json:"due"`
    LastReview    *time.Time `json:"last_review"`
    Reps          int        `json:"reps"`
    Lapses        int        `json:"lapses"`
    ScheduledDays int        `json:"scheduled_days"`
    ElapsedDays   int        `json:"elapsed_days"`
}
```

### UserSettings changes

```go
type UserSettings struct {
    UserID           uuid.UUID
    NewCardsPerDay   int       // default: 20
    ReviewsPerDay    int       // default: 200
    MaxIntervalDays  int       // default: 365
    DesiredRetention float64   // default: 0.9, range [0.7, 0.97]
    Timezone         string    // default: "UTC"
    UpdatedAt        time.Time
}
```

## Database Migration

### Cards table

Remove SM-2 columns (ease_factor, learning_step, next_review_at, interval_days, status/learning_status), add FSRS columns (state, step, stability, difficulty, due, last_review, reps, lapses, scheduled_days, elapsed_days).

New index: `(user_id, state, due)`.

### User settings table

Add `desired_retention FLOAT NOT NULL DEFAULT 0.9`.

### Review logs table

No schema change — `prev_state` JSONB format changes to store FSRS fields.

## Package Structure

```
internal/service/study/
  fsrs/
    algorithm.go       — FSRS-5 formulas (pure functions)
    scheduler.go       — BasicScheduler (short-term) + LongTermScheduler
    parameters.go      — Parameters struct, DefaultWeights, constants
    fuzz.go            — interval fuzzing (±5% for intervals ≥ 3 days)
    algorithm_test.go  — formula validation against reference values
    scheduler_test.go  — state transition scenarios
  service.go           — study service (uses fsrs/ package)
  review_card.go       — ReviewCard (calls fsrs scheduler)
  study_queue.go       — GetStudyQueue (due + new cards)
  dashboard.go         — Dashboard, CardStats, CardHistory
  session.go           — session management (unchanged logic)
  card_crud.go         — CreateCard, DeleteCard, BatchCreateCards
  undo_review.go       — UndoReview (snapshot-based, unchanged logic)
```

Old `srs.go` is deleted entirely.

## GraphQL Schema Changes

- `LearningStatus` enum → `CardState` enum (NEW, LEARNING, REVIEW, RELEARNING)
- Card type: remove `easeFactor`, `nextReviewAt`, `intervalDays`; add `stability`, `difficulty`, `due`, `reps`, `lapses`, `scheduledDays`
- UserSettings: add `desiredRetention`
- CardStatusCounts: 4 states instead of previous

## Config Changes

```go
type SRSConfig struct {
    Weights           [19]float64
    DefaultRetention  float64         // default: 0.9
    MaxIntervalDays   int             // default: 365
    EnableFuzz        bool            // default: true
    LearningSteps     []time.Duration // default: [1m, 10m]
    RelearningSteps   []time.Duration // default: [10m]
    NewCardsPerDay    int             // default: 20
    ReviewsPerDay     int             // default: 200
    UndoWindowMinutes int             // default: 10
}
```

Removed SM-2 params: DefaultEaseFactor, MinEaseFactor, GraduatingInterval, EasyInterval, IntervalModifier, HardIntervalModifier, EasyBonus, LapseNewInterval.

## Testing Strategy

1. **Unit tests for `fsrs/` package:** Validate all formulas against known reference values. Test boundary conditions (D=1, D=10, S near 0, R near 0/1).
2. **Scheduler scenario tests:** Full card lifecycle through all state transitions.
3. **Service tests:** Updated to use new Card fields.
4. **E2E tests:** Updated assertions for new field names and values.

## What Stays the Same

- Undo mechanism (snapshot-based, same logic, different fields)
- Session management (start/finish/abandon)
- Study queue logic (due cards + new cards with daily limits)
- Dashboard (streak, reviewed today, status counts)
- Review logging
- Audit logging
- All error handling (sentinel errors)
