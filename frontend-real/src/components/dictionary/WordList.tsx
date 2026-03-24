import { useTranslation } from 'react-i18next'

import type { DictionaryEntry } from '@/types/dictionary'
import type { DisplayOptions } from './DictionaryToolbar'
import { getPosColors } from '@/lib/pos-colors'

interface WordListProps {
  entries: DictionaryEntry[]
  loading: boolean
  selectedWordId: string | null
  onWordClick: (id: string) => void
  displayOptions: DisplayOptions
  sortedAlphabetically?: boolean
  renderDetail?: (entry: DictionaryEntry, index: number) => React.ReactNode
}

const POS_SHORT: Record<string, string> = {
  NOUN: 'n', VERB: 'v', ADJECTIVE: 'adj', ADVERB: 'adv',
  PRONOUN: 'pron', PREPOSITION: 'prep', CONJUNCTION: 'conj',
  INTERJECTION: 'interj', PHRASE: 'phr', IDIOM: 'idiom', OTHER: '?',
}

function WordListSkeleton() {
  return (
    <div className="flex flex-col gap-0.5">
      {Array.from({ length: 12 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 py-2.5 px-3">
          <div
            className="h-4 rounded bg-surface-disabled animate-pulse"
            style={{ width: 60 + Math.random() * 100 }}
          />
          <div
            className="h-3 rounded bg-surface-disabled animate-pulse opacity-50"
            style={{ width: 30 + Math.random() * 60 }}
          />
          <div className="flex-1" />
          <div
            className="h-3 rounded bg-surface-disabled animate-pulse opacity-30"
            style={{ width: 40 + Math.random() * 80 }}
          />
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

  let lastLetter = ''

  return (
    <div className="flex flex-col">
      {entries.map((entry, index) => {
        const isSelected = selectedWordId === entry.id

        const translation = displayOptions.translation
          ? entry.senses[0]?.translations[0]?.text
          : undefined
        const ipa = displayOptions.transcription
          ? entry.pronunciations[0]?.transcription
          : undefined
        const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined
        const primaryPos = entry.senses[0]?.partOfSpeech
        const showPos = displayOptions.partOfSpeech && primaryPos

        // Alphabetical section headers
        let letterHeader: React.ReactNode = null
        if (sortedAlphabetically) {
          const firstChar = entry.text[0]?.toUpperCase() ?? ''
          if (firstChar !== lastLetter) {
            lastLetter = firstChar
            letterHeader = (
              <div className="sticky top-0 z-[5] flex items-center gap-3 pt-5 pb-1.5 bg-bg-page">
                <span className="text-sm font-bold tracking-wider text-cornflower">
                  {firstChar}
                </span>
                <div className="flex-1 h-px bg-border-default" />
              </div>
            )
          }
        }

        return (
          <div key={entry.id}>
            {letterHeader}
            <div data-word-id={entry.id} style={{ scrollMarginTop: '4rem' }}>
              {isSelected ? (
                <div className="my-0.5">
                  {renderDetail?.(entry, index)}
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => onWordClick(entry.id)}
                  className="dict-word-row w-full grid items-center gap-3 py-2.5 px-3.5 text-left cursor-pointer"
                  style={{ gridTemplateColumns: '32px 1fr' }}
                >
                  {/* Index */}
                  <span className="text-[12px] font-mono font-medium text-text-disabled select-none">
                    {index + 1}
                  </span>

                  {/* Word + meta */}
                  <div className="flex items-baseline gap-3.5 min-w-0">
                    <span className="text-[16.5px] font-semibold text-text-primary shrink-0">
                      {entry.text}
                    </span>

                    {showPos && (
                      <span className={`text-[11.5px] font-medium uppercase tracking-wide shrink-0 ${getPosColors(primaryPos).text}`}>
                        {POS_SHORT[primaryPos] ?? primaryPos.toLowerCase()}
                      </span>
                    )}

                    {cleanIpa && (
                      <span className="font-mono text-[12.5px] text-text-disabled shrink-0">
                        {cleanIpa}
                      </span>
                    )}

                    {translation && (
                      <span className="text-[14.5px] text-text-secondary truncate min-w-0">
                        {translation}
                      </span>
                    )}

                    {!translation && <span className="flex-1" />}
                  </div>

                </button>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
