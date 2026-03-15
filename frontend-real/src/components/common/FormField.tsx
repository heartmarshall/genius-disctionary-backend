import type { ReactNode } from 'react'

interface FormFieldProps {
  label: string
  error?: ReactNode
  required?: boolean
  id?: string
  children: ReactNode
}

export function FormField({ label, error, required, id, children }: FormFieldProps) {
  return (
    <div className="flex flex-col gap-1.5">
      <label htmlFor={id} className="text-sm font-medium text-text-primary">
        {label}
        {required && <span className="text-poppy"> *</span>}
      </label>
      {children}
      {error && (
        <p className="text-xs text-poppy-fg" role="alert">{error}</p>
      )}
    </div>
  )
}
