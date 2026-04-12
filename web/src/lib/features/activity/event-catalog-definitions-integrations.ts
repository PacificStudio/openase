import type { ActivityEventCatalogDefinition } from './event-catalog-types'

export const activityEventIntegrationDefinitions: ActivityEventCatalogDefinition[] = [
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
