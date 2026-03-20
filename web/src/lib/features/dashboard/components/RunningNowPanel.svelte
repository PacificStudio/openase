<script lang="ts">
  import { Bot, HeartPulse } from '@lucide/svelte'
  import { Badge } from '$lib/components/ui/badge'
  import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
  } from '$lib/components/ui/card'
  import {
    heartbeatBadgeClass,
    heartbeatLabel,
    streamBadgeClass,
  } from '$lib/features/workspace/metrics'
  import type { Agent } from '$lib/features/workspace/types'
  import type { StreamConnectionState } from '$lib/api/sse'

  let {
    agents = [],
    busy = false,
    error = '',
    selectedAgentId = '',
    heartbeatNow,
    agentStreamState = 'idle',
    onSelectAgent,
  }: {
    agents?: Agent[]
    busy?: boolean
    error?: string
    selectedAgentId?: string
    heartbeatNow: number
    agentStreamState?: StreamConnectionState
    onSelectAgent?: (agentId: string) => void
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <div class="flex items-center justify-between gap-3">
      <div>
        <CardTitle class="flex items-center gap-2">
          <Bot class="size-4" />
          <span>Running now</span>
        </CardTitle>
        <CardDescription>Current agents and heartbeat freshness.</CardDescription>
      </div>
      <Badge class={streamBadgeClass(agentStreamState)}>{agentStreamState}</Badge>
    </div>
  </CardHeader>

  <CardContent class="space-y-3">
    {#if busy}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border px-4 py-5 text-sm"
      >
        Loading agent telemetry…
      </div>
    {:else if error}
      <div
        class="text-destructive border-destructive/25 bg-destructive/10 rounded-3xl border px-4 py-5 text-sm"
      >
        {error}
      </div>
    {:else if agents.length === 0}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        No agents registered for this project.
      </div>
    {:else}
      {#each agents as agent}
        <button
          type="button"
          class={`w-full rounded-3xl border px-4 py-4 text-left transition ${
            agent.id === selectedAgentId
              ? 'border-foreground/30 bg-foreground text-background shadow-lg shadow-black/10'
              : 'border-border/70 bg-background/60 hover:border-foreground/15 hover:bg-background'
          }`}
          onclick={() => onSelectAgent?.(agent.id)}
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div class="flex items-center gap-2">
                <p class="text-sm font-semibold">{agent.name}</p>
                <Badge variant={agent.id === selectedAgentId ? 'secondary' : 'outline'}>
                  {agent.status}
                </Badge>
              </div>
              <p
                class={`mt-2 text-xs ${agent.id === selectedAgentId ? 'text-background/75' : 'text-muted-foreground'}`}
              >
                {agent.current_ticket_id ? `Ticket ${agent.current_ticket_id}` : 'No active ticket'}
              </p>
            </div>

            <div class="text-right">
              <Badge class={heartbeatBadgeClass(agent.last_heartbeat_at, heartbeatNow)}>
                <HeartPulse class="mr-1 size-3" />
                {heartbeatLabel(agent.last_heartbeat_at, heartbeatNow)}
              </Badge>
              <p
                class={`mt-2 text-xs ${agent.id === selectedAgentId ? 'text-background/75' : 'text-muted-foreground'}`}
              >
                {agent.total_tickets_completed} done · {agent.total_tokens_used} tokens
              </p>
            </div>
          </div>
        </button>
      {/each}
    {/if}
  </CardContent>
</Card>
