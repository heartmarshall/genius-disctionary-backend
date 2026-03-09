import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Volume2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordOverviewProps {
  entry: DictionaryEntry
  className?: string
}

export function WordOverview({ entry, className }: WordOverviewProps) {
  const { t } = useTranslation('dictionary')

  const playAudio = (url: string) => {
    new Audio(url).play()
  }

  return (
    <div className={cn('space-y-6', className)}>
      {/* Header */}
      <div className="space-y-2">
        <h2 className="font-orelega text-4xl text-foreground">{entry.text}</h2>
        {entry.pronunciations.length > 0 && (
          <div className="flex items-center gap-3 flex-wrap">
            {entry.pronunciations.map((pron) => (
              <div key={pron.id} className="flex items-center gap-1.5">
                <span className="font-serif text-muted-foreground">
                  /{pron.transcription}/
                </span>
                {pron.region && (
                  <Badge variant="outline" className="text-xs">
                    {pron.region}
                  </Badge>
                )}
                {pron.audioUrl && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => playAudio(pron.audioUrl!)}
                  >
                    <Volume2 className="h-4 w-4" />
                  </Button>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Topics */}
      {entry.topics.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          {entry.topics.map((topic) => (
            <Badge key={topic.id} variant="outline">
              {topic.name}
            </Badge>
          ))}
        </div>
      )}

      {/* Senses */}
      {entry.senses.length > 0 && (
        <div className="space-y-4">
          {entry.senses.map((sense, index) => (
            <div key={sense.id} className="space-y-2">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-muted-foreground">
                  {index + 1}.
                </span>
                {sense.partOfSpeech && (
                  <Badge variant="secondary">
                    {t(`pos.${sense.partOfSpeech}`)}
                  </Badge>
                )}
              </div>
              {sense.definition && (
                <p className="text-foreground">{sense.definition}</p>
              )}
              {sense.translations.length > 0 && (
                <p className="text-muted-foreground">
                  {sense.translations.map((tr) => tr.text).join(', ')}
                </p>
              )}
              {sense.examples.length > 0 && (
                <div className="space-y-2 pl-4">
                  {sense.examples.map((ex) => (
                    <div key={ex.id}>
                      <blockquote className="font-serif italic text-foreground border-l-2 border-border pl-3">
                        {ex.sentence}
                      </blockquote>
                      {ex.translation && (
                        <p className="text-muted-foreground text-sm pl-3 mt-0.5">
                          {ex.translation}
                        </p>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Images */}
      {entry.images.length > 0 && (
        <div className="grid grid-cols-2 gap-2">
          {entry.images.map((img) => (
            <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
          ))}
        </div>
      )}

      {/* Notes */}
      {entry.notes && (
        <>
          <Separator />
          <div className="space-y-1">
            <p className="text-sm font-medium">{t('edit.notes')}</p>
            <p className="text-muted-foreground">{entry.notes}</p>
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
        className="rounded-md object-cover aspect-square w-full"
        onError={() => setError(true)}
      />
      {caption && (
        <p className="text-xs text-muted-foreground mt-1">{caption}</p>
      )}
    </div>
  )
}
