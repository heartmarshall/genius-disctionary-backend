import { useTranslation } from 'react-i18next'
import { motion, AnimatePresence } from 'framer-motion'
import { X } from 'lucide-react'
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

const POS_OPTIONS: { value: PartOfSpeech; activeClass: string }[] = [
  { value: 'NOUN', activeClass: 'bg-cornflower-light text-cornflower-fg border-cornflower' },
  { value: 'VERB', activeClass: 'bg-poppy-light text-poppy-fg border-poppy' },
  { value: 'ADJECTIVE', activeClass: 'bg-goldenrod-light text-goldenrod-fg border-goldenrod' },
  { value: 'ADVERB', activeClass: 'bg-thyme-light text-thyme-fg border-thyme' },
  { value: 'PRONOUN', activeClass: 'bg-cornflower-light text-cornflower-fg border-cornflower' },
  { value: 'PREPOSITION', activeClass: 'bg-goldenrod-light text-goldenrod-fg border-goldenrod' },
  { value: 'CONJUNCTION', activeClass: 'bg-thyme-light text-thyme-fg border-thyme' },
  { value: 'INTERJECTION', activeClass: 'bg-poppy-light text-poppy-fg border-poppy' },
  { value: 'PHRASE', activeClass: 'bg-text-primary text-text-on-accent border-text-primary' },
  { value: 'IDIOM', activeClass: 'bg-text-primary text-text-on-accent border-text-primary' },
  { value: 'OTHER', activeClass: 'bg-text-primary text-text-on-accent border-text-primary' },
]

const SORT_OPTIONS: { field: EntrySortField; dir: SortDirection; key: string }[] = [
  { field: 'TEXT', dir: 'ASC', key: 'filter.sort_az' },
  { field: 'CREATED_AT', dir: 'DESC', key: 'filter.sort_newest' },
  { field: 'UPDATED_AT', dir: 'DESC', key: 'filter.sort_updated' },
]

export function FilterPanel({
  open,
  onClose,
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

  const chipClass = 'px-3 py-1.5 rounded-full text-sm border transition-all duration-150'
  const inactiveClass = 'border-border-subtle text-text-secondary hover:border-border-default'

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
          <div className="py-4 space-y-3 border-b border-border-subtle mb-4">
            {/* POS + Sort in one row */}
            <div className="flex flex-wrap items-center gap-2">
              {POS_OPTIONS.slice(0, 8).map((opt) => {
                const active = selectedPOS.includes(opt.value)
                return (
                  <button
                    key={opt.value}
                    type="button"
                    onClick={() => togglePOS(opt.value)}
                    className={cn(chipClass, active ? opt.activeClass : inactiveClass)}
                  >
                    {t(`pos.${opt.value}`)}
                  </button>
                )
              })}

              <span className="text-border-default mx-1">|</span>

              {SORT_OPTIONS.map((opt) => {
                const active = sortBy === opt.field && sortDir === opt.dir
                return (
                  <button
                    key={opt.key}
                    type="button"
                    onClick={() => onSortChange(opt.field, opt.dir)}
                    className={cn(
                      chipClass,
                      active
                        ? 'bg-text-primary text-text-on-accent border-text-primary'
                        : inactiveClass,
                    )}
                  >
                    {t(opt.key)}
                  </button>
                )
              })}

              <button
                type="button"
                onClick={onClose}
                className="ml-auto text-text-tertiary hover:text-text-primary transition-colors p-1"
                aria-label={t('filter.close')}
              >
                <X size={18} />
              </button>
            </div>

            {/* Topics row — only if topics exist */}
            {topics.length > 0 && (
              <div className="flex flex-wrap items-center gap-2">
                <span className="text-xs text-text-tertiary uppercase tracking-wide mr-1">
                  {t('filter.topics')}:
                </span>
                {topics.map((topic) => {
                  const active = selectedTopicIds.includes(topic.id)
                  return (
                    <button
                      key={topic.id}
                      type="button"
                      onClick={() => toggleTopic(topic.id)}
                      className={cn(
                        chipClass,
                        active
                          ? 'bg-text-primary text-text-on-accent border-text-primary'
                          : inactiveClass,
                      )}
                    >
                      {topic.name}
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}
