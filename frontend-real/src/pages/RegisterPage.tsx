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
  const { t } = useTranslation('auth')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const redirectTo = searchParams.get('redirect') || '/dashboard'
  const auth = useAuth()

  const {
    register,
    handleSubmit,
    setError,
    getValues,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({ mode: 'onBlur' })

  const onSubmit = async (data: RegisterFormValues) => {
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

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-3xl font-bold text-text-primary">{t('register.title')}</h1>

      <GoogleSignInButton redirectTo={redirectTo} />

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <FormField label={t('field.email')} required error={errors.email?.message} id="register-email">
          <Input
            id="register-email"
            type="email"
            autoComplete="email"
            placeholder={t('field.placeholder.email')}
            {...register('email', { validate: validateEmail })}
          />
        </FormField>

        <FormField label={t('field.username')} required error={errors.username?.message} id="register-username">
          <Input
            id="register-username"
            autoComplete="username"
            placeholder={t('field.placeholder.username')}
            {...register('username', { validate: validateUsername })}
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
