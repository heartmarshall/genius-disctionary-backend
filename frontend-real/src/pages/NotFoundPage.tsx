import { Link } from 'react-router'

function NotFoundPage() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
      <h1 className="font-heading text-4xl text-accent mb-sm">404</h1>
      <p className="text-text-secondary mb-lg">Page not found</p>
      <Link
        to="/dashboard"
        className="text-accent hover:text-accent-hover underline underline-offset-4"
      >
        Back to Dashboard
      </Link>
    </div>
  )
}

export default NotFoundPage
