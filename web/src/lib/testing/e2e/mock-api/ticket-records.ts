import { PROJECT_ID, nowIso } from './constants'

export function createMockTicketRecord(input: {
  id: string
  identifier: string
  title: string
  description?: string
  statusId: string
  statusName: string
  workflowId: string
  createdAt?: string
}) {
  return {
    id: input.id,
    project_id: PROJECT_ID,
    identifier: input.identifier,
    title: input.title,
    description: input.description ?? '',
    status_id: input.statusId,
    status_name: input.statusName,
    priority: 'medium',
    type: 'feature',
    workflow_id: input.workflowId,
    current_run_id: null,
    target_machine_id: null,
    created_by: 'playwright',
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
    created_at: input.createdAt ?? nowIso,
  }
}
