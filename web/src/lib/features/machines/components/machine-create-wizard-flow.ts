import type { MachineMutationInput } from '../types'
import type {
  MachineWizardLocationAnswer,
  MachineWizardStep,
  MachineWizardStrategy,
} from './machine-create-wizard-types'

export const DEFAULT_MACHINE_SSH_KEY_PATH = '~/.ssh/id_ed25519'

export type MachineCreateWizardDraft = {
  step: MachineWizardStep
  location: MachineWizardLocationAnswer | null
  name: string
  host: string
  strategy: MachineWizardStrategy | null
  sshUser: string
  sshKeyPath: string
  advertisedEndpoint: string
}

export function createMachineCreateWizardDraft(): MachineCreateWizardDraft {
  return {
    step: 'location',
    location: null,
    name: '',
    host: '',
    strategy: null,
    sshUser: '',
    sshKeyPath: DEFAULT_MACHINE_SSH_KEY_PATH,
    advertisedEndpoint: '',
  }
}

export function computeMachineWizardStepOrder(
  location: MachineWizardLocationAnswer | null,
  strategy: MachineWizardStrategy | null,
): MachineWizardStep[] {
  if (location === 'local') return ['location', 'identity', 'review']

  const stepOrder: MachineWizardStep[] = ['location', 'identity', 'strategy']
  if (strategy === 'direct-open') stepOrder.push('advertised-endpoint')
  if (strategy === 'ssh-install-listener' || strategy === 'reverse') {
    stepOrder.push('credentials')
  }
  stepOrder.push('review')
  return stepOrder
}

export function canAdvanceMachineWizardStep(draft: MachineCreateWizardDraft): boolean {
  switch (draft.step) {
    case 'location':
      return draft.location !== null
    case 'identity':
      if (draft.location === 'local') return draft.name.trim().length > 0
      return draft.name.trim().length > 0 && draft.host.trim().length > 0
    case 'strategy':
      return draft.strategy !== null
    case 'credentials':
      return draft.sshUser.trim().length > 0 && draft.sshKeyPath.trim().length > 0
    case 'advertised-endpoint':
      return draft.advertisedEndpoint.trim().length > 0
    case 'review':
      return true
  }
}

export function applyMachineWizardLocationChoice(
  draft: MachineCreateWizardDraft,
  location: MachineWizardLocationAnswer,
): MachineCreateWizardDraft {
  return {
    ...draft,
    location,
    strategy: location === 'local' ? null : (draft.strategy ?? 'ssh-install-listener'),
    name: location === 'local' ? draft.name || 'local' : draft.name,
  }
}

export function buildMachineCreateMutationInput(
  draft: MachineCreateWizardDraft,
): MachineMutationInput {
  if (draft.location === 'local') {
    return {
      name: draft.name.trim() || 'local',
      host: 'local',
      port: 22,
      reachability_mode: 'local',
      execution_mode: 'local_process',
      ssh_user: '',
      ssh_key_path: '',
      advertised_endpoint: '',
      description: '',
      labels: [],
      status: 'online',
      workspace_root: '',
      agent_cli_path: '',
      env_vars: [],
    }
  }

  const name = draft.name.trim()
  const host = draft.host.trim()
  const strategy = draft.strategy
  if (strategy === 'reverse') {
    return {
      name,
      host,
      port: 22,
      reachability_mode: 'reverse_connect',
      execution_mode: 'websocket',
      ssh_user: draft.sshUser.trim(),
      ssh_key_path: draft.sshKeyPath.trim(),
      advertised_endpoint: '',
      description: '',
      labels: [],
      status: 'offline',
      workspace_root: '',
      agent_cli_path: '',
      env_vars: [],
    }
  }

  const advertisedEndpoint =
    strategy === 'direct-open'
      ? draft.advertisedEndpoint.trim()
      : `ws://${host}:19837/openase/runtime`

  return {
    name,
    host,
    port: 22,
    reachability_mode: 'direct_connect',
    execution_mode: 'websocket',
    ssh_user: strategy === 'ssh-install-listener' ? draft.sshUser.trim() : '',
    ssh_key_path: strategy === 'ssh-install-listener' ? draft.sshKeyPath.trim() : '',
    advertised_endpoint: advertisedEndpoint,
    description: '',
    labels: [],
    status: 'offline',
    workspace_root: '',
    agent_cli_path: '',
    env_vars: [],
  }
}
