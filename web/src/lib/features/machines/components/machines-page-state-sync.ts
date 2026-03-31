import { createEditorSelectionState, createEmptyState, createListErrorState } from '../page-state'
import type { MachinesPageViewState } from '../page-state'
import type { MachineItem } from '../types'

export function syncMachineListState(args: {
  orgId: string
  nextMachines: MachineItem[]
  nextListError: string | null
  selectedId: string
  searchQuery: string
}): {
  viewState: MachinesPageViewState
  selectedMachineId: string | null
} {
  const { orgId, nextMachines, nextListError, selectedId, searchQuery } = args
  if (nextListError) {
    return {
      viewState: createListErrorState(nextListError),
      selectedMachineId: null,
    }
  }

  if (nextMachines.length === 0) {
    return {
      viewState: createEmptyState(orgId),
      selectedMachineId: null,
    }
  }

  const nextMachine = selectedId
    ? (nextMachines.find((machine) => machine.id === selectedId) ?? null)
    : null

  if (!nextMachine) {
    return {
      viewState: {
        ...createEmptyState(orgId),
        machines: nextMachines,
        workspaceState: 'ready',
        searchQuery,
      },
      selectedMachineId: null,
    }
  }

  return {
    viewState: {
      ...createEditorSelectionState(orgId, nextMachines, nextMachine),
      searchQuery,
    },
    selectedMachineId: nextMachine.id,
  }
}
