<script lang="ts">
  import { Badge } from '$ui/badge'
  import { cn } from '$lib/utils'
  import {
    machineConnectionModeLabel,
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionMessage,
    machineDetectionStatusLabel,
    machineModeGuide,
    normalizeConnectionMode,
  } from '../model'
  import type { MachineConnectionMode, MachineDraft, MachineItem } from '../types'

  const connectionModeOptions: MachineConnectionMode[] = [
    'local',
    'ssh',
    'ws_reverse',
    'ws_listener',
  ]

  let {
    machine,
    draft,
    onSelectMode,
  }: {
    machine: MachineItem | null
    draft: MachineDraft
    onSelectMode?: (mode: MachineConnectionMode) => void
  } = $props()

  const connectionMode = $derived(normalizeConnectionMode(draft.connectionMode, draft.host))
  const modeGuide = $derived(machineModeGuide(connectionMode))
  const detectionStatusLabel = $derived(machineDetectionStatusLabel(machine?.detection_status))
  const detectionBadgeClass = $derived(machineDetectionBadgeClass(machine?.detection_status))
  const detectedOSLabel = $derived(machineDetectedOSLabel(machine?.detected_os))
  const detectedArchLabel = $derived(machineDetectedArchLabel(machine?.detected_arch))
  const detectionSummary = $derived(machineDetectionMessage(machine, draft))
</script>

<section class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">Connection mode</h3>
    <p class="text-muted-foreground mt-1 text-xs">
      Choose how OpenASE reaches this machine before filling transport-specific fields.
    </p>
  </div>

  <div class="grid gap-2 sm:grid-cols-2">
    {#each connectionModeOptions as option (option)}
      <button
        type="button"
        class={cn(
          'border-border bg-card hover:bg-muted/50 rounded-xl border px-4 py-3 text-left transition-colors',
          connectionMode === option && 'border-primary bg-primary/6 ring-primary/20 ring-1',
        )}
        data-testid={`machine-connection-mode-${option}`}
        onclick={() => onSelectMode?.(option)}
      >
        <div class="flex items-center justify-between gap-2">
          <span class="text-foreground text-sm font-medium">
            {machineConnectionModeLabel(option)}
          </span>
          {#if connectionMode === option}
            <Badge variant="secondary" class="text-[10px]">Selected</Badge>
          {/if}
        </div>
        <p class="text-muted-foreground mt-1 text-xs">{machineModeGuide(option).summary}</p>
      </button>
    {/each}
  </div>

  <div class="border-border bg-card space-y-3 rounded-xl border px-4 py-4">
    <div class="flex flex-wrap items-center gap-2">
      <Badge variant="outline">{modeGuide.label}</Badge>
      <Badge variant="outline" class={detectionBadgeClass}>
        {detectionStatusLabel}
      </Badge>
      <Badge variant="secondary">{detectedOSLabel}</Badge>
      <Badge variant="secondary">{detectedArchLabel}</Badge>
    </div>

    <p class="text-foreground text-sm">{detectionSummary}</p>

    <dl class="grid gap-3 text-sm md:grid-cols-2">
      <div>
        <dt class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
          Required fields
        </dt>
        <dd class="text-foreground mt-1">{modeGuide.requiredFields}</dd>
      </div>
      <div>
        <dt class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">Install path</dt>
        <dd class="text-foreground mt-1">{modeGuide.installMethod}</dd>
      </div>
      <div>
        <dt class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">
          Connection test
        </dt>
        <dd class="text-foreground mt-1">{modeGuide.testSemantics}</dd>
      </div>
      <div>
        <dt class="text-muted-foreground text-[11px] tracking-[0.12em] uppercase">Common errors</dt>
        <dd class="text-foreground mt-1">{modeGuide.commonErrors}</dd>
      </div>
    </dl>
  </div>
</section>
