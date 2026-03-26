import { useState, useCallback } from 'react'
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
import { getPosColors } from '@/lib/pos-colors'

// ─── Types ───────────────────────────────────────────────────────────────────

interface WordDetailInlineProps {
  wordId: string
  index: number
  onClose: () => void
}

type TabId = 'word' | 'context' | 'progress'

// ─── Helpers ─────────────────────────────────────────────────────────────────

function formatDate(iso: string): string {
  return iso.split('T')[0]
}

const POS_LABEL: Record<string, string> = {
  NOUN: 'noun', VERB: 'verb', ADJECTIVE: 'adjective', ADVERB: 'adverb',
  PRONOUN: 'pronoun', PREPOSITION: 'preposition', CONJUNCTION: 'conjunction',
  INTERJECTION: 'interjection', PHRASE: 'phrase', IDIOM: 'idiom', OTHER: 'other',
}

const TAG_STYLES = [
  { bg: 'var(--cornflower-light)', color: 'var(--cornflower)' },
  { bg: 'var(--thyme-light)', color: 'var(--thyme)' },
  { bg: 'var(--goldenrod-light)', color: 'var(--goldenrod)' },
  { bg: 'var(--source-book-light)', color: 'var(--source-book)' },
  { bg: 'var(--source-screen-light)', color: 'var(--source-screen)' },
]

// ─── Skeleton ────────────────────────────────────────────────────────────────

function ContentSkeleton() {
  return (
    <div className="space-y-3 py-2">
      {[180, 130, 220, 100].map((w, i) => (
        <div key={i} className="h-4 rounded-md dict-shimmer" style={{ width: w, animationDelay: `${i * 0.1}s` }} />
      ))}
    </div>
  )
}

// ─── Icon button ─────────────────────────────────────────────────────────────

function IconBtn({ onClick, children, title, danger }: {
  onClick: () => void; children: React.ReactNode; title: string; danger?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={title}
      className={`w-8 h-8 rounded-lg flex items-center justify-center text-text-disabled transition-colors duration-100 ${
        danger ? 'hover:bg-poppy-light hover:text-poppy' : 'hover:bg-surface-secondary hover:text-text-secondary'
      }`}
    >
      {children}
    </button>
  )
}

// ─── Sense block ─────────────────────────────────────────────────────────────

