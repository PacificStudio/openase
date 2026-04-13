import type { TranslationKey } from '$lib/i18n'

export type ActivityEventTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

export type ActivityEventCatalogDefinition = {
  eventType: string
  labelKey: TranslationKey
  tone: ActivityEventTone
}
