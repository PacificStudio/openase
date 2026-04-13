import {
  localeLabel,
  parseLocale,
  resolveInitialLocale,
  setStoredLocale,
  translate,
  type AppLocale,
  type TranslationKey,
  type TranslationParams,
} from './index'

function createI18nStore() {
  let locale = $state<AppLocale>('en')
  let initialized = $state(false)

  return {
    get locale() {
      return locale
    },
    get initialized() {
      return initialized
    },
    init() {
      if (initialized) {
        return
      }
      locale = resolveInitialLocale()
      initialized = true
    },
    setLocale(nextLocale: string) {
      locale = parseLocale(nextLocale)
      initialized = true
      setStoredLocale(locale)
    },
    t(key: TranslationKey, params?: TranslationParams) {
      return translate(locale, key, params)
    },
    labelForLocale(targetLocale: AppLocale) {
      return localeLabel(targetLocale, locale)
    },
  }
}

export const i18nStore = createI18nStore()
