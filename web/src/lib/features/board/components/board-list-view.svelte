<script lang="ts">
  import { Badge } from '$ui/badge'
  import { cn, formatRelativeTime } from '$lib/utils'
  import type { BoardColumn, BoardTicket } from '../types'

  let {
    columns,
    class: className = '',
    onticketclick,
  }: {
    columns: BoardColumn[]
    class?: string
    onticketclick?: (ticket: BoardTicket) => void
  } = $props()

  const rows = $derived(
    columns.flatMap((column) =>
      column.tickets.map((ticket) => ({
        ticket,
        statusName: column.name,
        statusColor: column.color,
      })),
    ),
  )

  const priorityColors: Record<BoardTicket['priority'], string> = {
    urgent: 'bg-red-500',
    high: 'bg-orange-500',
    medium: 'bg-blue-500',
    low: 'bg-zinc-400',
  }
</script>

<div class={cn('flex-1 overflow-x-auto', className)}>
  {#if rows.length === 0}
    <div
      class="text-muted-foreground flex h-full items-center justify-center rounded-md border px-4 py-10 text-sm"
    >
      No tickets match the current filters.
    </div>
  {:else}
    <div class="border-border rounded-md border">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-border text-muted-foreground border-b text-left text-xs">
            <th class="px-4 py-2.5 font-medium">Ticket</th>
            <th class="px-4 py-2.5 font-medium">Status</th>
            <th class="px-4 py-2.5 font-medium">Priority</th>
            <th class="px-4 py-2.5 font-medium">Workflow</th>
            <th class="px-4 py-2.5 font-medium">Agent</th>
            <th class="px-4 py-2.5 text-right font-medium">Updated</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as row (row.ticket.id)}
            <tr
              class="border-border hover:bg-muted/50 cursor-pointer border-b transition-colors last:border-0"
              onclick={() => onticketclick?.(row.ticket)}
            >
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <span class="text-muted-foreground font-mono text-xs"
                    >{row.ticket.identifier}</span
                  >
                  <span class="text-foreground">{row.ticket.title}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <Badge variant="outline" class="gap-1.5 text-xs">
                  <span class="size-2 rounded-full" style="background-color: {row.statusColor}"
                  ></span>
                  {row.statusName}
                </Badge>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  <span class={cn('size-2 rounded-full', priorityColors[row.ticket.priority])}
                  ></span>
                  <span class="text-muted-foreground text-xs capitalize">{row.ticket.priority}</span
                  >
                </div>
              </td>
              <td class="px-4 py-3">
                <span class="text-muted-foreground text-xs">
                  {row.ticket.workflowType ?? 'Unassigned'}
                </span>
              </td>
              <td class="px-4 py-3">
                <span class="text-muted-foreground text-xs"
                  >{row.ticket.agentName ?? 'Unassigned'}</span
                >
              </td>
              <td class="px-4 py-3 text-right">
                <span class="text-muted-foreground text-xs">
                  {formatRelativeTime(row.ticket.updatedAt)}
                </span>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
