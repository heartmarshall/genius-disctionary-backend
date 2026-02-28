# Phase 3: Dictionary — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Full-featured personal dictionary — browse, search, filter, add words from catalog or manually, edit entries (senses, translations, examples), delete/restore, bulk import, card management. This is the largest phase by scope.

**Architecture:** All dictionary operations use GraphQL (`POST /query`). The word list uses cursor-based pagination with Apollo cache. Entry detail page loads a single entry with all nested data. Catalog search uses autocomplete with debounce. Content editing (senses, translations, examples) is inline on the entry detail page.

**Key references:**
- API: `backend_v4/docs/API.md` — Dictionary section (lines 56–110), Content Editing (lines 112–129), Card management (lines 157–159)
- Domain model: `docs/business/DOMAIN_MODEL.md` — Entry, Sense, Translation, Example, Card relationships
- Business rules: `docs/business/BUSINESS_RULES.md` — entry limit (10,000), text lengths, max senses (20), max translations (20), max examples (20), duplicate detection, import limits (5,000)
- Workflows: `docs/business/WORKFLOWS.md` — "Building a Personal Dictionary" (add from catalog, custom, bulk import)
- Design: `frontend-real/design-docs/palette-v3.html` — Vocab Card, Input Fields, Buttons, Status Pills, Source Tags, Modals

---

## Task 1: Dictionary GraphQL Layer

**Goal:** All GraphQL queries and mutations for the dictionary domain, typed and ready.

**What to do:**
- Create `src/graphql/queries/dictionary.ts` — queries:
  - `DICTIONARY_QUERY` — word list with cursor pagination, filters, sorting
  - `DICTIONARY_ENTRY_QUERY` — single entry with all nested data (senses, translations, examples, topics, card)
  - `SEARCH_CATALOG_QUERY` — autocomplete search in reference catalog
  - `PREVIEW_REF_ENTRY_QUERY` — full preview of catalog entry before adding
  - `DELETED_ENTRIES_QUERY` — trash list (offset pagination)
- Create `src/graphql/mutations/dictionary.ts` — mutations:
  - `CREATE_ENTRY_FROM_CATALOG` — add from catalog with selected senses
  - `CREATE_ENTRY_CUSTOM` — add custom word
  - `DELETE_ENTRY` — soft delete
  - `RESTORE_ENTRY` — restore from trash
  - `IMPORT_ENTRIES` — bulk import
- Create `src/graphql/mutations/content.ts` — content editing mutations:
  - Senses: `ADD_SENSE`, `UPDATE_SENSE`, `DELETE_SENSE`, `REORDER_SENSES`
  - Translations: `ADD_TRANSLATION`, `UPDATE_TRANSLATION`, `DELETE_TRANSLATION`
  - Examples: `ADD_EXAMPLE`, `UPDATE_EXAMPLE`, `DELETE_EXAMPLE`
  - Images: `ADD_USER_IMAGE`, `DELETE_USER_IMAGE`
- Create `src/graphql/mutations/cards.ts`:
  - `CREATE_CARD`, `BATCH_CREATE_CARDS`, `DELETE_CARD`
- Define TypeScript interfaces in `src/types/dictionary.ts`:
  - `Entry`, `Sense`, `Translation`, `Example`, `UserImage`, `Card`, `RefEntry`, `RefSense`
  - `DictionaryConnection` (edges/pageInfo/totalCount), `DictionaryFilters`
  - `ImportResult` (created, skipped, errors)
- Configure Apollo cache type policy for `dictionary` query cursor pagination (merge function for edges)

**Commit:** `feat(dictionary): add GraphQL queries, mutations, and types`

---

## Task 2: Dictionary List Page

**Goal:** Browsable word list with search, filters, sort, and infinite scroll or pagination.

**Context:** The `dictionary` query uses cursor-based pagination (`first`, `after`). Returns `edges[].node` (Entry with basic card info), `pageInfo` (hasNextPage, endCursor), `totalCount`. Vocab Card from `palette-v3.html` shows how each entry should look in the list.

