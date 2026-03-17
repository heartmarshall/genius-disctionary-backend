import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Volume2, StickyNote } from 'lucide-react'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { getPosColors } from '@/lib/pos-colors'
import { cn } from '@/lib/utils'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordOverviewProps {
  entry: DictionaryEntry
  className?: string
  hideTitle?: boolean
  hideHeader?: boolean
  /** POS already visible in parent header — skip it in single-sense cards to reduce redundancy */
  primaryPosShownInHeader?: string
}

export function WordOverview({ entry, className, hideTitle, hideHeader, primaryPosShownInHeader }: WordOverviewProps) {
  const { t } = useTranslation('dictionary')

  const playAudio = async (url: string) => {
    try {
      await new Audio(url).play()
    } catch {
      toast.error(t('error.audioFailed'))
    }
  }

  return (
    <div className={cn('space-y-8', className)}>
      {!hideTitle && !hideHeader && (
        <h2 className="text-4xl font-medium tracking-[-0.03em] text-text-primary leading-tight">{entry.text}</h2>
      )}

      {!hideHeader && entry.pronunciations.length > 0 && (
        <div className="flex items-center gap-3 flex-wrap">
          {entry.pronunciations.map((pron) => (
            <div key={pron.id} className="inline-flex items-center gap-2">
              <span className="text-sm text-text-secondary">
                /{pron.transcription.replace(/^\/+|\/+$/g, '')}/
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

      {!hideHeader && entry.topics.length > 0 && (
        <div className="flex gap-2 flex-wrap">
          {entry.topics.map((topic) => (
            <span key={topic.id} className="text-[11px] uppercase tracking-widest text-text-tertiary">
              {topic.name}
            </span>
          ))}
        </div>
      )}

      {!hideHeader && !hideTitle && <div className="h-px bg-border-subtle/60" />}

      {entry.senses.length > 0 && (
        <div className="space-y-6">
          {entry.senses.map((sense, index) => {
            const sensePos = sense.partOfSpeech
            const senseColors = getPosColors(sensePos)

            // Skip redundant POS label when it matches the one already shown in parent header
            const skipPosLabel = primaryPosShownInHeader && sensePos === primaryPosShownInHeader && entry.senses.length === 1

            return (
              <div
                key={sense.id}
                className={cn(
                  'space-y-3',
                  index > 0 && 'pt-6 border-t border-border-subtle/40',
                )}
              >
                {/* Sense header: number + POS */}
                <div className="flex items-center gap-2.5">
                  {entry.senses.length > 1 && (
                    <span className="text-xs text-text-disabled tabular-nums font-medium">
                      {index + 1}
                    </span>
                  )}
                  {sensePos && !skipPosLabel && (
                    <span className={cn('text-[11px] uppercase tracking-widest font-medium', senseColors.text)}>
                      {t(`pos.${sensePos}`)}
                    </span>
                  )}
                </div>

                {/* Definition */}
                {sense.definition && (
                  <p className="text-base text-text-primary leading-relaxed">
                    {sense.definition}
                  </p>
                )}

                {/* Translation */}
                {sense.translations.length > 0 && (
                  <p className="text-sm text-text-secondary">
                    {sense.translations.map((tr) => tr.text).join(', ')}
                  </p>
                )}

                {/* Examples */}
                {sense.examples.length > 0 && (
                  <div className="space-y-2 pl-4 border-l-2 border-border-subtle/60">
                    {sense.examples.map((ex) => (
                      <div key={ex.id}>
                        <p className="italic text-sm text-text-primary leading-snug">{ex.sentence}</p>
                        {ex.translation && (
                          <p className="text-xs text-text-tertiary mt-0.5">{ex.translation}</p>
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

      {(entry.catalogImages.length > 0 || entry.userImages.length > 0) && (
        <>
          <div className="h-px bg-border-subtle/60" />
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {[...entry.catalogImages, ...entry.userImages].map((img) => (
              <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
            ))}
          </div>
        </>
      )}

      {entry.notes && (
        <>
          <div className="h-px bg-border-subtle/60" />
          <div className="max-w-2xl space-y-2">
            <div className="flex items-center gap-2">
              <StickyNote className="h-3.5 w-3.5 text-goldenrod" />
              <span className="text-xs uppercase tracking-widest font-medium text-goldenrod-fg">{t('edit.notes')}</span>
            </div>
            <p className="text-base text-text-primary leading-relaxed">{entry.notes}</p>
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
      {caption && <p className="text-xs text-text-tertiary mt-2">{caption}</p>}
    </div>
  )
}
