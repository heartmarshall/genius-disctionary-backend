import { useTranslation } from 'react-i18next'
import { Search, SlidersHorizontal, X, AlignJustify, List } from 'lucide-react'
import type { ViewMode } from '@/components/dictionary/WordFlow'
import type { ListDisplayOptions } from '@/components/dictionary/WordFlow'
import { Skeleton } from '@/components/ui/skeleton'
import { useScrollCompact } from '@/hooks/useScrollCompact'
import { cn } from '@/lib/utils'

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
  listDisplay: ListDisplayOptions
  onListDisplayChange: (options: ListDisplayOptions) => void
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
  listDisplay,
  onListDisplayChange,
}: DictionaryHeroProps) {
  const { t } = useTranslation('dictionary')
  const { isCompact, sentinelRef } = useScrollCompact()

  const isInitialLoad = loading && totalCount === 0

  return (
    <>
      <div ref={sentinelRef} className="h-0" />

      {/* === EXPANDED === */}
      <div className={isCompact ? 'hidden' : 'block'}>
        {/* Title + Stats */}
        <div className="pt-8 md:pt-10">
          {isInitialLoad ? (
            <Skeleton className="h-8 w-44" />
          ) : (
            <h1 className="text-3xl font-bold tracking-tight text-text-primary leading-none">
              {t('page.title')}
            </h1>
          )}
          {isInitialLoad ? (
            <Skeleton className="h-4 w-28 mt-2" />
          ) : (
            <p className="text-sm text-text-tertiary tabular-nums mt-2">
              {totalCount} {t('stats.words')}
              <span className="mx-1.5">&middot;</span>
              {topicsCount} {t('stats.topics')}
            </p>
          )}
        </div>

        {/* Toolbar */}
        <div className={cn('flex items-center gap-2.5 mt-5', viewMode === 'list' ? 'pb-3' : 'pb-5 border-b border-border-subtle')}>
          {/* Search */}
          <div className="relative flex-1 max-w-md">
            <Search
              size={16}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary pointer-events-none"
            />
            <input
              type="text"
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder={t('search.placeholder')}
              className="w-full h-10 pl-9 pr-4 rounded-lg bg-surface-secondary text-text-primary placeholder:text-text-tertiary text-sm border border-border-subtle focus:outline-none focus:border-text-tertiary transition-colors duration-150"
            />
          </div>

          {/* View toggle */}
          <div className="flex shrink-0 rounded-lg bg-surface-secondary border border-border-subtle p-0.5">
            <button
              type="button"
              onClick={() => onViewModeChange('flow')}
              className={`rounded-md flex items-center justify-center w-9 h-9 transition-all duration-150 ${
                viewMode === 'flow'
                  ? 'bg-bg-page text-text-primary shadow-sm'
                  : 'text-text-tertiary hover:text-text-secondary'
              }`}
              aria-label="Flow view"
            >
              <AlignJustify size={15} />
            </button>
            <button
              type="button"
              onClick={() => onViewModeChange('list')}
              className={`rounded-md flex items-center justify-center w-9 h-9 transition-all duration-150 ${
                viewMode === 'list'
                  ? 'bg-bg-page text-text-primary shadow-sm'
                  : 'text-text-tertiary hover:text-text-secondary'
              }`}
              aria-label="List view"
            >
              <List size={15} />
            </button>
          </div>

          {/* Filter button */}
          <div className="relative shrink-0">
            <button
              type="button"
              onClick={onFiltersToggle}
              className="h-10 px-3 rounded-lg flex items-center gap-1.5 text-sm text-text-secondary hover:text-text-primary hover:bg-surface-secondary transition-colors duration-150"
              aria-label={t('filter.title')}
            >
              <SlidersHorizontal size={15} />
              <span className="hidden sm:inline">{t('filter.title')}</span>
            </button>
            {hasActiveFilters && (
              <button
                type="button"
                onClick={onClearFilters}
                className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-poppy text-white flex items-center justify-center hover:bg-poppy-hover transition-colors duration-150"
                aria-label={t('filter.clearAll')}
              >
                <X size={10} />
              </button>
            )}
          </div>
        </div>

        {/* Display options — list mode only */}
        {viewMode === 'list' && (
          <div className="flex flex-wrap items-center gap-1.5 pb-5 border-b border-border-subtle">
            <span className="text-xs text-text-tertiary mr-1">{t('display.label')}</span>
            {(Object.keys(listDisplay) as (keyof ListDisplayOptions)[]).map((key) => (
              <button
                key={key}
                type="button"
                onClick={() => onListDisplayChange({ ...listDisplay, [key]: !listDisplay[key] })}
                className={cn(
                  'h-7 px-2.5 rounded-md text-xs transition-colors duration-150',
                  listDisplay[key]
                    ? 'bg-cornflower text-text-on-accent'
                    : 'bg-surface-secondary text-text-secondary hover:text-text-primary border border-border-subtle',
                )}
              >
                {t(`display.${key}`)}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* === COMPACT STICKY === */}
      <div
        className={`sticky top-0 z-[5] bg-white/95 backdrop-blur-sm ${
          isCompact ? 'block' : 'hidden'
        }`}
      >
        <div className={cn('flex items-center gap-2.5 py-2.5', viewMode !== 'list' && 'border-b border-border-subtle')}>
          <h1 className="text-sm font-semibold text-text-primary shrink-0 hidden md:block">
            {t('page.title')}
          </h1>
          <span className="hidden md:block w-px h-4 bg-border-subtle" />

          {/* Search */}
          <div className="relative flex-1 max-w-sm">
            <Search
              size={15}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-text-tertiary pointer-events-none"
            />
            <input
              type="text"
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder={t('search.placeholder')}
              className="w-full h-8 pl-9 pr-4 rounded-lg bg-surface-secondary text-text-primary placeholder:text-text-tertiary text-sm border border-border-subtle focus:outline-none focus:border-text-tertiary transition-colors duration-150"
            />
          </div>

          {/* View toggle */}
          <div className="flex shrink-0 rounded-lg bg-surface-secondary border border-border-subtle p-0.5">
            <button
              type="button"
              onClick={() => onViewModeChange('flow')}
              className={`rounded-md flex items-center justify-center w-7 h-7 transition-all duration-150 ${
                viewMode === 'flow'
                  ? 'bg-bg-page text-text-primary shadow-sm'
                  : 'text-text-tertiary hover:text-text-secondary'
              }`}
              aria-label="Flow view"
            >
              <AlignJustify size={13} />
            </button>
            <button
              type="button"
              onClick={() => onViewModeChange('list')}
              className={`rounded-md flex items-center justify-center w-7 h-7 transition-all duration-150 ${
                viewMode === 'list'
                  ? 'bg-bg-page text-text-primary shadow-sm'
                  : 'text-text-tertiary hover:text-text-secondary'
              }`}
              aria-label="List view"
            >
              <List size={13} />
            </button>
          </div>

          {/* Filter button */}
          <div className="relative shrink-0">
            <button
              type="button"
              onClick={onFiltersToggle}
              className="h-8 px-2.5 rounded-lg flex items-center gap-1.5 text-xs text-text-secondary hover:text-text-primary hover:bg-surface-secondary transition-colors duration-150"
              aria-label={t('filter.title')}
            >
              <SlidersHorizontal size={14} />
              <span className="hidden sm:inline">{t('filter.title')}</span>
            </button>
            {hasActiveFilters && (
              <button
                type="button"
                onClick={onClearFilters}
                className="absolute -top-1 -right-1 w-4 h-4 rounded-full bg-poppy text-white flex items-center justify-center hover:bg-poppy-hover transition-colors duration-150"
                aria-label={t('filter.clearAll')}
              >
                <X size={10} />
              </button>
            )}
          </div>
        </div>

        {/* Display options — compact, list mode only */}
        {viewMode === 'list' && (
          <div className="flex flex-wrap items-center gap-1 pb-2 border-b border-border-subtle">
            <span className="text-[11px] text-text-tertiary mr-0.5">{t('display.label')}</span>
            {(Object.keys(listDisplay) as (keyof ListDisplayOptions)[]).map((key) => (
              <button
                key={key}
                type="button"
                onClick={() => onListDisplayChange({ ...listDisplay, [key]: !listDisplay[key] })}
                className={cn(
                  'h-6 px-2 rounded text-[11px] transition-colors duration-150',
                  listDisplay[key]
                    ? 'bg-cornflower text-text-on-accent'
                    : 'bg-surface-secondary text-text-secondary hover:text-text-primary border border-border-subtle',
                )}
              >
                {t(`display.${key}`)}
              </button>
            ))}
          </div>
        )}
      </div>
    </>
  )
}
