<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Terminal } from '@lucide/svelte'
  import type { AgentRunInstance } from '../types'

  let {
    agentRuns,
    onSelectTicket,
    onViewOutput,
  }: {
    agentRuns: AgentRunInstance[]
    onSelectTicket?: (ticketId: string) => void
    onViewOutput?: (agentId: string) => void
  } = $props()

  const statusColors: Record<AgentRunInstance['status'], string> = {
    launching: 'bg-amber-500',
    ready: 'bg-sky-500',
    executing: 'bg-blue-500',
    completed: 'bg-emerald-500',
    errored: 'bg-red-500',
    terminated: 'bg-slate-500',
  }

  const statusLabels: Record<AgentRunInstance['status'], string> = {
    launching: 'Launching',
    ready: 'Ready',
    executing: 'Executing',
    completed: 'Completed',
    errored: 'Errored',
    terminated: 'Terminated',
  }

  function shortRunId(runId: string) {
    return runId.slice(0, 8)
  }
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead>
      <tr class="border-border text-muted-foreground border-b text-left text-xs">
        <th class="pr-2 pb-2 pl-3 font-medium">Status</th>
        <th class="px-2 pb-2 font-medium">Run</th>
        <th class="px-2 pb-2 font-medium">Agent</th>
        <th class="px-2 pb-2 font-medium">Workflow</th>
        <th class="px-2 pb-2 font-medium">Ticket</th>
        <th class="px-2 pb-2 font-medium">Last Heartbeat</th>
        <th class="pr-3 pb-2 pl-2 text-right font-medium">Actions</th>
      </tr>
    </thead>
    <tbody>
      {#if agentRuns.length === 0}
        <tr>
          <td colspan="7" class="text-muted-foreground px-3 py-8 text-center text-sm">
            No AgentRuns recorded for this project yet.
          </td>
        </tr>
      {:else}
        {#each agentRuns as agentRun (agentRun.id)}
          <tr class="group border-border/50 hover:bg-muted/30 border-b transition-colors">
            <td class="py-2.5 pr-2 pl-3">
              <div class="flex items-center gap-2">
                <span class={cn('size-2 rounded-full', statusColors[agentRun.status])}></span>
                <span class="text-muted-foreground text-xs">{statusLabels[agentRun.status]}</span>
              </div>
            </td>
            <td class="px-2 py-2.5">
              <div class="text-foreground font-medium">run {shortRunId(agentRun.id)}</div>
              <div class="text-muted-foreground text-xs">
                {#if agentRun.sessionId}
                  session {agentRun.sessionId}
                {:else}
                  created {formatRelativeTime(agentRun.createdAt)}
                {/if}
              </div>
            </td>
            <td class="px-2 py-2.5">
              <div class="flex items-center gap-2">
                <span class="text-foreground font-medium">{agentRun.agentName}</span>
                <Badge variant="secondary" class="text-[10px]">{agentRun.providerName}</Badge>
              </div>
              <div class="text-muted-foreground text-xs">{agentRun.modelName}</div>
            </td>
            <td class="px-2 py-2.5">
              <div class="text-foreground text-xs">{agentRun.workflowName}</div>
            </td>
            <td class="px-2 py-2.5">
              <button
                type="button"
                onclick={() => onSelectTicket?.(agentRun.ticket.id)}
                class="text-primary text-xs hover:underline"
              >
                {agentRun.ticket.identifier}
              </button>
              <div class="text-muted-foreground max-w-48 truncate text-xs">
                {agentRun.ticket.title}
              </div>
            </td>
            <td class="px-2 py-2.5">
              <div class="text-muted-foreground text-xs">
                {#if agentRun.lastHeartbeat}
                  {formatRelativeTime(agentRun.lastHeartbeat)}
                {:else if agentRun.runtimeStartedAt}
                  started {formatRelativeTime(agentRun.runtimeStartedAt)}
                {:else}
                  &mdash;
                {/if}
              </div>
              {#if agentRun.lastError}
                <div class="truncate pt-1 text-xs text-red-600 dark:text-red-300">
                  {agentRun.lastError}
                </div>
              {/if}
            </td>
            <td class="py-2.5 pr-3 pl-2">
              <div
                class="flex items-center justify-end gap-1 opacity-0 transition-opacity group-hover:opacity-100"
              >
                <Button
                  variant="ghost"
                  size="icon-xs"
                  aria-label="View output"
                  title="View runtime output"
                  onclick={() => onViewOutput?.(agentRun.agentId)}
                >
                  <Terminal class="size-3.5" />
                </Button>
              </div>
            </td>
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>