function SenseBlock({ sense, index }: { sense: Sense; index: number }) {
  const { t } = useTranslation('dictionary')
  const posLabel = sense.partOfSpeech ? t(`pos.${sense.partOfSpeech}`) : undefined

  return (
    <div className="bg-card-head rounded-[14px] p-5">
      <div className="flex items-center gap-2.5 mb-2.5">
        <span
          className="w-[22px] h-[22px] rounded-full flex items-center justify-center font-serif text-[13px] font-bold"
          style={{ background: 'var(--cornflower-light)', color: 'var(--cornflower)' }}
        >
          {index + 1}
        </span>
        {posLabel && (
          <span className="font-mono text-[10px] text-text-disabled uppercase tracking-wider">
            {posLabel}
          </span>
        )}
      </div>

      {sense.definition && (
        <p className="text-[15px] leading-relaxed text-text-primary mb-1 pl-8 font-medium">
          {sense.definition}
        </p>
      )}

      {sense.translations.length > 0 && (
        <p className="text-[13px] italic mb-3.5 pl-8 text-cornflower opacity-85">
          {sense.translations.map(tr => tr.text).join(', ')}
        </p>
      )}

      {sense.examples.length > 0 && (
        <div className="space-y-1.5 pl-8">
          {sense.examples.map((ex) => (
            <div key={ex.id} className="bg-surface-secondary rounded-[10px] px-4 py-3">
              <p className="text-[13.5px] text-text-secondary italic leading-snug">
                &ldquo;{ex.sentence}&rdquo;
              </p>
              {ex.translation && (
                <p className="text-[12.5px] text-text-tertiary mt-1.5">{ex.translation}</p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── Tab button ──────────────────────────────────────────────────────────────

function TabBtn({ label, active, badge, onClick }: {
  label: string; active: boolean; badge?: number | null; onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex items-center gap-1.5 px-4 pb-2.5 pt-2 font-mono text-[11px] transition-all duration-150"
      style={{
        color: active ? 'var(--cornflower)' : 'var(--text-disabled)',
        fontWeight: active ? 600 : 500,
        borderBottom: active ? '2px solid var(--cornflower)' : '2px solid transparent',
      }}
    >
      {label}
      {badge != null && badge > 0 && (
        <span className="text-[9px] font-bold px-1.5 py-0.5 rounded-lg" style={{ background: 'var(--cornflower-light)', color: 'var(--cornflower)' }}>
          {badge}
        </span>
      )}
    </button>
  )
}

// ─── GRADE_COLOR ─────────────────────────────────────────────────────────────

const GRADE_COLOR: Record<string, string> = {
  AGAIN: 'bg-poppy', HARD: 'bg-goldenrod', GOOD: 'bg-thyme', EASY: 'bg-cornflower',
}

// ─── Word Tab ────────────────────────────────────────────────────────────────

function WordTab({ entry, playAudio }: {
  entry: NonNullable<ReturnType<typeof useWordDetail>['entry']>
  playAudio: (url: string) => void
}) {
  const allImages = [...entry.catalogImages, ...entry.userImages]

  return (
    <div className="grid gap-2.5" style={{ gridTemplateColumns: '1fr 200px' }}>
      <div className="flex flex-col gap-2.5">
        {entry.senses.length > 0 ? (
          entry.senses.map((sense, i) => <SenseBlock key={sense.id} sense={sense} index={i} />)
        ) : (
          <div className="bg-card-head rounded-[14px] p-5">
            <p className="text-xs text-text-disabled italic">No definitions yet</p>
          </div>
        )}
      </div>

      {/* Right column */}
      <div className="flex flex-col gap-2.5">
        <div className="bg-card-head rounded-[14px] overflow-hidden">
          <div className="h-[90px] flex items-center justify-center">
            {allImages.length > 0 ? (
              <img src={allImages[0].url} alt={entry.text} className="w-full h-full object-cover" />
            ) : (
              <div className="w-full h-full bg-surface-secondary flex items-center justify-center text-text-disabled">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" className="opacity-30">
                  <rect x="3" y="3" width="18" height="18" rx="2" />
                  <circle cx="8.5" cy="8.5" r="1.5" />
                  <path d="m21 15-5-5L5 21" />
                </svg>
              </div>
            )}
          </div>
          <div className="flex border-t" style={{ borderColor: 'var(--border-subtle)' }}>
            {entry.pronunciations.length > 0 ? (
              entry.pronunciations.map((pron, i) => (
                <button
                  key={pron.id}
                  type="button"
                  onClick={() => pron.audioUrl && playAudio(pron.audioUrl)}
                  className="flex-1 py-1.5 text-[10px] font-mono font-semibold text-text-tertiary hover:text-cornflower transition-colors flex items-center justify-center gap-1"
                >
                  {pron.audioUrl && <Volume2 size={10} />}
                  {pron.region ? pron.region.toUpperCase() : `#${i + 1}`}
                </button>
              ))
            ) : (
              <>
                <button type="button" className="flex-1 py-1.5 text-[10px] font-mono font-semibold text-text-disabled">UK</button>
                <button type="button" className="flex-1 py-1.5 text-[10px] font-mono font-semibold text-text-disabled">US</button>
              </>
            )}
          </div>
        </div>

        {allImages.length > 1 && (
          <div className="bg-card-head rounded-[14px] p-3">
            <div className="grid grid-cols-3 gap-1">
              {allImages.slice(1, 4).map(img => (
                <img key={img.id} src={img.url} alt="" className="w-full aspect-square object-cover rounded" />
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Context Tab ─────────────────────────────────────────────────────────────

function ContextTab({ entry, onEdit }: {
  entry: NonNullable<ReturnType<typeof useWordDetail>['entry']>; onEdit: () => void
}) {
  const [notesExpanded, setNotesExpanded] = useState(false)

  return (
    <div className="flex flex-col gap-2.5">
      <div className="bg-card-head rounded-[14px] p-5">
        <span className="font-mono text-[10px] text-text-disabled uppercase tracking-widest block mb-3">Notes</span>
        {entry.notes ? (
          <div className="relative">
            <div
              className="text-[13px] text-text-secondary leading-relaxed whitespace-pre-wrap"
              style={{ maxHeight: notesExpanded ? undefined : 72, overflow: 'hidden' }}
            >
              {entry.notes}
            </div>
            {!notesExpanded && entry.notes.length > 120 && (
              <div className="absolute bottom-0 left-0 right-0 h-10 pointer-events-none" style={{ background: 'linear-gradient(transparent, var(--card-head))' }} />
            )}
            {entry.notes.length > 120 && (
              <button
                type="button"
                onClick={() => setNotesExpanded(!notesExpanded)}
                className="mt-2 font-mono text-[11px] text-cornflower opacity-80 hover:opacity-100 transition-opacity flex items-center gap-1.5"
              >
                {notesExpanded ? 'Show less' : 'Show more'}
                <span className="inline-block text-[9px] transition-transform duration-200" style={{ transform: notesExpanded ? 'rotate(180deg)' : '' }}>▼</span>
              </button>
            )}
          </div>
        ) : (
          <button type="button" onClick={onEdit} className="text-text-disabled text-[13px] hover:text-text-tertiary transition-colors">
            + Add notes...
          </button>
        )}
      </div>
    </div>
  )
}

// ─── Progress Tab ────────────────────────────────────────────────────────────

const SRS_STAGES = [
  { key: 'NEW', label: 'new', color: 'var(--text-disabled)' },
  { key: 'LEARNING', label: 'learning', color: 'var(--goldenrod)' },
  { key: 'REVIEW', label: 'review', color: 'var(--cornflower)' },
  { key: 'RELEARNING', label: 'relearning', color: 'var(--poppy)' },
] as const

function ProgressTab({ entry, stats, logs, onCreateCard, creating }: {
  entry: NonNullable<ReturnType<typeof useWordDetail>['entry']>
  stats?: CardStats; logs?: ReviewLog[]; onCreateCard: () => void; creating?: boolean
}) {
  if (!entry.card) {
    return (
      <div className="bg-card-head rounded-[14px] p-5">
        <span className="font-mono text-[10px] text-text-disabled uppercase tracking-widest block mb-3">Progress</span>
        <button
          type="button" onClick={onCreateCard} disabled={creating}
          className="w-full py-4 rounded-lg border border-dashed text-xs font-medium text-text-disabled hover:text-cornflower hover:border-cornflower transition-colors disabled:opacity-50"
          style={{ borderColor: 'var(--text-disabled)' }}
        >
          {creating ? 'Creating...' : '+ Create card'}
        </button>
      </div>
    )
  }

  if (!stats) return null

  const accuracyPct = Math.round(stats.accuracy * 100)
  const streak = (() => { if (!logs?.length) return 0; let c = 0; for (const l of logs) { if (l.grade === 'GOOD' || l.grade === 'EASY') c++; else break } return c })()
  const nextReviewLabel = (() => {
    if (!entry.card?.due) return '—'
    const d = Math.ceil((new Date(entry.card.due).getTime() - Date.now()) / 86400000)
    if (d <= 0) return 'Now'; if (d === 1) return 'Tomorrow'; return `${d}d`
  })()
  const currentState = entry.card?.state ?? 'NEW'

  return (
    <div className="flex flex-col gap-2.5">
      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-2.5">
        {[
          { value: streak, label: 'Streak', color: streak > 5 ? 'var(--thyme)' : streak > 0 ? 'var(--goldenrod)' : 'var(--text-disabled)' },
          { value: `${accuracyPct}%`, label: 'Accuracy', color: accuracyPct > 80 ? 'var(--thyme)' : accuracyPct > 50 ? 'var(--goldenrod)' : 'var(--text-disabled)' },
          { value: stats.totalReviews, label: 'Reviews', color: 'var(--text-secondary)' },
          { value: nextReviewLabel, label: 'Next review', color: nextReviewLabel === 'Now' ? 'var(--poppy)' : 'var(--text-secondary)' },
        ].map((s, i) => (
          <div key={i} className="bg-card-head rounded-[14px] py-4 px-3 text-center">
            <div className="font-serif text-[26px] font-bold leading-none" style={{ color: s.color }}>{s.value}</div>
            <div className="font-mono text-[9px] text-text-disabled uppercase tracking-wider mt-1.5">{s.label}</div>
          </div>
        ))}
      </div>

      {/* Review history */}
      {logs && logs.length > 0 && (
        <div className="bg-card-head rounded-[14px] p-5">
          <div className="flex justify-between items-center mb-3.5">
            <span className="font-mono text-[10px] text-text-disabled uppercase tracking-widest">Review history</span>
            <span className="font-mono text-[10px] text-text-disabled">Last {Math.min(logs.length, 20)}</span>
          </div>
          <div className="flex gap-1.5 items-end h-8">
            {logs.slice(0, 20).reverse().map((log, i) => {
              const ok = log.grade === 'GOOD' || log.grade === 'EASY'
              return (
                <div
                  key={log.id}
                  className={`flex-1 rounded ${GRADE_COLOR[log.grade] ?? 'bg-text-disabled'}`}
                  style={{ height: ok ? `${28 + Math.round((i / 20) * 12)}px` : '8px', opacity: ok ? 0.3 + (i / 20) * 0.7 : 1, minWidth: 4, maxWidth: 12 }}
                  title={`${log.grade} — ${formatDate(log.reviewedAt)}`}
                />
              )
            })}
          </div>
          <div className="flex gap-3 mt-3 font-mono text-[10px] text-text-disabled">
            <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-cornflower opacity-50" /> correct</span>
            <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-sm bg-surface-secondary" /> missed</span>
            <span className="ml-auto">{logs.filter(l => l.grade === 'GOOD' || l.grade === 'EASY').length}/{logs.length} correct</span>
          </div>
        </div>
      )}

      {/* SRS stage */}
      <div className="bg-card-head rounded-[14px] p-5">
        <span className="font-mono text-[10px] text-text-disabled uppercase tracking-widest block mb-3">SRS Stage</span>
        <div className="flex gap-1 items-center">
          {SRS_STAGES.map((stage) => {
            const active = stage.key === currentState
            const past = SRS_STAGES.findIndex(s => s.key === stage.key) < SRS_STAGES.findIndex(s => s.key === currentState)
            return (
              <div key={stage.key} className="flex-1 flex flex-col gap-1.5">
                <div className="h-1.5 rounded-full" style={{ background: active ? stage.color : past ? `${stage.color}60` : 'var(--surface-secondary)' }} />
                <div className="font-mono text-[9px] uppercase tracking-wider text-center" style={{ color: active ? stage.color : past ? 'var(--text-tertiary)' : 'var(--text-disabled)', fontWeight: active ? 700 : 400 }}>
                  {stage.label}
                </div>
              </div>
            )
          })}
        </div>
        <div className="mt-3.5 text-[12px] text-text-tertiary leading-relaxed">
          {currentState === 'NEW' && "This word hasn't been reviewed yet."}
          {currentState === 'LEARNING' && `Actively learning. Streak: ${streak}.`}
          {currentState === 'REVIEW' && `In review cycle. ${accuracyPct}% accuracy.`}
          {currentState === 'RELEARNING' && 'Needs more practice.'}
        </div>
      </div>
    </div>
  )
}

// ─── Unfold animation ────────────────────────────────────────────────────────

const ENTER = {
  clipPath: 'inset(0 0% 0% 0% round 18px)',
  scaleY: 1,
  y: 0,
}
const INITIAL = {
  clipPath: 'inset(0 1% 40% 1% round 18px)',
  scaleY: 0.92,
  y: -3,
}
const EXIT = {
  clipPath: 'inset(0 1% 40% 1% round 18px)',
  scaleY: 0.94,
  y: -2,
}
const ENTER_TRANSITION = { duration: 0.2, ease: [0.22, 1, 0.36, 1] }
const EXIT_TRANSITION = { duration: 0.15, ease: [0.4, 0, 0.7, 1] }

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
  const [activeTab, setActiveTab] = useState<TabId>('word')
  const [closing, setClosing] = useState(false)

  const handleClose = useCallback(() => {
    if (closing) return
    setClosing(true)
    setTimeout(() => onClose(), EXIT_TRANSITION.duration * 1000)
  }, [closing, onClose])

  const primaryPos = entry?.senses[0]?.partOfSpeech
  const posLabel = primaryPos ? POS_LABEL[primaryPos] ?? primaryPos.toLowerCase() : ''
  const ipa = entry?.pronunciations[0]?.transcription
  const cleanIpa = ipa ? `/${ipa.replace(/^\/+|\/+$/g, '')}/` : undefined

  const playAudio = async (url: string) => {
    try { await new Audio(url).play() } catch { toast.error(t('error.audioFailed')) }
  }

  if (loading) {
    return (
      <motion.div
        initial={INITIAL}
        animate={closing ? EXIT : ENTER}
        transition={closing ? EXIT_TRANSITION : ENTER_TRANSITION}
        style={{ transformOrigin: 'top center' }}
        className="rounded-[18px] overflow-hidden shadow-bento mx-[-24px] my-3"
      >
        <div className="bg-card-head p-7"><ContentSkeleton /></div>
      </motion.div>
    )
  }

  if (error) {
    return (
      <div className="rounded-[18px] overflow-hidden shadow-bento mx-[-24px] my-3">
        <div className="bg-card-head p-7">
          <p className="text-sm text-poppy">{t('error.loadFailed')}</p>
          <button type="button" className="mt-1.5 text-sm text-cornflower hover:underline" onClick={() => window.location.reload()}>{t('error.tryAgain')}</button>
        </div>
      </div>
    )
  }

  if (!entry) return null

  const contextCount = entry.notes ? 1 : 0

  return (
    <motion.div
      initial={INITIAL}
      animate={closing ? EXIT : ENTER}
      transition={closing ? EXIT_TRANSITION : ENTER_TRANSITION}
      style={{ transformOrigin: 'top center' }}
      className="rounded-[18px] overflow-hidden shadow-bento mx-[-24px] my-3 relative z-[2]"
    >
      {/* ─── HEADER ─── card-head bg */}
      <div className="bg-card-head px-8 pt-7 pb-0">
        <div className="flex justify-between items-start">
          <div className="flex items-baseline gap-3 min-w-0">
            <h2
              onClick={handleClose}
              className="text-[28px] font-serif font-bold text-text-primary tracking-tight leading-none cursor-pointer transition-colors duration-150 hover:text-cornflower"
            >
              {entry.text}
            </h2>
            {cleanIpa && <span className="font-mono text-[14px] text-text-disabled">{cleanIpa}</span>}
            {posLabel && (
              <span className="text-[11.5px] text-text-tertiary bg-surface-secondary px-2.5 py-0.5 rounded-full">{posLabel}</span>
            )}
          </div>
          <div className="flex items-center gap-1 shrink-0 ml-4">
            <IconBtn onClick={() => setEditWordId(entry.id)} title="Edit"><Pencil size={15} /></IconBtn>
            <IconBtn onClick={() => requestDelete(entry)} title="Delete" danger><Trash2 size={15} /></IconBtn>
            <IconBtn onClick={handleClose} title="Close"><X size={16} /></IconBtn>
          </div>
        </div>

        <div className="flex items-center gap-1.5 mt-3 flex-wrap">
          {entry.topics.map((tp, i) => (
            <span key={tp.id} className="px-2.5 py-0.5 rounded-full text-[11px] font-medium"
              style={{ background: TAG_STYLES[i % TAG_STYLES.length].bg, color: TAG_STYLES[i % TAG_STYLES.length].color }}>
              {tp.name}
            </span>
          ))}
          <span className="font-mono text-[10px] text-text-disabled ml-1">
            added {formatDate(entry.createdAt)}
          </span>
        </div>

        {/* Tabs inside header */}
        <div className="flex gap-0 mt-4">
          <TabBtn label="Word" active={activeTab === 'word'} onClick={() => setActiveTab('word')} />
          <TabBtn label="Context" active={activeTab === 'context'} badge={contextCount || null} onClick={() => setActiveTab('context')} />
          <TabBtn label="Progress" active={activeTab === 'progress'} onClick={() => setActiveTab('progress')} />
        </div>
      </div>

      {/* ─── TAB CONTENT ─── card-body bg */}
      <div className="bg-card-body p-4">
        {activeTab === 'word' && <WordTab entry={entry} playAudio={playAudio} />}
        {activeTab === 'context' && <ContextTab entry={entry} onEdit={() => setEditWordId(entry.id)} />}
        {activeTab === 'progress' && (
          <ProgressTab entry={entry} stats={cardStats} logs={reviewLogs} onCreateCard={() => createCard()} creating={creatingCard} />
        )}
      </div>

      {/* ─── FOOTER ─── card-head bg */}
      <div className="flex items-center justify-end gap-1.5 px-6 py-3 bg-card-head">
        <span className="hidden md:inline-flex items-center gap-1.5 font-mono text-[10px] text-text-disabled">
          <span className="dict-kbd">&uarr;</span>
          <span className="dict-kbd">&darr;</span>
          <span className="mr-1">navigate</span>
          <span className="dict-kbd">Esc</span>
          <span>close</span>
        </span>
      </div>

      {/* Edit dialog */}
      <WordEditDialog wordId={editWordId} open={editWordId !== null} onOpenChange={(open) => !open && setEditWordId(null)} />

      {/* Delete confirmation */}
      <AlertDialog open={pendingEntry !== null} onOpenChange={(open) => !open && cancelDelete()}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('delete.confirm')}</AlertDialogTitle>
            <AlertDialogDescription>{t('delete.confirmDescription')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={cancelDelete}>{t('delete.confirmCancel')}</AlertDialogCancel>
            <AlertDialogAction className="bg-poppy text-white hover:bg-poppy/90" onClick={() => { confirmDelete(); handleClose() }}>
              {t('delete.confirmDelete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </motion.div>
  )
}
