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
  const entries = currentData?.dictionary.edges.map((e) => e.node) ?? []
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
      updateQuery: (prev, { fetchMoreResult }) => {
        if (!fetchMoreResult) return prev
        return {
          dictionary: {
            ...fetchMoreResult.dictionary,
            edges: [...prev.dictionary.edges, ...fetchMoreResult.dictionary.edges],
          },
        }
      },
    })
  }

  return {
    entries,
    totalCount,
    pageInfo,
    topics: topicsData?.topics ?? [],
    loading,
    error,
    loadMore,
  }
}
