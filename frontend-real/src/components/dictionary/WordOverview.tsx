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
        <div className="w-full overflow-x-auto">
          <table className="w-full border-collapse text-sm">
            <thead>
              <tr className="border-b border-border-subtle/40">
                <th className="text-left text-[10px] uppercase tracking-widest text-text-tertiary font-medium py-2 pr-3 w-6">#</th>
                <th className="text-left text-[10px] uppercase tracking-widest text-text-tertiary font-medium py-2 px-3 w-20">{t('table.pos', 'POS')}</th>
                <th className="text-left text-[10px] uppercase tracking-widest text-text-tertiary font-medium py-2 px-3">{t('table.definition', 'Definition')}</th>
                <th className="text-left text-[10px] uppercase tracking-widest text-text-tertiary font-medium py-2 px-3 w-[160px]">{t('table.translation', 'Translation')}</th>
                <th className="text-left text-[10px] uppercase tracking-widest text-text-tertiary font-medium py-2 pl-3">{t('table.example', 'Example')}</th>
              </tr>
            </thead>
            <tbody>
              {entry.senses.map((sense, index) => {
                const sensePos = sense.partOfSpeech
                const senseColors = getPosColors(sensePos)

                return (
                  <tr
                    key={sense.id}
                    className="border-b border-border-subtle/20 last:border-b-0 align-top"
                  >
                    <td className="py-3 pr-3">
                      <span className="text-[11px] text-text-disabled tabular-nums">{index + 1}</span>
                    </td>

                    <td className="py-3 px-3">
                      {sensePos && (
                        <span className={cn('text-[11px] uppercase tracking-widest font-medium whitespace-nowrap', senseColors.text)}>
                          {t(`pos.${sensePos}`)}
                        </span>
                      )}
                    </td>

                    <td className="py-3 px-3">
                      {sense.definition && (
                        <p className="text-sm text-text-primary leading-snug">{sense.definition}</p>
                      )}
                    </td>

                    <td className="py-3 px-3">
                      {sense.translations.length > 0 && (
                        <p className="text-sm text-text-secondary leading-snug">
                          {sense.translations.map((tr) => tr.text).join(', ')}
                        </p>
                      )}
                    </td>

                    <td className="py-3 pl-3">
                      {sense.examples.length > 0 && (
                        <div className="space-y-2">
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
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
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
