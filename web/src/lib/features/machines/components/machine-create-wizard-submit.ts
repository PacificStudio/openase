import type { Machine, MachineSSHBootstrapResult } from '$lib/api/contracts'
import {
  buildMachineCreateMutationInput,
  type MachineCreateWizardDraft,
} from './machine-create-wizard-flow'
import { runMachineHealthRefresh, runMachineSSHBootstrap, saveMachine } from './machines-page-api'

export class MachineCreateWizardSubmitError extends Error {
  stage: 'create' | 'bootstrap'
  override cause: unknown

  constructor(stage: 'create' | 'bootstrap', cause: unknown) {
    super(stage)
    this.name = 'MachineCreateWizardSubmitError'
    this.stage = stage
    this.cause = cause
  }
}

export async function submitMachineCreateWizard(input: {
  organizationId: string
  draft: MachineCreateWizardDraft
  setBootstrapping: (value: boolean) => void
}): Promise<{ machine: Machine; bootstrapResult: MachineSSHBootstrapResult | null }> {
  const { organizationId, draft, setBootstrapping } = input
  let createdMachine: Machine
  try {
    const created = await saveMachine(
      organizationId,
      null,
      'create',
      buildMachineCreateMutationInput(draft),
    )
    createdMachine = created.machine
  } catch (cause) {
    throw new MachineCreateWizardSubmitError('create', cause)
  }

  if (draft.strategy !== 'ssh-install-listener' && draft.strategy !== 'reverse') {
    return { machine: createdMachine, bootstrapResult: null }
  }

  setBootstrapping(true)
  try {
    const bootstrapResult = await runMachineSSHBootstrap(
      createdMachine.id,
      draft.strategy === 'ssh-install-listener'
        ? { topology: 'remote-listener', listener_address: '0.0.0.0:19837' }
        : { topology: 'reverse-connect' },
    )
    const refreshed = await runMachineHealthRefresh(createdMachine.id)
    return { machine: refreshed.machine, bootstrapResult }
  } catch (cause) {
    throw new MachineCreateWizardSubmitError('bootstrap', cause)
  } finally {
    setBootstrapping(false)
  }
}
