import { useContext } from 'react'
import { AuthContext, type AuthContextValue } from './AuthProvider'

export function useAuth(): AuthContextValue {
  return useContext(AuthContext)
}
