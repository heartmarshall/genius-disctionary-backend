import { useContext } from 'react'
import { ToastContext, type ToastContextValue } from './ToastProvider'

export function useToast(): ToastContextValue {
  return useContext(ToastContext)
}
