import { getAccessToken } from './auth'

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

interface RequestOptions extends Omit<RequestInit, 'body'> {
  body?: unknown
}

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { body, headers: customHeaders, ...rest } = options

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...customHeaders as Record<string, string>,
  }

  const token = getAccessToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(`${BASE_URL}${path}`, {
    ...rest,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }))
    throw Object.assign(new Error(error.error || response.statusText), {
      status: response.status,
      data: error,
    })
  }

  return response.json()
}
