import { cleanup, fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type {
  ActivityPayload,
  AgentPayload,
  Project,
  StatusPayload,
  Ticket,
  TicketPayload,
  WorkflowListPayload,
} from '$lib/api/contracts'
import {
  TicketsPage,
  markProjectBoardCacheDirty,
  resetTicketBoardToolbarStoreForTests,
} from '$lib/features/tickets'
import { orderedStatusPayloadFixture } from '$lib/features/board/test-fixtures'
import { resetProjectBoardCacheForTests } from '$lib/features/tickets'
import { appStore } from '$lib/stores/app.svelte'
import { ticketViewStore } from '$lib/stores/ticket-view.svelte'

const {
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
  toastStore,
  updateStatus,
  updateTicket,
  connectEventStream,
} = vi.hoisted(() => ({
  listActivity: vi.fn(),
  listAgents: vi.fn(),
  listStatuses: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
    info: vi.fn(),
    clear: vi.fn(),
  },
  updateStatus: vi.fn(),
  updateTicket: vi.fn(),
  connectEventStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  listActivity,
  listAgents,
  listStatuses,
  listTickets,
  listWorkflows,
  updateStatus,
  updateTicket,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
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

const agentsFixture: AgentPayload = { agents: [] }
const activityFixture: ActivityPayload = { events: [] }

function createDeferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((nextResolve, nextReject) => {
    resolve = nextResolve
    reject = nextReject
  })
  return { promise, resolve, reject }
}

describe('TicketsPage board controls', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    ticketViewStore.setMode('board')
  })

  afterEach(async () => {
    cleanup()
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
    resetProjectBoardCacheForTests()
    resetTicketBoardToolbarStoreForTests()
    appStore.currentProject = null
    ticketViewStore.setMode('board')
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('reuses the cached board snapshot when remounting the tickets page in the same project', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(cloneValue(ticketsFixture))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)

    const firstRender = render(TicketsPage)
    expect(await firstRender.findByText('ASE-202')).toBeTruthy()

    expect(listStatuses).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listWorkflows).toHaveBeenCalledTimes(1)
    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)

    firstRender.unmount()

    const secondRender = render(TicketsPage)
    expect(await secondRender.findByText('ASE-202')).toBeTruthy()

    expect(listStatuses).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listWorkflows).toHaveBeenCalledTimes(1)
    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)
  })

  it('shows the cached board immediately and refreshes in the background once cache is dirty', async () => {
    appStore.currentProject = projectFixture

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockResolvedValue(cloneValue(ticketsFixture))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)

    const firstRender = render(TicketsPage)
    expect(await firstRender.findByText('ASE-202')).toBeTruthy()
    firstRender.unmount()

    markProjectBoardCacheDirty(projectFixture.id)

    const deferredStatuses = createDeferred<StatusPayload>()
    const deferredTickets = createDeferred<TicketPayload>()
    const deferredWorkflows = createDeferred<WorkflowListPayload>()
    const deferredAgents = createDeferred<AgentPayload>()
    const deferredActivity = createDeferred<ActivityPayload>()

    listStatuses.mockImplementationOnce(() => deferredStatuses.promise)
    listTickets.mockImplementationOnce(() => deferredTickets.promise)
    listWorkflows.mockImplementationOnce(() => deferredWorkflows.promise)
    listAgents.mockImplementationOnce(() => deferredAgents.promise)
    listActivity.mockImplementationOnce(() => deferredActivity.promise)

    const secondRender = render(TicketsPage)
    expect(await secondRender.findByText('ASE-202')).toBeTruthy()

    expect(listStatuses).toHaveBeenCalledTimes(2)
    expect(listTickets).toHaveBeenCalledTimes(2)
    expect(listWorkflows).toHaveBeenCalledTimes(2)
    expect(listAgents).toHaveBeenCalledTimes(2)
    expect(listActivity).toHaveBeenCalledTimes(2)

    deferredStatuses.resolve(statusesFixture)
    deferredTickets.resolve(cloneValue(ticketsFixture))
    deferredWorkflows.resolve(workflowsFixture)
    deferredAgents.resolve(agentsFixture)
    deferredActivity.resolve(activityFixture)

    await waitFor(() => {
      expect(secondRender.getByText('ASE-202')).toBeTruthy()
    })
  })

  it('shows tickets in their status columns and updates status and priority from the board card controls', async () => {
    appStore.currentProject = projectFixture

    const currentTickets: Ticket[] = cloneValue([
      { ...ticketsFixture.tickets[0], id: 'ticket-1', status_name: 'Todo', priority: 'high' },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-2',
        identifier: 'ASE-201',
        title: 'Triage incoming board work',
        status_id: 'status-3',
        status_name: 'Inbox',
        priority: 'low',
      },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-3',
        identifier: 'ASE-203',
        title: 'Ship workflow runner polish',
        status_id: 'status-2',
        status_name: 'Doing',
        priority: 'medium',
      },
    ])

    listStatuses.mockResolvedValue(statusesFixture)
    listTickets.mockImplementation(async () => ({ tickets: cloneValue(currentTickets) }))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockImplementation(
      async (ticketId: string, patch: { status_id?: string; priority?: string }) => {
        const ticket = currentTickets.find((item) => item.id === ticketId)
        if (!ticket) throw new Error(`unknown ticket ${ticketId}`)
        if (patch.status_id) {
          const status = statusesFixture.statuses.find((item) => item.id === patch.status_id)
          if (!status) throw new Error(`unknown status ${patch.status_id}`)
          ticket.status_id = status.id
          ticket.status_name = status.name
        }
        if (patch.priority) ticket.priority = patch.priority as Ticket['priority']
        return { ticket: cloneValue(ticket) }
      },
    )
    connectEventStream.mockReturnValue(() => {})

    const { findByRole, findByText, getByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    expect(await findByText('ASE-201')).toBeTruthy()
    expect(
      within(await findByRole('list', { name: 'Inbox tickets' })).getByText('ASE-201'),
    ).toBeTruthy()
    expect(
      within(await findByRole('list', { name: 'Todo tickets' })).getByText('ASE-202'),
    ).toBeTruthy()
    expect(
      within(await findByRole('list', { name: 'Doing tickets' })).getByText('ASE-203'),
    ).toBeTruthy()

    await fireEvent.click(
      within(ticketCardFor(getByRole('list', { name: 'Todo tickets' }), 'ASE-202')).getByLabelText(
        'Change status',
      ),
    )
    await fireEvent.click(await findByRole('button', { name: 'Doing' }))

    await waitFor(() => {
      expect(updateTicket).toHaveBeenNthCalledWith(1, 'ticket-1', { status_id: 'status-2' })
      expect(within(getByRole('list', { name: 'Doing tickets' })).getByText('ASE-202')).toBeTruthy()
      expect(within(getByRole('list', { name: 'Todo tickets' })).queryByText('ASE-202')).toBeNull()
    })

    await fireEvent.click(
      within(ticketCardFor(getByRole('list', { name: 'Doing tickets' }), 'ASE-202')).getByLabelText(
        'Change priority',
      ),
    )
    await fireEvent.click(await findByRole('button', { name: /Urgent/i }))

    await waitFor(() => {
      expect(updateTicket).toHaveBeenNthCalledWith(2, 'ticket-1', { priority: 'urgent' })
      expect(
        within(
          ticketCardFor(getByRole('list', { name: 'Doing tickets' }), 'ASE-202'),
        ).getByLabelText('Priority: urgent'),
      ).toBeTruthy()
    })
  })

  it('enables move left and move right only within the current stage', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-backlog',
          project_id: 'project-1',
          name: 'Inbox',
          stage: 'backlog' as const,
          color: '#64748b',
          icon: '',
          is_default: false,
          description: '',
          position: 0,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-triage',
          project_id: 'project-1',
          name: 'Triage',
          stage: 'unstarted' as const,
          color: '#0ea5e9',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-doing',
          project_id: 'project-1',
          name: 'Doing',
          stage: 'started' as const,
          color: '#f59e0b',
          icon: '',
          is_default: false,
          description: '',
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-review',
          project_id: 'project-1',
          name: 'Review',
          stage: 'started' as const,
          color: '#f97316',
          icon: '',
          is_default: false,
          description: '',
          position: 4,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-done',
          project_id: 'project-1',
          name: 'Done',
          stage: 'completed' as const,
          color: '#10b981',
          icon: '',
          is_default: false,
          description: '',
          position: 5,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await assertColumnMoveState(findByRole, 'Inbox', {
      leftDisabled: true,
      rightDisabled: true,
    })
    await assertColumnMoveState(findByRole, 'Todo', {
      leftDisabled: true,
      rightDisabled: false,
    })
    await assertColumnMoveState(findByRole, 'Triage', {
      leftDisabled: false,
      rightDisabled: true,
    })
    await assertColumnMoveState(findByRole, 'Doing', {
      leftDisabled: true,
      rightDisabled: false,
    })
    await assertColumnMoveState(findByRole, 'Review', {
      leftDisabled: false,
      rightDisabled: true,
    })
    await assertColumnMoveState(findByRole, 'Done', {
      leftDisabled: true,
      rightDisabled: true,
    })
  })

  it('shows concurrency actions in the column menu and only enables clear when a limit exists', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-doing',
          project_id: 'project-1',
          name: 'Doing',
          stage: 'started' as const,
          color: '#f59e0b',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 1,
          max_active_runs: 3,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    expect(await findByRole('menuitem', { name: 'Set concurrency' })).toBeTruthy()
    expect(
      (await findByRole('menuitem', { name: 'Clear concurrency' })).hasAttribute('data-disabled'),
    ).toBe(true)
    await fireEvent.keyDown(document.body, { key: 'Escape' })

    await openColumnActionMenu(findByRole, 'Doing')
    expect(await findByRole('menuitem', { name: 'Set concurrency' })).toBeTruthy()
    expect(
      (await findByRole('menuitem', { name: 'Clear concurrency' })).hasAttribute('data-disabled'),
    ).toBe(false)
  })

  it('updates a status concurrency limit from the column menu prompt', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(
      async (statusId: string, patch: { max_active_runs?: number | null }) => {
        const status = currentStatuses.statuses.find((item) => item.id === statusId)
        if (!status) throw new Error(`unknown status ${statusId}`)
        status.max_active_runs = patch.max_active_runs ?? null
        return { status: cloneValue(status) }
      },
    )
    connectEventStream.mockReturnValue(() => {})
    const promptSpy = vi.spyOn(window, 'prompt').mockReturnValue('3')

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Set concurrency' }))

    await waitFor(() => {
      expect(promptSpy).toHaveBeenCalledWith(
        'Set concurrency limit for "Todo". Leave blank for Unlimited.',
        '',
      )
      expect(updateStatus).toHaveBeenCalledWith('status-todo', { max_active_runs: 3 })
    })
  })

  it('clears a status concurrency limit and ignores invalid prompt values', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 1,
          max_active_runs: 2,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(
      async (statusId: string, patch: { max_active_runs?: number | null }) => {
        const status = currentStatuses.statuses.find((item) => item.id === statusId)
        if (!status) throw new Error(`unknown status ${statusId}`)
        status.max_active_runs = patch.max_active_runs ?? null
        return { status: cloneValue(status) }
      },
    )
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    const promptSpy = vi.spyOn(window, 'prompt').mockReturnValue('0')
    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Set concurrency' }))
    expect(updateStatus).not.toHaveBeenCalled()
    expect(promptSpy).toHaveBeenCalledWith(
      'Set concurrency limit for "Todo". Leave blank for Unlimited.',
      '2',
    )

    promptSpy.mockRestore()
    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Clear concurrency' }))

    await waitFor(() => {
      expect(updateStatus).toHaveBeenCalledWith('status-todo', { max_active_runs: null })
    })
  })

  it('moves a column right by swapping positions with the next status in the same stage', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses = cloneValue({
      statuses: [
        {
          id: 'status-backlog',
          project_id: 'project-1',
          name: 'Inbox',
          stage: 'backlog' as const,
          color: '#64748b',
          icon: '',
          is_default: false,
          description: '',
          position: 0,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-triage',
          project_id: 'project-1',
          name: 'Triage',
          stage: 'unstarted' as const,
          color: '#0ea5e9',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-doing',
          project_id: 'project-1',
          name: 'Doing',
          stage: 'started' as const,
          color: '#f59e0b',
          icon: '',
          is_default: false,
          description: '',
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(async (statusId: string, patch: { position?: number }) => {
      const status = currentStatuses.statuses.find((item) => item.id === statusId)
      if (!status || typeof patch.position !== 'number') {
        throw new Error(`unknown status update ${statusId}`)
      }
      status.position = patch.position
      return { status: cloneValue(status) }
    })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Move right' }))

    await waitFor(() => {
      expect(updateStatus).toHaveBeenNthCalledWith(1, 'status-todo', { position: 2 })
      expect(updateStatus).toHaveBeenNthCalledWith(2, 'status-triage', { position: 1 })
      expect(listColumnNames()).toEqual(['Inbox', 'Triage', 'Todo', 'Doing'])
    })
  })

  it('moves a column left by swapping positions with the previous status in the same stage', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-triage',
          project_id: 'project-1',
          name: 'Triage',
          stage: 'unstarted' as const,
          color: '#0ea5e9',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-review',
          project_id: 'project-1',
          name: 'Review',
          stage: 'started' as const,
          color: '#f97316',
          icon: '',
          is_default: false,
          description: '',
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-doing',
          project_id: 'project-1',
          name: 'Doing',
          stage: 'started' as const,
          color: '#f59e0b',
          icon: '',
          is_default: false,
          description: '',
          position: 4,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({ tickets: [] })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    updateStatus.mockImplementation(async (statusId: string, patch: { position?: number }) => {
      const status = currentStatuses.statuses.find((item) => item.id === statusId)
      if (!status || typeof patch.position !== 'number') {
        throw new Error(`unknown status update ${statusId}`)
      }
      status.position = patch.position
      return { status: cloneValue(status) }
    })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Doing')
    await fireEvent.click(await findByRole('menuitem', { name: 'Move left' }))

    await waitFor(() => {
      expect(updateStatus).toHaveBeenNthCalledWith(1, 'status-doing', { position: 3 })
      expect(updateStatus).toHaveBeenNthCalledWith(2, 'status-review', { position: 4 })
      expect(listColumnNames()).toEqual(['Todo', 'Triage', 'Doing', 'Review'])
    })
  })

  it('archives all non-archived tickets in a column via the archived flag', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-doing',
          project_id: 'project-1',
          name: 'Doing',
          stage: 'started' as const,
          color: '#f59e0b',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-cancelled',
          project_id: 'project-1',
          name: 'Cancelled',
          stage: 'canceled' as const,
          color: '#4b5563',
          icon: '',
          is_default: false,
          description: '',
          position: 3,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    const currentTickets: Ticket[] = cloneValue([
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-1',
        identifier: 'ASE-202',
        title: 'Wire board page to runtime data',
        status_id: 'status-todo',
        status_name: 'Todo',
        archived: false,
      },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-2',
        identifier: 'ASE-203',
        title: 'Ship workflow runner polish',
        status_id: 'status-todo',
        status_name: 'Todo',
        archived: false,
      },
      {
        ...ticketsFixture.tickets[0],
        id: 'ticket-3',
        identifier: 'ASE-204',
        title: 'Keep existing archived ticket',
        status_id: 'status-cancelled',
        status_name: 'Cancelled',
        archived: true,
      },
    ])

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockImplementation(async () => ({ tickets: cloneValue(currentTickets) }))
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    updateTicket.mockImplementation(async (ticketId: string, patch: { archived?: boolean }) => {
      const ticket = currentTickets.find((item) => item.id === ticketId)
      if (!ticket || typeof patch.archived !== 'boolean') {
        throw new Error(`unknown ticket update ${ticketId}`)
      }
      ticket.archived = patch.archived
      return { ticket: cloneValue(ticket) }
    })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Archive all' }))

    await waitFor(() => {
      expect(updateTicket).toHaveBeenNthCalledWith(1, 'ticket-1', { archived: true })
      expect(updateTicket).toHaveBeenNthCalledWith(2, 'ticket-2', { archived: true })
      expect(toastStore.success).toHaveBeenCalledWith('2 tickets archived.')
    })

    expect(
      within(await findByRole('list', { name: 'Todo tickets' })).queryByText('ASE-202'),
    ).toBeNull()
    expect(
      within(await findByRole('list', { name: 'Todo tickets' })).queryByText('ASE-203'),
    ).toBeNull()
  })

  it('does not call archive when a column only contains archived tickets', async () => {
    appStore.currentProject = projectFixture

    const currentStatuses: StatusPayload = cloneValue({
      statuses: [
        {
          id: 'status-todo',
          project_id: 'project-1',
          name: 'Todo',
          stage: 'unstarted' as const,
          color: '#2563eb',
          icon: '',
          is_default: true,
          description: '',
          position: 1,
          active_runs: 0,
          max_active_runs: null,
        },
        {
          id: 'status-cancelled',
          project_id: 'project-1',
          name: 'Cancelled',
          stage: 'canceled' as const,
          color: '#4b5563',
          icon: '',
          is_default: false,
          description: '',
          position: 2,
          active_runs: 0,
          max_active_runs: null,
        },
      ],
    })

    listStatuses.mockImplementation(async () => cloneValue(currentStatuses))
    listTickets.mockResolvedValue({
      tickets: [
        {
          ...ticketsFixture.tickets[0],
          id: 'ticket-1',
          identifier: 'ASE-202',
          status_id: 'status-cancelled',
          status_name: 'Cancelled',
          archived: true,
        },
      ],
    })
    listWorkflows.mockResolvedValue(workflowsFixture)
    listAgents.mockResolvedValue(agentsFixture)
    listActivity.mockResolvedValue(activityFixture)
    updateStatus.mockResolvedValue({ status: currentStatuses.statuses[0] })
    updateTicket.mockResolvedValue({ ticket: ticketsFixture.tickets[0] })
    connectEventStream.mockReturnValue(() => {})

    const { findByRole } = render(TicketsPage)
    await showEmptyColumns(findByRole)

    await openColumnActionMenu(findByRole, 'Todo')
    await fireEvent.click(await findByRole('menuitem', { name: 'Archive all' }))

    expect(updateTicket).not.toHaveBeenCalled()
    expect(toastStore.error).not.toHaveBeenCalled()
  })
})

