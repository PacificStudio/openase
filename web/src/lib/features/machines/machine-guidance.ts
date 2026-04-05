import type { Machine } from '$lib/api/contracts'
import type {
  MachineConnectionMode,
  MachineDetectedArch,
  MachineDetectedOS,
  MachineDetectionStatus,
  MachineDraft,
  MachineExecutionGuide,
  MachineExecutionMode,
  MachineModeGuide,
  MachineReachabilityMode,
  WorkspaceRootRecommendation,
  WorkspaceRootState,
} from './types'

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
      'Machine identity, a remote workspace root, and either websocket listener details or SSH compatibility access.',
    installMethod:
      'Preferred: expose a websocket listener endpoint. Compatibility: keep SSH helper access while migrating.',
    testSemantics:
      'Tests either dial the advertised websocket listener or use legacy SSH compatibility during rollout.',
    commonErrors:
      'Missing advertised endpoints, unreachable listener URLs, or stale SSH compatibility credentials.',
  },
  reverse_connect: {
    mode: 'reverse_connect',
    label: 'Reverse Connect',
    summary: 'The machine daemon dials out to the control plane.',
    requiredFields:
      'Machine identity, daemon registration, and a workspace root the daemon can access.',
    installMethod:
      'Install and register the daemon on the remote host, then keep the CLI path aligned with that host.',
    testSemantics:
      'Connection tests rely on daemon session state and any reported platform metadata.',
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
      'Legacy compatibility path during rollout. Keep only until the machine is migrated to websocket execution.',
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

export function normalizeDetectedOS(value: string | null | undefined): MachineDetectedOS {
  return value === 'darwin' || value === 'linux' ? value : 'unknown'
}

export function normalizeDetectedArch(value: string | null | undefined): MachineDetectedArch {
  return value === 'amd64' || value === 'arm64' ? value : 'unknown'
}

export function normalizeDetectionStatus(value: string | null | undefined): MachineDetectionStatus {
  return value === 'pending' || value === 'ok' || value === 'degraded' || value === 'unknown'
    ? value
    : 'unknown'
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

export function machineDetectedOSLabel(value: string | null | undefined): string {
  switch (normalizeDetectedOS(value)) {
    case 'darwin':
      return 'macOS'
    case 'linux':
      return 'Linux'
    default:
      return 'unknown'
  }
}

export function machineDetectedArchLabel(value: string | null | undefined): string {
  switch (normalizeDetectedArch(value)) {
    case 'amd64':
      return 'amd64'
    case 'arm64':
      return 'arm64'
    default:
      return 'unknown'
  }
}

export function machineDetectionStatusLabel(value: string | null | undefined): string {
  switch (normalizeDetectionStatus(value)) {
    case 'ok':
      return 'Detected'
    case 'degraded':
      return 'Degraded'
    case 'pending':
      return 'Pending'
    default:
      return 'Unknown'
  }
}

export function machineDetectionBadgeClass(value: string | null | undefined): string {
  switch (normalizeDetectionStatus(value)) {
    case 'ok':
      return 'border-emerald-500/30 bg-emerald-500/12 text-emerald-700'
    case 'degraded':
      return 'border-amber-500/30 bg-amber-500/14 text-amber-700'
    case 'pending':
      return 'border-sky-500/30 bg-sky-500/12 text-sky-700'
    default:
      return 'border-slate-500/20 bg-slate-500/10 text-slate-700'
  }
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
    return 'This machine still uses SSH compatibility during rollout. Keep SSH helper credentials in place until it is migrated to websocket execution.'
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
        : 'Fallback remote workspace root while this machine still uses SSH compatibility.',
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
