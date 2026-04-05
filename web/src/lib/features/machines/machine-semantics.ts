import type {
  MachineConnectionMode,
  MachineDraft,
  MachineExecutionMode,
  MachineReachabilityMode,
} from './types'

export function resolveConnectionMode(
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

export function resolveLegacyConnectionMode(
  reachabilityMode: string,
  executionMode: string,
  host: string,
  normalizeReachabilityMode: (
    value: string | null | undefined,
    host: string | null | undefined,
    connectionMode?: string | null | undefined,
  ) => MachineReachabilityMode,
  normalizeExecutionMode: (
    value: string | null | undefined,
    host: string | null | undefined,
    connectionMode?: string | null | undefined,
  ) => MachineExecutionMode,
): MachineConnectionMode {
  return resolveConnectionMode(
    normalizeReachabilityMode(reachabilityMode, host),
    normalizeExecutionMode(executionMode, host),
  )
}

export function coerceExecutionMode(
  reachabilityMode: MachineReachabilityMode,
  executionMode: MachineExecutionMode,
  advertisedEndpoint = '',
): MachineExecutionMode {
  if (reachabilityMode === 'local') {
    return 'local_process'
  }
  if (reachabilityMode === 'reverse_connect') {
    return 'websocket'
  }
  if (executionMode === 'local_process') {
    return 'websocket'
  }
  if (executionMode === 'ssh_compat' && advertisedEndpoint.trim()) {
    return 'websocket'
  }
  return executionMode
}

export function parseMachinePort(rawPort: string): number | null {
  const portText = rawPort.trim()
  const port = portText ? Number(portText) : 22
  return Number.isInteger(port) && port > 0 && port <= 65535 ? port : null
}

export function validateMachineIdentity(name: string, host: string): string | null {
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

export function validateTransportFields(
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

export function splitLabels(raw: string): string[] {
  return raw
    .split(/[\n,]/)
    .map((value) => value.trim())
    .filter(Boolean)
}

export function splitLines(raw: string): string[] {
  return raw
    .split('\n')
    .map((value) => value.trim())
    .filter(Boolean)
}
