import { useState, useEffect, useCallback, useRef, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { DictionaryToolbar } from '@/components/dictionary/DictionaryToolbar'
import { WordList } from '@/components/dictionary/WordList'
import { WordDetailInline } from '@/components/dictionary/WordDetailInline'
import { useDictionary } from '@/hooks/useDictionary'
import type { PartOfSpeech, EntrySortField, SortDirection, DictionaryFilterInput } from '@/types/dictionary'
import { DEFAULT_DISPLAY } from '@/components/dictionary/DictionaryToolbar'
import type { DisplayOptions } from '@/components/dictionary/DictionaryToolbar'

/**
 * Find the scrollable ancestor (overflow-y: auto/scroll) for a given element.
 */
function getScrollParent(el: Element): Element {
  let parent = el.parentElement
  while (parent) {
    const style = getComputedStyle(parent)
    if (style.overflowY === 'auto' || style.overflowY === 'scroll') return parent
    parent = parent.parentElement
  }
  return document.documentElement
}

export function DictionaryPage() {
  const { t } = useTranslation('dictionary')

  // Filter state
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [selectedPOS, setSelectedPOS] = useState<PartOfSpeech[]>([])
  const [selectedTopicIds, setSelectedTopicIds] = useState<string[]>([])
  const [sortBy, setSortBy] = useState<EntrySortField>('TEXT')
  const [sortDir, setSortDir] = useState<SortDirection>('ASC')
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(() => {
    try {
      const saved = localStorage.getItem('dict-display')
      if (saved) return { ...DEFAULT_DISPLAY, ...JSON.parse(saved) }
    } catch {}
    return DEFAULT_DISPLAY
  })

  const updateDisplayOptions = (opts: DisplayOptions) => {
    setDisplayOptions(opts)
    try { localStorage.setItem('dict-display', JSON.stringify(opts)) } catch {}
  }

  // Detail view state
  const [selectedWordId, setSelectedWordId] = useState<string | null>(null)

  // Scroll stabilization
  const pendingScrollFix = useRef<{
    mode: 'close' | 'open-fresh' | 'switch'
    wordId: string
    viewportY: number
  } | null>(null)

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

  // Separate unfiltered query to count POS across the whole dictionary
  const baseFilter: DictionaryFilterInput = {
    search: debouncedSearch || undefined,
    topicId: selectedTopicIds[0] ?? undefined,
    sortField: sortBy,
    sortDirection: sortDir,
  }
  const { entries: allEntries } = useDictionary(baseFilter)

  const posCounts = useMemo(() => {
    const counts: Partial<Record<PartOfSpeech, number>> = {}
    for (const entry of allEntries) {
      const pos = entry.senses[0]?.partOfSpeech
      if (pos) counts[pos] = (counts[pos] ?? 0) + 1
    }
    return counts
  }, [allEntries])

  const hasActiveFilters = selectedPOS.length > 0 || selectedTopicIds.length > 0 || debouncedSearch !== ''

  const clearAllFilters = () => {
    setSearch('')
    setDebouncedSearch('')
    setSelectedPOS([])
    setSelectedTopicIds([])
    setSortBy('TEXT')
    setSortDir('ASC')
  }

  const closeWord = useCallback((closingId: string) => {
    const el = document.querySelector(`[data-word-id="${closingId}"]`)
    if (el) {
      pendingScrollFix.current = {
        mode: 'close',
        wordId: closingId,
        viewportY: el.getBoundingClientRect().top,
      }
    }
    setSelectedWordId(null)
  }, [])

  const openWord = useCallback((id: string, switching: boolean) => {
    pendingScrollFix.current = {
      mode: switching ? 'switch' : 'open-fresh',
      wordId: id,
      viewportY: 0,
    }
    setSelectedWordId(id)
  }, [])

  // After render: stabilize scroll or smooth-scroll to card
  useEffect(() => {
    const fix = pendingScrollFix.current
    if (!fix) return
    pendingScrollFix.current = null

    requestAnimationFrame(() => {
      const el = document.querySelector(`[data-word-id="${fix.wordId}"]`)
      if (!el) return

      const scroller = getScrollParent(el) as HTMLElement
      const scrollerRect = scroller.getBoundingClientRect()
      const elRect = el.getBoundingClientRect()
      const margin = 20
      const targetScrollTop = scroller.scrollTop + (elRect.top - scrollerRect.top) - margin

      if (fix.mode === 'close') {
        const delta = elRect.top - fix.viewportY
        if (Math.abs(delta) > 1) {
          scroller.scrollTop += delta
        }
      } else {
        const isAboveViewport = elRect.top < scrollerRect.top + margin
        const isBelowViewport = elRect.top > scrollerRect.bottom - margin
        const notEnoughRoom = elRect.top > scrollerRect.bottom - 200
        if (isAboveViewport || isBelowViewport || notEnoughRoom) {
          const behavior = fix.mode === 'switch' ? 'instant' as const : 'smooth' as const
          scroller.scrollTo({ top: targetScrollTop, behavior })
        }
      }
    })
  }, [selectedWordId])

  const handleWordClick = (id: string) => {
    if (selectedWordId === id) {
      closeWord(id)
      window.history.pushState(null, '', '/dictionary')
      return
    }

    openWord(id, selectedWordId !== null)
    window.history.pushState(null, '', `/dictionary/${id}`)
  }

  const handleClose = useCallback(() => {
    if (selectedWordId) {
      closeWord(selectedWordId)
    }
    window.history.pushState(null, '', '/dictionary')
  }, [selectedWordId, closeWord])

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

  // Keyboard navigation
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
      openWord(nextEntry.id, true)
      window.history.replaceState(null, '', `/dictionary/${nextEntry.id}`)
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [selectedWordId, entries, handleClose, openWord])

  // Stats — count from filtered entries
  const filteredPosCounts = useMemo(() => {
    const counts: Partial<Record<PartOfSpeech, number>> = {}
    for (const entry of entries) {
      const pos = entry.senses[0]?.partOfSpeech
      if (pos) counts[pos] = (counts[pos] ?? 0) + 1
    }
    return counts
  }, [entries])
  const nounCount = filteredPosCounts['NOUN'] ?? 0
  const adjCount = filteredPosCounts['ADJECTIVE'] ?? 0
  const uniqueTopicsCount = topics.length

  return (
    <div className="min-h-full bg-bg-page">
      <div className="max-w-[960px] mx-auto px-5 md:px-12 pt-10">
      {/* Page header */}
      <div className="pb-0">
        <div className="mb-6">
          <h1 className="text-[30px] font-bold text-text-primary font-serif tracking-tight mb-2.5">
            Dictionary
          </h1>
          <div className="flex items-center gap-3.5 text-[14px] text-text-tertiary">
            <span>
              <strong className="text-text-secondary font-semibold">{totalCount}</strong> words
            </span>
            <span className="w-[3px] h-[3px] rounded-full bg-text-disabled" />
            <span>
              <strong className="text-text-secondary font-semibold">{nounCount}</strong> nouns
            </span>
            <span>
              <strong className="text-text-secondary font-semibold">{adjCount}</strong> adj
            </span>
            <span className="w-[3px] h-[3px] rounded-full bg-text-disabled" />
            <span>{uniqueTopicsCount} topics</span>
            {loading && (
              <span className="animate-pulse text-text-disabled">loading...</span>
            )}
          </div>
        </div>
      </div>

      {/* Toolbar */}
      <div className="mt-3">
        <DictionaryToolbar
          totalCount={totalCount}
          topicsCount={topics.length}
          loading={loading}
          search={search}
          onSearchChange={setSearch}
          selectedPOS={selectedPOS}
          onPOSChange={setSelectedPOS}
          posCounts={posCounts}
          sortBy={sortBy}
          sortDir={sortDir}
          onSortChange={(field, dir) => { setSortBy(field); setSortDir(dir) }}
          topics={topics}
          selectedTopicIds={selectedTopicIds}
          onTopicIdsChange={setSelectedTopicIds}
          displayOptions={displayOptions}
          onDisplayOptionsChange={updateDisplayOptions}
        />
      </div>

      {/* Error */}
      {error && (
        <div className="mt-4">
          <div className="py-4 border-t border-border-subtle">
            <p className="text-sm text-poppy">
              {t('error.loadFailed')}
            </p>
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="mt-2 text-sm text-cornflower hover:underline"
            >
              {t('error.tryAgain')}
            </button>
          </div>
        </div>
      )}

      {/* Word list */}
      {!error && (
        <div className="mt-1">
          <WordList
            entries={entries}
            loading={loading}
            selectedWordId={selectedWordId}
            onWordClick={handleWordClick}
            displayOptions={displayOptions}
            sortedAlphabetically={sortBy === 'TEXT' && sortDir === 'ASC'}
            renderDetail={(entry, index) => (
              <WordDetailInline
                key={entry.id}
                wordId={entry.id}
                index={index}
                onClose={handleClose}
              />
            )}
          />

          {/* Empty state */}
          {!loading && entries.length === 0 && (
            <div className="py-16 text-center">
              <p className="text-sm text-text-tertiary">
                {hasActiveFilters ? 'No words match your current filters' : 'Your dictionary is empty'}
              </p>
              {hasActiveFilters && (
                <button
                  type="button"
                  onClick={clearAllFilters}
                  className="mt-3 text-sm text-cornflower hover:underline"
                >
                  Clear all filters
                </button>
              )}
            </div>
          )}

          {/* Load more */}
          {!loading && entries.length > 0 && pageInfo?.hasNextPage && (
            <div className="py-4 mt-3 border-t border-border-subtle">
              <div className="flex items-center gap-3">
                <div className="flex-1 dict-progress-track">
                  <div
                    className="dict-progress-fill"
                    style={{ width: `${Math.round((entries.length / totalCount) * 100)}%` }}
                  />
                </div>
                <span className="text-xs tabular-nums text-text-disabled shrink-0">
                  {entries.length} of {totalCount}
                </span>
              </div>
              <button
                type="button"
                onClick={loadMore}
                className="mt-2 w-full py-2 rounded-lg text-sm font-medium text-cornflower hover:bg-cornflower-light transition-colors duration-150"
              >
                Load more
              </button>
            </div>
          )}

          {/* EOF */}
          {!loading && entries.length > 0 && !pageInfo?.hasNextPage && (
            <div className="py-4 mt-3 border-t border-border-subtle">
              <span className="text-xs text-text-disabled">
                {totalCount} entries total
              </span>
            </div>
          )}

          <div className="h-4" />
        </div>
      )}
      </div>
    </div>
  )
}
