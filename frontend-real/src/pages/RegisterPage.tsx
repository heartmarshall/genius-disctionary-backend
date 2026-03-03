import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '@/providers/AuthProvider'
import { registerUser, isApiError } from '@/lib/auth-api'
import { validateEmail, validateUsername, validatePassword } from '@/lib/validation'
import { FormField } from '@/components/common/FormField'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface RegisterFormValues {
  email: string
  username: string
  password: string
  confirmPassword: string
}

export function RegisterPage() {
  const navigate = useNavigate()
  const auth = useAuth()

  const {
    register,
    handleSubmit,
    setError,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormValues>({ mode: 'onBlur' })

  const onSubmit = async (data: RegisterFormValues) => {
    try {
      const response = await registerUser(data.email, data.username, data.password)
      auth.login({ accessToken: response.accessToken, refreshToken: response.refreshToken }, response.user)
      navigate('/dashboard')
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 429) {
          toast.error('Слишком много попыток. Попробуй позже.')
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
      toast.error('Ошибка регистрации. Попробуй ещё раз.')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-3xl font-bold text-text-primary">Создать аккаунт</h1>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        <FormField label="Email" required error={errors.email?.message}>
          <Input
            type="email"
            placeholder="email@example.com"
            {...register('email', { validate: validateEmail })}
          />
        </FormField>

        <FormField label="Имя пользователя" required error={errors.username?.message}>
          <Input
            placeholder="username"
            {...register('username', { validate: validateUsername })}
          />
        </FormField>

        <FormField label="Пароль" required error={errors.password?.message}>
          <Input
            type="password"
            placeholder="Минимум 8 символов"
            {...register('password', { validate: validatePassword })}
          />
        </FormField>

        <FormField label="Подтверди пароль" required error={errors.confirmPassword?.message}>
          <Input
            type="password"
            placeholder="Повтори пароль"
            {...register('confirmPassword', {
              validate: (value) =>
                value === watch('password') || 'Пароли не совпадают',
            })}
          />
        </FormField>

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {isSubmitting ? 'Создаём аккаунт...' : 'Создать аккаунт'}
        </Button>
      </form>

      <p className="text-center text-sm text-text-secondary">
        Уже есть аккаунт?{' '}
        <Link to="/login" className="text-poppy hover:underline">
          Войти
        </Link>
      </p>
    </div>
  )
}
