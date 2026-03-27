<script lang="ts">
  import { Badge } from '$ui/badge'
  import { formatRelativeTime } from '$lib/utils'
  import type { MachineItem, MachineProbeResult, MachineSnapshot } from '../types'

  let {
    machine,
    snapshot,
    probe,
    loading = false,
  }: {
    machine: MachineItem | null
    snapshot: MachineSnapshot | null
    probe: MachineProbeResult | null
    loading?: boolean
  } = $props()

  const statCards = $derived.by(() => {
    if (!snapshot) {
      return []
    }

    return [
      {
        label: 'Reachability',
        value:
          snapshot.monitor.l1?.reachable === undefined
            ? 'Unknown'
            : snapshot.monitor.l1.reachable
              ? 'Reachable'
              : 'Unavailable',
        meta: snapshot.monitor.l1?.latencyMs
          ? `${snapshot.monitor.l1.latencyMs.toFixed(0)} ms`
          : (snapshot.transport ?? 'No transport'),
      },
      {
        label: 'CPU',
        value:
          snapshot.cpuUsagePercent === undefined
            ? 'Pending'
            : `${snapshot.cpuUsagePercent.toFixed(1)}%`,
        meta:
          snapshot.cpuCores === undefined
            ? 'No core count'
            : `${snapshot.cpuCores.toFixed(0)} cores`,
      },
      {
        label: 'Memory',
        value:
          snapshot.memoryUsedGB === undefined
            ? 'Pending'
            : `${snapshot.memoryUsedGB.toFixed(1)} / ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB`,
        meta:
          snapshot.memoryAvailableGB === undefined
            ? 'No free memory data'
            : `${snapshot.memoryAvailableGB.toFixed(1)} GB free`,
      },
      {
        label: 'Disk',
        value:
          snapshot.diskAvailableGB === undefined
            ? 'Pending'
            : `${snapshot.diskAvailableGB.toFixed(1)} GB free`,
        meta:
          snapshot.gpuDispatchable === undefined
            ? 'GPU dispatch unknown'
            : snapshot.gpuDispatchable
              ? 'GPU dispatchable'
              : 'GPU dispatch blocked',
      },
    ]
  })
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Health snapshot</h3>
      <p class="text-muted-foreground mt-1 text-xs">
        {#if snapshot?.checkedAt}
          Snapshot collected {formatRelativeTime(snapshot.checkedAt)} and reflects detected machine state.
        {:else if machine?.last_heartbeat_at}
          Last heartbeat {formatRelativeTime(machine.last_heartbeat_at)}.
        {:else}
          No heartbeat has been recorded yet.
        {/if}
      </p>
    </div>
    {#if loading}
      <Badge variant="outline">Refreshing…</Badge>
    {/if}
  </div>

  {#if !snapshot}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-8 text-center text-sm"
    >
      No monitor snapshot is available for this machine yet.
    </div>
  {:else}
    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
      {#each statCards as card (card.label)}
        <div class="border-border bg-card rounded-xl border px-4 py-3">
          <p class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">{card.label}</p>
          <p class="text-foreground mt-2 text-sm font-semibold">{card.value}</p>
          <p class="text-muted-foreground mt-1 text-xs">{card.meta}</p>
        </div>
      {/each}
    </div>

    {#if snapshot.monitorErrors.length > 0}
      <div class="border-destructive/40 bg-destructive/10 rounded-xl border px-4 py-3">
        <p class="text-destructive text-sm font-medium">Monitor warnings</p>
        <ul class="text-destructive mt-2 space-y-1 text-xs">
          {#each snapshot.monitorErrors as error, index (`${error}-${index}`)}
            <li>{error}</li>
          {/each}
        </ul>
      </div>
    {/if}

    {#if snapshot.gpus.length > 0}
      <div class="border-border bg-card rounded-xl border">
        <div class="border-border flex items-center justify-between border-b px-4 py-3">
          <div>
            <h4 class="text-foreground text-sm font-semibold">GPU inventory</h4>
            <p class="text-muted-foreground mt-1 text-xs">
              {snapshot.gpuDispatchable
                ? 'At least one GPU has free memory.'
                : 'No GPU is currently dispatchable.'}
            </p>
          </div>
          <Badge variant="secondary">{snapshot.gpus.length} GPU</Badge>
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
    {/if}
  {/if}

  {#if probe}
    <div class="border-border bg-card rounded-xl border px-4 py-4">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h4 class="text-foreground text-sm font-semibold">Latest connection test</h4>
          <p class="text-muted-foreground mt-1 text-xs">
            {formatRelativeTime(probe.checked_at)}
          </p>
        </div>
        <Badge variant="outline">{probe.transport}</Badge>
      </div>

      <pre
        class="bg-muted/50 text-foreground mt-4 overflow-x-auto rounded-lg px-3 py-3 text-xs whitespace-pre-wrap">{probe.output ||
          'Probe completed without output.'}</pre>
    </div>
  {/if}
</div>
