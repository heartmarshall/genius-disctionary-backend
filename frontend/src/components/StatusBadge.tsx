const STATUS_COLORS: Record<string, string> = {
  NEW: 'bg-blue-100 text-blue-800',
  LEARNING: 'bg-yellow-100 text-yellow-800',
  REVIEW: 'bg-purple-100 text-purple-800',
  MASTERED: 'bg-green-100 text-green-800',
}

interface Props {
  status: string | null | undefined
  className?: string
}

export function StatusBadge({ status, className = '' }: Props) {
  if (!status)
    return (
      <span
        className={`text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-500 ${className}`}
      >
        No Card
      </span>
    )
  const colors = STATUS_COLORS[status] ?? 'bg-gray-100 text-gray-800'
  return (
    <span
      className={`text-xs px-2 py-0.5 rounded-full font-medium ${colors} ${className}`}
    >
      {status}
    </span>
  )
}
