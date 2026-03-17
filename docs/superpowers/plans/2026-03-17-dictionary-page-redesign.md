# Dictionary Page Redesign — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the Dictionary page with toolbar-first layout, list-only word display, and inline detail card expansion — replacing the current hero/flow/side-panel architecture.

**Architecture:** Replace `DictionaryHero` + `FilterPanel` with unified `DictionaryToolbar`. Replace `WordFlow` (dual-mode) with `WordList` (list-only with inline expansion). Replace side-panel `DictionaryDetailView` with `WordDetailInline` that expands below the selected word. Remove `WordHoverCard`, `WordConnector`, and all flow-view code.

**Tech Stack:** React 19, TypeScript, TailwindCSS 4, Framer Motion, Apollo Client, shadcn/ui, Lucide icons, react-i18next

**Spec:** `docs/superpowers/specs/2026-03-17-dictionary-page-redesign-design.md`

---

## File Structure

### New files
- `frontend-real/src/components/dictionary/DictionaryToolbar.tsx` — unified toolbar with search, POS filters, sort, topic filters, display options dropdown
- `frontend-real/src/components/dictionary/WordList.tsx` — list-only word display with inline detail expansion
- `frontend-real/src/components/dictionary/WordDetailInline.tsx` — inline detail card (adapted from DictionaryDetailView)

### Modified files
- `frontend-real/src/pages/DictionaryPage.tsx` — rewrite orchestration to use new components
- `frontend-real/src/components/dictionary/WordOverview.tsx` — change notes bg from `bg-goldenrod-light/50` to `bg-goldenrod-light`

### Deleted files
- `frontend-real/src/components/dictionary/DictionaryHero.tsx`
- `frontend-real/src/components/dictionary/FilterPanel.tsx`
- `frontend-real/src/components/dictionary/WordFlow.tsx`
- `frontend-real/src/components/dictionary/WordHoverCard.tsx`
- `frontend-real/src/components/dictionary/WordConnector.tsx`

### Unchanged files (reused as-is)
- `frontend-real/src/hooks/useDictionary.ts`
- `frontend-real/src/hooks/useWordDetail.ts`
- `frontend-real/src/hooks/useScrollCompact.ts`
- `frontend-real/src/hooks/useDeleteWord.ts`
- `frontend-real/src/lib/pos-colors.ts`
- `frontend-real/src/components/dictionary/WordEditDialog.tsx`

---

## Task 1: Create DictionaryToolbar component

**Files:**
- Create: `frontend-real/src/components/dictionary/DictionaryToolbar.tsx`

This component replaces both `DictionaryHero` and `FilterPanel`. It renders two rows: search+stats and filters+sort+topics. It has a compact sticky mode on scroll.

- [ ] **Step 1: Create DictionaryToolbar with Row 1 (search + stats)**

Create `frontend-real/src/components/dictionary/DictionaryToolbar.tsx` with:

