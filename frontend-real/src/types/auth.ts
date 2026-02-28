export interface User {
  id: string
  email: string
  username: string
  name: string
  avatarUrl: string | null
  role: 'user' | 'admin'
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
}

export interface AuthResponse {
  accessToken: string
  refreshToken: string
  user: User
}

export interface ValidationFieldError {
  field: string
  message: string
}

export interface ValidationErrorResponse {
  error: string
  code: 'VALIDATION'
  fields: ValidationFieldError[]
}
