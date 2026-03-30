import { ApiError } from '$lib/api/client'
import {
  createMachine,
  deleteMachine,
  getMachineResources,
  listMachines,
  refreshMachineHealth,
  testMachineConnection,
  updateMachine,
} from '$lib/api/openase'
import { parseMachineSnapshot } from '../model'
import type {
  MachineItem,
  MachineMutationInput,
  MachineProbeResult,
  MachineSnapshot,
} from '../types'

export async function loadMachines(orgId: string): Promise<MachineItem[]> {
  const payload = await listMachines(orgId)
  return payload.machines ?? []
}

export async function loadMachineSnapshot(machineId: string): Promise<MachineSnapshot | null> {
  const payload = await getMachineResources(machineId)
  return parseMachineSnapshot(payload.resources)
}

export async function saveMachine(
  orgId: string,
  selectedMachine: MachineItem | null,
  mode: 'create' | 'edit',
  draft: MachineMutationInput,
): Promise<{ machine: MachineItem; feedback: string }> {
  if (mode === 'create') {
    const payload = await createMachine(orgId, draft)
    return { machine: payload.machine, feedback: 'Machine created.' }
  }

  if (!selectedMachine) {
    throw new Error('Selected machine is unavailable.')
  }

  const payload = await updateMachine(selectedMachine.id, draft)
  return { machine: payload.machine, feedback: 'Machine updated.' }
}

export async function runMachineConnectionTest(machineId: string): Promise<{
  machine: MachineItem
  probe: MachineProbeResult | null
  snapshot: MachineSnapshot | null
}> {
  const payload = await testMachineConnection(machineId)
  return {
    machine: payload.machine,
    probe: payload.probe,
    snapshot: parseMachineSnapshot(payload.machine.resources),
  }
}

export async function runMachineHealthRefresh(machineId: string): Promise<{
  machine: MachineItem
  snapshot: MachineSnapshot | null
}> {
  const payload = await refreshMachineHealth(machineId)
  return {
    machine: payload.machine,
    snapshot: parseMachineSnapshot(payload.machine.resources),
  }
}

export async function removeMachine(machineId: string): Promise<void> {
  await deleteMachine(machineId)
}

export function machineErrorMessage(caughtError: unknown, fallback: string): string {
  return caughtError instanceof ApiError ? caughtError.detail : fallback
}
