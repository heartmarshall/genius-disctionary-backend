import { useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { motion, AnimatePresence } from 'framer-motion'
import { Skeleton } from '@/components/ui/skeleton'
import { WordHoverCard } from './WordHoverCard'
import { getPosColors } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordFlowProps {
  entries: DictionaryEntry[]
  loading: boolean
  onWordClick: (id: string) => void
}

const WORD_STYLE = 'text-2xl'

const TOOLTIP_DELAY = 400

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

export function WordFlow({ entries, loading, onWordClick }: WordFlowProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const [hoveredId, setHoveredId] = useState<string | null>(null)
  const [cardVisible, setCardVisible] = useState(false)
  const [visibleEntry, setVisibleEntry] = useState<DictionaryEntry | null>(null)
  const [anchorRect, setAnchorRect] = useState({ x: 0, y: 0, bottom: 0, width: 0 })
  const leaveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const delayTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const shownOnceRef = useRef(false)

  const clearTimers = useCallback(() => {
    if (leaveTimerRef.current) clearTimeout(leaveTimerRef.current)
    if (delayTimerRef.current) clearTimeout(delayTimerRef.current)
  }, [])

  const showCard = useCallback((entry: DictionaryEntry, el: HTMLElement) => {
    clearTimers()
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

    if (shownOnceRef.current || cardVisible) {
      // Already shown once or still visible — show immediately
      setCardVisible(true)
    } else {
      delayTimerRef.current = setTimeout(() => {
        setCardVisible(true)
        shownOnceRef.current = true
      }, TOOLTIP_DELAY)
    }
  }, [clearTimers, cardVisible])

  const startLeave = useCallback(() => {
    clearTimers()
    leaveTimerRef.current = setTimeout(() => {
      setCardVisible(false)
      setHoveredId(null)
      // Reset shownOnce after a longer pause — so moving between words stays instant
      setTimeout(() => {
        shownOnceRef.current = false
        setVisibleEntry(null)
      }, 300)
    }, 150)
  }, [clearTimers])

  const cancelLeave = useCallback(() => {
    if (leaveTimerRef.current) {
      clearTimeout(leaveTimerRef.current)
      leaveTimerRef.current = null
    }
  }, [])

  if (loading) return <WordFlowSkeleton />

  const containerWidth = containerRef.current?.offsetWidth ?? 800

  return (
    <div ref={containerRef} className="relative">
      {/* Word flow */}
      <motion.div
        className="leading-[3] py-2"
        initial="hidden"
        animate="visible"
        variants={{
          hidden: {},
          visible: { transition: { staggerChildren: 0.025 } },
        }}
      >
        {entries.map((entry) => {
          const firstSense = entry.senses[0]
          const pos = firstSense?.partOfSpeech
          const posColors = getPosColors(pos)
          const isHovered = hoveredId === entry.id

          return (
            <motion.button
              key={entry.id}
              type="button"
              variants={{
                hidden: { opacity: 0, y: 12 },
                visible: { opacity: 1, y: 0 },
              }}
              transition={{ duration: 0.3, ease: 'easeOut' }}
              className={cn(
                'inline-block font-orelega leading-tight mr-5 my-1 rounded-lg py-0.5',
                'transition-all duration-250 ease-out',
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
              <span
                className="inline-block transition-all duration-250 ease-out"
                style={{
                  color: isHovered ? `var(${posColors.cssVar})` : undefined,
                  transform: isHovered ? 'translateY(-3px)' : 'translateY(0)',
                }}
              >
                {entry.text}
              </span>
            </motion.button>
          )
        })}
      </motion.div>


      {/* Hover card */}
      <AnimatePresence>
        {visibleEntry && cardVisible && (
          <WordHoverCard
            key="hover-card"
            entry={visibleEntry}
            anchorRect={anchorRect}
            containerWidth={containerWidth}
            onOpen={onWordClick}
            onMouseEnter={cancelLeave}
            onMouseLeave={startLeave}
          />
        )}
      </AnimatePresence>
    </div>
  )
}
