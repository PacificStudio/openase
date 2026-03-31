import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type {
  AgentOutputPayload,
  AgentStepPayload,
  Organization,
  Project,
} from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import type { AgentsPageData } from '../data'
import AgentsPage from './agents-page.svelte'

const { listAgentOutput, listAgentSteps, connectEventStream, loadAgentsPageResult } = vi.hoisted(
  () => ({
    listAgentOutput: vi.fn(),
    listAgentSteps: vi.fn(),
    connectEventStream: vi.fn(),
    loadAgentsPageResult: vi.fn(),
  }),
)

vi.mock('$lib/api/openase', () => ({
  listAgentOutput,
  listAgentSteps,
}))

vi.mock('$lib/api/sse', () => ({
  connectEventStream,
}))

vi.mock('../page-data', () => ({
  loadAgentsPageResult,
}))

const projectFixture: Project = {
  id: 'project-1',
  organization_id: 'org-1',
  name: 'OpenASE',
  slug: 'openase',
  description: '',
  status: 'active',
  default_workflow_id: null,
  default_agent_provider_id: null,
  accessible_machine_ids: [],
  max_concurrent_agents: 4,
}

const orgFixture: Organization = {
  id: 'org-1',
  name: 'Acme',
  slug: 'acme',
  status: 'active',
  default_agent_provider_id: null,
}

const agentsPageDataFixture: AgentsPageData = {
  agents: [
    {
      id: 'agent-1',
      name: 'Codex Worker',
      providerId: 'provider-1',
      providerName: 'Codex',
      modelName: 'gpt-5.4',
      status: 'running',
      runtimePhase: 'ready',
      runtimeControlState: 'active',
      activeRunCount: 1,
      currentTicket: {
        id: 'ticket-1',
        identifier: 'ASE-277',
        title: 'Align runtime event pipeline',
      },
      sessionId: 'session-1',
      currentStepStatus: 'planning',
      currentStepSummary: 'Inspecting runtime events.',
      currentStepChangedAt: '2026-03-27T12:00:00Z',
      todayCompleted: 0,
      todayCost: 0,
    },
  ],
  agentRuns: [
    {
      id: 'run-1',
      agentId: 'agent-1',
      agentName: 'Codex Worker',
      providerId: 'provider-1',
      providerName: 'Codex',
      modelName: 'gpt-5.4',
      workflowId: 'workflow-1',
      workflowName: 'Coding',
      status: 'ready',
      ticket: {
        id: 'ticket-1',
        identifier: 'ASE-277',
        title: 'Align runtime event pipeline',
      },
      lastHeartbeat: '2026-03-27T12:00:00Z',
      runtimeStartedAt: '2026-03-27T12:00:00Z',
      sessionId: 'session-1',
      lastError: '',
      createdAt: '2026-03-27T12:00:00Z',
    },
  ],
  providers: [
    {
      id: 'provider-1',
      machineId: 'machine-1',
      machineName: 'Localhost',
      machineHost: '127.0.0.1',
      machineStatus: 'online',
      machineWorkspaceRoot: '/workspace',
      name: 'Codex',
      adapterType: 'codex-app-server',
      availabilityState: 'ready',
      available: true,
      availabilityCheckedAt: '2026-03-27T12:00:00Z',
      availabilityReason: null,
      cliCommand: 'codex',
      cliArgs: [],
      authConfig: {},
      modelName: 'gpt-5.4',
      modelTemperature: 0,
      modelMaxTokens: 4096,
      costPerInputToken: 0,
      costPerOutputToken: 0,
      agentCount: 1,
      isDefault: true,
    },
  ],
  providerItems: [
    {
      id: 'provider-1',
      organization_id: 'org-1',
      machine_id: 'machine-1',
      machine_name: 'Localhost',
      machine_host: '127.0.0.1',
      machine_status: 'online',
      machine_ssh_user: null,
      machine_workspace_root: '/workspace',
      name: 'Codex',
      adapter_type: 'codex-app-server',
      availability_state: 'ready',
      available: true,
      availability_checked_at: '2026-03-27T12:00:00Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'codex',
      cli_args: [],
      auth_config: {},
      model_name: 'gpt-5.4',
      model_temperature: 0,
      model_max_tokens: 4096,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
    },
  ],
  machineItems: [
    {
      id: 'machine-1',
      organization_id: 'org-1',
      name: 'Localhost',
      host: '127.0.0.1',
      port: 22,
      ssh_user: null,
      ssh_key_path: null,
      description: '',
      labels: [],
      status: 'online',
      workspace_root: '/workspace',
      mirror_root: '/mirrors',
      agent_cli_path: null,
      env_vars: [],
      last_heartbeat_at: null,
      resources: {},
    },
  ],
}

const emptyOutputFixture: AgentOutputPayload = { entries: [] }
const emptyStepFixture: AgentStepPayload = { entries: [] }

describe('AgentsPage', () => {
  afterEach(() => {
    cleanup()
    appStore.currentOrg = null
    appStore.currentProject = null
    vi.clearAllMocks()
  })

  it('opens separate trace and step streams for the output drawer', async () => {
    appStore.currentOrg = orgFixture
    appStore.currentProject = projectFixture

    loadAgentsPageResult.mockResolvedValue({ ok: true, data: agentsPageDataFixture })
    listAgentOutput.mockResolvedValue(emptyOutputFixture)
    listAgentSteps.mockResolvedValue(emptyStepFixture)
    connectEventStream.mockReturnValue(() => {})

    const { findAllByLabelText } = render(AgentsPage)

    const buttons = await findAllByLabelText('View output')
    await fireEvent.click(buttons[0]!)

    await waitFor(() => {
      expect(connectEventStream).toHaveBeenCalledWith(
        '/api/v1/projects/project-1/agents/agent-1/output/stream',
        expect.any(Object),
      )
      expect(connectEventStream).toHaveBeenCalledWith(
        '/api/v1/projects/project-1/agents/agent-1/steps/stream',
        expect.any(Object),
      )
    })
  })
})
