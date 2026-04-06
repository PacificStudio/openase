import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import StepProvider from './step-provider.svelte'

const { listProviders, refreshMachineHealth, updateProject } = vi.hoisted(() => ({
  listProviders: vi.fn(),
  refreshMachineHealth: vi.fn(),
  updateProject: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  listProviders,
  refreshMachineHealth,
  updateProject,
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('StepProvider', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
  })

  beforeEach(() => {
    appStore.currentProject = {
      id: 'project-1',
      organization_id: 'org-1',
      name: 'OpenASE',
      slug: 'openase',
      description: '',
      status: 'Planned',
      default_agent_provider_id: null,
      accessible_machine_ids: [],
      max_concurrent_agents: 4,
    }
    listProviders.mockResolvedValue({ providers: [] })
    refreshMachineHealth.mockResolvedValue({ machine: { id: 'machine-1' } })
  })

  afterEach(() => {
    cleanup()
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('always renders the built-in onboarding cards even before any provider is registered', async () => {
    const { findByText, getAllByText } = render(StepProvider, {
      props: {
        projectId: 'project-1',
        orgId: 'org-1',
        initialState: {
          providers: [],
          selectedProviderId: '',
        },
        onComplete: vi.fn(),
      },
    })

    expect(getAllByText('Claude Code').length).toBeGreaterThan(0)
    expect(getAllByText('OpenAI Codex').length).toBeGreaterThan(0)
    expect(getAllByText('Gemini CLI').length).toBeGreaterThan(0)
    expect(await findByText('No providers registered yet')).toBeTruthy()
  })

  it('opens a provider-specific guide with install, auth, and verify commands for unavailable providers', async () => {
    listProviders.mockResolvedValue({
      providers: [makeProvider({ availability_reason: 'not_logged_in' })],
    })

    const { findAllByRole, findByText } = render(StepProvider, {
      props: {
        projectId: 'project-1',
        orgId: 'org-1',
        initialState: {
          providers: [makeProvider({ availability_reason: 'not_logged_in' })],
          selectedProviderId: '',
        },
        onComplete: vi.fn(),
      },
    })

    const guideButtons = await findAllByRole('button', { name: 'Guide' })
    await fireEvent.click(guideButtons[1]!)

    expect(await findByText('OpenAI Codex setup guide')).toBeTruthy()
    expect(await findByText('npm i -g @openai/codex')).toBeTruthy()
    expect(await findByText('codex --login')).toBeTruthy()
    expect(await findByText('codex --version')).toBeTruthy()
  })

  it('lets users select an available provider as the project default from onboarding', async () => {
    const provider = makeProvider({
      availability_state: 'available',
      availability_reason: null,
      available: true,
    })
    const onComplete = vi.fn()

    listProviders.mockResolvedValue({ providers: [provider] })
    updateProject.mockResolvedValue({
      project: {
        id: 'project-1',
        organization_id: 'org-1',
        name: 'OpenASE',
        slug: 'openase',
        description: '',
        status: 'Planned',
        default_agent_provider_id: provider.id,
        accessible_machine_ids: [],
        max_concurrent_agents: 4,
      },
    })

    const { getAllByText } = render(StepProvider, {
      props: {
        projectId: 'project-1',
        orgId: 'org-1',
        initialState: {
          providers: [provider],
          selectedProviderId: '',
        },
        onComplete,
      },
    })

    await fireEvent.click(getAllByText('Use this provider')[0]!)

    await waitFor(() => {
      expect(updateProject).toHaveBeenCalledWith('project-1', {
        default_agent_provider_id: provider.id,
      })
      expect(onComplete).toHaveBeenCalledWith(provider.id)
    })
  })
})

function makeProvider(overrides: Partial<AgentProvider> = {}): AgentProvider {
  return {
    id: 'provider-codex',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'local',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'OpenAI Codex Local',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'unavailable',
    available: false,
    availability_checked_at: '2026-04-01T12:00:00Z',
    availability_reason: 'not_logged_in',
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
    cli_command: 'codex',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 16384,
    max_parallel_runs: 1,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
    ...overrides,
  }
}
