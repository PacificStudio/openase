<script lang="ts">
  import type { StreamConnectionState } from '$lib/api/sse'
  import * as Tabs from '$ui/tabs'
  import TicketCommentsThread from './ticket-comments-thread.svelte'
  import TicketRunHistoryPanel from './ticket-run-history-panel.svelte'
  import type {
    TicketCommentRevision,
    TicketDetail,
    TicketRun,
    TicketRunTranscriptBlock,
    TicketTimelineItem,
  } from '../types'

  let {
    ticket,
    timeline,
    runs = [],
    currentRun = null,
    runBlocks = [],
    runsLoaded = false,
    loadingRuns = false,
    runsError = '',
    loadingRunId = null,
    runStreamState = 'idle',
    recoveringRunTranscript = false,
    savingFields = false,
    creatingComment = false,
    updatingCommentId = null,
    deletingCommentId = null,
    resumingRetry = false,
    onViewChange,
    onLoadRuns,
    onSaveFields,
    onSelectRun,
    onResumeRetry,
    onCreateComment,
    onUpdateComment,
    onDeleteComment,
    onLoadCommentHistory,
  }: {
    ticket: TicketDetail
    timeline: TicketTimelineItem[]
    runs?: TicketRun[]
    currentRun?: TicketRun | null
    runBlocks?: TicketRunTranscriptBlock[]
    runsLoaded?: boolean
    loadingRuns?: boolean
    runsError?: string
    loadingRunId?: string | null
    runStreamState?: StreamConnectionState
    recoveringRunTranscript?: boolean
    savingFields?: boolean
    creatingComment?: boolean
    updatingCommentId?: string | null
    deletingCommentId?: string | null
    resumingRetry?: boolean
    onViewChange?: (view: 'comments' | 'runs') => void
    onLoadRuns?: () => Promise<void> | void
    onSaveFields?: (draft: { title: string; description: string; statusId: string }) => void
    onSelectRun?: (runId: string) => Promise<void> | void
    onResumeRetry?: () => Promise<void> | void
    onCreateComment?: (body: string) => Promise<boolean> | boolean
    onUpdateComment?: (commentId: string, body: string) => Promise<boolean> | boolean
    onDeleteComment?: (commentId: string) => Promise<boolean> | boolean
    onLoadCommentHistory?: (
      commentId: string,
    ) => Promise<TicketCommentRevision[]> | TicketCommentRevision[]
  } = $props()

  let activeView = $state('discussion')
  let previousTicketId = ''

  $effect(() => {
    if (ticket.id === previousTicketId) {
      return
    }

    previousTicketId = ticket.id
    activeView = 'discussion'
  })

  $effect(() => {
    onViewChange?.(activeView === 'runs' ? 'runs' : 'comments')
  })

  $effect(() => {
    if (activeView !== 'runs' || runsLoaded || loadingRuns) {
      return
    }

    void onLoadRuns?.()
  })
</script>

<Tabs.Root bind:value={activeView} class="flex flex-col gap-0">
  <div class="border-border bg-background sticky top-0 z-10 border-b px-4">
    <Tabs.List variant="line" class="h-7 gap-0">
      <Tabs.Trigger value="discussion" class="px-2.5 py-1 text-xs">Discussion</Tabs.Trigger>
      <Tabs.Trigger value="runs" class="px-2.5 py-1 text-xs"
        >Runs{runs.length > 0 ? ` (${runs.length})` : ''}</Tabs.Trigger
      >
    </Tabs.List>
  </div>

  <Tabs.Content value="runs">
    <TicketRunHistoryPanel
      {ticket}
      {runs}
      {currentRun}
      blocks={runBlocks}
      {loadingRuns}
      {runsError}
      {loadingRunId}
      {runStreamState}
      {recoveringRunTranscript}
      {resumingRetry}
      {onSelectRun}
      onRetryLoadRuns={onLoadRuns}
      {onResumeRetry}
    />
  </Tabs.Content>

  <Tabs.Content value="discussion">
    <TicketCommentsThread
      {ticket}
      {timeline}
      {savingFields}
      {creatingComment}
      {updatingCommentId}
      {deletingCommentId}
      {onSaveFields}
      {onCreateComment}
      {onUpdateComment}
      {onDeleteComment}
      {onLoadCommentHistory}
    />
  </Tabs.Content>
</Tabs.Root>
