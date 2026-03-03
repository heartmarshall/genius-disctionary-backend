import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-bg-page">
      <h1 className="text-3xl font-bold text-text-primary">404</h1>
      <p className="text-text-secondary">Страница не найдена</p>
      <Link
        to="/dashboard"
        className="text-sm font-medium text-poppy hover:text-poppy-hover transition-colors duration-150"
      >
        На главную
      </Link>
    </div>
  )
}
