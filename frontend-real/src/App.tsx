import { RouterProvider } from 'react-router-dom'
import { ApolloProvider } from '@/providers/ApolloProvider'
import { AuthProvider } from '@/providers/AuthProvider'
import { router } from './router'

export default function App() {
  return (
    <ApolloProvider>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
    </ApolloProvider>
  )
}
