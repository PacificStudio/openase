<script lang="ts">
  import { cn, formatRelativeTime } from '$lib/utils'
  import Separator from '$ui/separator/separator.svelte'
  import {
    Settings, Clock, RotateCcw, Layers, Activity, Zap,
  } from '@lucide/svelte'
  import type { WorkflowSummary } from '../types'

  let {
    workflow,
    class: className = '',
  }: {
    workflow: WorkflowSummary
    class?: string
  } = $props()

  const statItems = $derived([
    { label: 'Max Concurrent', value: workflow.maxConcurrent, icon: Layers },
    { label: 'Timeout', value: `${workflow.timeoutMinutes}m`, icon: Clock },
    { label: 'Max Retry', value: workflow.maxRetry, icon: RotateCcw },
  ])
</script>

<div class={cn('flex h-full flex-col overflow-y-auto border-l border-border', className)}>
  <div class="px-4 py-3">
    <h3 class="text-sm font-medium text-foreground">{workflow.name}</h3>
    <div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
      <span class="capitalize">{workflow.type}</span>
      <span>v{workflow.version}</span>
      <span class={cn(
        'size-1.5 rounded-full',
        workflow.isActive ? 'bg-emerald-500' : 'bg-neutral-500',
      )}></span>
      <span>{workflow.isActive ? 'Active' : 'Inactive'}</span>
    </div>
  </div>

  <Separator />

  <div class="px-4 py-3">
    <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
      <Zap class="size-3" />
      Status Flow
    </div>
    <div class="mt-2 space-y-1.5">
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Pickup</span>
        <span class="font-mono text-foreground">{workflow.pickupStatus}</span>
      </div>
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Finish</span>
        <span class="font-mono text-foreground">{workflow.finishStatus}</span>
      </div>
    </div>
  </div>

  <Separator />

  <div class="px-4 py-3">
    <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
      <Settings class="size-3" />
      Configuration
    </div>
    <div class="mt-2 space-y-2">
      {#each statItems as item}
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2 text-xs text-muted-foreground">
            <item.icon class="size-3" />
            {item.label}
          </div>
          <span class="text-xs font-mono text-foreground">{item.value}</span>
        </div>
      {/each}
    </div>
  </div>

  <Separator />

  <div class="px-4 py-3">
    <div class="flex items-center gap-2 text-xs font-medium text-muted-foreground">
      <Activity class="size-3" />
      Recent Stats
    </div>
    <div class="mt-2 space-y-2">
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Success Rate</span>
        <span class={cn(
          'font-mono',
          workflow.recentSuccessRate >= 80
            ? 'text-emerald-400'
            : workflow.recentSuccessRate >= 50
              ? 'text-amber-400'
              : 'text-red-400',
        )}>
          {workflow.recentSuccessRate}%
        </span>
      </div>
      <div class="h-1.5 overflow-hidden rounded-full bg-muted">
        <div
          class={cn(
            'h-full rounded-full',
            workflow.recentSuccessRate >= 80
              ? 'bg-emerald-500'
              : workflow.recentSuccessRate >= 50
                ? 'bg-amber-500'
                : 'bg-red-500',
          )}
          style="width: {workflow.recentSuccessRate}%"
        ></div>
      </div>
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Last modified</span>
        <span class="text-foreground">{formatRelativeTime(workflow.lastModified)}</span>
      </div>
    </div>
  </div>
</div>
