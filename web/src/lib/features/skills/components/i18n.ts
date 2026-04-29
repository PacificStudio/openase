import type { TranslationKey, TranslationParams } from '$lib/i18n/index'
import { i18nStore } from '$lib/i18n/store.svelte'

export function skillsT(key: TranslationKey, params?: TranslationParams) {
  return i18nStore.t(key, params)
}
