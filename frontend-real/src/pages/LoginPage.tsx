import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate, useLocation } from 'react-router'
import { toast } from 'sonner'
import { useAuth } from '@/providers/AuthProvider'
import { loginPassword, parseAuthError } from '@/lib/api/auth'
import { loginResolver, type LoginFormValues } from '@/lib/validation/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { GoogleSignInButton } from '@/components/auth/GoogleSignInButton'
import { OAuthDivider } from '@/components/auth/OAuthDivider'
import { Eye, EyeOff, Loader2 } from 'lucide-react'

function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login } = useAuth()
  const [showPassword, setShowPassword] = useState(false)
  const [genericError, setGenericError] = useState<string | null>(null)

  const from = (location.state as { from?: { pathname: string } })?.from?.pathname || '/dashboard'

  const {
    register: field,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({
    resolver: loginResolver,
    mode: 'onBlur',
    defaultValues: { email: '', password: '' },
  })

  const onSubmit = async (data: LoginFormValues) => {
    setGenericError(null)

    try {
      const response = await loginPassword(data.email, data.password)
      login({ accessToken: response.accessToken, refreshToken: response.refreshToken }, response.user)
      navigate(from, { replace: true })
    } catch (err) {
      const parsed = parseAuthError(err)

      // Never show field-specific errors for login (prevents enumeration)
      if (parsed.type === 'rate_limited') {
        toast.error(parsed.message)
      } else {
        setGenericError(parsed.message)
      }
    }
  }

  return (
    <div>
      <h2 className="font-heading text-2xl mb-xs">Welcome back</h2>
      <p className="text-text-secondary text-sm mb-lg">Sign in to continue learning</p>

      {genericError && (
        <div className="bg-poppy-light border border-poppy/20 text-poppy-fg rounded-md px-md py-sm text-sm mb-md">
          {genericError}
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-md" noValidate>
        {/* Email */}
        <div>
          <label htmlFor="email" className="block text-sm font-medium mb-xs">
            Email
          </label>
          <Input
            id="email"
            type="email"
            autoComplete="email"
            placeholder="you@example.com"
            aria-invalid={!!errors.email}
            {...field('email')}
          />
          {errors.email && (
            <p className="text-poppy text-xs mt-xs">{errors.email.message}</p>
          )}
        </div>

        {/* Password */}
        <div>
          <label htmlFor="password" className="block text-sm font-medium mb-xs">
            Password
          </label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? 'text' : 'password'}
              autoComplete="current-password"
              placeholder="Enter your password"
              aria-invalid={!!errors.password}
              {...field('password')}
            />
            <button
              type="button"
              className="absolute right-2 top-1/2 -translate-y-1/2 text-text-tertiary hover:text-text-primary p-1"
              onClick={() => setShowPassword(!showPassword)}
              tabIndex={-1}
              aria-label={showPassword ? 'Hide password' : 'Show password'}
            >
              {showPassword ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>
          </div>
          {errors.password && (
            <p className="text-poppy text-xs mt-xs">{errors.password.message}</p>
          )}
        </div>

        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Signing in…
            </>
          ) : (
            'Sign In'
          )}
        </Button>
      </form>

      <OAuthDivider />
      <GoogleSignInButton />

      <p className="text-center text-sm text-text-secondary mt-lg">
        Don&apos;t have an account?{' '}
        <Link to="/register" className="text-accent hover:underline font-medium">
          Create one
        </Link>
      </p>
    </div>
  )
}

export default LoginPage
