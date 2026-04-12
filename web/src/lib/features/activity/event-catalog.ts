import type { TranslationKey } from '$lib/i18n'
import { i18nStore } from '$lib/i18n/store.svelte'

export type ActivityEventTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

export type ActivityEventCatalogEntry = {
  eventType: string
  label: string
  tone: ActivityEventTone
}

type ActivityEventCatalogDefinition = {
  eventType: string
  labelKey: TranslationKey
  tone: ActivityEventTone
}

function translateRaw(key: TranslationKey) {
  return i18nStore.t(key)
}

const activityEventDefinitions: ActivityEventCatalogDefinition[] = [
  { eventType: 'project.created', labelKey: 'activityEvent.label.project.created', tone: 'info' },
  { eventType: 'project.updated', labelKey: 'activityEvent.label.project.updated', tone: 'info' },
  { eventType: 'project.archived', labelKey: 'activityEvent.label.project.archived', tone: 'warning' },
  {
    eventType: 'project.status_changed',
    labelKey: 'activityEvent.label.project.status_changed',
    tone: 'info',
  },
  {
    eventType: 'project.project_ai_retention_updated',
    labelKey: 'activityEvent.label.project.project_ai_retention_updated',
    tone: 'info',
  },
  {
    eventType: 'project.default_provider_changed',
    labelKey: 'activityEvent.label.project.default_provider_changed',
    tone: 'info',
  },
  {
    eventType: 'project.concurrency_changed',
    labelKey: 'activityEvent.label.project.concurrency_changed',
    tone: 'info',
  },
  {
    eventType: 'project_repo.created',
    labelKey: 'activityEvent.label.project_repo.created',
    tone: 'info',
  },
  {
    eventType: 'project_repo.updated',
    labelKey: 'activityEvent.label.project_repo.updated',
    tone: 'info',
  },
  {
    eventType: 'project_repo.deleted',
    labelKey: 'activityEvent.label.project_repo.deleted',
    tone: 'warning',
  },
  {
    eventType: 'ticket_status.created',
    labelKey: 'activityEvent.label.ticket_status.created',
    tone: 'info',
  },
  {
    eventType: 'ticket_status.updated',
    labelKey: 'activityEvent.label.ticket_status.updated',
    tone: 'info',
  },
  {
    eventType: 'ticket_status.reordered',
    labelKey: 'activityEvent.label.ticket_status.reordered',
    tone: 'info',
  },
  {
    eventType: 'ticket_status.concurrency_changed',
    labelKey: 'activityEvent.label.ticket_status.concurrency_changed',
    tone: 'info',
  },
  {
    eventType: 'ticket_status.deleted',
    labelKey: 'activityEvent.label.ticket_status.deleted',
    tone: 'warning',
  },
  {
    eventType: 'ticket_status.reset',
    labelKey: 'activityEvent.label.ticket_status.reset',
    tone: 'warning',
  },
  {
    eventType: 'workflow.created',
    labelKey: 'activityEvent.label.workflow.created',
    tone: 'info',
  },
  {
    eventType: 'workflow.updated',
    labelKey: 'activityEvent.label.workflow.updated',
    tone: 'info',
  },
  {
    eventType: 'workflow.activated',
    labelKey: 'activityEvent.label.workflow.activated',
    tone: 'success',
  },
  {
    eventType: 'workflow.deactivated',
    labelKey: 'activityEvent.label.workflow.deactivated',
    tone: 'warning',
  },
  {
    eventType: 'workflow.deleted',
    labelKey: 'activityEvent.label.workflow.deleted',
    tone: 'warning',
  },
  {
    eventType: 'workflow.harness_updated',
    labelKey: 'activityEvent.label.workflow.harness_updated',
    tone: 'info',
  },
  {
    eventType: 'workflow.hooks_updated',
    labelKey: 'activityEvent.label.workflow.hooks_updated',
    tone: 'info',
  },
  {
    eventType: 'workflow.agent_changed',
    labelKey: 'activityEvent.label.workflow.agent_changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.pickup_statuses_changed',
    labelKey: 'activityEvent.label.workflow.pickup_statuses_changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.finish_statuses_changed',
    labelKey: 'activityEvent.label.workflow.finish_statuses_changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.concurrency_changed',
    labelKey: 'activityEvent.label.workflow.concurrency_changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.retry_policy_changed',
    labelKey: 'activityEvent.label.workflow.retry_policy_changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.timeout_changed',
    labelKey: 'activityEvent.label.workflow.timeout_changed',
    tone: 'info',
  },
  {
    eventType: 'provider.created',
    labelKey: 'activityEvent.label.provider.created',
    tone: 'info',
  },
  {
    eventType: 'provider.updated',
    labelKey: 'activityEvent.label.provider.updated',
    tone: 'info',
  },
  {
    eventType: 'provider.availability_changed',
    labelKey: 'activityEvent.label.provider.availability_changed',
    tone: 'warning',
  },
  {
    eventType: 'provider.machine_binding_changed',
    labelKey: 'activityEvent.label.provider.machine_binding_changed',
    tone: 'info',
  },
  {
    eventType: 'provider.rate_limit_updated',
    labelKey: 'activityEvent.label.provider.rate_limit_updated',
    tone: 'info',
  },
  {
    eventType: 'secret.created',
    labelKey: 'activityEvent.label.secret.created',
    tone: 'info',
  },
  {
    eventType: 'secret.rotated',
    labelKey: 'activityEvent.label.secret.rotated',
    tone: 'warning',
  },
  {
    eventType: 'secret.bound',
    labelKey: 'activityEvent.label.secret.bound',
    tone: 'info',
  },
  {
    eventType: 'secret.unbound',
    labelKey: 'activityEvent.label.secret.unbound',
    tone: 'warning',
  },
  {
    eventType: 'secret.disabled',
    labelKey: 'activityEvent.label.secret.disabled',
    tone: 'warning',
  },
  {
    eventType: 'secret.deleted',
    labelKey: 'activityEvent.label.secret.deleted',
    tone: 'danger',
  },
  {
    eventType: 'agent.created',
    labelKey: 'activityEvent.label.agent.created',
    tone: 'info',
  },
  {
    eventType: 'agent.updated',
    labelKey: 'activityEvent.label.agent.updated',
    tone: 'info',
  },
  {
    eventType: 'agent.resumed',
    labelKey: 'activityEvent.label.agent.resumed',
    tone: 'success',
  },
  {
    eventType: 'agent.deleted',
    labelKey: 'activityEvent.label.agent.deleted',
    tone: 'warning',
  },
  {
    eventType: 'scheduled_job.created',
    labelKey: 'activityEvent.label.scheduled_job.created',
    tone: 'info',
  },
  {
    eventType: 'scheduled_job.updated',
    labelKey: 'activityEvent.label.scheduled_job.updated',
    tone: 'info',
  },
  {
    eventType: 'scheduled_job.enabled',
    labelKey: 'activityEvent.label.scheduled_job.enabled',
    tone: 'success',
  },
  {
    eventType: 'scheduled_job.disabled',
    labelKey: 'activityEvent.label.scheduled_job.disabled',
    tone: 'warning',
  },
  {
    eventType: 'scheduled_job.deleted',
    labelKey: 'activityEvent.label.scheduled_job.deleted',
    tone: 'warning',
  },
  {
    eventType: 'scheduled_job.triggered',
    labelKey: 'activityEvent.label.scheduled_job.triggered',
    tone: 'info',
  },
  {
    eventType: 'ticket_comment.created',
    labelKey: 'activityEvent.label.ticket_comment.created',
    tone: 'info',
  },
  {
    eventType: 'ticket_comment.edited',
    labelKey: 'activityEvent.label.ticket_comment.edited',
    tone: 'info',
  },
  {
    eventType: 'ticket_comment.deleted',
    labelKey: 'activityEvent.label.ticket_comment.deleted',
    tone: 'warning',
  },
  {
    eventType: 'project_update_thread.created',
    labelKey: 'activityEvent.label.project_update_thread.created',
    tone: 'info',
  },
  {
    eventType: 'project_update_thread.edited',
    labelKey: 'activityEvent.label.project_update_thread.edited',
    tone: 'info',
  },
  {
    eventType: 'project_update_thread.deleted',
    labelKey: 'activityEvent.label.project_update_thread.deleted',
    tone: 'warning',
  },
  {
    eventType: 'project_update_thread.status_changed',
    labelKey: 'activityEvent.label.project_update_thread.status_changed',
    tone: 'warning',
  },
  {
    eventType: 'project_update_comment.created',
    labelKey: 'activityEvent.label.project_update_comment.created',
    tone: 'info',
  },
  {
    eventType: 'project_update_comment.edited',
    labelKey: 'activityEvent.label.project_update_comment.edited',
    tone: 'info',
  },
  {
    eventType: 'project_update_comment.deleted',
    labelKey: 'activityEvent.label.project_update_comment.deleted',
    tone: 'warning',
  },
  {
    eventType: 'ticket.created',
    labelKey: 'activityEvent.label.ticket.created',
    tone: 'info',
  },
  {
    eventType: 'ticket.updated',
    labelKey: 'activityEvent.label.ticket.updated',
    tone: 'info',
  },
  {
    eventType: 'ticket.archived',
    labelKey: 'activityEvent.label.ticket.archived',
    tone: 'warning',
  },
  {
    eventType: 'ticket.unarchived',
    labelKey: 'activityEvent.label.ticket.unarchived',
    tone: 'info',
  },
  {
    eventType: 'ticket.status_changed',
    labelKey: 'activityEvent.label.ticket.status_changed',
    tone: 'info',
  },
  {
    eventType: 'ticket.completed',
    labelKey: 'activityEvent.label.ticket.completed',
    tone: 'success',
  },
  {
    eventType: 'ticket.cancelled',
    labelKey: 'activityEvent.label.ticket.cancelled',
    tone: 'warning',
  },
  {
    eventType: 'ticket.retry_scheduled',
    labelKey: 'activityEvent.label.ticket.retry_scheduled',
    tone: 'warning',
  },
  {
    eventType: 'ticket.retry_paused',
    labelKey: 'activityEvent.label.ticket.retry_paused',
    tone: 'warning',
  },
  {
    eventType: 'ticket.retry_resumed',
    labelKey: 'activityEvent.label.ticket.retry_resumed',
    tone: 'info',
  },
  {
    eventType: 'ticket.budget_exhausted',
    labelKey: 'activityEvent.label.ticket.budget_exhausted',
    tone: 'danger',
  },
  {
    eventType: 'agent.claimed',
    labelKey: 'activityEvent.label.agent.claimed',
    tone: 'info',
  },
  {
    eventType: 'agent.launching',
    labelKey: 'activityEvent.label.agent.launching',
    tone: 'info',
  },
  {
    eventType: 'agent.ready',
    labelKey: 'activityEvent.label.agent.ready',
    tone: 'success',
  },
  {
    eventType: 'agent.executing',
    labelKey: 'activityEvent.label.agent.executing',
    tone: 'success',
  },
  {
    eventType: 'agent.paused',
    labelKey: 'activityEvent.label.agent.paused',
    tone: 'warning',
  },
  {
    eventType: 'agent.failed',
    labelKey: 'activityEvent.label.agent.failed',
    tone: 'danger',
  },
  {
    eventType: 'agent.completed',
    labelKey: 'activityEvent.label.agent.completed',
    tone: 'success',
  },
  {
    eventType: 'agent.terminated',
    labelKey: 'activityEvent.label.agent.terminated',
    tone: 'neutral',
  },
  {
    eventType: 'project_conversation.deleted',
    labelKey: 'activityEvent.label.project_conversation.deleted',
    tone: 'warning',
  },
  {
    eventType: 'project_conversation.cleanup_run',
    labelKey: 'activityEvent.label.project_conversation.cleanup_run',
    tone: 'info',
  },
  {
    eventType: 'project_conversation.cleanup_skipped',
    labelKey: 'activityEvent.label.project_conversation.cleanup_skipped',
    tone: 'warning',
  },
  {
    eventType: 'hook.started',
    labelKey: 'activityEvent.label.hook.started',
    tone: 'info',
  },
  {
    eventType: 'hook.passed',
    labelKey: 'activityEvent.label.hook.passed',
    tone: 'success',
  },
  {
    eventType: 'hook.failed',
    labelKey: 'activityEvent.label.hook.failed',
    tone: 'danger',
  },
  {
    eventType: 'pr.opened',
    labelKey: 'activityEvent.label.pr.opened',
    tone: 'info',
  },
  {
    eventType: 'pr.merged',
    labelKey: 'activityEvent.label.pr.merged',
    tone: 'success',
  },
  {
    eventType: 'pr.closed',
    labelKey: 'activityEvent.label.pr.closed',
    tone: 'warning',
  },
]

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
