export type ActivityEventTone = 'info' | 'success' | 'warning' | 'danger' | 'neutral'

export type ActivityEventCatalogEntry = {
  eventType: string
  label: string
  tone: ActivityEventTone
}

export const activityEventCatalog: ActivityEventCatalogEntry[] = [
  { eventType: 'project.created', label: 'Project created', tone: 'info' },
  { eventType: 'project.updated', label: 'Project updated', tone: 'info' },
  { eventType: 'project.archived', label: 'Project archived', tone: 'warning' },
  { eventType: 'project.status_changed', label: 'Project status changed', tone: 'info' },
  {
    eventType: 'project.default_provider_changed',
    label: 'Project default provider changed',
    tone: 'info',
  },
  { eventType: 'project.concurrency_changed', label: 'Project concurrency changed', tone: 'info' },
  { eventType: 'project_repo.created', label: 'Project repo created', tone: 'info' },
  { eventType: 'project_repo.updated', label: 'Project repo updated', tone: 'info' },
  { eventType: 'project_repo.deleted', label: 'Project repo deleted', tone: 'warning' },
  { eventType: 'ticket_status.created', label: 'Ticket status created', tone: 'info' },
  { eventType: 'ticket_status.updated', label: 'Ticket status updated', tone: 'info' },
  { eventType: 'ticket_status.reordered', label: 'Ticket status reordered', tone: 'info' },
  {
    eventType: 'ticket_status.concurrency_changed',
    label: 'Ticket status concurrency changed',
    tone: 'info',
  },
  { eventType: 'ticket_status.deleted', label: 'Ticket status deleted', tone: 'warning' },
  { eventType: 'ticket_status.reset', label: 'Ticket statuses reset', tone: 'warning' },
  { eventType: 'workflow.created', label: 'Workflow created', tone: 'info' },
  { eventType: 'workflow.updated', label: 'Workflow updated', tone: 'info' },
  { eventType: 'workflow.activated', label: 'Workflow activated', tone: 'success' },
  { eventType: 'workflow.deactivated', label: 'Workflow deactivated', tone: 'warning' },
  { eventType: 'workflow.deleted', label: 'Workflow deleted', tone: 'warning' },
  { eventType: 'workflow.harness_updated', label: 'Workflow harness updated', tone: 'info' },
  { eventType: 'workflow.hooks_updated', label: 'Workflow hooks updated', tone: 'info' },
  { eventType: 'workflow.agent_changed', label: 'Workflow agent changed', tone: 'info' },
  {
    eventType: 'workflow.pickup_statuses_changed',
    label: 'Workflow pickup statuses changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.finish_statuses_changed',
    label: 'Workflow finish statuses changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.concurrency_changed',
    label: 'Workflow concurrency changed',
    tone: 'info',
  },
  {
    eventType: 'workflow.retry_policy_changed',
    label: 'Workflow retry policy changed',
    tone: 'info',
  },
  { eventType: 'workflow.timeout_changed', label: 'Workflow timeout changed', tone: 'info' },
  { eventType: 'provider.created', label: 'Provider created', tone: 'info' },
  { eventType: 'provider.updated', label: 'Provider updated', tone: 'info' },
  {
    eventType: 'provider.availability_changed',
    label: 'Provider availability changed',
    tone: 'warning',
  },
  {
    eventType: 'provider.machine_binding_changed',
    label: 'Provider machine binding changed',
    tone: 'info',
  },
  { eventType: 'provider.rate_limit_updated', label: 'Provider rate limit updated', tone: 'info' },
  { eventType: 'agent.created', label: 'Agent created', tone: 'info' },
  { eventType: 'agent.updated', label: 'Agent updated', tone: 'info' },
  { eventType: 'agent.resumed', label: 'Agent resumed', tone: 'success' },
  { eventType: 'agent.deleted', label: 'Agent deleted', tone: 'warning' },
  { eventType: 'scheduled_job.created', label: 'Scheduled job created', tone: 'info' },
  { eventType: 'scheduled_job.updated', label: 'Scheduled job updated', tone: 'info' },
  { eventType: 'scheduled_job.enabled', label: 'Scheduled job enabled', tone: 'success' },
  { eventType: 'scheduled_job.disabled', label: 'Scheduled job disabled', tone: 'warning' },
  { eventType: 'scheduled_job.deleted', label: 'Scheduled job deleted', tone: 'warning' },
  { eventType: 'scheduled_job.triggered', label: 'Scheduled job triggered', tone: 'info' },
  { eventType: 'ticket_comment.created', label: 'Ticket comment created', tone: 'info' },
  { eventType: 'ticket_comment.edited', label: 'Ticket comment edited', tone: 'info' },
  { eventType: 'ticket_comment.deleted', label: 'Ticket comment deleted', tone: 'warning' },
  {
    eventType: 'project_update_thread.created',
    label: 'Project update created',
    tone: 'info',
  },
  {
    eventType: 'project_update_thread.edited',
    label: 'Project update edited',
    tone: 'info',
  },
  {
    eventType: 'project_update_thread.deleted',
    label: 'Project update deleted',
    tone: 'warning',
  },
  {
    eventType: 'project_update_thread.status_changed',
    label: 'Project update status changed',
    tone: 'warning',
  },
  {
    eventType: 'project_update_comment.created',
    label: 'Project update comment created',
    tone: 'info',
  },
  {
    eventType: 'project_update_comment.edited',
    label: 'Project update comment edited',
    tone: 'info',
  },
  {
    eventType: 'project_update_comment.deleted',
    label: 'Project update comment deleted',
    tone: 'warning',
  },
  { eventType: 'ticket.created', label: 'Ticket created', tone: 'info' },
  { eventType: 'ticket.updated', label: 'Ticket updated', tone: 'info' },
  { eventType: 'ticket.archived', label: 'Ticket archived', tone: 'warning' },
  { eventType: 'ticket.unarchived', label: 'Ticket unarchived', tone: 'info' },
  { eventType: 'ticket.status_changed', label: 'Ticket status changed', tone: 'info' },
  { eventType: 'ticket.completed', label: 'Ticket completed', tone: 'success' },
  { eventType: 'ticket.cancelled', label: 'Ticket cancelled', tone: 'warning' },
  { eventType: 'ticket.retry_scheduled', label: 'Ticket retry scheduled', tone: 'warning' },
  { eventType: 'ticket.retry_paused', label: 'Ticket retry paused', tone: 'warning' },
  { eventType: 'ticket.retry_resumed', label: 'Ticket retry resumed', tone: 'info' },
  { eventType: 'ticket.budget_exhausted', label: 'Ticket budget exhausted', tone: 'danger' },
  { eventType: 'agent.claimed', label: 'Agent claimed', tone: 'info' },
  { eventType: 'agent.launching', label: 'Agent launching', tone: 'info' },
  { eventType: 'agent.ready', label: 'Agent ready', tone: 'success' },
  { eventType: 'agent.executing', label: 'Agent executing', tone: 'success' },
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
