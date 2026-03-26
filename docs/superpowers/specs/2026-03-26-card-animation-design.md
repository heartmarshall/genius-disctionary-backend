# Word Detail Card Animation — Design Spec

## Goal

Replace the current simple spring animation on WordDetailInline with a professional "smooth unfold" animation that includes:
- **Enter:** clip-path unfold + height expansion from the word row
- **Exit:** reverse unfold + height collapse
- **Smooth reflow:** list items below the card shift smoothly as it opens/closes
- **No cascading:** all card content appears simultaneously
- **Timing:** Snappy — 250ms enter, 200ms exit

## Current State

- WordDetailInline uses framer-motion `motion.div` with `scale + y` spring animation
- No exit animation — card unmounts instantly
- List items jump when card opens/closes (instant reflow)
- EntryItem in WordList.tsx renders either WordRow or WordDetailInline via ternary

## Architecture

### Two-layer animation wrapper

**Outer layer (height wrapper)** — handles smooth reflow:
- Lives in `EntryItem` (WordList.tsx), wraps the card render
- `AnimatePresence` around a `motion.div`
- Animates `height: 0 → auto` on enter, `height → 0` on exit
- `overflow: hidden` during animation, `overflow: visible` when settled
- This ensures list items below shift smoothly

**Inner layer (visual unfold)** — handles the unfold visual effect:
- Lives inside `WordDetailInline.tsx`, wraps all card content
- Animates `clipPath`, `scaleY`, `translateY` for the unfold look
- `transform-origin: top center`

### Animation Values

**Enter (250ms, ease: cubic-bezier(0.22, 1, 0.36, 1)):**
```
outer:  height 0 → auto
inner:  clipPath inset(0 3% 100% 3% round 18px) → inset(0 0% 0% 0% round 18px)
        scaleY 0.6 → 1
        y -6px → 0
```

**Exit (200ms, ease: cubic-bezier(0.4, 0, 0.7, 1)):**
```
outer:  height → 0
inner:  clipPath inset(0 0% 0% 0% round 18px) → inset(0 3% 100% 3% round 18px)
        scaleY 1 → 0.7
        y 0 → -4px
```

## File Changes

### 1. WordList.tsx — EntryItem component (~lines 234-250)

Add `AnimatePresence` import. Change EntryItem to:
- Always render WordRow when not selected
- When selected: hide WordRow, render card inside a `motion.div` height wrapper with AnimatePresence
- The height wrapper handles `initial`, `animate`, `exit` for height + overflow

```tsx
import { AnimatePresence, motion } from 'framer-motion'

// In EntryItem:
<div data-word-id={entry.id} style={{ scrollMarginTop: '4rem' }}>
  {!isSelected && (
    <WordRow ... />
  )}
  <AnimatePresence>
    {isSelected && (
      <motion.div
        key={`detail-${entry.id}`}
        initial={{ height: 0, overflow: 'hidden' }}
        animate={{ height: 'auto', overflow: 'visible', transition: { duration: 0.25, ease: [0.22, 1, 0.36, 1], overflow: { delay: 0.25 } } }}
        exit={{ height: 0, overflow: 'hidden', transition: { duration: 0.2, ease: [0.4, 0, 0.7, 1] } }}
      >
        {renderDetail?.(entry, globalIndex)}
      </motion.div>
    )}
  </AnimatePresence>
</div>
```

Note: `overflow: visible` is delayed to the end of enter animation so content doesn't leak during unfold. On exit, overflow is immediately set to hidden.

### 2. WordDetailInline.tsx — inner unfold animation

Replace the current `motion.div` wrapper (both loading and loaded states) with clip-path unfold:

```tsx
// Shared transition configs
const unfoldEnter = {
  clipPath: 'inset(0 0% 0% 0% round 18px)',
  scaleY: 1,
  y: 0,
}
const unfoldInitial = {
  clipPath: 'inset(0 3% 100% 3% round 18px)',
  scaleY: 0.6,
  y: -6,
}
const unfoldExit = {
  clipPath: 'inset(0 3% 100% 3% round 18px)',
  scaleY: 0.7,
  y: -4,
}
const enterTransition = { duration: 0.25, ease: [0.22, 1, 0.36, 1] }
const exitTransition = { duration: 0.2, ease: [0.4, 0, 0.7, 1] }
```

Apply to both the loading skeleton wrapper and the main card wrapper:
```tsx
<motion.div
  initial={unfoldInitial}
  animate={unfoldEnter}
  exit={unfoldExit}
  transition={enterTransition}
  style={{ transformOrigin: 'top center' }}
  className="rounded-[18px] overflow-hidden shadow-bento mx-[-24px] my-3 relative z-[2]"
>
```

### 3. DictionaryPage.tsx — scroll stabilization review

The existing scroll stabilization logic (lines 56-170) may partially conflict with smooth reflow. Since height now animates smoothly, the scroll position changes gradually rather than jumping. Review and simplify:
- Keep scroll-into-view on open (smooth scroll if card outside viewport)
- Remove or reduce the complex RAF-based scroll stabilization on close, since the height collapse is now animated

## Edge Cases

- **Switching between words:** When clicking a different word while one is open, the old card exits and new one enters. AnimatePresence handles this via keyed children.
- **Loading state:** The skeleton also uses the unfold animation, so the transition from skeleton to loaded content happens inside the already-open card (no re-animation).
- **Rapid clicks:** AnimatePresence queues exit/enter — no broken states.

## Not In Scope

- Tab switching animations within the card
- Hover micro-interactions
- Staggered/cascade animations for card internals
