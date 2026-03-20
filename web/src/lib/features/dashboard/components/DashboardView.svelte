<script lang="ts">
  import ProjectHealthPanel from './ProjectHealthPanel.svelte'
  import RunningNowPanel from './RunningNowPanel.svelte'
  import ExceptionsPanel from './ExceptionsPanel.svelte'
  import ActivityFeedPanel from './ActivityFeedPanel.svelte'
  import type {
    OnboardingSummary,
    Project,
    Agent,
    HRAdvisorPayload,
    ActivityEvent,
  } from '$lib/features/workspace'
  import type { StreamConnectionState } from '$lib/api/sse'

  let {
    project = null,
    workflowCount = 0,
    statusCount = 0,
    ticketCount = 0,
    onboardingSummary,
    hrAdvisor = null,
    hrAdvisorBusy = false,
    hrAdvisorError = '',
    agents = [],
    selectedAgentId = '',
    heartbeatNow,
    dashboardBusy = false,
    dashboardError = '',
    agentStreamState = 'idle',
    activityEvents = [],
    activityStreamState = 'idle',
    selectedAgentName = 'All agents',
    stalledAgentCount = 0,
    pendingMutationCount = 0,
    boardError = '',
    onSelectAgent,
  }: {
    project?: Project | null
    workflowCount?: number
    statusCount?: number
    ticketCount?: number
    onboardingSummary: OnboardingSummary
    hrAdvisor?: HRAdvisorPayload | null
    hrAdvisorBusy?: boolean
    hrAdvisorError?: string
    agents?: Agent[]
    selectedAgentId?: string
    heartbeatNow: number
    dashboardBusy?: boolean
    dashboardError?: string
    agentStreamState?: StreamConnectionState
    activityEvents?: ActivityEvent[]
    activityStreamState?: StreamConnectionState
    selectedAgentName?: string
    stalledAgentCount?: number
    pendingMutationCount?: number
    boardError?: string
    onSelectAgent?: (agentId: string) => void
  } = $props()
</script>

<div class="grid gap-4 xl:grid-cols-[minmax(0,1.08fr)_minmax(0,0.92fr)]">
  <ProjectHealthPanel
    {project}
    {workflowCount}
    {statusCount}
    {ticketCount}
    {onboardingSummary}
    {hrAdvisor}
    {hrAdvisorBusy}
    {hrAdvisorError}
  />

    <div class="grid gap-4">
    <RunningNowPanel
      {agents}
      busy={dashboardBusy}
      error={dashboardError}
      {selectedAgentId}
      {heartbeatNow}
      {agentStreamState}
      {onSelectAgent}
    />
    <ExceptionsPanel
      {boardError}
      {dashboardError}
      {hrAdvisorError}
      {stalledAgentCount}
      {pendingMutationCount}
    />
  </div>

      <div class="xl:col-span-2">
    <ActivityFeedPanel {activityEvents} {activityStreamState} {selectedAgentName} />
  </div>
</div>