```tsx
import { useTranslation } from 'react-i18next'
import { Search, SlidersHorizontal, ArrowDownAZ, Clock, RefreshCw } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { useScrollCompact } from '@/hooks/useScrollCompact'
import { cn } from '@/lib/utils'
import type { PartOfSpeech, EntrySortField, SortDirection, Topic } from '@/types/dictionary'

export interface DisplayOptions {
  translation: boolean
  transcription: boolean
  partOfSpeech: boolean
  definition: boolean
  topic: boolean
}

export const DEFAULT_DISPLAY: DisplayOptions = {
  translation: false,
  transcription: false,
  partOfSpeech: false,
  definition: false,
  topic: false,
}

interface DictionaryToolbarProps {
  totalCount: number
  topicsCount: number
  loading: boolean
  search: string
  onSearchChange: (value: string) => void
  selectedPOS: PartOfSpeech[]
  onPOSChange: (pos: PartOfSpeech[]) => void
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (field: EntrySortField, dir: SortDirection) => void
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
  displayOptions: DisplayOptions
  onDisplayOptionsChange: (options: DisplayOptions) => void
}

const POS_OPTIONS: {
  value: PartOfSpeech
  inactiveBg: string
  activeBg: string
  activeText: string
  activeBorder: string
  hoverBg: string
}[] = [
  { value: 'NOUN', inactiveBg: '', activeBg: 'bg-cornflower-light', activeText: 'text-cornflower-fg', activeBorder: 'border-cornflower', hoverBg: 'hover:bg-cornflower-light/60' },
  { value: 'VERB', inactiveBg: '', activeBg: 'bg-poppy-light', activeText: 'text-poppy-fg', activeBorder: 'border-poppy', hoverBg: 'hover:bg-poppy-light/60' },
  { value: 'ADJECTIVE', inactiveBg: '', activeBg: 'bg-goldenrod-light', activeText: 'text-goldenrod-fg', activeBorder: 'border-goldenrod', hoverBg: 'hover:bg-goldenrod-light/60' },
  { value: 'ADVERB', inactiveBg: '', activeBg: 'bg-thyme-light', activeText: 'text-thyme-fg', activeBorder: 'border-thyme', hoverBg: 'hover:bg-thyme-light/60' },
  { value: 'PRONOUN', inactiveBg: '', activeBg: 'bg-cornflower-light', activeText: 'text-cornflower-fg', activeBorder: 'border-cornflower', hoverBg: 'hover:bg-cornflower-light/60' },
  { value: 'PREPOSITION', inactiveBg: '', activeBg: 'bg-goldenrod-light', activeText: 'text-goldenrod-fg', activeBorder: 'border-goldenrod', hoverBg: 'hover:bg-goldenrod-light/60' },
  { value: 'CONJUNCTION', inactiveBg: '', activeBg: 'bg-thyme-light', activeText: 'text-thyme-fg', activeBorder: 'border-thyme', hoverBg: 'hover:bg-thyme-light/60' },
  { value: 'INTERJECTION', inactiveBg: '', activeBg: 'bg-poppy-light', activeText: 'text-poppy-fg', activeBorder: 'border-poppy', hoverBg: 'hover:bg-poppy-light/60' },
]

const SORT_OPTIONS: {
  field: EntrySortField
  dir: SortDirection
  key: string
  icon: typeof ArrowDownAZ
}[] = [
  { field: 'TEXT', dir: 'ASC', key: 'filter.sort_az', icon: ArrowDownAZ },
  { field: 'CREATED_AT', dir: 'DESC', key: 'filter.sort_newest', icon: Clock },
  { field: 'UPDATED_AT', dir: 'DESC', key: 'filter.sort_updated', icon: RefreshCw },
]

export function DictionaryToolbar({
  totalCount,
  topicsCount,
  loading,
  search,
  onSearchChange,
  selectedPOS,
  onPOSChange,
  sortBy,
  sortDir,
  onSortChange,
  topics,
  selectedTopicIds,
  onTopicIdsChange,
  displayOptions,
  onDisplayOptionsChange,
}: DictionaryToolbarProps) {
  const { t } = useTranslation('dictionary')
  const { isCompact, sentinelRef } = useScrollCompact()
  const isInitialLoad = loading && totalCount === 0

  const togglePOS = (pos: PartOfSpeech) => {
    if (selectedPOS.includes(pos)) {
      onPOSChange(selectedPOS.filter((p) => p !== pos))
    } else {
      onPOSChange([...selectedPOS, pos])
    }
  }

  const toggleTopic = (id: string) => {
    if (selectedTopicIds.includes(id)) {
      onTopicIdsChange(selectedTopicIds.filter((t) => t !== id))
    } else {
      onTopicIdsChange([...selectedTopicIds, id])
    }
  }

  const hasActiveFilters = selectedPOS.length > 0 || selectedTopicIds.length > 0

  const [displayOpen, setDisplayOpen] = useState(false)
  const displayRef = useRef<HTMLDivElement>(null)

  // Close display dropdown on outside click
  useEffect(() => {
    if (!displayOpen) return
    const handler = (e: MouseEvent) => {
      if (displayRef.current && !displayRef.current.contains(e.target as Node)) {
        setDisplayOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [displayOpen])

  /* ── Expanded toolbar ── */
  const expandedToolbar = (
    <div className="space-y-4 pt-6 pb-4">
      {/* Row 1: Search + Stats */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-[55%]">
          <Search
            size={16}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary pointer-events-none"
          />
          <input
            type="text"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t('search.placeholder')}
            className="w-full h-9 pl-9 pr-4 rounded-lg bg-surface-secondary text-text-primary placeholder:text-text-tertiary text-sm focus:outline-none focus:border-border-default border border-transparent transition-colors duration-150"
          />
        </div>

        {/* Display options dropdown */}
        <div ref={displayRef} className="relative shrink-0">
          <button
            type="button"
            onClick={() => setDisplayOpen(!displayOpen)}
            className="h-9 px-2.5 rounded-lg flex items-center gap-1.5 text-sm text-text-secondary hover:text-text-primary hover:bg-surface-secondary transition-colors duration-150"
          >
            <SlidersHorizontal size={15} />
          </button>
          {displayOpen && (
            <div className="absolute right-0 top-full mt-1 z-20 bg-bg-card border border-border-default rounded-lg shadow-sm py-1 min-w-[180px]">
              {(Object.keys(displayOptions) as (keyof DisplayOptions)[]).map((key) => (
                <button
                  key={key}
                  type="button"
                  onClick={() => onDisplayOptionsChange({ ...displayOptions, [key]: !displayOptions[key] })}
                  className="w-full flex items-center gap-2.5 px-3 py-2 text-sm text-text-secondary hover:bg-surface-secondary transition-colors"
                >
                  <span className={cn(
                    'w-4 h-4 rounded border flex items-center justify-center text-[10px] transition-colors',
                    displayOptions[key]
                      ? 'bg-cornflower border-cornflower text-white'
                      : 'border-border-default'
                  )}>
                    {displayOptions[key] && '✓'}
                  </span>
                  {t(`display.${key}`)}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="ml-auto shrink-0">
          {isInitialLoad ? (
            <Skeleton className="h-4 w-24" />
          ) : (
            <p className="text-sm text-text-tertiary tabular-nums">
              {totalCount} {t('stats.words')}
              {topicsCount > 0 && (
                <>
                  <span className="mx-1.5">&middot;</span>
                  {topicsCount} {t('stats.topics')}
                </>
              )}
            </p>
          )}
        </div>
      </div>

      {/* Row 2: POS chips | Sort | Topics */}
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        {/* POS chips */}
        <div className="flex flex-wrap items-center gap-1.5">
          {POS_OPTIONS.map((opt) => {
            const active = selectedPOS.includes(opt.value)
            return (
              <button
                key={opt.value}
                type="button"
                onClick={() => togglePOS(opt.value)}
                className={cn(
                  'px-3 py-1.5 rounded-lg text-sm border select-none cursor-pointer transition-all duration-150',
                  active
                    ? `${opt.activeBg} ${opt.activeText} ${opt.activeBorder} font-medium`
                    : `bg-transparent border-border-subtle text-text-secondary ${opt.hoverBg}`,
                )}
              >
                {t(`pos.${opt.value}`)}
              </button>
            )
          })}
        </div>

        {/* Separator */}
        <span className="w-px h-5 bg-border-subtle shrink-0" />

        {/* Sort */}
        <div className="flex items-center gap-1">
          {SORT_OPTIONS.map((opt) => {
            const active = sortBy === opt.field && sortDir === opt.dir
            const Icon = opt.icon
            return (
              <button
                key={opt.key}
                type="button"
                onClick={() => onSortChange(opt.field, opt.dir)}
                className={cn(
                  'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-all duration-150',
                  active
                    ? 'bg-surface-secondary text-text-primary font-medium'
                    : 'text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary/50',
                )}
              >
                <Icon size={14} />
                {t(opt.key)}
              </button>
            )
          })}
        </div>

        {/* Topics */}
        {topics.length > 0 && (
          <>
            <span className="w-px h-5 bg-border-subtle shrink-0" />
            <div className="flex flex-wrap items-center gap-1.5">
              {topics.map((topic) => {
                const active = selectedTopicIds.includes(topic.id)
                return (
                  <button
                    key={topic.id}
                    type="button"
                    onClick={() => toggleTopic(topic.id)}
                    className={cn(
                      'px-3 py-1.5 rounded-lg text-sm border select-none cursor-pointer transition-all duration-150',
                      active
                        ? 'bg-text-primary text-text-on-accent border-text-primary'
                        : 'border-border-subtle text-text-secondary hover:border-border-default hover:text-text-primary hover:bg-surface-secondary/50',
                    )}
                  >
                    {topic.name}
                  </button>
                )
              })}
            </div>
          </>
        )}
      </div>
    </div>
  )

  /* ── Compact sticky toolbar ── */
  const compactToolbar = (
    <div className="sticky top-0 z-10 bg-bg-page -mx-6 px-6 border-b border-border-subtle">
      <div className="flex items-center gap-3 py-2.5">
        {/* Search */}
        <div className="relative flex-1 max-w-sm">
          <Search
            size={15}
            className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary pointer-events-none"
          />
          <input
            type="text"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder={t('search.placeholder')}
            className="w-full h-8 pl-9 pr-4 rounded-lg bg-surface-secondary text-text-primary placeholder:text-text-tertiary text-sm focus:outline-none focus:border-border-default border border-transparent transition-colors duration-150"
          />
        </div>

        {/* Active filter chips (compact) */}
        {hasActiveFilters && (
          <div className="flex items-center gap-1">
            {selectedPOS.map((pos) => {
              const opt = POS_OPTIONS.find((o) => o.value === pos)
              return opt ? (
                <span key={pos} className={cn('px-2 py-0.5 rounded text-xs', opt.activeBg, opt.activeText)}>
                  {t(`pos.${pos}`)}
                </span>
              ) : null
            })}
            {selectedTopicIds.map((id) => {
              const topic = topics.find((tp) => tp.id === id)
              return topic ? (
                <span key={id} className="px-2 py-0.5 rounded text-xs bg-text-primary text-text-on-accent">
                  {topic.name}
                </span>
              ) : null
            })}
          </div>
        )}

        {/* Stats */}
        <p className="text-xs text-text-tertiary tabular-nums ml-auto shrink-0">
          {totalCount} {t('stats.words')}
        </p>
      </div>
    </div>
  )

  return (
    <>
      <div ref={sentinelRef} className="h-0" />
      {isCompact ? compactToolbar : expandedToolbar}
    </>
  )
}
```

