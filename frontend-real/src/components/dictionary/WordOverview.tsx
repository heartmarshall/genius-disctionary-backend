import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import type { DictionaryEntry, Sense } from '@/types/dictionary'

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

function K({ children }: { children: React.ReactNode }) {
  return <span className="text-[10px] inline-block w-[52px] shrink-0 text-right mr-2" style={{ color: 'var(--term-text-dim)' }}>{children}</span>
}

function V({ children, color }: { children: React.ReactNode; color?: string }) {
  return <span className="text-[12px]" style={{ color: color ?? 'var(--term-text-bright)' }}>{children}</span>
}

// ─── Sense subtree ───────────────────────────────────────────────────────────

function SenseTree({ sense, index, isLast }: { sense: Sense; index: number; isLast: boolean }) {
  const { t } = useTranslation('dictionary')
  const posColor = sense.partOfSpeech
    ? (TERM_POS_COLORS[sense.partOfSpeech] ?? 'var(--term-text-muted)')
    : undefined

  const prefix = isLast ? '└' : '├'
  const lineChar = isLast ? ' ' : '│'

  return (
    <div className="ml-3">
      {/* Sense header */}
      <div className="flex items-baseline leading-[1.6]">
        <span className="text-[10px] w-3 shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>{prefix}</span>
        <span className="text-[11px] font-bold" style={{ color: 'var(--term-green)' }}>[{index}]</span>
        {sense.partOfSpeech && (
          <span className="text-[10px] ml-2" style={{ color: posColor }}>{t(`pos.${sense.partOfSpeech}`)}</span>
        )}
      </div>

      {/* Fields */}
      <div className="ml-3">
        {sense.definition && (
          <div className="flex items-baseline leading-[1.6]">
            <span className="text-[10px] w-3 shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>{lineChar}</span>
            <K>def</K>
            <V>{sense.definition}</V>
          </div>
        )}
        {sense.translations.length > 0 && (
          <div className="flex items-baseline leading-[1.6]">
            <span className="text-[10px] w-3 shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>{lineChar}</span>
            <K>trans</K>
            <V color="var(--term-text)">{sense.translations.map(tr => tr.text).join(', ')}</V>
          </div>
        )}
        {sense.examples.map((ex, ei) => (
          <div key={ex.id}>
            <div className="flex items-baseline leading-[1.6]">
              <span className="text-[10px] w-3 shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>{lineChar}</span>
              <K>ex[{ei}]</K>
              <span className="text-[11px] italic" style={{ color: 'var(--term-text-bright)' }}>
                "{ex.sentence}"
              </span>
            </div>
            {ex.translation && (
              <div className="flex items-baseline leading-[1.6]">
                <span className="text-[10px] w-3 shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>{lineChar}</span>
                <K></K>
                <span className="text-[10px]" style={{ color: 'var(--term-text-muted)' }}>
                  {ex.translation}
                </span>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

// ─── Main component ──────────────────────────────────────────────────────────

interface WordOverviewProps {
  entry: DictionaryEntry
  className?: string
  hideTitle?: boolean
  hideHeader?: boolean
  primaryPosShownInHeader?: string
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
    <div className={`font-mono ${className ?? ''}`}>
      {!hideTitle && !hideHeader && (
        <div className="leading-[1.6] mb-0.5">
          <span className="text-[13px] font-bold" style={{ color: 'var(--term-text-bright)' }}>{entry.text}</span>
        </div>
      )}

      {/* IPA */}
      {!hideHeader && entry.pronunciations.length > 0 && (
        <div className="flex items-baseline leading-[1.6]">
          <K>ipa</K>
          <span className="inline-flex items-center gap-2 flex-wrap">
            {entry.pronunciations.map((pron) => (
              <span key={pron.id} className="inline-flex items-baseline gap-1">
                <V color="var(--term-text)">
                  /{pron.transcription.replace(/^\/+|\/+$/g, '')}/
                </V>
                {pron.region && (
                  <span className="text-[9px] uppercase tracking-wider" style={{ color: 'var(--term-text-dim)' }}>
                    {pron.region}
                  </span>
                )}
                {pron.audioUrl && (
                  <button
                    type="button"
                    className="text-[10px] hover:underline"
                    style={{ color: 'var(--term-cyan)' }}
                    onClick={() => playAudio(pron.audioUrl!)}
                  >
                    [play]
                  </button>
                )}
              </span>
            ))}
          </span>
        </div>
      )}

      {/* Topics */}
      {!hideHeader && entry.topics.length > 0 && (
        <div className="flex items-baseline leading-[1.6]">
          <K>topics</K>
          <V color="var(--term-text-muted)">
            {entry.topics.map(t => t.name).join(', ')}
          </V>
        </div>
      )}

      {/* Senses */}
      {entry.senses.length > 0 && (
        <div className="mt-1.5">
          <div className="flex items-baseline leading-[1.6]">
            <span className="text-[10px]" style={{ color: 'var(--term-text-dim)' }}>
              senses: {entry.senses.length}
            </span>
          </div>
          {entry.senses.map((sense, si) => (
            <SenseTree key={sense.id} sense={sense} index={si} isLast={si === entry.senses.length - 1} />
          ))}
        </div>
      )}

      {/* Images */}
      {allImages.length > 0 && (
        <div className="mt-2">
          <div className="flex items-baseline leading-[1.6]">
            <span className="text-[10px]" style={{ color: 'var(--term-text-dim)' }}>
              images: {allImages.length}
            </span>
          </div>
          <div className="ml-3 py-1">
            <div className="grid grid-cols-4 md:grid-cols-6 gap-1.5">
              {allImages.map(img => (
                <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Note */}
      {entry.notes && (
        <div className="flex items-baseline leading-[1.6] mt-1.5">
          <K>note</K>
          <V color="var(--term-yellow)">{entry.notes}</V>
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
        className="object-cover aspect-square w-full"
        style={{ filter: 'grayscale(0.3) contrast(1.1)' }}
        onError={() => setError(true)}
      />
      {caption && (
        <p className="text-[9px] mt-0.5 truncate" style={{ color: 'var(--term-text-dim)' }}>
          {caption}
        </p>
      )}
    </div>
  )
}
