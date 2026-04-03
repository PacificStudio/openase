<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import { ArrowRight, Circle } from '@lucide/svelte'
  import { workflowFamilyColors } from '../model'
  import type { WorkflowSummary } from '../types'

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
</script>

<div class={cn('border-border flex h-full flex-col overflow-hidden border-r', className)}>
  <div class="text-muted-foreground px-3 py-2 text-xs font-medium tracking-wider uppercase">
    Workflows
  </div>
  <div class="flex-1 overflow-y-auto">
    {#each workflows as wf (wf.id)}
      <button
        class={cn(
          'border-border hover:bg-muted/50 w-full cursor-pointer border-b px-3 py-3 text-left transition-colors',
          selectedId === wf.id && 'bg-muted',
        )}
        onclick={() => onselect?.(wf.id)}
      >
        <div class="flex items-center gap-2">
          <Circle
            class={cn(
              'size-2',
              wf.isActive
                ? 'animate-pulse-dot fill-emerald-500 text-emerald-500'
                : 'fill-neutral-500 text-neutral-500',
            )}
          />
          <span class="text-foreground flex-1 truncate text-sm font-medium">
            {wf.name}
          </span>
          <span
            class={cn(
              'rounded-md border px-1.5 py-0.5 text-[10px] font-medium',
              workflowFamilyColors[wf.workflowFamily],
            )}
          >
            {wf.type}
          </span>
        </div>

        <div class="text-muted-foreground mt-1.5 flex items-center gap-1 text-xs">
          <span class="truncate">{wf.pickupStatusLabel}</span>
          <ArrowRight class="size-3 shrink-0" />
          <span class="truncate">{wf.finishStatusLabel}</span>
        </div>

        <div class="text-muted-foreground mt-1.5 text-[10px]">
          {formatRelativeTime(wf.lastModified)}
        </div>
      </button>
    {/each}
  </div>
</div>
