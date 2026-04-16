import type { MockState } from './constants'
import {
  CLAUDE_PROVIDER_ID,
  DEFAULT_AGENT_ID,
  DEFAULT_JOB_ID,
  DEFAULT_PROVIDER_ID,
  DEFAULT_REPO_ID,
  DEFAULT_STATUS_IDS,
  DEFAULT_TICKET_ID,
  DEFAULT_WORKFLOW_ID,
  GEMINI_PROVIDER_ID,
  GPU_MACHINE_ID,
  LOCAL_MACHINE_ID,
  OPENAI_PROVIDER_ID,
  ORG_ID,
  PROJECT_ID,
  nowIso,
} from './constants'
import { createDefaultSecuritySettings } from './security'
import { createMockTicketRecord } from './ticket-data'
import { defaultHarnessContent, machineResourceSnapshot } from './entities'
import { asNumber, asString } from './helpers'

export function createInitialState(): MockState {
  const organizations = [
    {
      id: ORG_ID,
      name: 'E2E Org',
      slug: 'e2e-org',
      default_agent_provider_id: DEFAULT_PROVIDER_ID,
    },
  ]

  const projects = [
    {
      id: PROJECT_ID,
      org_id: ORG_ID,
      name: 'TodoApp',
      slug: 'todo-app',
      description: 'Playwright fixture project',
      status: 'active',
      default_agent_provider_id: DEFAULT_PROVIDER_ID,
      max_concurrent_agents: 0,
    },
  ]

  const machines = [
    {
      id: LOCAL_MACHINE_ID,
      org_id: ORG_ID,
      name: 'local',
      host: 'local',
      port: 22,
      reachability_mode: 'local',
      execution_mode: 'local_process',
      execution_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
      ssh_helper_enabled: false,
      ssh_user: '',
      ssh_key_path: '',
      advertised_endpoint: null,
      daemon_status: {
        registered: true,
        last_registered_at: nowIso,
        current_session_id: null,
        session_state: 'connected',
      },
      detected_os: 'linux',
      detected_arch: 'amd64',
      detection_status: 'ok',
      detection_message: 'Detected amd64 on Linux.',
      channel_credential: {
        kind: 'none',
        token_id: null,
        certificate_id: null,
      },
      description: 'Seeded local runner',
      labels: ['local', 'default'],
      status: 'online',
      workspace_root: '~/.openase/workspace',
      agent_cli_path: '/usr/local/bin/codex',
      env_vars: ['OPENASE_ENV=dev'],
      last_heartbeat_at: nowIso,
      resources: machineResourceSnapshot('local', {
        cpuUsagePercent: 18,
        memoryTotalGB: 16,
        memoryUsedGB: 5.5,
        memoryAvailableGB: 10.5,
        diskTotalGB: 512,
        diskAvailableGB: 320,
        gpuDispatchable: false,
        gpus: [],
      }),
    },
    {
      id: GPU_MACHINE_ID,
      org_id: ORG_ID,
      name: 'gpu-runner-01',
      host: '10.0.0.42',
      port: 22,
      reachability_mode: 'direct_connect',
      execution_mode: 'websocket',
      execution_capabilities: ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming'],
      ssh_helper_enabled: true,
      ssh_user: 'openase',
      ssh_key_path: '~/.ssh/openase_rsa',
      advertised_endpoint: 'ws://10.0.0.42:19840/runtime',
      daemon_status: {
        registered: false,
        last_registered_at: null,
        current_session_id: null,
        session_state: 'unknown',
      },
      detected_os: 'linux',
      detected_arch: 'amd64',
      detection_status: 'ok',
      detection_message: 'Detected amd64 on Linux.',
      channel_credential: {
        kind: 'none',
        token_id: null,
        certificate_id: null,
      },
      description: 'Primary remote runner',
      labels: ['gpu', 'remote'],
      status: 'online',
      workspace_root: '/srv/openase/workspace',
      agent_cli_path: '/usr/bin/codex',
      env_vars: ['CUDA_VISIBLE_DEVICES=0'],
      last_heartbeat_at: nowIso,
      resources: machineResourceSnapshot('ssh', {
        cpuUsagePercent: 46,
        memoryTotalGB: 64,
        memoryUsedGB: 21.4,
        memoryAvailableGB: 42.6,
        diskTotalGB: 1024,
        diskAvailableGB: 612,
        gpuDispatchable: true,
        gpus: [
          {
            index: 0,
            name: 'NVIDIA A100',
            memory_total_gb: 80,
            memory_used_gb: 22,
            utilization_percent: 54,
          },
        ],
      }),
    },
  ]

  const providers = [
    {
      id: DEFAULT_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'Fake Codex Validation Provider',
      adapter_type: 'codex-app-server',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'python3',
      cli_args: ['/home/user/workspace/openase/scripts/dev/fake_codex_app_server.py'],
      auth_config: {},
      secret_bindings: [],
      model_name: 'gpt-5.4',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
    {
      id: CLAUDE_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'Claude Code',
      adapter_type: 'claude-code-cli',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'claude',
      cli_args: [],
      auth_config: {},
      secret_bindings: [],
      model_name: 'claude-opus-4-6',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
    {
      id: GEMINI_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'Gemini CLI',
      adapter_type: 'gemini-cli',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'gemini',
      cli_args: [],
      auth_config: {},
      secret_bindings: [],
      model_name: 'gemini-2.5-pro',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
    {
      id: OPENAI_PROVIDER_ID,
      organization_id: ORG_ID,
      org_id: ORG_ID,
      machine_id: LOCAL_MACHINE_ID,
      machine_name: 'local',
      machine_host: 'local',
      machine_status: 'online',
      machine_workspace_root: '~/.openase/workspace',
      name: 'OpenAI Codex',
      adapter_type: 'codex-app-server',
      availability_state: 'available',
      available: true,
      availability_checked_at: '2026-04-01T02:54:26Z',
      availability_reason: null,
      capabilities: {
        ephemeral_chat: {
          state: 'available',
          reason: null,
        },
      },
      cli_command: 'codex',
      cli_args: ['app-server', '--listen', 'stdio://'],
      auth_config: {},
      secret_bindings: [],
      model_name: 'gpt-5.4',
      model_temperature: 0,
      model_max_tokens: 16384,
      max_parallel_runs: 0,
      cost_per_input_token: 0,
      cost_per_output_token: 0,
      pricing_config: {},
    },
  ]

  const agents = [
    {
      id: DEFAULT_AGENT_ID,
      project_id: PROJECT_ID,
      provider_id: DEFAULT_PROVIDER_ID,
      name: 'coding-main',
      runtime_control_state: 'active',
      total_tickets_completed: 12,
      runtime: {
        status: 'ready',
        runtime_phase: 'ready',
        active_run_count: 1,
        current_run_id: 'run-1',
        current_ticket_id: DEFAULT_TICKET_ID,
        last_heartbeat_at: nowIso,
        runtime_started_at: '2026-03-27T09:00:00.000Z',
        session_id: 'sess-1',
        last_error: '',
      },
    },
  ]

  const agentRuns = [
    {
      id: 'run-1',
      project_id: PROJECT_ID,
      agent_id: DEFAULT_AGENT_ID,
      provider_id: DEFAULT_PROVIDER_ID,
      workflow_id: DEFAULT_WORKFLOW_ID,
      ticket_id: DEFAULT_TICKET_ID,
      status: 'executing',
      last_heartbeat_at: nowIso,
      runtime_started_at: '2026-03-27T09:00:00.000Z',
      session_id: 'sess-1',
      last_error: '',
      created_at: '2026-03-27T09:00:00.000Z',
    },
  ]

  const activityEvents = [
    {
      id: 'activity-1',
      project_id: PROJECT_ID,
      ticket_id: DEFAULT_TICKET_ID,
      agent_id: DEFAULT_AGENT_ID,
      event_type: 'agent.executing',
      message: 'coding-main started work.',
      metadata: {
        agent_name: 'coding-main',
      },
      created_at: nowIso,
    },
  ]

  const projectUpdates: Record<string, unknown>[] = []

  const tickets = [
    createMockTicketRecord({
      id: DEFAULT_TICKET_ID,
      identifier: 'ASE-101',
      title: 'Improve machine management UX',
      statusId: DEFAULT_STATUS_IDS.todo,
      statusName: 'Todo',
      workflowId: DEFAULT_WORKFLOW_ID,
    }),
  ]

  const statuses = [
    {
      id: DEFAULT_STATUS_IDS.todo,
      project_id: PROJECT_ID,
      name: 'Todo',
      stage: 'unstarted',
      color: '#2563eb',
      icon: '',
      position: 1,
      active_runs: 1,
      max_active_runs: null,
    },
    {
      id: DEFAULT_STATUS_IDS.review,
      project_id: PROJECT_ID,
      name: 'In Review',
      stage: 'started',
      color: '#f59e0b',
      icon: '',
      position: 2,
      active_runs: 0,
      max_active_runs: null,
    },
    {
      id: DEFAULT_STATUS_IDS.done,
      project_id: PROJECT_ID,
      name: 'Done',
      stage: 'completed',
      color: '#16a34a',
      icon: '',
      position: 3,
      active_runs: 0,
      max_active_runs: null,
    },
  ]

  const repos = [
    {
      id: DEFAULT_REPO_ID,
      project_id: PROJECT_ID,
      name: 'TodoApp',
      repository_url: 'https://github.com/BetterAndBetterII/TodoApp.git',
      default_branch: 'main',
      workspace_dirname: 'TodoApp',
      labels: [],
    },
  ]

  const workflows = [
    {
      id: DEFAULT_WORKFLOW_ID,
      project_id: PROJECT_ID,
      name: 'Coding Workflow',
      type: 'coding',
      agent_id: DEFAULT_AGENT_ID,
      pickup_status_ids: [DEFAULT_STATUS_IDS.todo],
      finish_status_ids: [DEFAULT_STATUS_IDS.review],
      max_concurrent: 0,
      max_retry_attempts: 1,
      timeout_minutes: 30,
      stall_timeout_minutes: 5,
      is_active: true,
      harness_path: '.openase/harnesses/coding-workflow.md',
      version: 3,
    },
  ]

  const harnessByWorkflowId = {
    [DEFAULT_WORKFLOW_ID]: {
      content: defaultHarnessContent('Coding Workflow'),
      path: '.openase/harnesses/coding-workflow.md',
      version: 3,
      history: [
        {
          id: `${DEFAULT_WORKFLOW_ID}-v3`,
          version: 3,
          created_by: 'user:manual',
          created_at: nowIso,
        },
        {
          id: `${DEFAULT_WORKFLOW_ID}-v2`,
          version: 2,
          created_by: 'user:manual',
          created_at: '2026-03-26T10:00:00.000Z',
        },
      ],
    },
  }

  const scheduledJobs = [
    {
      id: DEFAULT_JOB_ID,
      project_id: PROJECT_ID,
      workflow_id: DEFAULT_WORKFLOW_ID,
      name: 'Nightly regression sweep',
      cron_expression: '0 2 * * *',
      is_enabled: true,
      ticket_template: {
        title: 'Run nightly regression sweep',
        description: 'Verify the core project workflows still behave as expected.',
        priority: 'medium',
        type: 'feature',
        budget_usd: 12,
        created_by: 'scheduler',
      },
      last_run_at: '2026-03-26T02:00:00.000Z',
      next_run_at: '2026-03-28T02:00:00.000Z',
    },
  ]

  const skills = [
    {
      id: 'skill-commit',
      project_id: PROJECT_ID,
      name: 'commit',
      description: 'Create a well-formed git commit.',
      path: '/skills/commit',
      current_version: 2,
      is_builtin: true,
      is_enabled: true,
      created_by: 'system:init',
      created_at: nowIso,
      bound_workflows: [{ id: DEFAULT_WORKFLOW_ID }],
      content: '# Commit\n\nCreate a well-formed git commit.',
      history: [
        { id: 'skill-commit-v2', version: 2, created_by: 'user:manual', created_at: nowIso },
        {
          id: 'skill-commit-v1',
          version: 1,
          created_by: 'system:init',
          created_at: '2026-03-26T10:00:00.000Z',
        },
      ],
    },
    {
      id: 'skill-deploy-openase',
      project_id: PROJECT_ID,
      name: 'deploy-openase',
      description: 'Build and redeploy OpenASE locally.',
      path: '/skills/deploy-openase',
      current_version: 1,
      is_builtin: false,
      is_enabled: true,
      created_by: 'user:manual',
      created_at: nowIso,
      bound_workflows: [],
      content: '# Deploy OpenASE\n\nBuild and redeploy OpenASE locally.',
      history: [
        {
          id: 'skill-deploy-openase-v1',
          version: 1,
          created_by: 'user:manual',
          created_at: nowIso,
        },
      ],
    },
  ]

  const builtinRoles = [
    {
      id: 'builtin-coding',
      workflow_type: 'coding',
      name: 'coding',
      content: defaultHarnessContent('Coding Workflow'),
    },
  ]

  const harnessVariables = {
    groups: [
      {
        name: 'ticket',
        variables: [
          { name: 'ticket.identifier', description: 'Ticket identifier' },
          { name: 'ticket.title', description: 'Ticket title' },
        ],
      },
      {
        name: 'project',
        variables: [{ name: 'project.name', description: 'Project name' }],
      },
    ],
  }

  return {
    organizations,
    projects,
    machines,
    providers,
    agents,
    agentRuns,
    activityEvents,
    projectUpdates,
    tickets,
    statuses,
    repos,
    workflows,
    harnessByWorkflowId,
    scheduledJobs,
    projectConversations: [],
    projectConversationEntries: [],
    skills,
    builtinRoles,
    securitySettingsByProjectId: {
      [PROJECT_ID]: createDefaultSecuritySettings(PROJECT_ID),
    },
    harnessVariables,
    counters: {
      machine: 2,
      repo: 1,
      workflow: 1,
      agent: 1,
      skill: 2,
      scheduledJob: 1,
      projectUpdateThread: 0,
      projectUpdateComment: 0,
      projectConversation: 0,
      projectConversationEntry: 0,
      projectConversationTurn: 0,
    },
  }
}

export function seedBoardState(state: MockState, countsByStatusID: Record<string, number>) {
  const statusNameByID = new Map(
    state.statuses.map((status) => [asString(status.id) ?? '', asString(status.name) ?? 'Todo']),
  )
  const seededTickets: Record<string, unknown>[] = []
  let sequence = 0

  for (const status of state.statuses) {
    const statusId = asString(status.id)
    if (!statusId) {
      continue
    }
    const count = Math.max(0, asNumber(countsByStatusID[statusId]) ?? 0)
    for (let index = 0; index < count; index += 1) {
      sequence += 1
      seededTickets.push(
        createMockTicketRecord({
          id: `ticket-seeded-${sequence}`,
          identifier: `ASE-${100 + sequence}`,
          title: `Seeded ticket ${sequence}`,
          statusId,
          statusName: statusNameByID.get(statusId) ?? 'Todo',
          workflowId: DEFAULT_WORKFLOW_ID,
          createdAt: new Date(Date.parse(nowIso) + sequence * 60_000).toISOString(),
        }),
      )
    }
  }

  state.tickets = seededTickets
  state.activityEvents = []
}
