import type { GraphQLResult } from './types'

let getToken: () => string | null = () => null

export function setTokenGetter(fn: () => string | null) {
  getToken = fn
}

const API_URL = '/query'

export async function graphql<T = unknown>(
  query: string,
  variables?: Record<string, unknown>,
): Promise<GraphQLResult<T>> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const body = JSON.stringify({ query, variables })

  const res = await fetch(API_URL, {
    method: 'POST',
    headers,
    body,
  })

  const json = await res.json()

  return {
    data: json.data ?? null,
    errors: json.errors ?? null,
    raw: {
      query,
      variables: variables ?? null,
      response: json,
      status: res.status,
    },
  }
}
