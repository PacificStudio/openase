import type { createBoardStore } from '$lib/features/board'
import type { createDashboardStore } from '$lib/features/dashboard'
import { buildOnboardingSummary, defaultProjectForm, defaultWorkflowForm } from './mappers'
import type { createWorkspaceState } from './state.svelte'

type WorkspaceState = ReturnType<typeof createWorkspaceState>
type BoardStore = ReturnType<typeof createBoardStore>
type DashboardStore = ReturnType<typeof createDashboardStore>

export function buildWorkspaceOnboarding(
  state: WorkspaceState,
  board: BoardStore,
  dashboard: DashboardStore,
) {
  return buildOnboardingSummary({
    organizationCount: state.organizations.length,
    projectCount: state.projects.length,
    selectedOrgName: state.selectedOrg?.name ?? '',
    selectedProjectName: state.selectedProject?.name ?? '',
    statusCount: board.statuses.length,
    workflowCount: state.workflows.length,
    ticketCount: board.tickets.length,
    agentCount: dashboard.agents.length,
    runningAgentCount: dashboard.runningAgentCount(),
    activityCount: dashboard.activityEvents.length,
    hasAutomationSignal: dashboard.hasSignal(board.tickets),
  })
}

export function resetSelectedWorkflowState(state: WorkspaceState, board: BoardStore) {
  state.selectedWorkflowId = ''
  state.selectedWorkflow = null
  state.editWorkflowForm = defaultWorkflowForm(board.statuses)
  state.harnessDraft = ''
  state.harnessPath = ''
  state.harnessVersion = 0
  state.validationBusy = false
  state.harnessIssues = []
}

export function clearWorkflowState(state: WorkspaceState, board: BoardStore) {
  state.workflows = []
  state.skills = []
  state.createWorkflowForm = defaultWorkflowForm()
  state.selectedBuiltinRoleSlug = ''
  resetSelectedWorkflowState(state, board)
}

export function clearProjectState(
  state: WorkspaceState,
  board: BoardStore,
  dashboard: DashboardStore,
  disconnectProjectStreams: () => void,
) {
  state.selectedProjectId = ''
  state.selectedProject = null
  state.editProjectForm = defaultProjectForm()
  board.reset()
  dashboard.reset()
  clearWorkflowState(state, board)
  disconnectProjectStreams()
}

export function chainCleanups(...cleanups: Array<() => void>) {
  return () => {
    cleanups.forEach((cleanup) => cleanup())
  }
}
