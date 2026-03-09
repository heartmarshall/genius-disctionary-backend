import { useTranslation } from 'react-i18next'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import type { DictionaryEntry } from '@/types/dictionary'

interface WordCardGridProps {
  entries: DictionaryEntry[]
  loading: boolean
  onWordClick: (id: string) => void
}

function WordCardSkeleton() {
  return (
    <Card className="p-4 space-y-3">
      <Skeleton className="h-6 w-32" />
      <Skeleton className="h-4 w-20" />
      <div className="flex gap-2">
        <Skeleton className="h-5 w-16" />
        <Skeleton className="h-5 w-24" />
      </div>
      <Skeleton className="h-4 w-full" />
      <div className="flex gap-1.5">
        <Skeleton className="h-5 w-14" />
        <Skeleton className="h-5 w-14" />
      </div>
    </Card>
  )
}

export function WordCardGrid({ entries, loading, onWordClick }: WordCardGridProps) {
  const { t } = useTranslation('dictionary')

  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => (
          <WordCardSkeleton key={i} />
        ))}
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
      {entries.map((entry) => {
        const firstSense = entry.senses[0]
        const firstPronunciation = entry.pronunciations[0]
        const translations = firstSense?.translations.map((tr) => tr.text).join(', ')

        return (
          <Card
            key={entry.id}
            className="border border-border hover:border-poppy transition-colors duration-150 cursor-pointer"
            onClick={() => onWordClick(entry.id)}
          >
            <CardHeader className="pb-2">
              <h3 className="font-orelega text-xl text-foreground">{entry.text}</h3>
              {firstPronunciation && (
                <span className="font-serif text-sm text-muted-foreground">
                  /{firstPronunciation.transcription}/
                </span>
              )}
            </CardHeader>
            <CardContent className="space-y-2">
              {firstSense && (
                <div className="flex items-center gap-2 flex-wrap">
                  {firstSense.partOfSpeech && (
                    <Badge variant="secondary">
                      {t(`pos.${firstSense.partOfSpeech}`)}
                    </Badge>
                  )}
                  {translations && (
                    <span className="text-sm text-muted-foreground truncate">
                      {translations}
                    </span>
                  )}
                </div>
              )}
              {entry.topics.length > 0 && (
                <div className="flex gap-1.5 flex-wrap">
                  {entry.topics.slice(0, 3).map((topic) => (
                    <Badge key={topic.id} variant="outline" className="text-xs">
                      {topic.name}
                    </Badge>
                  ))}
                  {entry.topics.length > 3 && (
                    <Badge variant="outline" className="text-xs">
                      +{entry.topics.length - 3}
                    </Badge>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}
