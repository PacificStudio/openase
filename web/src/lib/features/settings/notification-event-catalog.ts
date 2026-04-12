import type { NotificationRuleEventType } from '$lib/api/contracts'
import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'

function translateRaw(key: TranslationKey) {
  return i18nStore.t(key)
}

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

type TemplateVariableDefinition = {
  name: string
  descriptionKey: TranslationKey
}

type TemplateVariableGroupDefinition = {
  labelKey: TranslationKey
  variables: TemplateVariableDefinition[]
}

const COMMON_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'event_type', descriptionKey: 'settings.notification.variable.eventType' },
  { name: 'project_id', descriptionKey: 'settings.notification.variable.projectId' },
  { name: 'published_at', descriptionKey: 'settings.notification.variable.publishedAt' },
]

const TICKET_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'ticket.identifier', descriptionKey: 'settings.notification.variable.ticketIdentifierExample' },
  { name: 'ticket.title', descriptionKey: 'settings.notification.variable.ticketTitle' },
  { name: 'ticket.status_name', descriptionKey: 'settings.notification.variable.currentStatusName' },
  { name: 'ticket.priority', descriptionKey: 'settings.notification.variable.priorityLevel' },
]

const TICKET_SHORTHAND_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'identifier', descriptionKey: 'settings.notification.variable.ticketIdentifierShorthand' },
  { name: 'title', descriptionKey: 'settings.notification.variable.ticketTitleShorthand' },
  { name: 'status_name', descriptionKey: 'settings.notification.variable.statusNameShorthand' },
  { name: 'priority', descriptionKey: 'settings.notification.variable.priorityShorthand' },
]

const HOOK_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'ticket_identifier', descriptionKey: 'settings.notification.variable.ticketIdentifier' },
  { name: 'hook_name', descriptionKey: 'settings.notification.variable.hookName' },
  { name: 'message', descriptionKey: 'settings.notification.variable.eventMessage' },
]

const PR_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'ticket_identifier', descriptionKey: 'settings.notification.variable.ticketIdentifier' },
  { name: 'pull_request_url', descriptionKey: 'settings.notification.variable.pullRequestUrl' },
  { name: 'message', descriptionKey: 'settings.notification.variable.eventMessage' },
]

const MACHINE_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'machine_id', descriptionKey: 'settings.notification.variable.machineId' },
  { name: 'session_id', descriptionKey: 'settings.notification.variable.sessionId' },
  { name: 'transport_mode', descriptionKey: 'settings.notification.variable.transportMode' },
  { name: 'connection_mode', descriptionKey: 'settings.notification.variable.connectionMode' },
]

const AGENT_VARIABLES: TemplateVariableDefinition[] = [
  { name: 'agent.name', descriptionKey: 'settings.notification.variable.agentName' },
  { name: 'agent.status', descriptionKey: 'settings.notification.variable.agentStatus' },
  { name: 'current_ticket_id', descriptionKey: 'settings.notification.variable.currentTicketId' },
  { name: 'ticket_id', descriptionKey: 'settings.notification.variable.ticketIdAlias' },
  { name: 'name', descriptionKey: 'settings.notification.variable.agentNameShorthand' },
  { name: 'status', descriptionKey: 'settings.notification.variable.agentStatusShorthand' },
]

