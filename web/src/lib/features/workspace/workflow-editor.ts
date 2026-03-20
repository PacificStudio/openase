import { api, toErrorMessage } from './api'
import { createHarnessActions } from './harness-actions'
import { defaultWorkflowForm, workflowHasSkill } from './mappers'
import type {
  BuiltinRole,
  HarnessDocument,
  Skill,
  SkillListPayload,
  TicketStatus,
  Workflow,
  WorkflowDetailPayload,
} from './types'

type WorkflowState = {
  workflowBusy: boolean
  harnessBusy: boolean
  validationBusy: boolean
  skillBusy: boolean
  pendingSkillName: string
  errorMessage: string
  notice: string
  createWorkflowForm: {
    name: string
    type: Workflow['type']
    pickupStatusId: string
    finishStatusId: string
    maxConcurrent: number
    maxRetryAttempts: number
    timeoutMinutes: number
    stallTimeoutMinutes: number
    isActive: boolean
  }
  editWorkflowForm: {
    name: string
    type: Workflow['type']
    pickupStatusId: string
    finishStatusId: string
    maxConcurrent: number
    maxRetryAttempts: number
    timeoutMinutes: number
    stallTimeoutMinutes: number
    isActive: boolean
  }
  selectedProjectId: string
  selectedBuiltinRoleSlug: string
  selectedWorkflowId: string
  selectedWorkflow: Workflow | null
  workflows: Workflow[]
  skills: Skill[]
  builtinRoles: BuiltinRole[]
  harnessDraft: string
  harnessPath: string
  harnessVersion: number
  harnessIssues: Array<{
    level: 'error' | 'warning' | string
    message: string
    line?: number
    column?: number
  }>
}

type Dependencies = {
  state: WorkflowState
  getStatuses: () => TicketStatus[]
  loadWorkflowContext: (projectId: string, preferredWorkflowId?: string) => Promise<void>
  loadWorkflowDetail: (workflowId: string) => Promise<void>
}

