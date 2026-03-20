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

<div class={cn('rounded-md border border-border bg-card', className)}>
  <div class="flex items-center justify-between border-b border-border px-4 py-3">
    <h3 class="text-sm font-medium text-foreground">Projects</h3>
    <span class="text-xs text-muted-foreground">{projects.length} total</span>
  </div>

  <div class="divide-y divide-border">
    {#each projects as project (project.id)}
      <div class="flex items-center gap-4 px-4 py-3 transition-colors hover:bg-muted/50">
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <span class={cn('size-2 rounded-full shrink-0', healthColor[project.health])}></span>
          <span class="text-sm font-medium text-foreground truncate">{project.name}</span>
        </div>

        <Badge variant={healthVariant[project.health]} class="text-[10px] capitalize">
          {project.health}
        </Badge>

        <div class="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
          <Bot class="size-3" />
          <span>{project.activeAgents}</span>
        </div>

        <div class="flex items-center gap-1 text-xs text-muted-foreground shrink-0">
          <Ticket class="size-3" />
          <span>{project.activeTickets}</span>
        </div>

        <span class="text-xs text-muted-foreground shrink-0 w-16 text-right">
          {formatRelativeTime(project.lastActivity)}
        </span>
      </div>
    {/each}
  </div>
</div>
