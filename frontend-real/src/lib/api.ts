import {
  getAccessToken,
  getRefreshToken,
  setAccessToken,
  setRefreshToken,
  clearTokens,
} from './auth'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
  auth?: boolean
}

// ── Refresh queue (shared with apollo.ts via auth.ts storage) ──

let refreshPromise: Promise<boolean> | null = null

async function tryRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken()
  if (!refreshToken) return false

  try {
    const response = await fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    })

    if (!response.ok) return false

    const data: { accessToken: string; refreshToken: string } = await response.json()
    setAccessToken(data.accessToken)
    setRefreshToken(data.refreshToken)
    return true
  } catch {
    return false
  }
}

function refreshTokens(): Promise<boolean> {
  // Deduplicate concurrent refresh attempts
  if (!refreshPromise) {
    refreshPromise = tryRefresh().finally(() => {
      refreshPromise = null
    })
  }
  return refreshPromise
}

// ── Main fetch helper ──

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await doFetch(path, options)

  // On 401 with auth enabled, try refresh and retry once
  if (response.status === 401 && options.auth !== false) {
    const refreshed = await refreshTokens()
    if (refreshed) {
      const retryResponse = await doFetch(path, options)
      if (!retryResponse.ok) {
        throw await buildApiError(retryResponse)
      }
      return retryResponse.json() as Promise<T>
    }

    // Refresh failed — force logout
    clearTokens()
    window.location.href = '/login'
    throw await buildApiError(response)
  }

  if (!response.ok) {
    throw await buildApiError(response)
  }

  return response.json() as Promise<T>
}

async function doFetch(path: string, options: RequestOptions): Promise<Response> {
  const { body, auth = true, headers: extraHeaders, ...rest } = options

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(extraHeaders as Record<string, string>),
  }

  if (auth) {
    const token = getAccessToken()
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
  }

  return fetch(`${API_URL}${path}`, {
    ...rest,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })
}

async function buildApiError(response: Response): Promise<ApiError> {
  const data = await response.json().catch(() => ({ error: response.statusText }))
  const retryAfter = response.headers.get('Retry-After')
  return new ApiError(response.status, data, retryAfter ? parseInt(retryAfter, 10) : undefined)
}

export class ApiError extends Error {
  status: number
  data: unknown
  retryAfter?: number

  constructor(status: number, data: unknown, retryAfter?: number) {
    super(`API Error: ${status}`)
    this.name = 'ApiError'
    this.status = status
    this.data = data
    this.retryAfter = retryAfter
  }
}

export { API_URL }
