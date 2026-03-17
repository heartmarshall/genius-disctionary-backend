import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Volume2, StickyNote } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { getPosColors, getPosAccentBorder } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordOverviewProps {
  entry: DictionaryEntry
  className?: string
  hideTitle?: boolean
  /** Hides title, pronunciations, topics, and first divider (used when header is rendered externally) */
  hideHeader?: boolean
}

export function WordOverview({ entry, className, hideTitle, hideHeader }: WordOverviewProps) {
  const { t } = useTranslation('dictionary')

  const playAudio = async (url: string) => {
    try {
      await new Audio(url).play()
    } catch {
      toast.error(t('error.audioFailed'))
    }
  }

  return (
    <div className={cn('space-y-5', className)}>
      {/* Word title */}
      {!hideTitle && !hideHeader && (
        <h2 className="font-orelega text-4xl text-text-primary leading-tight">{entry.text}</h2>
      )}

      {/* Pronunciations */}
      {!hideHeader && entry.pronunciations.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          {entry.pronunciations.map((pron) => (
            <div
              key={pron.id}
              className="inline-flex items-center gap-1.5 rounded-full bg-surface-secondary px-3 py-1"
            >
              <span className="font-serif text-sm text-text-secondary">
                /{pron.transcription}/
              </span>
              {pron.region && (
                <span className="text-[10px] font-medium uppercase tracking-wider text-text-tertiary">
                  {pron.region}
                </span>
              )}
              {pron.audioUrl && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5 text-text-tertiary hover:text-text-primary"
                  onClick={() => playAudio(pron.audioUrl!)}
                >
                  <Volume2 className="h-3 w-3" />
                </Button>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Topics */}
      {!hideHeader && entry.topics.length > 0 && (
        <div className="flex gap-1.5 flex-wrap">
          {entry.topics.map((topic) => (
            <span
              key={topic.id}
              className="text-[10px] uppercase tracking-wider text-text-tertiary border border-border-subtle rounded-full px-2.5 py-0.5"
            >
              {topic.name}
            </span>
          ))}
        </div>
      )}

      {/* Divider */}
      {!hideHeader && <div className="h-px bg-border-subtle" />}

      {/* Senses */}
      {entry.senses.length > 0 && (
        <div className="space-y-4">
          {entry.senses.map((sense, index) => {
            const sensePos = sense.partOfSpeech
            const senseColors = getPosColors(sensePos)
            const accentBorder = getPosAccentBorder(sensePos)

            return (
              <div
                key={sense.id}
                className={cn('border-l-2 pl-4 space-y-2', accentBorder)}
              >
                {/* POS + number */}
                <div className="flex items-center gap-2">
                  <span className="text-[11px] font-medium text-text-disabled tabular-nums">
                    {index + 1}
                  </span>
                  {sensePos && (
                    <span className={cn('text-[11px] uppercase tracking-wider font-medium', senseColors.text)}>
                      {t(`pos.${sensePos}`)}
                    </span>
                  )}
                </div>

                {/* Definition */}
                {sense.definition && (
                  <p className="text-text-primary leading-relaxed">{sense.definition}</p>
                )}

                {/* Translations */}
                {sense.translations.length > 0 && (
                  <p className="text-sm text-text-secondary">
                    {sense.translations.map((tr) => tr.text).join(', ')}
                  </p>
                )}

                {/* Examples */}
                {sense.examples.length > 0 && (
                  <div className="space-y-2 pt-1">
                    {sense.examples.map((ex) => (
                      <div key={ex.id} className="rounded-lg bg-surface-secondary px-3 py-2.5">
                        <p className="font-serif italic text-sm text-text-primary leading-relaxed">
                          {ex.sentence}
                        </p>
                        {ex.translation && (
                          <p className="text-xs text-text-tertiary mt-1">
                            {ex.translation}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Images */}
      {(entry.catalogImages.length > 0 || entry.userImages.length > 0) && (
        <>
          <div className="h-px bg-border-subtle" />
          <div className="grid grid-cols-2 gap-2">
            {[...entry.catalogImages, ...entry.userImages].map((img) => (
              <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
            ))}
          </div>
        </>
      )}

      {/* Notes */}
      {entry.notes && (
        <>
          <div className="h-px bg-border-subtle" />
          <div className="rounded-lg bg-goldenrod-light px-4 py-3 space-y-1">
            <div className="flex items-center gap-1.5">
              <StickyNote className="h-3.5 w-3.5 text-goldenrod" />
              <span className="text-xs font-medium text-goldenrod-fg">{t('edit.notes')}</span>
            </div>
            <p className="text-sm text-text-primary leading-relaxed">{entry.notes}</p>
          </div>
        </>
      )}
    </div>
  )
}

function ImageWithFallback({ src, caption }: { src: string; caption?: string }) {
  const [error, setError] = useState(false)

  if (error) return null

  return (
    <div>
      <img
        src={src}
        alt={caption ?? ''}
        loading="lazy"
        className="rounded-lg object-cover aspect-square w-full"
        onError={() => setError(true)}
      />
      {caption && (
        <p className="text-xs text-text-tertiary mt-1">{caption}</p>
      )}
    </div>
  )
}
