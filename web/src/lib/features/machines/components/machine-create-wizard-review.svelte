<script lang="ts">
  import { Loader2 } from '@lucide/svelte'
  import { fly } from 'svelte/transition'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Button } from '$ui/button'
  import type { MachineSSHBootstrapResult } from '$lib/api/contracts'
  import type {
    MachineWizardLocationAnswer,
    MachineWizardStrategy,
  } from './machine-create-wizard-types'

  let {
    location,
    name,
    host,
    strategy,
    sshUser,
    advertisedEndpoint,
    strategyNeedsSSH,
    saving = false,
    bootstrapping = false,
    bootstrapResult = null,
    errorMessage = '',
    onClose,
  }: {
    location: MachineWizardLocationAnswer | null
    name: string
    host: string
    strategy: MachineWizardStrategy | null
    sshUser: string
    advertisedEndpoint: string
    strategyNeedsSSH: boolean
    saving?: boolean
    bootstrapping?: boolean
    bootstrapResult?: MachineSSHBootstrapResult | null
    errorMessage?: string
    onClose?: () => void
  } = $props()

  function strategyLabel(value: MachineWizardStrategy | null): string {
    switch (value) {
      case 'ssh-install-listener':
        return i18nStore.t('machines.machineCreateWizard.strategy.sshInstall.title')
      case 'reverse':
        return i18nStore.t('machines.machineCreateWizard.strategy.reverse.title')
      case 'direct-open':
        return i18nStore.t('machines.machineCreateWizard.strategy.directOpen.title')
      default:
        return ''
    }
  }
</script>

<div class="space-y-3" in:fly={{ y: 6, duration: 180 }}>
  <div class="space-y-1">
    <p class="text-foreground text-sm font-medium">
      {i18nStore.t('machines.machineCreateWizard.review.question')}
    </p>
    <p class="text-muted-foreground text-xs">
      {i18nStore.t('machines.machineCreateWizard.review.helper')}
    </p>
  </div>

  {#if !bootstrapResult && !errorMessage}
    <dl class="border-border bg-card divide-border divide-y rounded-lg border text-xs">
      <div class="flex items-center justify-between gap-3 px-3 py-2">
        <dt class="text-muted-foreground">
          {i18nStore.t('machines.machineCreateWizard.review.nameLabel')}
        </dt>
        <dd class="text-foreground font-medium">{name || '—'}</dd>
      </div>
      {#if location === 'remote'}
        <div class="flex items-center justify-between gap-3 px-3 py-2">
          <dt class="text-muted-foreground">
            {i18nStore.t('machines.machineCreateWizard.review.hostLabel')}
          </dt>
          <dd class="text-foreground font-medium">{host || '—'}</dd>
        </div>
        <div class="flex items-center justify-between gap-3 px-3 py-2">
          <dt class="text-muted-foreground">
            {i18nStore.t('machines.machineCreateWizard.review.strategyLabel')}
          </dt>
          <dd class="text-foreground font-medium">{strategyLabel(strategy)}</dd>
        </div>
        {#if strategyNeedsSSH}
          <div class="flex items-center justify-between gap-3 px-3 py-2">
            <dt class="text-muted-foreground">
              {i18nStore.t('machines.machineCreateWizard.review.sshLabel')}
            </dt>
            <dd class="text-foreground font-medium">{sshUser || '—'}@{host || '—'}</dd>
          </div>
        {/if}
        {#if strategy === 'direct-open'}
          <div class="flex items-center justify-between gap-3 px-3 py-2">
            <dt class="text-muted-foreground">
              {i18nStore.t('machines.machineCreateWizard.review.endpointLabel')}
            </dt>
            <dd class="text-foreground font-medium break-all">{advertisedEndpoint || '—'}</dd>
          </div>
        {/if}
      {/if}
    </dl>
  {/if}

  {#if saving || bootstrapping}
    <div
      class="border-primary/30 bg-primary/5 flex items-center gap-2 rounded-lg border px-3 py-2.5 text-xs"
    >
      <Loader2 class="text-primary size-3.5 animate-spin" />
      <span class="text-foreground">
        {bootstrapping
          ? i18nStore.t('machines.machineCreateWizard.review.bootstrapProgress')
          : i18nStore.t('machines.machineCreateWizard.review.saveProgress')}
      </span>
    </div>
  {/if}

  {#if bootstrapResult}
    <div class="border-primary/30 bg-primary/5 space-y-1 rounded-lg border px-3 py-2.5 text-[11px]">
      <p class="text-foreground text-xs font-medium">
        {i18nStore.t('machines.machineCreateWizard.review.bootstrapDone')}
      </p>
      <p class="text-muted-foreground">{bootstrapResult.summary}</p>
      <div class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5">
        <span>
          <span class="text-muted-foreground"
            >{i18nStore.t('machines.machineCreateWizard.review.serviceStatus')}</span
          >
          <span class="text-foreground ml-1">{bootstrapResult.service_status}</span>
        </span>
        <span>
          <span class="text-muted-foreground"
            >{i18nStore.t('machines.machineCreateWizard.review.connectionTarget')}</span
          >
          <span class="text-foreground ml-1 break-all">{bootstrapResult.connection_target}</span>
        </span>
      </div>
    </div>
  {/if}

  {#if errorMessage}
    <div class="border-destructive/40 bg-destructive/10 rounded-lg border px-3 py-2.5 text-xs">
      <p class="text-destructive">{errorMessage}</p>
    </div>
  {/if}

  {#if bootstrapResult || (!saving && !bootstrapping && errorMessage)}
    <Button size="sm" onclick={onClose} data-testid="machine-wizard-done">
      {i18nStore.t('machines.machineCreateWizard.actions.done')}
    </Button>
  {/if}
</div>
