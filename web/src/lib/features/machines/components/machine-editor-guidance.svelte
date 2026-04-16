<!-- eslint-disable max-lines -->
<script lang="ts">
  import { Badge } from '$ui/badge'
  import { cn } from '$lib/utils'
  import {
    detectedPlatformFromSnapshot,
    machineDetectedArchLabel,
    machineDetectedOSLabel,
    machineDetectionBadgeClass,
    machineDetectionMessage,
    machineDetectionStatusLabel,
    machineModeGuide,
    parseMachineSnapshot,
    machineReachabilityLabel,
    normalizeReachabilityMode,
  } from '../model'
  import { buildMachineSetupGuide } from '../machine-setup'
  import type { MachineDraft, MachineItem, MachineReachabilityMode } from '../types'
  import {
    Monitor,
    ArrowLeftRight,
    Radio,
    Check,
    ArrowRight,
    KeyRound,
    Loader2,
    Play,
  } from '@lucide/svelte'
  import type { TranslationKey } from '$lib/i18n'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { slide } from 'svelte/transition'
  import { Button } from '$ui/button'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { runMachineSSHBootstrap, machineErrorMessage } from './machines-page-api'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'

  type WsStrategy = 'direct-open' | 'ssh-install-listener' | 'reverse'
  type WsOption = {
    strategy: WsStrategy
    mode: MachineReachabilityMode
    icon: typeof Monitor
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
  const reachabilityGuide = $derived(machineModeGuide(reachabilityMode))

  // Progressive flow state. Starts null, synced from current reachability so
  // existing machines land in the correct step without user interaction.
  type LocationAnswer = 'local' | 'remote' | null
  let locationAnswer = $state<LocationAnswer>(null)
  let wsStrategy = $state<WsStrategy | null>(null)

  // Only pre-fill answers for an existing machine; for new machines let the
  // user click through the guided flow from step 1. The 'direct-open' vs
  // 'ssh-install-listener' distinction can't be inferred from reachability
  // alone, so existing direct_connect machines default to 'direct-open' —
  // the user can re-pick if they used the SSH-installed listener path.
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
  const detectedOSLabel = $derived(
    machineDetectedOSLabel(machine?.detected_os ?? detectedPlatform.os),
  )
  const detectedArchLabel = $derived(
    machineDetectedArchLabel(machine?.detected_arch ?? detectedPlatform.arch),
  )
  const detectionSummary = $derived(machineDetectionMessage(machine, draft))
  const setupGuide = $derived(buildMachineSetupGuide({ machine, draft }))
  const detectedPlatform = $derived(
    detectedPlatformFromSnapshot(parseMachineSnapshot(machine?.resources)),
  )

  const allCommands = $derived(setupGuide.commands)

  // Map the user's chosen WS strategy onto the CLI topology flag so the
  // in-process ssh-bootstrap handler picks the same service manifest the CLI
  // would.
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

  <!-- Step 1: where is the machine -->
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
            <span class="text-foreground text-sm font-medium">
              {i18nStore.t(option.descKey)}
            </span>
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

  <!-- Step 2: only shown for remote -->
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

  <!-- Step 3: topology summary, revealed once the flow is answered -->
  {#if flowComplete}
    <div
      class="border-border bg-card rounded-lg border px-3.5 py-3"
      transition:slide={{ duration: 220 }}
    >
      <div class="flex items-center gap-2">
        <span
          class="bg-primary/10 text-primary flex size-5 items-center justify-center rounded-full text-[10px] font-semibold"
        >
          3
        </span>
        <p class="text-foreground text-sm font-medium">
          {i18nStore.t('machines.machineEditorGuidance.progressive.step3.title')}
        </p>
      </div>
      <div class="mt-2.5 flex flex-wrap items-center gap-2">
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
          <span class="text-muted-foreground">
            {i18nStore.t('machines.machineEditorGuidance.labels.required')}
          </span>
          <span class="text-foreground ml-1">{reachabilityGuide.requiredFields}</span>
        </div>
        <div>
          <span class="text-muted-foreground">
            {i18nStore.t('machines.machineEditorGuidance.labels.state')}
          </span>
          <span class="text-foreground ml-1">{setupGuide.stateSummary}</span>
        </div>
      </div>
    </div>

    <!-- Step 4: runtime + next steps -->
    <div transition:slide={{ duration: 260, delay: 80 }}>
      <div class="mb-2 flex items-center gap-2">
        <span
          class="bg-primary/10 text-primary flex size-5 items-center justify-center rounded-full text-[10px] font-semibold"
        >
          4
        </span>
        <p class="text-foreground text-sm font-medium">
          {i18nStore.t('machines.machineEditorGuidance.progressive.step4.title')}
        </p>
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
          <p class="text-foreground text-sm font-medium">
            {i18nStore.t('machines.machineEditorGuidance.heading.nextStep')}
          </p>
          <ul class="text-muted-foreground mt-2 space-y-1.5 text-xs leading-relaxed">
            {#each setupGuide.nextSteps as step, index (`${step}-${index}`)}
              <li>{step}</li>
            {/each}
          </ul>
        </div>
      </div>
    </div>

    <!-- Step 5: concrete commands + one-click bootstrap (if applicable) -->
    {#if allCommands.length > 0 || canRunBootstrap}
      <div transition:slide={{ duration: 300, delay: 160 }}>
        <div class="mb-2 flex items-center gap-2">
          <span
            class="bg-primary/10 text-primary flex size-5 items-center justify-center rounded-full text-[10px] font-semibold"
          >
            5
          </span>
          <p class="text-foreground text-sm font-medium">
            {i18nStore.t('machines.machineEditorGuidance.progressive.step5.title')}
          </p>
        </div>

        {#if canRunBootstrap}
          <div class="border-primary/30 bg-primary/5 mb-3 rounded-lg border px-3.5 py-3">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <p class="text-foreground text-sm font-medium">
                  {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.title')}
                </p>
                <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
                  {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.description')}
                </p>
              </div>
              <Button
                type="button"
                size="sm"
                disabled={bootstrapRunning}
                onclick={handleRunBootstrap}
                data-testid="machine-ssh-bootstrap-run"
              >
                {#if bootstrapRunning}
                  <Loader2 class="size-3.5 animate-spin" />
                  {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.running')}
                {:else}
                  <Play class="size-3.5" />
                  {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.action')}
                {/if}
              </Button>
            </div>
            {#if bootstrapError}
              <div
                class="border-destructive/20 bg-destructive/5 text-destructive mt-3 rounded-md border px-3 py-2 text-[11px] leading-relaxed"
              >
                {bootstrapError}
              </div>
            {/if}
            {#if bootstrapResult}
              <div class="border-primary/20 mt-3 space-y-1 border-t pt-3 text-[11px]">
                <div class="flex flex-wrap gap-x-4 gap-y-1">
                  <span>
                    <span class="text-muted-foreground">
                      {i18nStore.t(
                        'machines.machineEditorGuidance.progressive.bootstrap.serviceManager',
                      )}
                    </span>
                    <span class="text-foreground ml-1">{bootstrapResult.service_manager}</span>
                  </span>
                  <span>
                    <span class="text-muted-foreground">
                      {i18nStore.t(
                        'machines.machineEditorGuidance.progressive.bootstrap.serviceName',
                      )}
                    </span>
                    <span class="text-foreground ml-1">{bootstrapResult.service_name}</span>
                  </span>
                  <span>
                    <span class="text-muted-foreground">
                      {i18nStore.t(
                        'machines.machineEditorGuidance.progressive.bootstrap.serviceStatus',
                      )}
                    </span>
                    <span class="text-foreground ml-1">{bootstrapResult.service_status}</span>
                  </span>
                </div>
                {#if bootstrapResult.connection_target}
                  <p class="text-muted-foreground">
                    <span>
                      {i18nStore.t(
                        'machines.machineEditorGuidance.progressive.bootstrap.connectionTarget',
                      )}
                    </span>
                    <span class="text-foreground ml-1">{bootstrapResult.connection_target}</span>
                  </p>
                {/if}
              </div>
            {/if}
          </div>
        {/if}

        {#if allCommands.length > 0}
          <div class="grid gap-3">
            {#each allCommands as command (command.title)}
              <div class="border-border bg-card rounded-lg border px-3.5 py-3">
                <p class="text-foreground text-sm font-medium">{command.title}</p>
                <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
                  {command.description}
                </p>
                <pre
                  class="bg-muted/60 text-foreground mt-3 overflow-x-auto rounded-md px-3 py-2 text-xs whitespace-pre-wrap">{command.command}</pre>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</section>
