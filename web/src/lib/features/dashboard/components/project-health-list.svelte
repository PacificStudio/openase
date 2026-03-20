<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import type { ProjectSummary } from '../types'
  import { Bot, Ticket } from '@lucide/svelte'

  let {
    projects,
    class: className = '',
  }: {
    projects: ProjectSummary[]
    class?: string
  } = $props()

  const healthColor: Record<string, string> = {
    healthy: 'bg-emerald-500',
    warning: 'bg-amber-500',
    blocked: 'bg-red-500',
  }

  const healthVariant: Record<string, 'default' | 'secondary' | 'destructive'> = {
    healthy: 'secondary',
    warning: 'secondary',
    blocked: 'destructive',
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Projects</h3>
    <span class="text-muted-foreground text-xs">{projects.length} total</span>
  </div>

  <div class="divide-border divide-y">
    {#each projects as project (project.id)}
      <div class="hover:bg-muted/50 flex items-center gap-4 px-4 py-3 transition-colors">
        <div class="flex min-w-0 flex-1 items-center gap-2">
          <span class={cn('size-2 shrink-0 rounded-full', healthColor[project.health])}></span>
          <span class="text-foreground truncate text-sm font-medium">{project.name}</span>
        </div>

        <Badge variant={healthVariant[project.health]} class="text-[10px] capitalize">
          {project.health}
        </Badge>

        <div class="text-muted-foreground flex shrink-0 items-center gap-1 text-xs">
          <Bot class="size-3" />
          <span>{project.activeAgents}</span>
        </div>

        <div class="text-muted-foreground flex shrink-0 items-center gap-1 text-xs">
          <Ticket class="size-3" />
          <span>{project.activeTickets}</span>
        </div>

        <span class="text-muted-foreground w-16 shrink-0 text-right text-xs">
          {formatRelativeTime(project.lastActivity)}
        </span>
      </div>
    {/each}
  </div>
</div>
