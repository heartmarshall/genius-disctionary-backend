import { useTranslation } from 'react-i18next'
import { Search, SlidersHorizontal, X, AlignJustify, List } from 'lucide-react'
import type { ViewMode } from '@/components/dictionary/WordFlow'
import { Skeleton } from '@/components/ui/skeleton'
import { useScrollCompact } from '@/hooks/useScrollCompact'

interface DictionaryHeroProps {
  totalCount: number
  topicsCount: number
  loading: boolean
  search: string
  onSearchChange: (value: string) => void
  filtersOpen: boolean
  onFiltersToggle: () => void
  hasActiveFilters: boolean
  onClearFilters: () => void
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
}

export function DictionaryHero({
  totalCount,
  topicsCount,
  loading,
  search,
  onSearchChange,
  onFiltersToggle,
  hasActiveFilters,
  onClearFilters,
  viewMode,
  onViewModeChange,
}: DictionaryHeroProps) {
  const { t } = useTranslation('dictionary')
  const { isCompact, sentinelRef } = useScrollCompact()

  const isInitialLoad = loading && totalCount === 0

  const searchInput = (compact: boolean) => (
    <div className={`relative ${compact ? 'flex-1 max-w-sm' : 'flex-1 max-w-sm'}`}>
      <Search
        size={compact ? 15 : 16}
        className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary pointer-events-none"
      />
      <input
        type="text"
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        placeholder={t('search.placeholder')}
        className={`w-full pl-9 pr-4 rounded-lg bg-surface-disabled text-text-primary placeholder:text-text-tertiary border-0 focus:outline-none focus:bg-border-subtle ${
          compact ? 'h-8 text-sm' : 'h-9 text-sm'
        }`}
      />
    </div>
  )

  const viewToggle = (compact: boolean) => (
    <div className="flex shrink-0 rounded-lg bg-surface-secondary p-0.5">
      <button
        type="button"
        onClick={() => onViewModeChange('flow')}
        className={`rounded-md flex items-center justify-center ${
          compact ? 'w-7 h-7' : 'w-8 h-8'
        } ${viewMode === 'flow' ? 'bg-bg-page text-text-primary' : 'text-text-tertiary hover:text-text-secondary'}`}
        style={{ transition: 'color 150ms, background-color 150ms' }}
        aria-label="Flow view"
      >
        <AlignJustify size={compact ? 13 : 14} />
      </button>
      <button
        type="button"
        onClick={() => onViewModeChange('list')}
        className={`rounded-md flex items-center justify-center ${
          compact ? 'w-7 h-7' : 'w-8 h-8'
        } ${viewMode === 'list' ? 'bg-bg-page text-text-primary' : 'text-text-tertiary hover:text-text-secondary'}`}
        style={{ transition: 'color 150ms, background-color 150ms' }}
        aria-label="List view"
      >
        <List size={compact ? 13 : 14} />
      </button>
    </div>
  )

  const filterButton = (compact: boolean) => (
    <div className="relative shrink-0 pt-1 pr-1">
      <button
        type="button"
        onClick={onFiltersToggle}
        className={`rounded-lg flex items-center gap-1.5 text-text-secondary hover:text-text-primary hover:bg-surface-secondary ${
          compact ? 'h-8 px-2.5 text-xs' : 'h-9 px-3 text-sm'
        }`}
        style={{ transition: 'color 150ms, background-color 150ms' }}
        aria-label={t('filter.title')}
      >
        <SlidersHorizontal size={compact ? 14 : 15} />
        <span className="hidden sm:inline">{t('filter.title')}</span>
      </button>
      {hasActiveFilters && (
        <button
          type="button"
          onClick={onClearFilters}
          className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-poppy text-white flex items-center justify-center hover:bg-poppy-hover"
          style={{ transition: 'background-color 150ms' }}
          aria-label={t('filter.clearAll')}
        >
          <X size={10} />
        </button>
      )}
    </div>
  )

  return (
    <>
      <div ref={sentinelRef} className="h-0" />

      {/* === EXPANDED === */}
      <div className={isCompact ? 'hidden' : 'block'}>
        {/* Page title */}
        <div className="pt-8 md:pt-12 pb-6 md:pb-8">
          {isInitialLoad ? (
            <Skeleton className="h-10 w-48" />
          ) : (
            <h1 className="text-4xl md:text-5xl font-bold tracking-tight text-text-primary leading-none">
              {t('page.title')}
            </h1>
          )}
        </div>

        {/* Toolbar */}
        <div className="flex items-center gap-2 pb-4 border-b border-border-subtle mb-1">
          {/* Stats */}
          {isInitialLoad ? (
            <Skeleton className="h-5 w-20" />
          ) : (
            <span className="text-xs text-text-tertiary tabular-nums shrink-0 mr-1 min-w-28">
              {totalCount} {t('stats.words')}
              <span className="mx-1.5 text-border-default">&middot;</span>
              {topicsCount} {t('stats.topics')}
            </span>
          )}

          <span className="w-px h-4 bg-border-subtle mx-1" />

          {searchInput(false)}
          {viewToggle(false)}
          {filterButton(false)}
        </div>
      </div>

      {/* === COMPACT STICKY === */}
      <div
        className={`sticky top-0 z-[5] bg-white/95 backdrop-blur-sm ${
          isCompact ? 'block' : 'hidden'
        }`}
      >
        <div className="flex items-center gap-3 py-2.5 border-b border-border-subtle">
          <h1 className="text-sm font-semibold text-text-primary shrink-0 hidden md:block">
            {t('page.title')}
          </h1>
          <span className="hidden md:block w-px h-4 bg-border-subtle" />
          {searchInput(true)}
          {viewToggle(true)}
          {filterButton(true)}
        </div>
      </div>
    </>
  )
}
