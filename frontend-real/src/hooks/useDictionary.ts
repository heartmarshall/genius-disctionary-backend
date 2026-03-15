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
  const { data, loading, error, fetchMore } = useQuery<DictionaryData>(GET_DICTIONARY, {
    variables: { input: filter },
  })

  const { data: topicsData } = useQuery<TopicsData>(GET_TOPICS)

  const entries = data?.dictionary.edges.map((e) => e.node) ?? []
  const pageInfo = data?.dictionary.pageInfo
  const totalCount = data?.dictionary.totalCount ?? 0

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
