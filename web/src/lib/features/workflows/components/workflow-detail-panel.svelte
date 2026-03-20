<script lang="ts">
  import { cn } from '$lib/utils'
  import Separator from '$ui/separator/separator.svelte'
  import { Settings, Clock, RotateCcw, Layers, FileCode, Zap } from '@lucide/svelte'
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
    { label: 'Stall Timeout', value: `${workflow.stallTimeoutMinutes}m`, icon: Clock },
  ])
</script>

<div class={cn('border-border flex h-full flex-col overflow-y-auto border-l', className)}>
  <div class="px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">{workflow.name}</h3>
    <div class="text-muted-foreground mt-1 flex items-center gap-2 text-xs">
      <span class="capitalize">{workflow.type}</span>
      <span>v{workflow.version}</span>
      <span
        class={cn('size-1.5 rounded-full', workflow.isActive ? 'bg-emerald-500' : 'bg-neutral-500')}
      ></span>
      <span>{workflow.isActive ? 'Active' : 'Inactive'}</span>
    </div>
  </div>

  <Separator />

  <div class="px-4 py-3">
    <div class="text-muted-foreground flex items-center gap-2 text-xs font-medium">
      <Zap class="size-3" />
      Status Flow
    </div>
    <div class="mt-2 space-y-1.5">
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Pickup</span>
        <span class="text-foreground font-mono">{workflow.pickupStatus}</span>
      </div>
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Finish</span>
        <span class="text-foreground font-mono">{workflow.finishStatus}</span>
      </div>
    </div>
  </div>

  <Separator />

  <div class="px-4 py-3">
    <div class="text-muted-foreground flex items-center gap-2 text-xs font-medium">
      <Settings class="size-3" />
      Configuration
    </div>
    <div class="mt-2 space-y-2">
      {#each statItems as item}
        <div class="flex items-center justify-between">
          <div class="text-muted-foreground flex items-center gap-2 text-xs">
            <item.icon class="size-3" />
            {item.label}
          </div>
          <span class="text-foreground font-mono text-xs">{item.value}</span>
        </div>
      {/each}
    </div>
  </div>

  <Separator />

  <div class="px-4 py-3">
    <div class="text-muted-foreground flex items-center gap-2 text-xs font-medium">
      <FileCode class="size-3" />
      Harness
    </div>
    <div class="mt-2 space-y-1.5">
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Path</span>
        <span class="text-foreground font-mono">{workflow.harnessPath}</span>
      </div>
      <div class="flex items-center justify-between text-xs">
        <span class="text-muted-foreground">Version</span>
        <span class="text-foreground font-mono">v{workflow.version}</span>
      </div>
    </div>
  </div>
</div>
