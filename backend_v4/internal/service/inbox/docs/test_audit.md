# Test Audit: `inbox` package

**Date:** 2026-02-19
**Package:** `internal/service/inbox`
**Test framework:** stdlib `testing` + `moq` generated mocks

## Summary

| Metric | Before | After |
|---|---|---|
| Total tests | 30 | 40 |
| Passing | 30 | 40 |
| Coverage | 92.1% | 98.9% |
| Fitted tests | 0 | 0 |
| Race conditions | 0 | 0 |

## Existing Test Verdicts

All 30 original tests are well-written contract tests. None are fitted.

### CreateItem (11 tests)

| Test | Verdict | Notes |
|---|---|---|
| `TestCreateItem_Success` | GOOD | Happy path, verifies returned fields + repo call counts |
| `TestCreateItem_WithoutContext` | GOOD | nil context stays nil through to repo |
| `TestCreateItem_EmptyText` | GOOD | Validation contract for empty text |
| `TestCreateItem_WhitespaceOnlyText` | GOOD | Whitespace-only treated as empty |
| `TestCreateItem_TextTooLong` | GOOD | Boundary at 501 chars |
| `TestCreateItem_ContextTooLong` | GOOD | Boundary at 2001 chars |
| `TestCreateItem_EmptyContextToNil` | GOOD | trimOrNil behavior — whitespace-only becomes nil |
| `TestCreateItem_TextTrimmed` | GOOD | Whitespace stripped before storage |
| `TestCreateItem_InboxFull` | GOOD | Capacity limit + verifies Create NOT called |
| `TestCreateItem_DuplicateAllowed` | GOOD | Documents that duplicate text is allowed |
| `TestCreateItem_Unauthorized` | GOOD | No user in context |

### ListItems (7 tests)

| Test | Verdict | Notes |
|---|---|---|
| `TestListItems_Success` | GOOD | Happy path with parameter passthrough verification |
| `TestListItems_Empty` | GOOD | Empty result set |
| `TestListItems_Pagination` | GOOD | Offset + limit forwarded correctly |
| `TestListItems_DefaultLimit` | GOOD | limit=0 becomes DefaultLimit |
| `TestListItems_InvalidLimit` | GOOD | Negative limit rejected |
| `TestListItems_LimitTooLarge` | GOOD | 201 rejected |
| `TestListItems_Unauthorized` | GOOD | No user in context |

### GetItem (4 tests)

| Test | Verdict | Notes |
|---|---|---|
| `TestGetItem_Success` | GOOD | Happy path |
| `TestGetItem_NotFound` | GOOD | Repo returns ErrNotFound, properly wrapped |
| `TestGetItem_WrongUser` | GOOD | Cross-user isolation (scoped via repo) |
| `TestGetItem_Unauthorized` | GOOD | No user in context |

### DeleteItem (5 tests)

| Test | Verdict | Notes |
|---|---|---|
| `TestDeleteItem_Success` | GOOD | Happy path + repo call verification |
| `TestDeleteItem_NotFound` | GOOD | Repo returns ErrNotFound |
| `TestDeleteItem_WrongUser` | GOOD | Cross-user isolation |
| `TestDeleteItem_NilID` | GOOD | Nil UUID validation |
| `TestDeleteItem_Unauthorized` | GOOD | No user in context |

### DeleteAll (3 tests)

| Test | Verdict | Notes |
|---|---|---|
| `TestDeleteAll_Success` | GOOD | Returns deleted count |
| `TestDeleteAll_EmptyInbox` | GOOD | Zero is valid |
| `TestDeleteAll_Unauthorized` | GOOD | No user in context |

## Coverage Gaps Found & Fixed

### Gap 1: Repo error propagation in `CreateItem` (85% -> 100%)

**Missing paths:** `Count` returning error, `Create` returning error.

**Added:**
- `TestCreateItem_CountRepoError` — verifies error wrapping with `"count inbox items"` context, and that `Create` is not called.
- `TestCreateItem_CreateRepoError` — verifies error wrapping with `"create inbox item"` context.

### Gap 2: Repo error propagation in `ListItems` (91.7% -> 100%)

**Missing path:** `List` returning error.

**Added:**
- `TestListItems_RepoError` — verifies error wrapping with `"list inbox items"` context.

### Gap 3: Negative offset validation in `ListItemsInput.Validate` (90% -> 100%)

**Missing path:** `offset < 0` branch.

**Added:**
- `TestListItems_NegativeOffset` — verifies ValidationError with field `"offset"`.

### Gap 4: Repo error propagation in `DeleteAll` (87.5% -> 100%)

**Missing path:** `DeleteAll` returning error.

**Added:**
- `TestDeleteAll_RepoError` — verifies error wrapping with `"delete all inbox items"` context.

### Gap 5: Boundary values

**Added:**
- `TestCreateItem_TextExactlyAtLimit` — 500 chars accepted (off-by-one check).
- `TestCreateItem_ContextExactlyAtLimit` — 2000 chars accepted.
- `TestCreateItem_InboxAtCapacityMinusOne` — count=499, create succeeds (the 500th item).
- `TestListItems_LimitExactlyAtMax` — limit=200 accepted.

## Remaining Uncovered

| Function | Coverage | Reason |
|---|---|---|
| `NewService` | 0% | Trivial constructor (assigns fields). Tests construct `Service` directly. Not worth testing. |

## Final Per-Function Coverage

| Function | Coverage |
|---|---|
| `CreateItem` | 100% |
| `DeleteItem` | 100% |
| `DeleteAll` | 100% |
| `CreateItemInput.Validate` | 100% |
| `ListItemsInput.Validate` | 100% |
| `DeleteItemInput.Validate` | 100% |
| `ListItems` | 100% |
| `GetItem` | 100% |
| `trimOrNil` | 100% |
| `NewService` | 0% (trivial) |
| **Total** | **98.9%** |

## Potential Issues Noticed

None. The package is simple, clean, and well-tested. All error paths are now covered and the error wrapping strategy (`fmt.Errorf("{operation}: %w", err)`) is consistently verified.
