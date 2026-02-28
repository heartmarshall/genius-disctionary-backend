import { Navigate } from 'react-router'
import { useAuth } from '@/providers/AuthProvider'

interface Props {
  children: React.ReactNode
}

export function GuestRoute({ children }: Props) {
  const { isAuthenticated, isLoading } = useAuth()

  // While checking auth status, show nothing (AuthLayout handles the shell)
  if (isLoading) return null

  // Already logged in — redirect to dashboard
  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />
  }

  return <>{children}</>
}
