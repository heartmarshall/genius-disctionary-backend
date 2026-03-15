import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Volume2, StickyNote } from 'lucide-react'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { getPosColors } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordOverviewProps {
  entry: DictionaryEntry
  className?: string
  hideTitle?: boolean
}

export function WordOverview({ entry, className, hideTitle }: WordOverviewProps) {
  const { t } = useTranslation('dictionary')

  const playAudio = async (url: string) => {
    try {
      await new Audio(url).play()
    } catch {
      toast.error(t('error.audioFailed'))
    }
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      {!hideTitle && (
        <div className="space-y-2">
          <h2 className="font-orelega text-4xl text-foreground">{entry.text}</h2>
          <PronunciationList pronunciations={entry.pronunciations} onPlay={playAudio} />
        </div>
      )}

      {/* Pronunciations when title is hidden */}
      {hideTitle && entry.pronunciations.length > 0 && (
        <PronunciationList pronunciations={entry.pronunciations} onPlay={playAudio} />
      )}

      {/* Topics */}
      {entry.topics.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          {entry.topics.map((topic) => (
            <Badge
              key={topic.id}
              variant="outline"
              className="font-normal bg-surface-secondary border-border-subtle text-text-secondary"
            >
              {topic.name}
            </Badge>
          ))}
        </div>
      )}

      {/* Senses */}
      {entry.senses.length > 0 && (
        <div className="space-y-5">
          {entry.senses.map((sense, index) => {
            const posColors = getPosColors(sense.partOfSpeech)

            return (
              <div key={sense.id} className="space-y-3">
                {/* Sense header */}
                <div className="flex items-center gap-2">
                  <span className={cn(
                    'flex items-center justify-center h-6 w-6 rounded-full text-xs font-medium',
                    posColors.bg,
                    posColors.text,
                  )}>
                    {index + 1}
                  </span>
                  {sense.partOfSpeech && (
                    <Badge
                      className={cn(
                        'border text-xs font-medium',
                        posColors.bg,
                        posColors.text,
                        posColors.border,
                      )}
                    >
                      {t(`pos.${sense.partOfSpeech}`)}
                    </Badge>
                  )}
                </div>

                {/* Definition */}
                {sense.definition && (
                  <p className="text-foreground leading-relaxed">{sense.definition}</p>
                )}

                {/* Translations */}
                {sense.translations.length > 0 && (
                  <p className="text-text-secondary font-medium">
                    {sense.translations.map((tr) => tr.text).join(', ')}
                  </p>
                )}

                {/* Examples */}
                {sense.examples.length > 0 && (
                  <div className="space-y-2 pl-4">
                    {sense.examples.map((ex) => (
                      <div key={ex.id}>
                        <blockquote className={cn(
                          'font-serif italic text-foreground border-l-2 pl-3',
                          posColors.dot ? `border-l-current ${posColors.text}` : 'border-l-border-default',
                        )}>
                          <span className="text-foreground">{ex.sentence}</span>
                        </blockquote>
                        {ex.translation && (
                          <p className="text-text-secondary text-sm pl-3 mt-0.5">
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
        <div className="grid grid-cols-2 gap-2">
          {[...entry.catalogImages, ...entry.userImages].map((img) => (
            <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
          ))}
        </div>
      )}

      {/* Notes */}
      {entry.notes && (
        <div className="rounded-md bg-goldenrod-light border border-goldenrod/20 p-4 space-y-1.5">
          <div className="flex items-center gap-2">
            <StickyNote className="h-4 w-4 text-goldenrod-fg" />
            <p className="text-sm font-medium text-goldenrod-fg">{t('edit.notes')}</p>
          </div>
          <p className="text-text-primary text-sm">{entry.notes}</p>
        </div>
      )}
    </div>
  )
}

function PronunciationList({
  pronunciations,
  onPlay,
}: {
  pronunciations: DictionaryEntry['pronunciations']
  onPlay: (url: string) => void
}) {
  if (pronunciations.length === 0) return null

  return (
    <div className="flex items-center gap-3 flex-wrap">
      {pronunciations.map((pron) => (
        <div
          key={pron.id}
          className="flex items-center gap-1.5 bg-cornflower-light rounded-full px-3 py-1"
        >
          <span className="font-serif text-cornflower-fg">
            /{pron.transcription}/
          </span>
          {pron.region && (
            <span className="text-xs text-cornflower-fg/70 uppercase">
              {pron.region}
            </span>
          )}
          {pron.audioUrl && (
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 text-cornflower hover:text-cornflower-hover hover:bg-cornflower/10"
              onClick={() => onPlay(pron.audioUrl!)}
            >
              <Volume2 className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
      ))}
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
        className="rounded-md object-cover aspect-square w-full"
        onError={() => setError(true)}
      />
      {caption && (
        <p className="text-xs text-text-secondary mt-1">{caption}</p>
      )}
    </div>
  )
}
