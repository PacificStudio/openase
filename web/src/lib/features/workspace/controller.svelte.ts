import { createBoardStore } from '$lib/features/board'
import { createDashboardStore } from '$lib/features/dashboard'
import { api, toErrorMessage } from './api'
import {
  buildWorkspaceOnboarding,
  chainCleanups,
  clearProjectState,
  clearWorkflowState,
  resetSelectedWorkflowState,
} from './controller-helpers'
import { createEntityMutationActions } from './entity-mutations'
import {
  defaultWorkflowForm,
  orderTicketStatuses,
  slugify,
  toOrganizationForm,
  toProjectForm,
  toWorkflowForm,
} from './mappers'
import { createWorkspaceState } from './state.svelte'
import { createWorkflowEditorActions } from './workflow-editor'
import type {
  BuiltinRolePayload,
  Organization,
  OrganizationPayload,
  Project,
  ProjectPayload,
  SkillListPayload,
  StatusPayload,
  Workflow,
  WorkflowDetailPayload,
  WorkflowListPayload,
} from './types'

export type WorkspaceStartOptions = {
  preferredOrgId?: string
  preferredProjectId?: string
  preferredWorkflowId?: string
}

export function createWorkspaceController() {
  const dashboard = createDashboardStore()
  const board = createBoardStore((projectId) => void dashboard.loadHRAdvisor(projectId))
  const state = createWorkspaceState()
  let heartbeatTimer: number | null = null
  let streamCleanup: (() => void) | null = null
  let started = false
  const entityActions = createEntityMutationActions({ state, loadOrganizations, loadProjects })
  const workflowEditor = createWorkflowEditorActions({
    state,
    getStatuses: () => board.statuses,
    loadWorkflowContext,
    loadWorkflowDetail,
  })
  const {
    destroy: destroyWorkflowEditor,
    selectedBuiltinRole: readSelectedBuiltinRole,
    ...workflowActions
  } = workflowEditor

  function selectedBuiltinRole() {
    return readSelectedBuiltinRole()
  }

  function onboardingSummary() {
    return buildWorkspaceOnboarding(state, board, dashboard)
  }

  async function start(options: WorkspaceStartOptions = {}) {
    if (!started) {
      heartbeatTimer = window.setInterval(() => dashboard.tickHeartbeat(), 15000)
      started = true
    }

    await bootstrap(options)
  }

  function destroy() {
    started = false
    if (heartbeatTimer) {
      window.clearInterval(heartbeatTimer)
    }

    heartbeatTimer = null
    destroyWorkflowEditor()
    disconnectProjectStreams()
  }

  function toggleDrawer(force?: boolean) {
    state.drawerOpen = force ?? !state.drawerOpen
  }

  async function bootstrap(options: WorkspaceStartOptions) {
    state.booting = true
    state.errorMessage = ''
    try {
      await Promise.all([
        loadBuiltinRoles(),
        loadOrganizations(
          options.preferredOrgId,
          options.preferredProjectId,
          options.preferredWorkflowId,
        ),
      ])
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    } finally {
      state.booting = false
    }
  }

  async function loadBuiltinRoles() {
    const payload = await api<BuiltinRolePayload>('/api/v1/roles/builtin')
    state.builtinRoles = payload.roles
  }

  async function loadOrganizations(
    preferredOrgId?: string,
    preferredProjectId?: string,
    preferredWorkflowId?: string,
  ) {
    const payload = await api<OrganizationPayload>('/api/v1/orgs')
    state.organizations = payload.organizations
    const nextOrg =
      state.organizations.find((item) => item.id === preferredOrgId) ??
      state.organizations.find((item) => item.id === state.selectedOrgId) ??
      state.organizations[0] ??
      null

    if (!nextOrg) {
      state.selectedOrgId = ''
      state.selectedOrg = null
      state.editOrgForm = { name: '', slug: '' }
      state.projects = []
      clearProjectState(state, board, dashboard, disconnectProjectStreams)
      return
    }

    state.selectedOrgId = nextOrg.id
    state.selectedOrg = nextOrg
    state.editOrgForm = toOrganizationForm(nextOrg)
    await loadProjects(
      nextOrg.id,
      nextOrg.id === preferredOrgId ? preferredProjectId : undefined,
      preferredWorkflowId,
    )
  }

  async function loadProjects(
    orgId: string,
    preferredProjectId?: string,
    preferredWorkflowId?: string,
  ) {
    const payload = await api<ProjectPayload>(`/api/v1/orgs/${orgId}/projects`)
    state.projects = payload.projects
    const nextProject =
      state.projects.find((item) => item.id === preferredProjectId) ??
      state.projects.find((item) => item.id === state.selectedProjectId) ??
      state.projects[0] ??
      null

    if (!nextProject) {
      clearProjectState(state, board, dashboard, disconnectProjectStreams)
      return
    }

    state.selectedProjectId = nextProject.id
    state.selectedProject = nextProject
    state.editProjectForm = toProjectForm(nextProject)
    await loadWorkflowContext(
      nextProject.id,
      nextProject.id === preferredProjectId ? preferredWorkflowId : undefined,
    )
  }

  async function loadWorkflowContext(projectId: string, preferredWorkflowId?: string) {
    const [statusPayload, workflowPayload, skillPayload] = await Promise.all([
      api<StatusPayload>(`/api/v1/projects/${projectId}/statuses`),
      api<WorkflowListPayload>(`/api/v1/projects/${projectId}/workflows`),
      api<SkillListPayload>(`/api/v1/projects/${projectId}/skills`),
    ])
    board.setProject(projectId)
    board.setStatuses(orderTicketStatuses(statusPayload.statuses))
    state.workflows = workflowPayload.workflows
    state.skills = skillPayload.skills
    state.createWorkflowForm = defaultWorkflowForm(board.statuses)
    await activateProject(projectId)

    const nextWorkflow =
      state.workflows.find((item) => item.id === preferredWorkflowId) ??
      state.workflows.find((item) => item.id === state.selectedWorkflowId) ??
      state.workflows[0] ??
      null

    if (!nextWorkflow) {
      resetSelectedWorkflowState(state, board)
      return
    }

    await loadWorkflowDetail(nextWorkflow.id)
  }

  async function activateProject(projectId: string) {
    disconnectProjectStreams()
    dashboard.setProject(projectId)
    await Promise.all([
      board.load(projectId),
      dashboard.loadAgents(projectId),
      dashboard.loadActivityEvents(projectId),
      dashboard.loadHRAdvisor(projectId),
    ])
    streamCleanup = chainCleanups(board.connect(projectId), dashboard.connect(projectId))
  }

  async function loadWorkflowDetail(workflowId: string) {
    const payload = await api<WorkflowDetailPayload>(`/api/v1/workflows/${workflowId}`)
    state.selectedWorkflow = payload.workflow
    state.selectedWorkflowId = payload.workflow.id
    state.editWorkflowForm = toWorkflowForm(payload.workflow)
    state.harnessIssues = []
    state.validationBusy = false
    workflowActions.setHarnessDraft(payload.workflow.harness_content ?? '', false)
    state.harnessPath = payload.workflow.harness_path
    state.harnessVersion = payload.workflow.version
  }

  async function selectOrganization(org: Organization) {
    if (org.id === state.selectedOrgId) return
    state.errorMessage = ''
    state.selectedOrgId = org.id
    state.selectedOrg = org
    state.editOrgForm = toOrganizationForm(org)
    clearProjectState(state, board, dashboard, disconnectProjectStreams)
    try {
      await loadProjects(org.id)
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    }
  }

  async function selectProject(project: Project) {
    if (project.id === state.selectedProjectId) return
    state.selectedProjectId = project.id
    state.selectedProject = project
    state.editProjectForm = toProjectForm(project)
    state.errorMessage = ''
    clearWorkflowState(state, board)
    try {
      await loadWorkflowContext(project.id)
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    }
  }

  async function selectWorkflow(workflow: Workflow) {
    if (workflow.id === state.selectedWorkflowId) return
    state.errorMessage = ''
    try {
      await loadWorkflowDetail(workflow.id)
    } catch (error) {
      state.errorMessage = toErrorMessage(error)
    }
  }

  return {
    state,
    board,
    dashboard,
    start,
    destroy,
    toggleDrawer,
    onboardingSummary,
    selectedBuiltinRole,
    ...entityActions,
    ...workflowActions,
    selectOrganization,
    selectProject,
    selectWorkflow,
    slugify,
  }

  function disconnectProjectStreams() {
    streamCleanup?.()
    streamCleanup = null
  }
}
