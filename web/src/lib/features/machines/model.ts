import type { Machine } from '$lib/api/contracts'
import type {
  MachineConnectionMode,
  MachineDetectedArch,
  MachineDetectedOS,
  MachineDetectionStatus,
  MachineDraft,
  MachineDraftParseResult,
  MachineModeGuide,
  MachineStatus,
  WorkspaceRootRecommendation,
  WorkspaceRootState,
} from './types'

export { parseMachineSnapshot } from './snapshot'

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

type WorkspaceRootContext = {
  draft: MachineDraft
  machine: Machine | null
}

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

  return {
    ...draft,
    workspaceRoot: getWorkspaceRootRecommendation({ draft, machine: null }).value,
  }
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

  if (field === 'connectionMode') {
    const nextMode = normalizeConnectionMode(value, nextDraft.host)
    nextDraft.connectionMode = nextMode
    if (nextMode === 'local') {
      if (!nextDraft.name.trim() || nextDraft.name.trim().toLowerCase() === 'local') {
        nextDraft.name = 'local'
      }
      nextDraft.host = 'local'
      nextDraft.port = '22'
      nextDraft.sshUser = ''
      nextDraft.sshKeyPath = ''
      nextDraft.advertisedEndpoint = ''
    } else if (draft.connectionMode === 'local') {
      if (nextDraft.name.trim().toLowerCase() === 'local') nextDraft.name = ''
      if (nextDraft.host.trim().toLowerCase() === 'local') nextDraft.host = ''
    }
    if (nextMode !== 'ws_listener') {
      nextDraft.advertisedEndpoint = ''
    }
  }

  const followsRecommendation =
    draft.workspaceRoot.trim() === '' || draft.workspaceRoot.trim() === previousRecommendation
  if ((field === 'connectionMode' || field === 'sshUser') && followsRecommendation) {
    nextDraft.workspaceRoot = getWorkspaceRootRecommendation({ draft: nextDraft, machine }).value
  }

  return nextDraft
}

export function parseMachineDraft(draft: MachineDraft): MachineDraftParseResult {
  const connectionMode = normalizeConnectionMode(draft.connectionMode, draft.host)
  const localMode = connectionMode === 'local'
  const name = localMode ? 'local' : draft.name.trim()
  const host = localMode ? 'local' : draft.host.trim()

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
  if (connectionMode === 'ssh') {
    if (!sshUser) {
      return { ok: false, error: 'SSH user is required for SSH machines.' }
    }
    if (!sshKeyPath) {
      return { ok: false, error: 'SSH key path is required for SSH machines.' }
    }
  }

  const advertisedEndpoint = draft.advertisedEndpoint.trim()
  if (connectionMode === 'ws_listener' && !advertisedEndpoint) {
    return { ok: false, error: 'Advertised endpoint is required for websocket listener machines.' }
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
