import { updateStatus } from '$lib/api/openase'
import {
  createEmptyStatusDraft,
  type EditableStatus,
  type ParsedStatusDraft,
} from '$lib/features/statuses/public'

export type StatusSettingsUI = {
  statuses: EditableStatus[]
  createName: string
  createStage: EditableStatus['stage']
  createColor: string
  createDefault: boolean
  createMaxActiveRuns: string
  loading: boolean
  creating: boolean
  resetting: boolean
  busyStatusId: string
}

export function createStatusSettingsUI(): StatusSettingsUI {
  return {
    statuses: [],
    createName: '',
    createStage: createEmptyStatusDraft().stage,
    createColor: createEmptyStatusDraft().color,
    createDefault: false,
    createMaxActiveRuns: '',
    loading: false,
    creating: false,
    resetting: false,
    busyStatusId: '',
  }
}

export function resetCreateStatusDraft(ui: StatusSettingsUI) {
  ui.createName = ''
  ui.createStage = createEmptyStatusDraft().stage
  ui.createColor = createEmptyStatusDraft().color
  ui.createDefault = false
  ui.createMaxActiveRuns = ''
}

export function statusUpdateBody(current: EditableStatus, draft: ParsedStatusDraft) {
  const body: Parameters<typeof updateStatus>[1] = {}
  if (draft.name !== current.name) body.name = draft.name
  if (draft.stage !== current.stage) body.stage = draft.stage
  if (draft.color !== current.color) body.color = draft.color
  if (draft.isDefault !== current.isDefault) body.is_default = draft.isDefault
  if (draft.maxActiveRuns !== current.maxActiveRuns) body.max_active_runs = draft.maxActiveRuns
  return body
}

export function statusCreateBody(draft: ParsedStatusDraft, isDefault: boolean) {
  return {
    name: draft.name,
    stage: draft.stage,
    color: draft.color,
    is_default: isDefault,
    max_active_runs: draft.maxActiveRuns,
  }
}
