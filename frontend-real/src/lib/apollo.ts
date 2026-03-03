import {
  ApolloClient,
  InMemoryCache,
  HttpLink,
  ApolloLink,
  Observable,
  from,
} from '@apollo/client'
import { onError } from '@apollo/client/link/error'
import { ServerError } from '@apollo/client/errors'
import { getAccessToken, getRefreshToken, setAccessToken, setRefreshToken, clearTokens } from './auth'

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const httpLink = new HttpLink({
  uri: `${BASE_URL}/query`,
})

const authLink = new ApolloLink((operation, forward) => {
  const token = getAccessToken()
  if (token) {
    operation.setContext({
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })
  }
  return forward(operation)
})

const errorLink = onError(({ error, operation, forward }) => {
  if (ServerError.is(error) && error.statusCode === 401) {
    const refreshToken = getRefreshToken()
    if (!refreshToken) {
      clearTokens()
      window.location.href = '/login'
      return
    }

    return new Observable((observer) => {
      fetch(`${BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken }),
      })
        .then((res) => {
          if (!res.ok) throw new Error('Refresh failed')
          return res.json()
        })
        .then(({ accessToken, refreshToken: newRefreshToken }) => {
          setAccessToken(accessToken)
          setRefreshToken(newRefreshToken)

          operation.setContext({
            headers: {
              Authorization: `Bearer ${accessToken}`,
            },
          })

          forward(operation).subscribe(observer)
        })
        .catch(() => {
          clearTokens()
          window.location.href = '/login'
          observer.complete()
        })
    })
  }
})

const cache = new InMemoryCache({
  typePolicies: {
    Query: {
      fields: {
        dictionary: {
          keyArgs: ['input', ['search', 'sortField', 'sortDirection', 'hasCard', 'partOfSpeech', 'topicId', 'status']],
          merge(existing, incoming) {
            if (!existing) return incoming
            return {
              ...incoming,
              edges: [...existing.edges, ...incoming.edges],
            }
          },
        },
      },
    },
  },
})

export const apolloClient = new ApolloClient({
  link: from([authLink, errorLink, httpLink]),
  cache,
})