**What to do:**
- Create `src/hooks/useDictionary.ts`:
  - Wraps `useQuery(DICTIONARY_QUERY)` with variables for filters/sort/pagination
  - Exposes `fetchMore()` for loading next page
  - Accepts filter state: `search`, `sortField`, `sortDirection`, `hasCard`, `partOfSpeech`, `topicId`, `status`
- Replace stub `src/pages/DictionaryPage.tsx`:
  - **Search bar**: text input with debounce (~300ms), placed at top. Uses Herbarium input style.
  - **Filter bar**: dropdowns/selects for part of speech (enum from API), card status, hasCard toggle, topic (if topics exist). Use shadcn Select/DropdownMenu.
  - **Sort**: toggle between created_at / updated_at, ASC / DESC
  - **Word list**: render each entry as a VocabCard-style row:
    - Word text (Lisu Bosa font), first definition (truncated), part of speech tag, card status pill (if card exists)
    - Click → navigate to `/dictionary/:id`
  - **Pagination**: "Load more" button or infinite scroll (fetchMore on scroll to bottom)
  - **Total count**: display "X words" at top
- Create `src/components/dictionary/DictionaryEntryCard.tsx`:
  - Based on Vocab Card from `palette-v3.html` (`.vocab-card` styles)
  - Shows: word, first definition, status dot, part of speech, optional topic tags
  - Clickable → navigates to entry detail

**Commit:** `feat(dictionary): implement word list page with search, filters, and pagination`

---

## Task 3: Catalog Search & Add from Catalog

**Goal:** User can search the reference catalog and add words to their dictionary.

**Context:** Two-step flow per `docs/business/WORKFLOWS.md`: (1) `searchCatalog` — autocomplete returns brief matches, (2) `previewRefEntry` — full entry with all senses/translations/examples for the user to review before adding. User picks which senses to include and whether to create a flashcard. Then `createEntryFromCatalog` mutation.

**What to do:**
- Create `src/components/dictionary/CatalogSearch.tsx`:
  - Search input with autocomplete dropdown
  - Debounced query to `searchCatalog` (300ms, min 2 chars)
  - Dropdown shows matching words with brief definitions
  - On select → navigate to preview or open preview dialog
- Create `src/components/dictionary/CatalogPreview.tsx` (dialog/modal or dedicated page):
  - Calls `previewRefEntry` to get full entry data
  - Displays: word, all senses with definitions, translations, examples, pronunciations
  - Checkboxes to select which senses to import (default: all selected)
  - "Create flashcard" checkbox (default: checked)
  - Optional notes field
  - "Add to Dictionary" button → calls `createEntryFromCatalog`
  - On success → toast + redirect to new entry or back to dictionary
  - On error → handle duplicates ("word already in dictionary"), entry limit exceeded
- Add "Add Word" button to DictionaryPage header → opens CatalogSearch (as modal or navigates to search view)
- Wire the search into the main dictionary page flow — could be a top-level action button or integrated into the search bar with a "Search catalog" mode

**Commit:** `feat(dictionary): add catalog search and add-from-catalog flow`

---

## Task 4: Add Custom Word

**Goal:** User can add a word manually when it's not in the catalog.

**Context:** `createEntryCustom` mutation. User provides text + at least one sense (definition + optional partOfSpeech, translations, examples). See `BUSINESS_RULES.md` for limits: text 1-500 chars, max 20 senses, definition 1-2000 chars, translation 1-500 chars, example 1-2000 chars.

**What to do:**
- Create `src/components/dictionary/AddCustomWordForm.tsx`:
  - Word text input
  - Dynamic senses list (add/remove sense):
    - Each sense: definition (required), part of speech (select from enum), translations (dynamic list of text inputs), examples (dynamic list with sentence + optional translation)
  - "Create flashcard" checkbox (default: checked)
  - "Add Word" submit button
  - Client-side validation per business rules
  - On submit → `createEntryCustom` → toast + redirect
