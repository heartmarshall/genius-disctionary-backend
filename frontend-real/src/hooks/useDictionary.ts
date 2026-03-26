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
  const { data, previousData, loading, error, fetchMore } = useQuery<DictionaryData>(GET_DICTIONARY, {
    variables: { input: filter },
  })

  const { data: topicsData } = useQuery<TopicsData>(GET_TOPICS)

  // Use previous data while loading to prevent UI flicker
  const currentData = data ?? previousData
  const rawEntries = currentData?.dictionary.edges.map((e) => e.node) ?? []
  // Deduplicate — Apollo cache merges + loadMore can produce duplicate edges
  const seen = new Set<string>()
  const entries = rawEntries.filter((e) => {
    if (seen.has(e.id)) return false
    seen.add(e.id)
    return true
  })
  const pageInfo = currentData?.dictionary.pageInfo
  const totalCount = currentData?.dictionary.totalCount ?? 0

  const loadMore = () => {
    if (!pageInfo?.hasNextPage || !pageInfo.endCursor) return

    fetchMore({
      variables: {
        input: {
          ...filter,
          after: pageInfo.endCursor,
        },
      },
    })
  }

  // Only show loading skeleton on initial load, not during fetchMore
  const initialLoading = loading && !previousData

  return {
    entries,
    totalCount,
    pageInfo,
    topics: topicsData?.topics ?? [],
    loading: initialLoading,
    error,
    loadMore,
  }
}
