import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { motion } from 'framer-motion'
import { Button } from '@/components/ui/button'
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
import { Skeleton } from '@/components/ui/skeleton'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useDeleteWord } from '@/hooks/useDeleteWord'

interface DictionaryDetailViewProps {
  wordId: string
  onBack: () => void
}

function DetailSkeleton() {
  return (
    <div className="space-y-6 py-4">
      <Skeleton className="h-12 w-64" />
      <Skeleton className="h-6 w-40" />
      <div className="space-y-4 pt-4">
        <Skeleton className="h-4 w-full" />
        <Skeleton className="h-4 w-3/4" />
        <Skeleton className="h-4 w-1/2" />
      </div>
    </div>
  )
}

export function DictionaryDetailView({ wordId, onBack }: DictionaryDetailViewProps) {
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(wordId)
  const { requestDelete, confirmDelete, cancelDelete, pendingEntry } = useDeleteWord()
  const [editWordId, setEditWordId] = useState<string | null>(null)

  return (
    <motion.div
      className="min-h-full"
      initial={{ opacity: 0, x: 40 }}
      animate={{ opacity: 1, x: 0 }}
      exit={{ opacity: 0, x: 40 }}
      transition={{ duration: 0.35, ease: [0.4, 0, 0.2, 1] }}
    >
      {/* Back button */}
      <button
        type="button"
        onClick={onBack}
        className="flex items-center gap-2 text-sm text-text-secondary hover:text-text-primary transition-colors mb-8"
      >
        <ArrowLeft size={16} />
        {t('detail.back')}
      </button>

      {loading && <DetailSkeleton />}

      {error && (
        <div className="flex flex-col items-center gap-3 py-12 rounded-xl border border-poppy/20 bg-poppy-light">
          <p className="text-sm text-poppy-fg">{t('error.loadFailed')}</p>
          <Button
            variant="outline"
            size="sm"
            className="rounded-full border-poppy/30 text-poppy-fg hover:bg-poppy-light"
            onClick={() => window.location.reload()}
          >
            {t('error.tryAgain')}
          </Button>
        </div>
      )}

      {entry && (
        <div className="space-y-8">
          <WordOverview entry={entry} />

          {/* Actions */}
          <div className="flex gap-3">
            <Button
              onClick={() => setEditWordId(entry.id)}
              className="rounded-lg"
            >
              {t('actions.edit')}
            </Button>
            <Button
              variant="outline"
              onClick={() => requestDelete(entry)}
              className="rounded-lg"
            >
              {t('actions.delete')}
            </Button>
          </div>
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
        <AlertDialogContent className="rounded-xl">
          <AlertDialogHeader>
            <AlertDialogTitle>{t('delete.confirm')}</AlertDialogTitle>
            <AlertDialogDescription>{t('delete.confirmDescription')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="rounded-full" onClick={cancelDelete}>
              {t('delete.confirmCancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              className="rounded-full bg-poppy text-white hover:bg-poppy/90"
              onClick={() => { confirmDelete(); onBack() }}
            >
              {t('delete.confirmDelete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </motion.div>
  )
}
