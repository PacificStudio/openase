import type { MockState } from './constants'
import { DEFAULT_WORKFLOW_ID, ORG_ID, PROJECT_ID, nowIso } from './constants'
import { asBoolean, asNumber, asObject, asString, asStringArray } from './helpers'

export function createMachineRecord(state: MockState, body: Record<string, unknown>) {
  const id = `machine-${++state.counters.machine}`
  const host = asString(body.host) ?? '127.0.0.1'
  const reachabilityMode = resolveMachineReachabilityMode(body, host)
  const executionMode = resolveMachineExecutionMode(body, host)
  const connectionMode = resolveMachineConnectionMode(reachabilityMode, executionMode)
  const executionCapabilities = executionCapabilitiesForConnectionMode(connectionMode)
  const sshUser = asString(body.ssh_user) ?? ''
  const sshKeyPath = asString(body.ssh_key_path) ?? ''
  return {
    id,
    org_id: ORG_ID,
    name: asString(body.name) ?? `machine-${state.counters.machine}`,
    host,
    port: asNumber(body.port) ?? 22,
    reachability_mode: reachabilityMode,
    execution_mode: executionMode,
    execution_capabilities: executionCapabilities,
    ssh_helper_enabled: Boolean(sshUser || sshKeyPath),
    ssh_user: sshUser,
    ssh_key_path: sshKeyPath,
    advertised_endpoint: asString(body.advertised_endpoint),
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
    description: asString(body.description) ?? '',
    labels: asStringArray(body.labels),
    status: asString(body.status) ?? 'maintenance',
    workspace_root: asString(body.workspace_root) ?? '',
    agent_cli_path: asString(body.agent_cli_path) ?? '',
    env_vars: asStringArray(body.env_vars),
    last_heartbeat_at: null,
    resources: {},
  }
}

export function createRepoRecord(state: MockState, body: Record<string, unknown>) {
  return {
    id: `repo-${++state.counters.repo}`,
    project_id: PROJECT_ID,
    name: asString(body.name) ?? `repo-${state.counters.repo}`,
    repository_url: asString(body.repository_url) ?? '',
    default_branch: asString(body.default_branch) ?? 'main',
    workspace_dirname:
      asString(body.workspace_dirname) ?? asString(body.name) ?? `repo-${state.counters.repo}`,
    labels: asStringArray(body.labels),
  }
}

export function createScheduledJobRecord(state: MockState, body: Record<string, unknown>) {
  const ticketTemplate = asObject(body.ticket_template) ?? {}
  const repoScopes = Array.isArray(ticketTemplate.repo_scopes)
    ? ticketTemplate.repo_scopes
        .map((scope) => asObject(scope))
        .filter((scope): scope is Record<string, unknown> => Boolean(scope))
        .map((scope) => ({
          repo_id: asString(scope.repo_id) ?? '',
          branch_name: asString(scope.branch_name) ?? null,
        }))
        .filter((scope) => scope.repo_id)
    : []
  return {
    id: `job-${++state.counters.scheduledJob}`,
    project_id: PROJECT_ID,
    workflow_id: asString(body.workflow_id) ?? DEFAULT_WORKFLOW_ID,
    name: asString(body.name) ?? `Job ${state.counters.scheduledJob}`,
    cron_expression: asString(body.cron_expression) ?? '0 0 * * *',
    is_enabled: asBoolean(body.is_enabled) ?? true,
    ticket_template: {
      title: asString(ticketTemplate.title) ?? null,
      description: asString(ticketTemplate.description) ?? null,
      priority: asString(ticketTemplate.priority) ?? 'medium',
      status: asString(ticketTemplate.status) ?? 'Todo',
      type: asString(ticketTemplate.type) ?? 'feature',
      budget_usd: asNumber(ticketTemplate.budget_usd) ?? 0,
      created_by: asString(ticketTemplate.created_by) ?? null,
      repo_scopes: repoScopes,
    },
    last_run_at: null,
    next_run_at: '2026-03-28T02:00:00.000Z',
  }
}

export function applyMachineMutation(
  machine: Record<string, unknown>,
  body: Record<string, unknown>,
) {
  machine.name = asString(body.name) ?? machine.name
  machine.host = asString(body.host) ?? machine.host
  machine.port = asNumber(body.port) ?? machine.port
  machine.reachability_mode = resolveMachineReachabilityMode(body, asString(machine.host) ?? '')
  machine.execution_mode = resolveMachineExecutionMode(body, asString(machine.host) ?? '')
  machine.execution_capabilities = executionCapabilitiesForConnectionMode(
    resolveMachineConnectionMode(
      asString(machine.reachability_mode) ?? 'direct_connect',
      asString(machine.execution_mode) ?? 'websocket',
    ),
  )
  machine.ssh_user = asString(body.ssh_user) ?? machine.ssh_user
  machine.ssh_key_path = asString(body.ssh_key_path) ?? machine.ssh_key_path
  machine.ssh_helper_enabled = Boolean(
    (asString(machine.ssh_user) ?? '').trim() || (asString(machine.ssh_key_path) ?? '').trim(),
  )
  machine.advertised_endpoint = asString(body.advertised_endpoint) ?? machine.advertised_endpoint
  machine.description = asString(body.description) ?? machine.description
  machine.labels = asStringArray(body.labels)
  machine.status = asString(body.status) ?? machine.status
  machine.workspace_root = asString(body.workspace_root) ?? machine.workspace_root
  machine.agent_cli_path = asString(body.agent_cli_path) ?? machine.agent_cli_path
  machine.env_vars = asStringArray(body.env_vars)
}

