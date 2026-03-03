import type { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  error?: string
  required?: boolean
  children: ReactNode
}

export function FormField({ label, error, required, children }: FormFieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-sm font-medium text-text-primary">
        {label}
        {required && <span className="text-poppy"> *</span>}
      </label>
      {children}
      {error && (
        <p className="text-xs text-poppy-fg">{error}</p>
      )}
    </div>
  )
}
