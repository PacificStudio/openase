import { ApiError } from '$lib/api/client'
import { updateProject } from '$lib/api/openase'
import {
  markProjectOnboardingCompleted,
  readProjectOnboardingCompletion,
} from '$lib/features/onboarding'
import { createProjectUpdatesController } from '$lib/features/project-updates'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import { emptyDashboardStats } from './org-dashboard-controller-helpers'
import { buildActivityItems, buildExceptionItems, shouldShowProjectOnboarding } from '../model'
import { createOrgDashboardControllerApi } from './org-dashboard-controller-api'
import { startOrgDashboardDataLoader } from './org-dashboard-controller-loader'
import type { DashboardStats, HRAdvisorSnapshot, MemorySnapshot, ProjectStatus } from '../types'

export function createOrgDashboardController() {
  let loading = $state(false)
  let error = $state('')
  let stats = $state<DashboardStats>(emptyDashboardStats)
  let exceptions = $state<ReturnType<typeof buildExceptionItems>>([])
  let activities = $state<ReturnType<typeof buildActivityItems>>([])
  let hrAdvisor = $state<HRAdvisorSnapshot | null>(null)
  let memory = $state<MemorySnapshot | null>(null)
  let savingStatus = $state(false)
  let editingInfo = $state(false)
  let editName = $state('')
  let editDescription = $state('')
  let savingInfo = $state(false)
  let onboardingDismissed = $state(false)

  const totalTicketTokens = $derived(stats.ticketInputTokens + stats.ticketOutputTokens)
  const showOnboarding = $derived(
    shouldShowProjectOnboarding({
      dismissed: onboardingDismissed,
      loading,
      stats,
      projectId: appStore.currentProject?.id,
      orgId: appStore.currentOrg?.id,
    }),
  )
  const currentStatus = $derived((appStore.currentProject?.status ?? 'Planned') as ProjectStatus)
  const projectName = $derived(appStore.currentProject?.name ?? 'Untitled Project')
  const projectDescription = $derived(appStore.currentProject?.description ?? '')
  const projectUpdates = createProjectUpdatesController({
    getProjectId: () => appStore.currentProject?.id ?? '',
  })
  $effect(() => {
    const projectId = appStore.currentProject?.id ?? ''
    onboardingDismissed = projectId ? readProjectOnboardingCompletion(projectId) : false
  })
  function startEditInfo() {
    editName = projectName
    editDescription = projectDescription
    editingInfo = true
  }
  function cancelEditInfo() {
    editingInfo = false
  }
  async function saveInfo() {
    const projectId = appStore.currentProject?.id
    if (!projectId || savingInfo) return
    const name = editName.trim()
    if (!name) {
      toastStore.error('Project name is required.')
      return
    }

    savingInfo = true
    try {
      const payload = await updateProject(projectId, {
        name,
        description: editDescription.trim() || null,
      })
      appStore.currentProject = payload.project
      editingInfo = false
      toastStore.success('Project info updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update project info.',
      )
    } finally {
      savingInfo = false
    }
  }

  async function handleProjectStatusChange(status: ProjectStatus) {
    const projectId = appStore.currentProject?.id
    if (!projectId || savingStatus) return
    savingStatus = true
    try {
      const payload = await updateProject(projectId, { status })
      appStore.currentProject = payload.project
      toastStore.success('Project status updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update project status.',
      )
    } finally {
      savingStatus = false
    }
  }

  function dismissOnboarding(projectId: string) {
    markProjectOnboardingCompleted(projectId)
    onboardingDismissed = true
  }

  function resetDashboardState() {
    activities = []
    exceptions = []
    hrAdvisor = null
    memory = null
    stats = emptyDashboardStats
    error = ''
    loading = false
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId) {
      resetDashboardState()
      return
    }

    return startOrgDashboardDataLoader({
      projectId,
      orgId: orgId ?? '',
      getOnboardingDismissed: () => onboardingDismissed,
      setOnboardingDismissed: (value) => (onboardingDismissed = value),
      getStats: () => stats,
      setStats: (value) => (stats = value),
      setActivities: (value) => (activities = value),
      setExceptions: (value) => (exceptions = value),
      setHrAdvisor: (value) => (hrAdvisor = value),
      setMemory: (value) => (memory = value),
      setError: (value) => (error = value),
      setLoading: (value) => (loading = value),
    })
  })

  return createOrgDashboardControllerApi({
    getLoading: () => loading,
    getError: () => error,
    getStats: () => stats,
    getExceptions: () => exceptions,
    getActivities: () => activities,
    getHrAdvisor: () => hrAdvisor,
    getMemory: () => memory,
    getSavingStatus: () => savingStatus,
    getEditingInfo: () => editingInfo,
    getEditName: () => editName,
    setEditName: (value) => (editName = value),
    getEditDescription: () => editDescription,
    setEditDescription: (value) => (editDescription = value),
    getSavingInfo: () => savingInfo,
    getTotalTicketTokens: () => totalTicketTokens,
    getShowOnboarding: () => showOnboarding,
    getCurrentStatus: () => currentStatus,
    getProjectName: () => projectName,
    getProjectDescription: () => projectDescription,
    getProjectUpdates: () => projectUpdates,
    startEditInfo,
    cancelEditInfo,
    saveInfo,
    handleProjectStatusChange,
    dismissOnboarding,
  })
}
