<script lang="ts">
  import { Badge } from '$ui/badge'
  import * as Tooltip from '$ui/tooltip'
  import { cn, formatRelativeTime } from '$lib/utils'
  import MachineResourceBars from './machine-resource-bars.svelte'
  import MachineRowCardActions from './machine-row-card-actions.svelte'
  import {
    isLocalMachine,
    machineStatusBadgeClass,
    machineStatusDescription,
    machineStatusLabel,
    parseMachineSnapshot,
  } from '../model'
  import type { MachineItem } from '../types'

  type DetailChip = {
    label: string
    title: string
    description: string
    className?: string
  }

  type ResourceBar = {
    key: string
    label: string
    percent: number
    summary: string
    detail: string
    barClass: string
  }

  let {
    machine,
    selected = false,
    resetEnabled = false,
    testing = false,
    deleting = false,
    onOpen,
    onTest,
    onReset,
    onDelete,
  }: {
    machine: MachineItem
    selected?: boolean
    resetEnabled?: boolean
    testing?: boolean
    deleting?: boolean
    onOpen?: () => void
    onTest?: () => void
    onReset?: () => void
    onDelete?: () => void
  } = $props()

  const snapshot = $derived(parseMachineSnapshot(machine.resources))
  const localMachine = $derived(isLocalMachine(machine))

  const detailChips = $derived.by((): DetailChip[] => {
    const chips: DetailChip[] = [
      {
        label: machineStatusLabel(machine.status),
        title: 'Machine status',
        description: machineStatusDescription(machine.status),
        className: machineStatusBadgeClass(machine.status),
      },
      {
        label: localMachine ? 'Local runner' : 'Remote SSH',
        title: 'Execution mode',
        description: localMachine
          ? 'Runs on the same host as the OpenASE service.'
          : `OpenASE connects to ${machine.host}:${machine.port} using SSH.`,
      },
      {
        label: machine.last_heartbeat_at
          ? formatRelativeTime(machine.last_heartbeat_at)
          : 'No heartbeat',
        title: 'Heartbeat',
        description: machine.last_heartbeat_at
          ? `Last heartbeat received at ${new Date(machine.last_heartbeat_at).toLocaleString()}.`
          : 'No heartbeat has been recorded for this machine yet.',
      },
    ]

    if (snapshot?.monitorErrors.length) {
      chips.push({
        label: `${snapshot.monitorErrors.length} warning${snapshot.monitorErrors.length === 1 ? '' : 's'}`,
        title: 'Monitor warnings',
        description: snapshot.monitorErrors.join('\n'),
        className: 'border-amber-500/30 bg-amber-500/12 text-amber-700',
      })
    }

    return chips
  })

  const resourceBars = $derived.by((): ResourceBar[] => {
    const gpuAverage =
      snapshot && snapshot.gpus.length > 0
        ? snapshot.gpus.reduce((total, gpu) => total + gpu.utilizationPercent, 0) /
          snapshot.gpus.length
        : undefined

    return [
      {
        key: 'cpu',
        label: 'CPU',
        percent: snapshot?.cpuUsagePercent ?? 0,
        summary:
          snapshot?.cpuUsagePercent === undefined
            ? 'Pending'
            : `${snapshot.cpuUsagePercent.toFixed(0)}%`,
        detail:
          snapshot?.cpuUsagePercent === undefined
            ? 'CPU usage has not been collected yet.'
            : `${snapshot.cpuUsagePercent.toFixed(1)}% CPU in use across ${snapshot.cpuCores?.toFixed(0) ?? '?'} cores.`,
        barClass: toneForPercent(snapshot?.cpuUsagePercent),
      },
      {
        key: 'memory',
        label: 'Memory',
        percent:
          snapshot?.memoryTotalGB && snapshot.memoryUsedGB !== undefined
            ? clampPercent((snapshot.memoryUsedGB / snapshot.memoryTotalGB) * 100)
            : 0,
        summary:
          snapshot?.memoryUsedGB === undefined
            ? 'Pending'
            : `${snapshot.memoryUsedGB.toFixed(1)} / ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB`,
        detail:
          snapshot?.memoryUsedGB === undefined
            ? 'Memory usage has not been collected yet.'
            : `${snapshot.memoryAvailableGB?.toFixed(1) ?? '?'} GB free out of ${snapshot.memoryTotalGB?.toFixed(1) ?? '?'} GB total.`,
        barClass: toneForPercent(
          snapshot?.memoryTotalGB && snapshot.memoryUsedGB !== undefined
            ? (snapshot.memoryUsedGB / snapshot.memoryTotalGB) * 100
            : undefined,
        ),
      },
      {
        key: 'disk',
        label: 'Disk',
        percent:
          snapshot?.diskTotalGB && snapshot.diskAvailableGB !== undefined
            ? clampPercent(
                ((snapshot.diskTotalGB - snapshot.diskAvailableGB) / snapshot.diskTotalGB) * 100,
              )
            : 0,
        summary:
          snapshot?.diskAvailableGB === undefined
            ? 'Pending'
            : `${snapshot.diskAvailableGB.toFixed(1)} GB free`,
        detail:
          snapshot?.diskAvailableGB === undefined
            ? 'Disk usage has not been collected yet.'
            : `${snapshot.diskAvailableGB.toFixed(1)} GB free out of ${snapshot.diskTotalGB?.toFixed(1) ?? '?'} GB.`,
        barClass: toneForPercent(
          snapshot?.diskTotalGB && snapshot.diskAvailableGB !== undefined
            ? ((snapshot.diskTotalGB - snapshot.diskAvailableGB) / snapshot.diskTotalGB) * 100
            : undefined,
        ),
      },
      {
        key: 'gpu',
        label: 'GPU',
        percent: clampPercent(gpuAverage ?? 0),
        summary: snapshot?.gpus.length
          ? `${snapshot.gpus.length} GPU${snapshot.gpus.length === 1 ? '' : 's'}`
          : 'No GPU',
        detail: snapshot?.gpus.length
          ? `Average utilization ${gpuAverage?.toFixed(1) ?? '0.0'}%. ${snapshot.gpuDispatchable ? 'At least one GPU is dispatchable.' : 'No GPU is currently dispatchable.'}`
          : 'This machine has no GPU inventory in the latest snapshot.',
        barClass: snapshot?.gpuDispatchable ? 'bg-sky-500' : 'bg-slate-400',
      },
    ]
  })

  function clampPercent(value: number) {
    if (!Number.isFinite(value)) return 0
    return Math.max(0, Math.min(100, value))
  }

  function toneForPercent(value: number | undefined) {
    if (value === undefined) return 'bg-slate-400'
    if (value >= 85) return 'bg-rose-500'
    if (value >= 65) return 'bg-amber-500'
    return 'bg-emerald-500'
  }
