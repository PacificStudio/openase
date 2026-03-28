<script lang="ts">
  import { PageScaffold } from '$lib/components/layout'
  import { formatBytes, formatCount } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import {
    getHRAdvisor,
    getProject,
    getSystemDashboard,
    listActivity,
    listAgents,
    listTickets,
    updateProject,
  } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import StatCard from './stat-card.svelte'
  import ProjectHealthList from './project-health-list.svelte'
  import ExceptionPanel from './exception-panel.svelte'
  import ActivityFeedPanel from './activity-feed-panel.svelte'
  import CostSnapshotPanel from './cost-snapshot-panel.svelte'
  import HRAdvisorPanel from './hr-advisor-panel.svelte'
  import MemorySnapshotPanel from './memory-snapshot-panel.svelte'
  import { Bot, Coins, Ticket } from '@lucide/svelte'
  import {
    buildActivityItems,
    buildDashboardStats,
    buildExceptionItems,
    buildProjectSummary,
    findTopCostTicket,
    findTopTokenAgent,
  } from '../model'
  import type {
    DashboardStats,
    ProjectStatus,
    DashboardUsageLeader,
    HRAdvisorSnapshot,
    MemorySnapshot,
    ProjectSummary,
  } from '../types'

  const dashboardPollIntervalMs = 5000

  let loading = $state(false)
  let error = $state('')
  let stats = $state<DashboardStats>({
    runningAgents: 0,
    activeTickets: 0,
    pendingApprovals: 0,
    newTicketsTodayCost: 0,
    projectCost: 0,
    ticketsCreatedToday: 0,
    ticketsCompletedToday: 0,
    ticketInputTokens: 0,
    ticketOutputTokens: 0,
    totalAgentTokens: 0,
    avgCycleMinutes: 0,
    prMergeRate: 0,
  })
  let projects = $state<ProjectSummary[]>([])
  let exceptions = $state<ReturnType<typeof buildExceptionItems>>([])
  let activities = $state<ReturnType<typeof buildActivityItems>>([])
  let hrAdvisor = $state<HRAdvisorSnapshot | null>(null)
  let memory = $state<MemorySnapshot | null>(null)
  let topCostTicket = $state<DashboardUsageLeader | null>(null)
  let topTokenAgent = $state<DashboardUsageLeader | null>(null)
  let savingProjectStatusId = $state<string | null>(null)
  const totalTicketTokens = $derived(stats.ticketInputTokens + stats.ticketOutputTokens)

  async function handleProjectStatusChange(projectId: string, status: ProjectStatus) {
    if (savingProjectStatusId === projectId) return

    savingProjectStatusId = projectId

    try {
      const payload = await updateProject(projectId, { status })
      appStore.currentProject = payload.project
      projects = projects.map((project) =>
        project.id === projectId
          ? {
              ...project,
              description: payload.project.description,
              status: payload.project.status,
            }
          : project,
      )
      toastStore.success('Project status updated.')
    } catch (caughtError) {
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : 'Failed to update project status.',
      )
    } finally {
      savingProjectStatusId = null
    }
  }

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      projects = []
      activities = []
      exceptions = []
      hrAdvisor = null
      memory = null
      topCostTicket = null
      topTokenAgent = null
      return
    }

    let cancelled = false
    let hasLoaded = false
    let inFlight = false

    const load = async (showLoading: boolean) => {
      if (inFlight) return

      inFlight = true
      if (showLoading) {
        loading = true
      }

      try {
        const [
          projectPayload,
          agentPayload,
          ticketPayload,
          activityPayload,
          systemPayload,
          hrAdvisorPayload,
        ] = await Promise.all([
          getProject(projectId),
          listAgents(projectId),
          listTickets(projectId),
          listActivity(projectId, { limit: 24 }),
          getSystemDashboard(),
          getHRAdvisor(projectId).catch(() => null),
        ])

        if (cancelled) return

        stats = buildDashboardStats(agentPayload.agents, ticketPayload.tickets)
        topCostTicket = findTopCostTicket(ticketPayload.tickets)
        topTokenAgent = findTopTokenAgent(agentPayload.agents)
        memory = systemPayload.memory
        hrAdvisor = hrAdvisorPayload
          ? {
              summary: hrAdvisorPayload.summary,
              staffing: hrAdvisorPayload.staffing,
              recommendations: hrAdvisorPayload.recommendations,
            }
          : null

        projects = buildProjectSummary(
          projectPayload.project,
          stats,
          activityPayload.events[0]?.created_at ?? null,
        )
        activities = buildActivityItems(activityPayload.events)
        exceptions = buildExceptionItems(activityPayload.events)

        error = ''
        hasLoaded = true
      } catch (caughtError) {
        if (cancelled) return
        if (!hasLoaded) {
          error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load dashboard.'
        }
      } finally {
        inFlight = false
        if (showLoading && !cancelled) {
          loading = false
        }
      }
    }

    void load(true)

    const interval = window.setInterval(() => {
      void load(false)
    }, dashboardPollIntervalMs)

    return () => {
      cancelled = true
      window.clearInterval(interval)
    }
  })
</script>

<PageScaffold title="Dashboard" description="Project overview">
  <div class="space-y-6">
    {#if loading}
      <div
        class="border-border bg-card text-muted-foreground rounded-md border px-4 py-10 text-center text-sm"
      >
        Loading dashboard…
      </div>
    {:else if error}
      <div
        class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
      >
        {error}
      </div>
    {:else}
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard label="Running Agents" value={stats.runningAgents} icon={Bot} />
        <StatCard label="Active Tickets" value={stats.activeTickets} icon={Ticket} />
        <StatCard label="Ticket Tokens" value={formatCount(totalTicketTokens)} icon={Coins} />
        <StatCard label="Heap In Use" value={memory ? formatBytes(memory.heap_inuse_bytes) : '—'} />
      </div>

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <ProjectHealthList
          {projects}
          {savingProjectStatusId}
          onUpdateStatus={handleProjectStatusChange}
        />
        <CostSnapshotPanel
          newTicketsTodayCost={stats.newTicketsTodayCost}
          projectCost={stats.projectCost}
          ticketInputTokens={stats.ticketInputTokens}
          ticketOutputTokens={stats.ticketOutputTokens}
          totalAgentTokens={stats.totalAgentTokens}
          ticketsCreatedToday={stats.ticketsCreatedToday}
          ticketsCompletedToday={stats.ticketsCompletedToday}
          {topCostTicket}
          {topTokenAgent}
        />
        <ExceptionPanel {exceptions} />
      </div>

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <ActivityFeedPanel {activities} class="lg:col-span-2" />
        <MemorySnapshotPanel {memory} />
      </div>

      {#if hrAdvisor && appStore.currentProject}
        {#key appStore.currentProject.id}
          <HRAdvisorPanel projectId={appStore.currentProject.id} advisor={hrAdvisor} />
        {/key}
      {/if}
    {/if}
  </div>
</PageScaffold>
