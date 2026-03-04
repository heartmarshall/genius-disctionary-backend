import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

import enAuth from './locales/en/auth.json'
import enValidation from './locales/en/validation.json'
import ruAuth from './locales/ru/auth.json'
import ruValidation from './locales/ru/validation.json'

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { auth: enAuth, validation: enValidation },
      ru: { auth: ruAuth, validation: ruValidation },
    },
    fallbackLng: 'en',
    defaultNS: 'auth',
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'i18nextLng',
      caches: ['localStorage'],
    },
  })

export default i18n
