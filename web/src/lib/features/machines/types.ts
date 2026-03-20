import type { Machine, MachineProbe, Organization } from '$lib/api/contracts'

export type ResourceMap = Record<string, unknown>

export type MachineStatus = 'online' | 'offline' | 'degraded' | 'maintenance'

export type MachineDraft = {
  name: string
  host: string
  port: string
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
}

export type MachineGPUView = {
  index: number
  name: string
  memoryTotalGB: number
  memoryUsedGB: number
  utilizationPercent: number
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
  monitor: {
    l1?: MachineMonitorLevel
    l2?: MachineMonitorLevel
    l3?: MachineMonitorLevel
  }
  monitorErrors: string[]
}

export type MachineProbeResult = MachineProbe

export type MachineDraftField = keyof MachineDraft

export type MachineEditorMode = 'create' | 'edit'

export type MachineItem = Machine

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
