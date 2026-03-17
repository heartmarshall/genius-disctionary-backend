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
  primaryPosShownInHeader?: string
}

function ColLabel({ children }: { children: React.ReactNode }) {
  return (
    <th className="text-[10px] uppercase tracking-[0.1em] font-medium text-text-tertiary text-left align-bottom pb-3 select-none">
      {children}
    </th>
  )
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

  // Determine which columns the table needs across all senses
  const hasAnyDefinition = entry.senses.some((s) => s.definition)
  const hasAnyTranslation = entry.senses.some((s) => s.translations.length > 0)
  const hasAnyExamples = entry.senses.some((s) => s.examples.length > 0)

  return (
    <div className={cn('space-y-4', className)}>
      {!hideTitle && !hideHeader && (
        <h2 className="text-2xl font-medium tracking-[-0.03em] text-text-primary leading-tight">{entry.text}</h2>
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

      {/* Single table for all senses */}
      {entry.senses.length > 0 && (() => {
        const multipleSenses = entry.senses.length > 1
        return (
        <table className="w-full border-collapse table-fixed">
          <thead>
            <tr>
              {multipleSenses && <ColLabel>&nbsp;</ColLabel>}
              {hasAnyDefinition && <ColLabel>{t('sense.definition', 'Definition')}</ColLabel>}
              {hasAnyTranslation && <ColLabel>{t('sense.translation', 'Translation')}</ColLabel>}
              {hasAnyExamples && <ColLabel>{t('sense.examples', 'Examples')}</ColLabel>}
            </tr>
          </thead>
          <tbody>
            {entry.senses.map((sense, index) => {
              const sensePos = sense.partOfSpeech
              const senseColors = getPosColors(sensePos)

              return (
                <tr
                  key={sense.id}
                  className={cn(
                    'align-top',
                    index > 0 && 'border-t border-border-subtle/40',
                    index % 2 === 1 && 'bg-surface-secondary/30',
                  )}
                >
                  {/* Number + POS column — only when multiple senses */}
                  {multipleSenses && (
                    <td className="pr-3 pt-3 pb-3 w-0 whitespace-nowrap">
                      <div className="flex items-center gap-1.5">
                        <span className="text-xs text-text-disabled tabular-nums font-medium">
                          {index + 1}
                        </span>
                        {sensePos && (
                          <span className={cn('text-xs font-medium whitespace-nowrap', senseColors.text)}>
                            {t(`pos.${sensePos}`)}
                          </span>
                        )}
                      </div>
                    </td>
                  )}

                  {/* Definition */}
                  {hasAnyDefinition && (
                    <td className="pr-4 md:pr-6 pt-3 pb-3">
                      {sense.definition && (
                        <p className="text-sm text-text-primary leading-relaxed">
                          {sense.definition}
                        </p>
                      )}
                    </td>
                  )}

                  {/* Translation */}
                  {hasAnyTranslation && (
                    <td className="pr-4 md:pr-6 pt-3 pb-3">
                      {sense.translations.length > 0 && (
                        <p className="text-sm text-text-primary leading-relaxed">
                          {sense.translations.map((tr) => tr.text).join(', ')}
                        </p>
                      )}
                    </td>
                  )}

                  {/* Examples */}
                  {hasAnyExamples && (
                    <td className="pt-3 pb-3">
                      {sense.examples.length > 0 && (
                        <div className="space-y-2 pl-3 border-l-2 border-border-subtle/60">
                          {sense.examples.map((ex) => (
                            <div key={ex.id}>
                              <p className="italic text-xs text-text-primary leading-snug">{ex.sentence}</p>
                              {ex.translation && (
                                <p className="text-[11px] text-text-tertiary mt-0.5">{ex.translation}</p>
                              )}
                            </div>
                          ))}
                        </div>
                      )}
                    </td>
                  )}
                </tr>
              )
            })}
          </tbody>
        </table>
        )
      })()}

      {/* Images */}
      {(entry.catalogImages.length > 0 || entry.userImages.length > 0) && (
        <>
          <div className="h-px bg-border-subtle/60" />
          <div>
            <p className="text-[10px] uppercase tracking-[0.1em] font-medium text-text-tertiary pb-1.5 select-none">
              {t('sense.images', 'Images')}
            </p>
            <div className="grid grid-cols-3 md:grid-cols-4 gap-3">
              {[...entry.catalogImages, ...entry.userImages].map((img) => (
                <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
              ))}
            </div>
          </div>
        </>
      )}

      {/* Notes */}
      {entry.notes && (
        <>
          <div className="h-px bg-border-subtle/60" />
          <div className="max-w-2xl">
            <p className="text-[10px] uppercase tracking-[0.1em] font-medium text-text-tertiary pb-1.5 select-none inline-flex items-center gap-1.5">
              <StickyNote className="h-2.5 w-2.5 text-goldenrod" />
              {t('edit.notes')}
            </p>
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
      {caption && <p className="text-xs text-text-tertiary mt-1">{caption}</p>}
    </div>
  )
}
