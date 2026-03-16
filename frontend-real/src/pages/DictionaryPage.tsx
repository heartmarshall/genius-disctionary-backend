import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { motion, AnimatePresence } from 'framer-motion'
import { ChevronDown, BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { DictionaryHero } from '@/components/dictionary/DictionaryHero'
import { FilterPanel } from '@/components/dictionary/FilterPanel'
import { WordFlow } from '@/components/dictionary/WordFlow'
import { DictionaryDetailView } from '@/components/dictionary/DictionaryDetailView'
import { useDictionary } from '@/hooks/useDictionary'
import type { ViewMode, ListDisplayOptions } from '@/components/dictionary/WordFlow'
import { DEFAULT_LIST_DISPLAY } from '@/components/dictionary/WordFlow'
import type { PartOfSpeech, EntrySortField, SortDirection, DictionaryFilterInput } from '@/types/dictionary'

export function DictionaryPage() {
  const { t } = useTranslation('dictionary')

  // Filter state
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech[]>([])
  const [selectedTopicIds, setSelectedTopicIds] = useState<string[]>([])
  const [sortBy, setSortBy] = useState<EntrySortField>('TEXT')
  const [sortDir, setSortDir] = useState<SortDirection>('ASC')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [viewMode, setViewMode] = useState<ViewMode>('flow')
  const [listDisplay, setListDisplay] = useState<ListDisplayOptions>(DEFAULT_LIST_DISPLAY)

  // Detail view state
  const [selectedWordId, setSelectedWordId] = useState<string | null>(null)

  // Debounce search
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 500)
    return () => clearTimeout(timer)
  }, [search])

  const filter: DictionaryFilterInput = {
    search: debouncedSearch || undefined,
    topicId: selectedTopicIds[0] ?? undefined,
    partOfSpeech: selectedPOS[0] ?? undefined,
    sortField: sortBy,
    sortDirection: sortDir,
  }

  const { entries, totalCount, pageInfo, topics, loading, error, loadMore } = useDictionary(filter)

  const hasActiveFilters = selectedPOS.length > 0 || selectedTopicIds.length > 0 || debouncedSearch !== ''

  const clearAllFilters = () => {
    setSearch('')
    setSelectedPOS([])
    setSelectedTopicIds([])
    setSortBy('TEXT')
    setSortDir('ASC')
  }

  const handleWordClick = (id: string) => {
    setSelectedWordId(id)
    window.history.pushState(null, '', `/dictionary/${id}`)
  }

  const handleBack = () => {
    setSelectedWordId(null)
    window.history.pushState(null, '', '/dictionary')
  }

  // Handle browser back/forward
  useEffect(() => {
    const handlePopState = () => {
      const match = window.location.pathname.match(/^\/dictionary\/(.+)$/)
      setSelectedWordId(match ? match[1] : null)
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  // Handle direct URL access to /dictionary/:id
  useEffect(() => {
    const match = window.location.pathname.match(/^\/dictionary\/(.+)$/)
    if (match) setSelectedWordId(match[1])
  }, [])

  return (
    <div className="relative">
      <AnimatePresence mode="sync">
        {/* Dictionary view */}
        <motion.div
          key="dictionary"
          animate={
            selectedWordId
              ? { x: '-20%', scale: 0.95, opacity: 0.3, filter: 'blur(2px)' }
              : { x: '0%', scale: 1, opacity: 1 }
          }
          transition={{ duration: 0.4, ease: [0.4, 0, 0.2, 1] }}
          className="min-h-[80vh]"
          style={{ pointerEvents: selectedWordId ? 'none' : 'auto' }}
        >
          {/* Hero (title + stats + search + filters) */}
          <DictionaryHero
            totalCount={totalCount}
            topicsCount={topics.length}
            loading={loading}
            search={search}
            onSearchChange={setSearch}
            filtersOpen={filtersOpen}
            onFiltersToggle={() => setFiltersOpen(!filtersOpen)}
            hasActiveFilters={hasActiveFilters}
            onClearFilters={clearAllFilters}
            viewMode={viewMode}
            onViewModeChange={setViewMode}
            listDisplay={listDisplay}
            onListDisplayChange={setListDisplay}
          />

          {/* Filter panel */}
          <FilterPanel
            open={filtersOpen}
            onClose={() => setFiltersOpen(false)}
            selectedPOS={selectedPOS}
            onPOSChange={setSelectedPOS}
            selectedTopicIds={selectedTopicIds}
            onTopicIdsChange={setSelectedTopicIds}
            sortBy={sortBy}
            sortDir={sortDir}
            onSortChange={(field, dir) => { setSortBy(field); setSortDir(dir) }}
            topics={topics}
          />

          {/* Error */}
          {error && (
            <div className="flex flex-col items-center justify-center gap-3 py-12 mt-6 rounded-xl border border-poppy/20 bg-poppy-light">
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

          {/* Word flow */}
          {!error && (
            <div className="mt-6">
              <WordFlow
                entries={entries}
                loading={loading}
                onWordClick={handleWordClick}
                viewMode={viewMode}
                listDisplay={listDisplay}
              />
            </div>
          )}

          {/* Empty state */}
          {!error && !loading && entries.length === 0 && (
            <div className="flex flex-col items-center justify-center gap-4 py-20">
              <div className="flex items-center justify-center h-20 w-20 rounded-2xl bg-cornflower-light">
                <BookOpen className="h-10 w-10 text-cornflower" />
              </div>
              <p className="text-lg text-text-secondary">
                {hasActiveFilters ? t('empty.noResults') : t('empty.title')}
              </p>
              {hasActiveFilters && (
                <Button
                  variant="outline"
                  size="sm"
                  className="rounded-full"
                  onClick={() => {
                    setSearch('')
                    setSelectedPOS([])
                    setSelectedTopicIds([])
                  }}
                >
                  {t('filter.clearAll')}
                </Button>
              )}
            </div>
          )}

          {/* Load more */}
          {!error && !loading && entries.length > 0 && pageInfo?.hasNextPage && (
            <div className="flex flex-col items-center gap-3 pt-2 pb-4">
              <p className="text-sm text-text-tertiary">
                {t('pagination.showing', { count: entries.length, total: totalCount })}
              </p>
              <Button
                variant="outline"
                onClick={loadMore}
                className="gap-2 rounded-full px-6 border-border-subtle hover:bg-surface-secondary transition-all duration-200"
              >
                <ChevronDown className="h-4 w-4" />
                {t('pagination.loadMore')}
              </Button>
            </div>
          )}
        </motion.div>

        {/* Detail view — slides in from right */}
        {selectedWordId && (
          <motion.div
            key="detail"
            className="absolute inset-0 bg-bg-page overflow-y-auto"
            initial={{ x: '100%' }}
            animate={{ x: '0%' }}
            exit={{ x: '100%' }}
            transition={{ duration: 0.4, ease: [0.4, 0, 0.2, 1] }}
          >
            <DictionaryDetailView
              wordId={selectedWordId}
              onBack={handleBack}
            />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  )
}