- Access: could be a separate dialog/modal or a tab within the "Add Word" flow alongside catalog search
- Indicate clearly when catalog search finds nothing: "Not found in catalog? Add it manually" link

**Commit:** `feat(dictionary): add custom word creation form`

---

## Task 5: Entry Detail Page

**Goal:** Full view and editing of a single dictionary entry.

**Context:** `dictionaryEntry(id)` returns everything: text, notes, senses (with translations, examples), card info, linked topics. This page is the hub for viewing and editing entry content. Content editing uses inline forms (not separate pages).

**What to do:**
- Replace stub `src/pages/DictionaryEntryPage.tsx`:
  - **Header**: word text (Lisu Bosa, large), phonetic transcription if from catalog, part of speech tags
  - **Notes section**: editable text area for user notes (1-5000 chars)
  - **Senses section**: ordered list of senses, each showing:
    - Definition, part of speech
    - Translations (indented list)
    - Examples (styled per source type using Herbarium fonts: EB Garamond for books, Courier Prime for screen, Caveat for lyrics — use Example blocks from `palette-v3.html`)
    - Edit/delete buttons per item
  - **Card section**: if card exists — show state (StatusPill), due date, stability, difficulty, review count, lapse count. If no card — "Create Card" button.
  - **Topics section**: list of linked topics as tags, "Add to topic" button
  - **Actions**: "Delete Entry" button (danger), "Back to Dictionary" link
- Create `src/hooks/useDictionaryEntry.ts` — wraps `useQuery(DICTIONARY_ENTRY_QUERY)`

**Commit:** `feat(dictionary): implement entry detail page`

---

## Task 6: Inline Content Editing

**Goal:** Edit senses, translations, and examples directly on the entry detail page.

**Context:** Each sub-entity (sense, translation, example) has add/update/delete mutations. Senses also support reorder. Editing should feel lightweight — inline forms that expand in place, not full modals. See `BUSINESS_RULES.md` for all field limits and max counts.

