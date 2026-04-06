<script lang="ts">
  import { Badge } from '$ui/badge'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Ban, Cog, Loader, CircleX } from '@lucide/svelte'
  import * as Tooltip from '$ui/tooltip'
  import { formatBoardPriorityLabel } from '../priority'
  import { viewport } from '$lib/stores/viewport.svelte'
  import type { BoardColumn, BoardTicket } from '../types'
  import StageIcon from './stage-icon.svelte'
  import PriorityIcon from './priority-icon.svelte'

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
</script>

<div class={cn('flex-1 overflow-x-auto', className)}>
  {#if rows.length === 0}
    <div
      class="text-muted-foreground flex h-full items-center justify-center rounded-md border px-4 py-10 text-sm"
    >
      No tickets match the current filters.
    </div>
  {:else if viewport.isMobile}
    <!-- Mobile: card layout -->
    <div class="flex flex-col gap-2">
      {#each rows as row (row.ticket.id)}
        <button
          type="button"
          class="border-border bg-card hover:bg-muted/50 flex w-full flex-col gap-2 rounded-lg border p-3 text-left transition-colors"
          onclick={() => onticketclick?.(row.ticket)}
        >
          <div class="flex items-start gap-2">
            <StageIcon stage={row.ticket.stage} color={row.statusColor} class="mt-0.5 shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1.5">
                <span class="text-muted-foreground shrink-0 font-mono text-xs"
                  >{row.ticket.identifier}</span
                >
                {#if row.ticket.isBlocked}
                  <Badge
                    variant="outline"
                    class="h-4 gap-0.5 border-red-500/30 bg-red-500/10 py-0 text-[10px] text-red-500"
                  >
                    <Ban class="size-2.5" />
                    Blocked
                  </Badge>
                {/if}
              </div>
              <p class="text-foreground mt-0.5 text-sm leading-snug">{row.ticket.title}</p>
            </div>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <Badge variant="outline" class="gap-1 text-[11px]">
              <StageIcon stage={row.ticket.stage} color={row.statusColor} class="size-2.5" />
              {row.statusName}
            </Badge>
            <span class="flex items-center gap-1">
              <PriorityIcon priority={row.ticket.priority} />
              <span class="text-muted-foreground text-[11px]">
                {formatBoardPriorityLabel(row.ticket.priority)}
              </span>
            </span>
            {#if row.ticket.runtimePhase === 'executing'}
              <Cog class="size-3 animate-spin text-emerald-500" />
            {:else if row.ticket.runtimePhase === 'launching'}
              <Loader class="size-3 animate-spin text-amber-500 [animation-duration:2s]" />
            {:else if row.ticket.runtimePhase === 'failed'}
              <CircleX class="size-3 text-red-500" />
            {/if}
            <span class="text-muted-foreground ml-auto text-[11px]">
              {formatRelativeTime(row.ticket.updatedAt)}
            </span>
          </div>
        </button>
      {/each}
    </div>
  {:else}
    <!-- Desktop / Tablet: table layout -->
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
                  <StageIcon stage={row.ticket.stage} color={row.statusColor} />
                  <span class="text-muted-foreground font-mono text-xs"
                    >{row.ticket.identifier}</span
                  >
                  <span class="text-foreground">{row.ticket.title}</span>
                  {#if row.ticket.isBlocked}
                    <Badge
                      variant="outline"
                      class="h-4 gap-0.5 border-red-500/30 bg-red-500/10 py-0 text-[10px] text-red-500"
                    >
                      <Ban class="size-2.5" />
                      Blocked
                    </Badge>
                  {/if}
                </div>
              </td>
              <td class="px-4 py-3">
                <Badge variant="outline" class="gap-1.5 text-xs">
                  <StageIcon stage={row.ticket.stage} color={row.statusColor} class="size-3" />
                  {row.statusName}
                </Badge>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  <PriorityIcon priority={row.ticket.priority} />
                  <span class="text-muted-foreground text-xs">
                    {formatBoardPriorityLabel(row.ticket.priority)}
                  </span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span class="text-muted-foreground text-xs">
                  {row.ticket.workflowType ?? 'Unassigned'}
                </span>
              </td>
              <td class="px-4 py-3">
                <span class="text-muted-foreground flex items-center gap-1.5 text-xs">
                  {row.ticket.agentName ?? 'Unassigned'}
                  {#if row.ticket.runtimePhase === 'executing'}
                    <Cog class="size-3 animate-spin text-emerald-500" />
                  {:else if row.ticket.runtimePhase === 'launching'}
                    <Loader class="size-3 animate-spin text-amber-500 [animation-duration:2s]" />
                  {:else if row.ticket.runtimePhase === 'failed'}
                    {#if row.ticket.lastError}
                      <Tooltip.Provider>
                        <Tooltip.Root>
                          <Tooltip.Trigger class="inline-flex text-red-500">
                            <CircleX class="size-3" />
                          </Tooltip.Trigger>
                          <Tooltip.Portal>
                            <Tooltip.Content
                              side="top"
                              class="bg-popover text-popover-foreground max-w-64 rounded-md border px-3 py-2 text-xs shadow-md"
                            >
                              {row.ticket.lastError}
                            </Tooltip.Content>
                          </Tooltip.Portal>
                        </Tooltip.Root>
                      </Tooltip.Provider>
                    {:else}
                      <CircleX class="size-3 text-red-500" />
                    {/if}
                  {/if}
                </span>
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
