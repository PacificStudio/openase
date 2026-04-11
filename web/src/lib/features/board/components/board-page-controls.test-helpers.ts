import { fireEvent, within } from '@testing-library/svelte'
import { expect } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  StatusPayload,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { orderedStatusPayloadFixture } from '$lib/features/board/test-fixtures'

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

export const statusesFixture: StatusPayload = orderedStatusPayloadFixture

export const ticketsFixture: TicketPayload = {
  tickets: [
    {
      id: 'ticket-1',
      project_id: 'project-1',
      identifier: 'ASE-202',
      title: 'Wire board page to runtime data',
      description: '',
      status_id: 'status-1',
      status_name: 'Todo',
      priority: 'high',
      type: 'feature',
      archived: false,
      workflow_id: 'workflow-1',
      current_run_id: null,
      target_machine_id: null,
      created_by: 'codex',
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
      created_at: '2026-03-21T12:00:00Z',
    },
  ],
}

export const workflowsFixture: WorkflowListPayload = {
  workflows: [
    {
      id: 'workflow-1',
      project_id: 'project-1',
      agent_id: 'agent-1',
      name: 'Coding',
      type: 'coding',
      workflow_family: 'coding',
      workflow_classification: {
        family: 'coding',
        confidence: 1,
        reasons: ['fixture'],
      },
      harness_path: '.openase/harnesses/coding.md',
      harness_content: null,
      hooks: {},
      max_concurrent: 1,
      max_retry_attempts: 0,
      timeout_minutes: 30,
      stall_timeout_minutes: 10,
      version: 1,
      is_active: true,
      pickup_status_ids: ['status-1'],
      finish_status_ids: ['status-2'],
    },
  ],
}

export const agentsFixture: AgentPayload = { agents: [] }
export const activityFixture: ActivityPayload = { events: [], next_cursor: '', has_more: false }

export function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

export function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

export function ticketCardFor(container: HTMLElement, identifier: string) {
  const card = within(container).getByText(identifier).closest('button')
  if (!card) throw new Error(`ticket card not found for ${identifier}`)
  return card as HTMLButtonElement
}

type FindByRole = (role: string, options?: Record<string, unknown>) => Promise<HTMLElement>

export async function openColumnActionMenu(findByRole: FindByRole, columnName: string) {
  const ticketsList = (await findByRole('list', { name: `${columnName} tickets` })) as HTMLElement
  const column = ticketsList.parentElement
  if (!column) throw new Error(`column container not found for ${columnName}`)
  await fireEvent.click(within(column).getByLabelText('Column actions'))
}

export async function assertColumnMoveState(
  findByRole: FindByRole,
  columnName: string,
  expected: { leftDisabled: boolean; rightDisabled: boolean },
) {
  await openColumnActionMenu(findByRole, columnName)
  const moveLeft = await findByRole('menuitem', { name: 'Move left' })
  const moveRight = await findByRole('menuitem', { name: 'Move right' })
  expect(moveLeft.hasAttribute('data-disabled')).toBe(expected.leftDisabled)
  expect(moveRight.hasAttribute('data-disabled')).toBe(expected.rightDisabled)
  await fireEvent.keyDown(document.body, { key: 'Escape' })
}

export async function showEmptyColumns(findByRole: FindByRole) {
  await fireEvent.click(await findByRole('button', { name: 'Hide empty' }))
}

export function listColumnNames() {
  return Array.from(document.querySelectorAll('[role="list"][aria-label$=" tickets"]'))
    .map((node) => node.getAttribute('aria-label')?.replace(/ tickets$/, ''))
    .filter((value): value is string => !!value)
}
