import type { DashboardStats, HRAdvisorSnapshot, MemorySnapshot, ProjectStatus } from '../types'

type OrgDashboardControllerApiInput = {
  getLoading: () => boolean
  getError: () => string
  getStats: () => DashboardStats
  getExceptions: () => ReturnType<typeof import('../model').buildExceptionItems>
  getActivities: () => ReturnType<typeof import('../model').buildActivityItems>
  getHrAdvisor: () => HRAdvisorSnapshot | null
  getMemory: () => MemorySnapshot | null
  getSavingStatus: () => boolean
  getEditingInfo: () => boolean
  getEditName: () => string
  setEditName: (value: string) => void
  getEditDescription: () => string
  setEditDescription: (value: string) => void
  getSavingInfo: () => boolean
  getTotalTicketTokens: () => number
  getShowOnboarding: () => boolean
  getCurrentStatus: () => ProjectStatus
  getProjectName: () => string
  getProjectDescription: () => string
  getProjectUpdates: () => ReturnType<
    typeof import('$lib/features/project-updates').createProjectUpdatesController
  >
  startEditInfo: () => void
  cancelEditInfo: () => void
  saveInfo: () => Promise<void>
  handleProjectStatusChange: (status: ProjectStatus) => Promise<void>
  dismissOnboarding: (projectId: string) => void
}

export function createOrgDashboardControllerApi(input: OrgDashboardControllerApiInput) {
  return {
    get loading() {
      return input.getLoading()
    },
    get error() {
      return input.getError()
    },
    get stats() {
      return input.getStats()
    },
    get exceptions() {
      return input.getExceptions()
    },
    get activities() {
      return input.getActivities()
    },
    get hrAdvisor() {
      return input.getHrAdvisor()
    },
    get memory() {
      return input.getMemory()
    },
    get savingStatus() {
      return input.getSavingStatus()
    },
    get editingInfo() {
      return input.getEditingInfo()
    },
    get editName() {
      return input.getEditName()
    },
    get editDescription() {
      return input.getEditDescription()
    },
    get savingInfo() {
      return input.getSavingInfo()
    },
    get totalTicketTokens() {
      return input.getTotalTicketTokens()
    },
    get showOnboarding() {
      return input.getShowOnboarding()
    },
    get currentStatus() {
      return input.getCurrentStatus()
    },
    get projectName() {
      return input.getProjectName()
    },
    get projectDescription() {
      return input.getProjectDescription()
    },
    get projectUpdates() {
      return input.getProjectUpdates()
    },
    setEditName: input.setEditName,
    setEditDescription: input.setEditDescription,
    startEditInfo: input.startEditInfo,
    cancelEditInfo: input.cancelEditInfo,
    saveInfo: input.saveInfo,
    handleProjectStatusChange: input.handleProjectStatusChange,
    dismissOnboarding: input.dismissOnboarding,
  }
}
