import { Separator } from '@/components/ui/separator'

export function OAuthDivider() {
  return (
    <div className="flex items-center gap-md my-md">
      <Separator className="flex-1" />
      <span className="text-text-tertiary text-xs uppercase tracking-wider">or</span>
      <Separator className="flex-1" />
    </div>
  )
}
