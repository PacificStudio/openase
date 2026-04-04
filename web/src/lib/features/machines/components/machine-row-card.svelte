<script lang="ts">
  import { Badge } from '$ui/badge'
  import * as Tooltip from '$ui/tooltip'
  import { cn } from '$lib/utils'
  import MachineResourceBars from './machine-resource-bars.svelte'
  import MachineRowCardActions from './machine-row-card-actions.svelte'
  import { buildResourceBars, buildStatusDots, type StatusDot } from './machine-row-card-view'
  import {
    isLocalMachine,
    machineConnectionModeLabel,
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionStatusLabel,
    parseMachineSnapshot,
  } from '../model'
  import type { MachineItem } from '../types'

  const dotColorClass: Record<StatusDot['color'], string> = {
    green: 'bg-emerald-500',
    red: 'bg-rose-500',
    amber: 'bg-amber-500',
    gray: 'bg-slate-400',
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
  const connectionModeLabel = $derived(machineConnectionModeLabel(machine.connection_mode))
  const platformLabel = $derived(
    `${machineDetectedOSLabel(machine.detected_os)} / ${machineDetectedArchLabel(machine.detected_arch)}`,
  )
  const detectionLabel = $derived(machineDetectionStatusLabel(machine.detection_status))
  const detectionBadgeClass = $derived(machineDetectionBadgeClass(machine.detection_status))

  const statusDots = $derived.by((): StatusDot[] => buildStatusDots(machine, snapshot))
  const resourceBars = $derived.by(() => buildResourceBars(snapshot))
</script>

<article
  data-testid={`machine-card-${machine.id}`}
  class={cn(
    'border-border bg-card hover:bg-muted/20 hover-lift rounded-2xl border p-4 transition-colors',
    selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
  )}
>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,18rem)_minmax(0,1fr)_auto] xl:items-start">
    <div data-testid={`machine-open-${machine.id}`} class="min-w-0 space-y-3">
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
        <div class="mt-2 flex flex-wrap items-center gap-1.5">
          <Badge variant="outline" class="text-[10px]">{connectionModeLabel}</Badge>
          <Badge variant="secondary" class="text-[10px]">{platformLabel}</Badge>
          <Badge variant="outline" class={cn('text-[10px]', detectionBadgeClass)}>
            {detectionLabel}
          </Badge>
        </div>
      </div>

      <div class="flex items-center gap-1.5">
        {#each statusDots as dot (dot.key)}
          <Tooltip.Root>
            <Tooltip.Trigger>
              {#snippet child({ props })}
                <span
                  {...props}
                  class={cn('size-2 cursor-help rounded-full', dotColorClass[dot.color])}
                ></span>
              {/snippet}
            </Tooltip.Trigger>
            <Tooltip.Content side="top" sideOffset={6}>
              <span class="text-xs font-medium">{dot.label}</span>
              <span class="text-muted-foreground text-xs"> · {dot.description}</span>
            </Tooltip.Content>
          </Tooltip.Root>
        {/each}
      </div>
    </div>

    <div data-testid={`machine-resources-${machine.id}`}>
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
