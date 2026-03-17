import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X, Pencil, Trash2, Volume2 } from 'lucide-react'
import { toast } from 'sonner'
import { motion } from 'framer-motion'
import { cn } from '@/lib/utils'
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
import { getPosColors } from '@/lib/pos-colors'

interface WordDetailInlineProps {
  wordId: string
  onClose: () => void
}

function HeaderSkeleton() {
  return (
    <div className="space-y-5">
      <Skeleton className="h-12 w-64" />
      <Skeleton className="h-4 w-40" />
    </div>
  )
}

function ContentSkeleton() {
  return (
    <div className="space-y-4 mt-10">
      <Skeleton className="h-4 w-full max-w-xl" />
      <Skeleton className="h-4 w-3/4 max-w-md" />
      <Skeleton className="h-4 w-1/2 max-w-sm" />
    </div>
  )
}

export function WordDetailInline({ wordId, onClose }: WordDetailInlineProps) {
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(wordId)
  const { requestDelete, confirmDelete, cancelDelete, pendingEntry } = useDeleteWord()
  const [editWordId, setEditWordId] = useState<string | null>(null)

  const primaryPos = entry?.senses[0]?.partOfSpeech
  const posColors = getPosColors(primaryPos)

  const playAudio = async (url: string) => {
    try {
      await new Audio(url).play()
    } catch {
      toast.error(t('error.audioFailed'))
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
      className="my-4 py-1 -mx-5 md:-mx-6 rounded-2xl border border-border-subtle shadow-sm"
    >
      {/* Header — word title row with actions */}
      <div className="flex items-start justify-between gap-4 pt-3 md:pt-4 px-4 md:px-5">
        <div className="flex-1 min-w-0">
          {loading && <HeaderSkeleton />}

          {entry && (
            <>
              {/* Meta — POS (matching list style) + topics */}
              <div className="flex items-center gap-2 flex-wrap mb-2">
                {primaryPos && (
                  <span className={cn('text-xs font-medium', posColors.text)}>
                    {t(`pos.${primaryPos}`)}
                  </span>
                )}
                {entry.topics.length > 0 && (primaryPos ? (
                  <span className="text-text-disabled">·</span>
                ) : null)}
                {entry.topics.map((topic, i) => (
                  <span key={topic.id} className="text-[11px] uppercase tracking-widest text-text-tertiary">
                    {topic.name}{i < entry.topics.length - 1 ? ',' : ''}
                  </span>
                ))}
              </div>

              {/* Word + IPA inline */}
              <div className="flex items-baseline gap-4 flex-wrap">
                <h2
                  onClick={onClose}
                  className="text-2xl md:text-3xl font-medium tracking-[-0.03em] text-text-primary leading-tight cursor-pointer hover:opacity-70 transition-opacity duration-200"
                >
                  {entry.text}
                </h2>
                {entry.pronunciations.length > 0 && (
                  <div className="flex items-center gap-4 flex-wrap">
                    {entry.pronunciations.map((pron) => (
                      <div key={pron.id} className="inline-flex items-center gap-1.5">
                        <span className="text-sm text-text-secondary">
                          /{pron.transcription.replace(/^\/+|\/+$/g, '')}/
                        </span>
                        {pron.region && (
                          <span className="text-[9px] font-medium uppercase tracking-widest text-text-tertiary">
                            {pron.region}
                          </span>
                        )}
                        {pron.audioUrl && (
                          <button
                            type="button"
                            className="text-text-tertiary hover:text-text-primary transition-colors duration-200"
                            onClick={() => playAudio(pron.audioUrl!)}
                          >
                            <Volume2 className="h-3.5 w-3.5" />
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        {/* Actions — right side, larger touch targets (Fitts's Law) */}
        <div className="flex items-center shrink-0 gap-1">
          {entry && (
            <div className="flex items-center gap-0.5">
              <button
                type="button"
                onClick={() => setEditWordId(entry.id)}
                className="text-text-tertiary hover:text-text-primary hover:bg-surface-secondary transition-colors duration-200 p-2.5 rounded-lg"
                aria-label={t('actions.edit')}
              >
                <Pencil size={18} />
              </button>
              <button
                type="button"
                onClick={() => requestDelete(entry)}
                className="text-text-tertiary hover:text-poppy hover:bg-poppy-light transition-colors duration-200 p-2.5 rounded-lg"
                aria-label={t('actions.delete')}
              >
                <Trash2 size={18} />
              </button>
            </div>
          )}
          <div className="w-px h-5 bg-border-subtle/60 mx-1" />
          <button
            type="button"
            onClick={onClose}
            className="text-text-tertiary hover:text-text-primary hover:bg-surface-secondary transition-colors duration-200 p-2.5 rounded-lg"
            aria-label="Close"
          >
            <X size={18} />
          </button>
        </div>
      </div>

      {/* Content zone */}
      <div className="pt-3 pb-5 px-4 md:px-5">
        {loading && <ContentSkeleton />}

        {error && (
          <div className="flex flex-col items-center gap-3 py-12">
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

        {entry && <WordOverview entry={entry} hideHeader primaryPosShownInHeader={primaryPos} />}
      </div>

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
