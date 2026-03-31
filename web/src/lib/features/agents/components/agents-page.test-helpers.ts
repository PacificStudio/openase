import type {
  AgentOutputPayload,
  AgentStepPayload,
  Organization,
  Project,
} from '$lib/api/contracts'
import type { AgentsPageData } from '../data'
import type { AgentInstance } from '../types'

export const projectFixture: Project = {
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

export const orgFixture: Organization = {
  id: 'org-1',
  name: 'Acme',
  slug: 'acme',
  status: 'active',
  default_agent_provider_id: null,
}

export function makeAgent(overrides: Partial<AgentInstance> = {}): AgentInstance {
  return {
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
    ...overrides,
  }
}

export function makePageData(agent: AgentInstance): AgentsPageData {
  return {
    agents: [agent],
    agentRuns: [
      {
        id: 'run-1',
        agentId: agent.id,
        agentName: agent.name,
        providerId: agent.providerId,
        providerName: agent.providerName,
        modelName: agent.modelName,
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
    providers: [],
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
          ephemeral_chat: { state: 'available', reason: null },
        },
        cli_command: 'codex',
        cli_args: [],
        auth_config: {},
        model_name: 'gpt-5.4',
        model_temperature: 0,
        model_max_tokens: 4096,
        max_parallel_runs: 2,
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
}

export const outputEntriesFixture: AgentOutputPayload = {
  entries: [
    {
      id: 'output-1',
      project_id: 'project-1',
      agent_id: 'agent-1',
      agent_run_id: 'run-1',
      ticket_id: 'ticket-1',
      stream: 'assistant',
      output: 'Inspecting runtime events.',
      created_at: '2026-03-27T12:00:01Z',
    },
    {
      id: 'output-2',
      project_id: 'project-1',
      agent_id: 'agent-1',
      agent_run_id: 'run-1',
      ticket_id: 'ticket-1',
      stream: 'tool',
      output: 'read_file("/src/main.ts")',
      created_at: '2026-03-27T12:00:02Z',
    },
  ],
}

export const stepEntriesFixture: AgentStepPayload = {
  entries: [
    {
      id: 'step-1',
      project_id: 'project-1',
      agent_id: 'agent-1',
      ticket_id: 'ticket-1',
      agent_run_id: 'run-1',
      source_trace_event_id: null,
      step_status: 'planning',
      summary: 'Analyzing pipeline structure.',
      created_at: '2026-03-27T12:00:01Z',
    },
  ],
}

export const emptyOutputFixture: AgentOutputPayload = { entries: [] }
export const emptyStepFixture: AgentStepPayload = { entries: [] }
