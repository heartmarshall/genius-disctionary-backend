import { apiFetch, ApiError } from '@/lib/api'
import type { AuthResponse, ValidationErrorResponse, ValidationFieldError } from '@/types/auth'

// ── API Functions ──

export async function register(
  email: string,
  username: string,
  password: string,
): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/auth/register', {
    method: 'POST',
    body: { email: email.trim().toLowerCase(), username, password },
    auth: false,
  })
}

export async function loginPassword(email: string, password: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/auth/login/password', {
    method: 'POST',
    body: { email: email.trim().toLowerCase(), password },
    auth: false,
  })
}

export async function loginOAuth(provider: string, code: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/auth/login', {
    method: 'POST',
    body: { provider, code },
    auth: false,
  })
}

export async function refreshToken(token: string): Promise<AuthResponse> {
  return apiFetch<AuthResponse>('/auth/refresh', {
    method: 'POST',
    body: { refreshToken: token },
    auth: false,
  })
}

export async function logout(): Promise<void> {
  await apiFetch<void>('/auth/logout', { method: 'POST' })
}

// ── Error Helpers ──

export interface AuthApiError {
  type: 'validation' | 'unauthorized' | 'rate_limited' | 'unknown'
  message: string
  fieldErrors?: ValidationFieldError[]
  retryAfter?: number
}

export function parseAuthError(error: unknown): AuthApiError {
  if (!(error instanceof ApiError)) {
    return { type: 'unknown', message: 'An unexpected error occurred. Please try again.' }
  }

  switch (error.status) {
    case 400: {
      const data = error.data as ValidationErrorResponse
      if (data?.code === 'VALIDATION' && Array.isArray(data.fields)) {
        return {
          type: 'validation',
          message: data.error || 'Please fix the errors below.',
          fieldErrors: data.fields,
        }
      }
      return { type: 'validation', message: data?.error || 'Invalid request.' }
    }

    case 401:
      return { type: 'unauthorized', message: 'Invalid email or password.' }

    case 429: {
      return {
        type: 'rate_limited',
        message: 'Too many attempts. Please try again later.',
        retryAfter: error.retryAfter ?? 60,
      }
    }

    default:
      return { type: 'unknown', message: 'Something went wrong. Please try again.' }
  }
}

