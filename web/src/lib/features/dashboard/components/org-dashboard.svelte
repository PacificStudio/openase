<script lang="ts">
  import { formatBytes } from '$lib/utils'
  import { appStore } from '$lib/stores/app.svelte'
  import {
    getProject,
    getSystemDashboard,
    listActivity,
    listAgents,
    listTickets,
  } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import StatCard from './stat-card.svelte'
  import ProjectHealthList from './project-health-list.svelte'
  import ExceptionPanel from './exception-panel.svelte'
  import ActivityFeedPanel from './activity-feed-panel.svelte'
  import MemorySnapshotPanel from './memory-snapshot-panel.svelte'
  import { Bot, Ticket, ShieldCheck } from '@lucide/svelte'
  import type {
    ActivityItem,
    DashboardStats,
    ExceptionItem,
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
    todayCost: 0,
    weekCost: 0,
    ticketsCreatedToday: 0,
    ticketsCompletedToday: 0,
    avgCycleMinutes: 0,
    prMergeRate: 0,
  })
  let projects = $state<ProjectSummary[]>([])
  let exceptions = $state<ExceptionItem[]>([])
  let activities = $state<ActivityItem[]>([])
  let memory = $state<MemorySnapshot | null>(null)

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      projects = []
      activities = []
      exceptions = []
      memory = null
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
        const [projectPayload, agentPayload, ticketPayload, activityPayload, systemPayload] =
          await Promise.all([
            getProject(projectId),
            listAgents(projectId),
            listTickets(projectId),
            listActivity(projectId, { limit: 24 }),
            getSystemDashboard(),
          ])

        if (cancelled) return

        const activeTickets = ticketPayload.tickets.filter(
          (ticket) => !isTerminalStatus(ticket.status_name),
        )
        const runningAgents = agentPayload.agents.filter(
          (agent) => agent.status === 'running',
        ).length
        const totalCost = ticketPayload.tickets.reduce((sum, ticket) => sum + ticket.cost_amount, 0)
        const todayStart = new Date()
        todayStart.setHours(0, 0, 0, 0)

        stats = {
          runningAgents,
          activeTickets: activeTickets.length,
          pendingApprovals: 0,
          todayCost: ticketPayload.tickets
            .filter((ticket) => new Date(ticket.created_at) >= todayStart)
            .reduce((sum, ticket) => sum + ticket.cost_amount, 0),
          weekCost: totalCost,
          ticketsCreatedToday: ticketPayload.tickets.filter(
            (ticket) => new Date(ticket.created_at) >= todayStart,
          ).length,
          ticketsCompletedToday: ticketPayload.tickets.filter(
            (ticket) =>
              isTerminalStatus(ticket.status_name) && new Date(ticket.created_at) >= todayStart,
          ).length,
          avgCycleMinutes: 0,
          prMergeRate: 0,
        }
        memory = systemPayload.memory

        projects = [
          {
            id: projectPayload.project.id,
            name: projectPayload.project.name,
            health: projectHealth(projectPayload.project.status),
            activeAgents: runningAgents,
            activeTickets: activeTickets.length,
            lastActivity: activityPayload.events[0]?.created_at ?? new Date().toISOString(),
          },
        ]

        activities = activityPayload.events.slice(0, 6).map((event) => ({
          id: event.id,
          type: event.event_type,
          message: event.message,
          timestamp: event.created_at,
          ticketIdentifier: undefined,
          agentName: agentNameFromMetadata(event.metadata),
        }))

        exceptions = activityPayload.events
          .filter((event) => isExceptionEvent(event.event_type))
          .slice(0, 4)
          .map((event) => ({
            id: event.id,
            type: normalizeExceptionType(event.event_type),
            message: event.message,
            timestamp: event.created_at,
          }))

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

  function isTerminalStatus(statusName: string) {
    const value = statusName.toLowerCase()
    return value === 'done' || value === 'cancelled' || value === 'archived'
  }

  function projectHealth(status: string): ProjectSummary['health'] {
    const value = status.toLowerCase()
    if (value === 'healthy' || value === 'active') return 'healthy'
    if (value === 'blocked' || value === 'archived') return 'blocked'
    return 'warning'
  }

  function isExceptionEvent(eventType: string) {
    return ['hook_failed', 'budget_alert', 'agent_stalled', 'retry_paused'].includes(eventType)
  }

  function normalizeExceptionType(eventType: string): ExceptionItem['type'] {
    if (
      eventType === 'hook_failed' ||
      eventType === 'budget_alert' ||
      eventType === 'agent_stalled'
    ) {
      return eventType
    }

    return 'retry_paused'
  }

  function agentNameFromMetadata(metadata: Record<string, unknown>) {
    const value = metadata.agent_name
    return typeof value === 'string' ? value : undefined
  }
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-foreground text-lg font-semibold">Dashboard</h1>
    <p class="text-muted-foreground text-sm">Project overview</p>
  </div>

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
      <StatCard label="Pending Approvals" value={stats.pendingApprovals} icon={ShieldCheck} />
      <StatCard label="Heap In Use" value={memory ? formatBytes(memory.heap_inuse_bytes) : '—'} />
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <ProjectHealthList {projects} class="lg:col-span-2" />
      <ExceptionPanel {exceptions} />
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <ActivityFeedPanel {activities} class="lg:col-span-2" />
      <MemorySnapshotPanel {memory} />
    </div>
  {/if}
</div>
