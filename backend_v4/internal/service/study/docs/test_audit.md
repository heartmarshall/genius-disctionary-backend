# Test Audit Report: `study` package

**Date**: 2026-02-19
**Coverage before**: 83.4% (58 tests)
**Coverage after**: 88.7% (87 tests)
**Fitted tests found**: 0
**Tests fixed**: 1
**Tests added**: 29

## Coverage Summary

| Function | Before | After | Notes |
|---|---|---|---|
| `GetActiveSession` | 0.0% | **100.0%** | Was completely untested |
| `StartSession` | 57.1% | **85.7%** | Added race condition (ErrAlreadyExists) test |
| `GetStudyQueue` | 93.3% | **96.7%** | Added default limit test |
| All `Validate()` methods | 60-100% | **100.0%** | All 8 validators fully covered |
| `NewService` | 0.0% | 0.0% | Trivial constructor, intentionally skipped |
| All others | — | — | Unchanged, already well-tested |

## Audit Verdicts

### srs_test.go — `TestCalculateSRS` (37 cases) + `TestApplyFuzz` (4 cases)

**Verdict: ✅ GOOD**

Table-driven tests for a pure function (no DB, no context). Expected values represent the Anki SM-2 algorithm contract, not reverse-engineered from code. The test accounts for fuzz by using ±5% tolerance on intervals >= 3 days. Covers: all 4 statuses × 4 grades, boundary conditions (min ease, max interval, out-of-bounds step), relearning vs initial learning, lapse variations, empty config fallbacks.

### timezone_test.go — `TestDayStart`, `TestNextDayStart`, `TestParseTimezone`

**Verdict: ✅ GOOD**

`TestDayStart` properly tests UTC, EST, and JST with correct expected values. `TestParseTimezone` covers valid and invalid cases. `TestNextDayStart` only tests UTC (DST transition test would be nice but not critical since implementation uses `AddDate` which handles DST correctly).

### service_test.go — Service-level tests

#### GetStudyQueue (8 tests)

| Test | Verdict | Notes |
|---|---|---|
| `Success_MixOfDueAndNew` | ✅ GOOD | Tests core business rule: due first, then new up to daily limit |
| `NoUserID` | ✅ GOOD | |
| `InvalidInput` | ✅ GOOD | |
| `SettingsLoadError` | 🔧 **FIXED** | Had buggy `errors.Is(err, errors.New("db error"))` that never matches — fixed to check non-empty error |
| `CountNewTodayError` | ✅ GOOD | |
| `DueCardsError` | ✅ GOOD | |
| `DailyLimitReached` | ✅ GOOD | Verifies GetNewCards NOT called when limit exhausted |
| `OnlyDueCards` | ✅ GOOD | Verifies queue full = no new card fetch |

#### ReviewCard (10 tests)

| Test | Verdict | Notes |
|---|---|---|
| `Success_NewToLearning` | ✅ GOOD | Full flow: snapshot, SRS update, log, audit |
| `Success_LearningToReview` | ✅ GOOD | Tests graduation |
| `Success_ReviewIntervalIncrease` | ✅ GOOD | Asserts interval grows, not exact value |
| `NoUserID` | ✅ GOOD | |
| `InvalidInput` | ✅ GOOD | |
| `CardNotFound` | ✅ GOOD | |
| `SettingsLoadError` | ✅ GOOD | |
| `UpdateSRSError_TxRollback` | ✅ GOOD | Verifies Create NOT called after error |
| `CreateReviewLogError_TxRollback` | ✅ GOOD | Verifies Audit NOT called after error |
| `AuditError_TxRollback` | ✅ GOOD | |

#### UndoReview (10 tests)

| Test | Verdict | Notes |
|---|---|---|
| `Success` | ✅ GOOD | Tests full restore: prev state, log deletion, audit |
| `NoUserID` | ✅ GOOD | |
| `InvalidInput` | ✅ GOOD | |
| `CardNotFound` | ✅ GOOD | |
| `NoReviewLog` | ✅ GOOD | Business rule: no reviews to undo |
| `PrevStateNil` | ✅ GOOD | Business rule: review cannot be undone |
| `UndoWindowExpired` | ✅ GOOD | Business rule: 15-min window. Checks specific validation error message |
| `RestoreError_TxRollback` | ✅ GOOD | |
| `DeleteLogError_TxRollback` | ✅ GOOD | |
| `AuditError_TxRollback` | ✅ GOOD | |

#### Sessions (8 tests)

| Test | Verdict | Notes |
|---|---|---|
| `StartSession_Success_CreatesNew` | ✅ GOOD | |
| `StartSession_ReturnsExisting_Idempotent` | ✅ GOOD | Business rule: idempotent |
| `FinishSession_Success` | ✅ GOOD | Tests accuracy = (GOOD+EASY)/total |
| `FinishSession_AlreadyFinished` | ✅ GOOD | Business rule: can't finish twice |
| `FinishSession_EmptySession_NoReviews` | ✅ GOOD | Edge case: 0 reviews → accuracy 0 |
| `FinishSession_NotFound` | ✅ GOOD | |
| `AbandonSession_Success` | ✅ GOOD | |
| `AbandonSession_NoActive_Noop` | ✅ GOOD | Business rule: idempotent |

#### Card CRUD (6 + 6 tests)

All ✅ GOOD. CreateCard tests verify sense requirement, BatchCreateCards tests cover all 4 partial-success scenarios (not exist, already has card, no senses, success).

#### Dashboard (7 tests)

All ✅ GOOD. Tests streak calculation with gaps, starting from yesterday, 5-day streak, overdue placeholder, active session presence.

#### GetCardHistory + GetCardStats (5 tests)

All ✅ GOOD. `GetCardStats` correctly tests accuracy = 75% with mixed grades and average time calculation with nil handling.

### input_test.go — Validation tests (NEW, 21 cases)

**Verdict: ✅ GOOD** — Table-driven tests for all 8 input validators, covering valid and invalid boundary values.

### service_test.go — New service tests (8 tests)

**Verdict: ✅ GOOD** — `GetActiveSession` (4 tests: success, no session returns nil, no user ID, repo error), `StartSession` race condition, `StartSession` no user ID, `GetStudyQueue` default limit.

## Remaining Uncovered Code

| Function | Coverage | Reason |
|---|---|---|
| `NewService` | 0% | Trivial constructor — no logic, not worth testing |
| `ReviewCard` line 123 | ~3% gap | `updatedCard == nil` safety check — defensive code against nil from mock; hard to trigger without breaking mock contract |
| `UndoReview` line 107 | ~10% gap | `restoredCard == nil` safety check — same as above |
| `GetDashboard` | 77.5% | Some error branches (CountDue/CountNew/CountByStatus failing individually). Low-value to add tests for each. |
| `AbandonSession` | 75% | Missing: repo error on GetActive (non-ErrNotFound), repo error on Abandon |

## Observations

1. **No fitted tests found.** The existing suite tests business contracts, not implementation details. Mock assertions verify correct arguments are passed, not internal state.
2. **Good error path coverage.** Transaction rollback tests verify that downstream operations are NOT called after upstream errors.
3. **The `errors.Is(err, errors.New("db error"))` pattern at `service_test.go:185` was a real bug** — `errors.New` creates a new instance that never matches via `errors.Is`. Fixed to simply verify non-nil non-empty error.
4. **Test style is consistent**: stdlib testing, moq-generated mocks, `t.Parallel()` everywhere, `t.Helper()` not used but `ptr` generic helper keeps things clean.
