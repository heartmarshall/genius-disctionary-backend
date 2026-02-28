# Study Service Refactoring Design

**Date**: 2026-02-28
**Scope**: `backend_v4/internal/service/study/`
**Approach**: Decomposition into pure functions + conversion layer (single Service preserved)

## Problem Statement

The study module (~10,600 LOC, 16 public methods, 8 dependencies) has accumulated several architectural issues:

1. **Triple field duplication**: 10 identical fields copied manually between `domain.Card` → `fsrs.Card` → `domain.SRSUpdateParams` → `domain.CardSnapshot`
2. **Repeated boilerplate**: `userID extraction` (4 lines × 16 methods = 64 lines), tx+audit pattern (7 methods)
3. **Business logic buried in orchestration**: Session aggregation, batch filtering, streak calculation mixed with tx/repo/audit code
4. **BatchCreateCards uses N transactions**: One tx per card instead of single batch tx
5. **FSRS params scattered**: Parameters built from 3 sources (`fsrsWeights`, `settings`, `srsConfig`) inline
6. **No settings caching**: Same settings fetched repeatedly per request

## Design

### 1. Conversion Layer (`convert.go`)

New file with pure conversion functions:

```go
// cardToFSRS converts domain.Card to fsrs.Card for scheduling.
func cardToFSRS(card *domain.Card) fsrs.Card

// fsrsResultToUpdateParams converts fsrs.Card result to domain.SRSUpdateParams.
func fsrsResultToUpdateParams(result fsrs.Card) domain.SRSUpdateParams

// snapshotFromCard captures card SRS state before mutation.
func snapshotFromCard(card *domain.Card) *domain.CardSnapshot

// computeElapsedDays calculates days since last review.
func computeElapsedDays(lastReview *time.Time, now time.Time) int
```

**Impact**: Eliminates ~40 lines of manual field copying in review_card.go and undo_review.go. Single source of truth for field mapping.

### 2. Pure Functions for Business Logic

#### Session aggregation (`session.go`):
```go
// aggregateSessionResult computes session statistics from review logs.
func aggregateSessionResult(logs []*domain.ReviewLog, startedAt, now time.Time) domain.SessionResult
```
Extracts 30 lines from `finishSession` tx-callback. Testable without mocks.

#### Batch entry filtering (`card_crud.go`):
```go
// filterBatchEntries categorizes entries for batch card creation.
func filterBatchEntries(
    entryIDs []uuid.UUID,
    existMap, cardExistsMap map[uuid.UUID]bool,
    senseCounts map[uuid.UUID]int,
) (toCreate []uuid.UUID, skippedExisting, skippedNoSenses int, errors []BatchCreateError)
```
Extracts 3 filtering loops (~50 lines). Edge cases testable without mocks.

### 3. FSRS Parameters Builder

```go
// buildFSRSParams merges global SRS config with per-user settings.
func (s *Service) buildFSRSParams(settings *domain.UserSettings) fsrs.Parameters
```
Centralizes parameter construction from 3 sources.

### 4. UserID Helper

```go
// userID extracts the authenticated user's ID from context.
func (s *Service) userID(ctx context.Context) (uuid.UUID, error)
```
Reduces 4-line pattern to 3 lines across 16 methods. Single change point for auth logic.

### 5. BatchCreateCards — Single Transaction

Replace N individual transactions with one batch transaction:
```go
err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
    for _, entryID := range toCreate {
        card, createErr := s.cards.Create(txCtx, userID, entryID)
        if createErr != nil {
            result.Errors = append(result.Errors, ...)
            continue
        }
        result.Created++
        s.audit.Log(txCtx, ...)
    }
    return nil
})
```

**Trade-off**: Fatal DB error rolls back all cards. Acceptable — partial state without audit is worse.

## File Structure (After)

```
study/
├── service.go              # Service struct, deps, constructor (unchanged)
├── convert.go              # NEW: conversion functions, userID helper, buildFSRSParams
├── review_card.go          # SIMPLIFIED: uses convert.go
├── undo_review.go          # SIMPLIFIED: uses convert.go
├── card_crud.go            # REFACTORED: single tx for batch, uses filterBatchEntries
├── session.go              # SIMPLIFIED: uses aggregateSessionResult
├── study_queue.go          # minor: uses userID helper
├── study_queue_entries.go  # minor: uses userID helper
├── dashboard.go            # minor: uses userID helper
├── input.go                # unchanged
├── timezone.go             # unchanged
├── result.go               # unchanged
└── fsrs/                   # unchanged
```

## Expected Impact

- ~80-100 lines of boilerplate removed
- 3 pure functions testable without mocks
- Single source of truth for Card↔FSRS field mapping
- BatchCreateCards: 1 tx instead of N (up to 100x fewer DB round-trips)
- External API unchanged — fully backward-compatible

## What This Does NOT Address

- Settings caching (deferred — needs request-scoped cache infrastructure)
- God Service split into sub-services (user chose to keep single Service)
- Functional SRS bugs (separate effort in srs-comprehensive-fixes)
- TX+audit boilerplate across other services (project-wide concern)