Note: add `import { useState, useRef, useEffect } from 'react'` at the top.

- [ ] **Step 2: Verify DictionaryToolbar compiles**

Run: `cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npx tsc --noEmit --pretty 2>&1 | head -30`

Fix any type errors.

- [ ] **Step 3: Commit**

```bash
git add frontend-real/src/components/dictionary/DictionaryToolbar.tsx
git commit -m "feat(dictionary): add DictionaryToolbar component

Unified toolbar replacing DictionaryHero and FilterPanel.
Search, POS filters, sort, topic filters, and display options
all in one component with compact sticky mode on scroll."
```

---

## Task 2: Create WordList component

**Files:**
- Create: `frontend-real/src/components/dictionary/WordList.tsx`

List-only word display. Each row: POS-colored left bar + word (Orelega 24px). Optional second line with display options. Selected word state for inline detail.

- [ ] **Step 1: Create WordList component**

Create `frontend-real/src/components/dictionary/WordList.tsx`:

```tsx
import { useRef } from 'react'
import { Skeleton } from '@/components/ui/skeleton'
import { getPosColors, getPosAccentBorder } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import { useTranslation } from 'react-i18next'
import type { DictionaryEntry } from '@/types/dictionary'
import type { DisplayOptions } from './DictionaryToolbar'

interface WordListProps {
  entries: DictionaryEntry[]
  loading: boolean
  selectedWordId: string | null
  onWordClick: (id: string) => void
  displayOptions: DisplayOptions
  /** Render function for inline detail card, receives entry */
  renderDetail?: (entry: DictionaryEntry) => React.ReactNode
}

function WordListSkeleton() {
  return (
    <div className="space-y-0">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 py-4 border-b border-border-subtle">
          <div className="w-[3px] h-7 rounded-full bg-surface-secondary" />
          <Skeleton className="h-7 rounded" style={{ width: `${100 + Math.random() * 120}px` }} />
        </div>
      ))}
    </div>
  )
}

export function WordList({
  entries,
  loading,
  selectedWordId,
  onWordClick,
  displayOptions,
  renderDetail,
}: WordListProps) {
  const { t } = useTranslation('dictionary')

  if (loading && entries.length === 0) {
    return <WordListSkeleton />
  }

  const hasDisplayOptions = displayOptions.translation || displayOptions.transcription ||
    displayOptions.partOfSpeech || displayOptions.definition || displayOptions.topic

  return (
    <div>
      {entries.map((entry) => {
        const firstSense = entry.senses[0]
        const pos = firstSense?.partOfSpeech
        const posColors = getPosColors(pos)
        const accentBorder = getPosAccentBorder(pos)
        const isSelected = selectedWordId === entry.id

        // Build detail line parts
        const parts: string[] = []
        if (hasDisplayOptions) {
          if (displayOptions.translation) {
            const tr = firstSense?.translations.map((t) => t.text).join(', ')
            if (tr) parts.push(tr)
          }
          if (displayOptions.transcription) {
            const ipa = entry.pronunciations[0]?.transcription
            if (ipa) parts.push(`/${ipa}/`)
          }
          if (displayOptions.partOfSpeech && pos) {
            parts.push(t(`pos.${pos}`))
          }
          if (displayOptions.definition) {
            const def = firstSense?.definition
            if (def) parts.push(def)
          }
          if (displayOptions.topic && entry.topics.length > 0) {
            parts.push(entry.topics.map((tp) => tp.name).join(', '))
          }
        }

        return (
          <div key={entry.id}>
            <button
              type="button"
              data-word-id={entry.id}
              onClick={() => onWordClick(entry.id)}
              className={cn(
                'w-full text-left flex items-start gap-3 py-4 cursor-pointer transition-colors duration-150',
                'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                !isSelected && 'border-b border-border-subtle hover:bg-surface-secondary',
                isSelected && 'bg-surface-secondary',
              )}
              style={{ scrollMarginTop: '4rem' }}
            >
              {/* POS color bar */}
              <div
                className={cn(
                  'w-[3px] shrink-0 rounded-full self-stretch transition-opacity duration-150',
                  isSelected ? 'opacity-100' : 'opacity-60 group-hover:opacity-100',
                )}
                style={{ backgroundColor: `var(${posColors.cssVar})` }}
              />

              <div className="min-w-0">
                <span className="font-orelega text-[24px] leading-tight text-text-primary">
                  {entry.text}
                </span>
                {parts.length > 0 && (
                  <p className="text-sm text-text-secondary mt-0.5 line-clamp-1">
                    {parts.join(' · ')}
                  </p>
                )}
              </div>
            </button>

            {/* Inline detail card */}
            {isSelected && renderDetail?.(entry)}
          </div>
        )
      })}
    </div>
  )
}
```

