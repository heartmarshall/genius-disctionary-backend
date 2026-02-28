# SRS Bugfixes & Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix 2 critical bugs (desiredRetention not mapped/validated), eliminate N+1 in StudyQueue, and document the ReviewsPerDay design decision.

**Architecture:** All changes follow the existing hexagonal pattern — service input struct → validation → apply/audit → resolver mapping. The N+1 fix uses the existing `entryRepo.GetByIDs()` batch method exposed at study service level.

**Tech Stack:** Go 1.23, gqlgen, sqlc, testify, pgx

---

### Task 1: Add `DesiredRetention` to `UpdateSettingsInput` struct + validation

**Files:**
- Modify: `backend_v4/internal/service/user/input.go:36-42` (add field to struct)
- Modify: `backend_v4/internal/service/user/input.go:45-86` (add validation rules)
- Test: `backend_v4/internal/service/user/input_test.go`

**Step 1: Write failing tests for desiredRetention validation**

Add these test cases to the existing `TestUpdateSettingsInput_Validate` table in `input_test.go` (after the Timezone tests, before the "all nil" case):

```go
// DesiredRetention boundaries
{
    name:    "valid: desired_retention at min (0.70)",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(0.70)},
    wantErr: false,
},
{
    name:    "valid: desired_retention at max (0.99)",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(0.99)},
    wantErr: false,
},
{
    name:    "valid: desired_retention at 0.9 (default)",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(0.9)},
    wantErr: false,
},
{
    name:    "invalid: desired_retention below min (0.69)",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(0.69)},
    wantErr: true,
},
{
    name:    "invalid: desired_retention above max (1.0)",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(1.0)},
    wantErr: true,
},
{
    name:    "invalid: desired_retention zero",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(0.0)},
    wantErr: true,
},
{
    name:    "invalid: desired_retention negative",
    input:   UpdateSettingsInput{DesiredRetention: ptrF(-0.5)},
    wantErr: true,
},
```

Also add the `ptrF` helper if it doesn't exist (the existing `ptr` is generic so `ptr(0.9)` works — use `ptr(0.9)` instead of `ptrF`). Actually, `ptr[T any]` is defined in `service_test.go` but NOT in `input_test.go`. It is used in input_test.go already though (line 90: `ptr(1)`), so it must be accessible within the package. Use `ptr(0.70)` etc.

Also update `TestUpdateSettingsInput_Validate_MultipleErrors` to include desiredRetention:

```go
func TestUpdateSettingsInput_Validate_MultipleErrors(t *testing.T) {
    t.Parallel()

    input := UpdateSettingsInput{
        NewCardsPerDay:   ptr(0),
        ReviewsPerDay:    ptr(0),
        MaxIntervalDays:  ptr(0),
        DesiredRetention: ptr(0.0),
        Timezone:         ptr(""),
    }

    err := input.Validate()
    require.ErrorIs(t, err, domain.ErrValidation)

    var valErr *domain.ValidationError
    require.ErrorAs(t, err, &valErr)
    assert.Len(t, valErr.Errors, 5, "each invalid field should produce a separate error")
}
```

**Step 2: Run tests — expect FAIL**

```bash
cd backend_v4 && go test ./internal/service/user/ -run TestUpdateSettingsInput -v -count=1
```

Expected: compilation error — `UpdateSettingsInput` has no field `DesiredRetention`.

**Step 3: Add field to struct and validation logic**

In `input.go`, change the struct (lines 36-42):

```go
type UpdateSettingsInput struct {
    NewCardsPerDay   *int
    ReviewsPerDay    *int
    MaxIntervalDays  *int
    DesiredRetention *float64
    Timezone         *string
}
```

In `Validate()`, add after the MaxIntervalDays block (after line 70):

```go
if i.DesiredRetention != nil {
    if *i.DesiredRetention < 0.70 {
        errs = append(errs, domain.FieldError{Field: "desired_retention", Message: "must be at least 0.70"})
    } else if *i.DesiredRetention > 0.99 {
        errs = append(errs, domain.FieldError{Field: "desired_retention", Message: "must be at most 0.99"})
    }
}
```

**Step 4: Run tests — expect PASS**

```bash
cd backend_v4 && go test ./internal/service/user/ -run TestUpdateSettingsInput -v -count=1
```

Expected: all PASS.

**Step 5: Commit**

