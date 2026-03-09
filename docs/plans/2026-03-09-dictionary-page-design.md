# Dictionary Page — Design

## Overview

Full-featured dictionary page for MyEnglish. Migrates all functionality from frontend-playground except card/study-related features. Follows the Herbarium design system.

## Scope

Included: word list (grid + table), search, filters (topics, part of speech), sorting, quick preview (Sheet), full detail page, editing (modal dialog), deletion with undo.

Excluded: SRS cards, study-related features, import/export.

---

## Architecture & File Structure

### New Files

```
src/
  components/
    dictionary/
      WordCardGrid.tsx        # Grid view (cards)
      WordTable.tsx           # Table view
      SearchToolbar.tsx       # Search + filters + sort + view toggle
      WordSheet.tsx           # Side panel quick preview
      WordEditDialog.tsx      # Edit modal
      WordOverview.tsx        # Reusable word info block (Sheet + EntryPage)
  pages/
    DictionaryPage.tsx        # Main page (overwrite stub)
    DictionaryEntryPage.tsx   # Full word page (overwrite stub)
  graphql/
    queries/
      dictionary.ts           # GraphQL queries & mutations
  types/
    dictionary.ts             # Types: DictionaryEntry, Sense, WordFilter, etc.
  hooks/
    useDictionary.ts          # List query with filters
    useWordDetail.ts          # Single word query
    useDeleteWord.ts          # Delete with undo
    useWordEdit.ts            # Edit form state
  i18n/
    locales/
      en/dictionary.json
      ru/dictionary.json
```

### Data Flow

```
DictionaryPage
  +- SearchToolbar (filter state management)
  +- WordCardGrid | WordTable (display, word selection)
  +- WordSheet (quick preview, buttons: Open / Edit / Delete)
  +- WordEditDialog (edit modal)

DictionaryEntryPage
  +- WordOverview (full info)
  +- WordEditDialog (edit modal)
```

Filter, sort, and view mode state lives in DictionaryPage via useState. View mode (grid/table) persisted to localStorage. Apollo Client manages server state.

---

## Components

### SearchToolbar

Horizontal bar at the top of the page:
- **Search** — Input with Search icon, 500ms debounce
- **Topics filter** — dropdown button with multi-select, badge showing active filter count
- **Part of speech filter** — dropdown button with multi-select (Noun, Verb, Adjective...)
- **Sort** — dropdown: alphabetical, date created, date updated + ASC/DESC direction
- **View toggle** — two icon buttons (LayoutGrid / List), active one highlighted with poppy
- **Result count** — "42 words" on the right, text-secondary

### WordCardGrid

Grid `grid-cols-1 sm:grid-cols-2 lg:grid-cols-3` with gap-4. Each card:
- Word — Orelega One font, text-xl
- Transcription (if present) — EB Garamond, text-secondary
- First sense: part of speech badge + translation(s)
- Topics — small badges, max 3 with "+N"
- Hover: border-color transition to poppy (duration-150)
- Click opens WordSheet

### WordTable

Table with horizontal scroll on mobile. Sticky "Word" column:
- Columns: Word, Translations, Definition, Part of Speech, Topics, Updated
- Row click opens WordSheet
- Row context menu (three dots): Open, Edit, Delete
- Sort by clicking column headers (Word, Updated)

### WordSheet

Side panel (shadcn Sheet, side="right"), width ~480px:
- Header: word in Orelega One + transcription (EB Garamond) + audio play button
- Topic badges
- All senses: part of speech, definition, translations, examples
- Images (if any)
- Notes (if any)
- Footer buttons: "Open full page" (-> /dictionary/:id), "Edit" (-> WordEditDialog), "Delete"

### WordEditDialog

Modal (shadcn Dialog) for editing:
- "Word" text field
- Topic selector
- Notes textarea
- Senses section: add/remove sense, each with part of speech, definition, translations, examples
- Media section: images (URL + caption), pronunciations (transcription + audio URL + region)
- Buttons: Save / Cancel

### WordOverview

Reusable component (used in both Sheet and DictionaryEntryPage):
- Header with pronunciation and audio playback
- Senses with translations and examples
- Images
- Notes

### DictionaryEntryPage

Full page at /dictionary/:id:
- "Back to Dictionary" button
- WordOverview at full container width
- Buttons: "Edit", "Delete"

---

## States & Error Handling

### Loading

Skeletons (per design doc — no spinners):
- WordCardGrid — 6 skeleton cards (pulse animation)
- WordTable — 8 skeleton rows
- WordSheet — skeleton blocks for header, senses, images
- DictionaryEntryPage — skeleton for WordOverview

### Empty State

- Dictionary empty (no words) — "Your dictionary is empty" centered
- No search/filter results — "No words match your filters" + "Clear filters" button

### Error

- List load failure — inline message with reason + "Try again" button
- Save/delete failure — toast (Sonner) with problem description
- Delete undo — toast with "Undo" button for 5 seconds, optimistic Apollo cache update

### Optimistic Updates

- Delete — word removed from list immediately, restored on undo
- Edit — refetch list after successful mutation (modal closes)

---

## GraphQL & Data Types

### Types (types/dictionary.ts)

```typescript
DictionaryEntry {
  id, text, textNormalized, notes,
  senses: Sense[], images: Image[],
  pronunciations: Pronunciation[], topics: Topic[],
  createdAt, updatedAt
}

Sense { id, definition?, partOfSpeech?, translations: Translation[], examples: Example[] }
Translation { id, text }
Example { id, sentence, translation? }
Image { id, url, caption? }
Pronunciation { id, audioUrl?, transcription, region? }
Topic { id, name, description? }

WordFilter { search?, topicIDs?, partOfSpeech?, sortBy?, sortDir?, limit? }
```

### Queries (graphql/queries/dictionary.ts)

- GET_DICTIONARY — word list with filters (id, text, textNormalized, first senses with translations, topics, pronunciations, image count, updatedAt)
- GET_DICTIONARY_ENTRY — full word info (all fields including notes, all senses/examples, images, pronunciations)
- GET_TOPICS — topic list for filter

### Mutations

- UPDATE_WORD — edit word
- DELETE_WORD — delete word

### Apollo Cache

Uses existing typePolicies from apollo.ts — already configured for dictionary with merge strategy by keyArgs.

---

## i18n

Namespace: `dictionary` (en + ru).

Key structure:
- page.title, search.placeholder
- filter.topics, filter.partOfSpeech, filter.clearAll
- sort.alphabetical, sort.created, sort.updated, sort.asc, sort.desc
- view.grid, view.table
- count.words (with plural forms via i18next count interpolation)
- empty.title, empty.noResults
- table.word, table.translations, table.definition, table.partOfSpeech, table.topics, table.updated
- sheet.openFull, sheet.edit, sheet.delete
- edit.title, edit.word, edit.notes, edit.save, edit.cancel, edit.addSense, edit.addTranslation, edit.addExample
- delete.toast, delete.undo
- error.loadFailed, error.saveFailed, error.tryAgain
- entry.backToDict
- pos.noun ... pos.other (part of speech labels)

Russian plural forms: 3 forms via i18next count interpolation.
