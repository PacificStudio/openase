import { toastStore } from '$lib/stores/toast.svelte'
import { syncMachineListState } from './machines-page-state-sync'
import { loadMachines, machineErrorMessage } from './machines-page-api'
import { loadMachineResources } from './machines-page-controller-ops'
import type { MachinesPageControllerOpsState } from './machines-page-controller-types'

export type MachinesPageControllerLoadState = MachinesPageControllerOpsState & {
  get loading(): boolean
  set loading(value: boolean)
  get refreshing(): boolean
  set refreshing(value: boolean)
  get activeOrgId(): string
  get listRequestVersion(): number
  set listRequestVersion(value: number)
  get queuedReload(): boolean
  set queuedReload(value: boolean)
  get reloadInFlight(): boolean
  set reloadInFlight(value: boolean)
}

export async function loadMachineList(
  state: MachinesPageControllerLoadState,
  orgId: string,
  options: { background: boolean; cancelled?: () => boolean },
) {
  state.listRequestVersion += 1
  const requestVersion = state.listRequestVersion
  state.loading = !options.background
  state.refreshing = options.background
  try {
    const nextMachines = await loadMachines(orgId)
    if (
      options.cancelled?.() ||
      state.activeOrgId !== orgId ||
      requestVersion !== state.listRequestVersion
    ) {
      return
    }
    const nextState = syncMachineListState({
      orgId,
      nextMachines,
      nextListError: null,
      selectedId: state.selectedId,
      searchQuery: state.searchQuery,
    })
    state.editorOpen = nextState.selectedMachineId !== null
    state.applyViewState(nextState.viewState)
    state.persistMachinesPageCache(orgId)
    if (nextState.selectedMachineId) {
      await loadMachineResources(state, nextState.selectedMachineId)
    }
  } catch (caughtError) {
    if (
      options.cancelled?.() ||
      state.activeOrgId !== orgId ||
      requestVersion !== state.listRequestVersion
    ) {
      return
    }
    if (options.background && state.machines.length > 0) {
      toastStore.error(machineErrorMessage(caughtError, 'Failed to refresh machines.'))
    } else {
      state.editorOpen = false
      state.applyViewState(
        syncMachineListState({
          orgId,
          nextMachines: [],
          nextListError: 'Failed to load machines.',
          selectedId: state.selectedId,
          searchQuery: state.searchQuery,
        }).viewState,
      )
    }
  } finally {
    if (
      !options.cancelled?.() &&
      state.activeOrgId === orgId &&
      requestVersion === state.listRequestVersion
    ) {
      state.loading = false
      state.refreshing = false
    }
  }
}

export function requestMachineReload(
  state: MachinesPageControllerLoadState,
  orgId: string,
  cancelled?: () => boolean,
) {
  state.queuedReload = true
  void drainMachineReloadQueue(state, orgId, cancelled)
}

async function drainMachineReloadQueue(
  state: MachinesPageControllerLoadState,
  orgId: string,
  cancelled?: () => boolean,
) {
  if (!state.queuedReload || state.reloadInFlight || state.activeOrgId !== orgId || cancelled?.())
    return
  state.reloadInFlight = true
  state.queuedReload = false
  try {
    await loadMachineList(state, orgId, { background: true, cancelled })
  } finally {
    state.reloadInFlight = false
    if (state.queuedReload && state.activeOrgId === orgId && !cancelled?.()) {
      void drainMachineReloadQueue(state, orgId, cancelled)
    }
  }
}