```bash
cd backend_v4 && git add internal/service/user/input.go internal/service/user/input_test.go
git commit -m "fix(user): add DesiredRetention field + validation [0.70, 0.99] to UpdateSettingsInput"
```

---

### Task 2: Wire `DesiredRetention` through apply + audit + resolver

**Files:**
- Modify: `backend_v4/internal/service/user/settings.go:97-114` (applySettingsChanges)
- Modify: `backend_v4/internal/service/user/settings.go:117-146` (buildSettingsChanges)
- Modify: `backend_v4/internal/transport/graphql/resolver/user.resolvers.go:24-29` (resolver mapping)
- Test: `backend_v4/internal/service/user/service_test.go`

**Step 1: Write failing test for UpdateSettings with DesiredRetention**

Add to `service_test.go`. Find the existing `TestService_UpdateSettings_*` test pattern and add:

```go
func TestService_UpdateSettings_DesiredRetention(t *testing.T) {
    t.Parallel()

    userID := uuid.New()
    ctx := ctxutil.WithUserID(context.Background(), userID)

    current := domain.DefaultUserSettings(userID)
    newRetention := 0.85

    updated := current
    updated.DesiredRetention = newRetention

    settings := &settingsRepoMock{
        GetSettingsFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
            return &current, nil
        },
        UpdateSettingsFunc: func(ctx context.Context, uid uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error) {
            assert.Equal(t, newRetention, s.DesiredRetention)
            return &updated, nil
        },
    }

    audit := &auditRepoMock{
        CreateFunc: func(ctx context.Context, record domain.AuditRecord) (domain.AuditRecord, error) {
            // Verify desired_retention is in audit changes
            changes := record.Changes
            dr, ok := changes["desired_retention"]
            require.True(t, ok, "desired_retention should be in audit changes")
            drMap := dr.(map[string]any)
            assert.Equal(t, 0.9, drMap["old"])
            assert.Equal(t, 0.85, drMap["new"])
            return record, nil
        },
    }

    tx := &txManagerMock{
        RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
            return fn(ctx)
        },
    }

    svc := newTestService(nil, settings, audit, tx)
    result, err := svc.UpdateSettings(ctx, UpdateSettingsInput{
        DesiredRetention: &newRetention,
    })

    require.NoError(t, err)
    assert.Equal(t, newRetention, result.DesiredRetention)
}
```

**Step 2: Run test — expect FAIL**

```bash
cd backend_v4 && go test ./internal/service/user/ -run TestService_UpdateSettings_DesiredRetention -v -count=1
```

Expected: FAIL — `applySettingsChanges` doesn't apply DesiredRetention, so UpdateSettings gets called with old value 0.9.

**Step 3: Fix applySettingsChanges and buildSettingsChanges**

In `settings.go`, add to `applySettingsChanges` (after line 110, before `return result`):

```go
if input.DesiredRetention != nil {
    result.DesiredRetention = *input.DesiredRetention
}
```

In `buildSettingsChanges`, add (after the Timezone block, before `return changes`):

```go
if old.DesiredRetention != new.DesiredRetention {
    changes["desired_retention"] = map[string]any{
        "old": old.DesiredRetention,
        "new": new.DesiredRetention,
    }
}
```

**Step 4: Fix resolver mapping**

In `user.resolvers.go`, change lines 24-29 to:

```go
serviceInput := user.UpdateSettingsInput{
    NewCardsPerDay:   input.NewCardsPerDay,
    ReviewsPerDay:    input.ReviewsPerDay,
    MaxIntervalDays:  input.MaxIntervalDays,
    DesiredRetention: input.DesiredRetention,
    Timezone:         input.Timezone,
}
```

**Step 5: Run tests — expect PASS**

```bash
cd backend_v4 && go test ./internal/service/user/ -run TestService_UpdateSettings -v -count=1
```

Expected: all PASS.

**Step 6: Run full user package tests**

```bash
cd backend_v4 && go test ./internal/service/user/ -v -count=1 -race
```

Expected: all PASS.

**Step 7: Commit**

```bash
cd backend_v4 && git add internal/service/user/settings.go internal/transport/graphql/resolver/user.resolvers.go
git commit -m "fix(user): wire DesiredRetention through apply, audit, and GraphQL resolver"
```

---

### Task 3: Eliminate N+1 in StudyQueue resolver

