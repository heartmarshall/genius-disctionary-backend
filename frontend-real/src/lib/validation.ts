import i18n from '@/i18n'

const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function validateEmail(value: string): string | true {
  const t = i18n.getFixedT(null, 'validation')
  if (!value.trim()) return t('email.required')
  if (value.length > 254) return t('email.too_long')
  if (!EMAIL_REGEX.test(value)) return t('email.invalid')
  return true
}

export function validateUsername(value: string): string | true {
  const t = i18n.getFixedT(null, 'validation')
  if (!value.trim()) return t('username.required')
  if (value.trim().length < 2) return t('username.too_short')
  if (value.trim().length > 50) return t('username.too_long')
  return true
}

export function validatePassword(value: string): string | true {
  const t = i18n.getFixedT(null, 'validation')
  if (!value) return t('password.required')
  if (value.length < 8) return t('password.too_short')
  if (value.length > 72) return t('password.too_long')
  return true
}
