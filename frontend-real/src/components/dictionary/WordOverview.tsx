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

// ─── Tree drawing primitives ─────────────────────────────────────────────────

/** Tree connector character — ├ or └ */
function TreeConn({ last }: { last: boolean }) {
  return (
    <span className="text-xs shrink-0 select-none w-4 inline-block" style={{ color: 'var(--term-border)' }}>
      {last ? '└─' : '├─'}
    </span>
  )
}

/** Vertical continuation line for nested content under a non-last branch */
function TreeIndent({ last, children }: { last: boolean; children: React.ReactNode }) {
  return (
    <div
      className="ml-[7px] pl-[13px]"
      style={last ? undefined : { borderLeft: '1px solid var(--term-border)' }}
    >
      {children}
    </div>
  )
}

// ─── Sense subtree ───────────────────────────────────────────────────────────

function SenseTree({ sense, index, isLast, totalSenses }: { sense: Sense; index: number; isLast: boolean; totalSenses: number }) {
  const { t } = useTranslation('dictionary')
  const posColor = sense.partOfSpeech
    ? (TERM_POS_COLORS[sense.partOfSpeech] ?? 'var(--term-text-muted)')
    : undefined

  // Collect inner branches: def, trans, examples...
  const innerBranches: React.ReactNode[] = []

  if (sense.definition) {
    innerBranches.push(
      <div key="def" className="flex items-baseline leading-[1.7]">
        <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>def</span>
        <span className="text-sm" style={{ color: 'var(--term-text-bright)' }}>{sense.definition}</span>
      </div>
    )
  }

  if (sense.translations.length > 0) {
    innerBranches.push(
      <div key="trans" className="flex items-baseline leading-[1.7]">
        <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>trans</span>
        <span className="text-sm" style={{ color: 'var(--term-text)' }}>{sense.translations.map(tr => tr.text).join(', ')}</span>
      </div>
    )
  }

  sense.examples.forEach((ex, ei) => {
    innerBranches.push(
      <div key={ex.id}>
        <div className="flex items-baseline leading-[1.7]">
          <span className="text-sm italic" style={{ color: 'var(--term-text-bright)' }}>
            &quot;{ex.sentence}&quot;
          </span>
        </div>
        {ex.translation && (
          <div className="text-xs mt-0.5 ml-0.5" style={{ color: 'var(--term-text-muted)' }}>
            {ex.translation}
          </div>
        )}
      </div>
    )
  })

  return (
    <div>
      {/* Sense connector + label */}
      <div className="flex items-baseline gap-1">
        <TreeConn last={isLast} />
        {totalSenses > 1 && (
          <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>[{index}]</span>
        )}
        {sense.partOfSpeech && (
          <span className="text-xs font-bold" style={{ color: posColor }}>{t(`pos.${sense.partOfSpeech}`)}</span>
        )}
      </div>

      {/* Inner branches under tree indent */}
      <TreeIndent last={isLast}>
        {innerBranches.map((branch, bi) => {
          const isLastInner = bi === innerBranches.length - 1
          return (
            <div key={bi} className={isLastInner ? 'mb-0.5' : 'mb-1.5'}>
              <div className="flex items-baseline gap-1">
                <TreeConn last={isLastInner} />
                <div className="min-w-0 flex-1">{branch}</div>
              </div>
            </div>
          )
        })}
      </TreeIndent>
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

  // Collect top-level branches to correctly assign ├─ / └─
  type Branch = { key: string; node: React.ReactNode; nested?: React.ReactNode }
  const branches: Branch[] = []

  if (!hideHeader && entry.pronunciations.length > 0) {
    branches.push({
      key: 'ipa',
      node: (
        <div className="flex items-baseline leading-[1.7]">
          <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>ipa</span>
          <span className="inline-flex items-center gap-2 flex-wrap">
            {entry.pronunciations.map((pron) => (
              <span key={pron.id} className="inline-flex items-baseline gap-1">
                <span className="text-sm" style={{ color: 'var(--term-text)' }}>
                  /{pron.transcription.replace(/^\/+|\/+$/g, '')}/
                </span>
                {pron.region && (
                  <span className="text-[10px] uppercase tracking-wider" style={{ color: 'var(--term-text-dim)' }}>
                    {pron.region}
                  </span>
                )}
                {pron.audioUrl && (
                  <button
                    type="button"
                    className="text-xs hover:underline"
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
      ),
    })
  }

  if (!hideHeader && entry.topics.length > 0) {
    branches.push({
      key: 'topic',
      node: (
        <div className="flex items-baseline leading-[1.7]">
          <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>topic</span>
          <span className="text-sm" style={{ color: 'var(--term-text-muted)' }}>
            {entry.topics.map(t => t.name).join(', ')}
          </span>
        </div>
      ),
    })
  }

  // Senses are branches with their own sub-trees
  if (entry.senses.length > 0) {
    // If senses is the only section or the last, each sense handles its own └─
    entry.senses.forEach((sense, si) => {
      const hasMore = allImages.length > 0 || entry.notes
      const isLastSense = si === entry.senses.length - 1
      branches.push({
        key: `sense-${sense.id}`,
        node: null, // rendered directly as SenseTree
        nested: (
          <SenseTree
            sense={sense}
            index={si}
            isLast={isLastSense && !hasMore}
            totalSenses={entry.senses.length}
          />
        ),
      })
    })
  }

  if (allImages.length > 0) {
    const isLast = !entry.notes
    branches.push({
      key: 'images',
      node: (
        <div className="flex items-baseline leading-[1.7]">
          <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>img</span>
          <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>{allImages.length} attached</span>
        </div>
      ),
      nested: (
        <>
          <div className="flex items-baseline gap-1">
            <TreeConn last={isLast} />
            <div className="flex items-baseline leading-[1.7]">
              <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>img</span>
              <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>{allImages.length} attached</span>
            </div>
          </div>
          <TreeIndent last={isLast}>
            <div className="py-1">
              <div className="grid grid-cols-4 md:grid-cols-6 gap-1.5">
                {allImages.map(img => (
                  <ImageWithFallback key={img.id} src={img.url} caption={img.caption} />
                ))}
              </div>
            </div>
          </TreeIndent>
        </>
      ),
    })
  }

  if (entry.notes) {
    branches.push({
      key: 'note',
      node: (
        <div className="flex items-baseline leading-[1.7]">
          <span className="text-xs shrink-0 mr-2 select-none" style={{ color: 'var(--term-cyan)' }}>note</span>
          <span className="text-sm" style={{ color: 'var(--term-yellow)' }}>{entry.notes}</span>
        </div>
      ),
    })
  }

  return (
    <div className={`font-mono ${className ?? ''}`}>
      {!hideTitle && !hideHeader && (
        <div className="leading-[1.6] mb-1">
          <span className="text-base font-bold" style={{ color: 'var(--term-text-bright)' }}>{entry.text}</span>
        </div>
      )}

      {branches.map((branch, bi) => {
        const isLast = bi === branches.length - 1

        // SenseTree renders its own connector
        if (branch.nested) {
          return <div key={branch.key}>{branch.nested}</div>
        }

        // Simple branch with ├─ / └─ connector
        return (
          <div key={branch.key} className="flex items-baseline gap-1">
            <TreeConn last={isLast} />
            <div className="min-w-0 flex-1">{branch.node}</div>
          </div>
        )
      })}
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
        <p className="text-[10px] mt-0.5 truncate" style={{ color: 'var(--term-text-dim)' }}>
          {caption}
        </p>
      )}
    </div>
  )
}
