import type { Machine } from '$lib/api/contracts'
import type {
  MachineConnectionMode,
  MachineDraft,
  MachineDraftParseResult,
  MachineExecutionMode,
  MachineReachabilityMode,
  MachineStatus,
} from './types'
import {
  getWorkspaceRootRecommendation,
  normalizeConnectionMode,
  normalizeExecutionMode,
  normalizeReachabilityMode,
} from './machine-guidance'

export { parseMachineSnapshot } from './snapshot'
export * from './machine-guidance'

export function createEmptyMachineDraft(): MachineDraft {
  const draft: MachineDraft = {
    name: '',
    host: '',
    port: '22',
    reachabilityMode: 'direct_connect',
    executionMode: 'websocket',
    sshUser: '',
    sshKeyPath: '',
    advertisedEndpoint: '',
    description: '',
    labels: '',
    status: 'maintenance',
    workspaceRoot: '',
    agentCLIPath: '',
    envVars: '',
  }

  return { ...draft, workspaceRoot: getWorkspaceRootRecommendation({ draft, machine: null }).value }
}

export function machineToDraft(machine: Machine): MachineDraft {
  return {
    name: machine.name,
    host: machine.host,
    port: String(machine.port || 22),
    reachabilityMode: normalizeReachabilityMode(
      machine.reachability_mode,
      machine.host,
      machine.connection_mode,
    ),
    executionMode: normalizeExecutionMode(
      machine.execution_mode,
      machine.host,
      machine.connection_mode,
    ),
    sshUser: machine.ssh_user ?? '',
    sshKeyPath: machine.ssh_key_path ?? '',
    advertisedEndpoint: machine.advertised_endpoint ?? '',
    description: machine.description,
    labels: (machine.labels ?? []).join(', '),
    status: normalizeMachineStatus(machine.status),
    workspaceRoot: machine.workspace_root ?? '',
    agentCLIPath: machine.agent_cli_path ?? '',
    envVars: (machine.env_vars ?? []).join('\n'),
  }
}

export function updateMachineDraft(
  draft: MachineDraft,
  field: keyof MachineDraft,
  value: string,
  machine: Machine | null,
): MachineDraft {
  const previousRecommendation = getWorkspaceRootRecommendation({ draft, machine }).value
  const nextDraft: MachineDraft = { ...draft, [field]: value }

  if (field === 'reachabilityMode' || field === 'executionMode') {
    const reachabilityMode = normalizeReachabilityMode(nextDraft.reachabilityMode, nextDraft.host)
    const executionMode = normalizeExecutionMode(
      nextDraft.executionMode,
      nextDraft.host,
      resolveLegacyConnectionMode(
        nextDraft.reachabilityMode,
        nextDraft.executionMode,
        nextDraft.host,
      ),
    )

    nextDraft.reachabilityMode = reachabilityMode
    nextDraft.executionMode = coerceExecutionMode(reachabilityMode, executionMode)
    applyModeDefaults(draft, nextDraft)
  }

  if (field !== 'reachabilityMode' && field !== 'executionMode') {
    return maybeRefreshRecommendedWorkspaceRoot(
      draft,
      nextDraft,
      field,
      machine,
      previousRecommendation,
    )
  }

  return maybeRefreshRecommendedWorkspaceRoot(
    draft,
    nextDraft,
    field,
    machine,
    previousRecommendation,
  )
}

