import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuth } from '@/providers/AuthProvider'
import { loginPassword, isApiError } from '@/lib/auth-api'
import { GoogleSignInButton } from '@/components/auth/GoogleSignInButton'
import { FormField } from '@/components/common/FormField'
import { PasswordInput } from '@/components/common/PasswordInput'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface LoginFormValues {
  email: string
  password: string
}

export function LoginPage() {
  const { t } = useTranslation('auth')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const redirectTo = searchParams.get('redirect') || '/dashboard'
  const auth = useAuth()
  const [genericError, setGenericError] = useState('')

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormValues>({ mode: 'onBlur' })

  const onSubmit = async (data: LoginFormValues) => {
    setGenericError('')
    try {
      const response = await loginPassword(data.email, data.password)
      auth.login({ accessToken: response.accessToken, refreshToken: response.refreshToken }, response.user)
      navigate(redirectTo)
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 429) {
          toast.error(t('login.error.rate_limit'))
          return
        }
        if (err.status === 401) {
          setGenericError(t('login.error.credentials'))
          return
        }
      }
      toast.error(t('login.error.generic'))
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-3xl font-bold text-text-primary">{t('login.title')}</h1>

      <GoogleSignInButton redirectTo={redirectTo} />

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        {genericError && (
          <p className="text-sm text-poppy-fg" role="alert">{genericError}</p>
        )}

        <FormField label={t('field.email')} required error={errors.email?.message} id="login-email">
          <Input
            id="login-email"
            type="email"
            autoComplete="email"
            placeholder={t('field.placeholder.email')}
            {...register('email', { required: t('login.error.credentials') })}
          />
        </FormField>

        <FormField label={t('field.password')} required error={errors.password?.message} id="login-password">
          <PasswordInput
            id="login-password"
            autoComplete="current-password"
            placeholder={t('field.placeholder.password')}
            {...register('password', { required: t('login.error.credentials') })}
          />
        </FormField>

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {isSubmitting ? t('login.submitting') : t('login.submit')}
        </Button>
      </form>

      <p className="text-center text-sm text-text-secondary">
        {t('login.no_account')}{' '}
        <Link to="/register" className="text-poppy hover:underline">
          {t('login.signup_link')}
        </Link>
      </p>
    </div>
  )
}
