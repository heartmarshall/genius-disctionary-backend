import { useQuery } from '@apollo/client/react'
import { GET_DICTIONARY_ENTRY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

interface DictionaryEntryData {
  dictionaryEntry: DictionaryEntry
}

export function useWordDetail(id: string | undefined) {
  const { data, loading, error } = useQuery<DictionaryEntryData>(GET_DICTIONARY_ENTRY, {
    variables: { id },
    skip: !id,
  })

  return {
    entry: data?.dictionaryEntry,
    loading,
    error,
  }
}
