import { Outlet } from 'react-router'

function AuthLayout() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-surface px-md">
      <div className="w-full max-w-[var(--container-narrow)]">
        <div className="text-center mb-xl">
          <h1 className="font-heading text-3xl text-accent">MyEnglish</h1>
          <p className="text-text-tertiary text-sm mt-xs">Learn words that matter</p>
        </div>
        <div className="bg-bg-card rounded-lg border border-border p-lg shadow-2">
          <Outlet />
        </div>
      </div>
    </div>
  )
}

export default AuthLayout
