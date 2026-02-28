import { cn } from '@/lib/utils'

type CardStatus = 'new' | 'learning' | 'review' | 'mastered'

interface StatusPillProps {
  status: CardStatus
  className?: string
}

const statusConfig: Record<CardStatus, { label: string; bg: string; fg: string; dot: string }> = {
  new: {
    label: 'New',
    bg: 'bg-status-new',
    fg: 'text-status-new-fg',
    dot: 'bg-poppy',
  },
  learning: {
    label: 'Learning',
    bg: 'bg-status-learning',
    fg: 'text-status-learning-fg',
    dot: 'bg-goldenrod',
  },
  review: {
    label: 'Review',
    bg: 'bg-status-review',
    fg: 'text-status-review-fg',
    dot: 'bg-cornflower',
  },
  mastered: {
    label: 'Mastered',
    bg: 'bg-status-mastered',
    fg: 'text-status-mastered-fg',
    dot: 'bg-thyme',
  },
}

function StatusPill({ status, className }: StatusPillProps) {
  const config = statusConfig[status]

  return (
    <span
      className={cn(
        'inline-flex items-center gap-[5px] rounded-full px-3 py-1 text-[11px] font-medium',
        config.bg,
        config.fg,
        className,
      )}
    >
      <span className={cn('size-[7px] rounded-full', config.dot)} />
      {config.label}
    </span>
  )
}

export { StatusPill }
export type { CardStatus }
