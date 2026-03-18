import { useRef, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
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

// ─── Terminal POS color map ──────────────────────────────────────────────────

const TERM_POS_COLORS: Record<string, string> = {
  NOUN: 'var(--term-blue)',
  VERB: 'var(--term-red)',
  ADJECTIVE: 'var(--term-yellow)',
  ADVERB: 'var(--term-cyan)',
  PRONOUN: 'var(--term-blue)',
  PREPOSITION: 'var(--term-orange)',
  CONJUNCTION: 'var(--term-cyan)',
  INTERJECTION: 'var(--term-purple)',
}

const POS_OPTIONS: PartOfSpeech[] = [
  'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN', 'PREPOSITION', 'CONJUNCTION', 'INTERJECTION',
]

const POS_SHORT: Record<string, string> = {
  NOUN: 'n', VERB: 'v', ADJECTIVE: 'adj', ADVERB: 'adv',
  PRONOUN: 'pron', PREPOSITION: 'prep', CONJUNCTION: 'conj', INTERJECTION: 'interj',
}

const SORT_OPTIONS: { field: EntrySortField; dir: SortDirection; label: string }[] = [
  { field: 'TEXT', dir: 'ASC', label: 'a-z' },
  { field: 'CREATED_AT', dir: 'DESC', label: 'new' },
  { field: 'UPDATED_AT', dir: 'DESC', label: 'upd' },
]

const DISPLAY_KEYS: (keyof DisplayOptions)[] = [
  'translation', 'transcription', 'partOfSpeech', 'definition', 'topic',
]

const DISPLAY_SHORT: Record<string, string> = {
  translation: 'trans', transcription: 'ipa', partOfSpeech: 'pos', definition: 'def', topic: 'topic',
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
        className="text-xs hover:underline"
        style={{ color: 'var(--term-text-muted)' }}
      >
        [cols]
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 py-1 min-w-[140px] term-dropdown">
          {DISPLAY_KEYS.map((key) => {
            const checked = displayOptions[key]
            return (
              <button
                key={key}
                type="button"
                onClick={() => onDisplayOptionsChange({ ...displayOptions, [key]: !checked })}
                className="w-full text-left px-3 py-1 text-xs flex items-center gap-1.5 hover:underline"
                style={{ color: checked ? 'var(--term-green)' : 'var(--term-text-dim)' }}
              >
                <span className="w-3 text-center font-mono">{checked ? '+' : '-'}</span>
                {DISPLAY_SHORT[key]}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── Filter dropdown (topics) ───────────────────────────────────────────────

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
        className="text-xs hover:underline"
        style={{ color: activeCount > 0 ? 'var(--term-green)' : 'var(--term-text-muted)' }}
      >
        [topic{activeCount > 0 ? `:${activeCount}` : ''}]
      </button>
      {open && (
        <div className="absolute left-0 top-full mt-1 z-20 py-1 min-w-[180px] term-dropdown">
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
                className="w-full text-left px-3 py-1 text-xs flex items-center gap-1.5 hover:underline"
                style={{ color: active ? 'var(--term-green)' : 'var(--term-text-muted)' }}
              >
                <span className="w-3 text-center font-mono">{active ? '+' : '-'}</span>
                {topic.name}
              </button>
            )
          })}
          {activeCount > 0 && (
            <>
              <div className="mx-2 my-1" style={{ borderTop: '1px dotted var(--term-border)' }} />
              <button
                type="button"
                onClick={() => onTopicIdsChange([])}
                className="w-full text-left px-3 py-1 text-xs hover:underline"
                style={{ color: 'var(--term-text-muted)' }}
              >
                [clear]
              </button>
            </>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Main component ───────────────────────────────────────────────────────────

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
  topicsCount,
}: DictionaryToolbarProps) {
  const { t } = useTranslation('dictionary')
  const { isCompact, sentinelRef } = useScrollCompact()

  const togglePOS = (pos: PartOfSpeech) => {
    if (selectedPOS.includes(pos)) {
      onPOSChange(selectedPOS.filter(p => p !== pos))
    } else {
      onPOSChange([...selectedPOS, pos])
    }
  }

  // Search input
  const searchInput = (compact: boolean) => (
    <div className="relative flex-1" style={{ maxWidth: compact ? 200 : 280 }}>
      <span
        className="absolute left-0 top-1/2 -translate-y-1/2 text-xs select-none pointer-events-none font-mono"
        style={{ color: 'var(--term-text-dim)' }}
      >
        $&gt;
      </span>
      <input
        type="text"
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        placeholder={t('search.placeholder')}
        className="w-full bg-transparent border-none outline-none text-sm pl-5 py-1 font-mono"
        style={{
          color: 'var(--term-text-bright)',
          borderBottom: '1px solid var(--term-border)',
        }}
      />
    </div>
  )

  // Sort buttons
  const sortBtns = (
    <div className="flex items-baseline gap-0.5">
      <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>sort:</span>
      {SORT_OPTIONS.map(opt => {
        const active = sortBy === opt.field && sortDir === opt.dir
        return (
          <button
            key={opt.label}
            type="button"
            onClick={() => onSortChange(opt.field, opt.dir)}
            className="text-xs hover:underline"
            style={{ color: active ? 'var(--term-green)' : 'var(--term-text-muted)' }}
          >
            [{opt.label}]
          </button>
        )
      })}
    </div>
  )

  // POS inline filter — terminal [bracket] style
  const posInline = (
    <div className="flex items-baseline gap-0.5 flex-wrap">
      {POS_OPTIONS.map(pos => {
        const active = selectedPOS.includes(pos)
        const count = posCounts?.[pos]
        const hasWords = !posCounts || (count ?? 0) > 0
        const color = TERM_POS_COLORS[pos] ?? 'var(--term-text)'
        if (!hasWords && !active) return null
        return (
          <button
            key={pos}
            type="button"
            onClick={() => togglePOS(pos)}
            className="text-xs hover:underline tabular-nums"
            style={{
              color: active ? color : 'var(--term-text-dim)',
            }}
          >
            {active ? `[${POS_SHORT[pos]}:${count ?? '?'}]` : `${POS_SHORT[pos]}:${count ?? 0}`}
          </button>
        )
      })}
    </div>
  )

  return (
    <>
      <div ref={sentinelRef} className="h-0" />

      {/* ═══ EXPANDED ══════════════════════════════════════════════════════ */}
      <div className={isCompact ? 'hidden' : 'block'}>
        {/* Row 1: search + actions */}
        <div className="flex items-center gap-3 flex-wrap">
          {searchInput(false)}
          <TopicDropdown
            topics={topics}
            selectedTopicIds={selectedTopicIds}
            onTopicIdsChange={onTopicIdsChange}
          />
          <DisplayDropdown displayOptions={displayOptions} onDisplayOptionsChange={onDisplayOptionsChange} />
          <div className="flex-1" />
          <div className="hidden md:flex">{sortBtns}</div>
        </div>

        {/* Mobile sort */}
        <div className="flex md:hidden mt-1.5">{sortBtns}</div>

        {/* Row 2: POS inline */}
        <div className="mt-1.5">
          {posInline}
        </div>

        {/* Separator */}
        <div className="mt-2" style={{ borderBottom: '1px dotted var(--term-border)' }} />
      </div>

      {/* ═══ COMPACT STICKY ═══════════════════════════════════════════════ */}
      <div
        className={isCompact ? 'block sticky top-0 z-10 pt-1.5 pb-1.5' : 'hidden'}
        style={{ background: 'var(--term-bg)', borderBottom: '1px dotted var(--term-border)' }}
      >
        <div className="flex items-center gap-3">
          {searchInput(true)}
          <TopicDropdown
            topics={topics}
            selectedTopicIds={selectedTopicIds}
            onTopicIdsChange={onTopicIdsChange}
          />
          <DisplayDropdown displayOptions={displayOptions} onDisplayOptionsChange={onDisplayOptionsChange} />
          <div className="flex-1" />
          <span className="text-xs tabular-nums" style={{ color: 'var(--term-text-dim)' }}>
            {totalCount}w/{topicsCount}t
          </span>
        </div>
      </div>
    </>
  )
}
