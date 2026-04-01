<script lang="ts">
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
    loadingRunId = null,
    savingFields = false,
    creatingComment = false,
    updatingCommentId = null,
    deletingCommentId = null,
    resumingRetry = false,
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
    loadingRunId?: string | null
    savingFields?: boolean
    creatingComment?: boolean
    updatingCommentId?: string | null
    deletingCommentId?: string | null
    resumingRetry?: boolean
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

  let activeView = $state('runs')
  let previousTicketId = ''

  $effect(() => {
    if (ticket.id === previousTicketId) {
      return
    }

    previousTicketId = ticket.id
    activeView = 'runs'
  })
</script>

<Tabs.Root bind:value={activeView} class="flex flex-1 flex-col overflow-hidden">
  <div class="border-border border-b px-5 pt-2">
    <Tabs.List>
      <Tabs.Trigger value="runs">Runs</Tabs.Trigger>
      <Tabs.Trigger value="discussion">Discussion</Tabs.Trigger>
    </Tabs.List>
  </div>

  <Tabs.Content value="runs" class="flex min-h-0 flex-1 overflow-hidden">
    <TicketRunHistoryPanel
      {ticket}
      {runs}
      {currentRun}
      blocks={runBlocks}
      {loadingRunId}
      {resumingRetry}
      {onSelectRun}
      {onResumeRetry}
    />
  </Tabs.Content>

  <Tabs.Content value="discussion" class="flex min-h-0 flex-1 overflow-hidden">
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
