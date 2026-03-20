<script lang="ts">
  import { ArrowUpRight, Link2 } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import { ticketPriorityBadgeClass } from '$lib/features/workspace'
  import type { Ticket } from '$lib/features/workspace'

  let {
    ticket,
    workflowName,
    href,
    pending = false,
    onDragStart,
  }: {
    ticket: Ticket
    workflowName: string
    href: string
    pending?: boolean
    onDragStart?: (event: DragEvent, ticket: Ticket) => void
  } = $props()
</script>

<a
  {href}
  class={`block rounded-3xl border px-4 py-4 transition hover:-translate-y-0.5 ${
    pending
      ? 'border-amber-500/30 bg-amber-500/10 shadow-sm'
      : 'border-border/70 bg-background/80 hover:border-foreground/15 hover:bg-background'
  }`}
  draggable="true"
  ondragstart={(event) => onDragStart?.(event, ticket)}
>
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <div class="flex flex-wrap items-center gap-2">
        <p class="text-sm font-semibold">{ticket.identifier}</p>
        <Badge class={ticketPriorityBadgeClass(ticket.priority)}>{ticket.priority}</Badge>
      </div>
      <p class="mt-2 text-sm leading-6">{ticket.title}</p>
    </div>
    <ArrowUpRight class="text-muted-foreground size-4 shrink-0" />
  </div>

  <div class="text-muted-foreground mt-3 flex flex-wrap items-center gap-2 text-xs">
    <Badge variant="outline">{workflowName}</Badge>
    {#if ticket.external_ref}
      <Badge variant="outline">
        <Link2 class="mr-1 size-3" />
        {ticket.external_ref}
      </Badge>
    {/if}
    <Badge variant="outline">{ticket.type}</Badge>
  </div>

  <p class="text-muted-foreground mt-3 text-xs leading-5">
    Attempts {ticket.attempt_count} · Errors {ticket.consecutive_errors} · Cost ${ticket.cost_amount.toFixed(
      2,
    )}
  </p>
</a>
