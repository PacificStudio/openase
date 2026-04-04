import type { Machine } from '$lib/api/contracts'
import type { MachineDraft, MachineDraftParseResult, MachineStatus } from './types'
import { getWorkspaceRootRecommendation, normalizeConnectionMode } from './machine-guidance'

export { parseMachineSnapshot } from './snapshot'
export * from './machine-guidance'

export function createEmptyMachineDraft(): MachineDraft {
  const draft: MachineDraft = {
    name: '',
    host: '',
    port: '22',
    connectionMode: 'ssh',
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
    connectionMode: normalizeConnectionMode(machine.connection_mode, machine.host),
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

  if (field !== 'connectionMode') {
    return maybeRefreshRecommendedWorkspaceRoot(
      draft,
      nextDraft,
      field,
      machine,
      previousRecommendation,
    )
  }

  const nextMode = normalizeConnectionMode(value, nextDraft.host)
  nextDraft.connectionMode = nextMode
  applyModeDefaults(draft, nextDraft, nextMode)
  if (nextMode !== 'ws_listener') {
    nextDraft.advertisedEndpoint = ''
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
  const connectionMode = normalizeConnectionMode(draft.connectionMode, draft.host)
  const name = connectionMode === 'local' ? 'local' : draft.name.trim()
  const host = connectionMode === 'local' ? 'local' : draft.host.trim()
  const port = parseMachinePort(draft.port)
  if (port === null) {
    return { ok: false, error: 'Port must be an integer between 1 and 65535.' }
  }

  const identityError = validateMachineIdentity(name, host)
  if (identityError) {
    return { ok: false, error: identityError }
  }

  const transportError = validateTransportFields(draft, connectionMode)
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
      connection_mode: connectionMode,
      ssh_user: connectionMode === 'ssh' ? sshUser : '',
      ssh_key_path: connectionMode === 'ssh' ? sshKeyPath : '',
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
    normalizeConnectionMode(
      draft?.connectionMode ?? machine?.connection_mode,
      draft?.host ?? machine?.host,
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
  if ((field === 'connectionMode' || field === 'sshUser') && followsRecommendation) {
    nextDraft.workspaceRoot = getWorkspaceRootRecommendation({ draft: nextDraft, machine }).value
  }
  return nextDraft
}

function applyModeDefaults(
  draft: MachineDraft,
  nextDraft: MachineDraft,
  nextMode: MachineDraft['connectionMode'],
) {
  if (nextMode === 'local') {
    if (!nextDraft.name.trim() || nextDraft.name.trim().toLowerCase() === 'local') {
      nextDraft.name = 'local'
    }
    nextDraft.host = 'local'
    nextDraft.port = '22'
    nextDraft.sshUser = ''
    nextDraft.sshKeyPath = ''
    nextDraft.advertisedEndpoint = ''
    return
  }

  if (draft.connectionMode === 'local') {
    if (nextDraft.name.trim().toLowerCase() === 'local') nextDraft.name = ''
    if (nextDraft.host.trim().toLowerCase() === 'local') nextDraft.host = ''
  }
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
  connectionMode: MachineDraft['connectionMode'],
): string | null {
  if (connectionMode === 'ssh') {
    if (!draft.sshUser.trim()) return 'SSH user is required for SSH machines.'
    if (!draft.sshKeyPath.trim()) return 'SSH key path is required for SSH machines.'
  }
  if (connectionMode === 'ws_listener' && !draft.advertisedEndpoint.trim()) {
    return 'Advertised endpoint is required for websocket listener machines.'
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
