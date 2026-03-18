import { useTranslation } from 'react-i18next'
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

const TERM_POS_COLORS: Record<string, string> = {
  NOUN: 'var(--term-blue)',
  VERB: 'var(--term-red)',
  ADJECTIVE: 'var(--term-yellow)',
  ADVERB: 'var(--term-cyan)',
  PRONOUN: 'var(--term-blue)',
  PREPOSITION: 'var(--term-orange)',
  CONJUNCTION: 'var(--term-cyan)',
  INTERJECTION: 'var(--term-purple)',
}

function WordListSkeleton() {
  return (
    <div className="flex flex-col">
      {Array.from({ length: 12 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 py-1.5">
          <span className="text-xs tabular-nums w-6 text-right" style={{ color: 'var(--term-text-dim)' }}>
            {i + 1}
          </span>
          <div
            className="h-3 animate-pulse"
            style={{ width: 60 + Math.random() * 100, background: 'var(--term-border)', opacity: 0.6 }}
          />
          <div
            className="h-3 animate-pulse"
            style={{ width: 30 + Math.random() * 20, background: 'var(--term-text-dim)', opacity: 0.2 }}
          />
        </div>
      ))}
    </div>
  )
}

/** Format short POS label */
function posShort(pos: string): string {
  const map: Record<string, string> = {
    NOUN: 'n', VERB: 'v', ADJECTIVE: 'adj', ADVERB: 'adv',
    PRONOUN: 'pron', PREPOSITION: 'prep', CONJUNCTION: 'conj',
    INTERJECTION: 'interj', PHRASE: 'phr', IDIOM: 'idiom', OTHER: '?',
  }
  return map[pos] ?? pos.toLowerCase()
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
    <div className="flex flex-col font-mono">
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
        const definition = displayOptions.definition
          ? entry.senses[0]?.definition
          : undefined
        const topicNames = displayOptions.topic
          ? entry.topics.map((topic) => topic.name)
          : []

        // Build the right-side info fragments
        const infoParts: string[] = []
        if (translation) infoParts.push(translation)
        if (definition && !translation) infoParts.push(definition)

        const sensesCount = entry.senses.length

        // Alphabetical section headers
        let letterHeader: React.ReactNode = null
        if (sortedAlphabetically) {
          const firstChar = entry.text[0]?.toUpperCase() ?? ''
          if (firstChar !== lastLetter) {
            lastLetter = firstChar
            letterHeader = (
              <div className="sticky top-0 z-[5] flex items-center gap-2 pt-5 pb-1" style={{ background: 'var(--term-bg)' }}>
                <span className="text-sm font-bold uppercase" style={{ color: 'var(--term-green)' }}>
                  {firstChar}
                </span>
                <div className="flex-1" style={{ borderBottom: '1px dotted var(--term-border)' }} />
              </div>
            )
          }
        }

        const posColor = showPos ? (TERM_POS_COLORS[primaryPos] ?? 'var(--term-text-muted)') : undefined

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
                  className="term-word-row w-full flex items-baseline gap-2 py-1.5 text-left cursor-pointer group"
                >
                  {/* Index */}
                  <span className="text-xs tabular-nums w-6 text-right shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>
                    {index + 1}
                  </span>

                  {/* Word */}
                  <span className="text-sm font-bold shrink-0" style={{ color: 'var(--term-text-bright)' }}>
                    {entry.text}
                  </span>

                  {/* POS short */}
                  {showPos && (
                    <span className="text-xs shrink-0" style={{ color: posColor }}>
                      {posShort(primaryPos)}
                    </span>
                  )}

                  {/* Senses count if > 1 */}
                  {sensesCount > 1 && (
                    <span className="text-xs shrink-0" style={{ color: 'var(--term-text-dim)' }}>
                      {sensesCount}s
                    </span>
                  )}

                  {/* IPA */}
                  {cleanIpa && (
                    <span className="text-xs shrink-0" style={{ color: 'var(--term-text-dim)' }}>
                      {cleanIpa}
                    </span>
                  )}

                  {/* Translation / definition */}
                  {infoParts.length > 0 && (
                    <span className="text-xs truncate min-w-0 flex-1" style={{ color: 'var(--term-text-muted)' }}>
                      — {infoParts.join(' · ')}
                    </span>
                  )}

                  {/* Spacer when no info */}
                  {infoParts.length === 0 && <span className="flex-1" />}

                  {/* Topics */}
                  {topicNames.length > 0 && (
                    <span className="text-xs shrink-0 hidden md:inline" style={{ color: 'var(--term-text-dim)' }}>
                      [{topicNames.join(',')}]
                    </span>
                  )}
                </button>
              )}
            </div>
          </div>
        )
      })}
    </div>
  )
}
