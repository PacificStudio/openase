<script lang="ts">
  import { cn, formatCurrency } from '$lib/utils'
  import { DollarSign, TrendingUp } from '@lucide/svelte'

  let {
    todayCost,
    weekCost,
    topProject,
    topAgent,
    class: className = '',
  }: {
    todayCost: number
    weekCost: number
    topProject?: { name: string, cost: number }
    topAgent?: { name: string, cost: number }
    class?: string
  } = $props()
</script>

<div class={cn('rounded-md border border-border bg-card', className)}>
  <div class="flex items-center justify-between border-b border-border px-4 py-3">
    <h3 class="text-sm font-medium text-foreground">Cost Snapshot</h3>
    <DollarSign class="size-4 text-muted-foreground" />
  </div>

  <div class="p-4 space-y-4">
    <div class="grid grid-cols-2 gap-4">
      <div>
        <span class="text-xs text-muted-foreground">Today</span>
        <p class="text-lg font-semibold text-foreground">{formatCurrency(todayCost)}</p>
      </div>
      <div>
        <span class="text-xs text-muted-foreground">This week</span>
        <p class="text-lg font-semibold text-foreground">{formatCurrency(weekCost)}</p>
      </div>
    </div>

    <div class="h-px bg-border"></div>

    {#if topProject}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TrendingUp class="size-3 text-muted-foreground" />
          <span class="text-xs text-muted-foreground">Top project</span>
        </div>
        <div class="text-right">
          <span class="text-sm text-foreground">{topProject.name}</span>
          <span class="ml-2 text-xs text-muted-foreground">{formatCurrency(topProject.cost)}</span>
        </div>
      </div>
    {/if}

    {#if topAgent}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TrendingUp class="size-3 text-muted-foreground" />
          <span class="text-xs text-muted-foreground">Top agent</span>
        </div>
        <div class="text-right">
          <span class="text-sm text-foreground">{topAgent.name}</span>
          <span class="ml-2 text-xs text-muted-foreground">{formatCurrency(topAgent.cost)}</span>
        </div>
      </div>
    {/if}
  </div>
</div>
