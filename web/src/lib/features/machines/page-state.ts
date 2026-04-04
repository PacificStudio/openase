import { createEmptyMachineDraft, machineToDraft, parseMachineSnapshot } from './model'
import type {
  MachineDraft,
  MachineEditorMode,
  MachineItem,
  MachineProbeResult,
  MachineSnapshot,
  MachineWorkspaceState,
} from './types'

export type MachinesPageViewState = {
  routeOrgId: string
  machines: MachineItem[]
  searchQuery: string
  workspaceState: MachineWorkspaceState
  listMessage: string
  selectedId: string
  mode: MachineEditorMode
  draft: MachineDraft
  snapshot: MachineSnapshot | null
  probe: MachineProbeResult | null
}

export function createNoOrgState(): MachinesPageViewState {
  return {
    ...resetEditorState(),
    routeOrgId: '',
    machines: [],
    searchQuery: '',
    workspaceState: 'no-org',
    listMessage: '',
  }
}

export function createListErrorState(message: string): MachinesPageViewState {
  return {
    ...resetEditorState(),
    routeOrgId: '',
    machines: [],
    searchQuery: '',
    workspaceState: 'error',
    listMessage: message,
  }
}

export function createEmptyState(routeOrgId: string): MachinesPageViewState {
  return {
    ...resetEditorState(),
    routeOrgId,
    machines: [],
    searchQuery: '',
    workspaceState: 'empty',
    listMessage: '',
  }
}

export function createEditorSelectionState(
  routeOrgId: string,
  machines: MachineItem[],
  machine: MachineItem,
): MachinesPageViewState {
  return {
    routeOrgId,
    machines,
    searchQuery: '',
    workspaceState: 'ready',
    listMessage: '',
    selectedId: machine.id,
    mode: 'edit',
    draft: machineToDraft(machine),
    snapshot: parseMachineSnapshot(machine.resources),
    probe: null,
  }
}

export function createStartCreateState(
  routeOrgId: string,
  machines: MachineItem[],
): MachinesPageViewState {
  return {
    routeOrgId,
    machines,
    searchQuery: '',
    workspaceState: 'ready',
    listMessage: '',
    selectedId: '',
    mode: 'create',
    draft: createEmptyMachineDraft(),
    snapshot: null,
    probe: null,
  }
}

function resetEditorState() {
  return {
    selectedId: '',
    mode: 'edit' as const,
    draft: createEmptyMachineDraft(),
    snapshot: null,
    probe: null,
  }
}
