import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { motion } from 'framer-motion'
import { Pencil, Trash2, X, Volume2 } from 'lucide-react'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { useMutation } from '@apollo/client/react'
import { WordEditDialog } from './WordEditDialog'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useDeleteWord } from '@/hooks/useDeleteWord'
import { CREATE_CARD, GET_DICTIONARY_ENTRY } from '@/graphql/queries/dictionary'
import type { Sense, CardStats, ReviewLog } from '@/types/dictionary'

// ─── Types ───────────────────────────────────────────────────────────────────

interface WordDetailInlineProps {
  wordId: string
  index: number
  onClose: () => void
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  return iso.split('T')[0] // YYYY-MM-DD
}

const POS_LABEL: Record<string, string> = {
  NOUN: 'noun', VERB: 'verb', ADJECTIVE: 'adjective', ADVERB: 'adverb',
  PRONOUN: 'pronoun', PREPOSITION: 'preposition', CONJUNCTION: 'conjunction',
  INTERJECTION: 'interjection', PHRASE: 'phrase', IDIOM: 'idiom', OTHER: 'other',
}

// ─── Skeleton ────────────────────────────────────────────────────────────────

function ContentSkeleton() {
  return (
    <div className="space-y-3 py-2">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="flex items-center gap-2">
          <div
            className="h-4 rounded bg-surface-disabled animate-pulse"
            style={{ width: 60 + Math.random() * 180 }}
          />
        </div>
      ))}
    </div>
  )
}

// ─── Icon button ─────────────────────────────────────────────────────────────

function IconBtn({
  onClick,
  children,
  title,
  hoverClass = 'hover:bg-surface-secondary hover:text-text-secondary',
}: {
  onClick: () => void
  children: React.ReactNode
  title: string
  hoverClass?: string
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={`w-9 h-9 rounded-lg flex items-center justify-center text-text-disabled transition-all duration-100 ${hoverClass}`}
    >
      {children}
    </button>
  )
}

// ─── Sense block ─────────────────────────────────────────────────────────────

