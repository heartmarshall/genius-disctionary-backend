import { useState, useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuth } from '@/providers/AuthProvider'
import { registerUser, isApiError } from '@/lib/auth-api'
import { validateEmail, validateUsername, validatePassword } from '@/lib/validation'
import { GoogleSignInButton } from '@/components/auth/GoogleSignInButton'
import { FormField } from '@/components/common/FormField'
import { PasswordInput } from '@/components/common/PasswordInput'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import i18n from '@/i18n'

interface RegisterFormValues {
  email: string
  username: string
  password: string
  confirmPassword: string
}

export function RegisterPage() {
  const { t, i18n: { language } } = useTranslation('auth')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const redirectTo = searchParams.get('redirect') || '/dashboard'
  const auth = useAuth()

  // Store conflict field key so it re-translates on language change.
  const [conflictField, setConflictField] = useState<'email' | 'username' | null>(null)

  const {
    register,
    handleSubmit,
    setError,
    getValues,
    trigger,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({ mode: 'onBlur' })

  // Re-run client validation on language change so messages update.
  useEffect(() => {
    const touched = Object.keys(errors) as (keyof RegisterFormValues)[]
    if (touched.length > 0) {
      trigger(touched)
    }
  }, [language]) // eslint-disable-line react-hooks/exhaustive-deps

  const onSubmit = async (data: RegisterFormValues) => {
    setConflictField(null)
    try {
      const response = await registerUser(data.email, data.username, data.password)
      auth.login({ accessToken: response.accessToken, refreshToken: response.refreshToken }, response.user)
      navigate(redirectTo)
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 429) {
          toast.error(t('register.error.rate_limit'))
          return
        }
        if (err.status === 409) {
          if (err.field === 'email' || err.field === 'username') {
            setConflictField(err.field)
          } else {
            toast.error(t('register.error.already_exists'))
          }
          return
        }
        if (err.code === 'VALIDATION' && err.fields) {
          for (const fieldError of err.fields) {
            const fieldName = fieldError.field as keyof RegisterFormValues
            if (fieldName in data) {
              setError(fieldName, { message: fieldError.message })
            }
          }
          return
        }
      }
      toast.error(t('register.error.generic'))
    }
  }

  // Translate conflict error at render time (reacts to language change).
  const emailError = errors.email?.message
    || (conflictField === 'email'
      ? <>{t('register.error.email_exists')}{' '}<Link to="/login" className="text-poppy hover:underline">{t('register.error.email_exists_login')}</Link></>
      : undefined)
  const usernameError = errors.username?.message
    || (conflictField === 'username' ? t('register.error.username_exists') : undefined)

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-3xl font-bold text-text-primary">{t('register.title')}</h1>

      <GoogleSignInButton redirectTo={redirectTo} />

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <FormField label={t('field.email')} required error={emailError} id="register-email">
          <Input
            id="register-email"
            type="email"
            autoComplete="email"
            placeholder={t('field.placeholder.email')}
            {...register('email', { validate: validateEmail, onChange: () => setConflictField((v) => v === 'email' ? null : v) })}
          />
        </FormField>

        <FormField label={t('field.username')} required error={usernameError} id="register-username">
          <Input
            id="register-username"
            autoComplete="username"
            placeholder={t('field.placeholder.username')}
            {...register('username', { validate: validateUsername, onChange: () => setConflictField((v) => v === 'username' ? null : v) })}
          />
        </FormField>

        <FormField label={t('field.password')} required error={errors.password?.message} id="register-password">
          <PasswordInput
            id="register-password"
            autoComplete="new-password"
            placeholder={t('field.placeholder.password')}
            {...register('password', { validate: validatePassword })}
          />
        </FormField>

        <FormField label={t('field.confirm_password')} required error={errors.confirmPassword?.message} id="register-confirm-password">
          <PasswordInput
            id="register-confirm-password"
            autoComplete="new-password"
            placeholder={t('field.placeholder.confirm')}
            {...register('confirmPassword', {
              validate: (value) =>
                value === getValues('password') || i18n.t('password.mismatch', { ns: 'validation' }),
            })}
          />
        </FormField>

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {isSubmitting ? t('register.submitting') : t('register.submit')}
        </Button>
      </form>

      <p className="text-center text-sm text-text-secondary">
        {t('register.has_account')}{' '}
        <Link to="/login" className="text-poppy hover:underline">
          {t('register.login_link')}
        </Link>
      </p>
    </div>
  )
}
