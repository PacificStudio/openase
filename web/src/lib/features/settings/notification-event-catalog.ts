import type { NotificationRuleEventType } from '$lib/api/contracts'
import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'
import {
  DEFAULT_TEMPLATE_VARIABLE_GROUPS,
  SEVERITY_LABEL_KEYS,
  TEMPLATE_VARIABLE_GROUP_DEFINITIONS,
  type TemplateVariableGroupDefinition,
} from './notification-event-catalog-data'
import type { EventSeverity } from './notification-event-catalog-data'

export type { EventSeverity } from './notification-event-catalog-data'

function translateRaw(key: TranslationKey) {
  return i18nStore.t(key)
}

export type EventGroup = {
  key: string
  label: string
  events: CatalogEvent[]
}

export type CatalogEvent = {
  eventType: string
  label: string
  defaultTemplate: string
  severity: EventSeverity
  groupKey: string
}

export type TemplateVariable = {
  name: string
  description: string
}

export type TemplateVariableGroup = {
  label: string
  variables: TemplateVariable[]
}

function resolveTemplateVariableGroup(
  group: TemplateVariableGroupDefinition,
): TemplateVariableGroup {
  return {
    label: translateRaw(group.labelKey),
    variables: group.variables.map((variable) => ({
      name: variable.name,
      description: translateRaw(variable.descriptionKey),
    })),
  }
}

export function getTemplateVariables(eventType: string): TemplateVariableGroup[] {
  const definitions =
    TEMPLATE_VARIABLE_GROUP_DEFINITIONS[eventType] ?? DEFAULT_TEMPLATE_VARIABLE_GROUPS
  return definitions.map(resolveTemplateVariableGroup)
}

const EVENT_GROUP_OTHER_KEY: TranslationKey = 'settings.notification.eventGroup.other'
function normalizeSeverity(level: string | undefined): EventSeverity {
  if (level === 'critical' || level === 'warning' || level === 'info') return level
  return 'info'
}

function normalizeGroupKey(group: string): string {
  return group.toLowerCase().replace(/\s+/g, '_')
}

function buildCatalogEvent(et: NotificationRuleEventType): CatalogEvent {
  const groupLabel = et.group || translateRaw(EVENT_GROUP_OTHER_KEY)
  return {
    eventType: et.event_type,
    label: et.label,
    defaultTemplate: et.default_template,
    severity: normalizeSeverity(et.level),
    groupKey: normalizeGroupKey(groupLabel),
  }
}

export function buildEventCatalog(eventTypes: NotificationRuleEventType[]): EventGroup[] {
  const groupMap = new Map<string, { label: string; events: CatalogEvent[] }>()

  for (const et of eventTypes) {
    const catalogEvent = buildCatalogEvent(et)
    const groupLabel = et.group || translateRaw(EVENT_GROUP_OTHER_KEY)
    const groupKey = catalogEvent.groupKey

    const existing = groupMap.get(groupKey)
    if (existing) {
      existing.events.push(catalogEvent)
    } else {
      groupMap.set(groupKey, { label: groupLabel, events: [catalogEvent] })
    }
  }

  return Array.from(groupMap.entries()).map(([key, { label, events }]) => ({
    key,
    label,
    events,
  }))
}

export function getSeverity(
  eventType: string,
  eventTypes: NotificationRuleEventType[],
): EventSeverity {
  const et = eventTypes.find((e) => e.event_type === eventType)
  return normalizeSeverity(et?.level)
}

export function severityLabel(severity: EventSeverity): string {
  return translateRaw(SEVERITY_LABEL_KEYS[severity])
}
