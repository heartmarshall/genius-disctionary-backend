# Dictionary Hero Panel Redesign

## Summary

Redesign the dictionary page top panel from a utilitarian header into an "Editorial Hero" block with large typography, integrated stats, and a scroll-driven compact mode. Inspired by chromia.com, discofrogstudio.com, sanrita.ca.

## Scope

Everything above WordFlow: title, stats (word/topic counts), search input, filter button. FilterPanel remains unchanged.

## Design

### Expanded State (scroll at top)

```
┌─────────────────────────────────────────────────────┐
│                    py-10                             │
│                                                     │
│  Dictionary                          (text-5xl 700) │
│  247 words · 12 topics    (text-sm text-secondary)  │
│                                                     │
│  ┌─────────────────────────────┐  ┌──────────────┐  │
│  │ 🔍 Search words...          │  │ ⚙ Filters    │  │
│  └─────────────────────────────┘  └──────────────┘  │
│                    pb-6                              │
└─────────────────────────────────────────────────────┘
```

- Title: `text-5xl font-bold tracking-tight` (Neue Montreal 700). Intentional override of the design system `text-3xl` h1 scale for the editorial hero pattern.
- Stats below title: `text-sm text-text-secondary`, format "247 words · 12 topics"
- Search: `h-12 max-w-lg rounded-lg border-border-default bg-surface-secondary`, Search icon (Lucide `Search`, 18px) inside left padding
- Filter button: right of search, `h-12`, `SlidersHorizontal` icon + "Filters" text + poppy dot indicator. Same height as search input.
- Vertical padding: `py-10`

### Compact State (sticky, after scroll)

```
┌─────────────────────────────────────────────────────┐
│  Dictionary  ┌──────────────────┐  ┌─────────────┐  │
│  (text-xl)   │ Search words...  │  │ ⚙ Filters   │  │
│              └──────────────────┘  └─────────────┘  │
│                border-b border-border-subtle         │
└─────────────────────────────────────────────────────┘
```

- Title: `text-xl font-bold`
- Stats: hidden (`opacity-0 h-0 overflow-hidden`, plus `aria-hidden={isCompact || undefined}` per design system convention — avoids rendering `aria-hidden="false"`)
- Search: `h-10 flex-1 max-w-none`
- Filter button: `h-10`
- Layout: single row (flex items-center) — title left, search center, filters right
- Sticky: `position: sticky; top: 0; z-[5]` (below dropdowns at z-10)
- Background: `bg-white/95 backdrop-blur-sm` (semi-transparent for frosted glass effect). Note: `bg-bg-page/95` won't work because CSS variable `var(--bg-page)` contains a hex value that Tailwind cannot decompose for alpha. Using `bg-white/95` as a pragmatic equivalent since `--bg-page` is `#ffffff`.
- Border: `border-b border-border-subtle`
- Padding: `py-3`

### Scroll Behavior

- Trigger: IntersectionObserver on a sentinel `<div>` rendered by DictionaryHero **above** the sticky hero block (outside the sticky container so it scrolls away naturally)
- When sentinel leaves viewport → `isCompact = true`
- When sentinel re-enters viewport → `isCompact = false`
- CSS transitions only, no Framer Motion

### Transition Strategy

CSS `transition-all` cannot animate `flex-direction`, `position`, or smoothly interpolate large `font-size` changes (causes reflow). The approach:

**Title size change:** Use `transform: scale()` to visually shrink the title without reflow. The title is always rendered at `text-xl` base size. In expanded mode, `transform: scale(2.4)` + `transform-origin: left top` + `will-change-transform` scales it up to ~text-5xl equivalent. In compact mode, `transform: scale(1)`. The scale transition is GPU-accelerated and smooth. Note: `transform` does not affect layout — the title still occupies its `text-xl` box. The container's `py-10` padding absorbs the visual overflow of the scaled title (scaled height ~48px, layout height ~28px, overflow ~20px, absorbed by 40px top padding).

**Layout change (column → row):** This is an instant snap, not animated. The padding and opacity transitions on surrounding elements (stats fade out, search repositions) mask the layout shift, making it feel smooth despite the flex-direction change.

**Position:** The hero is always `position: sticky; top: 0`. In expanded mode, the sentinel div above it keeps it in natural flow. The stickiness only becomes visible when the user scrolls past the sentinel.

