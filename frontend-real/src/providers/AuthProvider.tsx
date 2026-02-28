import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react'
import { gql } from '@apollo/client/core'
import { apolloClient } from '@/lib/apollo'
import { apiFetch } from '@/lib/api'
import { setAccessToken, setRefreshToken, clearTokens, hasTokens } from '@/lib/auth'
import type { User, AuthTokens } from '@/types/auth'

interface AuthContextValue {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (tokens: AuthTokens, user: User) => void
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

const ME_QUERY = gql`
  query Me {
    me {
      id
      email
      username
      name
      role
    }
  }
`

interface Props {
  children: ReactNode
}

export function AuthProvider({ children }: Props) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    if (!hasTokens()) {
      setIsLoading(false)
      return
    }

    apolloClient
      .query<{ me: User }>({ query: ME_QUERY, fetchPolicy: 'network-only' })
      .then(({ data }) => {
        if (data?.me) setUser(data.me)
      })
      .catch(() => {
        clearTokens()
      })
      .finally(() => {
        setIsLoading(false)
      })
  }, [])

  const login = useCallback((tokens: AuthTokens, userData: User) => {
    setAccessToken(tokens.accessToken)
    setRefreshToken(tokens.refreshToken)
    setUser(userData)
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiFetch('/auth/logout', { method: 'POST' })
    } catch {
      // Logout best-effort — clear tokens regardless
    } finally {
      clearTokens()
      setUser(null)
      await apolloClient.clearStore()
    }
  }, [])

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: user !== null,
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
