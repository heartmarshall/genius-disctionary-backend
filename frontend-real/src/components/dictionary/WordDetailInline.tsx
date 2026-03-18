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
  index: number
  onClose: () => void
}

function ContentSkeleton() {
  return (
    <div className="space-y-2 py-1">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="flex items-center gap-2">
          <div
            className="h-3.5 animate-pulse"
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

export function WordDetailInline({ wordId, index, onClose }: WordDetailInlineProps) {
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(wordId)
  const { requestDelete, confirmDelete, cancelDelete, pendingEntry } = useDeleteWord()
  const [editWordId, setEditWordId] = useState<string | null>(null)

  const primaryPos = entry?.senses[0]?.partOfSpeech
  const posColor = primaryPos ? (TERM_POS_COLORS[primaryPos] ?? 'var(--term-text-muted)') : 'var(--term-border)'

  return (
    <motion.div
      initial={{ opacity: 0, y: -4 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.15, ease: 'easeOut' }}
      className="py-4 pr-3 overflow-hidden font-mono text-sm rounded-lg"
      style={{ background: 'var(--term-bg-raised)' }}
    >
      {/* Header: index  word  POS — same X positions as list row */}
      <div className="flex items-baseline justify-between gap-2">
        <div className="flex items-baseline gap-2 min-w-0 flex-wrap">
          <span className="tabular-nums w-6 text-right shrink-0 select-none" style={{ color: 'var(--term-text-dim)' }}>
            {index + 1}
          </span>
          <button
            type="button"
            onClick={onClose}
            className="font-bold hover:underline shrink-0"
            style={{ color: 'var(--term-text-bright)' }}
          >
            {entry?.text ?? '...'}
          </button>
        </div>
        <div className="flex items-baseline shrink-0 gap-2">
          {entry && (
            <>
              <button
                type="button"
                onClick={() => setEditWordId(entry.id)}
                className="hover:underline"
                style={{ color: 'var(--term-yellow)' }}
              >
                [edit]
              </button>
              <button
                type="button"
                onClick={() => requestDelete(entry)}
                className="hover:underline"
                style={{ color: 'var(--term-red)' }}
              >
                [del]
              </button>
            </>
          )}
          <button
            type="button"
            onClick={onClose}
            className="hover:underline"
            style={{ color: 'var(--term-text-muted)' }}
          >
            [x]
          </button>
        </div>
      </div>

      {/* Body */}
      <div className="mt-3 ml-8">
        {loading && <ContentSkeleton />}

        {error && (
          <div className="py-2">
            <p style={{ color: 'var(--term-red)' }}>{t('error.loadFailed')}</p>
            <button
              type="button"
              className="mt-1 hover:underline"
              style={{ color: 'var(--term-yellow)' }}
              onClick={() => window.location.reload()}
            >
              [retry]
            </button>
          </div>
        )}

        {entry && <WordOverview entry={entry} hideTitle hideHeader />}
      </div>

      {/* Footer */}
      {entry && (
        <div className="mt-3 ml-4 flex flex-wrap items-baseline gap-x-3 gap-y-0.5">
          <span style={{ color: 'var(--term-text-dim)' }}>
            created {formatDate(entry.createdAt)}
            {entry.updatedAt !== entry.createdAt && `  upd ${formatDate(entry.updatedAt)}`}
            {'  '}id:{entry.id.slice(0, 8)}
          </span>
          <span className="flex-1" />
          <span className="hidden md:inline-flex items-center gap-1 text-[10px]" style={{ color: 'var(--term-text-dim)' }}>
            <span className="term-kbd">↑</span><span className="term-kbd">↓</span> navigate
            <span className="ml-1 term-kbd">Esc</span> close
          </span>
        </div>
      )}

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
