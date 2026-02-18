import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from './useAuth'

export function LoginPage() {
  const { setToken, isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const [jwt, setJwt] = useState('')

  if (isAuthenticated) {
    navigate('/dictionary')
    return null
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white shadow-lg rounded-lg p-8 max-w-md w-full space-y-6">
        <h1 className="text-2xl font-bold text-center">MyEnglish Test Frontend</h1>

        <div className="space-y-3">
          <h2 className="text-sm font-medium text-gray-700">Paste JWT Token</h2>
          <textarea
            value={jwt}
            onChange={(e) => setJwt(e.target.value)}
            placeholder="eyJhbGciOiJIUzI1NiIs..."
            className="w-full border rounded p-2 text-xs font-mono h-24 resize-none"
          />
          <button
            onClick={() => {
              const trimmed = jwt.trim()
              if (trimmed) {
                setToken(trimmed)
                navigate('/dictionary')
              }
            }}
            disabled={!jwt.trim()}
            className="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50"
          >
            Use Token
          </button>
        </div>

        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300" />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white text-gray-500">or</span>
          </div>
        </div>

        <div className="space-y-3">
          <h2 className="text-sm font-medium text-gray-700">Google OAuth</h2>
          <p className="text-xs text-gray-500">
            Backend auth endpoints (login/refresh/logout) are not yet exposed via HTTP.
            Use manual JWT token for now.
          </p>
          <button
            disabled
            className="w-full bg-gray-200 text-gray-400 py-2 rounded cursor-not-allowed"
          >
            Login with Google (not yet available)
          </button>
        </div>

        <div className="text-xs text-gray-400 text-center">
          <p>To get a JWT: run the backend, create a user via tests or direct DB insert,
          then use the auth service to generate a token.</p>
        </div>
      </div>
    </div>
  )
}
