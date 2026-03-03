import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import { useApolloClient } from '@apollo/client/react'
import type { User, AuthTokens } from '@/types/auth'
import {
  getAccessToken,
  setAccessToken,
  setRefreshToken,
  clearTokens,
} from '@/lib/auth'
import { apiFetch } from '@/lib/api'
import { ME_QUERY } from '@/graphql/queries/me'

interface AuthContextValue {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (tokens: AuthTokens, user: User) => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const client = useApolloClient()

  useEffect(() => {
    const token = getAccessToken()
    if (!token) {
      setIsLoading(false)
      return
    }

    client
      .query<{ me: User }>({ query: ME_QUERY, fetchPolicy: 'network-only' })
      .then(({ data }) => { if (data?.me) setUser(data.me) })
      .catch(() => clearTokens())
      .finally(() => setIsLoading(false))
  }, [client])

  const login = useCallback((tokens: AuthTokens, userData: User) => {
    setAccessToken(tokens.accessToken)
    setRefreshToken(tokens.refreshToken)
    setUser(userData)
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiFetch('/auth/logout', { method: 'POST' })
    } catch {
      // Logout endpoint may fail if token is already expired — ignore
    } finally {
      clearTokens()
      setUser(null)
      await client.clearStore()
      window.location.href = '/login'
    }
  }, [client])

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return context
}
