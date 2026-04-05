<script lang="ts">
  import { Badge } from '$ui/badge'
  import { cn } from '$lib/utils'
  import {
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionMessage,
    machineDetectionStatusLabel,
    machineModeGuide,
    machineReachabilityLabel,
    normalizeReachabilityMode,
  } from '../model'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type { MachineDraft, MachineItem, MachineReachabilityMode } from '../types'
  import { Monitor, ArrowLeftRight, Radio, Check } from '@lucide/svelte'

  const reachabilityOptions: {
    mode: MachineReachabilityMode
    icon: typeof Monitor
    shortDesc: string
    keyTrait: string
  }[] = [
    {
      mode: 'local',
      icon: Monitor,
      shortDesc: 'Same host as the control plane',
      keyTrait: 'No network boundary',
    },
    {
      mode: 'direct_connect',
      icon: Radio,
      shortDesc: 'Control plane can dial the machine',
      keyTrait: 'Listener plus optional SSH helper',
    },
    {
      mode: 'reverse_connect',
      icon: ArrowLeftRight,
      shortDesc: 'Machine daemon dials back to OpenASE',
      keyTrait: 'Outbound-only friendly',
    },
  ]

  let {
    machine,
    draft,
    onSelectReachability,
  }: {
    machine: MachineItem | null
    draft: MachineDraft
    onSelectReachability?: (mode: MachineReachabilityMode) => void
  } = $props()

  const reachabilityMode = $derived(
    normalizeReachabilityMode(draft.reachabilityMode, draft.host, machine?.connection_mode),
  )
  const reachabilityGuide = $derived(machineModeGuide(reachabilityMode))
  const detectionStatusLabel = $derived(machineDetectionStatusLabel(machine?.detection_status))
  const detectionBadgeClass = $derived(machineDetectionBadgeClass(machine?.detection_status))
  const detectedOSLabel = $derived(machineDetectedOSLabel(machine?.detected_os))
  const detectedArchLabel = $derived(machineDetectedArchLabel(machine?.detected_arch))
  const detectionSummary = $derived(machineDetectionMessage(machine, draft))
  const setupGuide = $derived(buildMachineSetupGuide({ machine, draft }))
</script>

<section class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">Connection topology</h3>
    <p class="text-muted-foreground mt-0.5 text-xs">
      Choose how OpenASE reaches the machine. Setup commands and helper flows follow from that
      topology.
    </p>
  </div>

  <div class="grid gap-2 sm:grid-cols-3">
    {#each reachabilityOptions as option (option.mode)}
      {@const Icon = option.icon}
      {@const selected = reachabilityMode === option.mode}
      <button
        type="button"
        class={cn(
          'border-border bg-card hover:bg-muted/50 group relative rounded-lg border px-3.5 py-3 text-left transition-all',
          selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
        )}
        data-testid={`machine-reachability-mode-${option.mode}`}
        onclick={() => onSelectReachability?.(option.mode)}
      >
        <div class="flex items-start gap-3">
          <div
            class={cn(
              'mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-md',
              selected ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground',
            )}
          >
            <Icon class="size-4" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-2">
              <span class="text-foreground text-sm font-medium">
                {machineReachabilityLabel(option.mode)}
              </span>
              {#if selected}
                <Check class="text-primary size-3.5" />
              {/if}
            </div>
            <p class="text-muted-foreground mt-0.5 text-xs leading-relaxed">{option.shortDesc}</p>
            <span
              class="text-muted-foreground mt-1 inline-block text-[10px] font-medium tracking-wider uppercase"
            >
              {option.keyTrait}
            </span>
          </div>
        </div>
      </button>
    {/each}
  </div>

  <div class="border-border bg-card rounded-lg border px-3.5 py-3">
    <div class="flex flex-wrap items-center gap-2">
      <Badge variant="outline" class="text-[10px]">{reachabilityGuide.label}</Badge>
      <Badge variant="outline" class="text-[10px]">{setupGuide.runtimeLabel}</Badge>
      <Badge variant="outline" class="text-[10px]">{setupGuide.helperLabel}</Badge>
      <Badge variant="outline" class={cn('text-[10px]', detectionBadgeClass)}>
        {detectionStatusLabel}
      </Badge>
      <Badge variant="secondary" class="text-[10px]">{detectedOSLabel}</Badge>
      <Badge variant="secondary" class="text-[10px]">{detectedArchLabel}</Badge>
    </div>
    <p class="text-muted-foreground mt-2 text-xs leading-relaxed">{detectionSummary}</p>
    <div class="mt-2 grid gap-x-6 gap-y-1 text-xs sm:grid-cols-2">
      <div>
        <span class="text-muted-foreground">Required:</span>
        <span class="text-foreground ml-1">{reachabilityGuide.requiredFields}</span>
      </div>
      <div>
        <span class="text-muted-foreground">State:</span>
        <span class="text-foreground ml-1">{setupGuide.stateSummary}</span>
      </div>
    </div>
  </div>

  <div class="grid gap-3 lg:grid-cols-2">
    <div class="border-border bg-card rounded-lg border px-3.5 py-3">
      <div class="space-y-1">
        <p class="text-foreground text-sm font-medium">{setupGuide.runtimeLabel}</p>
        <p class="text-muted-foreground text-xs leading-relaxed">{setupGuide.runtimeSummary}</p>
      </div>
      <div class="mt-3 space-y-1">
        <p class="text-foreground text-sm font-medium">{setupGuide.helperLabel}</p>
        <p class="text-muted-foreground text-xs leading-relaxed">{setupGuide.helperSummary}</p>
      </div>
    </div>

    <div class="border-border bg-card rounded-lg border px-3.5 py-3">
      <p class="text-foreground text-sm font-medium">Next step guidance</p>
      <ul class="text-muted-foreground mt-2 space-y-1.5 text-xs leading-relaxed">
        {#each setupGuide.nextSteps as step, index (`${step}-${index}`)}
          <li>{step}</li>
        {/each}
      </ul>
    </div>
  </div>

  {#if setupGuide.commands.length > 0}
    <div class="grid gap-3">
      {#each setupGuide.commands as command (command.title)}
        <div class="border-border bg-card rounded-lg border px-3.5 py-3">
          <p class="text-foreground text-sm font-medium">{command.title}</p>
          <p class="text-muted-foreground mt-1 text-xs leading-relaxed">{command.description}</p>
          <pre
            class="bg-muted/60 text-foreground mt-3 overflow-x-auto rounded-md px-3 py-2 text-xs whitespace-pre-wrap">{command.command}</pre>
        </div>
      {/each}
    </div>
  {/if}
</section>
