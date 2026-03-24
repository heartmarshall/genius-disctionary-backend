import { useRef, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, SlidersHorizontal, Filter, X } from 'lucide-react'
import { useScrollCompact } from '@/hooks/useScrollCompact'
import type { PartOfSpeech, EntrySortField, SortDirection, Topic } from '@/types/dictionary'

// ─── Display Options ────────────────────────────────────────────────────────

export interface DisplayOptions {
  translation: boolean
  transcription: boolean
  partOfSpeech: boolean
  definition: boolean
  topic: boolean
}

export const DEFAULT_DISPLAY: DisplayOptions = {
  translation: true,
  transcription: true,
  partOfSpeech: true,
  definition: true,
  topic: false,
}

// ─── Props ───────────────────────────────────────────────────────────────────

interface DictionaryToolbarProps {
  totalCount: number
  topicsCount: number
  loading: boolean
  search: string
  onSearchChange: (value: string) => void
  selectedPOS: PartOfSpeech[]
  onPOSChange: (pos: PartOfSpeech[]) => void
  posCounts: Partial<Record<PartOfSpeech, number>>
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (field: EntrySortField, dir: SortDirection) => void
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
  displayOptions: DisplayOptions
  onDisplayOptionsChange: (options: DisplayOptions) => void
}

// ─── Constants ───────────────────────────────────────────────────────────────

const POS_OPTIONS: PartOfSpeech[] = [
  'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN', 'PREPOSITION', 'CONJUNCTION', 'INTERJECTION',
]

const POS_SHORT: Record<string, string> = {
  NOUN: 'n', VERB: 'v', ADJECTIVE: 'adj', ADVERB: 'adv',
  PRONOUN: 'pron', PREPOSITION: 'prep', CONJUNCTION: 'conj', INTERJECTION: 'interj',
}

const SORT_OPTIONS: { field: EntrySortField; dir: SortDirection; label: string; key: string }[] = [
  { field: 'TEXT', dir: 'ASC', label: 'A\u2013Z', key: 'az' },
  { field: 'CREATED_AT', dir: 'DESC', label: 'Newest', key: 'new' },
]

const DISPLAY_KEYS: (keyof DisplayOptions)[] = [
  'translation', 'transcription', 'partOfSpeech',
]

const DISPLAY_LABELS: Record<string, string> = {
  translation: 'Translation',
  transcription: 'IPA',
  partOfSpeech: 'Part of speech',
}

// ─── Display dropdown ────────────────────────────────────────────────────────

