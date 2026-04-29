<script lang="ts">
  import { ArrowLeftRight, ArrowRight, Check, KeyRound, Monitor, Radio } from '@lucide/svelte'
  import { slide } from 'svelte/transition'
  import type { Component } from 'svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import type { TranslationKey } from '$lib/i18n'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type {
    MachineDraft,
    MachineItem,
    MachineModeGuide,
    MachineReachabilityMode,
  } from '../types'
  import {
    detectedPlatformFromSnapshot,
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionMessage,
    machineDetectionStatusLabel,
    machineModeGuide,
    machineReachabilityLabel,
    normalizeReachabilityMode,
    parseMachineSnapshot,
  } from '../model'
  import { machineErrorMessage, runMachineSSHBootstrap } from './machines-page-api'
  import MachineEditorGuidanceSetup from './machine-editor-guidance-setup.svelte'

  type WsStrategy = 'direct-open' | 'ssh-install-listener' | 'reverse'
  type WsOption = {
    strategy: WsStrategy
    mode: MachineReachabilityMode
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    icon: Component<any>
    titleKey: TranslationKey
    descKey: TranslationKey
  }

  const wsOptions: WsOption[] = [
    {
      strategy: 'direct-open',
      mode: 'direct_connect',
      icon: Radio,
      titleKey: 'machines.machineEditorGuidance.progressive.q2.directOpen.title',
      descKey: 'machines.machineEditorGuidance.progressive.q2.directOpen.desc',
    },
    {
      strategy: 'ssh-install-listener',
      mode: 'direct_connect',
      icon: KeyRound,
      titleKey: 'machines.machineEditorGuidance.progressive.q2.sshInstallListener.title',
      descKey: 'machines.machineEditorGuidance.progressive.q2.sshInstallListener.desc',
    },
    {
      strategy: 'reverse',
      mode: 'reverse_connect',
      icon: ArrowLeftRight,
      titleKey: 'machines.machineEditorGuidance.progressive.q2.reverse.title',
      descKey: 'machines.machineEditorGuidance.progressive.q2.reverse.desc',
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

  const reachabilityMode = $derived(normalizeReachabilityMode(draft.reachabilityMode, draft.host))
  const reachabilityGuide = $derived<MachineModeGuide>(machineModeGuide(reachabilityMode))

  type LocationAnswer = 'local' | 'remote' | null
  let locationAnswer = $state<LocationAnswer>(null)
  let wsStrategy = $state<WsStrategy | null>(null)

  $effect(() => {
    if (!machine) return
    if (locationAnswer === null) {
      locationAnswer = reachabilityMode === 'local' ? 'local' : 'remote'
    }
    if (wsStrategy === null && reachabilityMode !== 'local') {
      wsStrategy =
        reachabilityMode === 'direct_connect'
          ? 'direct-open'
          : reachabilityMode === 'reverse_connect'
            ? 'reverse'
            : null
    }
  })

  function pickLocation(answer: LocationAnswer) {
    locationAnswer = answer
    if (answer === 'local') {
      wsStrategy = null
      onSelectReachability?.('local')
    }
  }

  function pickStrategy(option: WsOption) {
    wsStrategy = option.strategy
    onSelectReachability?.(option.mode)
  }

  function resetFlow() {
    locationAnswer = null
    wsStrategy = null
  }

  const flowComplete = $derived(locationAnswer === 'local' || wsStrategy !== null)
  const detectionStatusLabel = $derived(machineDetectionStatusLabel(machine?.detection_status))
  const detectionBadgeClass = $derived(machineDetectionBadgeClass(machine?.detection_status))
  const detectedPlatform = $derived(
    detectedPlatformFromSnapshot(parseMachineSnapshot(machine?.resources)),
  )
  const detectedOSLabel = $derived(
    machineDetectedOSLabel(machine?.detected_os ?? detectedPlatform.os),
  )
  const detectedArchLabel = $derived(
    machineDetectedArchLabel(machine?.detected_arch ?? detectedPlatform.arch),
  )
  const detectionSummary = $derived(machineDetectionMessage(machine, draft))
  const setupGuide = $derived(buildMachineSetupGuide({ machine, draft }))
  const bootstrapTopology = $derived(
    wsStrategy === 'ssh-install-listener'
      ? 'remote-listener'
      : wsStrategy === 'reverse'
        ? 'reverse-connect'
        : null,
  )
  const canRunBootstrap = $derived(Boolean(machine?.id && bootstrapTopology))

  let bootstrapRunning = $state(false)
  let bootstrapResult = $state<MachineSSHBootstrapResult | null>(null)
  let bootstrapError = $state('')

  async function handleRunBootstrap() {
    if (!machine?.id || !bootstrapTopology) return
    bootstrapRunning = true
    bootstrapResult = null
    bootstrapError = ''
    try {
      bootstrapResult = await runMachineSSHBootstrap(machine.id, { topology: bootstrapTopology })
      toastStore.success(
        bootstrapResult.summary ||
          i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.successFallback'),
      )
    } catch (caughtError) {
      bootstrapError = machineErrorMessage(
        caughtError,
        i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.failureFallback'),
      )
      toastStore.error(bootstrapError)
    } finally {
      bootstrapRunning = false
    }
  }
</script>

<section class="space-y-4">
  <div>
    <h3 class="text-foreground text-sm font-semibold">
      {i18nStore.t('machines.machineEditorGuidance.heading.connectionTopology')}
    </h3>
    <p class="text-muted-foreground mt-0.5 text-xs">
      {i18nStore.t('machines.machineEditorGuidance.description.connectionTopology')}
    </p>
  </div>

  <div class="border-border bg-card rounded-lg border px-3.5 py-3">
    <div class="flex items-center gap-2">
      <span
        class="bg-primary/10 text-primary flex size-5 items-center justify-center rounded-full text-[10px] font-semibold"
      >
        1
      </span>
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineEditorGuidance.progressive.q1.title')}
      </p>
    </div>
    <div class="mt-2.5 grid gap-2 sm:grid-cols-2">
      {#each [{ answer: 'local' as const, mode: 'local' as MachineReachabilityMode, icon: Monitor, descKey: 'machines.machineEditorGuidance.progressive.q1.local' as TranslationKey }, { answer: 'remote' as const, mode: null, icon: ArrowRight, descKey: 'machines.machineEditorGuidance.progressive.q1.remote' as TranslationKey }] as option (option.answer)}
        {@const Icon = option.icon}
        {@const selected = locationAnswer === option.answer}
        <button
          type="button"
          class={cn(
            'border-border bg-card hover:bg-muted/50 rounded-lg border px-3 py-2.5 text-left transition-all',
            selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
          )}
          data-testid={`machine-location-${option.answer}`}
          onclick={() => pickLocation(option.answer)}
        >
          <div class="flex items-center gap-2">
            <Icon class={cn('size-4', selected ? 'text-primary' : 'text-muted-foreground')} />
            <span class="text-foreground text-sm font-medium">{i18nStore.t(option.descKey)}</span>
            {#if selected}
              <Check class="text-primary ml-auto size-3.5" />
            {/if}
          </div>
          {#if option.mode}
            <p class="text-muted-foreground mt-1 text-[11px]">
              {machineReachabilityLabel(option.mode)}
            </p>
          {/if}
        </button>
      {/each}
    </div>
  </div>

  {#if locationAnswer === 'remote'}
    <div
      class="border-border bg-card rounded-lg border px-3.5 py-3"
      transition:slide={{ duration: 220 }}
    >
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <span
            class="bg-primary/10 text-primary flex size-5 items-center justify-center rounded-full text-[10px] font-semibold"
          >
            2
          </span>
          <p class="text-foreground text-sm font-medium">
            {i18nStore.t('machines.machineEditorGuidance.progressive.q2.title')}
          </p>
        </div>
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground text-[11px] underline-offset-2 hover:underline"
          onclick={resetFlow}
        >
          {i18nStore.t('machines.machineEditorGuidance.progressive.reset')}
        </button>
      </div>
      <div class="mt-2.5 grid gap-2 sm:grid-cols-3">
        {#each wsOptions as option (option.strategy)}
          {@const Icon = option.icon}
          {@const selected = wsStrategy === option.strategy}
          <button
            type="button"
            class={cn(
              'border-border bg-card hover:bg-muted/50 rounded-lg border px-3 py-2.5 text-left transition-all',
              selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
            )}
            data-testid={`machine-ws-strategy-${option.strategy}`}
            onclick={() => pickStrategy(option)}
          >
            <div class="flex items-start gap-2.5">
              <div
                class={cn(
                  'mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md',
                  selected ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground',
                )}
              >
                <Icon class="size-3.5" />
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2">
                  <span class="text-foreground text-sm font-medium">
                    {i18nStore.t(option.titleKey)}
                  </span>
                  {#if selected}
                    <Check class="text-primary size-3.5" />
                  {/if}
                </div>
                <p class="text-muted-foreground mt-0.5 text-[11px] leading-relaxed">
                  {machineReachabilityLabel(option.mode)} · {i18nStore.t(option.descKey)}
                </p>
              </div>
            </div>
          </button>
        {/each}
      </div>
    </div>
  {/if}

  {#if flowComplete}
    <MachineEditorGuidanceSetup
      {reachabilityGuide}
      {setupGuide}
      {detectionBadgeClass}
      {detectionStatusLabel}
      {detectedOSLabel}
      {detectedArchLabel}
      {detectionSummary}
      {canRunBootstrap}
      {bootstrapRunning}
      {bootstrapResult}
      {bootstrapError}
      onRunBootstrap={handleRunBootstrap}
    />
  {/if}
</section>
