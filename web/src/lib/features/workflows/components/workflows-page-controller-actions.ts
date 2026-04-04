import { ApiError } from '$lib/api/client'
import {
  bindWorkflowSkills,
  saveWorkflowHarness,
  unbindWorkflowSkills,
  validateHarness,
} from '$lib/api/openase'
import type { BuiltinRole, HarnessValidationIssue } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { type SkillState, toHarnessContent } from '../model'
import type { WorkflowAgentOption, WorkflowSummary, WorkflowTemplateDraft } from '../types'
import { loadWorkflowHarness } from '../data'
import {
  applyWorkflowVersionRefresh,
  buildWorkflowTemplateDraft,
} from '../workflow-page-actions-support'

type HarnessContent = ReturnType<typeof toHarnessContent>

export type WorkflowsPageControllerActionsState = {
  get selectedId(): string
  set selectedId(value: string)
  get harness(): HarnessContent | null
  set harness(value: HarnessContent | null)
  get draftHarness(): string
  set draftHarness(value: string)
  get loadedHarnessWorkflowId(): string
  set loadedHarnessWorkflowId(value: string)
  get skillStates(): SkillState[]
  set skillStates(value: SkillState[])
  get validationIssues(): HarnessValidationIssue[]
  set validationIssues(value: HarnessValidationIssue[])
  get workflows(): WorkflowSummary[]
  set workflows(value: WorkflowSummary[])
  get templateDraft(): WorkflowTemplateDraft | null
  set templateDraft(value: WorkflowTemplateDraft | null)
  get statuses(): { id: string }[]
  get agentOptions(): WorkflowAgentOption[]
  get isDirty(): boolean
}

export function createWorkflowsPageControllerActionsState(input: {
  getSelectedId(): string
  setSelectedId(value: string): void
  getHarness(): HarnessContent | null
  setHarness(value: HarnessContent | null): void
  getDraftHarness(): string
  setDraftHarness(value: string): void
  getLoadedHarnessWorkflowId(): string
  setLoadedHarnessWorkflowId(value: string): void
  getSkillStates(): SkillState[]
  setSkillStates(value: SkillState[]): void
  getValidationIssues(): HarnessValidationIssue[]
  setValidationIssues(value: HarnessValidationIssue[]): void
  getWorkflows(): WorkflowSummary[]
  setWorkflows(value: WorkflowSummary[]): void
  getTemplateDraft(): WorkflowTemplateDraft | null
  setTemplateDraft(value: WorkflowTemplateDraft | null): void
  getStatuses(): { id: string }[]
  getAgentOptions(): WorkflowAgentOption[]
  getIsDirty(): boolean
}): WorkflowsPageControllerActionsState {
  return {
    get selectedId() {
      return input.getSelectedId()
    },
    set selectedId(value) {
      input.setSelectedId(value)
    },
    get harness() {
      return input.getHarness()
    },
    set harness(value) {
      input.setHarness(value)
    },
    get draftHarness() {
      return input.getDraftHarness()
    },
    set draftHarness(value) {
      input.setDraftHarness(value)
    },
    get loadedHarnessWorkflowId() {
      return input.getLoadedHarnessWorkflowId()
    },
    set loadedHarnessWorkflowId(value) {
      input.setLoadedHarnessWorkflowId(value)
    },
    get skillStates() {
      return input.getSkillStates()
    },
    set skillStates(value) {
      input.setSkillStates(value)
    },
    get validationIssues() {
      return input.getValidationIssues()
    },
    set validationIssues(value) {
      input.setValidationIssues(value)
    },
    get workflows() {
      return input.getWorkflows()
    },
    set workflows(value) {
      input.setWorkflows(value)
    },
    get templateDraft() {
      return input.getTemplateDraft()
    },
    set templateDraft(value) {
      input.setTemplateDraft(value)
    },
    get statuses() {
      return input.getStatuses()
    },
    get agentOptions() {
      return input.getAgentOptions()
    },
    get isDirty() {
      return input.getIsDirty()
    },
  }
}

async function refreshSelectedWorkflowHarness(
  state: Pick<
    WorkflowsPageControllerActionsState,
    | 'selectedId'
    | 'harness'
    | 'draftHarness'
    | 'loadedHarnessWorkflowId'
    | 'skillStates'
    | 'workflows'
  >,
) {
  const projectId = appStore.currentProject?.id
  if (!projectId || !state.selectedId) return null
  const refreshed = await loadWorkflowHarness(projectId, state.selectedId)
  state.harness = refreshed.harness
  state.draftHarness = refreshed.harness.rawContent
  state.loadedHarnessWorkflowId = state.selectedId
  state.skillStates = refreshed.skillStates
  state.workflows = state.workflows.map((workflow) =>
    workflow.id === state.selectedId ? { ...workflow, history: refreshed.history } : workflow,
  )
  return refreshed
}