- [ ] **Step 2: Verify WordList compiles**

Run: `cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npx tsc --noEmit --pretty 2>&1 | head -30`

Fix any type errors.

- [ ] **Step 3: Commit**

```bash
git add frontend-real/src/components/dictionary/WordList.tsx
git commit -m "feat(dictionary): add WordList component

List-only word display with POS color bars, optional display
options, selected state, and slot for inline detail rendering."
```

---

## Task 3: Create WordDetailInline component

**Files:**
- Create: `frontend-real/src/components/dictionary/WordDetailInline.tsx`

Inline detail card adapted from `DictionaryDetailView`. Expands below the selected word with Framer Motion animation. Uses `WordOverview` for content.

- [ ] **Step 1: Create WordDetailInline component**

Create `frontend-real/src/components/dictionary/WordDetailInline.tsx`:

```tsx
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { motion } from 'framer-motion'
import { X, Pencil, Trash2, Volume2 } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { WordOverview } from './WordOverview'
import { WordEditDialog } from './WordEditDialog'
import { Skeleton } from '@/components/ui/skeleton'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useDeleteWord } from '@/hooks/useDeleteWord'
import { getPosColors } from '@/lib/pos-colors'

interface WordDetailInlineProps {
  wordId: string
  onClose: () => void
}

function HeaderSkeleton() {
  return (
    <div className="space-y-3 px-8 py-6">
      <Skeleton className="h-12 w-48" />
      <Skeleton className="h-5 w-32" />
      <div className="flex gap-2">
        <Skeleton className="h-6 w-28 rounded-full" />
        <Skeleton className="h-6 w-28 rounded-full" />
      </div>
    </div>
  )
}

function ContentSkeleton() {
  return (
    <div className="space-y-3 px-8 py-5">
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-4 w-1/2" />
    </div>
  )
}

export function WordDetailInline({ wordId, onClose }: WordDetailInlineProps) {
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(wordId)
  const { requestDelete, confirmDelete, cancelDelete, pendingEntry } = useDeleteWord()
  const [editWordId, setEditWordId] = useState<string | null>(null)

  const primaryPos = entry?.senses[0]?.partOfSpeech
  const posColors = getPosColors(primaryPos)

  const playAudio = async (url: string) => {
    try {
      await new Audio(url).play()
    } catch {
      toast.error(t('error.audioFailed'))
    }
  }

  return (
    <motion.div
      initial={{ height: 0, opacity: 0 }}
      animate={{ height: 'auto', opacity: 1 }}
      exit={{ height: 0, opacity: 0 }}
      transition={{
        height: { type: 'spring', stiffness: 300, damping: 30, mass: 0.8 },
        opacity: { duration: 0.2 },
      }}
      className="overflow-hidden"
    >
      <div className="border-x border-b border-border-default rounded-b-xl bg-bg-card">
        {/* ── Hero header zone — POS-colored background ── */}
        <div
          className="px-8 pt-5 pb-6"
          style={{
            backgroundColor: entry
              ? `color-mix(in srgb, var(${posColors.cssVar}) 22%, white)`
              : 'var(--surface-secondary)',
          }}
        >
          {/* Action bar */}
          <div className="flex items-center justify-between mb-4">
            {entry && (
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditWordId(entry.id)}
                  className="h-7 px-2 text-xs text-text-secondary hover:text-text-primary gap-1"
                >
                  <Pencil size={12} />
                  {t('actions.edit')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => requestDelete(entry)}
                  className="h-7 px-2 text-xs text-text-tertiary hover:text-poppy gap-1"
                >
                  <Trash2 size={12} />
                  {t('actions.delete')}
                </Button>
              </div>
            )}
            {!entry && <div />}
            <button
              type="button"
              onClick={onClose}
              className="flex items-center justify-center h-7 w-7 rounded-md text-text-tertiary hover:text-text-primary transition-colors"
              aria-label="Close"
            >
              <X size={16} />
            </button>
          </div>

          {loading && <HeaderSkeleton />}

          {entry && (
            <div className="space-y-3">
              {/* Word — hero size */}
              <h2 className="font-orelega text-4xl text-text-primary leading-tight">
                {entry.text}
              </h2>

              {/* IPA pronunciations — inline */}
              {entry.pronunciations.length > 0 && (
                <div className="flex items-center gap-3 flex-wrap">
                  {entry.pronunciations.map((pron) => (
                    <div key={pron.id} className="inline-flex items-center gap-1.5">
                      <span className="font-serif text-base text-text-secondary">
                        /{pron.transcription}/
                      </span>
                      {pron.region && (
                        <span className="text-[10px] font-medium uppercase tracking-wider text-text-tertiary">
                          {pron.region}
                        </span>
                      )}
                      {pron.audioUrl && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-5 w-5 text-text-tertiary hover:text-text-primary"
                          onClick={() => playAudio(pron.audioUrl!)}
                        >
                          <Volume2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* Topics */}
              {entry.topics.length > 0 && (
                <div className="flex gap-1.5 flex-wrap pt-1">
                  {entry.topics.map((topic) => (
                    <span
                      key={topic.id}
                      className="text-[10px] uppercase tracking-wider text-text-tertiary border border-border-subtle rounded-full px-2.5 py-0.5"
                    >
                      {topic.name}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Content zone — clean white ── */}
        <div className="px-8 py-5">
          {loading && <ContentSkeleton />}

          {error && (
            <div className="flex flex-col items-center gap-3 py-10 rounded-lg border border-poppy/20 bg-poppy-light">
              <p className="text-sm text-poppy-fg">{t('error.loadFailed')}</p>
              <Button
                variant="outline"
                size="sm"
                className="rounded-full border-poppy/30 text-poppy-fg hover:bg-poppy-light"
                onClick={() => window.location.reload()}
              >
                {t('error.tryAgain')}
              </Button>
            </div>
          )}

          {entry && <WordOverview entry={entry} hideHeader />}
        </div>
      </div>

      {/* Edit dialog */}
      <WordEditDialog
        wordId={editWordId}
        open={editWordId !== null}
        onOpenChange={(open) => !open && setEditWordId(null)}
      />

      {/* Delete confirmation */}
      <AlertDialog open={pendingEntry !== null} onOpenChange={(open) => !open && cancelDelete()}>
        <AlertDialogContent className="rounded-xl">
          <AlertDialogHeader>
            <AlertDialogTitle>{t('delete.confirm')}</AlertDialogTitle>
            <AlertDialogDescription>{t('delete.confirmDescription')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="rounded-full" onClick={cancelDelete}>
              {t('delete.confirmCancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              className="rounded-full bg-poppy text-white hover:bg-poppy/90"
              onClick={() => { confirmDelete(); onClose() }}
            >
              {t('delete.confirmDelete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </motion.div>
  )
}
```

