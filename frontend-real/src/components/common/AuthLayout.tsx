import { Navigate, Outlet } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/providers/AuthProvider'
import { Skeleton } from '@/components/ui/skeleton'

function LanguageToggle() {
  const { i18n } = useTranslation()
  const current = i18n.resolvedLanguage?.startsWith('ru') ? 'ru' : 'en'
  const next = current === 'en' ? 'ru' : 'en'
  const label = current.toUpperCase()

  return (
    <button
      type="button"
      onClick={() => i18n.changeLanguage(next)}
      className="text-sm font-medium text-text-secondary transition-colors duration-150 hover:text-poppy"
      aria-label={`Switch language to ${next === 'en' ? 'English' : 'Russian'}`}
    >
      {label}
    </button>
  )
}

export function AuthLayout() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-bg-page px-4">
        <div className="flex w-full max-w-md flex-col gap-4">
          <Skeleton className="h-8 w-1/2" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    )
  }

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-bg-page px-4">
      <div className="absolute right-4 top-4">
        <LanguageToggle />
      </div>
      <div className="w-full max-w-md">
        <Outlet />
      </div>
    </div>
  )
}
