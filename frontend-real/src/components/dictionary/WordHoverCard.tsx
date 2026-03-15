import React, { useRef, useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/badge'
import { getPosColors } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordHoverCardProps {
  entry: DictionaryEntry
  /** Position of the hovered word, relative to the flow container */
  anchorRect: { x: number; y: number; bottom: number; width: number }
  /** Width of the flow container */
  containerWidth: number
  onOpen: (id: string) => void
  onMouseEnter: () => void
  onMouseLeave: () => void
}

const CARD_WIDTH = 400
const GAP = 12

export function WordHoverCard({
  entry,
  anchorRect,
  containerWidth,
  onOpen,
  onMouseEnter,
  onMouseLeave,
}: WordHoverCardProps) {
  const { t } = useTranslation('dictionary')
  const cardRef = useRef<HTMLDivElement>(null)
  const [measuredHeight, setMeasuredHeight] = useState(0)

  useEffect(() => {
    if (cardRef.current) {
      setMeasuredHeight(cardRef.current.offsetHeight)
    }
  })

  const firstSense = entry.senses[0]
  const pos = firstSense?.partOfSpeech
  const posColors = getPosColors(pos)
  const translations = firstSense?.translations.map((tr) => tr.text).join(', ')
  const firstPronunciation = entry.pronunciations[0]
  const firstExample = firstSense?.examples?.[0]

  // Position: prefer below word, flip above if no space
  const spaceBelow = typeof window !== 'undefined' ? window.innerHeight - anchorRect.bottom : 400
  const showAbove = spaceBelow < (measuredHeight + GAP + 100)
  const top = showAbove
    ? anchorRect.y - measuredHeight - GAP
    : anchorRect.bottom + GAP

  // Horizontal: center on word, clamp to container
  let left = anchorRect.x + anchorRect.width / 2 - CARD_WIDTH / 2
  left = Math.max(0, Math.min(left, containerWidth - CARD_WIDTH))

  const accentColorVar = posColors.cssVar

  return (
    <motion.div
      ref={cardRef}
      className="absolute z-20"
      style={{ width: CARD_WIDTH }}
      initial={{ left, top, opacity: 0, scale: 0.97 }}
      animate={{ left, top, opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.97 }}
      transition={{
        left: { type: 'spring', stiffness: 280, damping: 28, mass: 0.8 },
        top: { type: 'spring', stiffness: 280, damping: 28, mass: 0.8 },
        opacity: { duration: 0.15 },
        scale: { type: 'spring', stiffness: 400, damping: 25 },
      }}
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      role="tooltip"
    >
      <div
        className="rounded-xl bg-bg-card overflow-hidden"
        style={{ boxShadow: 'var(--shadow-hover-card)' }}
      >
        {/* POS accent stripe */}
        <div
          className="h-1"
          style={{ backgroundColor: `var(${accentColorVar})` }}
        />

        <div className="p-5 space-y-3">
          {/* Word + POS */}
          <div className="flex items-start justify-between gap-3">
            <span className="font-orelega text-[28px] leading-tight text-foreground">
              {entry.text}
            </span>
            {pos && (
              <span className={cn(
                'text-[10px] uppercase tracking-[1.5px] shrink-0 pt-2',
                posColors.text,
              )}>
                {t(`pos.${pos}`)}
              </span>
            )}
          </div>

          {/* IPA */}
          {firstPronunciation && (
            <span className="block font-serif text-sm italic text-text-tertiary -mt-1">
              /{firstPronunciation.transcription}/
            </span>
          )}

          {/* Translation + Definition */}
          <div className="space-y-1">
            {translations && (
              <p className="text-[15px] font-medium text-foreground">{translations}</p>
            )}
            {firstSense?.definition && (
              <p className="text-sm text-text-secondary leading-relaxed line-clamp-2">
                {firstSense.definition}
              </p>
            )}
          </div>

          {/* Example */}
          {firstExample && (
            <>
              <div className="h-px bg-border-subtle" />
              <div
                className="pl-3 border-l-2"
                style={{ borderColor: `var(${accentColorVar})` }}
              >
                <p className="font-serif text-sm italic text-text-secondary leading-relaxed">
                  &ldquo;{firstExample.sentence}&rdquo;
                </p>
                {firstExample.translation && (
                  <p className="text-xs text-text-tertiary mt-0.5">
                    {firstExample.translation}
                  </p>
                )}
              </div>
            </>
          )}

          {/* Footer: topics + open link */}
          <div className="flex items-center justify-between pt-1">
            <div className="flex gap-1.5">
              {entry.topics.slice(0, 2).map((topic) => (
                <Badge
                  key={topic.id}
                  variant="outline"
                  className="text-[10px] uppercase tracking-[0.8px] font-normal border-border-subtle text-text-tertiary"
                >
                  {topic.name}
                </Badge>
              ))}
            </div>
            <button
              type="button"
              onClick={() => onOpen(entry.id)}
              className="text-xs font-medium text-poppy hover:text-poppy-hover transition-colors flex items-center gap-1"
            >
              {t('hover_card.open')} →
            </button>
          </div>
        </div>
      </div>
    </motion.div>
  )
}
