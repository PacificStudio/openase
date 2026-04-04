import { describe, expect, it } from 'vitest'

import {
  activityEventLabel,
  activityEventTone,
  getActivityEventCatalogEntry,
} from './event-catalog'

const canonicalCases = [
  ['project.created', 'Project created', 'info'],
  ['project.updated', 'Project updated', 'info'],
  ['project.archived', 'Project archived', 'warning'],
  ['project.status_changed', 'Project status changed', 'info'],
  ['project.default_provider_changed', 'Project default provider changed', 'info'],
  ['project.concurrency_changed', 'Project concurrency changed', 'info'],
  ['project_repo.created', 'Project repo created', 'info'],
  ['project_repo.updated', 'Project repo updated', 'info'],
  ['project_repo.deleted', 'Project repo deleted', 'warning'],
  ['ticket_status.created', 'Ticket status created', 'info'],
  ['ticket_status.updated', 'Ticket status updated', 'info'],
  ['ticket_status.reordered', 'Ticket status reordered', 'info'],
  ['ticket_status.concurrency_changed', 'Ticket status concurrency changed', 'info'],
  ['ticket_status.deleted', 'Ticket status deleted', 'warning'],
  ['ticket_status.reset', 'Ticket statuses reset', 'warning'],
  ['workflow.created', 'Workflow created', 'info'],
  ['workflow.updated', 'Workflow updated', 'info'],
  ['workflow.activated', 'Workflow activated', 'success'],
  ['workflow.deactivated', 'Workflow deactivated', 'warning'],
  ['workflow.deleted', 'Workflow deleted', 'warning'],
  ['workflow.harness_updated', 'Workflow harness updated', 'info'],
  ['workflow.hooks_updated', 'Workflow hooks updated', 'info'],
  ['workflow.agent_changed', 'Workflow agent changed', 'info'],
  ['workflow.pickup_statuses_changed', 'Workflow pickup statuses changed', 'info'],
  ['workflow.finish_statuses_changed', 'Workflow finish statuses changed', 'info'],
  ['workflow.concurrency_changed', 'Workflow concurrency changed', 'info'],
  ['workflow.retry_policy_changed', 'Workflow retry policy changed', 'info'],
  ['workflow.timeout_changed', 'Workflow timeout changed', 'info'],
  ['provider.created', 'Provider created', 'info'],
  ['provider.updated', 'Provider updated', 'info'],
  ['provider.availability_changed', 'Provider availability changed', 'warning'],
  ['provider.machine_binding_changed', 'Provider machine binding changed', 'info'],
  ['provider.rate_limit_updated', 'Provider rate limit updated', 'info'],
  ['agent.created', 'Agent created', 'info'],
  ['agent.updated', 'Agent updated', 'info'],
  ['agent.resumed', 'Agent resumed', 'success'],
  ['agent.executing', 'Agent executing', 'success'],
  ['agent.deleted', 'Agent deleted', 'warning'],
  ['scheduled_job.created', 'Scheduled job created', 'info'],
  ['scheduled_job.updated', 'Scheduled job updated', 'info'],
  ['scheduled_job.enabled', 'Scheduled job enabled', 'success'],
  ['scheduled_job.disabled', 'Scheduled job disabled', 'warning'],
  ['scheduled_job.deleted', 'Scheduled job deleted', 'warning'],
  ['scheduled_job.triggered', 'Scheduled job triggered', 'info'],
  ['ticket_comment.created', 'Ticket comment created', 'info'],
  ['ticket_comment.edited', 'Ticket comment edited', 'info'],
  ['ticket_comment.deleted', 'Ticket comment deleted', 'warning'],
  ['project_update_thread.created', 'Project update created', 'info'],
  ['project_update_thread.edited', 'Project update edited', 'info'],
  ['project_update_thread.deleted', 'Project update deleted', 'warning'],
  ['project_update_thread.status_changed', 'Project update status changed', 'warning'],
  ['project_update_comment.created', 'Project update comment created', 'info'],
  ['project_update_comment.edited', 'Project update comment edited', 'info'],
  ['project_update_comment.deleted', 'Project update comment deleted', 'warning'],
  ['ticket.archived', 'Ticket archived', 'warning'],
  ['ticket.unarchived', 'Ticket unarchived', 'info'],
] as const

describe('activity event catalog', () => {
  it('covers every newly added canonical type with a label and tone', () => {
    for (const [eventType, label, tone] of canonicalCases) {
      expect(getActivityEventCatalogEntry(eventType)).toEqual({ eventType, label, tone })
      expect(activityEventLabel(eventType)).toBe(label)
      expect(activityEventTone(eventType)).toBe(tone)
    }
  })
})
