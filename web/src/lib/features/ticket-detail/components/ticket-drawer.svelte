<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { statusSync } from '$lib/features/statuses/public'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { createTicketDrawerState } from '../drawer-state.svelte'
  import { connectTicketDetailStreams } from '../streams'
  import { createTicketDrawerActions } from '../ticket-drawer-actions'
  import TicketDrawerContent from './ticket-drawer-content.svelte'
  import TicketDrawerLoading from './ticket-drawer-loading.svelte'
  import type { TicketDetail } from '../types'

  let {
    open = $bindable(false),
    projectId,
    ticketId,
    onOpenChange,
  }: {
    open?: boolean
    projectId?: string | null
    ticketId?: string | null
    onOpenChange?: (open: boolean) => void
  } = $props()

  const drawerState = createTicketDrawerState()
  const drawerActions = createTicketDrawerActions({
    getProjectId: () => projectId,
    getTicketId: () => ticketId,
    drawerState,
    buildDrawerMutation,
  })
  let lastStatusSyncProjectId: string | null = null
  let lastStatusSyncVersion = -1

  function buildDrawerMutation(ticket: TicketDetail) {
    return {
      ticket,
      projectId,
      ticketId,
      load: drawerState.load,
      applyTicket: (nextTicket: TicketDetail) => {
        drawerState.ticket = nextTicket
      },
      clearMessages: drawerState.clearMutationMessages,
      setError: drawerState.setMutationError,
      setNotice: drawerState.setMutationNotice,
    }
  }

  $effect(() => {
    onOpenChange?.(open)
  })

  $effect(() => {
    const currentProjectId = projectId
    const statusVersion = statusSync.version
    if (!currentProjectId) {
      lastStatusSyncProjectId = null
      lastStatusSyncVersion = -1
      return
    }
    if (currentProjectId !== lastStatusSyncProjectId || statusVersion !== lastStatusSyncVersion) {
      lastStatusSyncProjectId = currentProjectId
      lastStatusSyncVersion = statusVersion
      drawerState.invalidateReferences(currentProjectId)
    }
  })

  $effect(() => {
    const currentProjectId = projectId
    const currentTicketId = ticketId
    if (!open || !currentProjectId || !currentTicketId) {
      if (!open) drawerState.reset()
      return
    }

    void drawerState.load(currentProjectId, currentTicketId)
  })

  $effect(() => {
    if (!open || !projectId || !ticketId) {
      return
    }

    let runStreamNeedsRecovery = false

    return connectTicketDetailStreams(projectId, ticketId, {
      onRelevantEvent: () => {
        void drawerState.refreshTimeline(projectId, ticketId)
      },
      onReferenceEvent: () => {
        drawerState.invalidateReferences(projectId)
        void drawerState.refreshReferences(projectId, ticketId)
      },
      onRunFrame: (frame) => {
        drawerState.applyRunStreamFrame(frame)
      },
      onRunStateChange: (state) => {
        drawerState.setRunStreamState(state)
        if (state === 'retrying') {
          runStreamNeedsRecovery = true
          return
        }
        if (state === 'live' && runStreamNeedsRecovery) {
          runStreamNeedsRecovery = false
          void drawerState.recoverRunTranscript(projectId, ticketId)
        }
      },
    })
  })
</script>

<Sheet bind:open>
  <SheetContent
    side="right"
    class="flex w-full flex-col gap-0 p-0 sm:max-w-[80vw]"
    showCloseButton={false}
  >
    <SheetHeader class="sr-only">
      <SheetTitle>{drawerState.ticket?.identifier ?? 'Ticket detail'}</SheetTitle>
      <SheetDescription>Ticket detail drawer</SheetDescription>
    </SheetHeader>

    {#if drawerState.loading}
      <TicketDrawerLoading />
    {:else if drawerState.error}
      <div
        class="text-destructive flex flex-1 items-center justify-center px-6 text-center text-sm"
      >
        {drawerState.error}
      </div>
    {:else if drawerState.ticket}
      <TicketDrawerContent
        projectId={projectId ?? ''}
        ticket={drawerState.ticket}
        hooks={drawerState.hooks}
        timeline={drawerState.timeline}
        runs={drawerState.runs}
        currentRun={drawerState.currentRun}
        runBlocks={drawerState.runBlocks}
        loadingRunId={drawerState.loadingRunId}
        runStreamState={drawerState.runStreamState}
        recoveringRunTranscript={drawerState.recoveringRunTranscript}
        statuses={drawerState.statuses}
        dependencyCandidates={drawerState.dependencyCandidates}
        repoOptions={drawerState.repoOptions}
        savingFields={drawerState.savingFields}
        creatingDependency={drawerState.creatingDependency}
        deletingDependencyId={drawerState.deletingDependencyId}
        creatingExternalLink={drawerState.creatingExternalLink}
        deletingExternalLinkId={drawerState.deletingExternalLinkId}
        creatingRepoScope={drawerState.creatingRepoScope}
        updatingRepoScopeId={drawerState.updatingRepoScopeId}
        deletingRepoScopeId={drawerState.deletingRepoScopeId}
        creatingComment={drawerState.creatingComment}
        updatingCommentId={drawerState.updatingCommentId}
        deletingCommentId={drawerState.deletingCommentId}
        resumingRetry={drawerState.resumingRetry}
        resettingWorkspace={drawerState.resettingWorkspace}
        onClose={appStore.closeRightPanel}
        archiving={drawerState.archiving}
        onSaveFields={drawerActions.handleSaveFields}
        onPriorityChange={drawerActions.handlePriorityChange}
        onArchive={drawerActions.handleArchive}
        onSelectRun={(runId) =>
          projectId && ticketId ? drawerState.selectRun(projectId, ticketId, runId) : undefined}
        onResumeRetry={drawerActions.handleResumeRetry}
        onResetWorkspace={drawerActions.handleResetWorkspace}
        onAddDependency={drawerActions.handleAddDependency}
        onDeleteDependency={drawerActions.handleDeleteDependency}
        onCreateExternalLink={drawerActions.handleCreateExternalLink}
        onDeleteExternalLink={drawerActions.handleDeleteExternalLink}
        onCreateScope={drawerActions.handleCreateRepoScope}
        onUpdateScope={drawerActions.handleUpdateRepoScope}
        onDeleteScope={drawerActions.handleDeleteRepoScope}
        onCreateComment={drawerActions.handleCreateComment}
        onUpdateComment={drawerActions.handleUpdateComment}
        onDeleteComment={drawerActions.handleDeleteComment}
        onLoadCommentHistory={drawerActions.handleLoadCommentHistory}
      />
    {/if}
  </SheetContent>
</Sheet>
