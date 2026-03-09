import { useQuery } from '@apollo/client/react'
import { GET_DICTIONARY, GET_TOPICS } from '@/graphql/queries/dictionary'
import type { DictionaryConnection, Topic, DictionaryFilterInput } from '@/types/dictionary'

interface DictionaryData {
  dictionary: DictionaryConnection
}

interface TopicsData {
  topics: Topic[]
}

export function useDictionary(filter: DictionaryFilterInput) {
  const { data, loading, error, refetch } = useQuery<DictionaryData>(GET_DICTIONARY, {
    variables: { input: filter },
  })

  const { data: topicsData } = useQuery<TopicsData>(GET_TOPICS)

  return {
    entries: data?.dictionary.edges.map((e) => e.node) ?? [],
    totalCount: data?.dictionary.totalCount ?? 0,
    pageInfo: data?.dictionary.pageInfo,
    topics: topicsData?.topics ?? [],
    loading,
    error,
    refetch,
  }
}
