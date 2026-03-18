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

const C = {
  bright: 'var(--term-text-bright)',
  text: 'var(--term-text)',
  muted: 'var(--term-text-muted)',
  dim: 'var(--term-text-dim)',
  cyan: 'var(--term-cyan)',
  yellow: 'var(--term-yellow)',
  border: 'var(--term-border)',
} as const

// ─── Tree line: prefix char + content ────────────────────────────────────────

function TL({ c, children }: { c: string; children: React.ReactNode }) {
  return (
    <div className="flex leading-[1.7]">
      <span className="shrink-0 select-none whitespace-pre" style={{ color: C.dim }}>{c}</span>
      <div className="flex-1 min-w-0">{children}</div>
    </div>
  )
}

// ─── Sense content (within tree) ─────────────────────────────────────────────

function SenseContent({ sense, tc }: { sense: Sense; tc: string }) {
  const { t } = useTranslation('dictionary')
  const posColor = sense.partOfSpeech
    ? (TERM_POS_COLORS[sense.partOfSpeech] ?? C.muted)
    : undefined

  return (
    <>
      {sense.partOfSpeech && (
        <TL c={tc}>
          <span style={{ color: C.muted }}>-pos:</span>{' '}
          <span style={{ color: posColor }}>{t(`pos.${sense.partOfSpeech}`)}</span>
        </TL>
      )}

      {sense.definition && (
        <TL c={tc}>
          <span style={{ color: C.muted }}>-def:</span>{' '}
          <span style={{ color: C.bright }}>{sense.definition}</span>
        </TL>
      )}

      {sense.translations.length > 0 && (
        <TL c={tc}>
          <span style={{ color: C.muted }}>-trans:</span>{' '}
          <span style={{ color: C.text }}>
            {sense.translations.map(tr => tr.text).join(', ')}
          </span>
        </TL>
      )}

      {sense.examples.length > 0 && (
        <>
          {sense.examples.map((ex, ei) => (
            <div key={ex.id}>
              <TL c={tc}>
                <span style={{ color: C.muted }}>-ex[{ei}]:</span>{' '}
                <span className="italic" style={{ color: C.bright }}>
                  &ldquo;{ex.sentence}&rdquo;
                </span>
              </TL>
              {ex.translation && (
                <TL c={tc}>
                  <span style={{ color: C.muted, marginLeft: `${7 + String(ei).length}ch` }}>
                    // {ex.translation}
                  </span>
                </TL>
              )}
            </div>
          ))}
        </>
      )}
    </>
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
    <div className={`font-mono text-sm flex flex-col ${className ?? ''}`}>
      {/* Title — full-page only */}
      {!hideTitle && !hideHeader && (
        <div className="mb-3">
          <span className="font-bold" style={{ color: C.bright }}>{entry.text}</span>
          {entry.senses[0]?.partOfSpeech && (
            <span className="ml-2 uppercase tracking-wider" style={{
              color: TERM_POS_COLORS[entry.senses[0].partOfSpeech] ?? C.muted,
            }}>
              {t(`pos.${entry.senses[0].partOfSpeech}`)}
            </span>
          )}
        </div>
      )}

      {/* -ipa: */}
      {entry.pronunciations.length > 0 && (
        <div className="leading-[1.7]">
          <span style={{ color: C.muted }}>-ipa:</span>{' '}
          {entry.pronunciations.map((pron, pi) => (
            <span key={pron.id}>
              {pi > 0 && ', '}
              <span style={{ color: C.text }}>
                /{pron.transcription.replace(/^\/+|\/+$/g, '')}/
              </span>
              {pron.region && (
                <span className="ml-1 text-[10px] uppercase tracking-wider" style={{ color: C.dim }}>
                  {pron.region}
                </span>
              )}
              {pron.audioUrl && (
                <button
                  type="button"
                  className="ml-1 hover:underline"
                  style={{ color: C.cyan }}
                  onClick={() => playAudio(pron.audioUrl!)}
                >
                  ▸
                </button>
              )}
            </span>
          ))}
        </div>
      )}

      {/* -topics: */}
      {entry.topics.length > 0 && (
        <div className="leading-[1.7]">
          <span style={{ color: C.muted }}>-topics:</span>{' '}
          <span style={{ color: C.text }}>{entry.topics.map(tp => tp.name).join(', ')}</span>
        </div>
      )}

      {/* Senses — always tree */}
      {entry.senses.length > 0 && (
        <div className="mt-1">
          {/* -senses (N) */}
          <div className="leading-[1.7]">
            <span style={{ color: C.muted }}>-senses</span>
          </div>

          {/* Tree */}
          <div className="ml-2">
            {entry.senses.map((sense, si) => {
              const isLast = si === entry.senses.length - 1

              return (
                <div key={sense.id}>
                  {/* ├─[n] or └─[n] */}
                  <div className="flex leading-[1.7]">
                    <span className="shrink-0 select-none whitespace-pre" style={{ color: C.dim }}>
                      {isLast ? '└─' : '├─'}
                    </span>
                    <span className="font-bold" style={{ color: C.dim }}>[{si}]</span>
                  </div>

                  {/* Content with tree continuation: "│  " or "   " */}
                  <SenseContent sense={sense} tc={isLast ? '   ' : '│  '} />
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* -img: */}
      {allImages.length > 0 && (
        <div className="mt-2">
          <div className="leading-[1.7]">
            <span style={{ color: C.muted }}>-img:</span>{' '}
            <span style={{ color: C.text }}>{allImages.length}</span>
          </div>
          <div className="mt-1 ml-2">
            <div className="grid grid-cols-4 md:grid-cols-6 gap-1.5">
              {allImages.map(img => (
                <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
              ))}
            </div>
          </div>
        </div>
      )}

      {/* -note: */}
      {entry.notes && (
        <div className="leading-[1.7] mt-1">
          <span style={{ color: C.muted }}>-note:</span>{' '}
          <span style={{ color: C.yellow }}>{entry.notes}</span>
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
        <p className="text-[10px] mt-0.5 truncate" style={{ color: C.dim }}>{caption}</p>
      )}
    </div>
  )
}