</script>

<article
  data-testid={`machine-card-${machine.id}`}
  class={cn(
    'border-border bg-card hover:bg-muted/20 rounded-2xl border p-4 transition-colors',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,18rem)_minmax(0,1fr)_auto] xl:items-center">
    <div
      data-testid={`machine-open-${machine.id}`}
      class="min-w-0 cursor-pointer space-y-3"
      role="button"
      tabindex="0"
      onclick={onOpen}
      onkeydown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onOpen?.()
        }
      }}
    >
      <div class="space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-foreground truncate text-base font-semibold">{machine.name}</h3>
          {#if localMachine}
            <Badge variant="secondary" class="text-[10px]">local</Badge>
          {/if}
        </div>
        <p class="text-muted-foreground truncate font-mono text-xs">
          {machine.host}:{machine.port}
        </p>
      </div>

      <div class="flex flex-wrap gap-2">
        {#each detailChips as chip (chip.title + chip.label)}
          <Tooltip.Root>
            <Tooltip.Trigger>
              {#snippet child({ props })}
                <Badge
                  {...props}
                  variant="outline"
                  class={cn('cursor-help whitespace-nowrap', chip.className)}
                >
                  {chip.label}
                </Badge>
              {/snippet}
            </Tooltip.Trigger>
            <Tooltip.Content
              side="top"
              sideOffset={6}
              class="bg-popover text-popover-foreground border-border max-w-[22rem] rounded-lg border p-3 shadow-xl"
              arrowClasses="bg-popover fill-popover"
            >
              <div class="space-y-1">
                <div class="text-sm font-medium">{chip.title}</div>
                <p class="text-muted-foreground text-xs leading-5 whitespace-pre-wrap">
                  {chip.description}
                </p>
              </div>
            </Tooltip.Content>
          </Tooltip.Root>
        {/each}
      </div>
    </div>

    <div
      data-testid={`machine-resources-${machine.id}`}
      role="button"
      tabindex="0"
      onclick={onOpen}
      onkeydown={(event) => {
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onOpen?.()
        }
      }}
    >
      <MachineResourceBars bars={resourceBars} />
    </div>

    <MachineRowCardActions
      machineName={machine.name}
      {localMachine}
      {resetEnabled}
      {testing}
      {deleting}
      {onOpen}
      {onTest}
      {onReset}
      {onDelete}
    />
  </div>
</article>
