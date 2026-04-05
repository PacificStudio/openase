import type { Machine } from '$lib/api/contracts'
import type {
  MachineConnectionMode,
  MachineDraft,
  MachineExecutionGuide,
  MachineExecutionMode,
  MachineModeGuide,
  MachineReachabilityMode,
  WorkspaceRootRecommendation,
  WorkspaceRootState,
} from './types'
import { normalizeDetectedOS } from './machine-detection'

export * from './machine-detection'

const defaultRemoteWorkspaceRoot = '/srv/openase/workspace'

const machineModeGuides: Record<MachineReachabilityMode, MachineModeGuide> = {
  local: {
    mode: 'local',
    label: 'Local',
    summary: 'Run OpenASE on the same host as the control plane.',
    requiredFields: 'Reserved local identity and a local workspace root.',
    installMethod: 'Use the local OpenASE CLI and local ticket workspaces under ~/.openase.',
    testSemantics: 'Connection tests verify the local runtime user, hostname, and platform.',
    commonErrors: 'A relative workspace root or a renamed local machine will be rejected.',
  },
  direct_connect: {
    mode: 'direct_connect',
    label: 'Direct Connect',
    summary: 'The control plane can reach the machine directly.',
    requiredFields:
      'Machine identity, a remote workspace root, and websocket listener details. SSH is optional helper access.',
    installMethod:
      'Expose a websocket listener endpoint for runtime execution. Keep SSH only when you need bootstrap, diagnostics, or emergency repair access.',
    testSemantics:
      'Connection tests dial the advertised websocket listener. Separate SSH helper diagnostics cover helper-only access.',
    commonErrors:
      'Missing advertised endpoints, unreachable listener URLs, or stale SSH helper credentials.',
  },
  reverse_connect: {
    mode: 'reverse_connect',
    label: 'Machine Dials Out',
    summary: 'The machine daemon dials out and carries native websocket runtime execution.',
    requiredFields:
      'Machine identity, daemon registration, and a workspace root the daemon can access.',
    installMethod:
      'Install and register the daemon on the remote host, keep the CLI path aligned with that host, and keep SSH only for helper bootstrap or diagnostics.',
    testSemantics:
      'Connection tests rely on daemon session state and reported platform metadata while runtime execution stays on the reverse websocket session.',
    commonErrors:
      'No registered daemon session, stale credentials, or a workspace root that the daemon cannot access.',
  },
}

const machineExecutionGuides: Record<MachineExecutionMode, MachineExecutionGuide> = {
  local_process: {
    mode: 'local_process',
    label: 'Local Process',
    summary: 'Commands run directly on the control-plane host.',
  },
  websocket: {
    mode: 'websocket',
    label: 'WebSocket',
    summary: 'Remote runtime execution uses websocket command and process channels.',
  },
  ssh_compat: {
    mode: 'ssh_compat',
    label: 'SSH Compat',
    summary:
      'Legacy record only. Migrate this machine to websocket execution and keep SSH only as a helper channel.',
  },
}

export type WorkspaceRootContext = {
  draft: MachineDraft
  machine: Machine | null
}

export function normalizeConnectionMode(
  mode: string | null | undefined,
  host: string | null | undefined,
  reachabilityMode?: string | null | undefined,
  executionMode?: string | null | undefined,
): MachineConnectionMode {
  const normalizedReachability = normalizeReachabilityMode(reachabilityMode, host, mode)
  const normalizedExecution = normalizeExecutionMode(executionMode, host, mode)

  if (normalizedReachability === 'local') {
    return 'local'
  }
  if (normalizedReachability === 'reverse_connect') {
    return 'ws_reverse'
  }
  return normalizedExecution === 'ssh_compat' ? 'ssh' : 'ws_listener'
}

export function normalizeReachabilityMode(
  value: string | null | undefined,
  host: string | null | undefined,
  connectionMode?: string | null | undefined,
): MachineReachabilityMode {
  switch (value) {
    case 'local':
    case 'direct_connect':
    case 'reverse_connect':
      return value
  }

  switch (connectionMode) {
    case 'local':
      return 'local'
    case 'ws_reverse':
      return 'reverse_connect'
    case 'ssh':
    case 'ws_listener':
      return 'direct_connect'
    default:
      return (host ?? '').trim().toLowerCase() === 'local' ? 'local' : 'direct_connect'
  }
}

