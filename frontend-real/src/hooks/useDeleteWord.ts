import { useMutation } from '@apollo/client/react'
import type { Reference } from '@apollo/client'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { DELETE_WORD } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

export function useDeleteWord() {
  const { t } = useTranslation('dictionary')
  const [deleteWordMutation] = useMutation(DELETE_WORD)

  const handleDelete = (entry: DictionaryEntry) => {
    let undone = false

    deleteWordMutation({
      variables: { id: entry.id },
      optimisticResponse: { deleteWord: true },
      update(cache) {
        cache.modify({
          fields: {
            dictionary(existingRefs: readonly Reference[] = [], { readField }: { readField: (field: string, ref: Reference) => unknown }) {
              return existingRefs.filter(
                (ref) => readField('id', ref) !== entry.id
              )
            },
          },
        })
      },
    }).catch(() => {
      if (!undone) {
        toast.error(t('error.deleteFailed'))
      }
    })

    toast(t('delete.toast'), {
      action: {
        label: t('delete.undo'),
        onClick: () => {
          undone = true
        },
      },
      duration: 5000,
    })
  }

  return { deleteWord: handleDelete }
}
