<script lang="ts">
  import { LoaderCircle } from '@lucide/svelte'
  import DashboardView from '$lib/features/dashboard/components/DashboardView.svelte'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
</script>

<svelte:head>
  <title>Overview · OpenASE</title>
  <meta
    name="description"
    content="OpenASE project overview with fixed shell, compact app bar, and bounded dashboard panels."
  />
</svelte:head>

{#if workspace.state.booting}
  <div
    class="border-border/80 bg-background/70 flex min-h-96 items-center justify-center rounded-[1.75rem] border"
  >
    <div class="text-muted-foreground flex items-center gap-3 text-sm">
      <LoaderCircle class="size-4 animate-spin" />
      <span>Loading project overview…</span>
    </div>
  </div>
{:else}
  <DashboardView
    project={workspace.state.selectedProject}
    workflowCount={workspace.state.workflows.length}
    statusCount={workspace.board.statuses.length}
    ticketCount={workspace.board.tickets.length}
    onboardingSummary={workspace.onboardingSummary()}
    hrAdvisor={workspace.dashboard.hrAdvisor}
    hrAdvisorBusy={workspace.dashboard.hrAdvisorBusy}
    hrAdvisorError={workspace.dashboard.hrAdvisorError}
    agents={workspace.dashboard.agents}
    selectedAgentId={workspace.dashboard.selectedAgentId}
    heartbeatNow={workspace.dashboard.heartbeatNow}
    dashboardBusy={workspace.dashboard.busy}
    dashboardError={workspace.dashboard.error}
    agentStreamState={workspace.dashboard.agentStreamState}
    activityEvents={workspace.dashboard.activityEvents}
    activityStreamState={workspace.dashboard.activityStreamState}
    selectedAgentName={workspace.dashboard.selectedAgentName()}
    stalledAgentCount={workspace.dashboard.stalledCount()}
    pendingMutationCount={workspace.board.tickets.filter((ticket) =>
      workspace.board.isTicketMutationPending(ticket.id),
    ).length}
    boardError={workspace.board.error}
    onSelectAgent={(agentId) => void workspace.dashboard.selectAgent(agentId)}
  />
{/if}
