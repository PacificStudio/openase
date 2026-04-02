import { toastStore } from '$lib/stores/toast.svelte'
import { createEmptyMachineDraft, machineToDraft } from '../model'
import { createNoOrgState, createStartCreateState } from '../page-state'
import {
  loadMachineSnapshot,
  machineErrorMessage,
  removeMachine,
  runMachineConnectionTest,
  runMachineHealthRefresh,
} from './machines-page-api'
import { syncMachineListState } from './machines-page-state-sync'
import { writeMachineSnapshotCache } from '../machines-page-cache'
import type { MachinesPageControllerOpsState } from './machines-page-controller-types'

export async function loadMachineResources(
  state: MachinesPageControllerOpsState,
  machineId: string,
) {
  const orgId = state.routeOrgId
  if (!orgId) return

  state.snapshotRequestVersion += 1
  const requestVersion = state.snapshotRequestVersion
  state.loadingHealth = true
  try {
    const nextSnapshot = await loadMachineSnapshot(machineId)
    if (
      state.activeOrgId !== orgId ||
      requestVersion !== state.snapshotRequestVersion ||
      state.selectedId !== machineId
    ) {
      return
    }
    state.snapshot = nextSnapshot
    writeMachineSnapshotCache(orgId, machineId, nextSnapshot)
  } catch (caughtError) {
    toastStore.error(machineErrorMessage(caughtError, 'Failed to load machine resources.'))
  } finally {
    if (
      state.activeOrgId === orgId &&
      requestVersion === state.snapshotRequestVersion &&
      state.selectedId === machineId
    ) {
      state.loadingHealth = false
    }
  }
}

export function resetMachineDraft(
  state: Pick<MachinesPageControllerOpsState, 'mode' | 'selectedId' | 'selectedMachine' | 'draft'>,
  machineId?: string,
) {
  if (machineId && machineId !== state.selectedId) return
  if (state.mode === 'create') {
    state.draft = createEmptyMachineDraft()
    return
  }
  if (state.selectedMachine) {
    state.draft = machineToDraft(state.selectedMachine)
  }
}

export function startMachineCreate(
  state: Pick<
    MachinesPageControllerOpsState,
    'routeOrgId' | 'machines' | 'searchQuery' | 'applyViewState' | 'editorOpen'
  >,
) {
  if (!state.routeOrgId) {
    state.applyViewState(createNoOrgState())
    return
  }
  state.applyViewState({
    ...createStartCreateState(state.routeOrgId, state.machines),
    searchQuery: state.searchQuery,
  })
  state.editorOpen = true
}

export async function handleMachineTest(state: MachinesPageControllerOpsState, machineId: string) {
  if (!state.machines.find((item) => item.id === machineId)) return
  state.testingMachineId = machineId
  try {
    const payload = await runMachineConnectionTest(machineId)
    state.machines = state.machines.map((machine) =>
      machine.id === payload.machine.id ? payload.machine : machine,
    )
    if (state.selectedId === machineId) {
      state.snapshot = payload.snapshot
      state.probe = payload.probe
      writeMachineSnapshotCache(state.routeOrgId, machineId, payload.snapshot)
    }
    state.persistMachinesPageCache(state.routeOrgId)
    toastStore.success('Connection test completed.')
  } catch (caughtError) {
    toastStore.error(machineErrorMessage(caughtError, 'Failed to run connection test.'))
  } finally {
    state.testingMachineId = ''
  }
}

export async function handleMachineHealthRefresh(
  state: MachinesPageControllerOpsState,
  machineId: string,
) {
  if (!state.machines.find((item) => item.id === machineId)) return
  state.refreshingHealthMachineId = machineId
  try {
    const payload = await runMachineHealthRefresh(machineId)
    state.machines = state.machines.map((item) =>
      item.id === payload.machine.id ? payload.machine : item,
    )
    if (state.selectedId === machineId) {
      state.snapshot = payload.snapshot
      writeMachineSnapshotCache(state.routeOrgId, machineId, payload.snapshot)
    }
    state.persistMachinesPageCache(state.routeOrgId)
    toastStore.success('Machine health checks refreshed.')
  } catch (caughtError) {
    toastStore.error(machineErrorMessage(caughtError, 'Failed to refresh machine health.'))
  } finally {
    state.refreshingHealthMachineId = ''
  }
}

export async function handleMachineDelete(
  state: MachinesPageControllerOpsState,
  machineId: string,
) {
  if (!state.machines.find((item) => item.id === machineId)) return
  state.deletingMachineId = machineId
  try {
    await removeMachine(machineId)
    const nextMachines = state.machines.filter((item) => item.id !== machineId)
    state.machines = nextMachines
    toastStore.success('Machine deleted.')
    if (state.selectedId === machineId) {
      state.probe = null
      state.snapshot = null
      state.editorOpen = false
      state.applyViewState(
        syncMachineListState({
          orgId: state.routeOrgId,
          nextMachines,
          nextListError: null,
          selectedId: '',
          searchQuery: state.searchQuery,
        }).viewState,
      )
    }
    state.persistMachinesPageCache(state.routeOrgId)
  } catch (caughtError) {
    toastStore.error(machineErrorMessage(caughtError, 'Failed to delete machine.'))
  } finally {
    state.deletingMachineId = ''
  }
}
