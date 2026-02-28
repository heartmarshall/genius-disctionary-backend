import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router'
import { toast } from 'sonner'
import { useAuth } from '@/providers/AuthProvider'
import { register } from '@/lib/api/auth'
import { parseAuthError } from '@/lib/api/auth'
import { registerResolver, type RegisterFormValues } from '@/lib/validation/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Eye, EyeOff, Loader2 } from 'lucide-react'

function RegisterPage() {
  const navigate = useNavigate()
  const { login } = useAuth()
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirm, setShowConfirm] = useState(false)

  const {
    register: field,
    handleSubmit,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({
    resolver: registerResolver,
    mode: 'onBlur',
    defaultValues: { email: '', username: '', password: '', confirmPassword: '' },
  })

  const onSubmit = async (data: RegisterFormValues) => {
    try {
      const response = await register(data.email, data.username, data.password)
      login({ accessToken: response.accessToken, refreshToken: response.refreshToken }, response.user)
      toast.success('Account created!')
      navigate('/dashboard', { replace: true })
    } catch (err) {
      const parsed = parseAuthError(err)

      if (parsed.type === 'validation' && parsed.fieldErrors) {
        for (const fe of parsed.fieldErrors) {
          if (fe.field === 'email' || fe.field === 'username' || fe.field === 'password') {
            setError(fe.field, { message: fe.message })
          }
        }
      } else {
        toast.error(parsed.message)
      }
    }
  }

  return (
    <div>
      <h2 className="font-heading text-2xl mb-xs">Create account</h2>
      <p className="text-text-secondary text-sm mb-lg">Start building your vocabulary</p>

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

        {/* Username */}
        <div>
          <label htmlFor="username" className="block text-sm font-medium mb-xs">
            Username
          </label>
          <Input
            id="username"
            type="text"
            autoComplete="username"
            placeholder="your_username"
            aria-invalid={!!errors.username}
            {...field('username')}
          />
          {errors.username && (
            <p className="text-poppy text-xs mt-xs">{errors.username.message}</p>
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
              autoComplete="new-password"
              placeholder="8+ characters"
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

        {/* Confirm Password */}
        <div>
          <label htmlFor="confirmPassword" className="block text-sm font-medium mb-xs">
            Confirm password
          </label>
          <div className="relative">
            <Input
              id="confirmPassword"
              type={showConfirm ? 'text' : 'password'}
              autoComplete="new-password"
              placeholder="Repeat password"
              aria-invalid={!!errors.confirmPassword}
              {...field('confirmPassword')}
            />
            <button
              type="button"
              className="absolute right-2 top-1/2 -translate-y-1/2 text-text-tertiary hover:text-text-primary p-1"
              onClick={() => setShowConfirm(!showConfirm)}
              tabIndex={-1}
              aria-label={showConfirm ? 'Hide password' : 'Show password'}
            >
              {showConfirm ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>
          </div>
          {errors.confirmPassword && (
            <p className="text-poppy text-xs mt-xs">{errors.confirmPassword.message}</p>
          )}
        </div>

        <Button type="submit" className="w-full" disabled={isSubmitting}>
          {isSubmitting ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Creating account…
            </>
          ) : (
            'Create Account'
          )}
        </Button>
      </form>

      <p className="text-center text-sm text-text-secondary mt-lg">
        Already have an account?{' '}
        <Link to="/login" className="text-accent hover:underline font-medium">
          Sign in
        </Link>
      </p>
    </div>
  )
}

export default RegisterPage
