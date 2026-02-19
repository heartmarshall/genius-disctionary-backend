# Test Audit: `dictionary` service

**Date:** 2026-02-19
**Package:** `backend_v4/internal/service/dictionary`
**Coverage:** 84.4% → **90.8%** (after fixes)
**Tests:** 75 → **97** (all passing, race-clean)

## Summary

The original test suite was **well-written** overall — no truly fitted tests were found.
Two tests were redundant/improvable and were replaced. The main gap was **missing coverage**:
8 operations lacked NoAuth tests, `CreateEntryCustom` was missing 4 edge cases, and
`CreateCustomInput.Validate` nested field paths were entirely untested.

## Fitted Test Detection

| # | Test | Verdict | Notes |
|---|------|---------|-------|
| 1 | SearchCatalog_EmptyQuery | ✅ GOOD | Tests behavioral contract: empty query → empty result |
| 2 | SearchCatalog_NormalQuery | ✅ GOOD | Verifies query+limit forwarding |
| 3 | SearchCatalog_LimitClamp | ✅ GOOD | Tests limit clamped to 50 |
| 4 | SearchCatalog_NoAuth | ✅ GOOD | Auth guard |
| 5 | PreviewRefEntry_Success | ✅ GOOD | Delegation test |
| 6 | ~~PreviewRefEntry_FetchOK~~ | ⚠️ REDUNDANT → **Replaced** with PreviewRefEntry_NoAuth | Identical path to _Success. PreviewRefEntry doesn't normalize text, so "World" vs "hello" tested nothing new. |
| 7 | PreviewRefEntry_APIError | ✅ GOOD | Error propagation |
| 8 | CreateFromCatalog_AllSenses | ✅ GOOD | Comprehensive: senses, pronunciations, images, audit |
| 9 | CreateFromCatalog_SelectedSenses | ✅ GOOD | Sense cherry-picking |
| 10 | CreateFromCatalog_WithCard | ✅ GOOD | Card creation with correct params |
| 11 | CreateFromCatalog_NoCard | ✅ GOOD | Card NOT created when not requested |
| 12 | CreateFromCatalog_WithNotes | ✅ GOOD | Notes passthrough |
| 13 | CreateFromCatalog_Duplicate | ✅ GOOD | Pre-check duplicate detection |
| 14 | CreateFromCatalog_DuplicateUniqueConstraint | ✅ GOOD | Concurrent duplicate (tx path) |
| 15 | CreateFromCatalog_LimitReached | ✅ GOOD | Entry limit enforcement |
| 16 | CreateFromCatalog_RefNotFound | ✅ GOOD | Ref → validation error conversion |
| 17 | CreateFromCatalog_InvalidSenseID | ✅ GOOD | Sense ID not in ref entry |
| 18 | CreateFromCatalog_InvalidInput | ✅ GOOD | Nil UUID validation |
| 19 | CreateFromCatalog_NoAuth | ✅ GOOD | Auth guard |
| 20 | CreateFromCatalog_RefWithoutSenses | ✅ GOOD | Edge case: ref with no senses |
| 21 | CreateCustom_HappyPath | ✅ GOOD | Full happy path with source slug |
| 22 | CreateCustom_EmptySenses | ✅ GOOD | Entry without senses |
| 23 | CreateCustom_NoDefinition | ✅ GOOD | Nil definition passthrough |
| 24 | CreateCustom_WithCard | ✅ GOOD | Card creation |
| 25 | CreateCustom_NormalizesText | ✅ GOOD | Text normalization contract |
| 26 | CreateCustom_Duplicate | ✅ GOOD | Duplicate detection |
| 27 | CreateCustom_InvalidText | ✅ GOOD | Empty text validation |
| 28 | CreateCustom_TooLongText | ✅ GOOD | Max length validation |
| 29 | CreateCustom_SourceSlugUser | ✅ GOOD | Source slug = "user" for all children |
| 30 | FindEntries_NoFiltersOffset | ✅ GOOD | Default pagination + sort |
| 31 | FindEntries_WithSearch | ✅ GOOD | Search normalization |
| 32 | FindEntries_SearchSpaces | ✅ GOOD | Spaces-only → nil search |
| 33 | FindEntries_HasCard | ✅ GOOD | Filter passthrough |
| 34 | FindEntries_TopicID | ✅ GOOD | Filter passthrough |
| 35 | FindEntries_ComboFilters | ✅ GOOD | Multiple filters |
| 36 | FindEntries_Cursor | ✅ GOOD | Cursor-based pagination |
| 37 | FindEntries_CursorWithOffset | ✅ GOOD | Cursor takes priority |
| 38 | FindEntries_LimitClamp | ✅ GOOD | Limit clamped to 200 |
| 39 | ~~FindEntries_DefaultSort~~ | 🔧 REDUNDANT → **Replaced** with FindEntries_CustomSort | Duplicated the assertion already in NoFiltersOffset. Now tests custom sort passthrough. |
| 40 | FindEntries_InvalidSortBy | ✅ GOOD | Validation |
| 41 | GetEntry_Found | ✅ GOOD | User-scoped delegation |
| 42 | GetEntry_NotFound | ✅ GOOD | Error propagation |
| 43 | GetEntry_NoAuth | ✅ GOOD | Auth guard |
| 44 | UpdateNotes_Set | ✅ GOOD | Notes update + audit diff |
| 45 | UpdateNotes_Clear | ✅ GOOD | Clearing notes |
| 46 | UpdateNotes_NotFound | ✅ GOOD | Error propagation |
| 47 | UpdateNotes_Validation | ✅ GOOD | Nil UUID validation |
| 48 | DeleteEntry_Happy | ✅ GOOD | Soft delete + audit |
| 49 | DeleteEntry_NotFound | ✅ GOOD | Error propagation |
| 50 | DeleteEntry_NoAuth | ✅ GOOD | Auth guard |
| 51 | FindDeletedEntries_WithEntries | ✅ GOOD | Happy path |
| 52 | FindDeletedEntries_Empty | ✅ GOOD | Empty result |
| 53 | FindDeletedEntries_LimitClamp | ✅ GOOD | Limit clamped to 200 |
| 54 | RestoreEntry_Happy | ✅ GOOD | Delegation |
| 55 | RestoreEntry_NotFound | ✅ GOOD | Error propagation |
| 56 | RestoreEntry_TextConflict | ✅ GOOD | Already exists error |
| 57 | BatchDelete_AllOK | ✅ GOOD | Happy path + audit |
| 58 | BatchDelete_Partial | ✅ GOOD | Partial failure |
| 59 | BatchDelete_Empty | ✅ GOOD | Validation |
| 60 | BatchDelete_TooMany | ✅ GOOD | Max 200 validation |
| 61 | BatchDelete_AuditOnlyOnSuccess | ✅ GOOD | Conditional audit |
| 62 | ImportEntries_Happy | ✅ GOOD | Import with translations |
| 63 | ImportEntries_DuplicateInFile | ✅ GOOD | Within-file dedup |
| 64 | ImportEntries_ExistingEntry | ✅ GOOD | DB-level dedup |
| 65 | ImportEntries_EmptyText | ✅ GOOD | Normalization skip |
| 66 | ImportEntries_ChunkFail | ✅ GOOD | Chunk rollback |
| 67 | ImportEntries_LimitExceeded | ✅ GOOD | Upfront limit check |
| 68 | ImportEntries_EmptyItems | ✅ GOOD | Validation |
| 69 | ExportEntries_Happy | ✅ GOOD | Full denormalized export |
| 70 | ExportEntries_Empty | ✅ GOOD | Empty dictionary |
| 71 | ExportEntries_WithData | ✅ GOOD | Multi-entry with notes |
| 72–75 | *Input.Validate_CollectsAllErrors (4 tests) | ✅ GOOD | Multi-error collection |

