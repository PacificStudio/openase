<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import Badge from '$ui/badge/badge.svelte'
  import { ArrowRight, Circle } from '@lucide/svelte'
  import type { WorkflowSummary, WorkflowType } from '../types'

  let {
    workflows,
    selectedId = '',
    onselect,
    class: className = '',
  }: {
    workflows: WorkflowSummary[]
    selectedId?: string
    onselect?: (id: string) => void
    class?: string
  } = $props()

  const typeColors: Record<WorkflowType, string> = {
    coding: 'bg-blue-500/15 text-blue-400',
    test: 'bg-emerald-500/15 text-emerald-400',
    doc: 'bg-violet-500/15 text-violet-400',
    security: 'bg-red-500/15 text-red-400',
    deploy: 'bg-amber-500/15 text-amber-400',
    'refine-harness': 'bg-cyan-500/15 text-cyan-400',
    custom: 'bg-neutral-500/15 text-neutral-400',
  }

  function rateColor(rate: number): string {
    if (rate >= 80) return 'bg-emerald-500'
    if (rate >= 50) return 'bg-amber-500'
    return 'bg-red-500'
  }
</script>

<div class={cn('flex h-full flex-col overflow-hidden border-r border-border', className)}>
  <div class="px-3 py-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">
    Workflows
  </div>
  <div class="flex-1 overflow-y-auto">
    {#each workflows as wf (wf.id)}
      <button
        class={cn(
          'w-full cursor-pointer border-b border-border px-3 py-3 text-left transition-colors hover:bg-muted/50',
          selectedId === wf.id && 'bg-muted',
        )}
        onclick={() => onselect?.(wf.id)}
      >
        <div class="flex items-center gap-2">
          <Circle
            class={cn('size-2', wf.isActive ? 'fill-emerald-500 text-emerald-500' : 'fill-neutral-500 text-neutral-500')}
          />
          <span class="flex-1 truncate text-sm font-medium text-foreground">
            {wf.name}
          </span>
          <span class={cn('rounded-md px-1.5 py-0.5 text-[10px] font-medium', typeColors[wf.type])}>
            {wf.type}
          </span>
        </div>

        <div class="mt-1.5 flex items-center gap-1 text-xs text-muted-foreground">
          <span class="truncate">{wf.pickupStatus}</span>
          <ArrowRight class="size-3 shrink-0" />
          <span class="truncate">{wf.finishStatus}</span>
        </div>

        <div class="mt-1.5 flex items-center gap-2">
          <div class="h-1 flex-1 overflow-hidden rounded-full bg-muted">
            <div
              class={cn('h-full rounded-full transition-all', rateColor(wf.recentSuccessRate))}
              style="width: {wf.recentSuccessRate}%"
            ></div>
          </div>
          <span class="text-[10px] tabular-nums text-muted-foreground">
            {wf.recentSuccessRate}%
          </span>
        </div>

        <div class="mt-1 text-[10px] text-muted-foreground">
          {formatRelativeTime(wf.lastModified)}
        </div>
      </button>
    {/each}
  </div>
</div>
