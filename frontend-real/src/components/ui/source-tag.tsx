import { cn } from '@/lib/utils'

type SourceType = 'book' | 'screen' | 'lyrics'

interface SourceTagProps {
  source: SourceType
  label?: string
  className?: string
}

const sourceConfig: Record<SourceType, { defaultLabel: string; bg: string; fg: string }> = {
  book: {
    defaultLabel: 'Book',
    bg: 'bg-source-book-light',
    fg: 'text-[oklch(0.45_0.05_290)]',
  },
  screen: {
    defaultLabel: 'TV/Movie',
    bg: 'bg-source-screen-light',
    fg: 'text-[oklch(0.43_0.05_200)]',
  },
  lyrics: {
    defaultLabel: 'Lyrics',
    bg: 'bg-source-lyrics-light',
    fg: 'text-[oklch(0.43_0.05_320)]',
  },
}

function SourceTag({ source, label, className }: SourceTagProps) {
  const config = sourceConfig[source]

  return (
    <span
      className={cn(
        'inline-block rounded-full px-[9px] py-[3px] text-[10px] font-medium',
        config.bg,
        config.fg,
        className,
      )}
    >
      {label ?? config.defaultLabel}
    </span>
  )
}

export { SourceTag }
export type { SourceType }
