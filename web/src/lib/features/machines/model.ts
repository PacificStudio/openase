import type { Machine } from '$lib/api/contracts'
import type {
  MachineDraft,
  MachineDraftParseResult,
  MachineGPUView,
  MachineMonitorLevel,
  MachineSnapshot,
  MachineStatus,
  ResourceMap,
} from './types'

export const machineStatusOptions: MachineStatus[] = [
  'online',
  'degraded',
  'offline',
  'maintenance',
]

export function createEmptyMachineDraft(): MachineDraft {
  return {
    name: '',
    host: '',
    port: '22',
    sshUser: '',
    sshKeyPath: '',
    description: '',
    labels: '',
    status: 'maintenance',
    workspaceRoot: '',
    agentCLIPath: '',
    envVars: '',
  }
}

export function machineToDraft(machine: Machine): MachineDraft {
  return {
    name: machine.name,
    host: machine.host,
    port: String(machine.port || 22),
    sshUser: machine.ssh_user ?? '',
    sshKeyPath: machine.ssh_key_path ?? '',
    description: machine.description,
    labels: (machine.labels ?? []).join(', '),
    status: normalizeMachineStatus(machine.status),
    workspaceRoot: machine.workspace_root ?? '',
    agentCLIPath: machine.agent_cli_path ?? '',
    envVars: (machine.env_vars ?? []).join('\n'),
  }
}

export function parseMachineDraft(draft: MachineDraft): MachineDraftParseResult {
  const name = draft.name.trim()
  const host = draft.host.trim()
  if (!name) {
    return { ok: false, error: 'Machine name is required.' }
  }
  if (!host) {
    return { ok: false, error: 'Host is required.' }
  }

  const portText = draft.port.trim()
  const port = portText ? Number(portText) : 22
  if (!Number.isInteger(port) || port <= 0 || port > 65535) {
    return { ok: false, error: 'Port must be an integer between 1 and 65535.' }
  }

  const normalizedHost = host.toLowerCase()
  const normalizedName = name.toLowerCase()
  if (normalizedHost === 'local' && normalizedName !== 'local') {
    return { ok: false, error: 'The local machine must be named "local".' }
  }
  if (normalizedName === 'local' && normalizedHost !== 'local') {
    return { ok: false, error: 'The machine named "local" must use host "local".' }
  }

  const sshUser = draft.sshUser.trim()
  const sshKeyPath = draft.sshKeyPath.trim()
  if (normalizedHost !== 'local') {
    if (!sshUser) {
      return { ok: false, error: 'SSH user is required for remote machines.' }
    }
    if (!sshKeyPath) {
      return { ok: false, error: 'SSH key path is required for remote machines.' }
    }
  }

  return {
    ok: true,
    value: {
      name,
      host,
      port,
      ssh_user: normalizedHost === 'local' ? '' : sshUser,
      ssh_key_path: normalizedHost === 'local' ? '' : sshKeyPath,
      description: draft.description.trim(),
      labels: splitLabels(draft.labels),
      status: normalizeMachineStatus(draft.status),
      workspace_root: draft.workspaceRoot.trim(),
      agent_cli_path: draft.agentCLIPath.trim(),
      env_vars: splitLines(draft.envVars),
    },
  }
}

