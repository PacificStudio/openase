<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { listActivity, listAgents, listTickets, getProject } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import StatCard from './stat-card.svelte'
  import ProjectHealthList from './project-health-list.svelte'
  import ExceptionPanel from './exception-panel.svelte'
  import ActivityFeedPanel from './activity-feed-panel.svelte'
  import CostSnapshotPanel from './cost-snapshot-panel.svelte'
  import { Bot, Ticket, ShieldCheck, DollarSign } from '@lucide/svelte'
  import type {
    ProjectSummary,
    DashboardStats,
    ExceptionItem,
    ActivityItem,
  } from '../types'
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

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      projects = []
      activities = []
      exceptions = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [projectPayload, agentPayload, ticketPayload, activityPayload] = await Promise.all([
          getProject(projectId),
          listAgents(projectId),
          listTickets(projectId),
          listActivity(projectId, { limit: '24' }),
        ])

        if (cancelled) return

        const activeTickets = ticketPayload.tickets.filter((ticket) => !isTerminalStatus(ticket.status_name))
        const runningAgents = agentPayload.agents.filter((agent) => agent.status === 'running').length
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
          ticketsCreatedToday: ticketPayload.tickets.filter((ticket) => new Date(ticket.created_at) >= todayStart).length,
          ticketsCompletedToday: ticketPayload.tickets.filter(
            (ticket) => isTerminalStatus(ticket.status_name) && new Date(ticket.created_at) >= todayStart,
          ).length,
          avgCycleMinutes: 0,
          prMergeRate: 0,
        }

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
      } catch (caughtError) {
        if (cancelled) return
        error =
          caughtError instanceof ApiError ? caughtError.detail : 'Failed to load dashboard.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    return () => {
      cancelled = true
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
    if (eventType === 'hook_failed' || eventType === 'budget_alert' || eventType === 'agent_stalled') {
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
    <h1 class="text-lg font-semibold text-foreground">Dashboard</h1>
    <p class="text-sm text-muted-foreground">Project overview</p>
  </div>

  {#if loading}
    <div class="rounded-md border border-border bg-card px-4 py-10 text-center text-sm text-muted-foreground">
      Loading dashboard…
    </div>
  {:else if error}
    <div class="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
      {error}
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard label="Running Agents" value={stats.runningAgents} icon={Bot} />
      <StatCard label="Active Tickets" value={stats.activeTickets} icon={Ticket} />
      <StatCard label="Pending Approvals" value={stats.pendingApprovals} icon={ShieldCheck} />
      <StatCard label="Today's Cost" value={'$' + stats.todayCost.toFixed(2)} icon={DollarSign} />
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <ProjectHealthList projects={projects} class="lg:col-span-2" />
      <ExceptionPanel exceptions={exceptions} />
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
      <ActivityFeedPanel activities={activities} class="lg:col-span-2" />
      <CostSnapshotPanel
        todayCost={stats.todayCost}
        weekCost={stats.weekCost}
        topProject={{ name: appStore.currentProject?.name ?? 'Current project', cost: stats.weekCost }}
        topAgent={{ name: 'Contract not exposed', cost: 0 }}
      />
    </div>
  {/if}
</div>
