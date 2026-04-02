import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectEventEnvelope } from '$lib/features/project-events'
import { appStore } from '$lib/stores/app.svelte'
import OrgDashboard from './org-dashboard.svelte'

const projectEventListeners = new Set<(event: ProjectEventEnvelope) => void>()

const {
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  getHRAdvisor,
  getSystemDashboard,
  listActivity,
  listAgents,
  listProjectUpdates,
  listTickets,
  updateProject,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
} = vi.hoisted(() => ({
  createProjectUpdateComment: vi.fn(),
  createProjectUpdateThread: vi.fn(),
  deleteProjectUpdateComment: vi.fn(),
  deleteProjectUpdateThread: vi.fn(),
  getHRAdvisor: vi.fn(),
  getSystemDashboard: vi.fn(),
  listActivity: vi.fn(),
  listAgents: vi.fn(),
  listProjectUpdates: vi.fn(),
  listTickets: vi.fn(),
  updateProject: vi.fn(),
  updateProjectUpdateComment: vi.fn(),
  updateProjectUpdateThread: vi.fn(),
}))

const { loadOrganizationDashboardSummary } = vi.hoisted(() => ({
  loadOrganizationDashboardSummary: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  createProjectUpdateComment,
  createProjectUpdateThread,
  deleteProjectUpdateComment,
  deleteProjectUpdateThread,
  getHRAdvisor,
  getSystemDashboard,
  listActivity,
  listAgents,
  listProjectUpdates,
  listTickets,
  updateProject,
  updateProjectUpdateComment,
  updateProjectUpdateThread,
}))

vi.mock('../organization-summary', () => ({
  loadOrganizationDashboardSummary,
}))

vi.mock('../model', async () => {
  const actual = await vi.importActual<typeof import('../model')>('../model')
  return {
    ...actual,
    shouldShowProjectOnboarding: vi.fn(() => false),
  }
})

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/features/onboarding', () => ({
  markProjectOnboardingCompleted: vi.fn(),
  readProjectOnboardingCompletion: vi.fn(() => true),
  OnboardingPanel: class {},
}))

vi.mock('$lib/features/project-events', async () => {
  const actual = await vi.importActual<typeof import('$lib/features/project-events')>(
    '$lib/features/project-events',
  )
  return {
    ...actual,
    subscribeProjectEvents: vi.fn((_: string, listener: (event: ProjectEventEnvelope) => void) => {
      projectEventListeners.add(listener)
      return () => {
        projectEventListeners.delete(listener)
      }
    }),
  }
})

const agentPayload = {
  agents: [
    {
      id: 'agent-1',
      provider_id: 'provider-1',
      project_id: 'project-1',
      name: 'Coding Agent',
      runtime_control_state: 'active',
      total_tokens_used: 12,
      total_tickets_completed: 0,
      runtime: {
        active_run_count: 1,
        current_run_id: 'run-1',
        status: 'running',
        current_ticket_id: 'ticket-1',
        session_id: 'session-1',
        runtime_phase: 'executing',
        runtime_started_at: '2026-04-02T10:00:00Z',
        last_error: '',
        last_heartbeat_at: '2026-04-02T10:00:05Z',
        current_step_status: 'executing',
        current_step_summary: 'Running checks',
        current_step_changed_at: '2026-04-02T10:00:05Z',
      },
    },
  ],
}

const ticketPayload = {
  tickets: [
    {
      id: 'ticket-1',
      identifier: 'ASE-101',
      project_id: 'project-1',
      title: 'Fix dashboard refresh',
      description: 'Investigate project dashboard traffic.',
      status_id: 'status-1',
      status_name: 'In Progress',
      priority: 'high',
      type: 'task',
      workflow_id: null,
      workflow_name: '',
      created_by: 'user:test',
      created_at: '2026-04-02T09:00:00Z',
      updated_at: '2026-04-02T10:00:00Z',
      budget_usd: 10,
      cost_amount: 1.5,
      cost_tokens_input: 100,
      cost_tokens_output: 50,
      retry_paused: false,
      pause_reason: '',
      children: [],
      dependencies: [],
      external_links: [],
      repo_scopes: [],
    },
  ],
}

const activityPayload = {
  events: [
    {
      id: 'activity-1',
      project_id: 'project-1',
      ticket_id: 'ticket-1',
      agent_id: 'agent-1',
      event_type: 'ticket.updated',
      message: 'Updated ticket ASE-101',
      metadata: { agent_name: 'Coding Agent' },
      created_at: '2026-04-02T10:00:00Z',
    },
  ],
}

const systemDashboardPayload = {
  memory: {
    heap_alloc_bytes: 10,
    heap_inuse_bytes: 20,
    heap_idle_bytes: 5,
    heap_released_bytes: 3,
    stack_inuse_bytes: 4,
    sys_bytes: 30,
    gc_cycles: 2,
    goroutines: 8,
  },
}

