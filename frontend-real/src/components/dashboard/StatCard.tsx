import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface StatCardProps {
  label: string
  value: number | string
  accent?: string
  icon?: ReactNode
  subtitle?: string
  className?: string
}

function StatCard({ label, value, accent, icon, subtitle, className }: StatCardProps) {
  return (
    <div
      className={cn(
        'rounded-md border border-border bg-bg-card p-md shadow-1',
        className,
      )}
    >
      <div className="flex items-start justify-between">
        <span className="text-[13px] font-medium text-text-secondary">{label}</span>
        {icon && <span className="text-text-tertiary">{icon}</span>}
      </div>
      <div className={cn('mt-sm text-[28px] font-heading leading-tight', accent ?? 'text-text-primary')}>
        {value}
      </div>
      {subtitle && (
        <p className="mt-xs text-[11px] text-text-tertiary">{subtitle}</p>
      )}
    </div>
  )
}

export { StatCard }
