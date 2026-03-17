# Dictionary Page Redesign

## Goal

Transform the Dictionary page from a utilitarian word list into a polished, functional interface that balances visual beauty with clean tooling. The page is where users browse their personal vocabulary collection, see progress, and recall words when needed.

## Key Design Principles

- **No "shy" colors** — if something is highlighted, it must be confidently highlighted. No barely-visible tints on white backgrounds. If it's not highlighted, keep it neutral.
- **Toolbar-first** — search and filters are first-class UI elements, not afterthoughts.
- **Full-width inline detail** — word detail expands inline within the list, not in a side panel.
- **Only list view** — flow/stream view is removed entirely.
- **Clean rhythm** — structured rows with consistent spacing, POS color accents, clear visual hierarchy.

## What Gets Removed

- `DictionaryHero` title "Dictionary" and stats hero section → replaced by toolbar
- Flow view mode (keep only list) → `ViewMode` type and toggle removed
- `WordConnector` component (SVG connector line) → no longer needed
- `WordHoverCard` component (popup on hover) → replaced by inline expansion
- `StatsStrip` component → already deleted
- "Show:" display options toggle row → moved to dropdown in toolbar
- `FilterPanel` as a hidden slide-down → filters always visible in toolbar

## Architecture

### Page Layout

```
┌─────────────────────────────────────────────────────────┐
│  TOOLBAR (sticky on scroll)                             │
│  Row 1: Search input (~50-60%) + stats (20 words)       │
│  Row 2: POS chips | Sort buttons | Topic filters        │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  WORD LIST (max-width ~900px, centered)                 │
│  ▎ word row                                             │
│  ▎ word row                                             │
│  ▎ word row (selected)                                  │
│  ┌────────────────────────────────────────────────────┐  │
│  │  INLINE DETAIL CARD (full width, animated)         │  │
│  └────────────────────────────────────────────────────┘  │
│  ▎ word row                                             │
│  ▎ word row                                             │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Component Changes

| Component | Action | Notes |
|-----------|--------|-------|
| `DictionaryPage.tsx` | Rewrite | New layout orchestration, remove flow/detail-panel logic |
| `DictionaryHero.tsx` | Rewrite → `DictionaryToolbar.tsx` | New toolbar component |
| `WordFlow.tsx` | Rewrite → `WordList.tsx` | List-only, with inline expansion |
| `DictionaryDetailView.tsx` | Adapt → `WordDetailInline.tsx` | Inline card instead of side panel |
| `FilterPanel.tsx` | Remove | Filters move into toolbar |
| `WordHoverCard.tsx` | Remove | No hover popups |
| `WordConnector.tsx` | Remove | No connector lines |
| `WordOverview.tsx` | Keep | Reused inside inline detail card |

## Detailed Design

### 1. Toolbar (`DictionaryToolbar.tsx`)

Replaces `DictionaryHero` and `FilterPanel`. Always visible, sticky on scroll.

**Row 1 — Search and meta:**
- Search input: `h-9`, `text-sm`, `bg-surface-secondary`, borderless
- On focus: thin `border-default` appears
- Width: ~50-60% of toolbar
- Right side: word count + topic count in `text-tertiary`

**Row 2 — Filters and sort:**
- POS chips always visible (not hidden behind "Filters" button)
- Chip size: `py-1.5 px-3`, `text-sm` — confident, not tiny
- Inactive: `bg-transparent`, `border-subtle`, `text-secondary`
- Active: POS-colored background, border, and text — use existing `getPosColors()` utility for class names (not dynamic string interpolation, which breaks Tailwind purge)
- Vertical separator (`border-subtle`, 20px height)
- Sort buttons: A-Z / Newest / Updated, same chip-like styling
- Topic filters after second separator (if topics exist)

**Display options:**
- Small dropdown button in toolbar (gear or sliders icon)
- Options: translation, transcription, part of speech, definition, topic
- Checkbox list in popover

**Compact mode on scroll:**
- Row 2 collapses: only active filters shown as small chips + "All filters" expand button
- If no active filters: row 2 hides entirely, only search remains
- Smooth animated transition

### 2. Word List (`WordList.tsx`)

Replaces `WordFlow.tsx`. Full-width, list-only.

**Container:**
- `max-width: 900px`, horizontally centered
- Padding: `px-6`

**Default row (no display options):**
- Only the word itself: Orelega font, `text-[24px]`, `text-primary`
- Left: 3px vertical POS-colored bar
- Rows separated by `border-subtle` (1px)
- Vertical padding: `py-4` per row

**Row with display options enabled:**
- Word on first line (same as above)
- Additional info on second line: `text-sm`, `text-secondary`
- Format: "двусмысленный, неоднозначный · adjective" (joined by middot)
- Only shows fields the user has toggled on

**Hover state:**
- `bg-surface-secondary` full row background
- POS bar: subtle width/opacity increase
- `cursor-pointer`
- Transition: 150ms

**Selected state (word with open detail):**
- `bg-surface-secondary` persistent
- POS bar at full intensity
- No bottom border (flows into card below)

**Skeleton loading:**
- Rows with animated pulse placeholders for word text
- 8-10 skeleton rows

### 3. Inline Detail Card (`WordDetailInline.tsx`)

Replaces `DictionaryDetailView.tsx` side panel. Opens inline below the selected word.

**Animation:**
- Framer Motion: `height: 0 → auto`, `opacity: 0 → 1`
- Spring animation, ~300ms
- List items below smoothly push down

**Card styling:**
- Full width of the list container
- Border: `border-default` on left, bottom, right (top merges with selected word row)
- Background: `bg-bg-card` (white)
- Inner padding: `px-8 py-6` — generous breathing room on full width
- POS-colored left border continues from the word row's POS bar (3px → full card left edge)

**Header zone:**
- Background: POS color at 22% mix (`color-mix(in srgb, var(--pos-color) 22%, white)`) — intentionally stronger than the current 14% to follow the "no shy colors" principle
- Word: `text-4xl` Orelega
- Pronunciation: IPA + region inline
- Topics: border chips, uppercase
- Action buttons: Edit, Delete — top right area
- Close button (×) — top right corner

**Content zone:**
- White background
- Uses existing `WordOverview` component with `hideHeader={true}`
- Definition with POS-colored left border accent
- Examples in `bg-surface-secondary` blocks
- Notes in `bg-goldenrod-light` at full opacity (no `/50` modifier) — confident warm tone, not barely-visible

**Auto-scroll:**
- When card opens, smooth scroll so selected word + top of card are in viewport
- Use `scroll-margin-top` CSS property on word row elements (set to toolbar height) so that `scrollIntoView({ behavior: 'smooth', block: 'start' })` respects the sticky toolbar

**Navigation:**
- Arrow up/down keys to switch to adjacent word without closing
- Card content crossfades to new word

**Closing:**
- Click × button
- Click the selected word again
- Press Escape key

### 4. Inline Detail Card — Error State

If fetching word detail fails:
- Show inline error message inside the card area with a "Retry" button
- Card remains open at reduced height, does not collapse
- Reuse existing error/retry pattern from `DictionaryDetailView`

### 5. Empty and No-Results States

**Empty state (0 words total):**
- Centered in content area
- Simple text: "Your dictionary is empty" / equivalent
- Minimal, no illustrations or heavy decoration

**No results (filters active but 0 matches):**
- Centered text: "No words match your filters"
- "Clear filters" button below, resets all active POS/sort/topic filters
- Preserves the distinction already present in the current codebase

### 6. URL Integration

- Selected word reflected in URL: `/dictionary/:wordId`
- Browser back/forward navigates between selections
- Direct link to a word opens the page with that word expanded

## Technical Notes

- Remove `ViewMode` type and all flow-related code
- Remove `ListDisplayOptions` from `WordFlow` — recreate simpler version in `WordList`
- `useScrollCompact` hook can be reused for toolbar compact behavior; sentinel placed at the bottom edge of the toolbar
- Existing `useDictionary` hook and data layer unchanged
- Existing `WordOverview` component reused as-is inside inline card
- Existing `WordEditDialog` and delete confirmation dialogs reused
- All colors via CSS custom properties, no hardcoded hex values
- Follow Herbarium design system tokens from `frontend-design.md`
- Update `docs/plans/frontend-design.md` to remove references to grid/flow view modes — only list view remains
- Existing "Load more" button with cursor pagination is preserved as-is
- Keyboard navigation (arrow up/down) only active when inline card is open; stops at first/last loaded word (does not auto-fetch more)
