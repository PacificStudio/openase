<script lang="ts">
  import { Card, CardContent } from '$lib/components/ui/card'
  import BoardToolbar from './BoardToolbar.svelte'
  import BoardColumn from './BoardColumn.svelte'
  import type { Ticket, TicketStatus } from '$lib/features/workspace'
  import type { StreamConnectionState } from '$lib/api/sse'

  let {
    projectName = '',
    statuses = [],
    ticketsForStatus,
    workflowName,
    ticketDetailHref,
    isTicketMutationPending,
    dragTargetStatusId = '',
    busy = false,
    error = '',
    streamState = 'idle',
    onDragStart,
    onDragOver,
    onDrop,
  }: {
    projectName?: string
    statuses?: TicketStatus[]
    ticketsForStatus: (statusID: string) => Ticket[]
    workflowName: (workflowID?: string | null) => string
    ticketDetailHref: (ticketID: string) => string
    isTicketMutationPending: (ticketID: string) => boolean
    dragTargetStatusId?: string
    busy?: boolean
    error?: string
    streamState?: StreamConnectionState
    onDragStart?: (event: DragEvent, ticket: Ticket) => void
    onDragOver?: (event: DragEvent, statusID: string) => void
    onDrop?: (event: DragEvent, statusID: string) => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 overflow-hidden backdrop-blur">
  <BoardToolbar
    {projectName}
    laneCount={statuses.length}
    ticketCount={statuses.reduce((count, status) => count + ticketsForStatus(status.id).length, 0)}
    {busy}
    {error}
    {streamState}
  />

  <CardContent class="p-4 sm:p-6">
    {#if statuses.length === 0}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-[1.75rem] border border-dashed px-4 py-8 text-sm"
      >
        Select a project to load its board columns.
      </div>
    {:else}
      <div class="grid gap-4 xl:grid-cols-3 2xl:grid-cols-4">
        {#each statuses as status}
          <BoardColumn
            {status}
            tickets={ticketsForStatus(status.id)}
            {dragTargetStatusId}
            {workflowName}
            {ticketDetailHref}
            {isTicketMutationPending}
            {onDragStart}
            {onDragOver}
            {onDrop}
          />
        {/each}
      </div>
    {/if}
  </CardContent>
</Card>
