import { PROJECT_AI_FOCUS_PRIORITY, type ProjectAIFocus } from '$lib/features/chat'
import type { MachineEditorMode, MachineItem, MachineSnapshot } from '../types'

export function buildMachinesPageProjectAIFocus(
  projectId: string,
  machine: MachineItem | null,
  snapshot: MachineSnapshot | null,
): ProjectAIFocus | null {
  if (!projectId || !machine) {
    return null
  }

  return {
    kind: 'machine',
    projectId,
    machineId: machine.id,
    machineName: machine.name,
    machineHost: machine.host,
    machineStatus: machine.status,
    selectedArea: snapshot ? 'health' : 'editor',
    healthSummary: summarizeMachineFocus(machine, snapshot),
  }
}

export function syncMachinesPageProjectAIFocus(input: {
  clearFocus: (owner: string) => void
  setFocus: (owner: string, focus: ProjectAIFocus, priority?: number) => void
  owner: string
  projectId: string
  editorOpen: boolean
  mode: MachineEditorMode
  selectedMachine: MachineItem | null
  snapshot: MachineSnapshot | null
}) {
  const focus =
    !input.editorOpen || input.mode !== 'edit'
      ? null
      : buildMachinesPageProjectAIFocus(input.projectId, input.selectedMachine, input.snapshot)
  if (!focus) {
    input.clearFocus(input.owner)
    return
  }
  input.setFocus(input.owner, focus, PROJECT_AI_FOCUS_PRIORITY.workspace)
  return () => {
    input.clearFocus(input.owner)
  }
}

function summarizeMachineFocus(machine: MachineItem, snapshot: MachineSnapshot | null) {
  const parts = [machine.status]
  if (snapshot?.checkedAt) {
    parts.push(`checked ${snapshot.checkedAt}`)
  }
  if (typeof snapshot?.agentDispatchable === 'boolean') {
    parts.push(snapshot.agentDispatchable ? 'agent ready' : 'agent blocked')
  }
  if ((snapshot?.monitorErrors?.length ?? 0) > 0) {
    parts.push(`${snapshot?.monitorErrors.length ?? 0} monitor errors`)
  }
  return parts.filter(Boolean).join(' · ')
}
