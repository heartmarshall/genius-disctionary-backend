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
import type { PartOfSpeech, SortBy, SortDir, WordFilter } from '@/types/dictionary'

export function DictionaryPage() {
  const { t } = useTranslation('dictionary')
  const navigate = useNavigate()
  const { deleteWord } = useDeleteWord()

  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedTopicIds, setSelectedTopicIds] = useState<string[]>([])
  const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech | null>(null)
  const [sortBy, setSortBy] = useState<SortBy>('TEXT')
  const [sortDir, setSortDir] = useState<SortDir>('ASC')
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

  const filter: WordFilter = {
    search: debouncedSearch || undefined,
    topicIDs: selectedTopicIds.length > 0 ? selectedTopicIds : undefined,
    partOfSpeech: selectedPOS ?? undefined,
    sortBy,
    sortDir,
  }

  const { entries, topics, loading, error } = useDictionary(filter)

  const hasActiveFilters = selectedTopicIds.length > 0 || selectedPOS !== null || debouncedSearch !== ''

  const handleSortChange = (newSortBy: SortBy, newSortDir: SortDir) => {
    setSortBy(newSortBy)
    setSortDir(newSortDir)
  }

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold text-foreground">{t('page.title')}</h1>

      <SearchToolbar
        search={search}
        onSearchChange={setSearch}
        selectedTopicIds={selectedTopicIds}
        onTopicIdsChange={setSelectedTopicIds}
        selectedPartOfSpeech={selectedPOS}
        onPartOfSpeechChange={setSelectedPOS}
        sortBy={sortBy}
        sortDir={sortDir}
        onSortChange={handleSortChange}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        topics={topics}
        resultCount={entries.length}
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
                setSelectedTopicIds([])
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