export function createWorkflowEditorActions({
  state,
  getStatuses,
  loadWorkflowContext,
  loadWorkflowDetail,
}: Dependencies) {
  const harnessActions = createHarnessActions(state)

  async function createWorkflow() {
    await createWorkflowFromRoleTemplate(
      selectedBuiltinRole(),
      state.createWorkflowForm.name,
      state.createWorkflowForm.type,
    )
  }

  async function createWorkflowFromRoleTemplate(
    role: BuiltinRole | null,
    workflowName: string,
    workflowType: Workflow['type'],
  ) {
    const selectedProjectId = state.selectedProjectId
    if (!selectedProjectId) {
      return
    }

    await runWorkflowMutation(async () => {
      const payload = await api<{ workflow: Workflow }>(
        `/api/v1/projects/${selectedProjectId}/workflows`,
        {
          method: 'POST',
          body: JSON.stringify({
            name: workflowName,
            type: workflowType,
            harness_path: role?.harness_path,
            harness_content: role?.content ?? '',
            pickup_status_id: state.createWorkflowForm.pickupStatusId,
            finish_status_id: state.createWorkflowForm.finishStatusId || null,
            max_concurrent: state.createWorkflowForm.maxConcurrent,
            max_retry_attempts: state.createWorkflowForm.maxRetryAttempts,
            timeout_minutes: state.createWorkflowForm.timeoutMinutes,
            stall_timeout_minutes: state.createWorkflowForm.stallTimeoutMinutes,
            is_active: state.createWorkflowForm.isActive,
          }),
        },
      )
      state.notice = `Workflow ${payload.workflow.name} created`
      await loadWorkflowContext(selectedProjectId, payload.workflow.id)
      state.createWorkflowForm = defaultWorkflowForm(getStatuses())
      state.selectedBuiltinRoleSlug = role?.slug ?? ''
    })
  }

  async function updateWorkflow() {
    const selectedWorkflow = state.selectedWorkflow
    if (!selectedWorkflow) {
      return
    }

    await runWorkflowMutation(async () => {
      const payload = await api<WorkflowDetailPayload>(`/api/v1/workflows/${selectedWorkflow.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          name: state.editWorkflowForm.name,
          type: state.editWorkflowForm.type,
          pickup_status_id: state.editWorkflowForm.pickupStatusId,
          finish_status_id: state.editWorkflowForm.finishStatusId || null,
          max_concurrent: state.editWorkflowForm.maxConcurrent,
          max_retry_attempts: state.editWorkflowForm.maxRetryAttempts,
          timeout_minutes: state.editWorkflowForm.timeoutMinutes,
          stall_timeout_minutes: state.editWorkflowForm.stallTimeoutMinutes,
          is_active: state.editWorkflowForm.isActive,
        }),
      })
      state.selectedWorkflow = {
        ...payload.workflow,
        harness_content: state.selectedWorkflow?.harness_content,
      }
      state.workflows = state.workflows.map((item) =>
        item.id === payload.workflow.id ? payload.workflow : item,
      )
      state.notice = `Workflow ${payload.workflow.name} updated`
    })
  }

  async function deleteWorkflow() {
    const selectedWorkflow = state.selectedWorkflow
    const selectedProjectId = state.selectedProjectId
    if (!selectedWorkflow || !selectedProjectId) {
      return
    }

    await runWorkflowMutation(async () => {
      await api(`/api/v1/workflows/${selectedWorkflow.id}`, { method: 'DELETE' })
      state.notice = `Workflow ${selectedWorkflow.name} deleted`
      await loadWorkflowContext(selectedProjectId)
    })
  }

  async function toggleSkillBinding(skill: Skill) {
    const selectedWorkflow = state.selectedWorkflow
    if (!selectedWorkflow || !state.selectedProjectId) {
      return
    }
    if (harnessActions.harnessDirty()) {
      state.errorMessage = 'Save or discard harness edits before changing workflow skill bindings.'
      return
    }

    state.skillBusy = true
    state.pendingSkillName = skill.name
    state.errorMessage = ''
    state.notice = ''
    try {
      const endpoint = workflowHasSkill(skill, selectedWorkflow.id) ? 'unbind' : 'bind'
      await api<{ harness: HarnessDocument }>(
        `/api/v1/workflows/${selectedWorkflow.id}/skills/${endpoint}`,
        {
          method: 'POST',
          body: JSON.stringify({ skills: [skill.name] }),
        },
      )
      await Promise.all([loadWorkflowDetail(selectedWorkflow.id), reloadSkills()])
      state.notice = `${endpoint === 'bind' ? 'Bound' : 'Unbound'} ${skill.name} ${endpoint === 'bind' ? 'to' : 'from'} ${selectedWorkflow.name}`
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    } finally {
      state.skillBusy = false
      state.pendingSkillName = ''
    }
  }

  function selectedBuiltinRole() {
    return state.builtinRoles.find((role) => role.slug === state.selectedBuiltinRoleSlug) ?? null
  }

  function selectBuiltinRole(role: BuiltinRole) {
    state.selectedBuiltinRoleSlug = role.slug
    state.createWorkflowForm = {
      ...state.createWorkflowForm,
      name: role.name,
      type: role.workflow_type,
    }
  }

  function clearBuiltinRoleSelection() {
    state.selectedBuiltinRoleSlug = ''
  }

  function loadRecommendedRole(recommendation: {
    role_slug: string
    suggested_workflow_name: string
  }) {
    const role = state.builtinRoles.find((item) => item.slug === recommendation.role_slug)
    if (!role) {
      state.errorMessage = `Role template ${recommendation.role_slug} is unavailable.`
      return
    }

    state.errorMessage = ''
    state.selectedBuiltinRoleSlug = role.slug
    state.createWorkflowForm = {
      ...defaultWorkflowForm(getStatuses()),
      name: recommendation.suggested_workflow_name || role.name,
      type: role.workflow_type,
    }
    state.notice = `${role.name} template loaded into workflow creation.`
  }

  function destroy() {
    harnessActions.destroy()
  }

  return {
    createWorkflow,
    createWorkflowFromRoleTemplate,
    updateWorkflow,
    deleteWorkflow,
    toggleSkillBinding,
    ...harnessActions,
    selectedBuiltinRole,
    selectBuiltinRole,
    clearBuiltinRoleSelection,
    loadRecommendedRole,
    destroy,
  }

  async function reloadSkills() {
    if (!state.selectedProjectId) {
      return
    }

    const payload = await api<SkillListPayload>(
      `/api/v1/projects/${state.selectedProjectId}/skills`,
    )
    state.skills = payload.skills
  }

  async function runWorkflowMutation(mutation: () => Promise<void>) {
    state.workflowBusy = true
    state.errorMessage = ''
    state.notice = ''
    try {
      await mutation()
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    } finally {
      state.workflowBusy = false
    }
  }
}
