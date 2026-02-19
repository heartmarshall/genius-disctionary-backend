# Test Audit Report: `refcatalog`

**Package**: `backend_v4/internal/service/refcatalog`
**Date**: 2026-02-19
**Auditor**: Claude (automated)

## Summary

| Metric | Before | After |
|--------|--------|-------|
| Total tests | 25 (22 top-level + 4 subtests) | 30 (25 top-level + 9 subtests) |
| Coverage | 95.6% | 100.0% |
| Fitted tests | 0 | 0 |
| Tests fixed | 0 | 0 |
| Tests added | 3 new test functions, 5 new `mapPartOfSpeech` subtests | - |
| Race conditions | None | None |

## Verdict

This is a **well-written test suite**. No fitted tests were found. All existing tests verify
behavioral contracts rather than implementation details. The mock strategy (func-field mocks)
is clean and idiomatic.

The only gaps were:
- Three uncovered error branches in `GetOrFetchEntry` (now covered)
- Limited part-of-speech mapping cases in `TestMapPartOfSpeech` (now extended)

## Coverage Before & After

### Before (95.6%)

| Function | Coverage |
|----------|----------|
| `GetOrFetchEntry` | 90.9% |
| `GetRefEntry` | 100.0% |
| `mapToRefEntry` | 100.0% |
| `mapPartOfSpeech` | 100.0% |
| `Search` | 100.0% |
| `NewService` | 100.0% |
| `clampLimit` | 100.0% |

### After (100.0%)

All functions at 100.0%.

## Per-Test Classification

### Search (4 tests)

| Test | Verdict | Rationale |
|------|---------|-----------|
| `TestService_Search_EmptyQuery` | GOOD | Verifies contract: empty query returns empty result without hitting repo |
| `TestService_Search_NormalQuery` | GOOD | Verifies passthrough to repo with correct args |
| `TestService_Search_LimitClampedToMax` | GOOD | Verifies clamping contract at upper bound (999 -> 50) |
| `TestService_Search_LimitClampedToMin` | GOOD | Verifies default-on-zero contract (0 -> 20) |

### GetOrFetchEntry (15 tests)

| Test | Verdict | Rationale |
|------|---------|-----------|
| `TestService_GetOrFetchEntry_WordInCatalog` | GOOD | Verifies early return + providers NOT called |
| `TestService_GetOrFetchEntry_FetchSuccessNoTranslations` | GOOD | Full fetch-create path, checks contract fields |
| `TestService_GetOrFetchEntry_FetchSuccessWithTranslations` | GOOD | Verifies key rule: translations only on first sense |
| `TestService_GetOrFetchEntry_WordNotFound` | GOOD | Tests `ErrWordNotFound` when dict returns nil |
| `TestService_GetOrFetchEntry_DictionaryProviderError` | GOOD | Tests error propagation with `ErrorIs` |
| `TestService_GetOrFetchEntry_TranslationProviderError` | GOOD | Tests graceful degradation (key business rule) |
| `TestService_GetOrFetchEntry_ConcurrentCreate` | GOOD | Tests race condition recovery via `ErrAlreadyExists` |
| `TestService_GetOrFetchEntry_EmptyText` | GOOD | Validates empty text -> `ValidationError` |
| `TestService_GetOrFetchEntry_TextOfSpaces` | GOOD | Whitespace normalization -> `ValidationError` |
| `TestService_GetOrFetchEntry_ProviderReturnsNoSenses` | GOOD | Edge case: dict result with no senses |
| `TestService_GetOrFetchEntry_SensesWithoutExamples` | GOOD | Edge case: sense with nil examples |
| `TestService_GetOrFetchEntry_TranslationsExistSensesEmpty` | GOOD | Edge case: translations dropped when no senses |
| `TestService_GetOrFetchEntry_RepoErrorOnLookup` | **NEW** | Covers unexpected repo error on initial lookup |
| `TestService_GetOrFetchEntry_CreateTransactionError` | **NEW** | Covers non-conflict transaction failure |
| `TestService_GetOrFetchEntry_ConflictThenFetchFails` | **NEW** | Covers conflict + re-fetch failure |

### GetRefEntry (2 tests)

| Test | Verdict | Rationale |
|------|---------|-----------|
| `TestService_GetRefEntry_Found` | GOOD | Happy path passthrough |
| `TestService_GetRefEntry_NotFound` | GOOD | Error passthrough with `ErrorIs` |

### mapToRefEntry (6 tests)

| Test | Verdict | Rationale |
|------|---------|-----------|
| `TestMapToRefEntry_FullResult` | GOOD | Comprehensive: all fields, parent refs, positions, source slugs |
| `TestMapToRefEntry_WithoutTranslations` | GOOD | Nil translations -> empty slice |
| `TestMapToRefEntry_WithTranslations` | GOOD | Translations only attached to first sense |
| `TestMapToRefEntry_WithoutPronunciations` | GOOD | No pronunciations -> empty slice |
| `TestMapToRefEntry_MultipleSensesPositions` | GOOD | Sequential position assignment verified |
| `TestMapToRefEntry_UUIDUniqueness` | GOOD | All generated UUIDs are unique across the tree |

### mapPartOfSpeech (1 test, 9 subtests)

| Test | Verdict | Rationale |
|------|---------|-----------|
| `TestMapPartOfSpeech` | IMPROVED | Was IMPROVABLE: only tested noun, verb, unknown, nil. Now also tests adjective, interjection, already-uppercase, mixed-case, and empty string |

## Changes Made

### New tests added to `service_test.go`

1. **`TestService_GetOrFetchEntry_RepoErrorOnLookup`** - When `GetFullTreeByText` returns
   an unexpected error (not `ErrNotFound`), the service should propagate it. This covered
   the branch at `get_or_fetch.go:27-29`.

2. **`TestService_GetOrFetchEntry_CreateTransactionError`** - When `CreateWithTree` fails
   with a non-conflict error (e.g., "disk full"), the service should wrap and return it.
   This covered the branch at `get_or_fetch.go:74`.

3. **`TestService_GetOrFetchEntry_ConflictThenFetchFails`** - When a concurrent insert
   causes `ErrAlreadyExists` but the subsequent re-fetch also fails, the service should
   propagate the re-fetch error. This covered the branch at `get_or_fetch.go:69-71`.

### Extended `TestMapPartOfSpeech` subtests

Added 5 new cases to the table-driven test:
- `adjective lowercase` - tests another valid POS value
- `interjection lowercase` - tests another valid POS value
- `already uppercase NOUN` - verifies idempotent uppercase conversion
- `mixed case Verb` - verifies case-insensitive matching
- `empty string maps to OTHER` - verifies empty string is treated as unknown

## Potential Code Issues Found

None. The error handling paths are all correctly implemented. The concurrent-create
recovery logic is sound. The graceful degradation for translation provider errors works
as documented.
