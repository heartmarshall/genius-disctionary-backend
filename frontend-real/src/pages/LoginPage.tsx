import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate, Link } from 'react-router-dom'
import { toast } from 'sonner'
import { useAuth } from '@/providers/AuthProvider'
import { loginPassword, isApiError } from '@/lib/auth-api'
import { FormField } from '@/components/common/FormField'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'

interface LoginFormValues {
  email: string
  password: string
}

export function LoginPage() {
  const navigate = useNavigate()
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
      navigate('/dashboard')
    } catch (err) {
      if (isApiError(err)) {
        if (err.status === 429) {
          toast.error('Слишком много попыток. Попробуй позже.')
          return
        }
        if (err.status === 401) {
          setGenericError('Неверный email или пароль')
          return
        }
      }
      toast.error('Ошибка входа. Попробуй ещё раз.')
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-3xl font-bold text-text-primary">Войти</h1>

      <form onSubmit={handleSubmit(onSubmit)} className="flex flex-col gap-4">
        {genericError && (
          <p className="text-sm text-poppy-fg" role="alert">{genericError}</p>
        )}

        <FormField label="Email" required error={errors.email?.message} id="login-email">
          <Input
            id="login-email"
            type="email"
            autoComplete="email"
            placeholder="email@example.com"
            {...register('email', { required: 'Введи email' })}
          />
        </FormField>

        <FormField label="Пароль" required error={errors.password?.message} id="login-password">
          <Input
            id="login-password"
            type="password"
            autoComplete="current-password"
            placeholder="Пароль"
            {...register('password', { required: 'Введи пароль' })}
          />
        </FormField>

        <Button type="submit" disabled={isSubmitting} className="w-full">
          {isSubmitting ? 'Входим...' : 'Войти'}
        </Button>
      </form>

      <p className="text-center text-sm text-text-secondary">
        Нет аккаунта?{' '}
        <Link to="/register" className="text-poppy hover:underline">
          Создать
        </Link>
      </p>
    </div>
  )
}
