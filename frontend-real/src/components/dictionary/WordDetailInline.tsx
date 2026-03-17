import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X, Pencil, Trash2, Volume2 } from 'lucide-react'
import { toast } from 'sonner'
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
import { getPosColors } from '@/lib/pos-colors'

interface WordDetailInlineProps {
  wordId: string
  onClose: () => void
}

function HeaderSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-10 w-48" />
      <Skeleton className="h-5 w-32" />
      <div className="flex gap-2">
        <Skeleton className="h-6 w-28 rounded-full" />
        <Skeleton className="h-6 w-28 rounded-full" />
      </div>
    </div>
  )
}

function ContentSkeleton() {
  return (
    <div className="space-y-3">
      <Skeleton className="h-4 w-full" />
      <Skeleton className="h-4 w-3/4" />
      <Skeleton className="h-4 w-1/2" />
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
      initial={{ height: 0, opacity: 0 }}
      animate={{ height: 'auto', opacity: 1 }}
      exit={{ height: 0, opacity: 0 }}
      transition={{
        height: { type: 'spring', stiffness: 300, damping: 30, mass: 0.8 },
        opacity: { duration: 0.2 },
      }}
      className="overflow-hidden"
    >
      <div className="border-x border-b border-border-default rounded-b-xl bg-bg-card">
        {/* ── Hero header zone — POS-colored background ── */}
        <div
          className="px-8 pt-5 pb-6"
          style={{
            backgroundColor: entry
              ? `color-mix(in srgb, var(${posColors.cssVar}) 22%, white)`
              : 'var(--surface-secondary)',
          }}
        >
          {/* Action bar */}
          <div className="flex items-center justify-between mb-5">
            {entry && (
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditWordId(entry.id)}
                  className="h-7 px-2 text-xs text-text-secondary hover:text-text-primary gap-1"
                >
                  <Pencil size={12} />
                  {t('actions.edit')}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => requestDelete(entry)}
                  className="h-7 px-2 text-xs text-text-tertiary hover:text-poppy gap-1"
                >
                  <Trash2 size={12} />
                  {t('actions.delete')}
                </Button>
              </div>
            )}
            {!entry && <div />}
            <button
              type="button"
              onClick={onClose}
              className="flex items-center justify-center h-7 w-7 rounded-md text-text-tertiary hover:text-text-primary transition-colors"
              aria-label="Close"
            >
              <X size={16} />
            </button>
          </div>

          {loading && <HeaderSkeleton />}

          {entry && (
            <div className="space-y-3">
              {/* Word — hero size */}
              <h2 className="font-orelega text-4xl text-text-primary leading-tight">
                {entry.text}
              </h2>

              {/* IPA pronunciations — inline, no pills */}
              {entry.pronunciations.length > 0 && (
                <div className="flex items-center gap-3 flex-wrap">
                  {entry.pronunciations.map((pron) => (
                    <div key={pron.id} className="inline-flex items-center gap-1.5">
                      <span className="font-serif text-base text-text-secondary">
                        /{pron.transcription}/
                      </span>
                      {pron.region && (
                        <span className="text-[10px] font-medium uppercase tracking-wider text-text-tertiary">
                          {pron.region}
                        </span>
                      )}
                      {pron.audioUrl && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-5 w-5 text-text-tertiary hover:text-text-primary"
                          onClick={() => playAudio(pron.audioUrl!)}
                        >
                          <Volume2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              )}

              {/* Topics */}
              {entry.topics.length > 0 && (
                <div className="flex gap-1.5 flex-wrap pt-1">
                  {entry.topics.map((topic) => (
                    <span
                      key={topic.id}
                      className="text-[10px] uppercase tracking-wider text-text-tertiary border border-border-subtle rounded-full px-2.5 py-0.5"
                    >
                      {topic.name}
                    </span>
                  ))}
                </div>
              )}
            </div>
          )}
        </div>

        {/* ── Content zone — clean white ── */}
        <div className="px-8 py-5">
          {loading && <ContentSkeleton />}

          {error && (
            <div className="flex flex-col items-center gap-3 py-10 rounded-lg border border-poppy/20 bg-poppy-light">
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

          {entry && <WordOverview entry={entry} hideHeader />}
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
      </div>
    </motion.div>
  )
}
