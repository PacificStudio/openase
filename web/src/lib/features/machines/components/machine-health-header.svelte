<script lang="ts">
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { formatRelativeTime } from '$lib/utils'
  import { RefreshCw } from '@lucide/svelte'
  import {
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionStatusLabel,
    machineReachabilityLabel,
  } from '../model'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type { MachineItem, MachineSnapshot } from '../types'

  let {
    machine,
    snapshot,
    loading = false,
    refreshing = false,
    onRefresh,
  }: {
    machine: MachineItem | null
    snapshot: MachineSnapshot | null
    loading?: boolean
    refreshing?: boolean
    onRefresh?: () => void
  } = $props()

  const setupGuide = $derived(buildMachineSetupGuide({ machine, snapshot }))
</script>

<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
  <div class="min-w-0 space-y-3">
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

    {#if machine}
      <div class="flex flex-wrap items-center gap-2">
        <Badge variant="outline">{machineReachabilityLabel(machine.reachability_mode)}</Badge>
        <Badge variant="outline">{setupGuide.runtimeLabel}</Badge>
        <Badge variant="outline">{setupGuide.helperLabel}</Badge>
        <Badge variant="secondary">{machineDetectedOSLabel(machine.detected_os)}</Badge>
        <Badge variant="secondary">{machineDetectedArchLabel(machine.detected_arch)}</Badge>
        <Badge variant="outline" class={machineDetectionBadgeClass(machine.detection_status)}>
          {machineDetectionStatusLabel(machine.detection_status)}
        </Badge>
      </div>
      <p class="text-muted-foreground max-w-2xl text-xs">
        {setupGuide.stateSummary}
      </p>
      <div class="text-muted-foreground flex flex-wrap items-center gap-2 text-[11px]">
        {#if machine.reachability_mode === 'direct_connect' && machine.advertised_endpoint}
          <span class="truncate font-mono">{machine.advertised_endpoint}</span>
        {/if}
        {#if machine.reachability_mode === 'reverse_connect'}
          <span>{setupGuide.stateLabel}</span>
        {/if}
      </div>
    {/if}
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
