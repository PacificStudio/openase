import {
  createWorkflowsPageControllerActionsState,
  type WorkflowsPageControllerActionsState,
} from './workflows-page-controller-actions'
import type {
  WorkflowAgentOption,
  WorkflowStatusOption,
  WorkflowSummary,
  WorkflowTemplateDraft,
} from '../types'
import type { SkillState, toHarnessContent } from '../model'
import type { HarnessValidationIssue } from '$lib/api/contracts'

type WorkflowsPageControllerStateInput = {
  getSelectedId: () => string
  setSelectedId: (value: string) => void
  getHarness: () => ReturnType<typeof toHarnessContent> | null
  setHarness: (value: ReturnType<typeof toHarnessContent> | null) => void
  getDraftHarness: () => string
  setDraftHarness: (value: string) => void
  getLoadedHarnessWorkflowId: () => string
  setLoadedHarnessWorkflowId: (value: string) => void
  getSkillStates: () => SkillState[]
  setSkillStates: (value: SkillState[]) => void
  getValidationIssues: () => HarnessValidationIssue[]
  setValidationIssues: (value: HarnessValidationIssue[]) => void
  getWorkflows: () => WorkflowSummary[]
  setWorkflows: (value: WorkflowSummary[]) => void
  getTemplateDraft: () => WorkflowTemplateDraft | null
  setTemplateDraft: (value: WorkflowTemplateDraft | null) => void
  getStatuses: () => WorkflowStatusOption[]
  getAgentOptions: () => WorkflowAgentOption[]
  getIsDirty: () => boolean
}

export function createWorkflowsPageControllerState(
  input: WorkflowsPageControllerStateInput,
): WorkflowsPageControllerActionsState {
  return createWorkflowsPageControllerActionsState(input)
}
