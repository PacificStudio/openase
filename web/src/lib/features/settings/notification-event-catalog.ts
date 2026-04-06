import type { NotificationRuleEventType } from '$lib/api/contracts'

export type EventSeverity = 'info' | 'warning' | 'critical'

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

function normalizeSeverity(level: string | undefined): EventSeverity {
  if (level === 'critical' || level === 'warning' || level === 'info') return level
  return 'info'
}

function normalizeGroupKey(group: string): string {
  return group.toLowerCase().replace(/\s+/g, '_')
}

function buildCatalogEvent(et: NotificationRuleEventType): CatalogEvent {
  const groupLabel = et.group || 'Other'
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
    const groupLabel = et.group || 'Other'
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

export function getSeverity(eventType: string, eventTypes: NotificationRuleEventType[]): EventSeverity {
  const et = eventTypes.find((e) => e.event_type === eventType)
  return normalizeSeverity(et?.level)
}

export function severityLabel(severity: EventSeverity): string {
  switch (severity) {
    case 'info':
      return 'Info'
    case 'warning':
      return 'Warning'
    case 'critical':
      return 'Critical'
  }
}
