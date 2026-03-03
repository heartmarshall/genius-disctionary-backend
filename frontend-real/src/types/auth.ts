export interface User {
  id: string
  email: string
  username: string
  name: string
  avatarUrl?: string
  role: 'user' | 'admin'
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
}
