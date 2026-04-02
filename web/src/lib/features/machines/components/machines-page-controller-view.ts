import type {
  MachineDraft,
  MachineItem,
  MachineProbeResult,
  MachineSnapshot,
  MachineWorkspaceState,
} from '../types'
import type { MachinesPageControllerView } from './machines-page-controller-types'

type MachinesPageControllerViewInput = {
  getRouteOrgId: () => string
  getLoading: () => boolean
  getRefreshing: () => boolean
  getWorkspaceState: () => MachineWorkspaceState
  getListMessage: () => string
  getMachines: () => MachineItem[]
  getFilteredMachines: () => MachineItem[]
  getSelectedId: () => string
  getSelectedMachine: () => MachineItem | null
  getMode: () => 'create' | 'edit'
  getDraft: () => MachineDraft
  setDraft: (value: MachineDraft) => void
  getSnapshot: () => MachineSnapshot | null
  getProbe: () => MachineProbeResult | null
  getLoadingHealth: () => boolean
  getRefreshingHealthMachineId: () => string
  getSaving: () => boolean
  getTestingMachineId: () => string
  getDeletingMachineId: () => string
  getEditorOpen: () => boolean
  setEditorOpen: (value: boolean) => void
  getSearchQuery: () => string
  setSearchQuery: (value: string) => void
  handleRefresh: () => Promise<void>
  startCreate: () => void
  openMachine: (machine: MachineItem, openEditorState?: boolean) => Promise<void>
  handleRefreshHealth: (machineId: string) => Promise<void>
  handleSave: () => Promise<void>
  handleTest: (machineId: string) => Promise<void>
  handleDelete: (machineId: string) => Promise<void>
  resetDraft: (machineId?: string) => void
}

export function createMachinesPageControllerView(
  input: MachinesPageControllerViewInput,
): MachinesPageControllerView {
  return {
    get routeOrgId() {
      return input.getRouteOrgId()
    },
    get loading() {
      return input.getLoading()
    },
    get refreshing() {
      return input.getRefreshing()
    },
    get workspaceState() {
      return input.getWorkspaceState()
    },
    get listMessage() {
      return input.getListMessage()
    },
    get machines() {
      return input.getMachines()
    },
    get filteredMachines() {
      return input.getFilteredMachines()
    },
    get selectedId() {
      return input.getSelectedId()
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
    get probe() {
      return input.getProbe()
    },
    get loadingHealth() {
      return input.getLoadingHealth()
    },
    get refreshingHealthMachineId() {
      return input.getRefreshingHealthMachineId()
    },
    get saving() {
      return input.getSaving()
    },
    get testingMachineId() {
      return input.getTestingMachineId()
    },
    get deletingMachineId() {
      return input.getDeletingMachineId()
    },
    get editorOpen() {
      return input.getEditorOpen()
    },
    set editorOpen(value) {
      input.setEditorOpen(value)
    },
    get searchQuery() {
      return input.getSearchQuery()
    },
    set searchQuery(value) {
      input.setSearchQuery(value)
    },
    handleRefresh: input.handleRefresh,
    startCreate: input.startCreate,
    openMachine: input.openMachine,
    handleRefreshHealth: input.handleRefreshHealth,
    handleSave: input.handleSave,
    handleTest: input.handleTest,
    handleDelete: input.handleDelete,
    resetDraft: input.resetDraft,
  }
}
