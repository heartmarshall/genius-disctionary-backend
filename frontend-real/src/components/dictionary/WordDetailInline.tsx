import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { motion } from 'framer-motion'
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
import { WordOverview } from './WordOverview'
import { WordEditDialog } from './WordEditDialog'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useDeleteWord } from '@/hooks/useDeleteWord'

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

interface WordDetailInlineProps {
  wordId: string
  onClose: () => void
}

function ContentSkeleton() {
  return (
    <div className="space-y-1.5 py-1">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="flex items-center gap-2">
          <div
            className="h-3 animate-pulse"
            style={{ width: 50 + Math.random() * 180, background: 'var(--term-border)', opacity: 0.4 }}
          />
        </div>
      ))}
    </div>
  )
}

function formatDate(iso: string): string {
  const d = new Date(iso)
  const pad = (n: number) => n.toString().padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function WordDetailInline({ wordId, onClose }: WordDetailInlineProps) {
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(wordId)
  const { requestDelete, confirmDelete, cancelDelete, pendingEntry } = useDeleteWord()
  const [editWordId, setEditWordId] = useState<string | null>(null)

  const primaryPos = entry?.senses[0]?.partOfSpeech
  const posColor = primaryPos ? (TERM_POS_COLORS[primaryPos] ?? 'var(--term-text-muted)') : 'var(--term-border)'

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.12 }}
      className="term-detail-card"
      style={{ borderLeftColor: posColor }}
    >
      {/* Header line */}
      <div className="flex items-baseline justify-between gap-3 px-4 py-2.5">
        <div className="flex items-baseline gap-2 min-w-0 flex-wrap">
          <button
            type="button"
            onClick={onClose}
            className="text-base font-bold font-mono hover:underline shrink-0"
            style={{ color: 'var(--term-text-bright)' }}
          >
            {entry?.text ?? '...'}
          </button>
          {primaryPos && (
            <span className="text-xs shrink-0" style={{ color: posColor }}>
              {t(`pos.${primaryPos}`)}
            </span>
          )}
          {entry && (
            <span className="text-xs shrink-0" style={{ color: 'var(--term-text-dim)' }}>
              {entry.senses.length}s {entry.senses.reduce((n, s) => n + s.examples.length, 0)}ex
            </span>
          )}
        </div>
        <div className="flex items-baseline shrink-0 gap-2">
          {entry && (
            <>
              <button
                type="button"
                onClick={() => setEditWordId(entry.id)}
                className="text-xs hover:underline"
                style={{ color: 'var(--term-yellow)' }}
              >
                [edit]
              </button>
              <button
                type="button"
                onClick={() => requestDelete(entry)}
                className="text-xs hover:underline"
                style={{ color: 'var(--term-red)' }}
              >
                [del]
              </button>
            </>
          )}
          <button
            type="button"
            onClick={onClose}
            className="text-xs hover:underline"
            style={{ color: 'var(--term-text-muted)' }}
          >
            [x]
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="px-4 pb-3">
        {loading && <ContentSkeleton />}

        {error && (
          <div className="py-2">
            <p className="text-xs" style={{ color: 'var(--term-red)' }}>
              stderr: {t('error.loadFailed')}
            </p>
            <button
              type="button"
              className="mt-1 text-xs hover:underline"
              style={{ color: 'var(--term-yellow)' }}
              onClick={() => window.location.reload()}
            >
              [retry]
            </button>
          </div>
        )}

        {entry && (
          <>
            <WordOverview entry={entry} hideHeader primaryPosShownInHeader={primaryPos} />

            {/* Metadata footer */}
            <div className="mt-3 pt-2 flex flex-wrap items-baseline gap-x-4 gap-y-0.5" style={{ borderTop: '1px dotted var(--term-border)' }}>
              <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>
                created {formatDate(entry.createdAt)}
              </span>
              {entry.updatedAt !== entry.createdAt && (
                <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>
                  updated {formatDate(entry.updatedAt)}
                </span>
              )}
              <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>
                id:{entry.id.slice(0, 8)}
              </span>
            </div>
          </>
        )}
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
