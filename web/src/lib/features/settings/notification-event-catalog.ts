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

export type TemplateVariable = {
  name: string
  description: string
}

export type TemplateVariableGroup = {
  label: string
  variables: TemplateVariable[]
}

const COMMON_VARS: TemplateVariable[] = [
  { name: 'event_type', description: 'Event type identifier' },
  { name: 'project_id', description: 'Project UUID' },
  { name: 'published_at', description: 'Event timestamp (RFC3339)' },
]

const TICKET_VARS: TemplateVariable[] = [
  { name: 'ticket.identifier', description: 'Ticket identifier (e.g. PROJ-42)' },
  { name: 'ticket.title', description: 'Ticket title' },
  { name: 'ticket.status_name', description: 'Current status name' },
  { name: 'ticket.priority', description: 'Priority level' },
]

const TICKET_SHORTHAND_VARS: TemplateVariable[] = [
  { name: 'identifier', description: 'Shorthand: ticket identifier' },
  { name: 'title', description: 'Shorthand: ticket title' },
  { name: 'status_name', description: 'Shorthand: status name' },
  { name: 'priority', description: 'Shorthand: priority' },
]

const HOOK_VARS: TemplateVariable[] = [
  { name: 'ticket_identifier', description: 'Ticket identifier' },
  { name: 'hook_name', description: 'Name of the hook' },
  { name: 'message', description: 'Event message' },
]

const PR_VARS: TemplateVariable[] = [
  { name: 'ticket_identifier', description: 'Ticket identifier' },
  { name: 'pull_request_url', description: 'Pull request URL' },
  { name: 'message', description: 'Event message' },
]

const MACHINE_VARS: TemplateVariable[] = [
  { name: 'machine_id', description: 'Machine UUID' },
  { name: 'session_id', description: 'Reverse websocket session ID' },
  { name: 'transport_mode', description: 'Transport mode, for example ws_reverse' },
  { name: 'connection_mode', description: 'Connection mode, for example reverse_websocket' },
]

const AGENT_VARS: TemplateVariable[] = [
  { name: 'agent.name', description: 'Agent name' },
  { name: 'agent.status', description: 'Agent status' },
  { name: 'current_ticket_id', description: 'ID of the assigned ticket' },
  { name: 'ticket_id', description: 'Alias: current ticket ID' },
  { name: 'name', description: 'Shorthand: agent name' },
  { name: 'status', description: 'Shorthand: agent status' },
]

const TEMPLATE_VARS: Record<string, TemplateVariableGroup[]> = {
  'ticket.created': [
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.updated': [
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.status_changed': [
    {
      label: 'Ticket',
      variables: [...TICKET_VARS, { name: 'new_status', description: 'New status after change' }],
    },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.completed': [
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.cancelled': [
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.retry_scheduled': [
    {
      label: 'Event',
      variables: [
        { name: 'next_retry_at', description: 'Scheduled retry timestamp (RFC3339)' },
        { name: 'consecutive_errors', description: 'Current consecutive failure count' },
      ],
    },
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.retry_resumed': [
    {
      label: 'Event',
      variables: [{ name: 'pause_reason', description: 'Previous retry pause reason' }],
    },
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.retry_paused': [
    {
      label: 'Event',
      variables: [
        { name: 'message', description: 'Event message' },
        { name: 'pause_reason', description: 'Reason pausing retry' },
      ],
    },
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'ticket.budget_exhausted': [
    {
      label: 'Event',
      variables: [
        { name: 'budget_usd', description: 'Configured ticket budget in USD' },
        { name: 'cost_amount', description: 'Observed ticket cost in USD' },
      ],
    },
    { label: 'Ticket', variables: TICKET_VARS },
    { label: 'Shorthands', variables: TICKET_SHORTHAND_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'agent.claimed': [
    { label: 'Agent', variables: AGENT_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'agent.failed': [
    { label: 'Agent', variables: AGENT_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'hook.passed': [
    { label: 'Hook', variables: HOOK_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'hook.failed': [
    {
      label: 'Hook',
      variables: [...HOOK_VARS, { name: 'error', description: 'Error message' }],
    },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'pr.opened': [
    { label: 'Pull Request', variables: PR_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'pr.closed': [
    { label: 'Pull Request', variables: PR_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'machine.connected': [
    { label: 'Machine', variables: MACHINE_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'machine.reconnected': [
    { label: 'Machine', variables: MACHINE_VARS },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'machine.disconnected': [
    {
      label: 'Machine',
      variables: [
        ...MACHINE_VARS,
        { name: 'reason', description: 'Disconnect reason when present' },
      ],
    },
    { label: 'Common', variables: COMMON_VARS },
  ],
  'machine.daemon_auth_failed': [
    {
      label: 'Machine',
      variables: [
        ...MACHINE_VARS,
        { name: 'failure_code', description: 'Structured auth failure code' },
        { name: 'error', description: 'Underlying auth error message' },
      ],
    },
    { label: 'Common', variables: COMMON_VARS },
  ],
}

export function getTemplateVariables(eventType: string): TemplateVariableGroup[] {
  return TEMPLATE_VARS[eventType] ?? [{ label: 'Common', variables: COMMON_VARS }]
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

export function getSeverity(
  eventType: string,
  eventTypes: NotificationRuleEventType[],
): EventSeverity {
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
