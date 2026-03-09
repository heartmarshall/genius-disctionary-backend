import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft, Pencil, Trash2 } from 'lucide-react'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useDeleteWord } from '@/hooks/useDeleteWord'
import { WordOverview } from '@/components/dictionary/WordOverview'
import { WordEditDialog } from '@/components/dictionary/WordEditDialog'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

export function DictionaryEntryPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(id)
  const { deleteWord } = useDeleteWord()
  const [editOpen, setEditOpen] = useState(false)

  const handleDelete = () => {
    if (!entry) return
    deleteWord(entry)
    navigate('/dictionary')
  }

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center justify-between">
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5"
          onClick={() => navigate('/dictionary')}
        >
          <ArrowLeft className="h-4 w-4" />
          {t('entry.backToDict')}
        </Button>

        {entry && (
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              className="gap-1.5"
              onClick={() => setEditOpen(true)}
            >
              <Pencil className="h-4 w-4" />
              {t('sheet.edit')}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              className="gap-1.5 text-poppy hover:text-poppy"
              onClick={handleDelete}
            >
              <Trash2 className="h-4 w-4" />
              {t('sheet.delete')}
            </Button>
          </div>
        )}
      </div>

      {loading && (
        <div className="space-y-4">
          <Skeleton className="h-12 w-48" />
          <Skeleton className="h-5 w-32" />
          <Skeleton className="h-4 w-full" />
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-24 w-full" />
        </div>
      )}

      {error && (
        <div className="flex flex-col items-center justify-center gap-3 py-12">
          <p className="text-sm text-muted-foreground">{t('error.loadFailed')}</p>
          <Button variant="outline" size="sm" onClick={() => window.location.reload()}>
            {t('error.tryAgain')}
          </Button>
        </div>
      )}

      {entry && <WordOverview entry={entry} />}

      <WordEditDialog
        wordId={editOpen ? id ?? null : null}
        open={editOpen}
        onOpenChange={setEditOpen}
      />
    </div>
  )
}
