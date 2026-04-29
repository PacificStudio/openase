<script lang="ts">
  import { ChevronDown, ChevronUp, Loader2, Wrench } from '@lucide/svelte'
  import { slide } from 'svelte/transition'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import MachineSetupCommandList from './machine-setup-command-list.svelte'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'
  import type { MachineSetupGuide } from '../machine-setup'

  let {
    setupGuide,
    hasTrouble,
    setupExpanded,
    showBootstrapRepair,
    bootstrapRunning = false,
    bootstrapResult = null,
    bootstrapError = '',
    onToggle,
    onBootstrapRepair,
  }: {
    setupGuide: MachineSetupGuide
    hasTrouble: boolean
    setupExpanded: boolean
    showBootstrapRepair: boolean
    bootstrapRunning?: boolean
    bootstrapResult?: MachineSSHBootstrapResult | null
    bootstrapError?: string
    onToggle?: () => void
    onBootstrapRepair?: () => void
  } = $props()
</script>

<div class="border-border bg-card rounded-xl border">
  <button
    type="button"
    class="hover:bg-muted/40 flex w-full items-center justify-between gap-3 rounded-xl px-4 py-2.5 text-left transition-colors"
    onclick={onToggle}
    aria-expanded={setupExpanded}
    data-testid="machine-health-setup-toggle"
  >
    <div class="flex min-w-0 items-center gap-2">
      <span class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineHealthPanel.heading.setupGuidance')}
      </span>
      <Badge variant="outline" class="text-[10px]">{setupGuide.topologyLabel}</Badge>
      {#if hasTrouble}
        <Badge variant="destructive" class="text-[10px]">
          {i18nStore.t('machines.machineHealthPanel.status.needsAttention')}
        </Badge>
      {/if}
    </div>
    {#if setupExpanded}
      <ChevronUp class="text-muted-foreground size-4" />
    {:else}
      <ChevronDown class="text-muted-foreground size-4" />
    {/if}
  </button>

  {#if setupExpanded}
    <div class="border-border border-t" transition:slide={{ duration: 200 }}>
      <div class="flex flex-wrap items-center justify-between gap-2 px-4 py-3">
        <Badge variant="outline">{setupGuide.stateLabel}</Badge>
        {#if showBootstrapRepair}
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={bootstrapRunning}
            onclick={onBootstrapRepair}
            data-testid="machine-health-bootstrap-repair"
          >
            {#if bootstrapRunning}
              <Loader2 class="size-3.5 animate-spin" />
              {i18nStore.t('machines.machineHealthPanel.bootstrapRepair.running')}
            {:else}
              <Wrench class="size-3.5" />
              {i18nStore.t('machines.machineHealthPanel.bootstrapRepair.action')}
            {/if}
          </Button>
        {/if}
      </div>
      <p class="text-muted-foreground px-4 pb-3 text-xs leading-relaxed">
        {hasTrouble && showBootstrapRepair
          ? i18nStore.t('machines.machineHealthPanel.bootstrapRepair.description')
          : setupGuide.topologySummary}
      </p>

      {#if bootstrapError}
        <div
          class="border-destructive/20 bg-destructive/5 text-destructive mx-4 mb-3 rounded-md border px-3 py-2 text-[11px] leading-relaxed"
        >
          {bootstrapError}
        </div>
      {/if}

      {#if bootstrapResult}
        <div
          class="border-primary/20 bg-primary/5 mx-4 mb-3 rounded-md border px-3 py-2 text-[11px] leading-relaxed"
        >
          <div class="flex flex-wrap gap-x-4 gap-y-1">
            <span>
              <span class="text-muted-foreground">
                {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.serviceManager')}
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
                {i18nStore.t('machines.machineEditorGuidance.progressive.bootstrap.serviceStatus')}
              </span>
              <span class="text-foreground ml-1">{bootstrapResult.service_status}</span>
            </span>
          </div>
          {#if bootstrapResult.connection_target}
            <p class="text-muted-foreground mt-1">
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

      <div class="grid gap-3 px-4 py-3 lg:grid-cols-3">
        <div class="space-y-1.5">
          <p class="text-foreground text-sm font-medium">{setupGuide.runtimeLabel}</p>
          <p class="text-muted-foreground text-xs leading-relaxed">{setupGuide.runtimeSummary}</p>
        </div>
        <div class="space-y-1.5">
          <p class="text-foreground text-sm font-medium">{setupGuide.helperLabel}</p>
          <p class="text-muted-foreground text-xs leading-relaxed">{setupGuide.helperSummary}</p>
        </div>
        <div class="space-y-1.5">
          <p class="text-foreground text-sm font-medium">
            {i18nStore.t('machines.machineHealthPanel.heading.nextSteps')}
          </p>
          <ul class="text-muted-foreground space-y-1.5 text-xs leading-relaxed">
            {#each setupGuide.nextSteps as step, index (`${step}-${index}`)}
              <li>{step}</li>
            {/each}
          </ul>
        </div>
      </div>

      {#if setupGuide.commands.length > 0}
        <div class="border-border border-t px-4 py-4">
          <MachineSetupCommandList
            commands={setupGuide.commands}
            dashed={true}
            className="grid gap-3"
          />
        </div>
      {/if}
    </div>
  {/if}
</div>
