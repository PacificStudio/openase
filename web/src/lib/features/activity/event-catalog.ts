import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'
import { activityEventDefinitions } from './event-catalog-definitions'
import type { ActivityEventTone } from './event-catalog-types'

export type { ActivityEventTone } from './event-catalog-types'

export type ActivityEventCatalogEntry = {
  eventType: string
  label: string
  tone: ActivityEventTone
}

function translateRaw(key: TranslationKey) {
  return i18nStore.t(key)
}

export const activityEventCatalog: ActivityEventCatalogEntry[] = activityEventDefinitions.map(
  (definition) => ({
    eventType: definition.eventType,
    tone: definition.tone,
    get label() {
      return translateRaw(definition.labelKey)
    },
  }),
)

const activityEventCatalogByType = new Map(
  activityEventCatalog.map(
    (item) => [item.eventType, item] satisfies [string, ActivityEventCatalogEntry],
  ),
)

type ActivityEventFilterDefinition = {
  value: string
  labelKey: TranslationKey
}

const activityEventFilterDefinitions: ActivityEventFilterDefinition[] = [
  { value: 'all', labelKey: 'activityEvent.filter.all' },
  ...activityEventDefinitions.map((definition) => ({
    value: definition.eventType,
    labelKey: definition.labelKey,
  })),
]

export const activityEventFilterOptions = activityEventFilterDefinitions.map((definition) => ({
  value: definition.value,
  get label() {
    return translateRaw(definition.labelKey)
  },
}))

export function getActivityEventCatalogEntry(eventType: string) {
  return activityEventCatalogByType.get(eventType)
}

export function activityEventLabel(eventType: string) {
  return getActivityEventCatalogEntry(eventType)?.label ?? humanizeActivityEventType(eventType)
}

export function activityEventTone(eventType: string): ActivityEventTone {
  return getActivityEventCatalogEntry(eventType)?.tone ?? 'neutral'
}

export function isActivityExceptionEvent(eventType: string) {
  return (
    eventType === 'hook.failed' ||
    eventType === 'ticket.retry_paused' ||
    eventType === 'ticket.budget_exhausted' ||
    eventType === 'agent.failed'
  )
}

export function isHookActivityEventType(eventType: string) {
  return eventType === 'hook.started' || eventType === 'hook.passed' || eventType === 'hook.failed'
}

function humanizeActivityEventType(value: string) {
  const normalized = value.replace(/[._]+/g, ' ').trim()
  if (!normalized) return translateRaw('activityEvent.fallback.systemActivity')
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
}
