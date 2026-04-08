import { cleanup, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider, Organization } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import OrganizationsPage from './organizations-page.svelte'

const { loadWorkspaceDashboardSummary } = vi.hoisted(() => ({
  loadWorkspaceDashboardSummary: vi.fn(),
}))

vi.mock('$lib/features/dashboard', async () => {
  const actual =
    await vi.importActual<typeof import('$lib/features/dashboard')>('$lib/features/dashboard')
  return {
    ...actual,
    loadWorkspaceDashboardSummary,
  }
})

const organizationFixtures: Organization[] = [
  {
    id: 'org-1',
    name: 'Acme',
    slug: 'acme',
    default_agent_provider_id: null,
    status: 'active',
  },
  {
    id: 'org-2',
    name: 'Beta',
    slug: 'beta',
    default_agent_provider_id: null,
    status: 'active',
  },
]

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

describe('OrganizationsPage', () => {
  beforeEach(() => {
    appStore.organizations = organizationFixtures
    appStore.providers = [providerFixture]
    appStore.appContextLoading = false
    appStore.appContextFetchedAt = 1

    loadWorkspaceDashboardSummary.mockResolvedValue({
      orgMetrics: {
        'org-1': {
          projectCount: 2,
          providerCount: 1,
          runningAgents: 3,
          activeTickets: 4,
          todayCost: 12.34,
        },
        'org-2': {
          projectCount: 1,
          providerCount: 0,
          runningAgents: 0,
          activeTickets: 1,
          todayCost: 0,
        },
      },
      workspaceStats: {
        runningAgents: 3,
        activeTickets: 5,
        todayCost: 12.34,
        totalTokens: 1200,
      },
      totalProjects: 3,
    })
  })

  afterEach(() => {
    cleanup()
    appStore.organizations = []
    appStore.providers = []
    appStore.appContextLoading = false
    appStore.appContextFetchedAt = 0
    vi.clearAllMocks()
  })

  it('renders the organizations loaded into app context for /orgs', async () => {
    const view = render(OrganizationsPage)
    const scrollContainer = view.getByTestId('route-scroll-container')

    expect(view.getByText('Workspace')).toBeTruthy()
    expect(view.getByText('Acme')).toBeTruthy()
    expect(view.getByText('Beta')).toBeTruthy()
    expect(scrollContainer.className).toContain('min-h-0')
    expect(scrollContainer.className).toContain('flex-1')
    expect(scrollContainer.className).toContain('overflow-y-auto')

    await waitFor(() => {
      expect(loadWorkspaceDashboardSummary).toHaveBeenCalledTimes(1)
    })

    const links = view.getAllByRole('link')
    expect(links.some((link) => link.getAttribute('href') === '/orgs/org-1')).toBe(true)
    expect(links.some((link) => link.getAttribute('href') === '/orgs/org-2')).toBe(true)
    expect(view.getByText('2 organizations · 3 projects · 1 provider')).toBeTruthy()
    expect(view.getByText('2 projects')).toBeTruthy()
    expect(view.getByText('1 project')).toBeTruthy()
  })

  it('shows the empty workspace state once app context finishes loading with no organizations', async () => {
    appStore.organizations = []
    appStore.providers = []
    appStore.appContextLoading = false
    loadWorkspaceDashboardSummary.mockResolvedValue({
      orgMetrics: {},
      workspaceStats: {
        runningAgents: 0,
        activeTickets: 0,
        todayCost: 0,
        totalTokens: 0,
      },
      totalProjects: 0,
    })

    const view = render(OrganizationsPage)

    expect(view.getByTestId('route-scroll-container').className).toContain('overflow-y-auto')
    expect(view.getByText('No organizations yet.')).toBeTruthy()
    expect(view.getByText('Create your first organization to get started')).toBeTruthy()

    await waitFor(() => {
      expect(loadWorkspaceDashboardSummary).toHaveBeenCalledTimes(1)
    })
  })
})
