import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'
import { cn } from '@/lib/utils'
import { getPosColors } from '@/lib/pos-colors'
import type { DictionaryEntry } from '@/types/dictionary'
import type { DisplayOptions } from './DictionaryToolbar'

interface WordListProps {
  entries: DictionaryEntry[]
  loading: boolean
  selectedWordId: string | null
  onWordClick: (id: string) => void
  displayOptions: DisplayOptions
  sortedAlphabetically?: boolean
  renderDetail?: (entry: DictionaryEntry, index: number) => React.ReactNode
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
  sortedAlphabetically,
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

  let lastLetter = ''

  return (
    <div className="flex flex-col">
      {entries.map((entry, index) => {
        const isSelected = selectedWordId === entry.id

        // Build second-line metadata
        const translation = displayOptions.translation
          ? entry.senses[0]?.translations[0]?.text
          : undefined
        const ipa = displayOptions.transcription
          ? entry.pronunciations[0]?.transcription
          : undefined
        const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined
        const primaryPos = entry.senses[0]?.partOfSpeech
        const showPos = displayOptions.partOfSpeech && primaryPos
        const definition = displayOptions.definition
          ? entry.senses[0]?.definition
          : undefined
        const topicNames = displayOptions.topic
          ? entry.topics.map((topic) => topic.name)
          : []

        const textParts: string[] = []
        if (translation) textParts.push(translation)
        if (cleanIpa) textParts.push(cleanIpa)
        if (definition) textParts.push(definition)
        if (topicNames.length > 0) textParts.push(topicNames.join(', '))

        const hasSecondLine = showPos || textParts.length > 0

        // Alphabetical section headers (Chunking)
        let letterHeader: React.ReactNode = null
        if (sortedAlphabetically) {
          const firstChar = entry.text[0]?.toUpperCase() ?? ''
          if (firstChar !== lastLetter) {
            lastLetter = firstChar
            letterHeader = (
              <div className="sticky top-0 z-[5] pt-6 pb-2 text-xs font-medium text-text-tertiary uppercase tracking-widest bg-bg-page">
                {firstChar}
              </div>
            )
          }
        }

        const posColors = showPos ? getPosColors(primaryPos) : null

        return (
          <div key={entry.id}>
            {letterHeader}
            <div data-word-id={entry.id} style={{ scrollMarginTop: '4rem' }}>
              {isSelected ? (
                <div className="border-b border-border-subtle/60">
                  {renderDetail?.(entry, index)}
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => onWordClick(entry.id)}
                  className={cn(
                    'w-full flex items-baseline gap-4 md:gap-8 py-4 md:py-5 text-left cursor-pointer',
                    'transition-all duration-200 ease-out',
                    'hover:bg-surface-secondary/60',
                    'border-b border-border-subtle/60',
                  )}
                >
                  <div className="flex flex-col min-w-0 flex-1">
                    <span className="text-[1.75rem] md:text-[2.25rem] font-medium leading-tight tracking-[-0.02em] text-text-primary">
                      {entry.text}
                    </span>
                    {hasAnyDisplayOption && hasSecondLine && (
                      <span className="flex items-center gap-1.5 mt-1 text-sm text-text-tertiary line-clamp-1">
                        {showPos && posColors && (
                          <>
                            <span className={cn('text-xs font-medium uppercase tracking-wide', posColors.text)}>
                              {t(`pos.${primaryPos}`)}
                            </span>
                            {textParts.length > 0 && (
                              <span className="text-text-disabled">·</span>
                            )}
                          </>
                        )}
                        {textParts.join(' · ')}
                      </span>
                    )}
                  </div>
                  <span className="text-xs text-text-tertiary tabular-nums shrink-0 self-center">
                    {index + 1}
                  </span>
                </button>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
