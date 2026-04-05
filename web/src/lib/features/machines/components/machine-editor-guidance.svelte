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
  import { Monitor, Terminal, ArrowLeftRight, Radio, Check } from '@lucide/svelte'

  const connectionModeOptions: {
    mode: MachineConnectionMode
    icon: typeof Monitor
    shortDesc: string
    keyTrait: string
  }[] = [
    {
      mode: 'local',
      icon: Monitor,
      shortDesc: 'Same host as the control plane',
      keyTrait: 'No network overhead',
    },
    {
      mode: 'ssh',
      icon: Terminal,
      shortDesc: 'Direct SSH to a remote host',
      keyTrait: 'Key-based auth',
    },
    {
      mode: 'ws_reverse',
      icon: ArrowLeftRight,
      shortDesc: 'Daemon connects back to OpenASE',
      keyTrait: 'NAT-friendly',
    },
    {
      mode: 'ws_listener',
      icon: Radio,
      shortDesc: 'OpenASE connects to machine endpoint',
      keyTrait: 'Custom endpoint',
    },
  ]

  const comparisonRows: { label: string; values: Record<MachineConnectionMode, string> }[] = [
    {
      label: 'Auth',
      values: {
        local: 'None',
        ssh: 'SSH key',
        ws_reverse: 'Daemon token',
        ws_listener: 'Endpoint TLS',
      },
    },
    {
      label: 'Firewall',
      values: {
        local: 'N/A',
        ssh: 'Inbound 22',
        ws_reverse: 'Outbound only',
        ws_listener: 'Inbound WS',
      },
    },
    {
      label: 'Setup',
      values: {
        local: 'Automatic',
        ssh: 'Key + user',
        ws_reverse: 'Install daemon',
        ws_listener: 'Run listener',
      },
    },
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

  let showComparison = $state(false)
</script>

<section class="space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Connection mode</h3>
      <p class="text-muted-foreground mt-0.5 text-xs">How OpenASE reaches this machine.</p>
    </div>
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground text-xs underline-offset-2 transition-colors hover:underline"
      onclick={() => (showComparison = !showComparison)}
    >
      {showComparison ? 'Hide comparison' : 'Compare modes'}
    </button>
  </div>

  <div class="grid gap-2 sm:grid-cols-2">
    {#each connectionModeOptions as option (option.mode)}
      {@const Icon = option.icon}
      {@const selected = connectionMode === option.mode}
      <button
        type="button"
        class={cn(
          'border-border bg-card hover:bg-muted/50 group relative rounded-lg border px-3.5 py-3 text-left transition-all',
          selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
        )}
        data-testid={`machine-connection-mode-${option.mode}`}
        onclick={() => onSelectMode?.(option.mode)}
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
                {machineConnectionModeLabel(option.mode)}
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

  {#if showComparison}
    <div class="border-border bg-card overflow-hidden rounded-lg border">
      <table class="w-full text-xs">
        <thead>
          <tr class="border-border border-b">
            <th class="text-muted-foreground px-3 py-2 text-left font-medium"></th>
            {#each connectionModeOptions as option (option.mode)}
              <th
                class={cn(
                  'px-3 py-2 text-center font-medium',
                  connectionMode === option.mode ? 'text-primary' : 'text-muted-foreground',
                )}
              >
                {machineConnectionModeLabel(option.mode)}
              </th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each comparisonRows as row (row.label)}
            <tr class="border-border/60 border-b last:border-0">
              <td class="text-muted-foreground px-3 py-2 font-medium">{row.label}</td>
              {#each connectionModeOptions as option (option.mode)}
                <td
                  class={cn(
                    'px-3 py-2 text-center',
                    connectionMode === option.mode
                      ? 'text-foreground font-medium'
                      : 'text-muted-foreground',
                  )}
                >
                  {row.values[option.mode]}
                </td>
              {/each}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <div class="border-border bg-card rounded-lg border px-3.5 py-3">
    <div class="flex flex-wrap items-center gap-2">
      <Badge variant="outline" class="text-[10px]">{modeGuide.label}</Badge>
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
        <span class="text-foreground ml-1">{modeGuide.requiredFields}</span>
      </div>
      <div>
        <span class="text-muted-foreground">Test:</span>
        <span class="text-foreground ml-1">{modeGuide.testSemantics}</span>
      </div>
    </div>
  </div>
</section>
