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
  editorError: string
  feedback: string
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

export function createListErrorState(
  message: string,
  feedback = '',
  preserveFeedback = false,
): MachinesPageViewState {
  return {
    ...resetEditorState(feedback, preserveFeedback),
    routeOrgId: '',
    machines: [],
    searchQuery: '',
    workspaceState: 'error',
    listMessage: message,
  }
}

export function createEmptyState(
  routeOrgId: string,
  feedback = '',
  preserveFeedback = false,
): MachinesPageViewState {
  return {
    ...resetEditorState(feedback, preserveFeedback),
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
  feedback = '',
  preserveFeedback = false,
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
    editorError: '',
    feedback: preserveFeedback ? feedback : '',
  }
}

export function createStartCreateState(
  routeOrgId: string,
  machines: MachineItem[],
  feedback = '',
  preserveFeedback = false,
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
    editorError: '',
    feedback: preserveFeedback ? feedback : '',
  }
}

function resetEditorState(feedback = '', preserveFeedback = false) {
  return {
    selectedId: '',
    mode: 'edit' as const,
    draft: createEmptyMachineDraft(),
    snapshot: null,
    probe: null,
    editorError: '',
    feedback: preserveFeedback ? feedback : '',
  }
}