function cloneValue<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T
}

function ticketCardFor(container: HTMLElement, identifier: string) {
  const card = within(container).getByText(identifier).closest('button')
  if (!card) throw new Error(`ticket card not found for ${identifier}`)
  return card as HTMLButtonElement
}

async function openColumnActionMenu(
  findByRole: (role: string, options?: Record<string, unknown>) => Promise<HTMLElement>,
  columnName: string,
) {
  const ticketsList = (await findByRole('list', { name: `${columnName} tickets` })) as HTMLElement
  const column = ticketsList.parentElement
  if (!column) throw new Error(`column container not found for ${columnName}`)
  await fireEvent.click(within(column).getByLabelText('Column actions'))
}

async function assertColumnMoveState(
  findByRole: (role: string, options?: Record<string, unknown>) => Promise<HTMLElement>,
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

async function showEmptyColumns(
  findByRole: (role: string, options?: Record<string, unknown>) => Promise<HTMLElement>,
) {
  await fireEvent.click(await findByRole('button', { name: 'Hide empty' }))
}

function listColumnNames() {
  return Array.from(document.querySelectorAll('[role="list"][aria-label$=" tickets"]'))
    .map((node) => node.getAttribute('aria-label')?.replace(/ tickets$/, ''))
    .filter((value): value is string => !!value)
}
