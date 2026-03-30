<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { formatRelativeTime } from '$lib/utils'
  import { RefreshCw } from '@lucide/svelte'
  import type { MachineItem, MachineProbeResult, MachineSnapshot } from '../types'
  import {
    buildAuditRows,
    buildLevelCards,
    buildStatCards,
    checkedAtLabel,
    runtimeLabel,
    stateBadgeVariant,
    stateLabel,
    truthyLabel,
  } from './machine-health-panel-view'

  let {
    machine,
    snapshot,
    probe,
    loading = false,
    refreshing = false,
    onRefresh,
  }: {
    machine: MachineItem | null
    snapshot: MachineSnapshot | null
    probe: MachineProbeResult | null
    loading?: boolean
    refreshing?: boolean
    onRefresh?: () => void
  } = $props()

  const statCards = $derived(snapshot ? buildStatCards(snapshot) : [])
  const levelCards = $derived(snapshot ? buildLevelCards(snapshot) : [])
  const runtimeRows = $derived(snapshot?.agentEnvironment ?? [])
  const auditRows = $derived(snapshot ? buildAuditRows(snapshot) : [])
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
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
    <div class="flex items-center gap-2">
      {#if loading || refreshing}
        <Badge variant="outline">{refreshing ? 'Running checks…' : 'Refreshing…'}</Badge>
      {/if}
      <Button
        variant="outline"
        size="sm"
        class="gap-1.5"
        onclick={onRefresh}
        disabled={loading || refreshing}
      >
        <RefreshCw class="size-3.5" />
        {refreshing ? 'Running checks…' : 'Run checks'}
      </Button>
    </div>
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

    <div class="border-border bg-card rounded-xl border">
      <div class="border-border border-b px-4 py-3">
        <h4 class="text-foreground text-sm font-semibold">Monitor levels</h4>
        <p class="text-muted-foreground mt-1 text-xs">
          Manual refresh runs the same multi-level machine checks used by the orchestrator.
        </p>
      </div>
      <div class="grid gap-3 px-4 py-4 md:grid-cols-2 xl:grid-cols-3">
        {#each levelCards as level (level.id)}
          <div class="border-border rounded-xl border px-4 py-3">
            <div class="flex items-center justify-between gap-2">
              <p class="text-foreground text-sm font-medium">{level.label}</p>
              <Badge variant={stateBadgeVariant(level.state)}>{stateLabel(level.state)}</Badge>
            </div>
            <p class="text-foreground mt-3 text-sm">{level.value}</p>
            <p class="text-muted-foreground mt-1 text-xs">{level.meta}</p>
          </div>
        {/each}
      </div>
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

    {#if runtimeRows.length > 0}
      <div class="border-border bg-card rounded-xl border">
        <div class="border-border border-b px-4 py-3">
          <h4 class="text-foreground text-sm font-semibold">Runtime providers</h4>
          <p class="text-muted-foreground mt-1 text-xs">
            L4 runtime status captured {checkedAtLabel(snapshot.agentEnvironmentCheckedAt)}
          </p>
        </div>
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-border text-muted-foreground border-b text-left text-xs">
                <th class="px-4 py-2 font-medium">Runtime</th>
                <th class="px-4 py-2 font-medium">Installed</th>
                <th class="px-4 py-2 font-medium">Auth</th>
                <th class="px-4 py-2 font-medium">Ready</th>
                <th class="px-4 py-2 font-medium">Version</th>
              </tr>
            </thead>
            <tbody>
              {#each runtimeRows as runtime (runtime.name)}
                <tr class="border-border/60 border-b last:border-0">
                  <td class="px-4 py-3 font-medium">{runtimeLabel(runtime)}</td>
                  <td class="px-4 py-3 text-xs">{truthyLabel(runtime.installed)}</td>
                  <td class="px-4 py-3 text-xs">
                    {[runtime.authStatus, runtime.authMode].filter(Boolean).join(' · ') ||
                      'Unknown'}
                  </td>
                  <td class="px-4 py-3 text-xs">{truthyLabel(runtime.ready)}</td>
                  <td class="text-muted-foreground px-4 py-3 text-xs"
                    >{runtime.version ?? 'Unknown'}</td
                  >
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {/if}

    {#if snapshot.fullAudit}
      <div class="border-border bg-card rounded-xl border">
        <div class="border-border border-b px-4 py-3">
          <h4 class="text-foreground text-sm font-semibold">Tooling audit</h4>
          <p class="text-muted-foreground mt-1 text-xs">
            L5 audit captured {checkedAtLabel(snapshot.fullAudit.checkedAt)}
          </p>
        </div>
        <div class="grid gap-3 px-4 py-4 md:grid-cols-2">
          {#each auditRows as row (row.label)}
            <div class="border-border rounded-xl border px-4 py-3">
              <p class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
                {row.label}
              </p>
              <p class="text-foreground mt-2 text-sm font-semibold">{row.value}</p>
              <p class="text-muted-foreground mt-1 text-xs leading-5 whitespace-pre-wrap">
                {row.detail}
              </p>
            </div>
          {/each}
        </div>
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
