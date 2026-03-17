import { useTranslation } from 'react-i18next'
import { Skeleton } from '@/components/ui/skeleton'
import { getPosColors } from '@/lib/pos-colors'
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
  const widths = [140, 200, 120, 180, 160, 220, 100, 170]
  return (
    <div className="flex flex-col">
      {widths.map((w, i) => (
        <div key={i} className="flex items-center gap-3 px-4 py-3 border-b border-border-subtle">
          <Skeleton className="w-[3px] h-8 rounded-full flex-shrink-0" />
          <Skeleton className="h-7 rounded" style={{ width: w }} />
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
  const { t } = useTranslation()

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
      {entries.map((entry) => {
        const isSelected = selectedWordId === entry.id
        const primaryPos = entry.senses[0]?.partOfSpeech
        const posColors = getPosColors(primaryPos)

        // Build second-line parts
        let secondLineParts: string[] = []
        if (hasAnyDisplayOption) {
          if (displayOptions.translation) {
            const translation = entry.senses[0]?.translations[0]?.text
            if (translation) secondLineParts.push(translation)
          }
          if (displayOptions.transcription) {
            const ipa = entry.pronunciations[0]?.transcription
            if (ipa) secondLineParts.push(`/${ipa}/`)
          }
          if (displayOptions.partOfSpeech && primaryPos) {
            secondLineParts.push(t(`pos.${primaryPos}`))
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

        return (
          <div key={entry.id}>
            <button
              type="button"
              data-word-id={entry.id}
              onClick={() => onWordClick(entry.id)}
              style={{ scrollMarginTop: '4rem' }}
              className={cn(
                'w-full flex items-stretch gap-3 px-4 py-3 text-left cursor-pointer',
                'transition-colors duration-150',
                'hover:bg-surface-secondary',
                isSelected ? 'bg-surface-secondary' : 'border-b border-border-subtle',
              )}
            >
              {/* POS color bar */}
              <div
                className={cn(
                  'w-[3px] rounded-full self-stretch flex-shrink-0 transition-opacity duration-150',
                  isSelected ? 'opacity-100' : 'opacity-60',
                )}
                style={{ backgroundColor: `var(${posColors.cssVar})` }}
              />

              {/* Word content */}
              <div className="flex flex-col min-w-0">
                <span className="font-orelega text-[24px] leading-tight text-text-primary">
                  {entry.text}
                </span>
                {hasAnyDisplayOption && secondLineParts.length > 0 && (
                  <span className="text-sm text-text-secondary mt-0.5 line-clamp-1">
                    {secondLineParts.join(' · ')}
                  </span>
                )}
              </div>
            </button>

            {/* Inline detail slot */}
            {isSelected && renderDetail?.(entry)}
          </div>
        )
      })}
    </div>
  )
}
