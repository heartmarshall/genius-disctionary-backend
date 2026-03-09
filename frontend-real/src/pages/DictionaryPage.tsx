import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { SearchToolbar } from '@/components/dictionary/SearchToolbar'
import { WordCardGrid } from '@/components/dictionary/WordCardGrid'
import { WordTable } from '@/components/dictionary/WordTable'
import { WordSheet } from '@/components/dictionary/WordSheet'
import { WordEditDialog } from '@/components/dictionary/WordEditDialog'
import { useDictionary } from '@/hooks/useDictionary'
import { useDeleteWord } from '@/hooks/useDeleteWord'
import type { PartOfSpeech, EntrySortField, SortDirection, DictionaryFilterInput } from '@/types/dictionary'

export function DictionaryPage() {
  const { t } = useTranslation('dictionary')
  const navigate = useNavigate()
  const { deleteWord } = useDeleteWord()

  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedTopicId, setSelectedTopicId] = useState<string | null>(null)
  const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech | null>(null)
  const [sortBy, setSortBy] = useState<EntrySortField>('TEXT')
  const [sortDir, setSortDir] = useState<SortDirection>('ASC')
  const [viewMode, setViewMode] = useState<'grid' | 'table'>(() =>
    (localStorage.getItem('dictionary-view') as 'grid' | 'table') || 'grid'
  )
  const [selectedWordId, setSelectedWordId] = useState<string | null>(null)
  const [editWordId, setEditWordId] = useState<string | null>(null)

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 500)
    return () => clearTimeout(timer)
  }, [search])

  // Persist view mode
  useEffect(() => {
    localStorage.setItem('dictionary-view', viewMode)
  }, [viewMode])

  const filter: DictionaryFilterInput = {
    search: debouncedSearch || undefined,
    topicId: selectedTopicId ?? undefined,
    partOfSpeech: selectedPOS ?? undefined,
    sortField: sortBy,
    sortDirection: sortDir,
  }

  const { entries, totalCount, topics, loading, error } = useDictionary(filter)

  const hasActiveFilters = selectedTopicId !== null || selectedPOS !== null || debouncedSearch !== ''

  const handleSortChange = (newSortBy: EntrySortField, newSortDir: SortDirection) => {
    setSortBy(newSortBy)
    setSortDir(newSortDir)
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold text-foreground">{t('page.title')}</h1>

      <SearchToolbar
        search={search}
        onSearchChange={setSearch}
        selectedTopicId={selectedTopicId}
        onTopicIdChange={setSelectedTopicId}
        selectedPartOfSpeech={selectedPOS}
        onPartOfSpeechChange={setSelectedPOS}
        sortBy={sortBy}
        sortDir={sortDir}
        onSortChange={handleSortChange}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        topics={topics}
        resultCount={totalCount}
      />

      {error && (
        <div className="flex flex-col items-center justify-center gap-3 py-12">
          <p className="text-sm text-muted-foreground">{t('error.loadFailed')}</p>
          <Button variant="outline" size="sm" onClick={() => window.location.reload()}>
            {t('error.tryAgain')}
          </Button>
        </div>
      )}

      {!error && viewMode === 'grid' && (
        <WordCardGrid
          entries={entries}
          loading={loading}
          onWordClick={setSelectedWordId}
        />
      )}

      {!error && viewMode === 'table' && (
        <WordTable
          entries={entries}
          loading={loading}
          sortBy={sortBy}
          sortDir={sortDir}
          onSortChange={handleSortChange}
          onWordClick={setSelectedWordId}
          onEditClick={(id) => setEditWordId(id)}
          onDeleteClick={deleteWord}
        />
      )}

      {!error && !loading && entries.length === 0 && (
        <div className="flex flex-col items-center justify-center gap-3 py-16">
          <BookOpen className="h-12 w-12 text-muted-foreground/50" />
          <p className="text-lg text-muted-foreground">
            {hasActiveFilters ? t('empty.noResults') : t('empty.title')}
          </p>
          {hasActiveFilters && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                setSearch('')
                setSelectedTopicId(null)
                setSelectedPOS(null)
              }}
            >
              {t('filter.clearAll')}
            </Button>
          )}
        </div>
      )}

      <WordSheet
        wordId={selectedWordId}
        open={selectedWordId !== null}
        onOpenChange={(open) => !open && setSelectedWordId(null)}
        onOpenFullPage={(id) => navigate(`/dictionary/${id}`)}
        onEdit={(id) => { setSelectedWordId(null); setEditWordId(id) }}
        onDelete={(entry) => { setSelectedWordId(null); deleteWord(entry) }}
      />

      <WordEditDialog
        wordId={editWordId}
        open={editWordId !== null}
        onOpenChange={(open) => !open && setEditWordId(null)}
      />
    </div>
  )
}
