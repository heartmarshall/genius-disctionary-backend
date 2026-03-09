// Stub — will be fully implemented in Task 12
interface WordEditDialogProps {
  wordId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function WordEditDialog({ open, onOpenChange }: WordEditDialogProps) {
  if (!open) return null
  return null
}
