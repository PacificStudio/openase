import { cleanup, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { retainProjectEventBus } from '$lib/features/project-events'
import {
  TicketsPage,
  resetProjectBoardCacheForTests,
  resetTicketBoardToolbarStoreForTests,
} from '$lib/features/tickets'
import { orderedStatusPayloadFixture } from '$lib/features/board/test-fixtures'
import { appStore } from '$lib/stores/app.svelte'

const { connectEventStream, listActivity, listAgents, listStatuses, listTickets, listWorkflows } =
  vi.hoisted(() => ({
    connectEventStream: vi.fn(),
    listActivity: vi.fn(),
    listAgents: vi.fn(),
    listStatuses: vi.fn(),
    listTickets: vi.fn(),
    listWorkflows: vi.fn(),
  }))

vi.mock('$lib/api/sse', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/sse')>('$lib/api/sse')
  return {
    ...actual,
    connectEventStream,
  }
})

vi.mock('$lib/api/openase', () => ({
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
}))

const projectFixture: Project = {
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

const ticketsFixture: TicketPayload = {
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

const workflowsFixture: WorkflowListPayload = {
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

const activityFixture: ActivityPayload = {
  events: [],
  next_cursor: '',
  has_more: false,
}

describe('TicketsPage reconnect', () => {
  afterEach(() => {
    cleanup()
    resetProjectBoardCacheForTests()
    resetTicketBoardToolbarStoreForTests()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('refreshes runtime badges after the shared project stream reconnects', async () => {
    appStore.currentProject = projectFixture

    const baseRuntime = {
      active_run_count: 1,
      status: 'running',
      current_run_id: null,
      current_ticket_id: 'ticket-1',
      session_id: 'session-1',
      runtime_started_at: null,
      last_error: '',
      last_heartbeat_at: null,
      current_step_status: null,
      current_step_summary: null,
      current_step_changed_at: null,
    }
    const initialAgents: AgentPayload = {
      agents: [
        {
          id: 'agent-1',
          provider_id: 'provider-1',
          project_id: 'project-1',
          name: 'Codex Worker',
          runtime_control_state: 'active',
          runtime: { ...baseRuntime, runtime_phase: 'ready' },
          total_tokens_used: 0,
          total_tickets_completed: 0,
        },
      ],
    }
    const executingAgents: AgentPayload = {
      agents: [
        {
          ...initialAgents.agents[0],
          runtime: { ...baseRuntime, runtime_phase: 'executing' },
        },
      ],
    }

    listStatuses.mockResolvedValue(orderedStatusPayloadFixture)
    listTickets.mockResolvedValue(ticketsFixture)
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValueOnce(initialAgents).mockResolvedValue(executingAgents)
    listActivity.mockResolvedValue(activityFixture)
    connectEventStream.mockReturnValue(() => {})

    const releaseShell = retainProjectEventBus(projectFixture.id)
    const view = render(TicketsPage)

    const initialCard = (await view.findByText('ASE-202')).closest('button')
    if (!initialCard) {
      throw new Error('ticket card not found')
    }
    expect(within(initialCard).getByTitle('Ready')).toBeTruthy()
    expect(listAgents).toHaveBeenCalledTimes(1)

    const projectBusCall = connectEventStream.mock.calls.find(
      ([url]) => url === `/api/v1/projects/${projectFixture.id}/events/stream`,
    )
    if (!projectBusCall) {
      throw new Error('project event stream was not opened')
    }

    const options = projectBusCall[1] as {
      onStateChange: (state: 'live' | 'idle' | 'connecting' | 'retrying') => void
    }
    options.onStateChange('connecting')
    options.onStateChange('live')
    options.onStateChange('retrying')
    options.onStateChange('live')

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(2)
      const updatedCard = view.getByText('ASE-202').closest('button')
      if (!updatedCard) {
        throw new Error('updated ticket card not found')
      }
      expect(within(updatedCard).getByTitle('Executing')).toBeTruthy()
    })

    releaseShell()
  })
})
