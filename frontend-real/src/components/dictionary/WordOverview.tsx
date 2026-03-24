import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Volume2 } from 'lucide-react'
import type { DictionaryEntry, Sense } from '@/types/dictionary'

// ─── Sense block ─────────────────────────────────────────────────────────────

function SenseBlock({ sense, index, total }: { sense: Sense; index: number; total: number }) {
  const { t } = useTranslation('dictionary')
  const posLabel = sense.partOfSpeech ? t(`pos.${sense.partOfSpeech}`) : undefined

  return (
    <div className="py-3 first:pt-0">
      {/* Sense header: number + POS */}
      <div className="flex items-center gap-2 mb-1.5">
        {total > 1 && (
          <span className="w-5 h-5 rounded-full bg-surface-secondary flex items-center justify-center text-[11px] font-semibold text-text-secondary tabular-nums">
            {index + 1}
          </span>
        )}
        {posLabel && (
          <span className="text-[10px] font-semibold uppercase tracking-wider px-1.5 py-px rounded border border-border-default text-text-tertiary">
            {posLabel}
          </span>
        )}
      </div>

      {/* Definition */}
      {sense.definition && (
        <p className="text-sm text-text-primary leading-relaxed">
          {sense.definition}
        </p>
      )}

      {/* Translations */}
      {sense.translations.length > 0 && (
        <p className="text-sm text-cornflower font-medium mt-1">
          {sense.translations.map(tr => tr.text).join(', ')}
        </p>
      )}

      {/* Examples */}
      {sense.examples.length > 0 && (
        <div className="mt-2.5 space-y-2">
          {sense.examples.map((ex) => (
            <div key={ex.id} className="pl-3 border-l-2 border-border-subtle">
              <p className="text-sm text-text-primary italic leading-relaxed">
                &ldquo;{ex.sentence}&rdquo;
              </p>
              {ex.translation && (
                <p className="text-sm text-text-tertiary mt-0.5">
                  {ex.translation}
                </p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── Main component ──────────────────────────────────────────────────────────

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

  const allImages = [...entry.catalogImages, ...entry.userImages]

  return (
    <div className={`flex flex-col ${className ?? ''}`}>
      {/* Title — full-page only */}
      {!hideTitle && !hideHeader && (
        <div className="mb-4">
          <h2 className="text-2xl font-display text-text-primary">{entry.text}</h2>
        </div>
      )}

      {/* Pronunciation */}
      {entry.pronunciations.length > 0 && (
        <div className="flex items-center gap-2 flex-wrap">
          {entry.pronunciations.map((pron, pi) => (
            <span key={pron.id} className="inline-flex items-center gap-1.5">
              {pi > 0 && <span className="text-text-disabled">&middot;</span>}
              <span className="text-base font-serif text-text-secondary">
                /{pron.transcription.replace(/^\/+|\/+$/g, '')}/
              </span>
              {pron.region && (
                <span className="text-[10px] uppercase tracking-wider text-text-disabled font-medium">
                  {pron.region}
                </span>
              )}
              {pron.audioUrl && (
                <button
                  type="button"
                  className="p-0.5 rounded hover:bg-surface-secondary text-cornflower hover:text-cornflower-hover transition-colors duration-150"
                  onClick={() => playAudio(pron.audioUrl!)}
                  aria-label={`Play ${pron.region ?? ''} pronunciation`}
                >
                  <Volume2 size={14} />
                </button>
              )}
            </span>
          ))}
        </div>
      )}

      {/* Topic chips */}
      {entry.topics.length > 0 && (
        <div className="flex items-center gap-1.5 flex-wrap mt-2.5">
          {entry.topics.map(tp => (
            <span
              key={tp.id}
              className="inline-flex items-center rounded px-2 py-0.5 text-[11px] font-medium bg-cornflower-light text-cornflower-fg border border-cornflower/20"
            >
              {tp.name}
            </span>
          ))}
        </div>
      )}

      {/* Senses */}
      {entry.senses.length > 0 && (
        <div className="mt-4 divide-y divide-border-subtle">
          {entry.senses.map((sense, si) => (
            <SenseBlock
              key={sense.id}
              sense={sense}
              index={si}
              total={entry.senses.length}
            />
          ))}
        </div>
      )}

      {/* Images */}
      {allImages.length > 0 && (
        <div className="mt-4">
          <p className="text-xs font-medium text-text-tertiary uppercase tracking-wider mb-2">
            Images
          </p>
          <div className="grid grid-cols-4 md:grid-cols-6 gap-2">
            {allImages.map(img => (
              <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
            ))}
          </div>
        </div>
      )}

      {/* Notes */}
      {entry.notes && (
        <div className="mt-3 p-2.5 rounded-md bg-cornflower-light border border-cornflower/10">
          <p className="text-sm text-cornflower-fg">{entry.notes}</p>
        </div>
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
        className="object-cover aspect-square w-full rounded-md"
        onError={() => setError(true)}
      />
      {caption && (
        <p className="text-[10px] mt-0.5 truncate text-text-tertiary">{caption}</p>
      )}
    </div>
  )
}
