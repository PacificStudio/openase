import { appStore } from '$lib/stores/app.svelte'

export const agentPayload = {
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

export const ticketPayload = {
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
      cost_tokens_total: 150,
      retry_paused: false,
      pause_reason: '',
      children: [],
      dependencies: [],
      external_links: [],
      repo_scopes: [],
    },
  ],
}

export const activityPayload = {
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

export const systemDashboardPayload = {
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

export const hrAdvisorPayload = {
  summary: {
    open_tickets: 1,
    coding_tickets: 1,
    failing_tickets: 0,
    blocked_tickets: 0,
    active_agents: 1,
    workflow_count: 1,
    recent_activity_count: 1,
    active_workflow_families: ['coding'],
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

export const organizationSummaryPayload = {
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

export function seedOrgDashboardStore() {
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
}
