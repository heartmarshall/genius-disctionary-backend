import { useMutation } from '@apollo/client/react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { DELETE_ENTRY, GET_DICTIONARY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

export function useDeleteWord() {
  const { t } = useTranslation('dictionary')
  const [deleteEntryMutation] = useMutation(DELETE_ENTRY, {
    refetchQueries: [{ query: GET_DICTIONARY }],
  })

  const handleDelete = (entry: DictionaryEntry) => {
    deleteEntryMutation({
      variables: { id: entry.id },
    }).catch(() => {
      toast.error(t('error.deleteFailed'))
    })

    toast(t('delete.toast'), {
      duration: 5000,
    })
  }

  return { deleteWord: handleDelete }
}
