import { ApolloProvider as BaseApolloProvider } from '@apollo/client/react'
import { apolloClient } from '@/lib/apollo'
import type { ReactNode } from 'react'

interface Props {
  children: ReactNode
}

export function ApolloProvider({ children }: Props) {
  return <BaseApolloProvider client={apolloClient}>{children}</BaseApolloProvider>
}
