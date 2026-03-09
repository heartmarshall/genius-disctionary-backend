import { useMutation } from '@apollo/client/react'
import { UPDATE_ENTRY_NOTES, GET_DICTIONARY } from '@/graphql/queries/dictionary'

export interface UpdateEntryNotesInput {
  entryId: string
  notes?: string
}

export function useWordEdit() {
  const [mutate, { loading }] = useMutation(UPDATE_ENTRY_NOTES, {
    refetchQueries: [{ query: GET_DICTIONARY }],
  })

  const updateNotes = async (entryId: string, notes?: string) => {
    await mutate({ variables: { input: { entryId, notes } } })
  }

  return { updateNotes, loading }
}
