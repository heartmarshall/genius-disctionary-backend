const EMAIL_REGEX = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

export function validateEmail(value: string): string | true {
  if (!value.trim()) return 'Введи email'
  if (!EMAIL_REGEX.test(value)) return 'Некорректный формат email'
  if (value.length > 254) return 'Email слишком длинный (макс. 254 символа)'
  return true
}

export function validateUsername(value: string): string | true {
  if (!value.trim()) return 'Введи имя пользователя'
  if (value.trim().length < 2) return 'Минимум 2 символа'
  if (value.trim().length > 50) return 'Максимум 50 символов'
  return true
}

export function validatePassword(value: string): string | true {
  if (!value) return 'Введи пароль'
  if (value.length < 8) return 'Минимум 8 символов'
  if (value.length > 72) return 'Максимум 72 символа'
  return true
}
