import { RouterProvider } from 'react-router-dom'
import { ApolloProvider } from '@/providers/ApolloProvider'
import { AuthProvider } from '@/providers/AuthProvider'
import { GoogleOAuthProvider } from '@/providers/GoogleOAuthProvider'
import { router } from './router'

export default function App() {
  return (
    <GoogleOAuthProvider>
      <ApolloProvider>
        <AuthProvider>
          <RouterProvider router={router} />
        </AuthProvider>
      </ApolloProvider>
    </GoogleOAuthProvider>
  )
}