**Verdicts: 73 GOOD, 2 REDUNDANT (replaced)**

## Tests Added (22 new)

### Missing auth guards (8 tests)
Every public method must reject unauthenticated context. These were missing:

| Test | Operation |
|------|-----------|
| `CreateCustom_NoAuth` | `CreateEntryCustom` |
| `UpdateNotes_NoAuth` | `UpdateNotes` |
| `FindDeletedEntries_NoAuth` | `FindDeletedEntries` |
| `RestoreEntry_NoAuth` | `RestoreEntry` |
| `BatchDelete_NoAuth` | `BatchDeleteEntries` |
| `ImportEntries_NoAuth` | `ImportEntries` |
| `ExportEntries_NoAuth` | `ExportEntries` |
| `FindEntries_NoAuth` | `FindEntries` |

### Missing `CreateEntryCustom` edge cases (4 tests)
| Test | What it covers |
|------|----------------|
| `CreateCustom_WhitespaceOnlyText` | Text passes empty check but normalizes to "" |
| `CreateCustom_LimitReached` | Entry limit enforcement (was only tested for catalog) |
| `CreateCustom_DuplicateUniqueConstraint` | Concurrent duplicate via tx (was only tested for catalog) |
| `CreateCustom_AuditRecord` | Audit record has correct action, entity type, and source |

