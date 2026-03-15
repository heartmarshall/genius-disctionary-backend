import { useState, useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { AnimatePresence, motion } from 'framer-motion'
import { ArrowRight } from 'lucide-react'
import { Skeleton } from '@/components/ui/skeleton'
import { WordHoverCard } from './WordHoverCard'
import { getPosColors } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

export type ViewMode = 'flow' | 'list'

interface WordFlowProps {
  entries: DictionaryEntry[]
  loading: boolean
  onWordClick: (id: string) => void
  viewMode?: ViewMode
}

const WORD_STYLE = 'text-[22px]'
const TOOLTIP_DELAY = 400
const LIST_CARD_DELAY = 100
const LIST_CARD_WIDTH = 280

/* ── Skeletons ── */

function WordFlowSkeleton() {
  const widths = [18, 26, 20, 14, 30, 18, 22, 16, 26, 20, 14, 24, 18, 22, 30, 16, 20, 26]
  return (
    <div className="leading-[3] py-4">
      {widths.map((w, i) => (
        <Skeleton
          key={i}
          className="inline-block rounded-md mx-2 my-1 align-baseline"
          style={{ width: `${w * 4}px`, height: '28px' }}
        />
      ))}
    </div>
  )
}

function ListViewSkeleton() {
  return (
    <div className="py-6 columns-3 gap-8">
      {Array.from({ length: 15 }).map((_, i) => (
        <div key={i} className="py-2.5 flex items-baseline gap-2 break-inside-avoid">
          <Skeleton className="h-3 w-4 shrink-0" />
          <Skeleton className="h-6 rounded" style={{ width: `${60 + Math.random() * 80}px` }} />
        </div>
      ))}
    </div>
  )
}

/* ── Popover card for list view ── */

function ListPopoverCard({
  entry,
  anchorRect,
  containerWidth,
  onOpen,
  onMouseEnter,
  onMouseLeave,
}: {
  entry: DictionaryEntry
  anchorRect: { x: number; y: number; bottom: number; width: number }
  containerWidth: number
  onOpen: (id: string) => void
  onMouseEnter: () => void
  onMouseLeave: () => void
}) {
  const { t } = useTranslation('dictionary')
  const cardRef = useRef<HTMLDivElement>(null)
  const [measuredHeight, setMeasuredHeight] = useState(0)

  useEffect(() => {
    if (cardRef.current) setMeasuredHeight(cardRef.current.offsetHeight)
  })

  const firstSense = entry.senses[0]
  const pos = firstSense?.partOfSpeech
  const posColors = getPosColors(pos)
  const translations = firstSense?.translations.map((tr) => tr.text).join(', ')
  const firstPronunciation = entry.pronunciations[0]
  const firstExample = firstSense?.examples?.[0]
  const accentColor = `var(${posColors.cssVar})`

  // Position: center card's arrow on the word's vertical center
  const GAP_X = 18
  const ARROW_FROM_TOP = 24
  const wordCenterX = anchorRect.x + anchorRect.width / 2
  const wordCenterY = anchorRect.y + (anchorRect.bottom - anchorRect.y) / 2
  const goRight = wordCenterX < containerWidth * 0.6

  let left: number
  if (goRight) {
    left = anchorRect.x + anchorRect.width + GAP_X
  } else {
    left = anchorRect.x - LIST_CARD_WIDTH - GAP_X
  }
  left = Math.max(8, Math.min(left, containerWidth - LIST_CARD_WIDTH - 8))

  const top = wordCenterY - ARROW_FROM_TOP
  const arrowTop = ARROW_FROM_TOP - 6

  return (
    <motion.div
        ref={cardRef}
        className="absolute z-20"
        style={{
          width: LIST_CARD_WIDTH,
          filter: 'none',
        }}
        initial={{ left, top, opacity: 0, x: goRight ? -8 : 8 }}
        animate={{ left, top, opacity: 1, x: 0 }}
        exit={{ opacity: 0, x: goRight ? -8 : 8 }}
        transition={{
          left: { type: 'spring', stiffness: 300, damping: 28, mass: 0.8 },
          top: { type: 'spring', stiffness: 300, damping: 28, mass: 0.8 },
          opacity: { duration: 0.12 },
          x: { type: 'spring', stiffness: 400, damping: 28 },
        }}
        onMouseEnter={onMouseEnter}
        onMouseLeave={onMouseLeave}
        role="tooltip"
      >
      {/* Arrow with border, z-10 to sit above card border */}
      <div
        className="absolute w-3 h-3 bg-bg-card rotate-45 z-10 border-border-default"
        style={{
          top: arrowTop,
          ...(goRight
            ? { left: -6, borderLeft: '1px solid', borderBottom: '1px solid', borderColor: 'var(--border-default)' }
            : { right: -6, borderRight: '1px solid', borderTop: '1px solid', borderColor: 'var(--border-default)' }),
        }}
      />
      <div className="relative rounded-xl bg-bg-card border border-border-default">
      <div className="px-4 py-3 space-y-2">
        {/* Pronunciation + POS */}
        <div className="flex items-baseline gap-2">
          {firstPronunciation && (
            <span className="font-serif text-[13px] italic text-text-tertiary">
              /{firstPronunciation.transcription}/
            </span>
          )}
          {pos && (
            <span className="text-[10px] uppercase tracking-wider" style={{ color: accentColor }}>
              {t(`pos.${pos}`, pos)}
            </span>
          )}
        </div>

        {/* Translation */}
        {translations && (
          <p className="text-[14px] text-text-primary leading-snug">{translations}</p>
        )}

        {/* Definition */}
        {firstSense?.definition && (
          <p className="text-[13px] text-text-secondary leading-relaxed line-clamp-2">
            {firstSense.definition}
          </p>
        )}

        {/* Example */}
        {firstExample && (
          <p className="font-serif text-[12px] italic text-text-tertiary leading-relaxed line-clamp-2">
            &ldquo;{firstExample.sentence}&rdquo;
          </p>
        )}

        {/* Topics + Open */}
        <div className="flex items-center gap-2 pt-0.5">
          {entry.topics.slice(0, 2).map((topic) => (
            <span key={topic.id} className="text-[9px] uppercase tracking-[0.8px] text-text-disabled">
              {topic.name}
            </span>
          ))}
          <button
            type="button"
            onClick={() => onOpen(entry.id)}
            className="group inline-flex items-center gap-1 text-[11px] font-medium hover:text-text-primary transition-colors ml-auto"
            style={{ color: accentColor }}
          >
            Open
            <ArrowRight size={11} className="transition-transform group-hover:translate-x-0.5" />
          </button>
        </div>
      </div>
      </div>
    </motion.div>
  )
}

/* ── Main component ── */

export function WordFlow({ entries, loading, onWordClick, viewMode = 'flow' }: WordFlowProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [cardVisible, setCardVisible] = useState(false)
  const [visibleEntry, setVisibleEntry] = useState<DictionaryEntry | null>(null)
  const [anchorRect, setAnchorRect] = useState({ x: 0, y: 0, bottom: 0, width: 0 })
  const timersRef = useRef<{ leave?: number; delay?: number }>({})
  const isShowingRef = useRef(false)

  const setShowing = (v: boolean) => {
    isShowingRef.current = v
    setCardVisible(v)
  }

  const clearAllTimers = () => {
    clearTimeout(timersRef.current.leave)
    clearTimeout(timersRef.current.delay)
    timersRef.current = {}
  }

  const showCard = (entry: DictionaryEntry, el: HTMLElement) => {
    clearAllTimers()
    const rect = el.getBoundingClientRect()
    const containerRect = containerRef.current?.getBoundingClientRect()
    if (!containerRect) return

    setAnchorRect({
      x: rect.left - containerRect.left,
      y: rect.top - containerRect.top,
      bottom: rect.bottom - containerRect.top,
      width: rect.width,
    })
    setVisibleEntry(entry)
    setHoveredId(entry.id)

    if (isShowingRef.current) return

    const delay = viewMode === 'list' ? LIST_CARD_DELAY : TOOLTIP_DELAY
    timersRef.current.delay = window.setTimeout(() => {
      setShowing(true)
    }, delay)
  }

  const startLeave = () => {
    clearAllTimers()
    timersRef.current.leave = window.setTimeout(() => {
      setShowing(false)
      setHoveredId(null)
      setVisibleEntry(null)
    }, 180)
  }

  const cancelLeave = () => {
    clearTimeout(timersRef.current.leave)
    timersRef.current.leave = undefined
  }

  if (loading && entries.length === 0) {
    return viewMode === 'list' ? <ListViewSkeleton /> : <WordFlowSkeleton />
  }

  const containerWidth = containerRef.current?.offsetWidth ?? 800

  return (
    <div ref={containerRef} className="relative">
      {viewMode === 'flow' ? (
        /* ── Flow view ── */
        <div className="leading-[3.2] py-4">
          {entries.map((entry, index) => {
            const firstSense = entry.senses[0]
            const pos = firstSense?.partOfSpeech
            const posColors = getPosColors(pos)
            const isHovered = hoveredId === entry.id

            return (
              <button
                key={entry.id}
                type="button"
                className={cn(
                  'inline-flex items-baseline gap-1 font-orelega leading-tight mr-6 my-1.5 py-0.5',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
                  'text-text-primary cursor-pointer',
                  WORD_STYLE,
                )}
                onClick={() => onWordClick(entry.id)}
                onMouseEnter={(e) => showCard(entry, e.currentTarget)}
                onMouseLeave={startLeave}
                onFocus={(e) => showCard(entry, e.currentTarget)}
                onBlur={startLeave}
              >
                <span className="text-[10px] font-sans font-normal text-text-disabled tabular-nums select-none" aria-hidden>
                  {index + 1}
                </span>
                <span
                  className="inline-block transition-[color,transform,border-color] duration-200 ease-out border-b-2 pb-px"
                  style={{
                    color: isHovered ? `var(${posColors.cssVar})` : undefined,
                    borderColor: isHovered ? `var(${posColors.cssVar})` : 'transparent',
                    transform: isHovered ? 'translateY(-2px)' : 'translateY(0)',
                  }}
                >
                  {entry.text}
                </span>
              </button>
            )
          })}
        </div>
      ) : (
        /* ── List view — 3-column, large text ── */
        <div className="py-6 columns-3 gap-8">
          {entries.map((entry, index) => {
            const firstSense = entry.senses[0]
            const pos = firstSense?.partOfSpeech
            const posColors = getPosColors(pos)
            const isActive = hoveredId === entry.id

            return (
              <div key={entry.id} className="py-2.5 flex items-baseline gap-2 break-inside-avoid">
                <span className="text-[10px] font-sans text-text-disabled tabular-nums select-none w-4 text-right shrink-0" aria-hidden>
                  {index + 1}
                </span>
                <button
                  type="button"
                  className="inline text-left cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                  onClick={() => onWordClick(entry.id)}
                  onMouseEnter={(e) => showCard(entry, e.currentTarget)}
                  onMouseLeave={startLeave}
                  onFocus={(e) => showCard(entry, e.currentTarget)}
                  onBlur={startLeave}
                >
                  <span
                    className="inline-block font-orelega text-[26px] leading-tight border-b-2 pb-px"
                    style={{
                      color: isActive ? `var(${posColors.cssVar})` : 'var(--text-primary)',
                      borderColor: isActive ? `var(${posColors.cssVar})` : 'transparent',
                      transform: isActive ? 'translateX(4px)' : 'translateX(0)',
                      transformOrigin: 'left center',
                      transition: 'color 200ms, border-color 200ms, transform 300ms cubic-bezier(0.23, 1, 0.32, 1)',
                    }}
                  >
                    {entry.text}
                  </span>
                </button>
              </div>
            )
          })}
        </div>
      )}

      {/* Floating card — both modes */}
      <AnimatePresence>
        {visibleEntry && cardVisible && (
          viewMode === 'flow' ? (
            <WordHoverCard
              key="hover-card"
              entry={visibleEntry}
              anchorRect={anchorRect}
              containerWidth={containerWidth}
              onOpen={onWordClick}
              onMouseEnter={cancelLeave}
              onMouseLeave={startLeave}
            />
          ) : (
            <ListPopoverCard
              key="list-popover"
              entry={visibleEntry}
              anchorRect={anchorRect}
              containerWidth={containerWidth}
              onOpen={onWordClick}
              onMouseEnter={cancelLeave}
              onMouseLeave={startLeave}
            />
          )
        )}
      </AnimatePresence>
    </div>
  )
}