| Property         | Expanded                    | Compact                     | Animated |
|-----------------|-----------------------------|-----------------------------|----------|
| title scale     | scale(2.4)                  | scale(1)                    | yes, transition-transform duration-300 |
| padding         | py-10                       | py-3                        | yes, transition-all duration-300 |
| stats opacity   | opacity-100                 | opacity-0                   | yes, transition-opacity duration-200 |
| stats height    | h-auto                      | h-0 overflow-hidden         | no (snap, masked by opacity fade) |
| search height   | h-12                        | h-10                        | yes, transition-all duration-300 |
| search max-w    | max-w-lg                    | max-w-none                  | no (instant) |
| filter h        | h-12                        | h-10                        | yes |
| flex-direction  | flex-col                    | flex-row items-center       | no (instant, masked by other transitions) |
| border-bottom   | border-transparent          | border-border-subtle        | yes, transition-colors duration-300 |
| background      | bg-bg-page                  | bg-white/95 backdrop-blur-sm | yes, transition-colors duration-300 |

### Loading State (skeleton)

When `loading=true`:
- Title: `<Skeleton className="h-10 w-48" />` (expanded) or `<Skeleton className="h-6 w-32" />` (compact)
- Stats: `<Skeleton className="h-4 w-32" />`
- Search and filter button render normally (interactive even during loading)

### Mobile Adaptation (< md)

**Expanded:**
- Title scale: `scale(1.8)` instead of `scale(2.4)` (roughly text-4xl equivalent)
- Padding: `py-6` instead of `py-10`
- Search: `max-w-full` (full width)
- Filter button: icon only (text hidden via `hidden sm:inline`)

**Compact:**
- Title visually hidden but accessible (`sr-only md:not-sr-only md:block`) — preserves page heading for screen readers
- Only search + filter button in sticky bar
- Maximizes space for word flow

### FilterPanel

No changes. Renders after DictionaryHero in DictionaryPage. In compact mode, FilterPanel expands below the sticky bar (it is not part of the sticky container).

## Component Structure

### New Files

**`src/components/dictionary/DictionaryHero.tsx`**

Props:
- `totalCount: number` — word count
- `topicsCount: number` — topic count
- `loading: boolean` — for skeleton state
- `search: string` — search value
- `onSearchChange: (value: string) => void`
- `filtersOpen: boolean`
- `onFiltersToggle: () => void`
- `hasActiveFilters: boolean`

Structure:
```
<>
  <div ref={sentinelRef} /> {/* sentinel, outside sticky */}
  <div className="sticky top-0 z-[5] ..."> {/* hero container */}
    <h1 style={{ transform: `scale(${isCompact ? 1 : 2.4})` }}>Dictionary</h1>
    <div className={stats classes}>247 words · 12 topics</div>
    <div className="flex gap-3">
      <input ... /> {/* search */}
      <button ... /> {/* filters */}
    </div>
  </div>
</>
```

Uses `useScrollCompact()` hook.

**`src/hooks/useScrollCompact.ts`**

```ts
function useScrollCompact(): { isCompact: boolean, sentinelRef: RefObject<HTMLDivElement | null> }
```

Uses IntersectionObserver internally. Observes the sentinel element. Returns `isCompact=true` when sentinel is not intersecting the viewport.

### Modified Files

**`src/pages/DictionaryPage.tsx`**
- Replace h1, search input, filter button, and StatsStrip with `<DictionaryHero />`
- Pass `topicsCount={topics.length}` instead of full topics array
- FilterPanel stays as separate component after DictionaryHero

### Deleted Files

**`src/components/dictionary/StatsStrip.tsx`**
- Stats functionality moves into DictionaryHero

## Design System Compliance

- Colors: all via Tailwind tokens, no hardcoded hex
- Spacing: standard Tailwind scale only
- Fonts: Neue Montreal for title/UI (default font-sans), no changes to word display
- Title size: intentional override of h1 scale (text-3xl → text-5xl via transform scale) for editorial hero pattern
- Icons: Lucide only (Search, SlidersHorizontal)
- No box-shadow
- No CSS-in-JS
- No Framer Motion for hero expand/compact transitions (CSS transitions only). Parent page transitions (dictionary ↔ detail view) remain Framer Motion as before.
- Focus ring preserved on search input and filter button
- Keyboard accessible
- `aria-hidden={isCompact || undefined}` on stats when compact (per design system `aria-hidden` convention)
- z-index: `z-[5]` for sticky hero, below dropdowns (z-10) per design system
