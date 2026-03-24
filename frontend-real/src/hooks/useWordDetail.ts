import { useQuery } from '@apollo/client/react'
import { GET_DICTIONARY_ENTRY, GET_CARD_STATS, GET_CARD_HISTORY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry, CardStats, ReviewLog } from '@/types/dictionary'

interface DictionaryEntryData {
  dictionaryEntry: DictionaryEntry
}

interface CardStatsData {
  cardStats: CardStats
}

interface CardHistoryData {
  cardHistory: {
    logs: ReviewLog[]
    totalCount: number
  }
}

export function useWordDetail(id: string | undefined) {
  const { data, loading, error } = useQuery<DictionaryEntryData>(GET_DICTIONARY_ENTRY, {
    variables: { id },
    skip: !id,
  })

  const cardId = data?.dictionaryEntry?.card?.id

  const { data: statsData } = useQuery<CardStatsData>(GET_CARD_STATS, {
    variables: { cardId },
    skip: !cardId,
  })

  const { data: historyData } = useQuery<CardHistoryData>(GET_CARD_HISTORY, {
    variables: { input: { cardId, limit: 20 } },
    skip: !cardId,
  })

  return {
    entry: data?.dictionaryEntry,
    cardStats: statsData?.cardStats,
    reviewLogs: historyData?.cardHistory?.logs,
    loading,
    error,
  }
}
