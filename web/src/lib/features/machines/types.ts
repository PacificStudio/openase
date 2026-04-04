import type { Machine, MachineProbe, Organization } from '$lib/api/contracts'

export type ResourceMap = Record<string, unknown>

export type MachineStatus = 'online' | 'offline' | 'degraded' | 'maintenance'
export type MachineConnectionMode = 'local' | 'ssh' | 'ws_reverse' | 'ws_listener'
export type MachineDetectedOS = 'darwin' | 'linux' | 'unknown'
export type MachineDetectedArch = 'amd64' | 'arm64' | 'unknown'
export type MachineDetectionStatus = 'pending' | 'ok' | 'degraded' | 'unknown'

export type MachineDraft = {
  name: string
  host: string
  port: string
  connectionMode: MachineConnectionMode
  advertisedEndpoint: string
  sshUser: string
  sshKeyPath: string
  description: string
  labels: string
  status: MachineStatus
  workspaceRoot: string
  agentCLIPath: string
  envVars: string
}

export type MachineMutationInput = {
  name: string
  host: string
  port: number
  connection_mode: MachineConnectionMode
  advertised_endpoint: string
  ssh_user: string
  ssh_key_path: string
  description: string
  labels: string[]
  status: MachineStatus
  workspace_root: string
  agent_cli_path: string
  env_vars: string[]
}

export type MachineDraftParseResult =
  | { ok: true; value: MachineMutationInput }
  | { ok: false; error: string }

export type MachineMonitorLevel = {
  checkedAt?: string
  error?: string
  transport?: string
  reachable?: boolean
  latencyMs?: number
  failureCause?: string
  consecutiveFailures?: number
  memoryLow?: boolean
  diskLow?: boolean
  available?: boolean
  gpuDispatchable?: boolean
  agentDispatchable?: boolean
}

export type MachineGPUView = {
  index: number
  name: string
  memoryTotalGB: number
  memoryUsedGB: number
  utilizationPercent: number
}

export type MachineCLIStatus = {
  name: string
  installed?: boolean
  version?: string
  authStatus?: string
  authMode?: string
  ready?: boolean
}

export type MachineGitAuditView = {
  installed?: boolean
  userName?: string
  userEmail?: string
}

export type MachineGitHubCLIAuditView = {
  installed?: boolean
  authStatus?: string
}

export type MachineGitHubTokenProbeView = {
  checkedAt?: string
  state?: string
  configured?: boolean
  valid?: boolean
  permissions: string[]
  repoAccess?: string
  lastError?: string
}

export type MachineNetworkAuditView = {
  githubReachable?: boolean
  pypiReachable?: boolean
  npmReachable?: boolean
}

export type MachineSnapshot = {
  transport?: string
  checkedAt?: string
  lastSuccess?: boolean
  cpuCores?: number
  cpuUsagePercent?: number
  memoryTotalGB?: number
  memoryUsedGB?: number
  memoryAvailableGB?: number
  diskTotalGB?: number
  diskAvailableGB?: number
  gpuDispatchable?: boolean
  gpus: MachineGPUView[]
  agentDispatchable?: boolean
  agentEnvironmentCheckedAt?: string
  agentEnvironment: MachineCLIStatus[]
  monitor: {
    l1?: MachineMonitorLevel
    l2?: MachineMonitorLevel
    l3?: MachineMonitorLevel
    l4?: MachineMonitorLevel
    l5?: MachineMonitorLevel
  }
  fullAudit?: {
    checkedAt?: string
    git?: MachineGitAuditView
    ghCLI?: MachineGitHubCLIAuditView
    githubTokenProbe?: MachineGitHubTokenProbeView
    network?: MachineNetworkAuditView
  }
  monitorErrors: string[]
}

export type MachineProbeResult = MachineProbe

export type MachineDraftField = keyof MachineDraft

export type MachineEditorMode = 'create' | 'edit'

export type MachineItem = Machine

export type MachineModeGuide = {
  mode: MachineConnectionMode
  label: string
  summary: string
  requiredFields: string
  installMethod: string
  testSemantics: string
  commonErrors: string
}

export type WorkspaceRootRecommendation = {
  value: string
  reason: string
}

export type WorkspaceRootState =
  | { kind: 'recommended'; label: string }
  | { kind: 'saved'; label: string }
  | { kind: 'manual'; label: string }
  | { kind: 'empty'; label: string }

export type MachineWorkspaceState = 'no-org' | 'loading' | 'error' | 'empty' | 'ready'

export type MachinesPageOrgContext =
  | { kind: 'ready'; org: Organization }
  | { kind: 'no-org' }
  | { kind: 'error'; message: string }

export type MachinesPageData = {
  orgContext: MachinesPageOrgContext
  initialMachines: MachineItem[]
  initialListError: string | null
}