export function normalizeExecutionMode(
  value: string | null | undefined,
  host: string | null | undefined,
  connectionMode?: string | null | undefined,
): MachineExecutionMode {
  switch (value) {
    case 'local_process':
    case 'websocket':
    case 'ssh_compat':
      return value
  }

  switch (connectionMode) {
    case 'local':
      return 'local_process'
    case 'ssh':
      return 'ssh_compat'
    case 'ws_reverse':
    case 'ws_listener':
      return 'websocket'
    default:
      return (host ?? '').trim().toLowerCase() === 'local' ? 'local_process' : 'websocket'
  }
}

export function machineReachabilityLabel(mode: string | null | undefined): string {
  return machineModeGuide(normalizeReachabilityMode(mode, null)).label
}

export function machineExecutionModeLabel(mode: string | null | undefined): string {
  return machineExecutionGuide(normalizeExecutionMode(mode, null)).label
}

export function machineModeGuide(mode: MachineReachabilityMode): MachineModeGuide {
  return machineModeGuides[mode]
}

export function machineExecutionGuide(mode: MachineExecutionMode): MachineExecutionGuide {
  return machineExecutionGuides[mode]
}

export function machineDetectionMessage(machine: Machine | null, draft?: MachineDraft): string {
  if (machine?.detection_message?.trim()) {
    return machine.detection_message
  }

  const reachabilityMode = normalizeReachabilityMode(
    draft?.reachabilityMode ?? machine?.reachability_mode,
    draft?.host ?? machine?.host,
    machine?.connection_mode,
  )
  const executionMode = normalizeExecutionMode(
    draft?.executionMode ?? machine?.execution_mode,
    draft?.host ?? machine?.host,
    machine?.connection_mode,
  )

  if (reachabilityMode === 'local') {
    return 'Local machines default to the local OpenASE workspace convention. Run a connection test to confirm the platform details.'
  }
  if (executionMode === 'ssh_compat') {
    return 'This machine still stores a legacy SSH compatibility execution mode. Resave it as websocket execution and keep SSH only for bootstrap or diagnostics.'
  }
  return 'Detection is optional. You can keep saving the machine, then confirm the platform and workspace details after websocket checks or daemon registration.'
}

export function getWorkspaceRootRecommendation(
  input: WorkspaceRootContext,
): WorkspaceRootRecommendation {
  const reachabilityMode = normalizeReachabilityMode(
    input.draft.reachabilityMode || input.machine?.reachability_mode,
    input.draft.host || input.machine?.host,
    input.machine?.connection_mode,
  )
  const executionMode = normalizeExecutionMode(
    input.draft.executionMode || input.machine?.execution_mode,
    input.draft.host || input.machine?.host,
    input.machine?.connection_mode,
  )
  const detectedOS = normalizeDetectedOS(input.machine?.detected_os)
  const sshUser = input.draft.sshUser.trim() || input.machine?.ssh_user || 'openase'

  if (reachabilityMode === 'local') {
    return {
      value: '~/.openase/workspace',
      reason: 'Recommended local OpenASE workspace root.',
    }
  }
  if (detectedOS === 'darwin') {
    return {
      value: `/Users/${sshUser}/.openase/workspace`,
      reason: 'Recommended from detected macOS home directory layout.',
    }
  }
  if (detectedOS === 'linux') {
    return {
      value: `/home/${sshUser}/.openase/workspace`,
      reason: 'Recommended from detected Linux home directory layout.',
    }
  }

  return {
    value: defaultRemoteWorkspaceRoot,
    reason:
      executionMode === 'websocket'
        ? 'Fallback websocket workspace root until the remote platform is detected.'
        : 'Fallback remote workspace root while you migrate this legacy machine to websocket execution.',
  }
}

export function getWorkspaceRootState(input: WorkspaceRootContext): WorkspaceRootState {
  const recommended = getWorkspaceRootRecommendation(input).value
  const currentValue = input.draft.workspaceRoot.trim()
  const savedValue = input.machine?.workspace_root?.trim() ?? ''

  if (!currentValue) {
    return { kind: 'empty', label: 'No workspace root saved yet.' }
  }
  if (currentValue === recommended) {
    return { kind: 'recommended', label: 'Using the recommended workspace root.' }
  }
  if (savedValue && currentValue === savedValue) {
    return { kind: 'saved', label: 'Keeping the saved workspace root override.' }
  }
  return { kind: 'manual', label: 'Using a manual workspace root override.' }
}
