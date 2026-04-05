/* eslint-disable max-lines */
import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  ProjectRepoPayload,
  TicketDetailPayload,
  TicketPayload,
  TicketRunListPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import { TicketsPage, resetTicketBoardToolbarStoreForTests } from '$lib/features/tickets'
import { orderedStatusPayloadFixture } from '$lib/features/board/test-fixtures'
import { appStore } from '$lib/stores/app.svelte'
import { ticketViewStore } from '$lib/stores/ticket-view.svelte'
import BoardPageTicketDrawerHost from './board-page-ticket-drawer-host.svelte'

const {
  getTicketDetail,
  getTicketRun,
  listActivity,
  listAgents,
  listProjectRepos,
  listStatuses,
  listTicketRuns,
  listTickets,
  listWorkflows,
  updateTicket,
} = vi.hoisted(() => ({
  getTicketDetail: vi.fn(),
  getTicketRun: vi.fn(),
  listActivity: vi.fn(),
  listAgents: vi.fn(),
  listProjectRepos: vi.fn(),
  listStatuses: vi.fn(),
  listTicketRuns: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
  updateTicket: vi.fn(),
}))

const {
  subscribeProjectEvents,
  subscribeProjectEventBusState,
  isTicketRunProjectEvent,
  projectEventReferencesTicket,
  toProjectEventFrame,
} = vi.hoisted(() => ({
  subscribeProjectEvents: vi.fn(),
  subscribeProjectEventBusState: vi.fn((_: string, listener: (state: string) => void) => {
    listener('live')
    return () => {}
  }),
  isTicketRunProjectEvent: (event: { topic?: string }) => event.topic === 'ticket.run.events',
  projectEventReferencesTicket: (
    event: { payload?: { entry?: { ticket_id?: string } } },
    ticketId: string,
  ) => event.payload?.entry?.ticket_id === ticketId,
  toProjectEventFrame: (event: { type: string; payload: unknown }) => ({
    event: event.type,
    data: JSON.stringify(event.payload),
  }),
}))

vi.mock('$lib/api/openase', () => ({
  getTicketDetail,
  getTicketRun,
  listActivity,
  listAgents,
  listProjectRepos,
  listStatuses,
  listTicketRuns,
  listTickets,
  listWorkflows,
  updateTicket,
}))

