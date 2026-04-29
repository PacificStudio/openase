<script lang="ts">
  import { Loader2, Play } from '@lucide/svelte'
  import { slide } from 'svelte/transition'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import MachineSetupCommandList from './machine-setup-command-list.svelte'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'
  import type { MachineSetupGuide } from '../machine-setup'
  import type { MachineModeGuide } from '../types'

  let {
    reachabilityGuide,
    setupGuide,
    detectionBadgeClass,
    detectionStatusLabel,
    detectedOSLabel,
    detectedArchLabel,
    detectionSummary,
    canRunBootstrap = false,
    bootstrapRunning = false,
    bootstrapResult = null,
    bootstrapError = '',
    onRunBootstrap,
  }: {
    reachabilityGuide: MachineModeGuide
    setupGuide: MachineSetupGuide
    detectionBadgeClass: string
    detectionStatusLabel: string
    detectedOSLabel: string
    detectedArchLabel: string
    detectionSummary: string
    canRunBootstrap?: boolean
    bootstrapRunning?: boolean
    bootstrapResult?: MachineSSHBootstrapResult | null
    bootstrapError?: string
    onRunBootstrap?: () => void
  } = $props()
</script>

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
    <Badge variant="outline" class={`text-[10px] ${detectionBadgeClass}`}
      >{detectionStatusLabel}</Badge
    >
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

{#if setupGuide.commands.length > 0 || canRunBootstrap}
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
            onclick={onRunBootstrap}
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
                  {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.serviceName')}
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

    {#if setupGuide.commands.length > 0}
      <MachineSetupCommandList commands={setupGuide.commands} className="grid gap-3" />
    {/if}
  </div>
{/if}
