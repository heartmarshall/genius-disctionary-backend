import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus } from 'lucide-react'
import { DictionaryToolbar } from '@/components/dictionary/DictionaryToolbar'
import { WordList } from '@/components/dictionary/WordList'
import { WordDetailInline } from '@/components/dictionary/WordDetailInline'
import { AddWordDialog } from '@/components/dictionary/AddWordDialog'
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

  // Add word dialog
  const [addWordOpen, setAddWordOpen] = useState(false)

  // Detail view state
  const [selectedWordId, setSelectedWordId] = useState<string | null>(null)
  const scrollRef = useRef<HTMLDivElement>(null)

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

  const closeWord = useCallback((_closingId: string) => {
    setSelectedWordId(null)
  }, [])

  const openWord = useCallback((id: string, _switching: boolean) => {
    setSelectedWordId(id)
  }, [])

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
      <div className="max-w-[960px] mx-auto px-5 md:px-10 pt-12">
      {/* Page header */}
      <div className="flex items-baseline justify-between mb-5">
        <div className="flex items-baseline gap-4">
          <h1 className="text-[34px] font-serif font-bold text-text-primary tracking-tight leading-none">
            Dictionary
          </h1>
          <div className="flex items-center gap-2 text-[13px] text-text-tertiary">
            <span><strong className="text-text-secondary font-semibold">{totalCount}</strong> words</span>
            <span className="w-[3px] h-[3px] rounded-full bg-text-disabled" />
            <span><strong className="text-text-secondary font-semibold">{nounCount}</strong> nouns</span>
            <span><strong className="text-text-secondary font-semibold">{adjCount}</strong> adj</span>
            <span className="w-[3px] h-[3px] rounded-full bg-text-disabled" />
            <span>{uniqueTopicsCount} topics</span>
            {loading && <span className="animate-pulse text-text-disabled">loading...</span>}
          </div>
        </div>
        <button
          type="button"
          onClick={() => setAddWordOpen(true)}
          className="inline-flex items-center gap-1.5 px-4 py-2 rounded-[10px] text-[13px] font-semibold bg-text-primary text-bg-page hover:opacity-85 transition-opacity duration-150"
        >
          <Plus size={15} />
          Add word
        </button>
      </div>

      {/* Toolbar + Alphabet — sticky on scroll */}
      <div className="sticky top-0 z-10 bg-bg-page pb-2">
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

        {/* Alphabet bar */}
        {sortBy === 'TEXT' && sortDir === 'ASC' && entries.length > 0 && (() => {
          const usedLetters = new Set(entries.map(e => e.text[0]?.toUpperCase()).filter(Boolean))
          const allLetters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ'.split('')
          return (
            <div className="flex gap-1 mt-3 flex-wrap">
              {allLetters.map(l => {
                const has = usedLetters.has(l)
                return (
                  <span
                    key={l}
                    role={has ? 'button' : undefined}
                    onClick={has ? () => {
                      const el = scrollRef.current?.querySelector(`[data-letter="${l}"]`)
                      el?.scrollIntoView({ behavior: 'smooth', block: 'start' })
                    } : undefined}
                    className={`font-mono text-[11px] font-semibold w-[28px] h-[28px] rounded-full flex items-center justify-center transition-all duration-150 ${
                      has ? 'hover:scale-125 hover:text-text-primary active:scale-110' : ''
                    }`}
                    style={{
                      color: has ? 'var(--text-secondary)' : 'var(--text-disabled)',
                      opacity: has ? 1 : 0.2,
                      backgroundColor: has ? 'var(--island)' : 'transparent',
                      boxShadow: has ? 'var(--island-shadow)' : 'none',
                      cursor: has ? 'pointer' : 'default',
                    }}
                  >
                    {l}
                  </span>
                )
              })}
            </div>
          )
        })()}
      </div>

      {/* Word list — contained scroll area */}
      <div className="mt-6 mx-[-24px] relative">
        {/* Fade edges */}
        <div className="pointer-events-none absolute inset-x-0 top-0 h-6 z-10" style={{ background: 'linear-gradient(to bottom, var(--bg-page), transparent)' }} />
        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-6 z-10" style={{ background: 'linear-gradient(to top, var(--bg-page), transparent)' }} />
        <div ref={scrollRef} className="overflow-y-auto dict-scroll-subtle" style={{ maxHeight: 'calc(100vh - 280px)' }}>
          <div className="px-[24px] py-4">
          {/* Error */}
          {error && (
            <div className="p-5">
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
          )}

          {/* Word list */}
          {!error && (
            <>
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
                <div className="py-4 px-4 border-t border-border-subtle">
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
                <div className="py-3 px-4 border-t border-border-subtle">
                  <span className="text-xs text-text-disabled">
                    {totalCount} entries total
                  </span>
                </div>
              )}
            </>
          )}
          </div>
        </div>
      </div>

      <div className="h-8" />
      </div>

      <AddWordDialog open={addWordOpen} onOpenChange={setAddWordOpen} />
    </div>
  )
}
