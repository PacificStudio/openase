import { toastStore } from '$lib/stores/toast.svelte'
import { createEmptyMachineDraft, machineToDraft } from '../model'
import { type MachinesPageViewState, createNoOrgState, createStartCreateState } from '../page-state'
import type {
  MachineDraft,
  MachineItem,
  MachineProbeResult,
  MachineSnapshot,
  MachineWorkspaceState,
} from '../types'
import {
  loadMachineSnapshot,
  machineErrorMessage,
  removeMachine,
  runMachineConnectionTest,
  runMachineHealthRefresh,
} from './machines-page-api'
import { syncMachineListState } from './machines-page-state-sync'
import { writeMachineSnapshotCache } from '../machines-page-cache'

export type MachinesPageControllerOpsState = {
  get routeOrgId(): string
  get selectedId(): string
  get searchQuery(): string
  get machines(): MachineItem[]
  set machines(value: MachineItem[])
  get selectedMachine(): MachineItem | null
  get mode(): 'create' | 'edit'
  get draft(): MachineDraft
  set draft(value: MachineDraft)
  get snapshot(): MachineSnapshot | null
  set snapshot(value: MachineSnapshot | null)
  get probe(): MachineProbeResult | null
  set probe(value: MachineProbeResult | null)
  get editorOpen(): boolean
  set editorOpen(value: boolean)
  get activeOrgId(): string
  get snapshotRequestVersion(): number
  set snapshotRequestVersion(value: number)
  get loadingHealth(): boolean
  set loadingHealth(value: boolean)
  get testingMachineId(): string
  set testingMachineId(value: string)
  get refreshingHealthMachineId(): string
  set refreshingHealthMachineId(value: string)
  get deletingMachineId(): string
  set deletingMachineId(value: string)
  applyViewState(nextState: MachinesPageViewState): void
  persistMachinesPageCache(orgId: string): void
}

export function createMachinesPageControllerOpsState(input: {
  getRouteOrgId(): string
  getSelectedId(): string
  getSearchQuery(): string
  getMachines(): MachineItem[]
  setMachines(value: MachineItem[]): void
  getSelectedMachine(): MachineItem | null
  getMode(): 'create' | 'edit'
  getDraft(): MachineDraft
  setDraft(value: MachineDraft): void
  getSnapshot(): MachineSnapshot | null
  setSnapshot(value: MachineSnapshot | null): void
  getProbe(): MachineProbeResult | null
  setProbe(value: MachineProbeResult | null): void
  getEditorOpen(): boolean
  setEditorOpen(value: boolean): void
  getActiveOrgId(): string
  getSnapshotRequestVersion(): number
  setSnapshotRequestVersion(value: number): void
  getLoadingHealth(): boolean
  setLoadingHealth(value: boolean): void
  getTestingMachineId(): string
  setTestingMachineId(value: string): void
  getRefreshingHealthMachineId(): string
  setRefreshingHealthMachineId(value: string): void
  getDeletingMachineId(): string
  setDeletingMachineId(value: string): void
  applyViewState(nextState: MachinesPageViewState): void
  persistMachinesPageCache(orgId: string): void
}): MachinesPageControllerOpsState {
  return {
    get routeOrgId() {
      return input.getRouteOrgId()
    },
    get selectedId() {
      return input.getSelectedId()
    },
    get searchQuery() {
      return input.getSearchQuery()
    },
    get machines() {
      return input.getMachines()
    },
    set machines(value) {
      input.setMachines(value)
    },
    get selectedMachine() {
      return input.getSelectedMachine()
    },
    get mode() {
      return input.getMode()
    },
    get draft() {
      return input.getDraft()
    },
    set draft(value) {
      input.setDraft(value)
    },
    get snapshot() {
      return input.getSnapshot()
    },
    set snapshot(value) {
      input.setSnapshot(value)
    },
    get probe() {
      return input.getProbe()
    },
    set probe(value) {
      input.setProbe(value)
    },
    get editorOpen() {
      return input.getEditorOpen()
    },
    set editorOpen(value) {
      input.setEditorOpen(value)
    },
    get activeOrgId() {
      return input.getActiveOrgId()
    },
    get snapshotRequestVersion() {
      return input.getSnapshotRequestVersion()
    },
    set snapshotRequestVersion(value) {
      input.setSnapshotRequestVersion(value)
    },
    get loadingHealth() {
      return input.getLoadingHealth()
    },
    set loadingHealth(value) {
      input.setLoadingHealth(value)
    },
    get testingMachineId() {
      return input.getTestingMachineId()
    },
    set testingMachineId(value) {
      input.setTestingMachineId(value)
    },
    get refreshingHealthMachineId() {
      return input.getRefreshingHealthMachineId()
    },
    set refreshingHealthMachineId(value) {
      input.setRefreshingHealthMachineId(value)
    },
    get deletingMachineId() {
      return input.getDeletingMachineId()
    },
    set deletingMachineId(value) {
      input.setDeletingMachineId(value)
    },
    applyViewState: input.applyViewState,
    persistMachinesPageCache: input.persistMachinesPageCache,
  }
}

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

export type MachinesPageControllerView = {
  readonly routeOrgId: string
  readonly loading: boolean
  readonly refreshing: boolean
  readonly workspaceState: MachineWorkspaceState
  readonly listMessage: string
  readonly machines: MachineItem[]
  readonly filteredMachines: MachineItem[]
  readonly selectedId: string
  readonly selectedMachine: MachineItem | null
  readonly mode: 'create' | 'edit'
  draft: MachineDraft
  readonly snapshot: MachineSnapshot | null
  readonly probe: MachineProbeResult | null
  readonly loadingHealth: boolean
  readonly refreshingHealthMachineId: string
  readonly saving: boolean
  readonly testingMachineId: string
  readonly deletingMachineId: string
  editorOpen: boolean
  searchQuery: string
  handleRefresh(): Promise<void>
  startCreate(): void
  openMachine(machine: MachineItem, openEditorState?: boolean): Promise<void>
  handleRefreshHealth(machineId: string): Promise<void>
  handleSave(): Promise<void>
  handleTest(machineId: string): Promise<void>
  handleDelete(machineId: string): Promise<void>
  resetDraft(machineId?: string): void
}
