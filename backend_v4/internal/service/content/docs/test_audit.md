# Test Audit Report — `internal/service/content`

**Date:** 2026-02-18
**Package:** `github.com/heartmarshall/myenglish-backend/internal/service/content`

## Summary

| Metric | Before | After |
|--------|--------|-------|
| Total tests | 68 | 126 |
| Coverage | 84.9% | 94.3% |
| Fitted tests | 0 | 0 |
| All tests pass | Yes | Yes |
| Race conditions | None | None |

## Test Quality Assessment

The existing test suite is **well-written**. No fitted tests were detected. Tests consistently verify behavioral contracts (auth, validation, ownership, limits, audit logging) rather than mirroring implementation details.

### Strengths

- Every service method has auth (NoUserID), not-found, and happy-path tests
- Audit log assertions verify entity type, action, and change structure — testing the *contract* of what gets audited
- Edge cases like "partial reorder", "all-nil update", "no-op skips audit" are covered
- Mocks use function fields allowing per-test behavior — clean and readable
- Table-driven format used where appropriate (AddUserImage_InvalidURL)

### What Was Missing (Now Fixed)

| Gap | Impact | Resolution |
|-----|--------|------------|
| `buildSenseChanges` at 47.8% | PartOfSpeech and CEFRLevel change tracking untested | Added 6 tests covering all branches, nil→set, no-change |
| Input `Validate()` methods at 63-78% | Boundary cases (text too long, invalid CEFR/POS) untested | Added table-driven tests for all 8 input types + `isValidHTTPURL` + `validateReorderItems` |
| `UpdateExample` no-change-skips-audit | Behavior tested for translations and images but not examples | Added `TestService_UpdateExample_NoChangeSkipsAudit` |

## Per-Test Verdicts

All existing tests: **GOOD**. No tests were rewritten or removed.

Notable observations:
- `TestService_AddSense_EntryDeleted` is mechanically identical to `TestService_AddSense_EntryNotFound` (same mock behavior), but serves as documentation that soft-deleted entries are treated as not found. Not flagged as fitted — it tests a conceptual scenario.
- `TestService_UpdateTranslation_AuditContainsOldAndNew` overlaps significantly with `TestService_UpdateTranslation_HappyPath`. Both verify audit changes on translation update. Could be merged, but having a focused audit-structure test is reasonable.

## Coverage Details (After)

### 100% Coverage

| Function | File |
|----------|------|
| `NewService` | service.go |
| All 8 `Validate()` methods except 2 reorder inputs | input.go |
| `isValidHTTPURL` | input.go |
| `fieldIndex` | input.go |
| `validateReorderItems` | input.go |
| `buildSenseChanges` | sense.go |
| `buildExampleChanges` | example.go |
| `buildImageCaptionChanges` | userimage.go |

### Remaining <100% (87-96%)

These are service methods where the uncovered lines are `fmt.Errorf("...: %w", err)` error-wrapping paths — when a repo call fails mid-transaction (e.g., `CountByEntry` returns an unexpected error). Testing these requires mocking repos to fail at specific points. This is valid but has diminishing returns since the wrapping is purely mechanical.

| Function | Coverage | Uncovered |
|----------|----------|-----------|
| `AddSense` | 91.7% | CountByEntry error, CreateCustom error, translation CreateCustom error |
| `UpdateSense` | 95.8% | Update error |
| `DeleteSense` | 92.9% | Delete error |
| `ReorderSenses` | 95.7% | GetByEntryID error |
| `AddTranslation` | 87.5% | CountBySense error, CreateCustom error |
| `UpdateTranslation` | 91.7% | Update error |
| `DeleteTranslation` | 92.9% | Delete error |
| `ReorderTranslations` | 87.0% | GetBySenseID error |
| `AddExample` | 89.3% | CountBySense error, CreateCustom error |
| `UpdateExample` | 92.3% | Update error |
| `DeleteExample` | 92.9% | Delete error |
| `ReorderExamples` | 87.0% | GetBySenseID error |
| `AddUserImage` | 92.9% | CountUserByEntry error, CreateUser error |
| `UpdateUserImage` | 92.0% | UpdateUser error |
| `DeleteUserImage` | 90.9% | DeleteUser error |

## Files Modified

- **`input_test.go`** (NEW) — 58 table-driven validation tests for all input types, `isValidHTTPURL`, and `validateReorderItems`
- **`sense_test.go`** — Added 6 `buildSenseChanges` unit tests
- **`example_test.go`** — Added `TestService_UpdateExample_NoChangeSkipsAudit`
