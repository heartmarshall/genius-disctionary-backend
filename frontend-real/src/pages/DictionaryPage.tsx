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
  const [displayOptions, setDisplayOptions] = useState<DisplayOptions>(DEFAULT_DISPLAY)

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

  // Build status line parts
  const filterDesc = hasActiveFilters
    ? `${entries.length} matches`
    : `${totalCount} entries`
  const sortDesc = sortBy === 'TEXT' ? 'a→z' : sortBy === 'CREATED_AT' ? 'newest' : 'updated'

  return (
    <div className="terminal-theme min-h-full font-mono" style={{ background: 'var(--term-bg)', color: 'var(--term-text)' }}>
      {/* Status line */}
      <div className="px-5 md:px-8 pt-5 pb-0">
        <div className="flex items-baseline gap-2 flex-wrap text-sm">
          <span className="font-bold" style={{ color: 'var(--term-green)' }}>dict://</span>
          <span style={{ color: 'var(--term-text-dim)' }}>//</span>
          <span className="tabular-nums" style={{ color: 'var(--term-text-muted)' }}>{filterDesc}</span>
          <span style={{ color: 'var(--term-text-dim)' }}>&middot;</span>
          <span style={{ color: 'var(--term-text-muted)' }}>{topics.length} topics</span>
          <span style={{ color: 'var(--term-text-dim)' }}>&middot;</span>
          <span style={{ color: 'var(--term-text-muted)' }}>sort:{sortDesc}</span>
          {selectedPOS.length > 0 && (
            <>
              <span style={{ color: 'var(--term-text-dim)' }}>&middot;</span>
              <span style={{ color: 'var(--term-blue)' }}>pos:{selectedPOS.map(p => t(`pos.${p}`).toLowerCase()).join(',')}</span>
            </>
          )}
          {loading && (
            <span className="animate-pulse" style={{ color: 'var(--term-text-dim)' }}>loading...</span>
          )}
        </div>
      </div>

      {/* Toolbar */}
      <div className="px-5 md:px-8 mt-2">
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
          onDisplayOptionsChange={setDisplayOptions}
        />
      </div>

      {/* Error */}
      {error && (
        <div className="px-5 md:px-8 mt-4">
          <div className="py-4" style={{ borderTop: '1px dashed var(--term-border)' }}>
            <p className="text-sm" style={{ color: 'var(--term-red)' }}>
              stderr: {t('error.loadFailed')}
            </p>
            <button
              type="button"
              onClick={() => window.location.reload()}
              className="mt-2 text-sm hover:underline"
              style={{ color: 'var(--term-yellow)' }}
            >
              [retry]
            </button>
          </div>
        </div>
      )}

      {/* Word list */}
      {!error && (
        <div className="px-5 md:px-8 mt-1">
          <WordList
            entries={entries}
            loading={loading}
            selectedWordId={selectedWordId}
            onWordClick={handleWordClick}
            displayOptions={displayOptions}
            sortedAlphabetically={sortBy === 'TEXT' && sortDir === 'ASC'}
            renderDetail={(entry) => (
              <WordDetailInline
                key={entry.id}
                wordId={entry.id}
                onClose={handleClose}
              />
            )}
          />

          {/* Empty state */}
          {!loading && entries.length === 0 && (
            <div className="py-12">
              <p className="text-sm" style={{ color: 'var(--term-text-muted)' }}>
                {hasActiveFilters ? '// 0 results — no entries match current filters' : '// dictionary is empty'}
              </p>
              {hasActiveFilters && (
                <button
                  type="button"
                  onClick={clearAllFilters}
                  className="mt-2 text-sm hover:underline"
                  style={{ color: 'var(--term-yellow)' }}
                >
                  [clear filters]
                </button>
              )}
            </div>
          )}

          {/* Load more */}
          {!loading && entries.length > 0 && pageInfo?.hasNextPage && (
            <div className="flex items-center gap-3 py-4 mt-2" style={{ borderTop: '1px dashed var(--term-border)' }}>
              <span className="text-xs tabular-nums" style={{ color: 'var(--term-text-dim)' }}>
                loaded {entries.length}/{totalCount}
              </span>
              <div className="flex-1 term-progress-track">
                <div
                  className="term-progress-fill"
                  style={{ width: `${Math.round((entries.length / totalCount) * 100)}%` }}
                />
              </div>
              <span className="text-xs tabular-nums" style={{ color: 'var(--term-text-dim)' }}>
                {Math.round((entries.length / totalCount) * 100)}%
              </span>
              <button
                type="button"
                onClick={loadMore}
                className="text-sm hover:underline"
                style={{ color: 'var(--term-green)' }}
              >
                [more]
              </button>
            </div>
          )}

          {/* EOF marker */}
          {!loading && entries.length > 0 && !pageInfo?.hasNextPage && (
            <div className="py-4 mt-2" style={{ borderTop: '1px dashed var(--term-border)' }}>
              <span className="text-xs" style={{ color: 'var(--term-text-dim)' }}>
                // EOF — {totalCount} entries total
              </span>
            </div>
          )}

          <div className="h-4" />
        </div>
      )}
    </div>
  )
}