- [ ] **Step 2: Verify WordDetailInline compiles**

Run: `cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npx tsc --noEmit --pretty 2>&1 | head -30`

- [ ] **Step 3: Commit**

```bash
git add frontend-real/src/components/dictionary/WordDetailInline.tsx
git commit -m "feat(dictionary): add WordDetailInline component

Inline detail card with Framer Motion height animation,
POS-colored header (22% mix), and WordOverview content.
Replaces the side-panel DictionaryDetailView."
```

---

## Task 4: Fix WordOverview notes background

**Files:**
- Modify: `frontend-real/src/components/dictionary/WordOverview.tsx:163`

- [ ] **Step 1: Change notes bg from shy to confident**

In `WordOverview.tsx`, change `bg-goldenrod-light/50` to `bg-goldenrod-light`:

```diff
- <div className="rounded-lg bg-goldenrod-light/50 px-4 py-3 space-y-1">
+ <div className="rounded-lg bg-goldenrod-light px-4 py-3 space-y-1">
```

- [ ] **Step 2: Commit**

```bash
git add frontend-real/src/components/dictionary/WordOverview.tsx
git commit -m "fix(dictionary): use confident goldenrod background for notes

Remove /50 opacity modifier — notes section should have a
clearly visible warm background, not a barely-visible tint."
```

