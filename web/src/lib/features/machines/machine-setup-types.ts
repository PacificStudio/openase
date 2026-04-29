import type { Machine } from '$lib/api/contracts'
import type { MachineDraft, MachineSnapshot } from './types'

export type MachineSetupCommand = {
  title: string
  description: string
  command: string
}

export type MachineSetupGuide = {
  topologyLabel: string
  topologySummary: string
  runtimeLabel: string
  runtimeSummary: string
  helperLabel: string
  helperSummary: string
  stateLabel: string
  stateSummary: string
  nextSteps: string[]
  commands: MachineSetupCommand[]
}

export type MachineLike = Pick<
  Machine,
  | 'id'
  | 'host'
  | 'status'
  | 'ssh_user'
  | 'ssh_key_path'
  | 'advertised_endpoint'
  | 'reachability_mode'
  | 'execution_mode'
  | 'ssh_helper_enabled'
  | 'daemon_status'
>

export type DraftLike = Pick<
  MachineDraft,
  | 'host'
  | 'status'
  | 'sshUser'
  | 'sshKeyPath'
  | 'advertisedEndpoint'
  | 'reachabilityMode'
  | 'executionMode'
>

export type BuildMachineSetupGuideInput = {
  machine: MachineLike | null
  draft?: DraftLike | null
  snapshot?: MachineSnapshot | null
}
