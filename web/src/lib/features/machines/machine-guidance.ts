import type { Machine } from '$lib/api/contracts'
import type {
  MachineConnectionMode,
  MachineDetectedArch,
  MachineDetectedOS,
  MachineDetectionStatus,
  MachineDraft,
  MachineModeGuide,
  WorkspaceRootRecommendation,
  WorkspaceRootState,
} from './types'

const defaultRemoteWorkspaceRoot = '/srv/openase/workspace'

const machineModeGuides: Record<MachineConnectionMode, MachineModeGuide> = {
  local: {
    mode: 'local',
    label: 'Local',
    summary: 'Run OpenASE on the same host as the control plane.',
    requiredFields: 'Reserved local identity and a local workspace root.',
    installMethod: 'Use the local OpenASE CLI and local ticket workspaces under ~/.openase.',
    testSemantics: 'Connection tests verify the local runtime user, hostname, and platform.',
    commonErrors: 'A relative workspace root or a renamed local machine will be rejected.',
  },
  ssh: {
    mode: 'ssh',
    label: 'SSH',
    summary: 'Connect directly to a remote host over SSH.',
    requiredFields: 'Host, SSH user, SSH key path, and a remote workspace root.',
    installMethod:
      'Install the agent CLI on the remote host and point OpenASE at its absolute path.',
    testSemantics:
      'Connection tests open an SSH session and probe user, hostname, OS, and architecture.',
    commonErrors:
      'Missing SSH credentials, unreachable hosts, and remote paths that are not absolute.',
  },
  ws_reverse: {
    mode: 'ws_reverse',
    label: 'WS Reverse',
    summary: 'A machine daemon connects back to OpenASE and keeps the session open.',
    requiredFields:
      'Machine identity, daemon registration, and a workspace root the daemon can access.',
    installMethod:
      'Install and register the daemon on the remote host, then keep the CLI path aligned with that host.',
    testSemantics:
      'Connection tests rely on the daemon session and any reported system information.',
    commonErrors:
      'No registered daemon session, stale credentials, or an unknown workspace path on the remote host.',
  },
  ws_listener: {
    mode: 'ws_listener',
    label: 'WS Listener',
    summary: 'OpenASE connects to a machine-advertised websocket endpoint.',
    requiredFields: 'Machine identity, advertised listener endpoint, and a workspace root.',
    installMethod:
      'Run the daemon in listener mode and keep the advertised endpoint reachable from OpenASE.',
    testSemantics:
      'Connection tests use the advertised websocket listener and its handshake metadata.',
    commonErrors:
      'Missing advertised endpoint, unreachable listener URLs, or mismatched transport expectations.',
  },
}

export type WorkspaceRootContext = {
  draft: MachineDraft
  machine: Machine | null
}

export function normalizeConnectionMode(
  mode: string | null | undefined,
  host: string | null | undefined,
): MachineConnectionMode {
  switch (mode) {
    case 'local':
    case 'ssh':
    case 'ws_reverse':
    case 'ws_listener':
      return mode
    default:
      return (host ?? '').trim().toLowerCase() === 'local' ? 'local' : 'ssh'
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

export function machineConnectionModeLabel(mode: string | null | undefined): string {
  return machineModeGuide(normalizeConnectionMode(mode, null)).label
}

export function machineModeGuide(mode: MachineConnectionMode): MachineModeGuide {
  return machineModeGuides[mode]
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

  const mode = normalizeConnectionMode(
    draft?.connectionMode ?? machine?.connection_mode,
    draft?.host ?? machine?.host,
  )
  if (mode === 'local') {
    return 'Local machines default to the local OpenASE workspace convention. Run a connection test to confirm the platform details.'
  }

  return 'Detection is optional. You can keep saving the machine, then confirm the platform and paths after a connection test or daemon registration.'
}

export function getWorkspaceRootRecommendation(
  input: WorkspaceRootContext,
): WorkspaceRootRecommendation {
  const mode = normalizeConnectionMode(
    input.draft.connectionMode || input.machine?.connection_mode,
    input.draft.host || input.machine?.host,
  )
  const detectedOS = normalizeDetectedOS(input.machine?.detected_os)
  const sshUser = input.draft.sshUser.trim() || input.machine?.ssh_user || 'openase'

  if (mode === 'local') {
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
      mode === 'ws_reverse' || mode === 'ws_listener'
        ? 'Fallback websocket workspace root until the daemon reports a platform.'
        : 'Fallback remote workspace root until system detection is available.',
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
