import type { ActivityEventCatalogDefinition } from './event-catalog-types'

export const activityEventCoreDefinitions: ActivityEventCatalogDefinition[] = [
  { eventType: 'project.created', labelKey: 'activityEvent.label.project.created', tone: 'info' },
  { eventType: 'project.updated', labelKey: 'activityEvent.label.project.updated', tone: 'info' },
  {
    eventType: 'project.archived',
    labelKey: 'activityEvent.label.project.archived',
    tone: 'warning',
  },
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
]
