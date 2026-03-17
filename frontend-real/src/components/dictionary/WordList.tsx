import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'
import type { DisplayOptions } from './DictionaryToolbar'

interface WordListProps {
  entries: DictionaryEntry[]
  loading: boolean
  selectedWordId: string | null
  onWordClick: (id: string) => void
  displayOptions: DisplayOptions
  renderDetail?: (entry: DictionaryEntry) => React.ReactNode
}

function WordListSkeleton() {
  const widths = [280, 360, 200, 320, 240, 400, 180, 300]
  return (
    <div className="flex flex-col">
      {widths.map((w, i) => (
        <div key={i} className="flex items-center justify-between py-5 border-b border-border-subtle/60">
          <Skeleton className="h-8 rounded" style={{ width: w }} />
          <Skeleton className="h-3 w-10 rounded" />
        </div>
      ))}
    </div>
  )
}

export function WordList({
  entries,
  loading,
  selectedWordId,
  onWordClick,
  displayOptions,
  renderDetail,
}: WordListProps) {
  const { t } = useTranslation('dictionary')

  if (loading) {
    return <WordListSkeleton />
  }

  const hasAnyDisplayOption =
    displayOptions.translation ||
    displayOptions.transcription ||
    displayOptions.partOfSpeech ||
    displayOptions.definition ||
    displayOptions.topic

  return (
    <div className="flex flex-col">
      {entries.map((entry, index) => {
        const isSelected = selectedWordId === entry.id

        let secondLineParts: string[] = []
        if (hasAnyDisplayOption) {
          if (displayOptions.translation) {
            const translation = entry.senses[0]?.translations[0]?.text
            if (translation) secondLineParts.push(translation)
          }
          if (displayOptions.transcription) {
            const ipa = entry.pronunciations[0]?.transcription
            if (ipa) {
              const clean = ipa.replace(/^\/+|\/+$/g, '')
              secondLineParts.push(`/${clean}/`)
            }
          }
          if (displayOptions.partOfSpeech) {
            const primaryPos = entry.senses[0]?.partOfSpeech
            if (primaryPos) secondLineParts.push(t(`pos.${primaryPos}`))
          }
          if (displayOptions.definition) {
            const definition = entry.senses[0]?.definition
            if (definition) secondLineParts.push(definition)
          }
          if (displayOptions.topic) {
            const topicNames = entry.topics.map((topic) => topic.name)
            if (topicNames.length > 0) secondLineParts.push(topicNames.join(', '))
          }
        }

        const nextIsSelected = index < entries.length - 1 && selectedWordId === entries[index + 1].id
        const indexLabel = String(index + 1).padStart(3, '0')

        return (
          <div key={entry.id} data-word-id={entry.id} style={{ scrollMarginTop: '4rem' }}>
            {isSelected ? (
              renderDetail?.(entry)
            ) : (
              <button
                type="button"
                onClick={() => onWordClick(entry.id)}
                className={cn(
                  'w-full flex items-baseline justify-between gap-4 md:gap-8 py-4 md:py-5 text-left cursor-pointer',
                  'transition-all duration-300 ease-out',
                  'hover:opacity-60',
                  !nextIsSelected && !isSelected && 'border-b border-border-subtle/60',
                )}
              >
                <div className="flex flex-col min-w-0">
                  <span className="text-[1.75rem] md:text-[2.25rem] font-medium leading-tight tracking-[-0.02em] text-text-primary">
                    {entry.text}
                  </span>
                  {hasAnyDisplayOption && secondLineParts.length > 0 && (
                    <span className="text-sm text-text-tertiary mt-1 line-clamp-1">
                      {secondLineParts.join(' · ')}
                    </span>
                  )}
                </div>

                <span className="text-[11px] md:text-xs text-text-tertiary tabular-nums shrink-0 pt-1">
                  {indexLabel}
                </span>
              </button>
            )}
          </div>
        )
      })}
    </div>
  )
}