**Files:**
- Modify: `backend_v4/internal/service/study/service.go:52-55` (add `GetByIDs` to entryRepo interface)
- Create: `backend_v4/internal/service/study/study_queue_entries.go` (new method)
- Modify: `backend_v4/internal/transport/graphql/resolver/study.resolvers.go:190-219` (use batch)
- Modify: `backend_v4/internal/transport/graphql/resolver/resolver.go:52-67` (add method to studyService interface)
- Test: `backend_v4/internal/service/study/service_test.go` (new test)

**Step 1: Write failing test for batch entry loading**

Add to `service_test.go` a test for a new `GetStudyQueueEntries` method:

```go
func TestService_GetStudyQueueEntries_BatchLoadsEntries(t *testing.T) {
    t.Parallel()

    userID := uuid.New()
    ctx := ctxutil.WithUserID(context.Background(), userID)
    now := time.Now()

    card1 := &domain.Card{ID: uuid.New(), EntryID: uuid.New(), State: domain.CardStateReview, Due: now.Add(-1 * time.Hour)}
    card2 := &domain.Card{ID: uuid.New(), EntryID: uuid.New(), State: domain.CardStateReview, Due: now.Add(-30 * time.Minute)}

    entry1 := domain.Entry{ID: card1.EntryID, UserID: userID, Headword: "hello"}
    entry2 := domain.Entry{ID: card2.EntryID, UserID: userID, Headword: "world"}

    settings := &domain.UserSettings{
        UserID:         userID,
        NewCardsPerDay: 20,
        Timezone:       "UTC",
    }

    mockSettings := &settingsRepoMock{
        GetByUserIDFunc: func(ctx context.Context, uid uuid.UUID) (*domain.UserSettings, error) {
            return settings, nil
        },
    }
    mockCards := &cardRepoMock{
        GetDueCardsFunc: func(ctx context.Context, uid uuid.UUID, now time.Time, limit int) ([]*domain.Card, error) {
            return []*domain.Card{card1, card2}, nil
        },
        GetNewCardsFunc: func(ctx context.Context, uid uuid.UUID, limit int) ([]*domain.Card, error) {
            return nil, nil
        },
    }
    mockReviews := &reviewLogRepoMock{
        CountNewTodayFunc: func(ctx context.Context, uid uuid.UUID, dayStart time.Time) (int, error) {
            return 0, nil
        },
    }

    var batchCalled bool
    mockEntries := &entryRepoMock{
        GetByIDsFunc: func(ctx context.Context, uid uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error) {
            batchCalled = true
            assert.Len(t, ids, 2)
            return []domain.Entry{entry1, entry2}, nil
        },
    }

    svc := newTestService(/* pass repos including mockEntries */)
    entries, err := svc.GetStudyQueueEntries(ctx, GetQueueInput{Limit: 50})

    require.NoError(t, err)
    assert.Len(t, entries, 2)
    assert.True(t, batchCalled, "should use batch GetByIDs, not individual GetByID")
}
```

Note: The test construction depends on how `newTestService` is set up in the study package. Adapt mock injection to match the existing pattern (check the existing tests at the top of `service_test.go` for the exact mock setup).

