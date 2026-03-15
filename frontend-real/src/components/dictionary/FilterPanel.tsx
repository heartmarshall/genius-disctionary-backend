import { useTranslation } from 'react-i18next'
import { motion, AnimatePresence } from 'framer-motion'
import { ArrowDownAZ, Clock, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { PartOfSpeech, EntrySortField, SortDirection, Topic } from '@/types/dictionary'

interface FilterPanelProps {
  open: boolean
  onClose: () => void
  selectedPOS: PartOfSpeech[]
  onPOSChange: (pos: PartOfSpeech[]) => void
  selectedTopicIds: string[]
  onTopicIdsChange: (ids: string[]) => void
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (field: EntrySortField, dir: SortDirection) => void
  topics: Topic[]
}

const POS_OPTIONS: {
  value: PartOfSpeech
  dot: string
  inactiveBg: string
  activeBg: string
  activeText: string
  activeBorder: string
  hoverBg: string
}[] = [
  {
    value: 'NOUN',
    dot: 'bg-cornflower',
    inactiveBg: 'bg-cornflower-light/50',
    hoverBg: 'hover:bg-cornflower-light',
    activeBg: 'bg-cornflower-light',
    activeText: 'text-cornflower-fg',
    activeBorder: 'border-cornflower',
  },
  {
    value: 'VERB',
    dot: 'bg-poppy',
    inactiveBg: 'bg-poppy-light/50',
    hoverBg: 'hover:bg-poppy-light',
    activeBg: 'bg-poppy-light',
    activeText: 'text-poppy-fg',
    activeBorder: 'border-poppy',
  },
  {
    value: 'ADJECTIVE',
    dot: 'bg-goldenrod',
    inactiveBg: 'bg-goldenrod-light/50',
    hoverBg: 'hover:bg-goldenrod-light',
    activeBg: 'bg-goldenrod-light',
    activeText: 'text-goldenrod-fg',
    activeBorder: 'border-goldenrod',
  },
  {
    value: 'ADVERB',
    dot: 'bg-thyme',
    inactiveBg: 'bg-thyme-light/50',
    hoverBg: 'hover:bg-thyme-light',
    activeBg: 'bg-thyme-light',
    activeText: 'text-thyme-fg',
    activeBorder: 'border-thyme',
  },
  {
    value: 'PRONOUN',
    dot: 'bg-cornflower',
    inactiveBg: 'bg-cornflower-light/50',
    hoverBg: 'hover:bg-cornflower-light',
    activeBg: 'bg-cornflower-light',
    activeText: 'text-cornflower-fg',
    activeBorder: 'border-cornflower',
  },
  {
    value: 'PREPOSITION',
    dot: 'bg-goldenrod',
    inactiveBg: 'bg-goldenrod-light/50',
    hoverBg: 'hover:bg-goldenrod-light',
    activeBg: 'bg-goldenrod-light',
    activeText: 'text-goldenrod-fg',
    activeBorder: 'border-goldenrod',
  },
  {
    value: 'CONJUNCTION',
    dot: 'bg-thyme',
    inactiveBg: 'bg-thyme-light/50',
    hoverBg: 'hover:bg-thyme-light',
    activeBg: 'bg-thyme-light',
    activeText: 'text-thyme-fg',
    activeBorder: 'border-thyme',
  },
  {
    value: 'INTERJECTION',
    dot: 'bg-poppy',
    inactiveBg: 'bg-poppy-light/50',
    hoverBg: 'hover:bg-poppy-light',
    activeBg: 'bg-poppy-light',
    activeText: 'text-poppy-fg',
    activeBorder: 'border-poppy',
  },
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

export function FilterPanel({
  open,
  selectedPOS,
  onPOSChange,
  selectedTopicIds,
  onTopicIdsChange,
  sortBy,
  sortDir,
  onSortChange,
  topics,
}: FilterPanelProps) {
  const { t } = useTranslation('dictionary')

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

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: 'auto', opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          transition={{ duration: 0.2, ease: 'easeInOut' }}
          className="overflow-hidden"
        >
          <div className="py-5 border-b border-border-subtle mb-4 space-y-4">
            {/* POS chips — each with its own color identity */}
            <div className="flex flex-wrap gap-2">
              {POS_OPTIONS.map((opt) => {
                const active = selectedPOS.includes(opt.value)
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => togglePOS(opt.value)}
                    className={cn(
                      'inline-flex items-center gap-2 px-3.5 py-2 rounded-lg text-sm border select-none cursor-pointer',
                      active
                        ? `${opt.activeBg} ${opt.activeText} ${opt.activeBorder}`
                        : `${opt.inactiveBg} border-transparent text-text-secondary ${opt.hoverBg}`,
                    )}
                    style={{ transition: 'background-color 150ms, color 150ms, border-color 150ms' }}
                  >
                    <span className={cn('w-2 h-2 rounded-full shrink-0', opt.dot)} />
                    {t(`pos.${opt.value}`)}
                  </button>
                )
              })}
            </div>

            {/* Sort + Topics row */}
            <div className="flex flex-wrap items-center gap-x-5 gap-y-3">
              {/* Sort as icon+text buttons */}
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
                        'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm',
                        active
                          ? 'bg-surface-secondary text-text-primary font-medium'
                          : 'text-text-tertiary hover:text-text-secondary hover:bg-surface-secondary/50',
                      )}
                      style={{ transition: 'background-color 150ms, color 150ms' }}
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
                  <span className="w-px h-5 bg-border-subtle" />
                  <div className="flex flex-wrap items-center gap-2">
                    {topics.map((topic) => {
                      const active = selectedTopicIds.includes(topic.id)
                      return (
                        <button
                          key={topic.id}
                          type="button"
                          onClick={() => toggleTopic(topic.id)}
                          className={cn(
                            'px-3 py-1.5 rounded-lg text-sm border select-none cursor-pointer',
                            active
                              ? 'bg-text-primary text-text-on-accent border-text-primary'
                              : 'border-border-subtle text-text-secondary hover:border-border-default hover:text-text-primary hover:bg-surface-secondary/50',
                          )}
                          style={{ transition: 'background-color 150ms, color 150ms, border-color 150ms' }}
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
        </motion.div>
      )}
    </AnimatePresence>
  )
}
