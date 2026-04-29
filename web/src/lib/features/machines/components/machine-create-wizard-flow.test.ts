import { describe, expect, it } from 'vitest'

import {
  applyMachineWizardLocationChoice,
  buildMachineCreateMutationInput,
  canAdvanceMachineWizardStep,
  computeMachineWizardStepOrder,
  createMachineCreateWizardDraft,
} from './machine-create-wizard-flow'

describe('machine create wizard flow', () => {
  it('uses the short local flow and seeds the local name', () => {
    const localDraft = applyMachineWizardLocationChoice(createMachineCreateWizardDraft(), 'local')

    expect(computeMachineWizardStepOrder(localDraft.location, localDraft.strategy)).toEqual([
      'location',
      'identity',
      'review',
    ])
    expect(localDraft.name).toBe('local')
    expect(localDraft.strategy).toBeNull()
  })

  it('requires endpoint and credentials only when the selected topology needs them', () => {
    expect(computeMachineWizardStepOrder('remote', 'ssh-install-listener')).toEqual([
      'location',
      'identity',
      'strategy',
      'credentials',
      'review',
    ])
    expect(computeMachineWizardStepOrder('remote', 'direct-open')).toEqual([
      'location',
      'identity',
      'strategy',
      'advertised-endpoint',
      'review',
    ])

    const pendingCredentials = {
      ...createMachineCreateWizardDraft(),
      step: 'credentials' as const,
      location: 'remote' as const,
      strategy: 'reverse' as const,
      sshUser: 'ubuntu',
      sshKeyPath: '',
    }
    expect(canAdvanceMachineWizardStep(pendingCredentials)).toBe(false)
  })

  it('builds the correct mutations for local, reverse, and direct-open machines', () => {
    const localMutation = buildMachineCreateMutationInput({
      ...createMachineCreateWizardDraft(),
      location: 'local',
      name: '  ',
    })
    expect(localMutation).toMatchObject({
      name: 'local',
      host: 'local',
      reachability_mode: 'local',
      execution_mode: 'local_process',
      status: 'online',
    })

    const reverseMutation = buildMachineCreateMutationInput({
      ...createMachineCreateWizardDraft(),
      location: 'remote',
      name: 'Reverse Builder',
      host: 'reverse.internal',
      strategy: 'reverse',
      sshUser: 'ubuntu',
      sshKeyPath: '/keys/id_ed25519',
    })
    expect(reverseMutation).toMatchObject({
      reachability_mode: 'reverse_connect',
      execution_mode: 'websocket',
      ssh_user: 'ubuntu',
      advertised_endpoint: '',
      status: 'offline',
    })

    const directOpenMutation = buildMachineCreateMutationInput({
      ...createMachineCreateWizardDraft(),
      location: 'remote',
      name: 'Direct Builder',
      host: 'builder.internal',
      strategy: 'direct-open',
      advertisedEndpoint: ' wss://builder.internal/openase/runtime ',
    })
    expect(directOpenMutation).toMatchObject({
      reachability_mode: 'direct_connect',
      execution_mode: 'websocket',
      ssh_user: '',
      ssh_key_path: '',
      advertised_endpoint: 'wss://builder.internal/openase/runtime',
    })
  })
})
