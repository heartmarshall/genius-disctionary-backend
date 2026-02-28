import { cn } from '@/lib/utils'

type Grade = 'again' | 'hard' | 'good' | 'easy'

interface SrsButtonProps {
  grade: Grade
  interval: string
  onClick?: () => void
  disabled?: boolean
}

interface SrsButtonsProps {
  intervals: { again: string; hard: string; good: string; easy: string }
  onGrade?: (grade: Grade) => void
  disabled?: boolean
  className?: string
}

const gradeConfig: Record<Grade, { label: string; bg: string }> = {
  again: { label: 'Again', bg: 'bg-srs-again' },
  hard: { label: 'Hard', bg: 'bg-srs-hard' },
  good: { label: 'Good', bg: 'bg-srs-good' },
  easy: { label: 'Easy', bg: 'bg-srs-easy' },
}

const grades: Grade[] = ['again', 'hard', 'good', 'easy']

function SrsButton({ grade, interval, onClick, disabled }: SrsButtonProps) {
  const config = gradeConfig[grade]

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        'flex-1 rounded-md py-[10px] px-1 text-center font-body text-[13px] font-semibold text-white transition-all',
        disabled
          ? 'cursor-not-allowed bg-bark-light text-text-disabled'
          : cn(
              config.bg,
              'cursor-pointer hover:-translate-y-0.5 hover:shadow-[0_4px_12px_oklch(0_0_0/0.15)] active:scale-[0.97]',
            ),
      )}
      style={{ transitionDuration: 'var(--duration-fast)' }}
    >
      {config.label}
      <small className="block text-[11px] font-normal opacity-80 mt-0.5">{interval}</small>
    </button>
  )
}

function SrsButtons({ intervals, onGrade, disabled, className }: SrsButtonsProps) {
  return (
    <div className={cn('flex gap-1.5', className)}>
      {grades.map((grade) => (
        <SrsButton
          key={grade}
          grade={grade}
          interval={intervals[grade]}
          onClick={() => onGrade?.(grade)}
          disabled={disabled}
        />
      ))}
    </div>
  )
}

export { SrsButtons, SrsButton }
export type { Grade, SrsButtonsProps }
