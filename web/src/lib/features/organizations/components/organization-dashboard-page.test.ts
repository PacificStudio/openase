import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider, Organization, Project } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import OrganizationDashboardPage from './organization-dashboard-page.svelte'

const { loadOrganizationDashboardSummary } = vi.hoisted(() => ({
  loadOrganizationDashboardSummary: vi.fn(),
}))

vi.mock('$lib/features/dashboard', async () => {
  const actual =
    await vi.importActual<typeof import('$lib/features/dashboard')>('$lib/features/dashboard')
  return {
    ...actual,
    loadOrganizationDashboardSummary,
  }
})

const organizationFixture: Organization = {
  id: 'org-1',
  name: 'Acme',
  slug: 'acme',
  default_agent_provider_id: 'provider-1',
  status: 'active',
}

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'Todo App',
  slug: 'todo-app',
  description: 'Track tickets and automation',
  status: 'active',
  default_agent_provider_id: 'provider-1',
  max_concurrent_agents: 2,
  accessible_machine_ids: [],
}

const providerFixture: AgentProvider = {
  id: 'provider-1',
  organization_id: 'org-1',
  machine_id: 'machine-1',
  machine_name: 'builder-01',
  machine_host: '127.0.0.1',
  machine_status: 'online',
  machine_ssh_user: null,
  machine_workspace_root: '/workspace',
  name: 'Codex',
  adapter_type: 'codex-app-server',
  permission_profile: 'unrestricted',
  availability_state: 'available',
  available: true,
  availability_checked_at: '2026-04-03T10:00:00Z',
  availability_reason: null,
  capabilities: {
    ephemeral_chat: { state: 'available', reason: null },
  },
  cli_command: 'codex',
  cli_args: [],
  auth_config: {},
  cli_rate_limit: null,
  cli_rate_limit_updated_at: null,
  model_name: 'gpt-5.4',
  model_temperature: 0,
  model_max_tokens: 4096,
  max_parallel_runs: 2,
  cost_per_input_token: 0,
  cost_per_output_token: 0,
  pricing_config: {},
}

describe('OrganizationDashboardPage', () => {
  beforeEach(() => {
    appStore.currentOrg = organizationFixture
    appStore.projects = [projectFixture]
    appStore.providers = [providerFixture]
    appStore.appContextLoading = false
    appStore.appContextFetchedAt = 1

    loadOrganizationDashboardSummary.mockResolvedValue({
      activeProjectCount: 1,
      orgStats: {
        runningAgents: 2,
        activeTickets: 5,
        pendingApprovals: 0,
        ticketSpendToday: 18.75,
        ticketSpendTotal: 120.5,
        ticketsCreatedToday: 2,
        ticketsCompletedToday: 1,
        ticketInputTokens: 1000,
        ticketOutputTokens: 300,
        agentLifetimeTokens: 1400,
        avgCycleMinutes: 32,
        prMergeRate: 0.5,
      },
      projectMetrics: {
        'project-1': {
          runningAgents: 2,
          activeTickets: 5,
          todayCost: 18.75,
          lastActivity: '2026-04-03T09:55:00Z',
        },
      },
    })
  })

  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.projects = []
    appStore.providers = []
    appStore.appContextLoading = false
    appStore.appContextFetchedAt = 0
    vi.clearAllMocks()
  })

  it('renders the org detail page from org-scoped app context', async () => {
    const view = render(OrganizationDashboardPage)

    expect(view.getByRole('heading', { name: 'Acme' })).toBeTruthy()
    expect(view.getByText('1 project · 1 provider')).toBeTruthy()
    expect(view.getByText('Todo App')).toBeTruthy()
    expect(view.getByText('Codex')).toBeTruthy()

    await waitFor(() => {
      expect(loadOrganizationDashboardSummary).toHaveBeenCalledWith(
        'org-1',
        expect.objectContaining({ signal: expect.any(AbortSignal) }),
      )
    })

    const links = view.getAllByRole('link')
    expect(links.some((link) => link.getAttribute('href') === '/orgs/org-1')).toBe(true)
    expect(
      links.some((link) => link.getAttribute('href') === '/orgs/org-1/projects/project-1'),
    ).toBe(true)
    expect(view.getByText('Active Projects')).toBeTruthy()
  })
})
