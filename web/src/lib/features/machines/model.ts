import type { Machine } from '$lib/api/contracts'
import type { MachineDraft, MachineDraftParseResult, MachineStatus } from './types'
export { parseMachineSnapshot } from './snapshot'

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
