import { cn } from '@/lib/utils'

const sourceConfig = {
  book: {
    label: 'Book',
    bg: 'bg-source-book-light',
    text: 'text-source-book',
    font: '',
  },
  screen: {
    label: 'Screen',
    bg: 'bg-source-screen-light',
    text: 'text-source-screen',
    font: 'tracking-wide uppercase',
  },
  music: {
    label: 'Music',
    bg: 'bg-source-music-light',
    text: 'text-source-music',
    font: 'italic',
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
