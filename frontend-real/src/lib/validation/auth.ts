// Validation rules matching backend: docs/business/BUSINESS_RULES.md

const EMAIL_MAX_LENGTH = 254
const USERNAME_MIN_LENGTH = 2
const USERNAME_MAX_LENGTH = 50
const PASSWORD_MIN_LENGTH = 8
const PASSWORD_MAX_LENGTH = 72

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function validateEmail(value: string): string | null {
  const trimmed = value.trim()
  if (!trimmed) return 'Email is required.'
  if (trimmed.length > EMAIL_MAX_LENGTH) return `Email must be at most ${EMAIL_MAX_LENGTH} characters.`
  if (!EMAIL_REGEX.test(trimmed)) return 'Please enter a valid email address.'
  return null
}

export function validateUsername(value: string): string | null {
  if (!value) return 'Username is required.'
  if (value.length < USERNAME_MIN_LENGTH)
    return `Username must be at least ${USERNAME_MIN_LENGTH} characters.`
  if (value.length > USERNAME_MAX_LENGTH)
    return `Username must be at most ${USERNAME_MAX_LENGTH} characters.`
  return null
}

export function validatePassword(value: string): string | null {
  if (!value) return 'Password is required.'
  if (value.length < PASSWORD_MIN_LENGTH)
    return `Password must be at least ${PASSWORD_MIN_LENGTH} characters.`
  if (value.length > PASSWORD_MAX_LENGTH)
    return `Password must be at most ${PASSWORD_MAX_LENGTH} characters.`
  return null
}

// ── react-hook-form integration ──

export interface RegisterFormValues {
  email: string
  username: string
  password: string
  confirmPassword: string
}

export interface LoginFormValues {
  email: string
  password: string
}

export function registerResolver(values: RegisterFormValues) {
  const errors: Record<string, { message: string }> = {}

  const emailError = validateEmail(values.email)
  if (emailError) errors.email = { message: emailError }

  const usernameError = validateUsername(values.username)
  if (usernameError) errors.username = { message: usernameError }

  const passwordError = validatePassword(values.password)
  if (passwordError) errors.password = { message: passwordError }

  if (!values.confirmPassword) {
    errors.confirmPassword = { message: 'Please confirm your password.' }
  } else if (values.password && values.confirmPassword !== values.password) {
    errors.confirmPassword = { message: 'Passwords do not match.' }
  }

  return {
    values: Object.keys(errors).length === 0 ? values : {},
    errors,
  }
}

export function loginResolver(values: LoginFormValues) {
  const errors: Record<string, { message: string }> = {}

  if (!values.email.trim()) errors.email = { message: 'Email is required.' }
  if (!values.password) errors.password = { message: 'Password is required.' }

  return {
    values: Object.keys(errors).length === 0 ? values : {},
    errors,
  }
}
