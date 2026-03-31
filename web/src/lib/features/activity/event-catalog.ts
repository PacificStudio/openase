export type ActivityEventTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

export type ActivityEventCatalogEntry = {
  eventType: string
  label: string
  tone: ActivityEventTone
}

export const activityEventCatalog: ActivityEventCatalogEntry[] = [
  { eventType: 'ticket.created', label: 'Ticket created', tone: 'info' },
  { eventType: 'ticket.updated', label: 'Ticket updated', tone: 'info' },
  { eventType: 'ticket.status_changed', label: 'Ticket status changed', tone: 'info' },
  { eventType: 'ticket.completed', label: 'Ticket completed', tone: 'success' },
  { eventType: 'ticket.cancelled', label: 'Ticket cancelled', tone: 'warning' },
  { eventType: 'ticket.retry_scheduled', label: 'Ticket retry scheduled', tone: 'warning' },
  { eventType: 'ticket.retry_paused', label: 'Ticket retry paused', tone: 'warning' },
  { eventType: 'ticket.budget_exhausted', label: 'Ticket budget exhausted', tone: 'danger' },
  { eventType: 'agent.claimed', label: 'Agent claimed', tone: 'info' },
  { eventType: 'agent.launching', label: 'Agent launching', tone: 'info' },
  { eventType: 'agent.ready', label: 'Agent ready', tone: 'success' },
  { eventType: 'agent.paused', label: 'Agent paused', tone: 'warning' },
  { eventType: 'agent.failed', label: 'Agent failed', tone: 'danger' },
  { eventType: 'agent.completed', label: 'Agent completed', tone: 'success' },
  { eventType: 'agent.terminated', label: 'Agent terminated', tone: 'neutral' },
  { eventType: 'hook.started', label: 'Hook started', tone: 'info' },
  { eventType: 'hook.passed', label: 'Hook passed', tone: 'success' },
  { eventType: 'hook.failed', label: 'Hook failed', tone: 'danger' },
  { eventType: 'pr.opened', label: 'PR opened', tone: 'info' },
  { eventType: 'pr.merged', label: 'PR merged', tone: 'success' },
  { eventType: 'pr.closed', label: 'PR closed', tone: 'warning' },
]

const activityEventCatalogByType = new Map(
  activityEventCatalog.map(
    (item) => [item.eventType, item] satisfies [string, ActivityEventCatalogEntry],
  ),
)

export const activityEventFilterOptions = [
  { value: 'all', label: 'All events' },
  ...activityEventCatalog.map((item) => ({ value: item.eventType, label: item.label })),
]

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
  if (!normalized) return 'System activity'
  return normalized.charAt(0).toUpperCase() + normalized.slice(1)
}