### Missing input validation paths (5 tests)
| Test | What it covers |
|------|----------------|
| `CreateCustomInput_Validate_SenseFields` | Definition too long, invalid POS, too many translations, too many examples |
| `CreateCustomInput_Validate_TranslationFields` | Empty translation, too-long translation |
| `CreateCustomInput_Validate_ExampleFields` | Empty sentence, too-long sentence, too-long translation |
| `FindInput_Validate_InvalidPartOfSpeech` | Invalid POS enum value |
| `FindInput_Validate_InvalidStatus` | Invalid learning status enum value |

### Missing edge cases (5 tests)
| Test | What it covers |
|------|----------------|
| `UpdateNotesInput_Validate_NotesTooLong` | Notes max 5000 validation |
| `FindEntries_OffsetPagination_HasNextPage` | HasNextPage=true when more results exist |
| `FindEntries_InvalidSortOrder` | Invalid sort order validation |
| `SearchCatalog_LimitDefaultsWhenZero` | Limit 0 → default 20 |
| `FindEntries_LimitDefaultsWhenZero` | Limit 0 → default 20 |
| `ExportEntries_EntryWithoutCard` | CardStatus is nil when no card exists |
| `ImportEntries_ChunkRollbackResetsSeenMap` | Seen map is cleaned up on chunk rollback |

## Coverage Improvements

| Function | Before | After | Delta |
|----------|--------|-------|-------|
| `PreviewRefEntry` | 66.7% | **100.0%** | +33.3% |
| `RestoreEntry` | 85.7% | **100.0%** | +14.3% |
| `FindDeletedEntries` | 80.0% | **100.0%** | +20.0% |
| `FindInput.Validate` | 85.7% | **100.0%** | +14.3% |
| `UpdateNotesInput.Validate` | 87.5% | **100.0%** | +12.5% |
| `fieldIndex` | 0.0% | **100.0%** | +100% |
| `fieldIndex2` | 0.0% | **100.0%** | +100% |
| `CreateEntryCustom` | 71.4% | **83.7%** | +12.3% |
| `CreateCustomInput.Validate` | 69.7% | **97.0%** | +27.3% |
| `FindEntries` | 90.7% | **95.3%** | +4.6% |
| `BatchDeleteEntries` | 92.3% | **96.2%** | +3.9% |
| `clampLimit` | 71.4% | **85.7%** | +14.3% |
| **Total** | **84.4%** | **90.8%** | **+6.4%** |

## Remaining Uncovered Paths

These paths are at ≤97% but are not worth testing with unit tests — they represent
internal error propagation that would require complex mock orchestration for low value:

| Function | Uncovered Path | Why |
|----------|---------------|-----|
| `CreateEntryCustom` (83.7%) | `CountByUser` repo error, audit repo error within tx | Internal error wrapping — would need mock to fail *after* entry creation succeeds |
| `CreateEntryFromCatalog` (83.6%) | `CountByUser` repo error, `GetByText` repo error (non-ErrNotFound) | Same — internal error paths |
| `DeleteEntry` (85.7%) | `SoftDelete` repo error, audit repo error within tx | Internal error wrapping |
| `UpdateNotes` (86.4%) | `UpdateNotes` repo error, audit repo error within tx | Internal error wrapping |
| `ImportInput.Validate` (86.7%) | Item text too long (>500 chars) | Already tested at the service level via CreateCustom_TooLongText |
| `clampLimit` (85.7%) | `limit < min` branch | Never reachable from current callers (limit is always 0/negative → default, or ≥1 → valid) |
| `ExportEntries` (91.5%) | `senses.GetByEntryIDs` / `cards.GetByEntryIDs` errors | Internal error wrapping |
| `ImportEntries` (90.4%) | `entries.Create` error within tx | Internal error wrapping |

## Observations

1. **Test infrastructure is solid.** The moq-style mocks with func fields, the `testDeps` helper,
   and `authCtx()` are well-designed and consistent across all tests.
2. **No fitted tests.** Every test verifies a behavioral contract (auth, validation, side effects,
   error codes) rather than duplicating implementation logic.
3. **Good use of capture variables** (`cardCreated`, `auditCreated`, `capturedSlug`) to verify
   side effects without over-coupling to implementation order.
4. **All tests are parallel.** Good practice.
5. **testify is used consistently** — `require` for fatal checks, `assert` for non-fatal. No mixing.
