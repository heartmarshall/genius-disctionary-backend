import { useTranslation } from 'react-i18next'
import { ArrowUp, ArrowDown, MoreHorizontal, ExternalLink, Pencil, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu'
import type { DictionaryEntry, EntrySortField, SortDirection } from '@/types/dictionary'

interface WordTableProps {
  entries: DictionaryEntry[]
  loading: boolean
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (sortBy: EntrySortField, sortDir: SortDirection) => void
  onWordClick: (id: string) => void
  onEditClick: (id: string) => void
  onDeleteClick: (entry: DictionaryEntry) => void
}

function SortIcon({ active, dir }: { active: boolean; dir: SortDirection }) {
  if (!active) return null
  return dir === 'ASC'
    ? <ArrowUp className="h-3.5 w-3.5 inline ml-1" />
    : <ArrowDown className="h-3.5 w-3.5 inline ml-1" />
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 8 }).map((_, i) => (
        <tr key={i} className="border-b">
          <td className="p-3 sticky left-0 bg-card z-[1]"><Skeleton className="h-4 w-24" /></td>
          <td className="p-3"><Skeleton className="h-4 w-32" /></td>
          <td className="p-3"><Skeleton className="h-4 w-40" /></td>
          <td className="p-3"><Skeleton className="h-5 w-16" /></td>
          <td className="p-3"><Skeleton className="h-5 w-20" /></td>
          <td className="p-3"><Skeleton className="h-4 w-20" /></td>
          <td className="p-3"><Skeleton className="h-4 w-8" /></td>
        </tr>
      ))}
    </>
  )
}

export function WordTable({
  entries,
  loading,
  sortBy,
  sortDir,
  onSortChange,
  onWordClick,
  onEditClick,
  onDeleteClick,
}: WordTableProps) {
  const { t } = useTranslation('dictionary')

  const toggleSort = (column: EntrySortField) => {
    if (sortBy === column) {
      onSortChange(column, sortDir === 'ASC' ? 'DESC' : 'ASC')
    } else {
      onSortChange(column, 'ASC')
    }
  }

  return (
    <div className="overflow-x-auto border border-border rounded-md">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th
              className="p-3 text-left font-medium sticky left-0 bg-muted/50 z-[1] cursor-pointer select-none"
              onClick={() => toggleSort('TEXT')}
            >
              {t('table.word')}
              <SortIcon active={sortBy === 'TEXT'} dir={sortDir} />
            </th>
            <th className="p-3 text-left font-medium">{t('table.translations')}</th>
            <th className="p-3 text-left font-medium">{t('table.definition')}</th>
            <th className="p-3 text-left font-medium">{t('table.partOfSpeech')}</th>
            <th className="p-3 text-left font-medium">{t('table.topics')}</th>
            <th
              className="p-3 text-left font-medium cursor-pointer select-none"
              onClick={() => toggleSort('UPDATED_AT')}
            >
              {t('table.updated')}
              <SortIcon active={sortBy === 'UPDATED_AT'} dir={sortDir} />
            </th>
            <th className="p-3 w-10" />
          </tr>
        </thead>
        <tbody>
          {loading ? (
            <SkeletonRows />
          ) : (
            entries.map((entry) => {
              const firstSense = entry.senses[0]
              const translations = firstSense?.translations.map((tr) => tr.text).join(', ')

              return (
                <tr
                  key={entry.id}
                  className="border-b hover:bg-muted/30 cursor-pointer transition-colors"
                  onClick={() => onWordClick(entry.id)}
                >
                  <td className="p-3 font-medium sticky left-0 bg-card z-[1]">
                    {entry.text}
                  </td>
                  <td className="p-3 text-muted-foreground truncate max-w-48">
                    {translations}
                  </td>
                  <td className="p-3 text-muted-foreground truncate max-w-64">
                    {firstSense?.definition}
                  </td>
                  <td className="p-3">
                    {firstSense?.partOfSpeech && (
                      <Badge variant="secondary">
                        {t(`pos.${firstSense.partOfSpeech}`)}
                      </Badge>
                    )}
                  </td>
                  <td className="p-3">
                    <div className="flex gap-1 flex-wrap">
                      {entry.topics.slice(0, 2).map((topic) => (
                        <Badge key={topic.id} variant="outline" className="text-xs">
                          {topic.name}
                        </Badge>
                      ))}
                      {entry.topics.length > 2 && (
                        <Badge variant="outline" className="text-xs">
                          +{entry.topics.length - 2}
                        </Badge>
                      )}
                    </div>
                  </td>
                  <td className="p-3 text-muted-foreground whitespace-nowrap">
                    {new Date(entry.updatedAt).toLocaleDateString()}
                  </td>
                  <td className="p-3">
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild onClick={(e) => e.stopPropagation()}>
                        <Button variant="ghost" size="icon" className="h-8 w-8">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onWordClick(entry.id) }}>
                          <ExternalLink className="h-4 w-4 mr-2" />
                          {t('actions.open')}
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onEditClick(entry.id) }}>
                          <Pencil className="h-4 w-4 mr-2" />
                          {t('actions.edit')}
                        </DropdownMenuItem>
                        <DropdownMenuItem
                          className="text-poppy focus:text-poppy"
                          onClick={(e) => { e.stopPropagation(); onDeleteClick(entry) }}
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          {t('actions.delete')}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </td>
                </tr>
              )
            })
          )}
        </tbody>
      </table>
    </div>
  )
}