function SenseBlock({ sense, index, isFirst }: { sense: Sense; index: number; isFirst: boolean }) {
  const { t } = useTranslation('dictionary')
  const posLabel = sense.partOfSpeech ? t(`pos.${sense.partOfSpeech}`) : undefined

  return (
    <div className={`pl-3 ${isFirst ? 'border-l-[3px] border-l-cornflower' : 'border-l-[3px] border-l-border-default'}`}>
      <div className="flex items-center gap-1.5 mb-1">
        <span className="text-[11px] font-bold text-cornflower">{index + 1}</span>
        {posLabel && (
          <span className="text-[10px] font-semibold uppercase text-text-disabled">{posLabel}</span>
        )}
      </div>

      {sense.definition && (
        <p className="text-[13.5px] leading-relaxed text-text-primary mb-0.5">{sense.definition}</p>
      )}

      {sense.translations.length > 0 && (
        <p className="text-xs text-cornflower font-medium mb-2">
          {sense.translations.map(tr => tr.text).join(', ')}
        </p>
      )}

      {sense.examples.length > 0 && (
        <div className="space-y-1">
          {sense.examples.map((ex) => (
            <div key={ex.id} className="bg-surface-secondary rounded-md px-2.5 py-2">
              <p className="text-xs text-text-primary italic leading-snug">
                &ldquo;{ex.sentence}&rdquo;
              </p>
              {ex.translation && (
                <p className="text-[11.5px] text-text-tertiary mt-0.5">{ex.translation}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── Progress widget ────────────────────────────────────────────────────────

const GRADE_COLOR: Record<string, string> = {
  AGAIN: 'bg-poppy',
  HARD: 'bg-goldenrod',
  GOOD: 'bg-thyme',
  EASY: 'bg-cornflower',
}

function ProgressWidget({ stats, logs, nextReview, hasCard, onCreateCard, creating }: {
  stats?: CardStats
  logs?: ReviewLog[]
  nextReview?: string
  hasCard: boolean
  onCreateCard: () => void
  creating?: boolean
}) {
  if (!hasCard) {
    return (
      <div className="dict-cell">
        <div className="dict-section-label">
          <span className="text-xs">{'\uD83D\uDCCA'}</span> Progress
        </div>
        <button
          type="button"
          onClick={onCreateCard}
          disabled={creating}
          className="w-full py-4 rounded-lg border border-dashed border-border-default text-xs font-medium text-text-disabled hover:text-cornflower hover:border-cornflower transition-colors duration-150 disabled:opacity-50"
        >
          {creating ? 'Creating...' : '+ Create card'}
        </button>
      </div>
    )
  }

  if (!stats) return null

  const accuracyPct = Math.round(stats.accuracy * 100)
  const totalReviews = stats.totalReviews

  // Compute streak from consecutive GOOD/EASY at the tail of logs
  const streak = (() => {
    if (!logs || logs.length === 0) return 0
    let count = 0
    for (const log of logs) {
      if (log.grade === 'GOOD' || log.grade === 'EASY') count++
      else break
    }
    return count
  })()

  // Format next review date
  const nextReviewLabel = (() => {
    if (!nextReview) return null
    const due = new Date(nextReview)
    const now = new Date()
    const diffMs = due.getTime() - now.getTime()
    const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))
    if (diffDays <= 0) return 'Due now'
    if (diffDays === 1) return 'Tomorrow'
    return `In ${diffDays} days`
  })()

  return (
    <div className="dict-cell">
      <div className="dict-section-label">
        <span className="text-xs">{'\uD83D\uDCCA'}</span> Progress
      </div>

      {/* Streak + accuracy */}
      <div className="flex items-baseline justify-between">
        <div className="flex items-baseline gap-1.5">
          <span className="text-sm leading-none">{'\uD83D\uDD25'}</span>
          <span className="text-xl font-bold text-text-primary tabular-nums leading-none">{streak}</span>
          <span className="text-[11px] text-text-disabled">streak</span>
        </div>
        <div className="flex items-baseline gap-1.5">
          <span className="text-xl font-bold text-text-primary tabular-nums leading-none">{accuracyPct}%</span>
          <span className="text-[11px] text-text-disabled">accuracy</span>
        </div>
      </div>

      {/* Mini stats */}
      <div className="flex items-center justify-between mt-2.5 px-0.5">
        <span className="text-[11px] text-text-tertiary">
          {totalReviews} review{totalReviews !== 1 ? 's' : ''}
        </span>
        {nextReviewLabel && (
          <span className="text-[11px] text-text-tertiary">
            {nextReviewLabel}
          </span>
        )}
      </div>

      {/* SRS history bar */}
      {logs && logs.length > 0 && (
        <div className="flex items-end gap-[3px] mt-2 h-5">
          {logs.slice(0, 20).reverse().map((log, i) => {
            const isCorrect = log.grade === 'GOOD' || log.grade === 'EASY'
            const height = isCorrect ? '100%' : '60%'
            const opacity = 0.4 + (i / 20) * 0.6
            return (
              <div
                key={log.id}
                className={`flex-1 rounded-sm ${GRADE_COLOR[log.grade] ?? 'bg-text-disabled'}`}
                style={{ height, opacity, minWidth: 4, maxWidth: 12 }}
                title={`${log.grade} — ${formatDate(log.reviewedAt)}`}
              />
            )
          })}
        </div>
      )}
    </div>
  )
}

// ─── Main component ──────────────────────────────────────────────────────────

export function WordDetailInline({ wordId, index, onClose }: WordDetailInlineProps) {
  const { t } = useTranslation('dictionary')
  const { entry, cardStats, reviewLogs, loading, error } = useWordDetail(wordId)
  const { requestDelete, confirmDelete, cancelDelete, pendingEntry } = useDeleteWord()
  const [createCard, { loading: creatingCard }] = useMutation(CREATE_CARD, {
    variables: { entryId: wordId },
    refetchQueries: [{ query: GET_DICTIONARY_ENTRY, variables: { id: wordId } }],
  })
  const [editWordId, setEditWordId] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<'notes' | null>(null)

  const primaryPos = entry?.senses[0]?.partOfSpeech
  const posLabel = primaryPos ? POS_LABEL[primaryPos] ?? primaryPos.toLowerCase() : ''
  const ipa = entry?.pronunciations[0]?.transcription
  const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined
  const allImages = entry ? [...entry.catalogImages, ...entry.userImages] : []

  const playAudio = async (url: string) => {
    try {
      await new Audio(url).play()
    } catch {
      toast.error(t('error.audioFailed'))
    }
  }

  // Loading state
  if (loading) {
    return (
      <div className="bg-bg-card rounded-[14px] border border-border-default shadow-bento my-1.5 p-2.5">
        <div className="dict-cell"><ContentSkeleton /></div>
      </div>
    )
  }

  // Error state
  if (error) {
    return (
      <div className="bg-bg-card rounded-[14px] border border-border-default shadow-bento my-1.5 p-2.5">
        <div className="dict-cell">
          <p className="text-sm text-poppy">{t('error.loadFailed')}</p>
          <button type="button" className="mt-1.5 text-sm text-cornflower hover:underline" onClick={() => window.location.reload()}>
            {t('error.tryAgain')}
          </button>
        </div>
      </div>
    )
  }

  if (!entry) return null

  return (
    <motion.div
      initial={{ opacity: 0, y: -6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2, ease: 'easeOut' }}
      className="bg-bg-card rounded-[14px] border border-border-default shadow-bento my-1.5 overflow-hidden p-2.5"
    >
      {/* ─── HERO ─── */}
      <div className="dict-cell px-5 py-4 mb-2">
        {/* Row 1: Word + pronunciation (left) — Actions (right) */}
        <div className="flex justify-between items-start">
          <div className="flex items-baseline gap-2.5 min-w-0">
            <button
              type="button"
              onClick={onClose}
              className="text-[28px] font-bold text-text-primary font-serif tracking-tight leading-none hover:text-cornflower transition-colors duration-150 cursor-pointer"
            >
              {entry.text}
            </button>
            {cleanIpa && (
              <span className="font-mono text-xs text-text-tertiary">{cleanIpa}</span>
            )}
            {posLabel && (
              <span className="text-[11px] text-text-disabled">{posLabel}</span>
            )}
          </div>

          {/* Action buttons — edit/delete separated from close */}
          <div className="flex items-center shrink-0 ml-4">
            <div className="flex items-center gap-1">
              <IconBtn onClick={() => setEditWordId(entry.id)} title="Edit">
                <Pencil size={15} />
              </IconBtn>
              <IconBtn onClick={() => requestDelete(entry)} title="Delete" hoverClass="hover:bg-poppy-light hover:text-poppy">
                <Trash2 size={15} />
              </IconBtn>
            </div>
            <div className="w-px h-5 bg-border-default mx-3" />
            <IconBtn onClick={onClose} title="Close" hoverClass="hover:bg-surface-secondary hover:text-text-primary">
              <X size={16} />
            </IconBtn>
          </div>
        </div>

        {/* Row 2: Metadata — tags + date (visually separated from word) */}
        <div className="flex items-center gap-1.5 mt-3 flex-wrap">
          {entry.topics.map(tp => (
            <span
              key={tp.id}
              className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-cornflower-light text-cornflower-fg border border-cornflower/20"
            >
              {tp.name}
            </span>
          ))}
          <span className="text-[10px] text-text-disabled ml-1">
            added {formatDate(entry.createdAt)}
            {entry.updatedAt !== entry.createdAt && ` · updated ${formatDate(entry.updatedAt)}`}
          </span>
        </div>
      </div>

      {/* ─── BENTO GRID ─── */}
      <div
        className="grid gap-2"
        style={{
          gridTemplateColumns: '1fr 1fr 220px',
          gridTemplateAreas: `'defs defs widgets' 'notes notes widgets'`,
        }}
      >
        {/* Definitions */}
        <div style={{ gridArea: 'defs' }} className="dict-cell">
          <div className="dict-section-label">
            <span className="text-xs">{'\uD83D\uDCD6'}</span> Definitions
          </div>
          {entry.senses.length > 0 ? (
            <div className="space-y-3.5">
              {entry.senses.map((sense, i) => (
                <SenseBlock key={sense.id} sense={sense} index={i} isFirst={i === 0} />
              ))}
            </div>
          ) : (
            <p className="text-xs text-text-disabled italic">No definitions yet</p>
          )}
        </div>

        {/* Widgets column */}
        <div style={{ gridArea: 'widgets' }} className="flex flex-col gap-2">
          {/* Media widget */}
          <div className="bg-surface-secondary rounded-[10px] overflow-hidden">
            <div className="h-20 relative">
              {allImages.length > 0 ? (
                <img src={allImages[0].url} alt={entry.text} className="w-full h-full object-cover" />
              ) : (
                <div className="w-full h-full bg-gradient-to-br from-surface-secondary to-surface-footer flex items-center justify-center text-text-disabled">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="opacity-40">
                    <rect x="3" y="3" width="18" height="18" rx="2" />
                    <circle cx="8.5" cy="8.5" r="1.5" />
                    <path d="m21 15-5-5L5 21" />
                  </svg>
                </div>
              )}
            </div>
            {/* Pronunciation buttons */}
            <div className="flex border-t border-border-subtle">
              {entry.pronunciations.length > 0 ? (
                entry.pronunciations.map((pron, i) => (
                  <button
                    key={pron.id}
                    type="button"
                    onClick={() => pron.audioUrl && playAudio(pron.audioUrl)}
                    className={`flex-1 py-1.5 text-[10px] font-semibold text-text-tertiary bg-surface-secondary hover:bg-cornflower-light hover:text-cornflower transition-all duration-100 flex items-center justify-center gap-1 ${
                      i === 0 ? 'rounded-bl-[10px]' : ''
                    } ${
                      i === entry.pronunciations.length - 1 ? 'rounded-br-[10px]' : ''
                    }`}
                  >
                    {pron.audioUrl && <Volume2 size={10} />}
                    {pron.region ? pron.region.toUpperCase() : `#${i + 1}`}
                  </button>
                ))
              ) : (
                <>
                  <button type="button" className="flex-1 py-1.5 text-[10px] font-semibold text-text-tertiary bg-surface-secondary hover:bg-cornflower-light hover:text-cornflower transition-all duration-100 rounded-bl-[10px]">
                    {'\uD83C\uDDEC\uD83C\uDDE7'} UK
                  </button>
                  <button type="button" className="flex-1 py-1.5 text-[10px] font-semibold text-text-tertiary bg-surface-secondary hover:bg-cornflower-light hover:text-cornflower transition-all duration-100 rounded-br-[10px]">
                    {'\uD83C\uDDFA\uD83C\uDDF8'} US
                  </button>
                </>
              )}
            </div>
          </div>

          {/* Progress widget */}
          <ProgressWidget
            stats={cardStats}
            logs={reviewLogs}
            nextReview={entry.card?.due}
            hasCard={!!entry.card}
            onCreateCard={() => createCard()}
            creating={creatingCard}
          />

          {/* Additional images */}
          {allImages.length > 1 && (
            <div className="dict-cell">
              <div className="grid grid-cols-3 gap-1">
                {allImages.slice(1, 4).map(img => (
                  <img key={img.id} src={img.url} alt="" className="w-full aspect-square object-cover rounded" />
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Notes */}
        <div style={{ gridArea: 'notes' }} className="dict-cell">
          <div className="flex items-center gap-1.5 mb-1.5">
            <span className="text-xs">{'\u270F\uFE0F'}</span>
            <span className="text-[10px] font-bold uppercase tracking-wider text-text-disabled">Notes</span>
          </div>
          {entry.notes ? (
            <>
              <p className="text-xs leading-relaxed text-text-secondary whitespace-pre-wrap line-clamp-3">
                {entry.notes}
              </p>
              <button
                type="button"
                onClick={() => setExpanded(expanded === 'notes' ? null : 'notes')}
                className="text-[11px] font-semibold text-text-disabled hover:text-cornflower transition-colors duration-100 mt-1.5"
              >
                {expanded === 'notes' ? '\u2190 Collapse' : 'Show more \u2192'}
              </button>
            </>
          ) : (
            <button
              type="button"
              onClick={() => setEditWordId(entry.id)}
              className="w-full py-3 rounded-lg border border-dashed border-border-default text-xs text-text-disabled hover:text-text-tertiary hover:border-text-disabled transition-colors duration-150 cursor-pointer"
            >
              + Add notes
            </button>
          )}
        </div>
      </div>

      {/* ─── EXPANDED PANEL ─── */}
      {expanded === 'notes' && entry.notes && (
        <div className="mt-2 bg-surface-secondary rounded-[10px] p-4 animate-card-in">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-1.5">
              <span className="text-sm">{'\u270F\uFE0F'}</span>
              <span className="text-[11px] font-bold uppercase tracking-wider text-text-tertiary">Notes</span>
            </div>
            <button
              type="button"
              onClick={() => setExpanded(null)}
              className="border border-border-default rounded-md px-2.5 py-0.5 text-[10px] font-semibold text-text-tertiary hover:text-text-secondary hover:border-text-tertiary transition-all duration-100"
            >
              {'\u2190'} Collapse
            </button>
          </div>
          <p className="text-[13px] leading-relaxed text-text-primary whitespace-pre-wrap">{entry.notes}</p>
        </div>
      )}

      {/* ─── FOOTER ─── */}
      <div className="px-4 py-2 mt-1 flex items-center justify-end">
        <span className="hidden md:inline-flex items-center gap-1.5 text-[10px] text-text-disabled">
          <span className="dict-kbd">{'\u2191'}</span>
          <span className="dict-kbd">{'\u2193'}</span>
          navigate
          <span className="ml-1 dict-kbd">Esc</span>
          close
        </span>
      </div>

      {/* Edit dialog */}
      <WordEditDialog
        wordId={editWordId}
        open={editWordId !== null}
        onOpenChange={(open) => !open && setEditWordId(null)}
      />

      {/* Delete confirmation */}
      <AlertDialog open={pendingEntry !== null} onOpenChange={(open) => !open && cancelDelete()}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('delete.confirm')}</AlertDialogTitle>
            <AlertDialogDescription>{t('delete.confirmDescription')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={cancelDelete}>
              {t('delete.confirmCancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              className="bg-poppy text-white hover:bg-poppy/90"
              onClick={() => { confirmDelete(); onClose() }}
            >
              {t('delete.confirmDelete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </motion.div>
  )
}