**Step 2: Run test — expect FAIL (method doesn't exist)**

```bash
cd backend_v4 && go test ./internal/service/study/ -run TestService_GetStudyQueueEntries -v -count=1
```

Expected: compilation error — `GetStudyQueueEntries` undefined.

**Step 3: Add `GetByIDs` to the study service's entryRepo interface**

In `service.go`, change the `entryRepo` interface (lines 52-55):

```go
type entryRepo interface {
    GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
    GetByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error)
    ExistByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error)
}
```

**Step 4: Create `study_queue_entries.go`**

```go
package study

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/heartmarshall/myenglish-backend/internal/domain"
)

// GetStudyQueueEntries returns entries for the study queue, using batch loading.
func (s *Service) GetStudyQueueEntries(ctx context.Context, input GetQueueInput) ([]*domain.Entry, error) {
    userID, ok := ctxutil.UserIDFromCtx(ctx)
    if !ok {
        return nil, domain.ErrUnauthorized
    }

    cards, err := s.GetStudyQueue(ctx, input)
    if err != nil {
        return nil, err
    }

    if len(cards) == 0 {
        return nil, nil
    }

    // Collect entry IDs preserving card order
    entryIDs := make([]uuid.UUID, len(cards))
    for i, c := range cards {
        entryIDs[i] = c.EntryID
    }

    // Single batch query instead of N individual queries
    entriesList, err := s.entries.GetByIDs(ctx, userID, entryIDs)
    if err != nil {
        return nil, fmt.Errorf("batch load entries: %w", err)
    }

    // Index by ID for O(1) lookup
    byID := make(map[uuid.UUID]*domain.Entry, len(entriesList))
    for i := range entriesList {
        byID[entriesList[i].ID] = &entriesList[i]
    }

    // Preserve card ordering
    result := make([]*domain.Entry, 0, len(cards))
    for _, c := range cards {
        if e, ok := byID[c.EntryID]; ok {
            result = append(result, e)
        }
    }

    return result, nil
}
```

**Step 5: Add method to resolver's studyService interface**

In `resolver.go`, add to the `studyService` interface (after line 66):

```go
GetStudyQueueEntries(ctx context.Context, input study.GetQueueInput) ([]*domain.Entry, error)
```

**Step 6: Update resolver to use batch method**

In `study.resolvers.go`, replace the `StudyQueue` resolver (lines 191-219):

```go
func (r *queryResolver) StudyQueue(ctx context.Context, limit *int) ([]*domain.Entry, error) {
    _, ok := ctxutil.UserIDFromCtx(ctx)
    if !ok {
        return nil, domain.ErrUnauthorized
    }

    l := 20
    if limit != nil {
        l = *limit
    }

    serviceInput := study.GetQueueInput{Limit: l}
    entries, err := r.study.GetStudyQueueEntries(ctx, serviceInput)
    if err != nil {
        return nil, err
    }

    return entries, nil
}
```

**Step 7: Update resolver mock test files**

The mock in `study_service_mock_test.go` and `study_test.go` need the new method added. Check the mock pattern used in that file and add `GetStudyQueueEntries` to match.

**Step 8: Run tests**

```bash
cd backend_v4 && go test ./internal/service/study/ -v -count=1 -race
cd backend_v4 && go test ./internal/transport/graphql/resolver/ -v -count=1 -race
```

Expected: all PASS.

**Step 9: Commit**

```bash
cd backend_v4 && git add internal/service/study/service.go internal/service/study/study_queue_entries.go \
    internal/transport/graphql/resolver/study.resolvers.go internal/transport/graphql/resolver/resolver.go
git commit -m "perf(study): eliminate N+1 in StudyQueue by batch-loading entries via GetByIDs"
```

---

### Task 4: Document `ReviewsPerDay` design decision

The code already has an intentional comment in `domain/study.go:17`:

```go
ReviewsPerDay int // Not enforced in study queue. Due cards are always shown regardless of this limit.
```

This is a deliberate Anki-style design: due cards must never be hidden (hiding them degrades long-term retention). `ReviewsPerDay` exists for UI display ("you've reached your daily goal") but does NOT gate access to reviews.

**Files:**
- Modify: `backend_v4/internal/service/study/study_queue.go:48` (expand comment)

**Step 1: Expand the comment in study_queue.go**

Change line 48 from:

```go
// Get due cards (overdue not limited by reviews_per_day)
```

to:

```go
// Due cards are always returned regardless of ReviewsPerDay setting.
// Design decision: hiding due cards degrades long-term retention (Anki behaviour).
// ReviewsPerDay is an informational goal shown in dashboard UI, not a hard limit.
```

**Step 2: Commit**

```bash
cd backend_v4 && git add internal/service/study/study_queue.go
git commit -m "docs(study): clarify ReviewsPerDay is informational goal, not hard limit"
```

---

### Task 5: Run full test suite and E2E verification

**Step 1: Run unit tests**

```bash
cd backend_v4 && make test
```

Expected: all PASS.

**Step 2: Run integration tests**

```bash
cd backend_v4 && make test-integration
```

Expected: all PASS.

**Step 3: Run E2E tests**

```bash
cd backend_v4 && make test-e2e
```

Expected: all PASS.

**Step 4: Run linter**

```bash
cd backend_v4 && make lint
```

Expected: no new warnings/errors.

**Step 5: Final commit if any lint fixes needed**

---

## Task Dependency Graph

```
Task 1 (input struct + validation)
  └──→ Task 2 (apply + audit + resolver) ──→ Task 5 (full test suite)
                                                ↑
Task 3 (N+1 fix) ──────────────────────────────┘
                                                ↑
Task 4 (docs) ─────────────────────────────────┘
```

Tasks 1→2 are sequential (2 depends on 1).
Tasks 3 and 4 are independent and can run in parallel with 1→2.
Task 5 runs after all others.
