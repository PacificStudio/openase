<script lang="ts">
  import { Badge } from '$lib/components/ui/badge'
  import TicketCard from './TicketCard.svelte'
  import type { Ticket, TicketStatus } from '$lib/features/workspace/types'

  let {
    status,
    tickets = [],
    dragTargetStatusId = '',
    workflowName,
    ticketDetailHref,
    isTicketMutationPending,
    onDragStart,
    onDragOver,
    onDrop,
  }: {
    status: TicketStatus
    tickets?: Ticket[]
    dragTargetStatusId?: string
    workflowName: (workflowID?: string | null) => string
    ticketDetailHref: (ticketID: string) => string
    isTicketMutationPending: (ticketID: string) => boolean
    onDragStart?: (event: DragEvent, ticket: Ticket) => void
    onDragOver?: (event: DragEvent, statusID: string) => void
    onDrop?: (event: DragEvent, statusID: string) => void
  } = $props()
</script>

<div
  role="region"
  aria-label={`${status.name} column`}
  class={`min-h-[18rem] rounded-[1.75rem] border p-4 transition ${
    dragTargetStatusId === status.id
      ? 'border-amber-500/45 bg-amber-500/10'
      : 'border-border/70 bg-background/60'
  }`}
  ondragover={(event) => onDragOver?.(event, status.id)}
  ondrop={(event) => onDrop?.(event, status.id)}
>
  <div class="flex items-center justify-between gap-3">
    <div>
      <p class="text-sm font-semibold">{status.name}</p>
      <p class="text-muted-foreground mt-1 text-xs leading-5">
        {status.description || 'No description for this lane yet.'}
      </p>
    </div>
    <Badge variant="outline">{tickets.length}</Badge>
  </div>

  <div class="mt-4 grid gap-3">
    {#if tickets.length === 0}
      <div
        class="text-muted-foreground border-border/70 bg-background/70 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        Drop a ticket here.
      </div>
    {:else}
      {#each tickets as ticket}
        <TicketCard
          {ticket}
          workflowName={workflowName(ticket.workflow_id)}
          href={ticketDetailHref(ticket.id)}
          pending={isTicketMutationPending(ticket.id)}
          {onDragStart}
        />
      {/each}
    {/if}
  </div>
</div>