vi.mock('$lib/features/project-events/project-event-bus', () => ({
  subscribeProjectEvents,
  subscribeProjectEventBusState,
  isTicketRunProjectEvent,
  projectEventReferencesTicket,
  toProjectEventFrame,
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

const statusesFixture = orderedStatusPayloadFixture

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

const agentsFixture: AgentPayload = {
  agents: [
    {
      id: 'agent-1',
      provider_id: 'provider-1',
      project_id: 'project-1',
      name: 'Codex Worker',
      runtime_control_state: 'active',
      runtime: {
        active_run_count: 1,
        status: 'running',
        current_run_id: null,
        current_ticket_id: 'ticket-1',
        session_id: 'session-1',
        runtime_phase: 'ready',
        runtime_started_at: null,
        last_error: '',
        last_heartbeat_at: null,
        current_step_status: null,
        current_step_summary: null,
        current_step_changed_at: null,
      },
      total_tokens_used: 0,
      total_tickets_completed: 0,
    },
  ],
}

const activityFixture: ActivityPayload = {
  events: [],
}

const ticketDetailFixture: TicketDetailPayload = {
  assigned_agent: {
    id: 'agent-1',
    name: 'todo-app-coding-01',
    provider: 'codex-cloud',
    runtime_control_state: 'active',
    runtime_phase: 'executing',
  },
  pickup_diagnosis: {
    state: 'running',
    primary_reason_code: 'running_current_run',
    primary_reason_message: 'Ticket already has an active run.',
    next_action_hint: 'Wait for the current run to finish or inspect the active runtime.',
    reasons: [
      {
        code: 'running_current_run',
        message: 'Current run is still attached to the ticket.',
        severity: 'info',
      },
    ],
    workflow: {
      id: 'workflow-1',
      name: 'Coding Workflow',
      is_active: true,
      pickup_status_match: true,
    },
    agent: {
      id: 'agent-1',
      name: 'todo-app-coding-01',
      runtime_control_state: 'active',
    },
    provider: {
      id: 'provider-1',
      name: 'codex-cloud',
      machine_id: 'machine-1',
      machine_name: 'builder-01',
      machine_status: 'online',
      availability_state: 'available',
      availability_reason: null,
    },
    retry: {
      attempt_count: 0,
      retry_paused: false,
      pause_reason: '',
      next_retry_at: null,
    },
    capacity: {
      workflow: { limited: false, active_runs: 1, capacity: 0 },
      project: { limited: false, active_runs: 1, capacity: 0 },
      provider: { limited: false, active_runs: 1, capacity: 0 },
      status: { limited: false, active_runs: 1, capacity: null },
    },
    blocked_by: [],
  },
  ticket: {
    ...ticketsFixture.tickets[0],
    description: 'Board drawer description',
  },
  repo_scopes: [],
  comments: [],
  timeline: [
    {
      id: 'description:ticket-1',
      ticket_id: 'ticket-1',
      item_type: 'description',
      actor_name: 'codex',
      actor_type: 'user',
      title: 'Wire board page to runtime data',
      body_markdown: 'Board drawer description',
      body_text: null,
      created_at: '2026-03-21T12:00:00Z',
      updated_at: '2026-03-21T12:00:00Z',
      edited_at: null,
      is_collapsible: false,
      is_deleted: false,
      metadata: { identifier: 'ASE-202' },
    },
    {
      id: 'comment:comment-1',
      ticket_id: 'ticket-1',
      item_type: 'comment',
      actor_name: 'user:reviewer',
      actor_type: 'user',
      title: null,
      body_markdown: 'Board drawer comment',
      body_text: null,
      created_at: '2026-03-21T12:05:00Z',
      updated_at: '2026-03-21T12:05:00Z',
      edited_at: null,
      is_collapsible: true,
      is_deleted: false,
      metadata: { edit_count: 0, revision_count: 1, last_edited_by: '' },
    },
  ],
  activity: [],
  hook_history: [],
}

const ticketRunsFixture: TicketRunListPayload = { runs: [] }
const projectReposFixture: ProjectRepoPayload = { repos: [] }

describe('TicketsPage runtime and drawer', () => {
  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    appStore.closeRightPanel()
    ticketViewStore.setMode('board')
    resetTicketBoardToolbarStoreForTests()
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('reloads board runtime state when the agents stream emits an event', async () => {
    appStore.currentProject = projectFixture
    const baseRuntime = agentsFixture.agents[0].runtime!
    const initialAgents: AgentPayload = {
      agents: [{ ...agentsFixture.agents[0], runtime: { ...baseRuntime, runtime_phase: 'ready' } }],
    }
    const executingAgents: AgentPayload = {
      agents: [
        { ...agentsFixture.agents[0], runtime: { ...baseRuntime, runtime_phase: 'executing' } },
      ],
    }

    let projectEventOnEvent: ((event?: unknown) => void) | undefined
    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(ticketsFixture)
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValueOnce(initialAgents).mockResolvedValue(executingAgents)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    subscribeProjectEvents.mockImplementation((_: string, onEvent: () => void) => {
      projectEventOnEvent = onEvent
      return () => {}
    })

    const { findByText } = render(TicketsPage)
    const initialCard = (await findByText('ASE-202')).closest('button')
    if (!initialCard) throw new Error('ticket card not found')
    expect(within(initialCard).getByTitle('Ready')).toBeTruthy()
    expect(within(initialCard).queryByTitle('Executing')).toBeNull()

    projectEventOnEvent?.({ topic: 'agent.events', type: 'agent.ready', payload: null })

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(2)
      const updatedCard = within(document.body).getByText('ASE-202').closest('button')
      if (!updatedCard) throw new Error('updated ticket card not found')
      expect(within(updatedCard).getByTitle('Executing')).toBeTruthy()
    })
  })

  it('opens ticket detail from the board and renders ticket content and metadata', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(ticketsFixture)
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    getTicketDetail.mockResolvedValue({ ...ticketDetailFixture })
    listProjectRepos.mockResolvedValue(projectReposFixture)
    listTicketRuns.mockResolvedValue(ticketRunsFixture)
    getTicketRun.mockResolvedValue(undefined)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    subscribeProjectEvents.mockReturnValue(() => {})

    const { findAllByText, findByRole, findByText } = render(BoardPageTicketDrawerHost)
    const ticketCard = (await findByText('ASE-202')).closest('button')
    if (!ticketCard) throw new Error('ticket card not found')

    await fireEvent.click(ticketCard)

    expect((await findAllByText('todo-app-coding-01')).length).toBeGreaterThan(0)
    expect((await findAllByText('codex-cloud')).length).toBeGreaterThan(0)

    await fireEvent.click(await findByRole('tab', { name: 'Discussion' }))
    expect(await findByText('Board drawer description')).toBeTruthy()
    expect(await findByText('Board drawer comment')).toBeTruthy()
  })
})
