import { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { AnimatePresence } from 'framer-motion'
import { ChevronDown, BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { DictionaryToolbar } from '@/components/dictionary/DictionaryToolbar'
import { WordList } from '@/components/dictionary/WordList'
import { WordDetailInline } from '@/components/dictionary/WordDetailInline'
import { useDictionary } from '@/hooks/useDictionary'
import type { PartOfSpeech, EntrySortField, SortDirection, DictionaryFilterInput } from '@/types/dictionary'
import { DEFAULT_DISPLAY } from '@/components/dictionary/DictionaryToolbar'
import type { DisplayOptions } from '@/components/dictionary/DictionaryToolbar'

export function DictionaryPage() {
  const { t } = useTranslation('dictionary')

  // Filter state
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech[]>([])
  const [selectedTopicIds, setSelectedTopicIds] = useState<string[]>([])
  const [sortBy, setSortBy] = useState<EntrySortField>('TEXT')
  const [sortDir, setSortDir] = useState<SortDirection>('ASC')
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(DEFAULT_DISPLAY)

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
    setDebouncedSearch('')
    setSelectedPOS([])
    setSelectedTopicIds([])
    setSortBy('TEXT')
    setSortDir('ASC')
  }

  const handleWordClick = (id: string) => {
    // Toggle: click again to close
    if (selectedWordId === id) {
      setSelectedWordId(null)
      window.history.pushState(null, '', '/dictionary')
      return
    }
    setSelectedWordId(id)
    window.history.pushState(null, '', `/dictionary/${id}`)

    // Auto-scroll to word
    requestAnimationFrame(() => {
      const el = document.querySelector(`[data-word-id="${id}"]`)
      el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    })
  }

  const handleClose = useCallback(() => {
    setSelectedWordId(null)
    window.history.pushState(null, '', '/dictionary')
  }, [])

  // Handle browser back/forward
  useEffect(() => {
    const handlePopState = () => {
      const match = window.location.pathname.match(/^\/dictionary\/(.+)$/)
      setSelectedWordId(match ? match[1] : null)
    }
    window.addEventListener('popstate', handlePopState)
    return () => window.removeEventListener('popstate', handlePopState)
  }, [])

  // Handle direct URL access
  useEffect(() => {
    const match = window.location.pathname.match(/^\/dictionary\/(.+)$/)
    if (match) setSelectedWordId(match[1])
  }, [])

  // Keyboard navigation: arrow up/down when detail is open
  useEffect(() => {
    if (!selectedWordId) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleClose()
        return
      }
      if (e.key !== 'ArrowUp' && e.key !== 'ArrowDown') return

      e.preventDefault()
      const currentIndex = entries.findIndex((entry) => entry.id === selectedWordId)
      if (currentIndex === -1) return

      const nextIndex = e.key === 'ArrowDown'
        ? Math.min(currentIndex + 1, entries.length - 1)
        : Math.max(currentIndex - 1, 0)

      if (nextIndex === currentIndex) return

      const nextEntry = entries[nextIndex]
      setSelectedWordId(nextEntry.id)
      window.history.replaceState(null, '', `/dictionary/${nextEntry.id}`)

      requestAnimationFrame(() => {
        const el = document.querySelector(`[data-word-id="${nextEntry.id}"]`)
        el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
      })
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedWordId, entries, handleClose])

  return (
    <div>
      {/* Toolbar */}
      <DictionaryToolbar
        totalCount={totalCount}
        topicsCount={topics.length}
        loading={loading}
        search={search}
        onSearchChange={setSearch}
        selectedPOS={selectedPOS}
        onPOSChange={setSelectedPOS}
        sortBy={sortBy}
        sortDir={sortDir}
        onSortChange={(field, dir) => { setSortBy(field); setSortDir(dir) }}
        topics={topics}
        selectedTopicIds={selectedTopicIds}
        onTopicIdsChange={setSelectedTopicIds}
        displayOptions={displayOptions}
        onDisplayOptionsChange={setDisplayOptions}
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

      {/* Word list */}
      {!error && (
        <div className="max-w-[900px] mx-auto px-6 mt-4">
          <WordList
            entries={entries}
            loading={loading}
            selectedWordId={selectedWordId}
            onWordClick={handleWordClick}
            displayOptions={displayOptions}
            renderDetail={(entry) => (
              <AnimatePresence>
                {selectedWordId === entry.id && (
                  <WordDetailInline
                    key={entry.id}
                    wordId={entry.id}
                    onClose={handleClose}
                  />
                )}
              </AnimatePresence>
            )}
          />

          {/* Empty state */}
          {!loading && entries.length === 0 && (
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
                  onClick={clearAllFilters}
                >
                  {t('filter.clearAll')}
                </Button>
              )}
            </div>
          )}

          {/* Load more */}
          {!loading && entries.length > 0 && pageInfo?.hasNextPage && (
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
        </div>
      )}
    </div>
  )
}
