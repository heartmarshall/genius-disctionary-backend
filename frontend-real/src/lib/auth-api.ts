import { apiFetch } from './api'
import type { AuthResponse, ApiError } from '@/types/auth'

/** Type guard for ApiError objects thrown by auth-api functions */
export function isApiError(err: unknown): err is ApiError {
  return (
    typeof err === 'object' &&
    err !== null &&
    'status' in err &&
    typeof (err as ApiError).status === 'number' &&
    'error' in err &&
    typeof (err as ApiError).error === 'string'
  )
}

/**
 * Convert the raw error thrown by apiFetch into a structured ApiError.
 * apiFetch throws: Object.assign(new Error(...), { status, data })
 */
function extractApiError(err: unknown): ApiError {
  if (
    err instanceof Error &&
    'status' in err &&
    'data' in err
  ) {
    const { status, data } = err as Error & { status: number; data: Record<string, unknown> }
    return {
      status,
      error: (data.error as string) || err.message,
      code: data.code as string | undefined,
      fields: data.fields as ApiError['fields'],
    }
  }
  return {
    status: 0,
    error: err instanceof Error ? err.message : 'Unknown error',
  }
}

export async function registerUser(
  email: string,
  username: string,
  password: string,
): Promise<AuthResponse> {
  try {
    return await apiFetch<AuthResponse>('/auth/register', {
      method: 'POST',
      body: { email, username, password },
    })
  } catch (err) {
    throw extractApiError(err)
  }
}

export async function loginPassword(
  email: string,
  password: string,
): Promise<AuthResponse> {
  try {
    return await apiFetch<AuthResponse>('/auth/login/password', {
      method: 'POST',
      body: { email, password },
    })
  } catch (err) {
    throw extractApiError(err)
  }
}

export async function loginOAuth(
  provider: 'google',
  code: string,
): Promise<AuthResponse> {
  try {
    return await apiFetch<AuthResponse>('/auth/login', {
      method: 'POST',
      body: { provider, code },
    })
  } catch (err) {
    throw extractApiError(err)
  }
}
