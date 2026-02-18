import { createContext, useState, useCallback, useEffect, useRef, type ReactNode } from 'react'

export interface Toast {
  id: number
  message: string
  type: 'success' | 'error' | 'warning' | 'info'
}

export interface ToastContextValue {
  addToast: (message: string, type: Toast['type']) => void
}

export const ToastContext = createContext<ToastContextValue>({
  addToast: () => {},
})

const AUTO_DISMISS_MS = 5000

const typeStyles: Record<Toast['type'], string> = {
  success: 'bg-green-600',
  error: 'bg-red-600',
  warning: 'bg-yellow-500',
  info: 'bg-blue-600',
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([])
  const nextId = useRef(1)

  const removeToast = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  const addToast = useCallback((message: string, type: Toast['type']) => {
    const id = nextId.current++
    setToasts((prev) => [...prev, { id, message, type }])
  }, [])

  return (
    <ToastContext.Provider value={{ addToast }}>
      {children}
      <ToastContainer toasts={toasts} onRemove={removeToast} />
    </ToastContext.Provider>
  )
}

function ToastContainer({
  toasts,
  onRemove,
}: {
  toasts: Toast[]
  onRemove: (id: number) => void
}) {
  if (toasts.length === 0) return null

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
      {toasts.map((toast) => (
        <ToastItem key={toast.id} toast={toast} onRemove={onRemove} />
      ))}
    </div>
  )
}

function ToastItem({
  toast,
  onRemove,
}: {
  toast: Toast
  onRemove: (id: number) => void
}) {
  useEffect(() => {
    const timer = setTimeout(() => {
      onRemove(toast.id)
    }, AUTO_DISMISS_MS)
    return () => clearTimeout(timer)
  }, [toast.id, onRemove])

  return (
    <div
      className={`${typeStyles[toast.type]} text-white px-4 py-3 rounded-lg shadow-lg flex items-start gap-2 transition-all duration-300 animate-in`}
      role="alert"
    >
      <span className="flex-1 text-sm">{toast.message}</span>
      <button
        onClick={() => onRemove(toast.id)}
        className="text-white/80 hover:text-white shrink-0 leading-none text-lg font-medium"
        aria-label="Close"
      >
        &times;
      </button>
    </div>
  )
}
