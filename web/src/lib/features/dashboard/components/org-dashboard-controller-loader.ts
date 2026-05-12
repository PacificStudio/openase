import { ApiError } from '$lib/api/client'
import {
  getHRAdvisor,
  getSystemDashboard,
  listActivity,
  listAgents,
  listTickets,
} from '$lib/api/openase'
import {
  createProjectReconnectRecoveryTask,
  isProjectDashboardRefreshEvent,
  readProjectDashboardRefreshSections,
  subscribeProjectEvents,
} from '$lib/features/project-events'
import { markProjectOnboardingCompleted } from '$lib/features/onboarding'
import {
  type DashboardSection,
  mergeDashboardSections,
  systemDashboardRefreshIntervalMs,
  toAdvisorSnapshot,
} from './org-dashboard-controller-helpers'
import { buildActivityItems, buildDashboardStats, buildExceptionItems } from '../model'
import { loadOrganizationDashboardSummary } from '../organization-summary'
import type { DashboardStats, HRAdvisorSnapshot, MemorySnapshot } from '../types'

export function startOrgDashboardDataLoader(input: {
  projectId: string
  orgId: string
  getOnboardingDismissed: () => boolean
  setOnboardingDismissed: (value: boolean) => void
  getStats: () => DashboardStats
  setStats: (value: DashboardStats) => void
  setActivities: (value: ReturnType<typeof buildActivityItems>) => void
  setExceptions: (value: ReturnType<typeof buildExceptionItems>) => void
  setHrAdvisor: (value: HRAdvisorSnapshot | null) => void
  setMemory: (value: MemorySnapshot | null) => void
  setError: (value: string) => void
  setLoading: (value: boolean) => void
}) {
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
      if (showLoading) input.setLoading(true)

      try {
        const [
          agentPayload,
          ticketPayload,
          activityPayload,
          systemPayload,
          hrAdvisorPayload,
          organizationSummary,
        ] = await Promise.all([
          sections.includes('agents') ? listAgents(input.projectId) : Promise.resolve(null),
          sections.includes('tickets') ? listTickets(input.projectId) : Promise.resolve(null),
          sections.includes('activity')
            ? listActivity(input.projectId, { limit: 24 })
            : Promise.resolve(null),
          sections.includes('memory') ? getSystemDashboard() : Promise.resolve(null),
          sections.includes('hr_advisor')
            ? getHRAdvisor(input.projectId).catch(() => null)
            : Promise.resolve(null),
          sections.includes('organization_summary') && input.orgId
            ? loadOrganizationDashboardSummary(input.orgId).catch(() => null)
            : Promise.resolve(null),
        ])

        if (cancelled) return
        if (agentPayload) cachedAgents = agentPayload.agents
        if (ticketPayload) {
          cachedTickets = ticketPayload.tickets
          if (cachedTickets.length > 0 && !input.getOnboardingDismissed()) {
            markProjectOnboardingCompleted(input.projectId)
            input.setOnboardingDismissed(true)
          }
        }

        if (
          sections.includes('agents') ||
          sections.includes('tickets') ||
          sections.includes('organization_summary')
        ) {
          input.setStats(
            buildDashboardStats(cachedAgents, cachedTickets, {
              ticketSpendToday:
                organizationSummary?.projectMetrics[input.projectId]?.todayCost ??
                input.getStats().ticketSpendToday,
            }),
          )
        }
        if (systemPayload) input.setMemory(systemPayload.memory)
        if (sections.includes('hr_advisor')) input.setHrAdvisor(toAdvisorSnapshot(hrAdvisorPayload))
        if (activityPayload) {
          input.setActivities(buildActivityItems(activityPayload.events))
          input.setExceptions(buildExceptionItems(activityPayload.events))
        }

        input.setError('')
        hasLoaded = true
      } catch (caughtError) {
        if (cancelled || hasLoaded) continue
        input.setError(
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load dashboard.',
        )
      } finally {
        if (showLoading && !cancelled) input.setLoading(false)
      }
    }

    inFlight = false
  }

  queueLoad(['agents', 'tickets', 'activity', 'memory', 'hr_advisor', 'organization_summary'], true)

  const unsubscribeDashboard = subscribeProjectEvents(
    input.projectId,
    (event) => {
      if (!isProjectDashboardRefreshEvent(event)) return
      const sections = readProjectDashboardRefreshSections(event)
      if (sections.length > 0) queueLoad(sections)
    },
    {
      onReconnectRecovery: createProjectReconnectRecoveryTask(() => {
        queueLoad(['agents', 'tickets', 'activity', 'memory', 'hr_advisor', 'organization_summary'])
      }),
    },
  )

  const memoryInterval = window.setInterval(() => {
    queueLoad(['memory'])
  }, systemDashboardRefreshIntervalMs)

  return () => {
    cancelled = true
    unsubscribeDashboard()
    window.clearInterval(memoryInterval)
  }
}
