<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import * as Select from '$ui/select'
  import type { ProjectStatus, ProjectSummary } from '../types'
  import { Bot, Ticket } from '@lucide/svelte'

  let {
    project,
    savingProjectStatusId = null,
    onUpdateStatus,
    class: className = '',
  }: {
    project: ProjectSummary
    savingProjectStatusId?: string | null
    onUpdateStatus?: (projectId: string, status: ProjectStatus) => void | Promise<void>
    class?: string
  } = $props()

  const projectStatusOptions: ProjectStatus[] = [
    'Backlog',
    'Planned',
    'In Progress',
    'Completed',
    'Canceled',
    'Archived',
  ]

  const statusClassName: Record<ProjectStatus, string> = {
    Backlog:
      'border-slate-200 bg-slate-100 text-slate-700 hover:bg-slate-200 dark:border-slate-800 dark:bg-slate-900 dark:text-slate-200',
    Planned:
      'border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-100 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-200',
    'In Progress':
      'border-emerald-200 bg-emerald-50 text-emerald-800 hover:bg-emerald-100 dark:border-emerald-900/60 dark:bg-emerald-950/40 dark:text-emerald-200',
    Completed:
      'border-sky-200 bg-sky-50 text-sky-800 hover:bg-sky-100 dark:border-sky-900/60 dark:bg-sky-950/40 dark:text-sky-200',
    Canceled:
      'border-rose-200 bg-rose-50 text-rose-800 hover:bg-rose-100 dark:border-rose-900/60 dark:bg-rose-950/40 dark:text-rose-200',
    Archived:
      'border-border bg-background text-muted-foreground hover:bg-muted dark:hover:bg-muted/60',
  }

  function statusLabel(status: ProjectStatus) {
    return status
  }
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Project</h3>
  </div>

  <div class="px-4 py-3">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0 flex-1">
        <div class="text-foreground truncate text-sm font-medium">{project.name}</div>
        <p class="text-muted-foreground mt-1 text-xs leading-5">
          {project.description || 'No description yet.'}
        </p>
      </div>

      <Select.Root
        type="single"
        value={project.status}
        onValueChange={(value) => {
          if (!value || value === project.status) return
          void onUpdateStatus?.(project.id, value as ProjectStatus)
        }}
      >
        <Select.Trigger
          class={cn(
            'h-auto min-h-5 w-auto rounded-4xl border px-2 py-0.5 text-xs font-medium shadow-none',
            statusClassName[project.status],
          )}
          disabled={savingProjectStatusId === project.id}
        >
          {statusLabel(project.status)}
        </Select.Trigger>
        <Select.Content>
          {#each projectStatusOptions as status (status)}
            <Select.Item value={status}>{statusLabel(status)}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="text-muted-foreground mt-3 flex items-center gap-4 text-xs">
      <div class="flex shrink-0 items-center gap-1">
        <Bot class="size-3" />
        <span>{project.activeAgents}</span>
      </div>

      <div class="flex shrink-0 items-center gap-1">
        <Ticket class="size-3" />
        <span>{project.activeTickets}</span>
      </div>

      <span class="ml-auto shrink-0 text-right">
        {project.lastActivity ? formatRelativeTime(project.lastActivity) : 'No activity yet'}
      </span>
    </div>
  </div>
</div>
