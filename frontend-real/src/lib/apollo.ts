import {
  ApolloClient,
  InMemoryCache,
  HttpLink,
  ApolloLink,
  Observable,
  type FetchResult,
} from '@apollo/client/core'
import { onError } from '@apollo/client/link/error'
import {
  getAccessToken,
  getRefreshToken,
  setAccessToken,
  setRefreshToken,
  clearTokens,
} from './auth'
import { API_URL } from './api'

const httpLink = new HttpLink({
  uri: `${API_URL}/query`,
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

let isRefreshing = false
let pendingRequests: Array<() => void> = []

function resolvePendingRequests() {
  pendingRequests.forEach((callback) => callback())
  pendingRequests = []
}

const errorLink = onError(({ error, operation, forward }) => {
  // Check for auth errors (401 / UNAUTHENTICATED)
  const errorMessage = String(error)
  const isAuthError = errorMessage.includes('401') || errorMessage.includes('UNAUTHENTICATED')

  if (!isAuthError) return

  if (isRefreshing) {
    return new Observable<FetchResult>((observer) => {
      pendingRequests.push(() => {
        forward(operation).subscribe(observer)
      })
    })
  }

  isRefreshing = true

  return new Observable<FetchResult>((observer) => {
    const refreshToken = getRefreshToken()
    if (!refreshToken) {
      clearTokens()
      window.location.href = '/login'
      observer.error(new Error('No refresh token'))
      return
    }

    fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refreshToken }),
    })
      .then((response) => {
        if (!response.ok) throw new Error('Refresh failed')
        return response.json()
      })
      .then((data: { accessToken: string; refreshToken: string }) => {
        setAccessToken(data.accessToken)
        setRefreshToken(data.refreshToken)
        isRefreshing = false
        resolvePendingRequests()
        forward(operation).subscribe(observer)
      })
      .catch(() => {
        isRefreshing = false
        pendingRequests = []
        clearTokens()
        window.location.href = '/login'
        observer.error(new Error('Token refresh failed'))
      })
  })
})

const cache = new InMemoryCache({
  typePolicies: {
    Query: {
      fields: {
        dictionary: {
          keyArgs: [
            'input',
            [
              'search',
              'sortField',
              'sortDirection',
              'hasCard',
              'partOfSpeech',
              'topicId',
              'status',
            ],
          ],
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
  link: ApolloLink.from([authLink, errorLink, httpLink]),
  cache,
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
    },
  },
})
