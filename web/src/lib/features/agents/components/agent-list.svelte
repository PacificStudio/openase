<script lang="ts">
  import { capabilityCatalog } from '$lib/features/capabilities'
  import { cn, formatRelativeTime, formatCurrency } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Terminal, Pause, Play } from '@lucide/svelte'
  import type { AgentInstance } from '../types'

  let {
    agents,
    onSelectTicket,
    onViewOutput,
  }: {
    agents: AgentInstance[]
    onSelectTicket?: (ticketId: string) => void
    onViewOutput?: (agentId: string) => void
  } = $props()

  const statusColors: Record<AgentInstance['status'], string> = {
    idle: 'bg-emerald-500',
    claimed: 'bg-amber-500',
    running: 'bg-blue-500',
    failed: 'bg-red-500',
    terminated: 'bg-slate-500',
  }

  const statusLabels: Record<AgentInstance['status'], string> = {
    idle: 'Idle',
    claimed: 'Claimed',
    running: 'Running',
    failed: 'Failed',
    terminated: 'Terminated',
  }

  const agentOutputCapability = capabilityCatalog.agentOutput
  const agentPauseCapability = capabilityCatalog.agentPause
  const agentResumeCapability = capabilityCatalog.agentResume
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-border text-muted-foreground border-b text-left text-xs">
        <th class="pr-2 pb-2 pl-3 font-medium">Status</th>
        <th class="px-2 pb-2 font-medium">Agent</th>
        <th class="px-2 pb-2 font-medium">Current Ticket</th>
        <th class="px-2 pb-2 font-medium">Last Heartbeat</th>
        <th class="px-2 pb-2 text-right font-medium">Completed</th>
        <th class="px-2 pb-2 text-right font-medium">Cost</th>
        <th class="pr-3 pb-2 pl-2 text-right font-medium">Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each agents as agent (agent.id)}
        <tr class="group border-border/50 hover:bg-muted/30 border-b transition-colors">
          <td class="py-2.5 pr-2 pl-3">
            <div class="flex items-center gap-2">
              <span class={cn('size-2 rounded-full', statusColors[agent.status])}></span>
              <span class="text-muted-foreground text-xs">{statusLabels[agent.status]}</span>
            </div>
          </td>
          <td class="px-2 py-2.5">
            <div class="flex items-center gap-2">
              <span class="text-foreground font-medium">{agent.name}</span>
              <Badge variant="secondary" class="text-[10px]">{agent.providerName}</Badge>
            </div>
            <div class="text-muted-foreground text-xs">{agent.modelName}</div>
          </td>
          <td class="px-2 py-2.5">
            {#if agent.currentTicket}
              <button
                type="button"
                onclick={() => agent.currentTicket && onSelectTicket?.(agent.currentTicket.id)}
                class="text-primary text-xs hover:underline"
              >
                {agent.currentTicket.identifier}
              </button>
              <div class="text-muted-foreground max-w-48 truncate text-xs">
                {agent.currentTicket.title}
              </div>
            {:else}
              <span class="text-muted-foreground/50 text-xs">&mdash;</span>
            {/if}
          </td>
          <td class="px-2 py-2.5">
            {#if agent.lastHeartbeat}
              <span class="text-muted-foreground text-xs">
                {formatRelativeTime(agent.lastHeartbeat)}
              </span>
            {:else}
              <span class="text-muted-foreground/50 text-xs">&mdash;</span>
            {/if}
          </td>
          <td class="px-2 py-2.5 text-right">
            <span class="text-foreground text-xs tabular-nums">{agent.todayCompleted}</span>
          </td>
          <td class="px-2 py-2.5 text-right">
            <span class="text-foreground text-xs tabular-nums">
              {formatCurrency(agent.todayCost)}
            </span>
          </td>
          <td class="py-2.5 pr-3 pl-2">
            <div
              class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100"
            >
              <Button
                variant="ghost"
                size="icon-xs"
                aria-label="View output"
                disabled={agentOutputCapability.state !== 'available'}
                title={agentOutputCapability.summary}
                onclick={() => onViewOutput?.(agent.id)}
              >
                <Terminal class="size-3.5" />
              </Button>
              {#if agent.status === 'running'}
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Pause agent"
                  disabled
                  title={agentPauseCapability.summary}
                >
                  <Pause class="size-3.5" />
                </Button>
              {:else}
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="Resume agent"
                  disabled
                  title={agentResumeCapability.summary}
                >
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
