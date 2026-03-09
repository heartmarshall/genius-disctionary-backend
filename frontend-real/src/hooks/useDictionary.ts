import { useQuery } from '@apollo/client'
import { GET_DICTIONARY, GET_TOPICS } from '@/graphql/queries/dictionary'
import type { DictionaryEntry, Topic, WordFilter } from '@/types/dictionary'

interface DictionaryData {
  dictionary: DictionaryEntry[]
}

interface TopicsData {
  topics: Topic[]
}

export function useDictionary(filter: WordFilter) {
  const { data, loading, error, refetch } = useQuery<DictionaryData>(GET_DICTIONARY, {
    variables: { filter },
  })

  const { data: topicsData } = useQuery<TopicsData>(GET_TOPICS)

  return {
    entries: data?.dictionary ?? [],
    topics: topicsData?.topics ?? [],
    loading,
    error,
    refetch,
  }
}
