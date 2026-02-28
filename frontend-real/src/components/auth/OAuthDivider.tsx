import { Separator } from '@/components/ui/separator'

const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID || ''

export function OAuthDivider() {
  // Don't render if no OAuth providers are configured
  if (!googleClientId) return null

  return (
    <div className="flex items-center gap-md my-md">
      <Separator className="flex-1" />
      <span className="text-text-tertiary text-xs uppercase tracking-wider">or</span>
      <Separator className="flex-1" />
    </div>
  )
}
