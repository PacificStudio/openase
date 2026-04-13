import enTranslationsJson from './locales/en.json'
import zhTranslationsJson from './locales/zh.json'

export const SUPPORTED_LOCALES = ['en', 'zh'] as const

export type AppLocale = (typeof SUPPORTED_LOCALES)[number]
export type TranslationParams = Record<string, string | number>

const enTranslations = enTranslationsJson

export type TranslationKey = keyof typeof enTranslations

const zhTranslations = zhTranslationsJson satisfies Record<TranslationKey, string>

const translations = {
  en: enTranslations,
  zh: zhTranslations,
} as const satisfies Record<AppLocale, Record<TranslationKey, string>>

const DEFAULT_LOCALE: AppLocale = 'en'
const STORAGE_KEY = 'openase.locale'
const LOCALE_LABEL_KEYS = {
  en: 'locale.english',
  zh: 'locale.chinese',
} as const satisfies Record<AppLocale, TranslationKey>

function fillTemplate(template: string, params: TranslationParams = {}) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = params[key]
    return value == null ? match : String(value)
  })
}

export function parseLocale(input: string | null | undefined): AppLocale {
  const normalized = input?.trim().toLowerCase() ?? ''
  if (!normalized) {
    return DEFAULT_LOCALE
  }
  if (normalized.startsWith('zh')) {
    return 'zh'
  }
  if (normalized.startsWith('en')) {
    return 'en'
  }
  return DEFAULT_LOCALE
}

export function readStoredLocale(): AppLocale {
  if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
    return DEFAULT_LOCALE
  }
  return parseLocale(window.localStorage.getItem(STORAGE_KEY))
}

export function detectPreferredLocale(): AppLocale {
  if (typeof navigator === 'undefined') {
    return DEFAULT_LOCALE
  }
  return parseLocale(navigator.language)
}

export function resolveInitialLocale(): AppLocale {
  const stored = readStoredLocale()
  if (
    stored !== DEFAULT_LOCALE ||
    (typeof window !== 'undefined' && window.localStorage.getItem(STORAGE_KEY))
  ) {
    return stored
  }
  return detectPreferredLocale()
}

export function translate(locale: AppLocale, key: TranslationKey, params?: TranslationParams) {
  return fillTemplate(translations[locale][key], params)
}

export function localeLabel(locale: AppLocale, displayLocale: AppLocale) {
  return translate(displayLocale, LOCALE_LABEL_KEYS[locale])
}

export function setStoredLocale(locale: AppLocale) {
  if (typeof window === 'undefined' || typeof window.localStorage === 'undefined') {
    return
  }
  window.localStorage.setItem(STORAGE_KEY, locale)
}

export function pageTitle(title: string, locale: AppLocale) {
  return `${title} - ${translate(locale, 'common.pageTitleSuffix')}`
}
