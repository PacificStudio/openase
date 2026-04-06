import type { Organization, Project } from '$lib/api/contracts'
import type { AgentsPageData } from '../data'
import type { AgentInstance } from '../types'

export const projectFixture: Project = {
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
    permissionProfile: 'unrestricted',
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
  } as AgentInstance
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
        permission_profile: 'unrestricted',
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
        cli_rate_limit: null,
        cli_rate_limit_updated_at: null,
        model_name: 'gpt-5.4',
        model_temperature: 0,
        model_max_tokens: 4096,
        max_parallel_runs: 2,
        cost_per_input_token: 0,
        cost_per_output_token: 0,
        pricing_config: {},
      },
    ],
    machineItems: [
      {
        id: 'machine-1',
        organization_id: 'org-1',
        name: 'Localhost',
        host: '127.0.0.1',
        port: 22,
        reachability_mode: 'direct_connect',
        execution_mode: 'websocket',
        execution_capabilities: [
          'probe',
          'workspace_prepare',
          'artifact_sync',
          'process_streaming',
        ],
        ssh_helper_enabled: false,
        ssh_user: null,
        ssh_key_path: null,
        advertised_endpoint: null,
        daemon_status: {
          registered: false,
          last_registered_at: null,
          current_session_id: null,
          session_state: 'unknown',
        },
        detected_os: 'unknown',
        detected_arch: 'unknown',
        detection_status: 'unknown',
        detection_message:
          'Operating system and architecture are still unknown. You can keep configuring the machine and verify the platform manually.',
        channel_credential: {
          kind: 'none',
          token_id: null,
          certificate_id: null,
        },
        description: '',
        labels: [],
        status: 'online',
        workspace_root: '/workspace',
        agent_cli_path: null,
        env_vars: [],
        last_heartbeat_at: null,
        resources: {},
      },
    ],
  }
}
