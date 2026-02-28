import { Navigate, useLocation } from 'react-router'
import { useAuth } from '@/providers/AuthProvider'
import { Skeleton } from '@/components/ui/skeleton'

interface Props {
  children: React.ReactNode
}

export function ProtectedRoute({ children }: Props) {
  const { isAuthenticated, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) {
    return (
      <div className="flex flex-col gap-md p-xl max-w-[var(--container-max)] mx-auto">
        <Skeleton className="h-8 w-48 bg-straw" />
        <Skeleton className="h-4 w-72 bg-straw" />
        <div className="flex gap-sm mt-lg">
          <Skeleton className="h-24 flex-1 bg-straw" />
          <Skeleton className="h-24 flex-1 bg-straw" />
          <Skeleton className="h-24 flex-1 bg-straw" />
        </div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <>{children}</>
}