function DisplayDropdown({
  displayOptions,
  onDisplayOptionsChange,
}: { displayOptions: DisplayOptions; onDisplayOptionsChange: (o: DisplayOptions) => void }) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(v => !v)}
        className="w-10 h-10 rounded-lg flex items-center justify-center text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary transition-colors duration-150"
        aria-label="Display options"
        title="Display"
      >
        <SlidersHorizontal size={17} />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-2.5 z-20 py-1.5 min-w-[160px] bg-bg-card border border-border-default rounded-lg shadow-bento">
          <p className="px-3 pb-1 text-[11px] font-medium text-text-tertiary uppercase tracking-wider">Show columns</p>
          {DISPLAY_KEYS.map((key) => {
            const checked = displayOptions[key]
            return (
              <button
                key={key}
                type="button"
                onClick={() => onDisplayOptionsChange({ ...displayOptions, [key]: !checked })}
                className="w-full text-left px-3 py-1.5 text-sm flex items-center gap-2 hover:bg-surface-secondary transition-colors duration-150"
              >
                <span className={`w-3.5 h-3.5 rounded border flex items-center justify-center text-[10px] ${checked ? 'bg-cornflower border-cornflower text-text-on-accent' : 'border-border-default'}`}>
                  {checked && '\u2713'}
                </span>
                <span className={checked ? 'text-text-primary' : 'text-text-secondary'}>
                  {DISPLAY_LABELS[key]}
                </span>
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── Topic dropdown ──────────────────────────────────────────────────────────

function TopicDropdown({
  topics,
  selectedTopicIds,
  onTopicIdsChange,
}: {
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  if (topics.length === 0) return null

  const activeCount = selectedTopicIds.length

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={() => setOpen(v => !v)}
        className={`flex items-center gap-1.5 rounded-md px-2.5 py-1.5 text-xs font-medium transition-colors duration-150 ${
          activeCount > 0
            ? 'bg-cornflower-light text-cornflower'
            : 'text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary'
        }`}
      >
        Topic
        {activeCount > 0 && (
          <span className="bg-cornflower text-text-on-accent rounded-full w-4 h-4 flex items-center justify-center text-[10px] font-semibold">
            {activeCount}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute left-0 top-full mt-2.5 z-20 py-1.5 min-w-[200px] bg-bg-card border border-border-default rounded-lg shadow-bento">
          <p className="px-3 pb-1 text-[11px] font-medium text-text-tertiary uppercase tracking-wider">Filter by topic</p>
          {topics.map(topic => {
            const active = selectedTopicIds.includes(topic.id)
            return (
              <button
                key={topic.id}
                type="button"
                onClick={() => {
                  if (active) onTopicIdsChange(selectedTopicIds.filter(tid => tid !== topic.id))
                  else onTopicIdsChange([...selectedTopicIds, topic.id])
                }}
                className="w-full text-left px-3 py-1.5 text-sm flex items-center gap-2 hover:bg-surface-secondary transition-colors duration-150"
              >
                <span className={`w-3.5 h-3.5 rounded border flex items-center justify-center text-[10px] ${active ? 'bg-cornflower border-cornflower text-text-on-accent' : 'border-border-default'}`}>
                  {active && '\u2713'}
                </span>
                <span className={active ? 'text-text-primary font-medium' : 'text-text-secondary'}>
                  {topic.name}
                </span>
              </button>
            )
          })}
          {activeCount > 0 && (
            <>
              <div className="mx-2 my-1 border-t border-border-subtle" />
              <button
                type="button"
                onClick={() => onTopicIdsChange([])}
                className="w-full text-left px-3 py-1.5 text-sm text-text-tertiary hover:text-text-primary hover:bg-surface-secondary transition-colors duration-150"
              >
                Clear selection
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Sort control (segmented) ────────────────────────────────────────────────

function SortControl({
  sortBy,
  sortDir,
  onSortChange,
}: {
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (field: EntrySortField, dir: SortDirection) => void
}) {
  return (
    <div className="flex gap-0.5 bg-surface-secondary rounded-md p-0.5">
      {SORT_OPTIONS.map(opt => {
        const active = sortBy === opt.field && sortDir === opt.dir
        return (
          <button
            key={opt.key}
            type="button"
            onClick={() => onSortChange(opt.field, opt.dir)}
            className={`px-3.5 py-1 rounded text-xs font-semibold transition-all duration-150 ${
              active
                ? 'bg-bg-card text-text-primary shadow-sm'
                : 'text-text-tertiary hover:text-text-secondary'
            }`}
          >
            {opt.label}
          </button>
        )
      })}
    </div>
  )
}

// ─── POS pills ───────────────────────────────────────────────────────────────

function POSPills({
  selectedPOS,
  onToggle,
  posCounts,
}: {
  selectedPOS: PartOfSpeech[]
  onToggle: (pos: PartOfSpeech) => void
  posCounts: Partial<Record<PartOfSpeech, number>>
}) {
  return (
    <div className="flex items-center gap-1 flex-wrap">
      {POS_OPTIONS.map(pos => {
        const active = selectedPOS.includes(pos)
        const count = posCounts?.[pos]
        const hasWords = !posCounts || (count ?? 0) > 0
        if (!hasWords && !active) return null

        return (
          <button
            key={pos}
            type="button"
            onClick={() => onToggle(pos)}
            className={`inline-flex items-center gap-1 rounded-md px-2.5 py-1 text-[11px] font-semibold uppercase tracking-wider transition-all duration-150 border ${
              active
                ? 'border-cornflower bg-cornflower-light text-cornflower'
                : 'border-transparent text-text-disabled hover:text-text-tertiary hover:border-border-default'
            }`}
          >
            {active && <span className="w-1.5 h-1.5 rounded-full bg-cornflower" />}
            <span>{POS_SHORT[pos]}</span>
            <span className="tabular-nums opacity-70">{count ?? 0}</span>
          </button>
        )
      })}
    </div>
  )
}

// ─── Filters bar (progressive disclosure) ─────────────────────────────────

function FiltersDropdown({
  open,
  onToggle,
  activeFilterCount,
  selectedPOS,
  onTogglePOS,
  posCounts,
  topics,
  selectedTopicIds,
  onTopicIdsChange,
}: {
  open: boolean
  onToggle: () => void
  activeFilterCount: number
  selectedPOS: PartOfSpeech[]
  onTogglePOS: (pos: PartOfSpeech) => void
  posCounts: Partial<Record<PartOfSpeech, number>>
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
}) {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) onToggle()
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open, onToggle])

  return (
    <div ref={ref} className="relative">
      <button
        type="button"
        onClick={onToggle}
        title="Filters"
        className={`relative w-10 h-10 rounded-lg flex items-center justify-center shrink-0 transition-colors duration-150 ${
          activeFilterCount > 0 || open
            ? 'bg-cornflower-light text-cornflower'
            : 'text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary'
        }`}
      >
        <Filter size={17} />
        {activeFilterCount > 0 && (
          <span className="absolute -top-1 -right-1 bg-cornflower text-text-on-accent rounded-full w-3.5 h-3.5 flex items-center justify-center text-[9px] font-bold">
            {activeFilterCount}
          </span>
        )}
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-2.5 z-20 py-2 px-2.5 min-w-[240px] bg-bg-card border border-border-default rounded-lg shadow-bento">
          <p className="px-1 pb-1.5 text-[11px] font-medium text-text-tertiary uppercase tracking-wider">Part of speech</p>
          <POSPills selectedPOS={selectedPOS} onToggle={onTogglePOS} posCounts={posCounts} />
          {topics.length > 0 && (
            <>
              <div className="my-2 border-t border-border-subtle" />
              <p className="px-1 pb-1.5 text-[11px] font-medium text-text-tertiary uppercase tracking-wider">Topic</p>
              <div className="flex flex-col">
                {topics.map(topic => {
                  const active = selectedTopicIds.includes(topic.id)
                  return (
                    <button
                      key={topic.id}
                      type="button"
                      onClick={() => {
                        if (active) onTopicIdsChange(selectedTopicIds.filter(tid => tid !== topic.id))
                        else onTopicIdsChange([...selectedTopicIds, topic.id])
                      }}
                      className="text-left px-1 py-1 text-sm flex items-center gap-2 rounded hover:bg-surface-secondary transition-colors duration-150"
                    >
                      <span className={`w-3.5 h-3.5 rounded border flex items-center justify-center text-[10px] ${active ? 'bg-cornflower border-cornflower text-text-on-accent' : 'border-border-default'}`}>
                        {active && '\u2713'}
                      </span>
                      <span className={active ? 'text-text-primary font-medium' : 'text-text-secondary'}>
                        {topic.name}
                      </span>
                    </button>
                  )
                })}
              </div>
            </>
          )}
          {activeFilterCount > 0 && (
            <>
              <div className="my-2 border-t border-border-subtle" />
              <button
                type="button"
                onClick={() => { onTopicIdsChange([]); selectedPOS.forEach(p => onTogglePOS(p)) }}
                className="w-full text-left px-1 py-1 text-sm text-text-tertiary hover:text-text-primary rounded hover:bg-surface-secondary transition-colors duration-150"
              >
                Clear all
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Main component ──────────────────────────────────────────────────────────

export function DictionaryToolbar({
  search,
  onSearchChange,
  selectedPOS,
  onPOSChange,
  posCounts,
  sortBy,
  sortDir,
  onSortChange,
  topics,
  selectedTopicIds,
  onTopicIdsChange,
  displayOptions,
  onDisplayOptionsChange,
  totalCount,
}: DictionaryToolbarProps) {
  const { sentinelRef, isCompact } = useScrollCompact()
  const [filtersOpen, setFiltersOpen] = useState(false)
  const activeFilterCount = selectedPOS.length + selectedTopicIds.length

  const togglePOS = (pos: PartOfSpeech) => {
    if (selectedPOS.includes(pos)) {
      onPOSChange(selectedPOS.filter(p => p !== pos))
    } else {
      onPOSChange([...selectedPOS, pos])
    }
  }

  return (
    <>
      <div ref={sentinelRef} className="h-0" />

      {/* ═══ EXPANDED ══════════════════════════════════════════════════════ */}
      <div className={isCompact ? 'hidden' : 'block'}>
        {/* Search row: card + action icons on the same line */}
        <div className="relative z-20 flex items-center gap-2">
          <div className="flex-1 flex items-center gap-3.5 px-4 py-3 bg-bg-card rounded-[12px] border border-border-default shadow-toolbar">
            <Search size={18} className="text-text-tertiary shrink-0" />
            <input
              type="text"
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder="Search words..."
              className="flex-1 bg-transparent border-none outline-none text-[15px] text-text-primary placeholder:text-text-tertiary font-sans"
            />
            {search && (
              <button
                type="button"
                onClick={() => onSearchChange('')}
                className="text-text-tertiary hover:text-text-secondary shrink-0"
                aria-label="Clear search"
              >
                <X size={16} />
              </button>
            )}
            <div className="h-5 w-px bg-border-default shrink-0" />
            <SortControl sortBy={sortBy} sortDir={sortDir} onSortChange={onSortChange} />
          </div>

          {/* Filters + Display — outside the card */}
          <FiltersDropdown
            open={filtersOpen}
            onToggle={() => setFiltersOpen(v => !v)}
            activeFilterCount={activeFilterCount}
            selectedPOS={selectedPOS}
            onTogglePOS={togglePOS}
            posCounts={posCounts}
            topics={topics}
            selectedTopicIds={selectedTopicIds}
            onTopicIdsChange={onTopicIdsChange}
          />
          <DisplayDropdown displayOptions={displayOptions} onDisplayOptionsChange={onDisplayOptionsChange} />
        </div>
      </div>

      {/* ═══ COMPACT STICKY ═══════════════════════════════════════════════ */}
      <div
        className={isCompact ? 'block sticky top-0 z-10 py-2 bg-bg-page' : 'hidden'}
      >
        <div className="flex items-center gap-3 px-3.5 py-2 bg-bg-card rounded-[10px] border border-border-default shadow-toolbar">
          <Search size={16} className="text-text-tertiary shrink-0" />
          <input
            type="text"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search words..."
            className="flex-1 bg-transparent border-none outline-none text-sm text-text-primary placeholder:text-text-tertiary font-sans"
            style={{ maxWidth: 200 }}
          />
          {search && (
            <button
              type="button"
              onClick={() => onSearchChange('')}
              className="text-text-tertiary hover:text-text-secondary shrink-0"
              aria-label="Clear search"
            >
              <X size={14} />
            </button>
          )}
          <div className="flex-1" />
          <span className="text-xs tabular-nums text-text-tertiary shrink-0">
            {totalCount} words
          </span>
        </div>
      </div>
    </>
  )
}
