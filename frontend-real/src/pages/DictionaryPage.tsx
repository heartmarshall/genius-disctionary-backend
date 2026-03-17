import { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, BookOpen } from 'lucide-react'
import { Button } from '@/components/ui/button'
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

  const hasActiveFilters = selectedPOS.length > 0 || selectedTopicIds.length > 0 || debouncedSearch !== ''

  const clearAllFilters = () => {
    setSearch('')
    setDebouncedSearch('')
    setSelectedPOS([])
    setSelectedTopicIds([])
    setSortBy('TEXT')
    setSortDir('ASC')
  }

  /**
   * Before changing selectedWordId, snapshot the target element's position
   * relative to the viewport so we can restore it after render.
   */
  /**
   * Close: snapshot the closing card's position → instant fix after render (no jump).
   * Open/switch: just set state, then smooth-scroll to the new card after render.
   */
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
        // Instant correction — keep the collapsed row at same viewport position
        const delta = elRect.top - fix.viewportY
        if (Math.abs(delta) > 1) {
          scroller.scrollTop += delta
        }
      } else {
        // Open or switch — scroll if card top is outside viewport
        // or too close to the bottom (less than 200px of space to see content)
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

  return (
    <div className="min-h-full">
      {/* Hero header */}
      <div className="px-8 md:px-16 pt-12 pb-6 md:pt-20 md:pb-10">
        <h1 className="text-[clamp(3rem,8vw,8rem)] font-medium leading-[0.9] tracking-[-0.04em] text-text-primary">
          Dictionary
        </h1>
        <p className="mt-4 md:mt-6 max-w-2xl text-sm md:text-base text-text-secondary leading-relaxed">
          {t('hero.subtitle', { defaultValue: 'Your personal vocabulary collection. Browse, search, and study the words you\'ve saved.' })}
        </p>
      </div>

      {/* Toolbar */}
      <div className="px-8 md:px-16">
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
      </div>

      {/* Error */}
      {error && (
        <div className="px-8 md:px-16 mt-6">
          <div className="flex flex-col items-center justify-center gap-3 py-12 rounded-xl border border-poppy/20 bg-poppy-light">
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
        </div>
      )}

      {/* Word list */}
      {!error && (
        <div className="px-8 md:px-16 mt-2">
          <WordList
            entries={entries}
            loading={loading}
            selectedWordId={selectedWordId}
            onWordClick={handleWordClick}
            displayOptions={displayOptions}
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
            <div className="flex flex-col items-center justify-center gap-4 py-32">
              <div className="flex items-center justify-center h-20 w-20 rounded-2xl bg-surface-secondary">
                <BookOpen className="h-10 w-10 text-text-tertiary" />
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
            <div className="flex flex-col items-center gap-3 pt-8 pb-16">
              <p className="text-sm text-text-tertiary">
                {t('pagination.showing', { count: entries.length, total: totalCount })}
              </p>
              <Button
                variant="outline"
                onClick={loadMore}
                className="gap-2 rounded-full px-6 border-border-default hover:bg-surface-secondary transition-all duration-200"
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
