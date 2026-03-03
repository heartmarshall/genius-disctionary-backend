import { cn } from '@/lib/utils'

const sourceConfig = {
  book: {
    label: 'Book',
    bg: 'bg-[#f0ecf5]',
    text: 'text-[#7a6898]',
    font: 'font-serif',
  },
  screen: {
    label: 'Screen',
    bg: 'bg-[#e8eff3]',
    text: 'text-[#4e7385]',
    font: 'font-mono',
  },
  music: {
    label: 'Music',
    bg: 'bg-[#f3ecf1]',
    text: 'text-[#8a6482]',
    font: 'font-serif italic',
  },
} as const

interface SourceTagProps {
  source: keyof typeof sourceConfig
  className?: string
}

export function SourceTag({ source, className }: SourceTagProps) {
  const config = sourceConfig[source]

  return (
    <span
      className={cn(
        'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
        config.bg,
        config.text,
        config.font,
        className
      )}
    >
      {config.label}
    </span>
  )
}
