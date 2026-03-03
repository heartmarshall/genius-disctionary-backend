export interface User {
  id: string
  email: string
  username: string
  name: string | null
  avatarUrl: string | null
  role: 'user' | 'admin'
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
}