function resolveMachineReachabilityMode(body: Record<string, unknown>, host: string) {
  const raw = asString(body.reachability_mode)
  if (raw === 'local' || raw === 'direct_connect' || raw === 'reverse_connect') {
    return raw
  }
  return host.trim().toLowerCase() === 'local' ? 'local' : 'direct_connect'
}

function resolveMachineExecutionMode(body: Record<string, unknown>, host: string) {
  const raw = asString(body.execution_mode)
  if (raw === 'local_process' || raw === 'websocket') {
    return raw
  }
  return host.trim().toLowerCase() === 'local' ? 'local_process' : 'websocket'
}

function resolveMachineConnectionMode(reachabilityMode: string, _executionMode: string) {
  if (reachabilityMode === 'local') {
    return 'local'
  }
  if (reachabilityMode === 'reverse_connect') {
    return 'ws_reverse'
  }
  return 'ws_listener'
}

function executionCapabilitiesForConnectionMode(connectionMode: string) {
  return ['probe', 'workspace_prepare', 'artifact_sync', 'process_streaming']
}

export function applyRepoMutation(repo: Record<string, unknown>, body: Record<string, unknown>) {
  repo.name = asString(body.name) ?? repo.name
  repo.repository_url = asString(body.repository_url) ?? repo.repository_url
  repo.default_branch = asString(body.default_branch) ?? repo.default_branch
  repo.workspace_dirname = asString(body.workspace_dirname) ?? repo.workspace_dirname
  repo.labels = asStringArray(body.labels)
}

export function applyScheduledJobMutation(
  job: Record<string, unknown>,
  body: Record<string, unknown>,
) {
  job.name = asString(body.name) ?? job.name
  job.cron_expression = asString(body.cron_expression) ?? job.cron_expression
  job.workflow_id = asString(body.workflow_id) ?? job.workflow_id
  if (typeof body.is_enabled === 'boolean') {
    job.is_enabled = body.is_enabled
  }
  const ticketTemplate = {
    ...(asObject(job.ticket_template) ?? {}),
    ...(asObject(body.ticket_template) ?? {}),
  }
  job.ticket_template = {
    title: asString(ticketTemplate.title) ?? null,
    description: asString(ticketTemplate.description) ?? null,
    priority: asString(ticketTemplate.priority) ?? 'medium',
    status: asString(ticketTemplate.status) ?? 'Todo',
    type: asString(ticketTemplate.type) ?? 'feature',
    budget_usd: asNumber(ticketTemplate.budget_usd) ?? 0,
    created_by: asString(ticketTemplate.created_by) ?? null,
    repo_scopes: Array.isArray(ticketTemplate.repo_scopes)
      ? ticketTemplate.repo_scopes
          .map((scope) => asObject(scope))
          .filter((scope): scope is Record<string, unknown> => Boolean(scope))
          .map((scope) => ({
            repo_id: asString(scope.repo_id) ?? '',
            branch_name: asString(scope.branch_name) ?? null,
          }))
          .filter((scope) => scope.repo_id)
      : [],
  }
}

export function machineResourceSnapshot(
  transport: string,
  values: {
    cpuUsagePercent: number
    memoryTotalGB: number
    memoryUsedGB: number
    memoryAvailableGB: number
    diskTotalGB: number
    diskAvailableGB: number
    gpuDispatchable: boolean
    gpus: Record<string, unknown>[]
  },
) {
  return {
    transport,
    checked_at: nowIso,
    last_success: true,
    cpu_cores: 16,
    cpu_usage_percent: values.cpuUsagePercent,
    memory_total_gb: values.memoryTotalGB,
    memory_used_gb: values.memoryUsedGB,
    memory_available_gb: values.memoryAvailableGB,
    disk_total_gb: values.diskTotalGB,
    disk_available_gb: values.diskAvailableGB,
    gpu_dispatchable: values.gpuDispatchable,
    gpu: values.gpus,
    monitor: {
      l1: {
        checked_at: nowIso,
        reachable: true,
        latency_ms: transport === 'local' ? 1 : 18,
        transport,
      },
      l2: {
        checked_at: nowIso,
        available: true,
        disk_low: false,
        memory_low: false,
      },
      l3: {
        checked_at: nowIso,
        gpu_dispatchable: values.gpuDispatchable,
      },
    },
  }
}

export function defaultHarnessContent(name: string) {
  return `---\nname: ${name}\n---\n\n# ${name}\n\n- Inspect the ticket context.\n- Make the requested change.\n- Summarize the result.`
}

export function dedupeById(items: Record<string, unknown>[]) {
  const seen = new Set<string>()
  return items.filter((item) => {
    const id = asString(item.id)
    if (!id || seen.has(id)) {
      return false
    }
    seen.add(id)
    return true
  })
}

export function slugify(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}
