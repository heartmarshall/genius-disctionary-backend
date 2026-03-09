import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { useWordDetail } from '@/hooks/useWordDetail'
import { useWordEdit } from '@/hooks/useWordEdit'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordEditDialogProps {
  wordId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WordEditDialog({ wordId, open, onOpenChange }: WordEditDialogProps) {
  const { entry } = useWordDetail(wordId ?? undefined)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        {entry && (
          <WordEditForm
            key={entry.id + entry.updatedAt}
            entry={entry}
            wordId={wordId!}
            onClose={() => onOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}

function WordEditForm({
  entry,
  wordId,
  onClose,
}: {
  entry: DictionaryEntry
  wordId: string
  onClose: () => void
}) {
  const { t } = useTranslation('dictionary')
  const { updateNotes, loading: saving } = useWordEdit()

  const [notes, setNotes] = useState(entry.notes ?? '')

  const handleSave = async () => {
    try {
      await updateNotes(wordId, notes || undefined)
      onClose()
    } catch {
      toast.error(t('error.saveFailed'))
    }
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>{t('edit.title')}: {entry.text}</DialogTitle>
        <DialogDescription className="sr-only">
          {t('edit.title')}
        </DialogDescription>
      </DialogHeader>

      <div className="space-y-4">
        <div className="space-y-2">
          <label className="text-sm font-medium">{t('edit.notes')}</label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={5}
            className="flex w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          />
        </div>
      </div>

      <DialogFooter>
        <Button variant="outline" onClick={onClose}>
          {t('edit.cancel')}
        </Button>
        <Button onClick={handleSave} disabled={saving}>
          {saving ? t('edit.saving') : t('edit.save')}
        </Button>
      </DialogFooter>
    </>
  )
}