---

## Task 5: Rewrite DictionaryPage to use new components

**Files:**
- Modify: `frontend-real/src/pages/DictionaryPage.tsx`

- [ ] **Step 1: Rewrite DictionaryPage**

Replace the entire content of `DictionaryPage.tsx`:

```tsx
import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { AnimatePresence } from 'framer-motion'
import { ChevronDown, BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { DictionaryToolbar } from '@/components/dictionary/DictionaryToolbar'
import { WordList } from '@/components/dictionary/WordList'
import { WordDetailInline } from '@/components/dictionary/WordDetailInline'
import { useDictionary } from '@/hooks/useDictionary'
import type { PartOfSpeech, EntrySortField, SortDirection, DictionaryFilterInput } from '@/types/dictionary'
import { DEFAULT_DISPLAY } from '@/components/dictionary/DictionaryToolbar'
import type { DisplayOptions } from '@/components/dictionary/DictionaryToolbar'

export function DictionaryPage() {
  const { t } = useTranslation('dictionary')

  // Filter state
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech[]>([])
  const [selectedTopicIds, setSelectedTopicIds] = useState<string[]>([])
  const [sortBy, setSortBy] = useState<EntrySortField>('TEXT')
  const [sortDir, setSortDir] = useState<SortDirection>('ASC')
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(DEFAULT_DISPLAY)

  // Detail view state
  const [selectedWordId, setSelectedWordId] = useState<string | null>(null)

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 500)
    return () => clearTimeout(timer)
  }, [search])

  const filter: DictionaryFilterInput = {
    search: debouncedSearch || undefined,
    topicId: selectedTopicIds[0] ?? undefined,
    partOfSpeech: selectedPOS[0] ?? undefined,
    sortField: sortBy,
    sortDirection: sortDir,
  }

  const { entries, totalCount, pageInfo, topics, loading, error, loadMore } = useDictionary(filter)

  const hasActiveFilters = selectedPOS.length > 0 || selectedTopicIds.length > 0 || debouncedSearch !== ''

  const clearAllFilters = () => {
    setSearch('')
    setDebouncedSearch('')
    setSelectedPOS([])
    setSelectedTopicIds([])
    setSortBy('TEXT')
    setSortDir('ASC')
  }

  const handleWordClick = (id: string) => {
    // Toggle: click again to close
    if (selectedWordId === id) {
      setSelectedWordId(null)
      window.history.pushState(null, '', '/dictionary')
      return
    }
    setSelectedWordId(id)
    window.history.pushState(null, '', `/dictionary/${id}`)

    // Auto-scroll to word
    requestAnimationFrame(() => {
      const el = document.querySelector(`[data-word-id="${id}"]`)
      el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  const handleClose = useCallback(() => {
    setSelectedWordId(null)
    window.history.pushState(null, '', '/dictionary')
  }, [])

  // Handle browser back/forward
  useEffect(() => {
    const handlePopState = () => {
      const match = window.location.pathname.match(/^\/dictionary\/(.+)$/)
      setSelectedWordId(match ? match[1] : null)
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  // Handle direct URL access
  useEffect(() => {
    const match = window.location.pathname.match(/^\/dictionary\/(.+)$/)
    if (match) setSelectedWordId(match[1])
  }, [])

  // Keyboard navigation: arrow up/down when detail is open
  useEffect(() => {
    if (!selectedWordId) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleClose()
        return
      }
      if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return

      e.preventDefault()
      const currentIndex = entries.findIndex((entry) => entry.id === selectedWordId)
      if (currentIndex === -1) return

      const nextIndex = e.key === 'ArrowDown'
        ? Math.min(currentIndex + 1, entries.length - 1)
        : Math.max(currentIndex - 1, 0)

      if (nextIndex === currentIndex) return

      const nextEntry = entries[nextIndex]
      setSelectedWordId(nextEntry.id)
      window.history.replaceState(null, '', `/dictionary/${nextEntry.id}`)

      requestAnimationFrame(() => {
        const el = document.querySelector(`[data-word-id="${nextEntry.id}"]`)
        el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      })
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedWordId, entries, handleClose])

  return (
    <div>
      {/* Toolbar */}
      <DictionaryToolbar
        totalCount={totalCount}
        topicsCount={topics.length}
        loading={loading}
        search={search}
        onSearchChange={setSearch}
        selectedPOS={selectedPOS}
        onPOSChange={setSelectedPOS}
        sortBy={sortBy}
        sortDir={sortDir}
        onSortChange={(field, dir) => { setSortBy(field); setSortDir(dir) }}
        topics={topics}
        selectedTopicIds={selectedTopicIds}
        onTopicIdsChange={setSelectedTopicIds}
        displayOptions={displayOptions}
        onDisplayOptionsChange={setDisplayOptions}
      />

      {/* Error */}
      {error && (
        <div className="flex flex-col items-center justify-center gap-3 py-12 mt-6 rounded-xl border border-poppy/20 bg-poppy-light">
          <p className="text-sm text-poppy-fg">{t('error.loadFailed')}</p>
          <Button
            variant="outline"
            size="sm"
            className="rounded-full border-poppy/30 text-poppy-fg hover:bg-poppy-light"
            onClick={() => window.location.reload()}
          >
            {t('error.tryAgain')}
          </Button>
        </div>
      )}

      {/* Word list */}
      {!error && (
        <div className="max-w-[900px] mx-auto px-6 mt-4">
          <WordList
            entries={entries}
            loading={loading}
            selectedWordId={selectedWordId}
            onWordClick={handleWordClick}
            displayOptions={displayOptions}
            renderDetail={(entry) => (
              <AnimatePresence>
                {selectedWordId === entry.id && (
                  <WordDetailInline
                    key={entry.id}
                    wordId={entry.id}
                    onClose={handleClose}
                  />
                )}
              </AnimatePresence>
            )}
          />

          {/* Empty state */}
          {!loading && entries.length === 0 && (
            <div className="flex flex-col items-center justify-center gap-4 py-20">
              <div className="flex items-center justify-center h-20 w-20 rounded-2xl bg-cornflower-light">
                <BookOpen className="h-10 w-10 text-cornflower" />
              </div>
              <p className="text-lg text-text-secondary">
                {hasActiveFilters ? t('empty.noResults') : t('empty.title')}
              </p>
              {hasActiveFilters && (
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-full"
                  onClick={clearAllFilters}
                >
                  {t('filter.clearAll')}
                </Button>
              )}
            </div>
          )}

          {/* Load more */}
          {!loading && entries.length > 0 && pageInfo?.hasNextPage && (
            <div className="flex flex-col items-center gap-3 pt-2 pb-4">
              <p className="text-sm text-text-tertiary">
                {t('pagination.showing', { count: entries.length, total: totalCount })}
              </p>
              <Button
                variant="outline"
                onClick={loadMore}
                className="gap-2 rounded-full px-6 border-border-subtle hover:bg-surface-secondary transition-all duration-200"
              >
                <ChevronDown className="h-4 w-4" />
                {t('pagination.loadMore')}
              </Button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
```

