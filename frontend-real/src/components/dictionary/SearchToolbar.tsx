import { useTranslation } from 'react-i18next'
import { Search, LayoutGrid, List, ArrowUpDown, Filter, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from '@/components/ui/dropdown-menu'
import type { PartOfSpeech, EntrySortField, SortDirection, Topic, UUID } from '@/types/dictionary'

type ViewMode = 'grid' | 'table'

const PART_OF_SPEECH_VALUES: PartOfSpeech[] = [
  'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
  'PREPOSITION', 'CONJUNCTION', 'INTERJECTION', 'PHRASE', 'IDIOM', 'OTHER',
]

interface SearchToolbarProps {
  search: string
  onSearchChange: (value: string) => void
  selectedTopicId: UUID | null
  onTopicIdChange: (id: UUID | null) => void
  selectedPartOfSpeech: PartOfSpeech | null
  onPartOfSpeechChange: (pos: PartOfSpeech | null) => void
  sortBy: EntrySortField
  sortDir: SortDirection
  onSortChange: (sortBy: EntrySortField, sortDir: SortDirection) => void
  viewMode: ViewMode
  onViewModeChange: (mode: ViewMode) => void
  topics: Topic[]
  resultCount: number
}

export function SearchToolbar({
  search,
  onSearchChange,
  selectedTopicId,
  onTopicIdChange,
  selectedPartOfSpeech,
  onPartOfSpeechChange,
  sortBy,
  sortDir,
  onSortChange,
  viewMode,
  onViewModeChange,
  topics,
  resultCount,
}: SearchToolbarProps) {
  const { t } = useTranslation('dictionary')

  const hasActiveFilters = selectedTopicId !== null || selectedPartOfSpeech !== null

  const handleTopicSelect = (topicId: string) => {
    onTopicIdChange(topicId === '' ? null : topicId)
  }

  const handlePosSelect = (pos: string) => {
    onPartOfSpeechChange(pos === '' ? null : (pos as PartOfSpeech))
  }

  const selectedTopicName = topics.find((tp) => tp.id === selectedTopicId)?.name

  return (
    <div className="flex items-center gap-3 flex-wrap">
      <div className="relative flex-1 min-w-48">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
        <Input
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder={t('search.placeholder')}
          className="pl-9"
        />
      </div>

      {/* Topics filter */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-1.5">
            <Filter className="h-4 w-4" />
            {t('filter.topics')}
            {selectedTopicName && (
              <Badge variant="secondary" className="ml-1 px-1.5 py-0 text-xs">
                {selectedTopicName}
              </Badge>
            )}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuRadioGroup
            value={selectedTopicId ?? ''}
            onValueChange={handleTopicSelect}
          >
            <DropdownMenuRadioItem value="">
              —
            </DropdownMenuRadioItem>
            {topics.map((topic) => (
              <DropdownMenuRadioItem key={topic.id} value={topic.id}>
                {topic.name}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Part of speech filter */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-1.5">
            {t('filter.partOfSpeech')}
            {selectedPartOfSpeech && (
              <Badge variant="secondary" className="ml-1 px-1.5 py-0 text-xs">
                {t(`pos.${selectedPartOfSpeech}`)}
              </Badge>
            )}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuRadioGroup
            value={selectedPartOfSpeech ?? ''}
            onValueChange={handlePosSelect}
          >
            <DropdownMenuRadioItem value="">
              —
            </DropdownMenuRadioItem>
            {PART_OF_SPEECH_VALUES.map((pos) => (
              <DropdownMenuRadioItem key={pos} value={pos}>
                {t(`pos.${pos}`)}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Sort */}
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="outline" size="sm" className="gap-1.5">
            <ArrowUpDown className="h-4 w-4" />
            {sortBy === 'TEXT' && t('sort.alphabetical')}
            {sortBy === 'CREATED_AT' && t('sort.created')}
            {sortBy === 'UPDATED_AT' && t('sort.updated')}
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start">
          <DropdownMenuLabel>{t('sort.alphabetical')}</DropdownMenuLabel>
          <DropdownMenuRadioGroup
            value={sortBy}
            onValueChange={(v) => onSortChange(v as EntrySortField, sortDir)}
          >
            <DropdownMenuRadioItem value="TEXT">
              {t('sort.alphabetical')}
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="CREATED_AT">
              {t('sort.created')}
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="UPDATED_AT">
              {t('sort.updated')}
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup
            value={sortDir}
            onValueChange={(v) => onSortChange(sortBy, v as SortDirection)}
          >
            <DropdownMenuRadioItem value="ASC">
              {t('sort.asc')}
            </DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="DESC">
              {t('sort.desc')}
            </DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Clear filters */}
      {hasActiveFilters && (
        <Button
          variant="ghost"
          size="sm"
          className="gap-1.5 text-muted-foreground"
          onClick={() => {
            onTopicIdChange(null)
            onPartOfSpeechChange(null)
          }}
        >
          <X className="h-4 w-4" />
          {t('filter.clearAll')}
        </Button>
      )}

      {/* View toggle */}
      <div className="flex items-center gap-1 ml-auto">
        <span className="text-sm text-muted-foreground mr-2">
          {t('count.words', { count: resultCount })}
        </span>
        <Button
          variant="ghost"
          size="icon"
          className={viewMode === 'grid' ? 'text-poppy bg-poppy/10' : ''}
          onClick={() => onViewModeChange('grid')}
          aria-label={t('view.grid')}
        >
          <LayoutGrid className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className={viewMode === 'table' ? 'text-poppy bg-poppy/10' : ''}
          onClick={() => onViewModeChange('table')}
          aria-label={t('view.table')}
        >
          <List className="h-4 w-4" />
        </Button>
      </div>
    </div>
  )
}
