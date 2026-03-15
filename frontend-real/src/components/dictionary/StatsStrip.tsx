import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'
import type { DictionaryEntry, Topic } from '@/types/dictionary'

interface StatsStripProps {
  totalCount: number
  topics: Topic[]
  entries: DictionaryEntry[]
  loading: boolean
}

function StatsStripSkeleton() {
  return (
    <div className="flex items-center gap-6 py-2 mb-4">
      <Skeleton className="h-5 w-20" />
      <Skeleton className="h-5 w-20" />
    </div>
  )
}

export function StatsStrip({ totalCount, topics, loading }: StatsStripProps) {
  const { t } = useTranslation('dictionary')

  if (loading) return <StatsStripSkeleton />

  return (
    <div className="flex items-center gap-6 py-2 mb-4">
      <span className="text-sm text-text-secondary">
        <span className="font-medium text-text-primary tabular-nums">{totalCount}</span>
        {' '}{t('stats.words')}
      </span>
      <span className="text-sm text-text-secondary">
        <span className="font-medium text-text-primary tabular-nums">{topics.length}</span>
        {' '}{t('stats.topics')}
      </span>
    </div>
  )
}