- [ ] **Step 2: Verify DictionaryPage compiles**

Run: `cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npx tsc --noEmit --pretty 2>&1 | head -30`

Fix any type errors.

- [ ] **Step 3: Commit**

```bash
git add frontend-real/src/pages/DictionaryPage.tsx
git commit -m "feat(dictionary): rewrite DictionaryPage with new layout

Toolbar-first design, full-width word list with inline detail
expansion. Removes side-panel, flow view, and hover popups.
Adds keyboard navigation (arrows, Escape) and auto-scroll."
```

---

## Task 6: Delete removed components

**Files:**
- Delete: `frontend-real/src/components/dictionary/DictionaryHero.tsx`
- Delete: `frontend-real/src/components/dictionary/FilterPanel.tsx`
- Delete: `frontend-real/src/components/dictionary/WordFlow.tsx`
- Delete: `frontend-real/src/components/dictionary/WordHoverCard.tsx`
- Delete: `frontend-real/src/components/dictionary/WordConnector.tsx`

- [ ] **Step 1: Delete old component files**

```bash
cd /home/alodi/playgorund/myprojects/genius-disctionary-backend
git rm frontend-real/src/components/dictionary/DictionaryHero.tsx
git rm frontend-real/src/components/dictionary/FilterPanel.tsx
git rm frontend-real/src/components/dictionary/WordFlow.tsx
git rm frontend-real/src/components/dictionary/WordHoverCard.tsx
git rm frontend-real/src/components/dictionary/WordConnector.tsx
```

