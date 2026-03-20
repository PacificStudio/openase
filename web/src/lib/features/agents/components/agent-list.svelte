<script lang="ts">
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Terminal, Pause, Play } from '@lucide/svelte'
  import type { AgentInstance } from '../types'

  let {
    agents,
    onSelectTicket,
  }: {
    agents: AgentInstance[]
    onSelectTicket?: (ticketId: string) => void
  } = $props()

  const statusColors: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500',
    running: 'bg-blue-500',
    offline: 'bg-red-500',
    stalled: 'bg-yellow-500',
  }

  const statusLabels: Record<AgentInstance['status'], string> = {
    idle: 'Idle',
    running: 'Running',
    offline: 'Offline',
    stalled: 'Stalled',
  }
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-b border-border text-left text-xs text-muted-foreground">
        <th class="pb-2 pl-3 pr-2 font-medium">Status</th>
        <th class="px-2 pb-2 font-medium">Agent</th>
        <th class="px-2 pb-2 font-medium">Current Ticket</th>
        <th class="px-2 pb-2 font-medium">Last Heartbeat</th>
        <th class="px-2 pb-2 text-right font-medium">Completed</th>
        <th class="px-2 pb-2 text-right font-medium">Cost</th>
        <th class="pb-2 pl-2 pr-3 text-right font-medium">Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each agents as agent (agent.id)}
        <tr class="group border-b border-border/50 transition-colors hover:bg-muted/30">
          <td class="py-2.5 pl-3 pr-2">
            <div class="flex items-center gap-2">
              <span class={cn('size-2 rounded-full', statusColors[agent.status])}></span>
              <span class="text-xs text-muted-foreground">{statusLabels[agent.status]}</span>
            </div>
          </td>
          <td class="px-2 py-2.5">
            <div class="flex items-center gap-2">
              <span class="font-medium text-foreground">{agent.name}</span>
              <Badge variant="secondary" class="text-[10px]">{agent.providerName}</Badge>
            </div>
            <div class="text-xs text-muted-foreground">{agent.modelName}</div>
          </td>
          <td class="px-2 py-2.5">
            {#if agent.currentTicket}
              <button
                type="button"
                onclick={() => {
                  if (agent.currentTicket) {
                    onSelectTicket?.(agent.currentTicket.id)
                  }
                }}
                class="text-xs text-primary hover:underline"
              >
                {agent.currentTicket.identifier}
              </button>
              <div class="max-w-48 truncate text-xs text-muted-foreground">
                {agent.currentTicket.title}
              </div>
            {:else}
              <span class="text-xs text-muted-foreground/50">&mdash;</span>
            {/if}
          </td>
          <td class="px-2 py-2.5">
            <span class="text-xs text-muted-foreground">
              {formatRelativeTime(agent.lastHeartbeat)}
            </span>
          </td>
          <td class="px-2 py-2.5 text-right">
            <span class="text-xs tabular-nums text-foreground">{agent.todayCompleted}</span>
          </td>
          <td class="px-2 py-2.5 text-right">
            <span class="text-xs tabular-nums text-foreground">
              {formatCurrency(agent.todayCost)}
            </span>
          </td>
          <td class="py-2.5 pl-2 pr-3">
            <div class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100">
              <Button variant="ghost" size="icon-xs" aria-label="View output" disabled title="Agent output is not exposed by the current API">
                <Terminal class="size-3.5" />
              </Button>
              {#if agent.status === 'running'}
                <Button variant="ghost" size="icon-xs" aria-label="Pause agent" disabled title="Agent pause is not exposed by the current API">
                  <Pause class="size-3.5" />
                </Button>
              {:else}
                <Button variant="ghost" size="icon-xs" aria-label="Resume agent" disabled title="Agent resume is not exposed by the current API">
                  <Play class="size-3.5" />
                </Button>
              {/if}
            </div>
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
