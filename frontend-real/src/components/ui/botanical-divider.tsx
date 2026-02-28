import { cn } from '@/lib/utils'

interface BotanicalDividerProps {
  className?: string
}

function BotanicalDivider({ className }: BotanicalDividerProps) {
  return (
    <div
      className={cn('h-px my-xl opacity-50', className)}
      style={{
        background:
          'linear-gradient(90deg, transparent, var(--sage) 20%, var(--dried-rose) 50%, var(--sage) 80%, transparent)',
      }}
    />
  )
}

export { BotanicalDivider }
