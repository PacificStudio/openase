<script lang="ts">
  import { LoaderCircle } from '@lucide/svelte'
  import { onDestroy, onMount } from 'svelte'
  import AppShell from '$lib/components/layout/AppShell.svelte'
  import RightDrawer from '$lib/components/layout/RightDrawer.svelte'
  import Sidebar from '$lib/components/layout/Sidebar.svelte'
  import TopBar from '$lib/components/layout/TopBar.svelte'
  import BoardView from '$lib/features/board/components/BoardView.svelte'
  import DashboardView from '$lib/features/dashboard/components/DashboardView.svelte'
  import { createWorkspaceController } from '$lib/features/workspace/controller.svelte'
  import WorkspaceControlDrawer from '$lib/features/workspace/components/WorkspaceControlDrawer.svelte'

  const workspace = createWorkspaceController()

  onMount(() => {
    void workspace.start()
  })

  onDestroy(() => {
    workspace.destroy()
  })

  function workflowName(workflowID?: string | null) {
    if (!workflowID) {
      return 'No workflow'
    }

    return (
      workspace.state.workflows.find((workflow) => workflow.id === workflowID)?.name ??
      'Detached workflow'
    )
  }

  function ticketDetailHref(ticketID: string) {
    if (!workspace.state.selectedProjectId) {
      return '/ticket'
    }

    return `/ticket?project=${encodeURIComponent(workspace.state.selectedProjectId)}&id=${encodeURIComponent(ticketID)}`
  }
</script>

<svelte:head>
  <title>OpenASE Workflow Management</title>
  <meta
    name="description"
    content="Feature-first OpenASE control plane with reusable shell, dashboard panels, and board components."
  />
</svelte:head>

{#snippet header()}
  <TopBar
    selectedOrg={workspace.state.selectedOrg}
    selectedProject={workspace.state.selectedProject}
    notice={workspace.state.notice}
    errorMessage={workspace.state.errorMessage}
    onToggleDrawer={() => workspace.toggleDrawer()}
  />
{/snippet}

{#snippet sidebar()}
  <Sidebar
    organizations={workspace.state.organizations}
    projects={workspace.state.projects}
    selectedOrgId={workspace.state.selectedOrgId}
    selectedProjectId={workspace.state.selectedProjectId}
    workflowCount={workspace.state.workflows.length}
    ticketCount={workspace.board.tickets.length}
    onSelectOrganization={workspace.selectOrganization}
    onSelectProject={workspace.selectProject}
  />
{/snippet}

{#snippet drawer()}
  <RightDrawer>
    <WorkspaceControlDrawer controller={workspace} />
  </RightDrawer>
{/snippet}

<AppShell {header} {sidebar} {drawer} drawerOpen={workspace.state.drawerOpen}>
  {#if workspace.state.booting}
    <div
      class="border-border/80 bg-background/70 flex min-h-96 items-center justify-center rounded-[2rem] border"
    >
      <div class="text-muted-foreground flex items-center gap-3 text-sm">
        <LoaderCircle class="size-4 animate-spin" />
        <span>Loading control plane…</span>
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

    <BoardView
      projectName={workspace.state.selectedProject?.name ?? ''}
      statuses={workspace.board.statuses}
      ticketsForStatus={workspace.board.ticketsForStatus}
      {workflowName}
      {ticketDetailHref}
      isTicketMutationPending={workspace.board.isTicketMutationPending}
      dragTargetStatusId={workspace.board.dragTargetStatusId}
      busy={workspace.board.busy}
      error={workspace.board.error}
      streamState={workspace.board.streamState}
      onDragStart={workspace.board.handleTicketDragStart}
      onDragOver={workspace.board.handleStatusDragOver}
      onDrop={(event, statusID) => void workspace.board.handleStatusDrop(event, statusID)}
    />
  {/if}
</AppShell>
