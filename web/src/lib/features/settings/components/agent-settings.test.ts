import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { AgentProvider, Organization, Project } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import AgentSettings from './agent-settings.svelte'

const {
  createProvider,
  listAgents,
  listMachines,
  listProviderModelOptions,
  listProviders,
  updateProject,
} = vi.hoisted(() => ({
  createProvider: vi.fn(),
  listAgents: vi.fn(),
  listMachines: vi.fn(),
  listProviderModelOptions: vi.fn(),
  listProviders: vi.fn(),
  updateProject: vi.fn(),
}))

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    warning: vi.fn(),
  },
}))

vi.mock('$lib/api/openase', () => ({
  createProvider,
  listAgents,
  listMachines,
  listProviderModelOptions,
  listProviders,
  updateProject,
  updateProvider: vi.fn(),
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

afterEach(() => {
  cleanup()
  appStore.currentOrg = null
  appStore.currentProject = null
  vi.clearAllMocks()
})

describe('Agent settings add provider entry', () => {
  it('shows the add provider entry and explains why it is disabled without machines', async () => {
    seedAppContext()
    listProviders.mockResolvedValue({ providers: [] })
    listAgents.mockResolvedValue({ agents: [] })
    listMachines.mockResolvedValue({ machines: [] })

    const { getByRole, getByText } = render(AgentSettings)

    await waitFor(() => expect(listProviders).toHaveBeenCalledWith('org-1'))

    expect((getByRole('button', { name: 'Add provider' }) as HTMLButtonElement).disabled).toBe(true)
    expect(
      getByText(
        'Register an execution machine in this organization before creating a provider from project settings.',
      ),
    ).toBeTruthy()
  })
})

// eslint-disable-next-line max-lines-per-function
describe('Agent settings provider creation', () => {
  it('creates a provider from project settings', async () => {
    seedAppContext()
    listProviders.mockResolvedValue({ providers: [] })
    listAgents.mockResolvedValue({ agents: [] })
    listMachines.mockResolvedValue({ machines: [machineFixture()] })
    listProviderModelOptions.mockResolvedValue({ adapter_model_options: [] })
    createProvider.mockResolvedValue({
      provider: providerFixture({ id: 'provider-2', name: 'Project Provider' }),
    })

    const { findByRole, findByLabelText, findByText, getByRole } = render(AgentSettings)

    await fireEvent.click(await findByRole('button', { name: 'Add provider' }))
    await waitFor(() => expect(listProviderModelOptions).toHaveBeenCalledTimes(1))

    await fireEvent.input(await findByLabelText('Provider name'), {
      target: { value: 'Project Provider' },
    })
    await fireEvent.input(await findByLabelText('Model name'), {
      target: { value: 'gpt-5.4' },
    })
    await fireEvent.click(getByRole('button', { name: 'Create provider' }))

    await waitFor(() =>
      expect(createProvider).toHaveBeenCalledWith('org-1', {
        machine_id: 'machine-1',
        name: 'Project Provider',
        adapter_type: 'custom',
        permission_profile: 'unrestricted',
        cli_command: '',
        cli_args: [],
        auth_config: {},
        model_name: 'gpt-5.4',
        model_temperature: 0,
        model_max_tokens: 1,
        max_parallel_runs: 0,
        cost_per_input_token: 0,
        cost_per_output_token: 0,
        pricing_config: {
          source_kind: 'custom',
          pricing_mode: 'flat',
          rates: {
            input_per_token: 0,
            output_per_token: 0,
          },
        },
      }),
    )

    expect(await findByText('Project Provider')).toBeTruthy()
  })

  it('refreshes provider state so the new provider can be selected immediately', async () => {
    seedAppContext()
    listProviders.mockResolvedValue({ providers: [] })
    listAgents.mockResolvedValue({ agents: [] })
    listMachines.mockResolvedValue({ machines: [machineFixture()] })
    listProviderModelOptions.mockResolvedValue({ adapter_model_options: [] })
    createProvider.mockResolvedValue({
      provider: providerFixture({ id: 'provider-2', name: 'Project Provider' }),
    })
    updateProject.mockResolvedValue({
      project: currentProject({ default_agent_provider_id: 'provider-2' }),
    })

    const { findByRole, findByLabelText, findByText, getByRole, getByTitle } = render(AgentSettings)

    await fireEvent.click(await findByRole('button', { name: 'Add provider' }))
    await waitFor(() => expect(listProviderModelOptions).toHaveBeenCalledTimes(1))

    await fireEvent.input(await findByLabelText('Provider name'), {
      target: { value: 'Project Provider' },
    })
    await fireEvent.input(await findByLabelText('Model name'), {
      target: { value: 'gpt-5.4' },
    })
    await fireEvent.click(getByRole('button', { name: 'Create provider' }))
    await findByText('Project Provider')

    await fireEvent.click(getByTitle('Set Project Provider as project default'))
    await fireEvent.click(getByRole('button', { name: 'Save default' }))

    await waitFor(() =>
      expect(updateProject).toHaveBeenCalledWith('project-1', {
        default_agent_provider_id: 'provider-2',
      }),
    )
  })
})

function seedAppContext() {
  appStore.currentOrg = currentOrg()
  appStore.currentProject = currentProject()
}

function currentOrg(overrides: Partial<Organization> = {}): Organization {
  return {
    id: 'org-1',
    name: 'Acme',
    slug: 'acme',
    status: 'active',
    default_agent_provider_id: null,
    ...overrides,
  }
}

function currentProject(overrides: Partial<Project> = {}): Project {
  return {
    id: 'project-1',
    organization_id: 'org-1',
    name: 'OpenASE',
    slug: 'openase',
    description: '',
    status: 'active',
    default_agent_provider_id: null,
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
    agent_run_summary_prompt: '',
    effective_agent_run_summary_prompt: '',
    agent_run_summary_prompt_source: 'builtin',
    ...overrides,
  }
}

function machineFixture() {
  return {
    id: 'machine-1',
    organization_id: 'org-1',
    name: 'Machine One',
    host: '127.0.0.1',
    port: 22,
    ssh_user: null,
    ssh_key_path: null,
    description: '',
    labels: [],
    status: 'online',
    workspace_root: '/workspace',
    agent_cli_path: null,
    env_vars: [],
    last_heartbeat_at: null,
    resources: {},
  }
}

function providerFixture(overrides: Partial<AgentProvider> = {}): AgentProvider {
  return {
    id: 'provider-1',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Machine One',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Codex',
    adapter_type: 'custom',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-04-04T10:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: { state: 'available', reason: null },
    },
    cli_command: '',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 1,
    max_parallel_runs: 0,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {
      source_kind: 'custom',
      pricing_mode: 'flat',
      rates: {
        input_per_token: 0,
        output_per_token: 0,
      },
    },
    ...overrides,
  }
}