export function parseMachineDraft(draft: MachineDraft): MachineDraftParseResult {
  const reachabilityMode = normalizeReachabilityMode(draft.reachabilityMode, draft.host)
  const executionMode = coerceExecutionMode(
    reachabilityMode,
    normalizeExecutionMode(draft.executionMode, draft.host),
  )
  const connectionMode = resolveConnectionMode(reachabilityMode, executionMode)
  const name = reachabilityMode === 'local' ? 'local' : draft.name.trim()
  const host = reachabilityMode === 'local' ? 'local' : draft.host.trim()
  const port = parseMachinePort(draft.port)
  if (port === null) {
    return { ok: false, error: 'Port must be an integer between 1 and 65535.' }
  }

  const identityError = validateMachineIdentity(name, host)
  if (identityError) {
    return { ok: false, error: identityError }
  }

  const transportError = validateTransportFields(draft, reachabilityMode, executionMode)
  if (transportError) {
    return { ok: false, error: transportError }
  }

  const sshUser = draft.sshUser.trim()
  const sshKeyPath = draft.sshKeyPath.trim()
  const advertisedEndpoint = draft.advertisedEndpoint.trim()
  if (connectionMode === 'ws_listener') {
    try {
      const parsed = new URL(advertisedEndpoint)
      if (parsed.protocol !== 'ws:' && parsed.protocol !== 'wss:') {
        return { ok: false, error: 'Advertised endpoint must use ws:// or wss://.' }
      }
      if (!parsed.host.trim()) {
        return { ok: false, error: 'Advertised endpoint must include a host.' }
      }
    } catch {
      return { ok: false, error: 'Advertised endpoint must be a valid websocket URL.' }
    }
  }

  return {
    ok: true,
    value: {
      name,
      host,
      port,
      reachability_mode: reachabilityMode,
      execution_mode: executionMode,
      connection_mode: connectionMode,
      ssh_user: sshUser,
      ssh_key_path: sshKeyPath,
      advertised_endpoint: connectionMode === 'ws_listener' ? advertisedEndpoint : '',
      description: draft.description.trim(),
      labels: splitLabels(draft.labels),
      status: normalizeMachineStatus(draft.status),
      workspace_root: draft.workspaceRoot.trim(),
      agent_cli_path: draft.agentCLIPath.trim(),
      env_vars: splitLines(draft.envVars),
    },
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

export function machineStatusLabel(status: string): string {
  return normalizeMachineStatus(status)
}

export function machineStatusDescription(status: string): string {
  switch (normalizeMachineStatus(status)) {
    case 'online':
      return 'Healthy and currently eligible for orchestration.'
    case 'degraded':
      return 'Reachable, but monitoring has detected issues that need attention.'
    case 'offline':
      return 'Currently unreachable or unable to report a healthy heartbeat.'
    case 'maintenance':
    default:
      return 'Held out of scheduling while configuration or maintenance work is in progress.'
  }
}

export function machineStatusBadgeClass(status: string): string {
  switch (normalizeMachineStatus(status)) {
    case 'online':
      return 'border-emerald-500/30 bg-emerald-500/12 text-emerald-700'
    case 'degraded':
      return 'border-amber-500/30 bg-amber-500/14 text-amber-700'
    case 'offline':
      return 'border-rose-500/30 bg-rose-500/12 text-rose-700'
    case 'maintenance':
    default:
      return 'border-slate-500/20 bg-slate-500/10 text-slate-700'
  }
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
      machine.reachability_mode,
      machine.execution_mode,
      machine.connection_mode,
      machine.advertised_endpoint,
      machine.detected_os,
      machine.detected_arch,
      (machine.labels ?? []).join(' '),
      machine.description,
    ]
      .join(' ')
      .toLowerCase()
      .includes(query),
  )
}

export function isLocalMachine(machine: Machine | null | undefined, draft?: MachineDraft): boolean {
  return (
    normalizeReachabilityMode(
      draft?.reachabilityMode ?? machine?.reachability_mode,
      draft?.host ?? machine?.host,
      machine?.connection_mode,
    ) === 'local'
  )
}

function maybeRefreshRecommendedWorkspaceRoot(
  draft: MachineDraft,
  nextDraft: MachineDraft,
  field: keyof MachineDraft,
  machine: Machine | null,
  previousRecommendation: string,
): MachineDraft {
  const followsRecommendation =
    draft.workspaceRoot.trim() === '' || draft.workspaceRoot.trim() === previousRecommendation
  if (
    (field === 'reachabilityMode' || field === 'executionMode' || field === 'sshUser') &&
    followsRecommendation
  ) {
    nextDraft.workspaceRoot = getWorkspaceRootRecommendation({ draft: nextDraft, machine }).value
  }
  return nextDraft
}

