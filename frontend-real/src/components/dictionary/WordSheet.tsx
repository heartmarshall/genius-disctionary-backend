import { useTranslation } from 'react-i18next'
import { ExternalLink, Pencil, Trash2 } from 'lucide-react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useWordDetail } from '@/hooks/useWordDetail'
import { WordOverview } from '@/components/dictionary/WordOverview'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordSheetProps {
  wordId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
  onOpenFullPage: (id: string) => void
  onEdit: (id: string) => void
  onDelete: (entry: DictionaryEntry) => void
}

export function WordSheet({
  wordId,
  open,
  onOpenChange,
  onOpenFullPage,
  onEdit,
  onDelete,
}: WordSheetProps) {
  const { t } = useTranslation('dictionary')
  const { entry, loading, error } = useWordDetail(wordId ?? undefined)

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="w-full sm:max-w-lg overflow-y-auto">
        {loading && (
          <div className="space-y-4 pt-6">
            <Skeleton className="h-10 w-48" />
            <Skeleton className="h-5 w-32" />
            <Skeleton className="h-4 w-full" />
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-20 w-full" />
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

        {entry && (
          <>
            <SheetHeader>
              <SheetTitle className="font-orelega text-2xl">{entry.text}</SheetTitle>
            </SheetHeader>

            <WordOverview entry={entry} className="mt-4" />

            <div className="flex items-center gap-2 mt-6 pt-4 border-t">
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => onOpenFullPage(entry.id)}
              >
                <ExternalLink className="h-4 w-4" />
                {t('sheet.openFull')}
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="gap-1.5"
                onClick={() => onEdit(entry.id)}
              >
                <Pencil className="h-4 w-4" />
                {t('sheet.edit')}
              </Button>
              <Button
                variant="ghost"
                size="sm"
                className="gap-1.5 text-poppy hover:text-poppy"
                onClick={() => onDelete(entry)}
              >
                <Trash2 className="h-4 w-4" />
                {t('sheet.delete')}
              </Button>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  )
}