export function parseMachineSnapshot(raw: ResourceMap | null | undefined): MachineSnapshot | null {
  if (!raw || Object.keys(raw).length === 0) {
    return null
  }

  const monitor = asObject(raw.monitor)
  const l1 = parseMonitorLevel(asObject(monitor?.l1))
  const l2 = parseMonitorLevel(asObject(monitor?.l2))
  const l3 = parseMonitorLevel(asObject(monitor?.l3))
  const monitorErrors = [l1?.error, l2?.error, l3?.error].filter((value): value is string =>
    Boolean(value),
  )

  return {
    transport: asString(raw.transport),
    checkedAt: asString(raw.checked_at) ?? asString(raw.collected_at),
    lastSuccess: asBoolean(raw.last_success),
    cpuCores: asNumber(raw.cpu_cores),
    cpuUsagePercent: asNumber(raw.cpu_usage_percent),
    memoryTotalGB: asNumber(raw.memory_total_gb),
    memoryUsedGB: asNumber(raw.memory_used_gb),
    memoryAvailableGB: asNumber(raw.memory_available_gb),
    diskTotalGB: asNumber(raw.disk_total_gb),
    diskAvailableGB: asNumber(raw.disk_available_gb),
    gpuDispatchable: asBoolean(raw.gpu_dispatchable),
    gpus: parseGPUViews(raw.gpu),
    monitor: { l1, l2, l3 },
    monitorErrors,
  }
}

export function normalizeMachineStatus(status: string): MachineStatus {
  if (
    status === 'online' ||
    status === 'offline' ||
    status === 'degraded' ||
    status === 'maintenance'
  ) {
    return status
  }

  return 'maintenance'
}

export function filterMachines(machines: Machine[], searchQuery: string): Machine[] {
  const query = searchQuery.trim().toLowerCase()
  if (!query) {
    return machines
  }

  return machines.filter((machine) =>
    [
      machine.name,
      machine.host,
      machine.status,
      (machine.labels ?? []).join(' '),
      machine.description,
    ]
      .join(' ')
      .toLowerCase()
      .includes(query),
  )
}

export function isLocalMachine(machine: Machine | null | undefined, draft?: MachineDraft): boolean {
  const host = draft?.host || machine?.host || ''
  const name = draft?.name || machine?.name || ''
  return host.trim().toLowerCase() === 'local' || name.trim().toLowerCase() === 'local'
}

function splitLabels(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((value) => value.trim())
    .filter(Boolean)
}

function splitLines(raw: string): string[] {
  return raw
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean)
}

function parseMonitorLevel(raw: ResourceMap | null | undefined): MachineMonitorLevel | undefined {
  if (!raw) {
    return undefined
  }

  return {
    checkedAt: asString(raw.checked_at),
    error: asString(raw.error),
    transport: asString(raw.transport),
    reachable: asBoolean(raw.reachable),
    latencyMs: asNumber(raw.latency_ms),
    failureCause: asString(raw.failure_cause),
    consecutiveFailures: asNumber(raw.consecutive_failures),
    memoryLow: asBoolean(raw.memory_low),
    diskLow: asBoolean(raw.disk_low),
    available: asBoolean(raw.available),
    gpuDispatchable: asBoolean(raw.gpu_dispatchable),
  }
}

function parseGPUViews(raw: unknown): MachineGPUView[] {
  if (!Array.isArray(raw)) {
    return []
  }

  return raw
    .map((item) => {
      const gpu = asObject(item)
      if (!gpu) {
        return null
      }

      const index = asNumber(gpu.index)
      const name = asString(gpu.name)
      const memoryTotalGB = asNumber(gpu.memory_total_gb)
      const memoryUsedGB = asNumber(gpu.memory_used_gb)
      const utilizationPercent = asNumber(gpu.utilization_percent)
      if (
        index === undefined ||
        !name ||
        memoryTotalGB === undefined ||
        memoryUsedGB === undefined ||
        utilizationPercent === undefined
      ) {
        return null
      }

      return {
        index,
        name,
        memoryTotalGB,
        memoryUsedGB,
        utilizationPercent,
      }
    })
    .filter((item): item is MachineGPUView => item !== null)
    .sort((left, right) => left.index - right.index)
}

function asObject(value: unknown): ResourceMap | null {
  return value && typeof value === 'object' && !Array.isArray(value) ? (value as ResourceMap) : null
}

function asString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value : undefined
}

function asNumber(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function asBoolean(value: unknown): boolean | undefined {
  return typeof value === 'boolean' ? value : undefined
}
