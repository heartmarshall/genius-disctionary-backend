import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import type { DictionaryEntry } from '@/types/dictionary'
import type { DisplayOptions } from './DictionaryToolbar'
import { getPosColors } from '@/lib/pos-colors'

interface WordListProps {
  entries: DictionaryEntry[]
  loading: boolean
  selectedWordId: string | null
  onWordClick: (id: string) => void
  displayOptions: DisplayOptions
  sortedAlphabetically?: boolean
  renderDetail?: (entry: DictionaryEntry, index: number) => React.ReactNode
}

const POS_SHORT: Record<string, string> = {
  NOUN: 'n', VERB: 'v', ADJECTIVE: 'adj', ADVERB: 'adv',
  PRONOUN: 'pron', PREPOSITION: 'prep', CONJUNCTION: 'conj',
  INTERJECTION: 'interj', PHRASE: 'phr', IDIOM: 'idiom', OTHER: '?',
}

const SKELETON_ROWS = [
  { word: 110, meta: 50, info: 80 },
  { word: 80, meta: 40, info: 120 },
  { word: 140, meta: 60, info: 70 },
  { word: 90, meta: 35, info: 100 },
  { word: 120, meta: 55, info: 90 },
  { word: 70, meta: 45, info: 110 },
  { word: 100, meta: 50, info: 60 },
  { word: 130, meta: 40, info: 85 },
  { word: 85, meta: 55, info: 95 },
  { word: 115, meta: 35, info: 75 },
  { word: 95, meta: 60, info: 105 },
  { word: 105, meta: 45, info: 65 },
]

function WordListSkeleton() {
  return (
    <div className="flex flex-col gap-0.5">
      {SKELETON_ROWS.map((row, i) => (
        <div key={i} className="flex items-center gap-3 py-2.5 px-3" style={{ animationDelay: `${i * 0.03}s` }}>
          <div className="h-4 rounded-md dict-shimmer" style={{ width: row.word, animationDelay: `${i * 0.05}s` }} />
          <div className="h-3 rounded-md dict-shimmer opacity-60" style={{ width: row.meta, animationDelay: `${i * 0.05 + 0.1}s` }} />
          <div className="flex-1" />
          <div className="h-3 rounded-md dict-shimmer opacity-40" style={{ width: row.info, animationDelay: `${i * 0.05 + 0.2}s` }} />
        </div>
      ))}
    </div>
  )
}

// ─── Island wrapper ──────────────────────────────────────────────────────────

function Island({
  label,
  count,
  rightLabel,
  children,
}: {
  label: string
  count?: number
  rightLabel?: string
  children: React.ReactNode
}) {
  return (
    <div className="bg-island shadow-island rounded-[14px] p-1.5 mb-3" data-letter={label} style={{ scrollMarginTop: '1rem' }}>
      <div className="px-4 pt-2.5 pb-1 flex items-baseline justify-between">
        <div className="flex items-baseline gap-2">
          <span className="font-serif text-[15px] font-bold text-text-disabled">
            {label}
          </span>
          {count != null && (
            <span className="font-mono text-[11px] text-text-disabled opacity-60">
              {count}
            </span>
          )}
        </div>
        {rightLabel && (
          <span className="font-mono text-[10px] text-text-disabled opacity-50">
            {rightLabel}
          </span>
        )}
      </div>
      {children}
    </div>
  )
}

// ─── Time grouping helpers ───────────────────────────────────────────────────

interface TimeGroup {
  key: string
  label: string
  rightLabel?: string
  entries: { entry: DictionaryEntry; globalIndex: number }[]
}

function getTimeBucket(dateStr: string): { key: string; label: string } {
  const d = new Date(dateStr)
  const now = new Date()
  const diff = Math.floor((now.getTime() - d.getTime()) / 86400000)

  if (diff <= 7) return { key: '0-week', label: 'This week' }
  if (diff <= 14) return { key: '1-2week', label: 'Last week' }
  if (diff <= 31) return { key: '2-month', label: 'This month' }

  const months = ['January', 'February', 'March', 'April', 'May', 'June',
    'July', 'August', 'September', 'October', 'November', 'December']
  return { key: `3-${d.getMonth()}`, label: months[d.getMonth()] }
}

function groupByTime(entries: DictionaryEntry[]): TimeGroup[] {
  const groups: Record<string, TimeGroup> = {}
  const order: string[] = []

  entries.forEach((entry, index) => {
    const bucket = getTimeBucket(entry.createdAt)
    if (!groups[bucket.key]) {
      groups[bucket.key] = { key: bucket.key, label: bucket.label, entries: [] }
      order.push(bucket.key)
    }
    groups[bucket.key].entries.push({ entry, globalIndex: index })
  })

  return order.map(k => groups[k])
}

// ─── Letter grouping helpers ─────────────────────────────────────────────────

interface LetterGroup {
  letter: string
  entries: { entry: DictionaryEntry; globalIndex: number }[]
}

function groupByLetter(entries: DictionaryEntry[]): LetterGroup[] {
  const groups: Record<string, LetterGroup> = {}

  entries.forEach((entry, index) => {
    const letter = entry.text[0]?.toUpperCase() ?? '?'
    if (!groups[letter]) {
      groups[letter] = { letter, entries: [] }
    }
    groups[letter].entries.push({ entry, globalIndex: index })
  })

  return Object.keys(groups).sort().map(l => groups[l])
}

// ─── Word Row ────────────────────────────────────────────────────────────────