function applyModeDefaults(draft: MachineDraft, nextDraft: MachineDraft) {
  if (nextDraft.reachabilityMode === 'local') {
    if (!nextDraft.name.trim() || nextDraft.name.trim().toLowerCase() === 'local') {
      nextDraft.name = 'local'
    }
    nextDraft.host = 'local'
    nextDraft.port = '22'
    nextDraft.executionMode = 'local_process'
    nextDraft.sshUser = ''
    nextDraft.sshKeyPath = ''
    nextDraft.advertisedEndpoint = ''
    return
  }

  if (draft.reachabilityMode === 'local') {
    if (nextDraft.name.trim().toLowerCase() === 'local') nextDraft.name = ''
    if (nextDraft.host.trim().toLowerCase() === 'local') nextDraft.host = ''
  }

  if (nextDraft.reachabilityMode === 'reverse_connect') {
    nextDraft.executionMode = 'websocket'
    nextDraft.advertisedEndpoint = ''
  }
  if (nextDraft.reachabilityMode === 'direct_connect' && nextDraft.executionMode !== 'websocket') {
    nextDraft.advertisedEndpoint = ''
  }
}

function resolveConnectionMode(
  reachabilityMode: MachineReachabilityMode,
  executionMode: MachineExecutionMode,
): MachineConnectionMode {
  if (reachabilityMode === 'local') {
    return 'local'
  }
  if (reachabilityMode === 'reverse_connect') {
    return 'ws_reverse'
  }
  return executionMode === 'ssh_compat' ? 'ssh' : 'ws_listener'
}

function resolveLegacyConnectionMode(
  reachabilityMode: string,
  executionMode: string,
  host: string,
): MachineConnectionMode {
  return resolveConnectionMode(
    normalizeReachabilityMode(reachabilityMode, host),
    normalizeExecutionMode(executionMode, host),
  )
}

function coerceExecutionMode(
  reachabilityMode: MachineReachabilityMode,
  executionMode: MachineExecutionMode,
): MachineExecutionMode {
  if (reachabilityMode === 'local') {
    return 'local_process'
  }
  if (reachabilityMode === 'reverse_connect') {
    return 'websocket'
  }
  return executionMode === 'local_process' ? 'websocket' : executionMode
}

function parseMachinePort(rawPort: string): number | null {
  const portText = rawPort.trim()
  const port = portText ? Number(portText) : 22
  return Number.isInteger(port) && port > 0 && port <= 65535 ? port : null
}

function validateMachineIdentity(name: string, host: string): string | null {
  if (!name) return 'Machine name is required.'
  if (!host) return 'Host is required.'

  const normalizedHost = host.toLowerCase()
  const normalizedName = name.toLowerCase()
  if (normalizedHost === 'local' && normalizedName !== 'local') {
    return 'The local machine must be named "local".'
  }
  if (normalizedName === 'local' && normalizedHost !== 'local') {
    return 'The machine named "local" must use host "local".'
  }
  return null
}

function validateTransportFields(
  draft: MachineDraft,
  reachabilityMode: MachineReachabilityMode,
  executionMode: MachineExecutionMode,
): string | null {
  const sshUser = draft.sshUser.trim()
  const sshKeyPath = draft.sshKeyPath.trim()
  const hasAnySSHHelper = sshUser.length > 0 || sshKeyPath.length > 0

  if (executionMode === 'ssh_compat') {
    if (!sshUser) return 'SSH user is required for SSH compatibility.'
    if (!sshKeyPath) return 'SSH key path is required for SSH compatibility.'
  }
  if (hasAnySSHHelper && (!sshUser || !sshKeyPath)) {
    return 'SSH helper access requires both an SSH user and an SSH key path.'
  }
  if (
    reachabilityMode === 'direct_connect' &&
    executionMode === 'websocket' &&
    !draft.advertisedEndpoint.trim()
  ) {
    return 'Advertised endpoint is required for direct-connect websocket machines.'
  }
  return null
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
