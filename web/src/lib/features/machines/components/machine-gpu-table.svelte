<script lang="ts">
  import { Badge } from '$ui/badge'
  import type { MachineSnapshot } from '../types'
  import {
    checkedAtLabel,
    levelState,
    stateBadgeVariant,
    stateLabel,
  } from './machine-health-panel-view'

  let { snapshot }: { snapshot: MachineSnapshot } = $props()

  const l3State = $derived(levelState(snapshot.monitor.l3))
</script>

<div class="border-border bg-card rounded-xl border">
  <div
    class="border-border flex flex-col gap-2 border-b px-4 py-3 sm:flex-row sm:items-center sm:justify-between"
  >
    <div class="flex flex-wrap items-center gap-2">
      <h4 class="text-foreground text-sm font-semibold">GPU inventory</h4>
      <Badge variant={stateBadgeVariant(l3State)}>{stateLabel(l3State)}</Badge>
      <Badge variant="secondary">{snapshot.gpus.length} GPU</Badge>
    </div>
    <span class="text-muted-foreground text-xs">
      {snapshot.gpuDispatchable ? 'Free memory available' : 'No GPU dispatchable'}
      · {checkedAtLabel(snapshot.monitor.l3?.checkedAt)}
    </span>
  </div>
  <div class="overflow-x-auto">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-border text-muted-foreground border-b text-left text-xs">
          <th class="px-4 py-2 font-medium">GPU</th>
          <th class="px-4 py-2 font-medium">Model</th>
          <th class="px-4 py-2 font-medium">Memory</th>
          <th class="px-4 py-2 text-right font-medium">Utilization</th>
        </tr>
      </thead>
      <tbody>
        {#each snapshot.gpus as gpu (gpu.index)}
          <tr class="border-border/60 border-b last:border-0">
            <td class="px-4 py-3 font-mono text-xs">{gpu.index}</td>
            <td class="px-4 py-3">{gpu.name}</td>
            <td class="text-muted-foreground px-4 py-3 text-xs">
              {gpu.memoryUsedGB.toFixed(1)} / {gpu.memoryTotalGB.toFixed(1)} GB
            </td>
            <td class="px-4 py-3 text-right text-xs">{gpu.utilizationPercent.toFixed(1)}%</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
