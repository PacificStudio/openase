<script lang="ts">
  import type { Project } from '$lib/api/contracts'
  import { projectPath } from '$lib/stores/app-context'
  import { formatCurrency, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Skeleton } from '$ui/skeleton'
  import { Bot, Coins, Ticket as TicketIcon } from '@lucide/svelte'

  type ProjectMetrics = {
    runningAgents: number
    activeTickets: number
    todayCost: number
    lastActivity: string | null
  }

  let {
    currentOrgId = null,
    projects = [],
    projectMetrics = {},
    loading = false,
    onCreateProject,
  }: {
    currentOrgId?: string | null
    projects?: Project[]
    projectMetrics?: Record<string, ProjectMetrics>
    loading?: boolean
    onCreateProject?: () => void
  } = $props()
</script>

<section class="space-y-4">
  <h2 class="text-foreground text-lg font-semibold">Projects</h2>

  {#if projects.length > 0}
    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      {#each projects as project (project.id)}
        {@const metrics = projectMetrics[project.id]}
        <a
          href={currentOrgId ? projectPath(currentOrgId, project.id) : '/'}
          class="border-border bg-card hover:bg-muted/30 hover-lift rounded-lg border p-5 transition-colors"
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
            <div
              class="text-muted-foreground mt-3 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs"
            >
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
            <div class="mt-3 flex items-center gap-4">
              <Skeleton class="h-3.5 w-16" />
              <Skeleton class="h-3.5 w-16" />
              <Skeleton class="h-3.5 w-20" />
            </div>
          {/if}
        </a>
      {/each}
    </div>
  {:else}
    <button
      type="button"
      class="border-border hover:border-foreground/20 hover:bg-card w-full rounded-lg border border-dashed px-4 py-12 text-center transition-colors"
      onclick={onCreateProject}
    >
      <p class="text-muted-foreground text-sm">No projects yet.</p>
      <p class="text-foreground mt-1 text-sm font-medium">
        Create your first project to get started
      </p>
    </button>
  {/if}
</section>
