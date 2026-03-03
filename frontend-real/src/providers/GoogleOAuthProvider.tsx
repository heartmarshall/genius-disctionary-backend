import type { ReactNode } from 'react'
import { GoogleOAuthProvider as Provider } from '@react-oauth/google'

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID

export function GoogleOAuthProvider({ children }: { children: ReactNode }) {
  if (!GOOGLE_CLIENT_ID) return <>{children}</>

  return <Provider clientId={GOOGLE_CLIENT_ID}>{children}</Provider>
}