- [ ] **Step 2: Verify build still compiles**

Run: `cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npx tsc --noEmit --pretty 2>&1 | head -30`

Check that no remaining files import the deleted components.

- [ ] **Step 3: Commit**

```bash
git add -A frontend-real/src/components/dictionary/
git commit -m "refactor(dictionary): remove old components

Remove DictionaryHero, FilterPanel, WordFlow, WordHoverCard,
and WordConnector — all replaced by new toolbar + list + inline
detail architecture."
```

---

## Task 7: Verify and fix

- [ ] **Step 1: Run full build**

```bash
cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npm run build 2>&1 | tail -20
```

- [ ] **Step 2: Fix any build errors**

Address any remaining type errors, missing imports, or unused imports.

- [ ] **Step 3: Manual visual check**

Start the dev server and visually verify:
```bash
cd /home/alodi/playgorund/myprojects/genius-disctionary-backend/frontend-real && npm run dev
```

Check:
- Toolbar renders with search, POS chips, sort, topics
- Word list shows words with POS color bars
- Clicking a word expands inline detail card below it
- Clicking again or × closes it
- Escape key closes detail
- Arrow keys navigate between words
- Display options dropdown works
- Compact toolbar appears on scroll
- Empty state and no-results state work
- Load more button works

- [ ] **Step 4: Final commit if any fixes were needed**

```bash
git add -A
git commit -m "fix(dictionary): address build/visual issues from redesign"
```