**What to do:**
- Create `src/components/dictionary/SenseEditor.tsx`:
  - Inline edit mode: click "Edit" → definition becomes editable, part of speech becomes select
  - Save/Cancel buttons appear on edit
  - "Add Sense" button at bottom of senses list → inline form for new sense
  - "Delete Sense" with confirmation (if it's the last sense, warn that card requires at least one)
  - Drag handle or up/down buttons for reorder (calls `reorderSenses`)
- Create `src/components/dictionary/TranslationEditor.tsx`:
  - Similar inline pattern: list of translations per sense, add/edit/delete
  - Simple text input for each translation
- Create `src/components/dictionary/ExampleEditor.tsx`:
  - Inline edit for examples: sentence + optional translation
  - Add/edit/delete pattern
- Create `src/components/dictionary/UserImageManager.tsx`:
  - List of user images with URL + caption
  - Add: URL input + caption input → `addUserImage`
  - Delete with confirmation
- All editors should update Apollo cache optimistically or refetch entry after mutation
- Validation per business rules: max counts, text length limits

**Commit:** `feat(dictionary): add inline content editing for senses, translations, examples`

---

## Task 7: Delete, Restore & Trash

**Goal:** Soft delete entries, view trash, restore from trash.

**Context:** `deleteEntry` soft-deletes (entry hidden but recoverable). `deletedEntries` lists trash with offset pagination. `restoreEntry` brings it back. Hard delete happens automatically after 30 days (server-side). See `BUSINESS_RULES.md`: batch delete up to 200.

**What to do:**
- Add delete action on DictionaryEntryCard (list item) and DictionaryEntryPage:
  - Confirmation dialog (shadcn Dialog): "Delete '{word}'? You can restore it from trash within 30 days."
  - Call `deleteEntry` → remove from list (update Apollo cache), show success toast
- Create `src/pages/TrashPage.tsx` (or section within dictionary):
  - List of soft-deleted entries with `deletedAt` date
  - "Restore" button per entry → `restoreEntry` → move back to dictionary, toast
  - "X days until permanent deletion" indicator
  - Offset pagination
- Add "Trash" link/button in dictionary page navigation
- Route: `/dictionary/trash`

**Commit:** `feat(dictionary): add soft delete, trash view, and restore`

---

## Task 8: Bulk Import

**Goal:** User can import a list of words at once.

**Context:** `importEntries` mutation accepts `items: [{ text, translations }]`, max 5,000 items. Returns `{ created, skipped, errors }`. Server processes in chunks (50 per transaction). Duplicates within import or existing dictionary are skipped, not failed.

**What to do:**
- Create `src/components/dictionary/BulkImportDialog.tsx`:
  - Text area where user pastes words (one per line, or CSV format: word, translation)
  - OR file upload (CSV/TXT)
  - Parse input into `[{ text, translations }]` array
  - Show preview: count of words to import, first few entries
  - Validate: max 5,000 items, check against entry limit (10,000 total)
  - "Import" button → call `importEntries`
  - Results screen: "X created, Y skipped, Z errors" with details
  - On success → refetch dictionary list
- Add "Import" button to DictionaryPage header (secondary button)

**Commit:** `feat(dictionary): add bulk import dialog`

---

## Task 9: Card Management

**Goal:** Create and delete flashcards from dictionary context.

**Context:** Entry can have 0 or 1 Card (1:1 relationship). Card can only be created if entry has at least one sense. `createCard` for single entry, `batchCreateCards` for multiple (max 100). `deleteCard` removes the card (entry remains).

**What to do:**
- On DictionaryEntryPage:
  - If no card → "Create Flashcard" button (btn-calm/thyme)
  - If card exists → show card info + "Delete Flashcard" button (btn-danger with confirmation)
- On DictionaryPage list:
  - Filter `hasCard: true/false` already in Task 2
  - Batch action: select multiple entries without cards → "Create Cards for Selected" → `batchCreateCards`
  - Or simpler: "Create Cards for All" button that selects all entries without cards (up to 100)
- Card creation error: if entry has zero senses → show error "Add at least one definition first"

**Commit:** `feat(dictionary): add card create/delete from dictionary`

---

## Task 10: Loading & Empty States

**Goal:** Polish all dictionary views for loading and empty data conditions.

**What to do:**
- **Dictionary list loading**: skeleton cards matching VocabCard layout (pulse animation per `palette-v3.html`)
- **Entry detail loading**: skeleton for word header + definition blocks
- **Empty dictionary**: "Your dictionary is empty" message with CTA → "Search catalog" or "Add a word"
- **Empty search results**: "No words matching '{query}'" with suggestion to try different filters
- **Empty trash**: "Trash is empty"
- **Catalog search no results**: "No matches in catalog. Add it as a custom word?" with link to custom form
- **Error states**: error message + retry button on query failures

**Commit:** `feat(dictionary): add loading skeletons and empty states`

---

## Summary

| Task | Description | Key files |
|------|-------------|-----------|
| 1 | GraphQL layer | `src/graphql/queries/dictionary.ts`, `mutations/dictionary.ts`, `mutations/content.ts`, `mutations/cards.ts`, `src/types/dictionary.ts` |
| 2 | Dictionary list page | `src/pages/DictionaryPage.tsx`, `src/components/dictionary/DictionaryEntryCard.tsx` |
| 3 | Catalog search & add | `CatalogSearch.tsx`, `CatalogPreview.tsx` |
| 4 | Custom word form | `AddCustomWordForm.tsx` |
| 5 | Entry detail page | `src/pages/DictionaryEntryPage.tsx` |
| 6 | Inline editing | `SenseEditor.tsx`, `TranslationEditor.tsx`, `ExampleEditor.tsx`, `UserImageManager.tsx` |
| 7 | Delete & trash | `src/pages/TrashPage.tsx`, delete confirmations |
| 8 | Bulk import | `BulkImportDialog.tsx` |
| 9 | Card management | Create/delete card actions |
| 10 | Loading & empty states | Skeletons, empty messages, error handling |

**Total:** 10 tasks, ~10 commits