const hrAdvisorPayload = {
  summary: {
    open_tickets: 1,
    coding_tickets: 1,
    failing_tickets: 0,
    blocked_tickets: 0,
    active_agents: 1,
    workflow_count: 1,
    recent_activity_count: 1,
    active_workflow_types: ['coding'],
  },
  staffing: {
    developers: 1,
    qa: 0,
    docs: 0,
    security: 0,
    product: 0,
    research: 0,
  },
  recommendations: [],
}

const organizationSummaryPayload = {
  activeProjectCount: 1,
  orgStats: {
    runningAgents: 1,
    activeTickets: 1,
    pendingApprovals: 0,
    ticketSpendToday: 4.2,
    ticketSpendTotal: 0,
    ticketsCreatedToday: 0,
    ticketsCompletedToday: 0,
    ticketInputTokens: 0,
    ticketOutputTokens: 0,
    agentLifetimeTokens: 0,
    avgCycleMinutes: 0,
    prMergeRate: 0,
  },
  projectMetrics: {
    'project-1': {
      runningAgents: 1,
      activeTickets: 1,
      todayCost: 4.2,
      lastActivity: '2026-04-02T10:00:00Z',
    },
  },
}

describe('OrgDashboard', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    appStore.currentOrg = {
      id: 'org-1',
      name: 'Acme',
      slug: 'acme',
      default_agent_provider_id: 'provider-1',
      status: 'active',
    }
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'Project One',
      slug: 'project-one',
      description: 'Project dashboard',
      status: 'In Progress',
      default_agent_provider_id: 'provider-1',
      max_concurrent_agents: 1,
      accessible_machine_ids: [],
    }

    listAgents.mockResolvedValue(agentPayload)
    listTickets.mockResolvedValue(ticketPayload)
    listActivity.mockResolvedValue(activityPayload)
    getSystemDashboard.mockResolvedValue(systemDashboardPayload)
    getHRAdvisor.mockResolvedValue(hrAdvisorPayload)
    listProjectUpdates.mockResolvedValue({ threads: [] })
    loadOrganizationDashboardSummary.mockResolvedValue(organizationSummaryPayload)
  })

  afterEach(() => {
    cleanup()
    projectEventListeners.clear()
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('loads once, stops 1s polling, and only refreshes memory on the slow interval', async () => {
    render(OrgDashboard)

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(1)
      expect(listTickets).toHaveBeenCalledTimes(1)
      expect(listActivity).toHaveBeenCalledTimes(1)
      expect(getSystemDashboard).toHaveBeenCalledTimes(1)
      expect(getHRAdvisor).toHaveBeenCalledTimes(1)
      expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
      expect(listProjectUpdates).toHaveBeenCalledTimes(1)
    })

    await vi.advanceTimersByTimeAsync(3000)

    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(getSystemDashboard).toHaveBeenCalledTimes(1)
    expect(getHRAdvisor).toHaveBeenCalledTimes(1)
    expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)

    await vi.advanceTimersByTimeAsync(10_000)

    await waitFor(() => {
      expect(getSystemDashboard).toHaveBeenCalledTimes(2)
    })

    expect(listAgents).toHaveBeenCalledTimes(1)
    expect(listTickets).toHaveBeenCalledTimes(1)
    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(getHRAdvisor).toHaveBeenCalledTimes(1)
    expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
  })

  it('refreshes only the dirty dashboard slices when the project bus emits a coalesced refresh event', async () => {
    render(OrgDashboard)

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(1)
      expect(listTickets).toHaveBeenCalledTimes(1)
      expect(listActivity).toHaveBeenCalledTimes(1)
      expect(getSystemDashboard).toHaveBeenCalledTimes(1)
      expect(getHRAdvisor).toHaveBeenCalledTimes(1)
      expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
    })

    for (const listener of [...projectEventListeners]) {
      listener({
        topic: 'project.dashboard.events',
        type: 'project.dashboard.refresh',
        payload: {
          project_id: 'project-1',
          dirty_sections: ['agents', 'tickets'],
        },
        publishedAt: '2026-04-02T10:00:01Z',
      })
    }

    await waitFor(() => {
      expect(listAgents).toHaveBeenCalledTimes(2)
      expect(listTickets).toHaveBeenCalledTimes(2)
    })

    expect(listActivity).toHaveBeenCalledTimes(1)
    expect(getSystemDashboard).toHaveBeenCalledTimes(1)
    expect(getHRAdvisor).toHaveBeenCalledTimes(1)
    expect(loadOrganizationDashboardSummary).toHaveBeenCalledTimes(1)
  })
})
