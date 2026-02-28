import { StatusPill, type CardStatus } from '@/components/ui/status-pill'
import { cn } from '@/lib/utils'
import type { StatusCounts } from '@/graphql/queries/dashboard'

interface StatusDistributionProps {
  statusCounts: StatusCounts
  className?: string
}

const statuses: Array<{ key: keyof StatusCounts; pill: CardStatus }> = [
  { key: 'new', pill: 'new' },
  { key: 'learning', pill: 'learning' },
  { key: 'review', pill: 'review' },
  { key: 'relearning', pill: 'learning' },
]

function StatusDistribution({ statusCounts, className }: StatusDistributionProps) {
  return (
    <div
      className={cn(
        'rounded-md border border-border bg-bg-card p-md shadow-1',
        className,
      )}
    >
      <div className="flex items-center justify-between mb-sm">
        <span className="text-[13px] font-medium text-text-secondary">
          Card Distribution
        </span>
        <span className="text-[13px] text-text-tertiary">
          {statusCounts.total} total
        </span>
      </div>

      {statusCounts.total > 0 && (
        <div className="mb-md flex h-2 overflow-hidden rounded-full bg-border-light">
          {statuses.map(({ key, pill }) => {
            const count = statusCounts[key]
            if (count === 0) return null
            const pct = (count / statusCounts.total) * 100
            return (
              <div
                key={key}
                className={cn(
                  pill === 'new' && 'bg-poppy',
                  pill === 'learning' && 'bg-goldenrod',
                  pill === 'review' && 'bg-cornflower',
                  pill === 'mastered' && 'bg-thyme',
                )}
                style={{ width: `${pct}%` }}
              />
            )
          })}
        </div>
      )}

      <div className="flex flex-wrap gap-sm">
        {statuses.map(({ key, pill }) => (
          <div key={key} className="flex items-center gap-xs">
            <StatusPill status={pill} />
            <span className="text-[13px] font-medium text-text-primary">
              {statusCounts[key]}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

export { StatusDistribution }
