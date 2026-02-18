import { useState, useCallback } from 'react'
import { graphql } from '../api/client'
import type { GraphQLError } from '../api/types'

export interface UseGraphQLState<T> {
  data: T | null
  errors: GraphQLError[] | null
  loading: boolean
  raw: { query: string; variables: unknown; response: unknown; status: number } | null
  execute: (query: string, variables?: Record<string, unknown>) => Promise<T | null>
  reset: () => void
}

export function useGraphQL<T = unknown>(): UseGraphQLState<T> {
  const [data, setData] = useState<T | null>(null)
  const [errors, setErrors] = useState<GraphQLError[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [raw, setRaw] = useState<UseGraphQLState<T>['raw']>(null)

  const execute = useCallback(async (query: string, variables?: Record<string, unknown>) => {
    setLoading(true)
    setErrors(null)
    try {
      const result = await graphql<T>(query, variables)
      setData(result.data)
      setErrors(result.errors)
      setRaw(result.raw)
      return result.data
    } catch (err) {
      const networkError: GraphQLError = { message: String(err), extensions: { code: 'NETWORK_ERROR' } }
      setErrors([networkError])
      setRaw(null)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const reset = useCallback(() => {
    setData(null)
    setErrors(null)
    setRaw(null)
  }, [])

  return { data, errors, loading, raw, execute, reset }
}
