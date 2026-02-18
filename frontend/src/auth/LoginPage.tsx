import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from './useAuth'

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID as string | undefined

function getGoogleAuthURL() {
  if (!GOOGLE_CLIENT_ID) return null
  const params = new URLSearchParams({
    client_id: GOOGLE_CLIENT_ID,
    redirect_uri: `${window.location.origin}/login`,
    response_type: 'code',
    scope: 'openid email profile',
    access_type: 'offline',
    prompt: 'consent',
  })
  return `https://accounts.google.com/o/oauth2/v2/auth?${params}`
}

type Tab = 'login' | 'register' | 'oauth' | 'jwt'

export function LoginPage() {
  const { setToken, isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()

  const [tab, setTab] = useState<Tab>('login')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  // Login form
  const [loginEmail, setLoginEmail] = useState('')
  const [loginPassword, setLoginPassword] = useState('')

  // Register form
  const [regEmail, setRegEmail] = useState('')
  const [regUsername, setRegUsername] = useState('')
  const [regPassword, setRegPassword] = useState('')

  // JWT paste
  const [jwt, setJwt] = useState('')

  // Handle Google OAuth callback (?code=...)
  useEffect(() => {
    const code = searchParams.get('code')
    if (!code || isAuthenticated) return

    setLoading(true)
    setError(null)

    fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ provider: 'google', code }),
    })
      .then(async (res) => {
        const data = await res.json()
        if (!res.ok) throw new Error(data.error || 'Login failed')
        localStorage.setItem('refresh_token', data.refreshToken)
        setToken(data.accessToken)
        navigate('/dictionary', { replace: true })
      })
      .catch((err) => {
        setError(err.message)
        navigate('/login', { replace: true })
      })
      .finally(() => setLoading(false))
  }, [searchParams, isAuthenticated, setToken, navigate])

  if (isAuthenticated) {
    navigate('/dictionary')
    return null
  }

  function handleAuthResponse(data: { accessToken: string; refreshToken: string }) {
    localStorage.setItem('refresh_token', data.refreshToken)
    setToken(data.accessToken)
    navigate('/dictionary', { replace: true })
  }

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const res = await fetch('/auth/login/password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: loginEmail, password: loginPassword }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Login failed')
      handleAuthResponse(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setLoading(false)
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const res = await fetch('/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: regEmail, username: regUsername, password: regPassword }),
      })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'Registration failed')
      handleAuthResponse(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  const googleAuthURL = getGoogleAuthURL()

  const tabs: { key: Tab; label: string }[] = [
    { key: 'login', label: 'Login' },
    { key: 'register', label: 'Register' },
    { key: 'oauth', label: 'OAuth' },
    { key: 'jwt', label: 'JWT' },
  ]

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="bg-white shadow-lg rounded-lg p-8 max-w-md w-full space-y-6">
        <h1 className="text-2xl font-bold text-center">MyEnglish</h1>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 text-sm rounded p-3">
            {error}
          </div>
        )}

        {loading && (
          <div className="text-center text-sm text-gray-500">Processing...</div>
        )}

        {/* Tabs */}
        <div className="flex border-b border-gray-200">
          {tabs.map((t) => (
            <button
              key={t.key}
              onClick={() => { setTab(t.key); setError(null) }}
              className={`flex-1 py-2 text-sm font-medium border-b-2 transition-colors ${
                tab === t.key
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>

        {/* Login with password */}
        {tab === 'login' && (
          <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                value={loginEmail}
                onChange={(e) => setLoginEmail(e.target.value)}
                placeholder="user@example.com"
                required
                className="w-full border rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
              <input
                type="password"
                value={loginPassword}
                onChange={(e) => setLoginPassword(e.target.value)}
                placeholder="min 8 characters"
                required
                minLength={8}
                className="w-full border rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <button
              type="submit"
              disabled={loading || !loginEmail || !loginPassword}
              className="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50 font-medium"
            >
              Sign In
            </button>
            <p className="text-xs text-gray-500 text-center">
              Don't have an account?{' '}
              <button type="button" onClick={() => setTab('register')} className="text-blue-600 hover:underline">
                Register
              </button>
            </p>
          </form>
        )}

        {/* Register */}
        {tab === 'register' && (
          <form onSubmit={handleRegister} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
              <input
                type="email"
                value={regEmail}
                onChange={(e) => setRegEmail(e.target.value)}
                placeholder="user@example.com"
                required
                className="w-full border rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
              <input
                type="text"
                value={regUsername}
                onChange={(e) => setRegUsername(e.target.value)}
                placeholder="2-50 characters"
                required
                minLength={2}
                maxLength={50}
                className="w-full border rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
              <input
                type="password"
                value={regPassword}
                onChange={(e) => setRegPassword(e.target.value)}
                placeholder="min 8 characters"
                required
                minLength={8}
                maxLength={72}
                className="w-full border rounded px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>
            <button
              type="submit"
              disabled={loading || !regEmail || !regUsername || !regPassword}
              className="w-full bg-green-600 text-white py-2 rounded hover:bg-green-700 disabled:opacity-50 font-medium"
            >
              Create Account
            </button>
            <p className="text-xs text-gray-500 text-center">
              Already have an account?{' '}
              <button type="button" onClick={() => setTab('login')} className="text-blue-600 hover:underline">
                Sign in
              </button>
            </p>
          </form>
        )}

        {/* Google OAuth */}
        {tab === 'oauth' && (
          <div className="space-y-3">
            <h2 className="text-sm font-medium text-gray-700">Google OAuth</h2>
            {googleAuthURL ? (
              <a
                href={googleAuthURL}
                className="block w-full bg-white border border-gray-300 text-gray-700 py-2 rounded hover:bg-gray-50 text-center font-medium"
              >
                Login with Google
              </a>
            ) : (
              <div>
                <p className="text-xs text-gray-500 mb-2">
                  Set VITE_GOOGLE_CLIENT_ID in .env to enable Google OAuth.
                </p>
                <button
                  disabled
                  className="w-full bg-gray-200 text-gray-400 py-2 rounded cursor-not-allowed"
                >
                  Login with Google (not configured)
                </button>
              </div>
            )}
          </div>
        )}

        {/* JWT paste */}
        {tab === 'jwt' && (
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
            <p className="text-xs text-gray-400 text-center">
              For development: paste a JWT from backend tests or direct generation.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
