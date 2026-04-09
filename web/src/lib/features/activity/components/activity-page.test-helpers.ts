import type { ActivityPayload, Project, TicketPayload } from '$lib/api/contracts'

export const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: '',
  status: 'active',
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

export const ticketPayload: TicketPayload = {
  tickets: [
    {
      id: 'ticket-1',
      identifier: 'ASE-101',
      project_id: 'project-1',
      title: 'Fix dashboard refresh',
      description: '',
      status_id: 'status-1',
      status_name: 'Todo',
      priority: 'high',
      type: 'task',
      archived: false,
      workflow_id: null,
      current_run_id: null,
      target_machine_id: null,
      created_by: 'user:test',
      parent: null,
      children: [],
      dependencies: [],
      external_links: [],
      pull_request_urls: [],
      external_ref: '',
      budget_usd: 0,
      cost_tokens_input: 0,
      cost_tokens_output: 0,
      cost_tokens_total: 0,
      cost_amount: 0,
      attempt_count: 0,
      consecutive_errors: 0,
      started_at: null,
      completed_at: null,
      next_retry_at: null,
      retry_paused: false,
      pause_reason: '',
      created_at: '2026-04-02T09:00:00Z',
    },
  ],
}

export function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

export function activityEvent(
  id: string,
  message: string,
  createdAt: string,
  overrides: Partial<ActivityPayload['events'][number]> = {},
): ActivityPayload['events'][number] {
  return {
    id,
    project_id: 'project-1',
    ticket_id: 'ticket-1',
    agent_id: 'agent-1',
    event_type: 'ticket.updated',
    message,
    metadata: { agent_name: 'Coding Agent' },
    created_at: createdAt,
    ...overrides,
  }
}

export function activityPayload(
  events: ActivityPayload['events'],
  pagination: { next_cursor?: string; has_more?: boolean } = {},
): ActivityPayload {
  return {
    events,
    next_cursor: pagination.next_cursor ?? '',
    has_more: pagination.has_more ?? false,
  }
}
