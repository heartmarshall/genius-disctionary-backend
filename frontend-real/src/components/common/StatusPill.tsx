import { cn } from '@/lib/utils'

const statusConfig = {
  new: { label: 'New', dot: 'bg-poppy', text: 'text-poppy', bg: 'bg-poppy-light' },
  learning: { label: 'Learning', dot: 'bg-goldenrod', text: 'text-goldenrod', bg: 'bg-goldenrod-light' },
  review: { label: 'Review', dot: 'bg-cornflower', text: 'text-cornflower', bg: 'bg-cornflower-light' },
  mastered: { label: 'Mastered', dot: 'bg-thyme', text: 'text-thyme', bg: 'bg-thyme-light' },
} as const

interface StatusPillProps {
  status: keyof typeof statusConfig
  className?: string
}

export function StatusPill({ status, className }: StatusPillProps) {
  const config = statusConfig[status]

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium',
        config.bg,
        config.text,
        className
      )}
    >
      <span className={cn('h-1.5 w-1.5 rounded-full', config.dot)} />
      {config.label}
    </span>
  )
}
