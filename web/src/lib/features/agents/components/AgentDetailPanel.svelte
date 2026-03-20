<script lang="ts">
  import { Bot, FolderGit2, Ticket } from '@lucide/svelte'
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
    type Agent,
    type Project,
  } from '$lib/features/workspace'

  let {
    agent = null,
    project = null,
    heartbeatNow,
  }: {
    agent?: Agent | null
    project?: Project | null
    heartbeatNow: number
  } = $props()
</script>

<Card class="border-border/80 bg-background/80 backdrop-blur">
  <CardHeader>
    <CardTitle class="flex items-center gap-2">
      <Bot class="size-4" />
      <span>Agent detail panel</span>
    </CardTitle>
    <CardDescription>
      Session, workload, workspace path, and capabilities stay grouped in the feature layer.
    </CardDescription>
  </CardHeader>

  <CardContent class="space-y-4">
    {#if agent}
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="secondary">{agent.status}</Badge>
        <Badge class={heartbeatBadgeClass(agent.last_heartbeat_at, heartbeatNow)}>
          {heartbeatLabel(agent.last_heartbeat_at, heartbeatNow)}
        </Badge>
        {#if project}
          <Badge variant="outline">{project.name}</Badge>
        {/if}
      </div>

      <div class="grid gap-3 sm:grid-cols-2">
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <div class="text-muted-foreground flex items-center gap-2 text-xs tracking-[0.18em] uppercase">
            <Ticket class="size-3.5" />
            Current ticket
          </div>
          <p class="mt-2 text-sm font-semibold">{agent.current_ticket_id ?? 'Idle queue'}</p>
        </div>
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <div class="text-muted-foreground flex items-center gap-2 text-xs tracking-[0.18em] uppercase">
            <FolderGit2 class="size-3.5" />
            Workspace
          </div>
          <p class="mt-2 break-all text-sm font-semibold">{agent.workspace_path}</p>
        </div>
      </div>

      <div class="grid gap-3 sm:grid-cols-3">
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Completed</p>
          <p class="mt-2 text-lg font-semibold">{agent.total_tickets_completed}</p>
        </div>
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Tokens</p>
          <p class="mt-2 text-lg font-semibold">{agent.total_tokens_used}</p>
        </div>
        <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
          <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Session</p>
          <p class="mt-2 break-all text-sm font-semibold">{agent.session_id}</p>
        </div>
      </div>

      <div class="border-border/70 bg-background/60 rounded-2xl border px-4 py-3">
        <p class="text-muted-foreground text-xs tracking-[0.18em] uppercase">Capabilities</p>
        <div class="mt-3 flex flex-wrap gap-2">
          {#if agent.capabilities.length === 0}
            <Badge variant="outline">No capabilities declared</Badge>
          {:else}
            {#each agent.capabilities as capability}
              <Badge variant="outline">{capability}</Badge>
            {/each}
          {/if}
        </div>
      </div>
    {:else}
      <div
        class="text-muted-foreground border-border/70 bg-muted/30 rounded-3xl border border-dashed px-4 py-5 text-sm"
      >
        Select an agent to inspect its live session details.
      </div>
    {/if}
  </CardContent>
</Card>
