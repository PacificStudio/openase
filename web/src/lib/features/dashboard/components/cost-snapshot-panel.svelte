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
    topProject?: { name: string; cost: number }
    topAgent?: { name: string; cost: number }
    class?: string
  } = $props()
</script>

<div class={cn('border-border bg-card rounded-md border', className)}>
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <h3 class="text-foreground text-sm font-medium">Cost Snapshot</h3>
    <DollarSign class="text-muted-foreground size-4" />
  </div>

  <div class="space-y-4 p-4">
    <div class="grid grid-cols-2 gap-4">
      <div>
        <span class="text-muted-foreground text-xs">Today</span>
        <p class="text-foreground text-lg font-semibold">{formatCurrency(todayCost)}</p>
      </div>
      <div>
        <span class="text-muted-foreground text-xs">This week</span>
        <p class="text-foreground text-lg font-semibold">{formatCurrency(weekCost)}</p>
      </div>
    </div>

    <div class="bg-border h-px"></div>

    {#if topProject}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TrendingUp class="text-muted-foreground size-3" />
          <span class="text-muted-foreground text-xs">Top project</span>
        </div>
        <div class="text-right">
          <span class="text-foreground text-sm">{topProject.name}</span>
          <span class="text-muted-foreground ml-2 text-xs">{formatCurrency(topProject.cost)}</span>
        </div>
      </div>
    {/if}

    {#if topAgent}
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <TrendingUp class="text-muted-foreground size-3" />
          <span class="text-muted-foreground text-xs">Top agent</span>
        </div>
        <div class="text-right">
          <span class="text-foreground text-sm">{topAgent.name}</span>
          <span class="text-muted-foreground ml-2 text-xs">{formatCurrency(topAgent.cost)}</span>
        </div>
      </div>
    {/if}
  </div>
</div>
