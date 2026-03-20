<script lang="ts">
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

  const now = new Date()
  function ago(minutes: number): string {
    return new Date(now.getTime() - minutes * 60_000).toISOString()
  }

  const stats: DashboardStats = {
    runningAgents: 7,
    activeTickets: 23,
    pendingApprovals: 3,
    todayCost: 42.87,
    weekCost: 187.54,
    ticketsCreatedToday: 12,
    ticketsCompletedToday: 9,
    avgCycleMinutes: 34,
    prMergeRate: 0.82,
  }

  const projects: ProjectSummary[] = [
    {
      id: 'p1',
      name: 'openase-core',
      health: 'healthy',
      activeAgents: 3,
      activeTickets: 8,
      lastActivity: ago(2),
    },
    {
      id: 'p2',
      name: 'openase-web',
      health: 'healthy',
      activeAgents: 2,
      activeTickets: 6,
      lastActivity: ago(5),
    },
    {
      id: 'p3',
      name: 'billing-service',
      health: 'warning',
      activeAgents: 1,
      activeTickets: 4,
      lastActivity: ago(18),
    },
    {
      id: 'p4',
      name: 'auth-gateway',
      health: 'blocked',
      activeAgents: 0,
      activeTickets: 3,
      lastActivity: ago(45),
    },
    {
      id: 'p5',
      name: 'data-pipeline',
      health: 'healthy',
      activeAgents: 1,
      activeTickets: 2,
      lastActivity: ago(12),
    },
  ]

  const exceptions: ExceptionItem[] = [
    {
      id: 'e1',
      type: 'hook_failed',
      message: 'Pre-commit hook failed: lint errors in auth-gateway',
      ticketIdentifier: 'AUTH-142',
      timestamp: ago(8),
    },
    {
      id: 'e2',
      type: 'budget_alert',
      message: 'billing-service approaching daily budget limit (85%)',
      timestamp: ago(22),
    },
    {
      id: 'e3',
      type: 'agent_stalled',
      message: 'Agent claude-dev-3 unresponsive for 15 minutes',
      ticketIdentifier: 'CORE-287',
      timestamp: ago(15),
    },
    {
      id: 'e4',
      type: 'retry_paused',
      message: 'Retry paused after 3 failures on CORE-301',
      ticketIdentifier: 'CORE-301',
      timestamp: ago(35),
    },
  ]

  const activities: ActivityItem[] = [
    {
      id: 'a1',
      type: 'pr_merged',
      message: 'PR #487 merged: Add user settings page',
      ticketIdentifier: 'WEB-102',
      agentName: 'claude-dev-1',
      timestamp: ago(3),
    },
    {
      id: 'a2',
      type: 'agent.launching',
      message: 'Agent launching Codex session for database migration',
      ticketIdentifier: 'CORE-290',
      agentName: 'claude-dev-2',
      timestamp: ago(7),
    },
    {
      id: 'a3',
      type: 'agent.ready',
      message: 'Codex session is ready and heartbeating for API endpoint refactor',
      ticketIdentifier: 'CORE-285',
      agentName: 'claude-dev-1',
      timestamp: ago(14),
    },
    {
      id: 'a4',
      type: 'pr_opened',
      message: 'PR #488 opened: Fix billing calculation edge case',
      ticketIdentifier: 'BILL-67',
      agentName: 'claude-dev-4',
      timestamp: ago(20),
    },
    {
      id: 'a5',
      type: 'comment',
      message: 'Review comment added on PR #485',
      ticketIdentifier: 'WEB-99',
      timestamp: ago(28),
    },
    {
      id: 'a6',
      type: 'agent_assigned',
      message: 'Agent assigned to implement OAuth flow',
      ticketIdentifier: 'AUTH-145',
      agentName: 'claude-dev-5',
      timestamp: ago(32),
    },
  ]
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-lg font-semibold text-foreground">Dashboard</h1>
    <p class="text-sm text-muted-foreground">Organization overview</p>
  </div>

  <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
    <StatCard
      label="Running Agents"
      value={stats.runningAgents}
      icon={Bot}
      trend={{ value: 12, positive: true }}
    />
    <StatCard
      label="Active Tickets"
      value={stats.activeTickets}
      icon={Ticket}
      trend={{ value: 8, positive: true }}
    />
    <StatCard
      label="Pending Approvals"
      value={stats.pendingApprovals}
      icon={ShieldCheck}
    />
    <StatCard
      label="Today's Cost"
      value={'$' + stats.todayCost.toFixed(2)}
      icon={DollarSign}
      trend={{ value: 5, positive: false }}
    />
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
      topProject={{ name: 'openase-core', cost: 72.30 }}
      topAgent={{ name: 'claude-dev-1', cost: 48.15 }}
    />
  </div>
</div>
