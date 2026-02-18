import { createContext, useState, useCallback, useEffect, type ReactNode } from 'react'
import { setTokenGetter } from '../api/client'

export interface AuthContextValue {
  token: string | null
  setToken: (token: string | null) => void
  isAuthenticated: boolean
  logout: () => void
}

export const AuthContext = createContext<AuthContextValue>({
  token: null,
  setToken: () => {},
  isAuthenticated: false,
  logout: () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setTokenState] = useState<string | null>(
    () => localStorage.getItem('jwt_token'),
  )

  const setToken = useCallback((t: string | null) => {
    setTokenState(t)
    if (t) {
      localStorage.setItem('jwt_token', t)
    } else {
      localStorage.removeItem('jwt_token')
    }
  }, [])

  const logout = useCallback(() => setToken(null), [setToken])

  useEffect(() => {
    setTokenGetter(() => token)
  }, [token])

  return (
    <AuthContext.Provider value={{ token, setToken, isAuthenticated: !!token, logout }}>
      {children}
    </AuthContext.Provider>
  )
}
