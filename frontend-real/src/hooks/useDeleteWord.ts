import { useCallback, useRef, useState } from 'react'
import { useMutation } from '@apollo/client/react'
import { toast } from 'sonner'
import { useTranslation } from 'react-i18next'
import { DELETE_ENTRY } from '@/graphql/queries/dictionary'
import type { DictionaryEntry } from '@/types/dictionary'

const UNDO_DELAY_MS = 5000

export function useDeleteWord() {
  const { t } = useTranslation('dictionary')
  const [pendingEntry, setPendingEntry] = useState<DictionaryEntry | null>(null)
  const [deletedIds, setDeletedIds] = useState<Set<string>>(new Set())
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  const [deleteEntryMutation] = useMutation(DELETE_ENTRY, {
    refetchQueries: ['GetDictionary'],
  })

  const requestDelete = (entry: DictionaryEntry) => {
    setPendingEntry(entry)
  }

  const confirmDelete = useCallback(() => {
    if (!pendingEntry) return
    const entry = pendingEntry
    setPendingEntry(null)

    // Optimistically hide the word
    setDeletedIds((prev) => new Set(prev).add(entry.id))

    // Schedule the actual deletion
    timerRef.current = setTimeout(() => {
      deleteEntryMutation({ variables: { id: entry.id } })
        .catch(() => {
          toast.error(t('error.deleteFailed'))
          // Restore on failure
          setDeletedIds((prev) => {
            const next = new Set(prev)
            next.delete(entry.id)
            return next
          })
        })
        .finally(() => {
          setDeletedIds((prev) => {
            const next = new Set(prev)
            next.delete(entry.id)
            return next
          })
        })
    }, UNDO_DELAY_MS)

    // Show toast with undo
    toast(t('delete.toast'), {
      duration: UNDO_DELAY_MS,
      action: {
        label: t('delete.undo'),
        onClick: () => {
          clearTimeout(timerRef.current)
          setDeletedIds((prev) => {
            const next = new Set(prev)
            next.delete(entry.id)
            return next
          })
        },
      },
    })
  }, [pendingEntry, deleteEntryMutation, t])

  const cancelDelete = () => {
    setPendingEntry(null)
  }

  return {
    requestDelete,
    confirmDelete,
    cancelDelete,
    pendingEntry,
    deletedIds,
  }
}