function WordRow({
  entry,
  index,
  onClick,
  displayOptions,
  showDate,
}: {
  entry: DictionaryEntry
  index: number
  onClick: () => void
  displayOptions: DisplayOptions
  showDate?: boolean
}) {
  const translation = displayOptions.translation
    ? entry.senses[0]?.translations[0]?.text
    : undefined
  const ipa = displayOptions.transcription
    ? entry.pronunciations[0]?.transcription
    : undefined
  const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined
  const primaryPos = entry.senses[0]?.partOfSpeech
  const showPos = displayOptions.partOfSpeech && primaryPos

  return (
    <button
      type="button"
      onClick={onClick}
      className="dict-word-row w-full flex items-center gap-4 py-3.5 px-5 text-left cursor-pointer"
    >
      <span className="font-mono text-[12px] font-medium text-text-disabled select-none min-w-[22px] text-right tabular-nums">
        {index + 1}
      </span>

      <span className="text-[18px] font-semibold text-text-primary shrink-0 tracking-[-0.01em]">
        {entry.text}
      </span>

      {cleanIpa && (
        <span className="font-mono text-[13px] text-text-disabled tracking-wide shrink-0">
          {cleanIpa}
        </span>
      )}

      {showPos && (
        <span className={`text-[12px] font-medium uppercase tracking-wide shrink-0 ${getPosColors(primaryPos).text}`}>
          {POS_SHORT[primaryPos] ?? primaryPos.toLowerCase()}
        </span>
      )}

      <span className="text-[15px] text-text-tertiary italic truncate min-w-0 flex-1">
        {translation ?? ''}
      </span>

      {showDate && (
        <span className="font-mono text-[10px] text-text-disabled shrink-0 opacity-70">
          {entry.createdAt.split('T')[0]?.slice(5)}
        </span>
      )}
    </button>
  )
}

// ─── Entry wrapper — row or expanded card ────────────────────────────────────

function EntryItem({
  entry,
  globalIndex,
  selectedWordId,
  onWordClick,
  displayOptions,
  renderDetail,
  showDate,
}: {
  entry: DictionaryEntry
  globalIndex: number
  selectedWordId: string | null
  onWordClick: (id: string) => void
  displayOptions: DisplayOptions
  renderDetail?: (entry: DictionaryEntry, index: number) => React.ReactNode
  showDate?: boolean
}) {
  const isSelected = selectedWordId === entry.id
  return (
    <div data-word-id={entry.id} style={{ scrollMarginTop: '4rem' }}>
      {isSelected ? (
        renderDetail?.(entry, globalIndex)
      ) : (
        <WordRow
          entry={entry}
          index={globalIndex}
          onClick={() => onWordClick(entry.id)}
          displayOptions={displayOptions}
          showDate={showDate}
        />
      )}
    </div>
  )
}

// ─── Main component ──────────────────────────────────────────────────────────

export function WordList({
  entries,
  loading,
  selectedWordId,
  onWordClick,
  displayOptions,
  sortedAlphabetically,
  renderDetail,
}: WordListProps) {
  const { t } = useTranslation('dictionary')

  const letterGroups = useMemo(
    () => sortedAlphabetically ? groupByLetter(entries) : [],
    [entries, sortedAlphabetically],
  )

  const timeGroups = useMemo(
    () => !sortedAlphabetically ? groupByTime(entries) : [],
    [entries, sortedAlphabetically],
  )

  if (loading) {
    return <WordListSkeleton />
  }

  // ─── A-Z mode ───
  if (sortedAlphabetically) {
    const allLetters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('')
    const usedLetters = new Set(letterGroups.map(g => g.letter))

    return (
      <div className="flex flex-col">
        {/* Alphabet bubbles */}
        <div className="flex gap-1 mb-5 flex-wrap">
          {allLetters.map(l => {
            const has = usedLetters.has(l)
            return (
              <span
                key={l}
                role={has ? 'button' : undefined}
                onClick={has ? () => {
                  const el = document.querySelector(`[data-letter="${l}"]`)
                  el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                } : undefined}
                className={`font-mono text-[11px] font-semibold w-[28px] h-[28px] rounded-full flex items-center justify-center transition-all duration-150 ${
                  has ? 'hover:scale-125 hover:text-text-primary active:scale-110' : ''
                }`}
                style={{
                  color: has ? 'var(--text-secondary)' : 'var(--text-disabled)',
                  opacity: has ? 1 : 0.2,
                  backgroundColor: has ? 'var(--island)' : 'transparent',
                  boxShadow: has ? 'var(--island-shadow)' : 'none',
                  cursor: has ? 'pointer' : 'default',
                }}
              >
                {l}
              </span>
            )
          })}
        </div>

        {letterGroups.map(group => (
          <Island key={group.letter} label={group.letter} count={group.entries.length}>
            {group.entries.map(({ entry, globalIndex }) => (
              <EntryItem
                key={entry.id}
                entry={entry}
                globalIndex={globalIndex}
                selectedWordId={selectedWordId}
                onWordClick={onWordClick}
                displayOptions={displayOptions}
                renderDetail={renderDetail}
              />
            ))}
          </Island>
        ))}
      </div>
    )
  }

  // ─── Newest mode ───
  return (
    <div className="flex flex-col">
      {timeGroups.map(group => (
        <Island
          key={group.key}
          label={group.label}
          count={group.entries.length}
          rightLabel={group.rightLabel}
        >
          {group.entries.map(({ entry, globalIndex }) => (
            <EntryItem
              key={entry.id}
              entry={entry}
              globalIndex={globalIndex}
              selectedWordId={selectedWordId}
              onWordClick={onWordClick}
              displayOptions={displayOptions}
              renderDetail={renderDetail}
              showDate
            />
          ))}
        </Island>
      ))}
    </div>
  )
}