const TEMPLATE_VARIABLE_GROUP_DEFINITIONS: Record<string, TemplateVariableGroupDefinition[]> = {
  'ticket.created': [
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.updated': [
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.status_changed': [
    {
      labelKey: 'settings.notification.variableGroup.ticket',
      variables: [
        ...TICKET_VARIABLES,
        { name: 'new_status', descriptionKey: 'settings.notification.variable.newStatus' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.completed': [
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.cancelled': [
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.retry_scheduled': [
    {
      labelKey: 'settings.notification.variableGroup.event',
      variables: [
        { name: 'next_retry_at', descriptionKey: 'settings.notification.variable.nextRetryAt' },
        { name: 'consecutive_errors', descriptionKey: 'settings.notification.variable.consecutiveErrors' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.retry_resumed': [
    {
      labelKey: 'settings.notification.variableGroup.event',
      variables: [{ name: 'pause_reason', descriptionKey: 'settings.notification.variable.previousRetryPauseReason' }],
    },
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.retry_paused': [
    {
      labelKey: 'settings.notification.variableGroup.event',
      variables: [
        { name: 'message', descriptionKey: 'settings.notification.variable.eventMessage' },
        { name: 'pause_reason', descriptionKey: 'settings.notification.variable.pauseReason' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'ticket.budget_exhausted': [
    {
      labelKey: 'settings.notification.variableGroup.event',
      variables: [
        { name: 'budget_usd', descriptionKey: 'settings.notification.variable.budgetUsd' },
        { name: 'cost_amount', descriptionKey: 'settings.notification.variable.costAmount' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.ticket', variables: TICKET_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.shorthands', variables: TICKET_SHORTHAND_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'agent.claimed': [
    { labelKey: 'settings.notification.variableGroup.agent', variables: AGENT_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'agent.failed': [
    { labelKey: 'settings.notification.variableGroup.agent', variables: AGENT_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'hook.passed': [
    { labelKey: 'settings.notification.variableGroup.hook', variables: HOOK_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'hook.failed': [
    {
      labelKey: 'settings.notification.variableGroup.hook',
      variables: [
        ...HOOK_VARIABLES,
        { name: 'error', descriptionKey: 'settings.notification.variable.errorMessage' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'pr.opened': [
    { labelKey: 'settings.notification.variableGroup.pullRequest', variables: PR_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'pr.closed': [
    { labelKey: 'settings.notification.variableGroup.pullRequest', variables: PR_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'machine.connected': [
    { labelKey: 'settings.notification.variableGroup.machine', variables: MACHINE_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'machine.reconnected': [
    { labelKey: 'settings.notification.variableGroup.machine', variables: MACHINE_VARIABLES },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'machine.disconnected': [
    {
      labelKey: 'settings.notification.variableGroup.machine',
      variables: [
        ...MACHINE_VARIABLES,
        { name: 'reason', descriptionKey: 'settings.notification.variable.disconnectReason' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
  'machine.daemon_auth_failed': [
    {
      labelKey: 'settings.notification.variableGroup.machine',
      variables: [
        ...MACHINE_VARIABLES,
        { name: 'failure_code', descriptionKey: 'settings.notification.variable.failureCode' },
        { name: 'error', descriptionKey: 'settings.notification.variable.errorMessage' },
      ],
    },
    { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
  ],
}

const DEFAULT_TEMPLATE_VARIABLE_GROUPS: TemplateVariableGroupDefinition[] = [
  { labelKey: 'settings.notification.variableGroup.common', variables: COMMON_VARIABLES },
]

function resolveTemplateVariableGroup(group: TemplateVariableGroupDefinition): TemplateVariableGroup {
  return {
    label: translateRaw(group.labelKey),
    variables: group.variables.map((variable) => ({
      name: variable.name,
      description: translateRaw(variable.descriptionKey),
    })),
  }
}

export function getTemplateVariables(eventType: string): TemplateVariableGroup[] {
  const definitions = TEMPLATE_VARIABLE_GROUP_DEFINITIONS[eventType] ?? DEFAULT_TEMPLATE_VARIABLE_GROUPS
  return definitions.map(resolveTemplateVariableGroup)
}

const EVENT_GROUP_OTHER_KEY: TranslationKey = 'settings.notification.eventGroup.other'
const SEVERITY_LABEL_KEYS: Record<EventSeverity, TranslationKey> = {
  critical: 'settings.notification.severity.critical',
  info: 'settings.notification.severity.info',
  warning: 'settings.notification.severity.warning',
}

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