export async function handleSaveWorkflow(
  state: Pick<
    WorkflowsPageControllerActionsState,
    | 'selectedId'
    | 'draftHarness'
    | 'harness'
    | 'loadedHarnessWorkflowId'
    | 'skillStates'
    | 'workflows'
  >,
) {
  if (!state.selectedId) return
  try {
    const payload = await saveWorkflowHarness(state.selectedId, state.draftHarness)
    const refreshed = await refreshSelectedWorkflowHarness(state)
    if (!refreshed) return
    state.workflows = applyWorkflowVersionRefresh(state.workflows, state.selectedId, {
      harnessPath: payload.harness.path,
      version: payload.harness.version,
      history: refreshed.history,
    })
    toastStore.success(
      payload.harness.version ? `Harness saved as v${payload.harness.version}.` : 'Harness saved.',
    )
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to save harness.',
    )
  }
}

export async function handleValidateWorkflow(
  state: Pick<WorkflowsPageControllerActionsState, 'draftHarness' | 'validationIssues'>,
) {
  try {
    const payload = await validateHarness(state.draftHarness)
    state.validationIssues = payload.issues
    toastStore[payload.valid ? 'success' : 'warning'](
      payload.valid ? 'Harness is valid.' : 'Harness has validation issues.',
    )
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to validate harness.',
    )
  }
}

export function handleCreateWorkflow(
  state: Pick<WorkflowsPageControllerActionsState, 'statuses' | 'agentOptions' | 'templateDraft'>,
) {
  if (state.statuses.length === 0 || state.agentOptions.length === 0) return false
  state.templateDraft = null
  return true
}

export function handleUseWorkflowTemplate(
  state: Pick<WorkflowsPageControllerActionsState, 'statuses' | 'agentOptions' | 'templateDraft'>,
  role: BuiltinRole,
) {
  if (state.statuses.length === 0 || state.agentOptions.length === 0) {
    toastStore.error('Configure statuses and agents before creating a workflow.')
    return false
  }
  state.templateDraft = buildWorkflowTemplateDraft(role)
  return true
}

export async function handleToggleWorkflowSkill(
  state: Pick<
    WorkflowsPageControllerActionsState,
    | 'selectedId'
    | 'isDirty'
    | 'harness'
    | 'draftHarness'
    | 'loadedHarnessWorkflowId'
    | 'skillStates'
    | 'workflows'
  >,
  skill: SkillState,
) {
  if (!state.selectedId) return
  if (state.isDirty) {
    toastStore.warning('Please save your harness changes before binding or unbinding skills.')
    return
  }
  try {
    const result = skill.bound
      ? await unbindWorkflowSkills(state.selectedId, [skill.name])
      : await bindWorkflowSkills(state.selectedId, [skill.name])
    const refreshed = await refreshSelectedWorkflowHarness(state)
    if (!refreshed) return
    state.workflows = applyWorkflowVersionRefresh(state.workflows, state.selectedId, {
      harnessPath: result.harness.path,
      version: result.harness.version,
      history: refreshed.history,
    })
    toastStore.success(skill.bound ? `Unbound ${skill.name}.` : `Bound ${skill.name}.`)
  } catch (caughtError) {
    toastStore.error(
      caughtError instanceof ApiError ? caughtError.detail : 'Failed to update workflow skills.',
    )
  }
}

export function applyAssistantDraft(
  state: Pick<WorkflowsPageControllerActionsState, 'draftHarness' | 'validationIssues'>,
  content: string,
) {
  state.draftHarness = content
  state.validationIssues = []
  toastStore.info('Applied AI suggestion to the harness draft.')
}

export function selectWorkflow(
  state: Pick<WorkflowsPageControllerActionsState, 'selectedId' | 'isDirty'>,
  nextSelectedId: string,
) {
  if (
    state.isDirty &&
    !confirm('You have unsaved changes. Are you sure you want to switch workflows?')
  ) {
    return
  }
  state.selectedId = nextSelectedId
}

export function handleWorkflowCreated(
  state: Pick<WorkflowsPageControllerActionsState, 'workflows' | 'selectedId'>,
  input: { workflow: WorkflowSummary; selectedId: string },
) {
  state.workflows = [...state.workflows, input.workflow]
  state.selectedId = input.selectedId
  toastStore.success('Workflow created.')
}
