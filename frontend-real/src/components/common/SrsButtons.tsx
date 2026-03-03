import { cn } from '@/lib/utils'

type Grade = 'again' | 'hard' | 'good' | 'easy'

const gradeConfig: Record<Grade, { label: string; bg: string; hover: string; text: string }> = {
  again: { label: 'Again', bg: 'bg-poppy', hover: 'hover:bg-poppy-hover', text: 'text-text-on-accent' },
  hard: { label: 'Hard', bg: 'bg-goldenrod', hover: 'hover:bg-goldenrod-hover', text: 'text-text-on-accent' },
  good: { label: 'Good', bg: 'bg-cornflower', hover: 'hover:bg-cornflower-hover', text: 'text-text-on-accent' },
  easy: { label: 'Easy', bg: 'bg-thyme', hover: 'hover:bg-thyme-hover', text: 'text-text-on-accent' },
}

const grades: Grade[] = ['again', 'hard', 'good', 'easy']

interface SrsButtonsProps {
  onGrade: (grade: Grade) => void
  intervals?: Partial<Record<Grade, string>>
  disabled?: boolean
  className?: string
}

export function SrsButtons({ onGrade, intervals, disabled, className }: SrsButtonsProps) {
  return (
    <div className={cn('flex gap-2', className)}>
      {grades.map((grade) => {
        const config = gradeConfig[grade]
        return (
          <button
            key={grade}
            type="button"
            disabled={disabled}
            onClick={() => onGrade(grade)}
            className={cn(
              'flex flex-1 flex-col items-center rounded-md px-3 py-2 text-sm font-medium transition-colors duration-150',
              config.bg,
              config.hover,
              config.text,
              disabled && 'bg-surface-disabled text-text-disabled cursor-not-allowed'
            )}
          >
            <span>{config.label}</span>
            {intervals?.[grade] && (
              <span className="text-xs opacity-80">{intervals[grade]}</span>
            )}
          </button>
        )
      })}
    </div>
  )
}
