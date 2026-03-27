<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { ExternalLink, Trash2 } from '@lucide/svelte'
  import type { GovernanceAgent } from './agent-settings-model'
  import { governanceAgentStatusClasses, governanceAgentStatusLabels } from './agent-settings-model'

  let {
    agents,
    agentsConsoleHref,
    deletingAgentId = null,
    onDelete,
  }: {
    agents: GovernanceAgent[]
    agentsConsoleHref: string
    deletingAgentId?: string | null
    onDelete?: (agent: GovernanceAgent) => Promise<void>
  } = $props()

  function deleteTitle(agent: GovernanceAgent) {
    if (deletingAgentId === agent.id) return 'Deleting agent...'
    if (agent.activeRunCount > 0) {
      return 'Finish or pause active runs from /agents before deleting this agent.'
    }
    return 'Delete this agent definition from the project.'
  }

  async function handleDelete(agent: GovernanceAgent) {
    if (!onDelete) return

    if (typeof window !== 'undefined') {
      const confirmed = window.confirm(
        `Delete agent "${agent.name}"? This removes the project agent definition. Existing runtime history may still block deletion.`,
      )
      if (!confirmed) return
    }

    await onDelete(agent)
  }
</script>

<Card.Root>
  <Card.Header class="flex-row items-start justify-between gap-3 space-y-0">
    <div>
      <Card.Title>Registered agents</Card.Title>
      <Card.Description>
        Governance inventory for provider coverage and runtime readiness.
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
                Runtime phase: {agent.runtimePhase} · Machine: {agent.machineName} · Active runs:
                {agent.activeRunCount}
              </div>
            </div>
            <div class="flex items-center gap-2">
              <div class="text-muted-foreground text-right text-xs">
                {#if agent.lastHeartbeat}
                  Last heartbeat {formatRelativeTime(agent.lastHeartbeat)}
                {:else}
                  No heartbeat yet
                {/if}
              </div>
              {#if onDelete}
                <Button
                  variant="destructive"
                  size="xs"
                  disabled={deletingAgentId === agent.id || agent.activeRunCount > 0}
                  title={deleteTitle(agent)}
                  aria-label={`Delete agent ${agent.name}`}
                  onclick={() => void handleDelete(agent)}
                >
                  <Trash2 class="size-3.5" />
                  {deletingAgentId === agent.id ? 'Deleting…' : 'Delete'}
                </Button>
              {/if}
            </div>
          </div>

          <div class="text-muted-foreground mt-3 text-xs">
            Ticket workspaces are derived at runtime from the bound project and machine.
          </div>
        </div>
      {/each}
    {/if}
  </Card.Content>
</Card.Root>
