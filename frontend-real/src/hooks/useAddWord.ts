import { useState, useEffect, useCallback } from 'react'
import { useLazyQuery, useMutation } from '@apollo/client/react'
import {
  SEARCH_CATALOG,
  PREVIEW_REF_ENTRY,
  CREATE_ENTRY_FROM_CATALOG,
} from '@/graphql/queries/dictionary'
import type { RefEntry } from '@/types/dictionary'

export function useCatalogSearch() {
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 300)
    return () => clearTimeout(timer)
  }, [query])

  const [search, { data, loading }] = useLazyQuery(SEARCH_CATALOG)

  useEffect(() => {
    if (debouncedQuery.trim().length >= 2) {
      search({ variables: { query: debouncedQuery.trim(), limit: 10 } })
    }
  }, [debouncedQuery, search])

  const results: RefEntry[] = data?.searchCatalog ?? []

  return { query, setQuery, results, loading }
}

export function useRefEntryPreview() {
  const [fetch, { data, loading }] = useLazyQuery(PREVIEW_REF_ENTRY, {
    fetchPolicy: 'network-only',
  })

  const preview = useCallback((text: string) => {
    fetch({ variables: { text } })
  }, [fetch])

  const entry: RefEntry | null = data?.previewRefEntry ?? null

  return { preview, entry, loading }
}

export function useAddFromCatalog() {
  const [createEntry, { loading }] = useMutation(CREATE_ENTRY_FROM_CATALOG, {
    refetchQueries: ['GetDictionary'],
  })

  const addFromCatalog = async (refEntryId: string, senseIds: string[], notes?: string) => {
    const result = await createEntry({
      variables: {
        input: {
          refEntryId,
          senseIds,
          notes: notes || undefined,
          createCard: true,
        },
      },
    })
    return result.data?.createEntryFromCatalog?.entry
  }

  return { addFromCatalog, loading }
}
