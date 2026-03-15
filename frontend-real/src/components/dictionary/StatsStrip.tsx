import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'
import type { DictionaryEntry, Topic } from '@/types/dictionary'

interface StatsStripProps {
  totalCount: number
  topics: Topic[]
  entries: DictionaryEntry[]
  loading: boolean
}

function getPosDistribution(entries: DictionaryEntry[]) {
  const counts: Record<string, number> = {}
  for (const entry of entries) {
    const pos = entry.senses[0]?.partOfSpeech
    if (pos) counts[pos] = (counts[pos] ?? 0) + 1
  }
  return counts
}

const POS_BAR_COLORS: Record<string, string> = {
  NOUN: 'bg-cornflower',
  PRONOUN: 'bg-cornflower',
  VERB: 'bg-poppy',
  INTERJECTION: 'bg-poppy',
  ADJECTIVE: 'bg-goldenrod',
  PREPOSITION: 'bg-goldenrod',
  ADVERB: 'bg-thyme',
  CONJUNCTION: 'bg-thyme',
}

function StatsStripSkeleton() {
  return (
    <div className="flex items-center gap-8 py-4 border-y border-border-subtle">
      <Skeleton className="h-6 w-16" />
      <Skeleton className="h-6 w-16" />
      <Skeleton className="h-2 w-40 rounded-full" />
    </div>
  )
}

export function StatsStrip({ totalCount, topics, entries, loading }: StatsStripProps) {
  const { t } = useTranslation('dictionary')

  if (loading) return <StatsStripSkeleton />

  const distribution = getPosDistribution(entries)
  const total = Object.values(distribution).reduce((a, b) => a + b, 0)

  return (
    <div className="flex items-center gap-8 py-4 border-y border-border-subtle">
      {/* Word count */}
      <div className="flex items-baseline gap-1.5">
        <span className="text-xl font-bold text-text-primary tracking-tight tabular-nums">
          {totalCount}
        </span>
        <span className="text-[10px] uppercase tracking-[1.2px] text-text-tertiary">
          {t('stats.words')}
        </span>
      </div>

      <div className="w-px h-5 bg-border-subtle" />

      {/* Topics count */}
      <div className="flex items-baseline gap-1.5">
        <span className="text-xl font-bold text-text-primary tracking-tight tabular-nums">
          {topics.length}
        </span>
        <span className="text-[10px] uppercase tracking-[1.2px] text-text-tertiary">
          {t('stats.topics')}
        </span>
      </div>

      <div className="w-px h-5 bg-border-subtle" />

      {/* POS distribution bar */}
      {total > 0 && (
        <div className="flex gap-0.5 h-1.5 rounded-full overflow-hidden flex-1 max-w-40">
          {Object.entries(distribution).map(([pos, count]) => (
            <div
              key={pos}
              className={`${POS_BAR_COLORS[pos] ?? 'bg-surface-disabled'} rounded-full`}
              style={{ flex: count }}
            />
          ))}
        </div>
      )}
    </div>
  )
}
