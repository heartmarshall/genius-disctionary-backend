# SRS Comprehensive Fixes — Design Document

**Date:** 2026-02-28
**Scope:** All 30 issues from SRS audit across FSRS algorithm, DB, service, GraphQL layers

## Context

Full audit of the spaced repetition system identified 4 critical, 7 high, 19 medium/low issues.
No production users. No backward compatibility constraints. Frontend in development.

## Implementation Groups (dependency order)

### Group 1: FSRS Algorithm (pure logic, no I/O)

| # | Fix |
|---|-----|
| 3 | `reviewReview`: compute all stabilities with pre-update difficulty |
| 6 | Remove extra `Lapses++` in RELEARNING AGAIN |
| 12 | Add weight validation (NaN/Inf/range checks) |
| 13 | Remove duplicate `CardState` from fsrs package, use `domain.CardState` |
| 21 | Return error for unknown CardState instead of silent fallback to NEW |
| 29 | Fix `reviewNew` Easy: use Good stability for goodInterval baseline |
| 30 | Improve FuzzSeed with better hash function |

### Group 2: Database Migration + Repos

New migration `00019_srs_fixes.sql`:
- Add `user_id` column to `review_logs` (backfill from cards JOIN)
- Add CHECK constraints: `stability >= 0`, `difficulty BETWEEN 0 AND 10`
- Add partial index: `ix_cards_new_created ON cards(user_id, created_at) WHERE state = 'NEW'`
- Add index: `ix_review_logs_user_reviewed ON review_logs(user_id, reviewed_at)`
- Add `user_id` to `getByEntryIDsSQL`
- Change `UpdateCardSRS` to `:one` with `RETURNING *`

### Group 3: Service Layer (race conditions + logic)

| # | Fix |
|---|-----|
| 1 | UndoReview: move card + log reads inside tx with SELECT FOR UPDATE |
| 2 | ReviewCard: move card read inside tx with SELECT FOR UPDATE |
| 5 | Streak SQL: `AT TIME ZONE` for user timezone |
| 8 | FinishSession: wrap GetByPeriod + Finish in single tx |
| 17 | Dashboard: carry full StudySession in domain, not just UUID |
| 20 | Store actual ElapsedDays (not always 0) |

### Group 4: GraphQL Schema + Resolvers

| # | Fix |
|---|-----|
| 4 | `GradeDistribution` nullable or return zero-value struct |
| 9 | Add `newToday` to Dashboard type |
| 10 | Rename `averageDurationMs` → `totalDurationMs`, add missing fields |
| 11 | Add `currentState`, `stability`, `difficulty`, `scheduledDays` to CardStats |
| 18 | `finishStudySession` without required sessionId |
| 19 | `cardHistory` returns paginated wrapper with totalCount |
| 25 | Add `step`, `lastReview` to Card type |
| 26 | Split `batchCreateCards` skip reasons |
| 27 | Align studyQueue default limit |
| 28 | Fix error presenter `errors.Is`/`errors.As` asymmetry |

### Group 5: Low Priority

| # | Fix |
|---|-----|
| 23 | Migration down: add partial index filter back |
| 22 | Add comment + validation test for JSONB key dependency |

## Key Decisions

1. **Unified CardState**: `domain.CardState` is the single source of truth. FSRS package accepts string values.
2. **review_logs.user_id**: denormalization for query performance. Backfilled in migration via JOIN.
3. **SELECT FOR UPDATE**: used for card row locking in ReviewCard and UndoReview transactions.
4. **GradeDistribution**: return zero-value `GradeCounts{0,0,0,0}` instead of nil for no-review cards.
5. **Dashboard.ActiveSession**: change from `*uuid.UUID` to `*StudySession` in domain to avoid double-fetch.
