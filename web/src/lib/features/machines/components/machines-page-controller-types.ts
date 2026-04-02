import type { MachinesPageViewState } from '../page-state'
import type {
  MachineDraft,
  MachineItem,
  MachineProbeResult,
  MachineSnapshot,
  MachineWorkspaceState,
} from '../types'

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
