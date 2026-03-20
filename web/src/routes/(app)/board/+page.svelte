<script lang="ts">
  import { LoaderCircle } from '@lucide/svelte'
  import BoardView from '$lib/features/board/components/BoardView.svelte'
  import { getWorkspaceContext } from '$lib/features/workspace'

  const workspace = getWorkspaceContext()

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

    const params = new URLSearchParams()
    if (workspace.state.selectedOrgId) {
      params.set('org', workspace.state.selectedOrgId)
    }
    params.set('project', workspace.state.selectedProjectId)
    params.set('id', ticketID)
    return `/ticket?${params.toString()}`
  }
</script>

<svelte:head>
  <title>Board · OpenASE</title>
</svelte:head>

{#if workspace.state.booting}
  <div
    class="border-border/80 bg-background/70 flex min-h-96 items-center justify-center rounded-[1.75rem] border"
  >
    <div class="text-muted-foreground flex items-center gap-3 text-sm">
      <LoaderCircle class="size-4 animate-spin" />
      <span>Loading board…</span>
    </div>
  </div>
{:else}
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
