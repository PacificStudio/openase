<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { ExternalLink } from '@lucide/svelte'
  import type { GovernanceAgent } from './agent-settings-model'
  import { governanceAgentStatusClasses, governanceAgentStatusLabels } from './agent-settings-model'

  let {
    agents,
    agentsConsoleHref,
  }: {
    agents: GovernanceAgent[]
    agentsConsoleHref: string
  } = $props()
</script>

<Card.Root>
  <Card.Header class="flex-row items-start justify-between gap-3 space-y-0">
    <div>
      <Card.Title>Registered agents</Card.Title>
      <Card.Description>
        Governance inventory for provider coverage and workspace paths.
      </Card.Description>
    </div>
    <Button href={agentsConsoleHref} variant="outline" size="sm">
      <ExternalLink class="size-3.5" />
      Open /agents
    </Button>
  </Card.Header>
  <Card.Content class="space-y-3">
    {#if agents.length === 0}
      <div class="text-muted-foreground text-sm">
        No agents have been registered for this project yet.
      </div>
    {:else}
      {#each agents as agent (agent.id)}
        <div class="border-border rounded-md border px-4 py-3">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-foreground truncate text-sm font-medium">{agent.name}</span>
                <Badge
                  variant="outline"
                  class={`text-[10px] ${governanceAgentStatusClasses[agent.status]}`}
                >
                  {governanceAgentStatusLabels[agent.status]}
                </Badge>
                <Badge variant="secondary" class="text-[10px]">{agent.providerName}</Badge>
              </div>
              <div class="text-muted-foreground mt-1 text-xs">
                Runtime phase: {agent.runtimePhase}
              </div>
            </div>
            <div class="text-muted-foreground text-right text-xs">
              {#if agent.lastHeartbeat}
                Last heartbeat {formatRelativeTime(agent.lastHeartbeat)}
              {:else}
                No heartbeat yet
              {/if}
            </div>
          </div>

          <div class="text-muted-foreground mt-3 grid gap-3 text-xs md:grid-cols-2">
            <div>
              <div class="text-foreground font-medium">Workspace</div>
              <div class="mt-1 font-mono break-all">{agent.workspacePath || 'Not provided'}</div>
            </div>
          </div>
        </div>
      {/each}
    {/if}
  </Card.Content>
</Card.Root>
