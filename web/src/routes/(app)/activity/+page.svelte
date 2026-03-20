<script lang="ts">
  import MasterDetailLayout from '$lib/components/layout/MasterDetailLayout.svelte'
  import {
    ActivityFeedPanel,
    ExceptionsPanel,
    RunningNowPanel,
  } from '$lib/features/dashboard'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()
</script>

<svelte:head>
  <title>Activity · OpenASE</title>
</svelte:head>

<MasterDetailLayout detailWidthClass="xl:grid-cols-[minmax(0,1fr)_22rem]">
  {#snippet main()}
    <ActivityFeedPanel
      activityEvents={workspace.dashboard.activityEvents}
      activityStreamState={workspace.dashboard.activityStreamState}
      selectedAgentName={workspace.dashboard.selectedAgentName()}
    />
  {/snippet}

  {#snippet detail()}
    <RunningNowPanel
      agents={workspace.dashboard.agents}
      busy={workspace.dashboard.busy}
      error={workspace.dashboard.error}
      selectedAgentId={workspace.dashboard.selectedAgentId}
      heartbeatNow={workspace.dashboard.heartbeatNow}
      agentStreamState={workspace.dashboard.agentStreamState}
      onSelectAgent={(agentId) => void workspace.dashboard.selectAgent(agentId)}
    />

    <ExceptionsPanel
      boardError={workspace.board.error}
      dashboardError={workspace.dashboard.error}
      hrAdvisorError={workspace.dashboard.hrAdvisorError}
      stalledAgentCount={workspace.dashboard.stalledCount()}
      pendingMutationCount={workspace.board.tickets.filter((ticket) =>
        workspace.board.isTicketMutationPending(ticket.id),
      ).length}
    />
  {/snippet}
</MasterDetailLayout>
