import { useRef, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, SlidersHorizontal, ArrowDownAZ, Clock, RefreshCw, Filter, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Skeleton } from '@/components/ui/skeleton'
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
  translation: false,
  transcription: false,
  partOfSpeech: false,
  definition: false,
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
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (field: EntrySortField, dir: SortDirection) => void
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
  displayOptions: DisplayOptions
  onDisplayOptionsChange: (options: DisplayOptions) => void
}

// ─── Static data ─────────────────────────────────────────────────────────────

const POS_OPTIONS: {
  value: PartOfSpeech
  activeBg: string
  activeText: string
  activeBorder: string
  hoverBg: string
}[] = [
  { value: 'NOUN', activeBg: 'bg-cornflower-light', activeText: 'text-cornflower-fg', activeBorder: 'border-cornflower', hoverBg: 'hover:bg-cornflower-light/60' },
  { value: 'VERB', activeBg: 'bg-poppy-light', activeText: 'text-poppy-fg', activeBorder: 'border-poppy', hoverBg: 'hover:bg-poppy-light/60' },
  { value: 'ADJECTIVE', activeBg: 'bg-goldenrod-light', activeText: 'text-goldenrod-fg', activeBorder: 'border-goldenrod', hoverBg: 'hover:bg-goldenrod-light/60' },
  { value: 'ADVERB', activeBg: 'bg-thyme-light', activeText: 'text-thyme-fg', activeBorder: 'border-thyme', hoverBg: 'hover:bg-thyme-light/60' },
  { value: 'PRONOUN', activeBg: 'bg-cornflower-light', activeText: 'text-cornflower-fg', activeBorder: 'border-cornflower', hoverBg: 'hover:bg-cornflower-light/60' },
  { value: 'PREPOSITION', activeBg: 'bg-goldenrod-light', activeText: 'text-goldenrod-fg', activeBorder: 'border-goldenrod', hoverBg: 'hover:bg-goldenrod-light/60' },
  { value: 'CONJUNCTION', activeBg: 'bg-thyme-light', activeText: 'text-thyme-fg', activeBorder: 'border-thyme', hoverBg: 'hover:bg-thyme-light/60' },
  { value: 'INTERJECTION', activeBg: 'bg-poppy-light', activeText: 'text-poppy-fg', activeBorder: 'border-poppy', hoverBg: 'hover:bg-poppy-light/60' },
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

const DISPLAY_KEYS: (keyof DisplayOptions)[] = [
  'translation',
  'transcription',
  'partOfSpeech',
  'definition',
  'topic',
]

// ─── Sub-components ───────────────────────────────────────────────────────────

interface SearchInputProps {
  value: string
  onChange: (v: string) => void
  placeholder: string
  className?: string
  inputClassName?: string
  iconSize?: number
}

function SearchInput({ value, onChange, placeholder, className, inputClassName, iconSize = 16 }: SearchInputProps) {
  return (
    <div className={cn('relative', className)}>
      <Search
        size={iconSize}
        className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary pointer-events-none"
      />
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className={cn(
          'w-full pl-9 pr-4 rounded-lg bg-surface-secondary text-text-primary placeholder:text-text-tertiary text-sm',
          'border border-transparent focus:border-border-default focus:outline-none transition-colors duration-150',
          inputClassName,
        )}
      />
    </div>
  )
}

interface DisplayDropdownProps {
  displayOptions: DisplayOptions
  onDisplayOptionsChange: (options: DisplayOptions) => void
}

function DisplayDropdown({ displayOptions, onDisplayOptionsChange }: DisplayDropdownProps) {
  const { t } = useTranslation('dictionary')
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const anyActive = DISPLAY_KEYS.some((k) => displayOptions[k])

  return (
    <div ref={containerRef} className="relative shrink-0">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'h-9 w-9 flex items-center justify-center rounded-lg transition-colors duration-150',
          anyActive
            ? 'bg-cornflower-light text-cornflower-fg border border-cornflower'
            : 'text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary border border-transparent',
        )}
        aria-label={t('display.label')}
      >
        <SlidersHorizontal size={15} />
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1.5 z-20 w-48 rounded-xl bg-bg-page border border-border-subtle shadow-md py-2">
          {DISPLAY_KEYS.map((key) => {
            const checked = displayOptions[key]
            return (
              <button
                key={key}
                type="button"
                onClick={() => onDisplayOptionsChange({ ...displayOptions, [key]: !checked })}
                className="w-full flex items-center gap-2.5 px-3 py-2 hover:bg-surface-secondary transition-colors duration-100 text-sm text-text-primary"
              >
                <span
                  className={cn(
                    'w-4 h-4 rounded flex items-center justify-center border shrink-0 transition-colors duration-100',
                    checked
                      ? 'bg-cornflower border-cornflower'
                      : 'bg-transparent border-border-default',
                  )}
                >
                  {checked && (
                    <svg width="10" height="8" viewBox="0 0 10 8" fill="none">
                      <path d="M1 4L3.5 6.5L9 1" stroke="white" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                    </svg>
                  )}
                </span>
                {t(`display.${key}`)}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── POS chip ─────────────────────────────────────────────────────────────────

interface POSChipsProps {
  selectedPOS: PartOfSpeech[]
  onPOSChange: (pos: PartOfSpeech[]) => void
  compact?: boolean
}

function POSChips({ selectedPOS, onPOSChange, compact }: POSChipsProps) {
  const { t } = useTranslation('dictionary')

  const togglePOS = (pos: PartOfSpeech) => {
    if (selectedPOS.includes(pos)) {
      onPOSChange(selectedPOS.filter((p) => p !== pos))
    } else {
      onPOSChange([...selectedPOS, pos])
    }
  }

  const visibleOptions = compact && selectedPOS.length > 0
    ? POS_OPTIONS.filter((o) => selectedPOS.includes(o.value))
    : POS_OPTIONS

  return (
    <>
      {visibleOptions.map((opt) => {
        const active = selectedPOS.includes(opt.value)
        return (
          <button
            key={opt.value}
            type="button"
            onClick={() => togglePOS(opt.value)}
            className={cn(
              'inline-flex items-center px-3 py-1.5 rounded-lg text-sm border select-none cursor-pointer transition-colors duration-150',
              compact ? 'py-1 text-xs' : '',
              active
                ? `${opt.activeBg} ${opt.activeText} ${opt.activeBorder} font-medium`
                : `bg-transparent border-border-subtle text-text-secondary ${opt.hoverBg}`,
            )}
          >
            {t(`pos.${opt.value}`)}
          </button>
        )
      })}
    </>
  )
}

// ─── Sort chips ───────────────────────────────────────────────────────────────

interface SortChipsProps {
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (field: EntrySortField, dir: SortDirection) => void
}

function SortChips({ sortBy, sortDir, onSortChange }: SortChipsProps) {
  const { t } = useTranslation('dictionary')

  return (
    <>
      {SORT_OPTIONS.map((opt) => {
        const active = sortBy === opt.field && sortDir === opt.dir
        const Icon = opt.icon
        return (
          <button
            key={opt.key}
            type="button"
            onClick={() => onSortChange(opt.field, opt.dir)}
            className={cn(
              'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm border select-none cursor-pointer transition-colors duration-150',
              active
                ? 'bg-surface-secondary text-text-primary border-border-default font-medium'
                : 'bg-transparent border-border-subtle text-text-secondary hover:bg-surface-secondary/50 hover:text-text-primary',
            )}
          >
            <Icon size={13} />
            {t(opt.key)}
          </button>
        )
      })}
    </>
  )
}

// ─── Topic chips ──────────────────────────────────────────────────────────────

interface TopicChipsProps {
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
  compact?: boolean
}

function TopicChips({ topics, selectedTopicIds, onTopicIdsChange, compact }: TopicChipsProps) {
  if (topics.length === 0) return null

  const toggleTopic = (id: string) => {
    if (selectedTopicIds.includes(id)) {
      onTopicIdsChange(selectedTopicIds.filter((t) => t !== id))
    } else {
      onTopicIdsChange([...selectedTopicIds, id])
    }
  }

  const visibleTopics = compact && selectedTopicIds.length > 0
    ? topics.filter((t) => selectedTopicIds.includes(t.id))
    : topics

  if (visibleTopics.length === 0) return null

  return (
    <>
      {visibleTopics.map((topic) => {
        const active = selectedTopicIds.includes(topic.id)
        return (
          <button
            key={topic.id}
            type="button"
            onClick={() => toggleTopic(topic.id)}
            className={cn(
              'px-3 py-1.5 rounded-lg text-sm border select-none cursor-pointer transition-colors duration-150',
              compact ? 'py-1 text-xs' : '',
              active
                ? 'bg-text-primary text-text-on-accent border-text-primary font-medium'
                : 'border-border-subtle text-text-secondary hover:border-border-default hover:text-text-primary hover:bg-surface-secondary/50',
            )}
          >
            {topic.name}
          </button>
        )
      })}
    </>
  )
}

// ─── Filter dropdown ─────────────────────────────────────────────────────────

interface FilterDropdownProps {
  selectedPOS: PartOfSpeech[]
  onPOSChange: (pos: PartOfSpeech[]) => void
  topics: Topic[]
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
}

function FilterDropdown({ selectedPOS, onPOSChange, topics, selectedTopicIds, onTopicIdsChange }: FilterDropdownProps) {
  const { t } = useTranslation('dictionary')
  const [open, setOpen] = useState(false)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const activeCount = selectedPOS.length + selectedTopicIds.length
  const hasTopics = topics.length > 0

  return (
    <div ref={containerRef} className="relative shrink-0">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className={cn(
          'h-9 inline-flex items-center gap-1.5 px-3 rounded-lg text-sm transition-colors duration-150',
          activeCount > 0
            ? 'bg-cornflower-light text-cornflower-fg border border-cornflower font-medium'
            : 'text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary border border-transparent',
        )}
        aria-label="Filter"
      >
        <Filter size={14} />
        <span className="hidden sm:inline">{t('filter.label', 'Filter')}</span>
        {activeCount > 0 && (
          <span className="ml-0.5 text-xs bg-cornflower text-white rounded-full w-4.5 h-4.5 inline-flex items-center justify-center leading-none font-medium">
            {activeCount}
          </span>
        )}
      </button>

      {open && (
        <div className="absolute left-0 top-full mt-1.5 z-20 w-[340px] rounded-xl bg-bg-page border border-border-subtle shadow-lg py-3">
          {/* POS section */}
          <div className="px-3 pb-3">
            <span className="text-[10px] uppercase tracking-widest text-text-tertiary font-medium select-none">
              {t('filter.pos_label', 'Part of speech')}
            </span>
            <div className="flex flex-wrap gap-1.5 mt-2">
              <POSChips selectedPOS={selectedPOS} onPOSChange={onPOSChange} />
            </div>
          </div>

          {/* Topics section */}
          {hasTopics && (
            <div className="px-3 pt-3 border-t border-border-subtle/60">
              <span className="text-[10px] uppercase tracking-widest text-text-tertiary font-medium select-none">
                {t('filter.topics_label', 'Topics')}
              </span>
              <div className="flex flex-wrap gap-1.5 mt-2">
                <TopicChips
                  topics={topics}
                  selectedTopicIds={selectedTopicIds}
                  onTopicIdsChange={onTopicIdsChange}
                />
              </div>
            </div>
          )}

          {/* Clear all */}
          {activeCount > 0 && (
            <div className="px-3 pt-3 mt-1 border-t border-border-subtle/60">
              <button
                type="button"
                onClick={() => { onPOSChange([]); onTopicIdsChange([]) }}
                className="text-xs text-text-tertiary hover:text-text-primary transition-colors duration-150"
              >
                {t('filter.clearAll', 'Clear all filters')}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Main component ───────────────────────────────────────────────────────────

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

  return (
    <>
      {/* Sentinel for IntersectionObserver */}
      <div ref={sentinelRef} className="h-0" />

      {/* ═══ EXPANDED ═══════════════════════════════════════════════════════ */}
      <div className={isCompact ? 'hidden' : 'block'}>
        {/* Row 1: Search + filter + display options + stats */}
        <div className="flex items-center gap-2.5">
          <SearchInput
            value={search}
            onChange={onSearchChange}
            placeholder={t('search.placeholder')}
            className="flex-1 max-w-[400px]"
            inputClassName="h-9"
          />

          <FilterDropdown
            selectedPOS={selectedPOS}
            onPOSChange={onPOSChange}
            topics={topics}
            selectedTopicIds={selectedTopicIds}
            onTopicIdsChange={onTopicIdsChange}
          />

          <DisplayDropdown
            displayOptions={displayOptions}
            onDisplayOptionsChange={onDisplayOptionsChange}
          />

          <div className="flex-1" />

          {/* Sort options inline */}
          <div className="hidden md:flex items-center gap-1.5">
            <span className="text-xs text-text-tertiary mr-1">Sort by:</span>
            {SORT_OPTIONS.map((opt) => {
              const active = sortBy === opt.field && sortDir === opt.dir
              return (
                <button
                  key={opt.key}
                  type="button"
                  onClick={() => onSortChange(opt.field, opt.dir)}
                  className={cn(
                    'text-xs transition-colors duration-150 px-1',
                    active
                      ? 'text-text-primary font-medium'
                      : 'text-text-tertiary hover:text-text-secondary',
                  )}
                >
                  {t(opt.key)}
                </button>
              )
            })}
          </div>

          {/* Word/topic count */}
          <div className="shrink-0 text-xs text-text-tertiary tabular-nums">
            {isInitialLoad ? (
              <Skeleton className="h-4 w-28" />
            ) : (
              <>
                {totalCount} {t('stats.words')}
                <span className="mx-1.5">&middot;</span>
                {topicsCount} {t('stats.topics')}
              </>
            )}
          </div>
        </div>

        {/* Row 2: Active filter pills (only shown when filters are active) */}
        {(selectedPOS.length > 0 || selectedTopicIds.length > 0) && (
          <div className="flex flex-wrap items-center gap-1.5 mt-3 pb-4 border-b border-border-subtle/60">
            {selectedPOS.map((pos) => {
              const opt = POS_OPTIONS.find((o) => o.value === pos)
              return (
                <button
                  key={pos}
                  type="button"
                  onClick={() => onPOSChange(selectedPOS.filter((p) => p !== pos))}
                  className={cn(
                    'inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium border transition-colors duration-150',
                    opt ? `${opt.activeBg} ${opt.activeText} ${opt.activeBorder}` : '',
                  )}
                >
                  {t(`pos.${pos}`)}
                  <X size={12} />
                </button>
              )
            })}
            {selectedTopicIds.map((id) => {
              const topic = topics.find((t) => t.id === id)
              if (!topic) return null
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => onTopicIdsChange(selectedTopicIds.filter((tid) => tid !== id))}
                  className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium border bg-text-primary text-text-on-accent border-text-primary transition-colors duration-150"
                >
                  {topic.name}
                  <X size={12} />
                </button>
              )
            })}
          </div>
        )}

        {/* Sort chips — mobile only (when no filters) */}
        {selectedPOS.length === 0 && selectedTopicIds.length === 0 && (
          <div className="flex md:hidden flex-wrap items-center gap-2 mt-3 pb-4 border-b border-border-subtle/60">
            <SortChips sortBy={sortBy} sortDir={sortDir} onSortChange={onSortChange} />
          </div>
        )}

        {/* Bottom border when no active filters on desktop */}
        {selectedPOS.length === 0 && selectedTopicIds.length === 0 && (
          <div className="hidden md:block pb-4 border-b border-border-subtle/60" />
        )}
      </div>

      {/* ═══ COMPACT STICKY ══════════════════════════════════════════════════ */}
      <div
        className={cn(
          'sticky top-0 z-10 bg-bg-page/95 backdrop-blur-sm pt-4 pb-3 border-b border-border-subtle/60',
          isCompact ? 'block' : 'hidden',
        )}
      >
        <div className="flex items-center gap-2">
          <SearchInput
            value={search}
            onChange={onSearchChange}
            placeholder={t('search.placeholder')}
            className="flex-1 max-w-[400px]"
            inputClassName="h-8"
            iconSize={15}
          />

          <FilterDropdown
            selectedPOS={selectedPOS}
            onPOSChange={onPOSChange}
            topics={topics}
            selectedTopicIds={selectedTopicIds}
            onTopicIdsChange={onTopicIdsChange}
          />

          <DisplayDropdown
            displayOptions={displayOptions}
            onDisplayOptionsChange={onDisplayOptionsChange}
          />

          <div className="flex-1" />

          <span className="shrink-0 text-xs text-text-tertiary tabular-nums">
            {totalCount} {t('stats.words')}
            <span className="mx-1">&middot;</span>
            {topicsCount} {t('stats.topics')}
          </span>
        </div>

        {(selectedPOS.length > 0 || selectedTopicIds.length > 0) && (
          <div className="flex flex-wrap items-center gap-1.5 mt-2">
            {selectedPOS.map((pos) => {
              const opt = POS_OPTIONS.find((o) => o.value === pos)
              return (
                <button
                  key={pos}
                  type="button"
                  onClick={() => onPOSChange(selectedPOS.filter((p) => p !== pos))}
                  className={cn(
                    'inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium border transition-colors duration-150',
                    opt ? `${opt.activeBg} ${opt.activeText} ${opt.activeBorder}` : '',
                  )}
                >
                  {t(`pos.${pos}`)}
                  <X size={10} />
                </button>
              )
            })}
            {selectedTopicIds.map((id) => {
              const topic = topics.find((t) => t.id === id)
              if (!topic) return null
              return (
                <button
                  key={id}
                  type="button"
                  onClick={() => onTopicIdsChange(selectedTopicIds.filter((tid) => tid !== id))}
                  className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium border bg-text-primary text-text-on-accent border-text-primary transition-colors duration-150"
                >
                  {topic.name}
                  <X size={10} />
                </button>
              )
            })}
          </div>
        )}
      </div>
    </>
  )
}
