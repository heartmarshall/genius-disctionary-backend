import { Flame } from 'lucide-react'
import { cn } from '@/lib/utils'

interface StreakDisplayProps {
  streak: number
  className?: string
}

function StreakDisplay({ streak, className }: StreakDisplayProps) {
  const hasStreak = streak > 0

  return (
    <div
      className={cn(
        'rounded-md border border-border bg-bg-card p-md shadow-1',
        className,
      )}
    >
      <div className="flex items-start justify-between">
        <span className="text-[13px] font-medium text-text-secondary">Streak</span>
        <Flame
          className={cn(
            'size-5',
            hasStreak ? 'text-goldenrod' : 'text-text-tertiary',
          )}
        />
      </div>
      <div className="mt-sm flex items-baseline gap-xs">
        <span
          className={cn(
            'text-[28px] font-heading leading-tight',
            hasStreak ? 'text-goldenrod-fg' : 'text-text-primary',
          )}
        >
          {streak}
        </span>
        <span className="text-[13px] text-text-tertiary">
          {streak === 1 ? 'day' : 'days'}
        </span>
      </div>
    </div>
  )
}

export { StreakDisplay }
