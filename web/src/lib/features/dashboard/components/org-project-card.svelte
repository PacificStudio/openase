<script lang="ts">
  import type { Project } from '$lib/api/contracts'
  import { projectPath } from '$lib/stores/app-context'
  import { formatCurrency, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Bot, Coins, Ticket as TicketIcon } from '@lucide/svelte'

  let {
    currentOrgId,
    project,
    metrics,
    loading = false,
  }: {
    currentOrgId: string | undefined
    project: Project
    metrics:
      | {
          runningAgents: number
          activeTickets: number
          todayCost: number
          lastActivity: string | null
        }
      | undefined
    loading?: boolean
  } = $props()
</script>

<a
  href={currentOrgId ? projectPath(currentOrgId, project.id) : '/'}
  class="border-border bg-card hover:bg-muted/30 rounded-lg border p-5 transition-colors"
>
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0 flex-1">
      <h3 class="text-foreground truncate text-sm font-semibold">{project.name}</h3>
      <p class="text-muted-foreground mt-1 truncate text-xs">
        {project.description || 'No description'}
      </p>
    </div>
    <Badge variant="secondary" class="shrink-0 text-[10px]">{project.status}</Badge>
  </div>

  {#if metrics}
    <div class="text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs">
      <span class="flex items-center gap-1">
        <Bot class="size-3" />
        {metrics.runningAgents} agent{metrics.runningAgents !== 1 ? 's' : ''}
      </span>
      <span class="flex items-center gap-1">
        <TicketIcon class="size-3" />
        {metrics.activeTickets} ticket{metrics.activeTickets !== 1 ? 's' : ''}
      </span>
      <span class="flex items-center gap-1">
        <Coins class="size-3" />
        {formatCurrency(metrics.todayCost)} today
      </span>
      {#if metrics.lastActivity}
        <span class="ml-auto">{formatRelativeTime(metrics.lastActivity)}</span>
      {/if}
    </div>
  {:else if loading}
    <div class="text-muted-foreground mt-3 text-xs">Loading metrics…</div>
  {/if}
</a>
