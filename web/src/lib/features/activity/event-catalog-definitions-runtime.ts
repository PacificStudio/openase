import type { ActivityEventCatalogDefinition } from './event-catalog-types'

export const activityEventRuntimeDefinitions: ActivityEventCatalogDefinition[] = [
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
]
