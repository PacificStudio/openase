import { ApiError } from '$lib/api/client'
import {
  getHRAdvisor,
  getSystemDashboard,
  listActivity,
  listAgents,
  listTickets,
  updateProject,
} from '$lib/api/openase'
import {
  isProjectDashboardRefreshEvent,
  readProjectDashboardRefreshSections,
  subscribeProjectEvents,
  type ProjectDashboardRefreshSection,
} from '$lib/features/project-events'
import {
  markProjectOnboardingCompleted,
  readProjectOnboardingCompletion,
} from '$lib/features/onboarding'
import { createProjectUpdatesController } from '$lib/features/project-updates'
import { appStore } from '$lib/stores/app.svelte'
import { toastStore } from '$lib/stores/toast.svelte'
import {
  buildActivityItems,
  buildDashboardStats,
  buildExceptionItems,
  shouldShowProjectOnboarding,
} from '../model'
import { createOrgDashboardControllerApi } from './org-dashboard-controller-api'
import { loadOrganizationDashboardSummary } from '../organization-summary'
import type { DashboardStats, HRAdvisorSnapshot, MemorySnapshot, ProjectStatus } from '../types'

const systemDashboardRefreshIntervalMs = 10_000
const emptyDashboardStats: DashboardStats = {
  runningAgents: 0,
  activeTickets: 0,
  totalTickets: 0,
  pendingApprovals: 0,
  ticketSpendToday: 0,
  ticketSpendTotal: 0,
  ticketsCreatedToday: 0,
  ticketsCompletedToday: 0,
  ticketInputTokens: 0,
  ticketOutputTokens: 0,
  agentLifetimeTokens: 0,
  avgCycleMinutes: 0,
  prMergeRate: 0,
}

type DashboardSection = ProjectDashboardRefreshSection | 'memory'

function mergeDashboardSections(
  current: DashboardSection[],
  incoming: Iterable<DashboardSection>,
): DashboardSection[] {
  const merged = [...current]
  for (const section of incoming) {
    if (!merged.includes(section)) merged.push(section)
  }
  return merged
}

const toAdvisorSnapshot = (
  payload: Awaited<ReturnType<typeof getHRAdvisor>> | null,
): HRAdvisorSnapshot | null =>
  payload
    ? {
        summary: payload.summary,
        staffing: payload.staffing,
        recommendations: payload.recommendations,
      }
    : null

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
  $effect(() => {
    const projectId = appStore.currentProject?.id
    const orgId = appStore.currentOrg?.id
    if (!projectId) {
      activities = []
      exceptions = []
      hrAdvisor = null
      memory = null
      stats = emptyDashboardStats
      error = ''
      loading = false
      return
    }

    let cancelled = false
    let hasLoaded = false
    let inFlight = false
    let pendingShowLoading = false
    let queuedSections: DashboardSection[] = []
    let cachedAgents = [] as Awaited<ReturnType<typeof listAgents>>['agents']
    let cachedTickets = [] as Awaited<ReturnType<typeof listTickets>>['tickets']

    const queueLoad = (sections: Iterable<DashboardSection>, showLoading = false) => {
      queuedSections = mergeDashboardSections(queuedSections, sections)
      pendingShowLoading = pendingShowLoading || showLoading
      if (!inFlight) void flushLoads()
    }

    const flushLoads = async () => {
      if (inFlight) return

      inFlight = true
      while (!cancelled && queuedSections.length > 0) {
        const sections = queuedSections
        queuedSections = []
        const showLoading = pendingShowLoading
        pendingShowLoading = false
        if (showLoading) loading = true

        try {
          const [
            agentPayload,
            ticketPayload,
            activityPayload,
            systemPayload,
            hrAdvisorPayload,
            organizationSummary,
          ] = await Promise.all([
            sections.includes('agents') ? listAgents(projectId) : Promise.resolve(null),
            sections.includes('tickets') ? listTickets(projectId) : Promise.resolve(null),
            sections.includes('activity')
              ? listActivity(projectId, { limit: 24 })
              : Promise.resolve(null),
            sections.includes('memory') ? getSystemDashboard() : Promise.resolve(null),
            sections.includes('hr_advisor')
              ? getHRAdvisor(projectId).catch(() => null)
              : Promise.resolve(null),
            sections.includes('organization_summary') && orgId
              ? loadOrganizationDashboardSummary(orgId).catch(() => null)
              : Promise.resolve(null),
          ])

          if (cancelled) return
          if (agentPayload) cachedAgents = agentPayload.agents
          if (ticketPayload) {
            cachedTickets = ticketPayload.tickets
            if (cachedTickets.length > 0 && !onboardingDismissed && projectId) {
              markProjectOnboardingCompleted(projectId)
              onboardingDismissed = true
            }
          }

          if (
            sections.includes('agents') ||
            sections.includes('tickets') ||
            sections.includes('organization_summary')
          ) {
            stats = buildDashboardStats(cachedAgents, cachedTickets, {
              ticketSpendToday:
                organizationSummary?.projectMetrics[projectId]?.todayCost ?? stats.ticketSpendToday,
            })
          }
          if (systemPayload) memory = systemPayload.memory
          if (sections.includes('hr_advisor')) hrAdvisor = toAdvisorSnapshot(hrAdvisorPayload)
          if (activityPayload) {
            activities = buildActivityItems(activityPayload.events)
            exceptions = buildExceptionItems(activityPayload.events)
          }

          error = ''
          hasLoaded = true
        } catch (caughtError) {
          if (cancelled || hasLoaded) continue
          error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load dashboard.'
        } finally {
          if (showLoading && !cancelled) loading = false
        }
      }

      inFlight = false
    }

    queueLoad(
      ['agents', 'tickets', 'activity', 'memory', 'hr_advisor', 'organization_summary'],
      true,
    )

    const unsubscribeDashboard = subscribeProjectEvents(projectId, (event) => {
      if (!isProjectDashboardRefreshEvent(event)) return
      const sections = readProjectDashboardRefreshSections(event)
      if (sections.length > 0) queueLoad(sections)
    })

    const memoryInterval = window.setInterval(() => {
      queueLoad(['memory'])
    }, systemDashboardRefreshIntervalMs)

    return () => {
      cancelled = true
      unsubscribeDashboard()
      window.clearInterval(memoryInterval)
    }
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
