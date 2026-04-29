<script lang="ts">
  import { Check } from '@lucide/svelte'
  import { fly } from 'svelte/transition'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { cn } from '$lib/utils'
  import type {
    MachineWizardLocationAnswer,
    MachineWizardOptionCard,
    MachineWizardStep,
    MachineWizardStrategy,
  } from './machine-create-wizard-types'

  let {
    step,
    location,
    name = $bindable(''),
    host = $bindable(''),
    strategy,
    sshUser = $bindable(''),
    sshKeyPath = $bindable(''),
    advertisedEndpoint = $bindable(''),
    locationOptions,
    strategyOptions,
    onPickLocation,
    onPickStrategy,
  }: {
    step: MachineWizardStep
    location: MachineWizardLocationAnswer | null
    name: string
    host: string
    strategy: MachineWizardStrategy | null
    sshUser: string
    sshKeyPath: string
    advertisedEndpoint: string
    locationOptions: MachineWizardOptionCard<MachineWizardLocationAnswer>[]
    strategyOptions: MachineWizardOptionCard<MachineWizardStrategy>[]
    onPickLocation?: (value: MachineWizardLocationAnswer) => void
    onPickStrategy?: (value: MachineWizardStrategy) => void
  } = $props()
</script>

{#if step === 'location'}
  <div class="space-y-3" in:fly={{ y: 6, duration: 180 }}>
    <div class="space-y-1">
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineCreateWizard.location.question')}
      </p>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('machines.machineCreateWizard.location.helper')}
      </p>
    </div>
    <div class="grid gap-2 sm:grid-cols-2">
      {#each locationOptions as option (option.value)}
        {@const Icon = option.icon}
        {@const selected = location === option.value}
        <button
          type="button"
          class={cn(
            'border-border bg-card hover:bg-muted/50 rounded-lg border px-3 py-3 text-left transition-all',
            selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
          )}
          onclick={() => onPickLocation?.(option.value)}
          data-testid={`machine-wizard-location-${option.value}`}
        >
          <div class="flex items-center gap-2">
            <Icon class={cn('size-4', selected ? 'text-primary' : 'text-muted-foreground')} />
            <span class="text-foreground text-sm font-medium">
              {i18nStore.t(option.titleKey)}
            </span>
            {#if selected}
              <Check class="text-primary ml-auto size-3.5" />
            {/if}
          </div>
          <p class="text-muted-foreground mt-1 text-xs leading-relaxed">
            {i18nStore.t(option.descKey)}
          </p>
        </button>
      {/each}
    </div>
  </div>
{:else if step === 'identity'}
  <div class="space-y-4" in:fly={{ y: 6, duration: 180 }}>
    <div class="space-y-1">
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineCreateWizard.identity.question')}
      </p>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('machines.machineCreateWizard.identity.helper')}
      </p>
    </div>
    <div class="space-y-3">
      <div class="space-y-1.5">
        <Label for="wizard-name" class="text-xs">
          {i18nStore.t('machines.machineCreateWizard.identity.nameLabel')}
        </Label>
        <Input
          id="wizard-name"
          bind:value={name}
          placeholder={i18nStore.t('machines.machineCreateWizard.identity.namePlaceholder')}
          autocomplete="off"
          data-testid="machine-wizard-name"
        />
      </div>
      {#if location === 'remote'}
        <div class="space-y-1.5">
          <Label for="wizard-host" class="text-xs">
            {i18nStore.t('machines.machineCreateWizard.identity.hostLabel')}
          </Label>
          <Input
            id="wizard-host"
            bind:value={host}
            placeholder={i18nStore.t('machines.machineCreateWizard.identity.hostPlaceholder')}
            autocomplete="off"
            data-testid="machine-wizard-host"
          />
          <p class="text-muted-foreground text-[11px] leading-relaxed">
            {i18nStore.t('machines.machineCreateWizard.identity.hostHint')}
          </p>
        </div>
      {/if}
    </div>
  </div>
{:else if step === 'strategy'}
  <div class="space-y-3" in:fly={{ y: 6, duration: 180 }}>
    <div class="space-y-1">
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineCreateWizard.strategy.question')}
      </p>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('machines.machineCreateWizard.strategy.helper')}
      </p>
    </div>
    <div class="grid gap-2">
      {#each strategyOptions as option (option.value)}
        {@const Icon = option.icon}
        {@const selected = strategy === option.value}
        <button
          type="button"
          class={cn(
            'border-border bg-card hover:bg-muted/50 rounded-lg border px-3 py-3 text-left transition-all',
            selected && 'border-primary bg-primary/5 ring-primary/20 ring-1',
          )}
          onclick={() => onPickStrategy?.(option.value)}
          data-testid={`machine-wizard-strategy-${option.value}`}
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
                {#if option.value === 'ssh-install-listener'}
                  <span
                    class="bg-primary/10 text-primary rounded-full px-1.5 py-px text-[10px] font-medium"
                  >
                    {i18nStore.t('machines.machineCreateWizard.strategy.recommendedBadge')}
                  </span>
                {/if}
                {#if selected}
                  <Check class="text-primary ml-auto size-3.5" />
                {/if}
              </div>
              <p class="text-muted-foreground mt-1 text-[11px] leading-relaxed">
                {i18nStore.t(option.descKey)}
              </p>
            </div>
          </div>
        </button>
      {/each}
    </div>
  </div>
{:else if step === 'credentials'}
  <div class="space-y-4" in:fly={{ y: 6, duration: 180 }}>
    <div class="space-y-1">
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineCreateWizard.credentials.question')}
      </p>
      <p class="text-muted-foreground text-xs leading-relaxed">
        {i18nStore.t('machines.machineCreateWizard.credentials.helper')}
      </p>
    </div>
    <div class="space-y-3">
      <div class="space-y-1.5">
        <Label for="wizard-ssh-user" class="text-xs">
          {i18nStore.t('machines.machineCreateWizard.credentials.userLabel')}
        </Label>
        <Input
          id="wizard-ssh-user"
          bind:value={sshUser}
          placeholder={i18nStore.t('machines.machineCreateWizard.credentials.userPlaceholder')}
          autocomplete="off"
          data-testid="machine-wizard-ssh-user"
        />
      </div>
      <div class="space-y-1.5">
        <Label for="wizard-ssh-key" class="text-xs">
          {i18nStore.t('machines.machineCreateWizard.credentials.keyLabel')}
        </Label>
        <Input
          id="wizard-ssh-key"
          bind:value={sshKeyPath}
          placeholder={i18nStore.t('machines.machineCreateWizard.credentials.keyPlaceholder')}
          autocomplete="off"
          data-testid="machine-wizard-ssh-key"
        />
        <p class="text-muted-foreground text-[11px] leading-relaxed">
          {i18nStore.t('machines.machineCreateWizard.credentials.keyHint')}
        </p>
      </div>
    </div>
  </div>
{:else if step === 'advertised-endpoint'}
  <div class="space-y-4" in:fly={{ y: 6, duration: 180 }}>
    <div class="space-y-1">
      <p class="text-foreground text-sm font-medium">
        {i18nStore.t('machines.machineCreateWizard.endpoint.question')}
      </p>
      <p class="text-muted-foreground text-xs leading-relaxed">
        {i18nStore.t('machines.machineCreateWizard.endpoint.helper')}
      </p>
    </div>
    <div class="space-y-1.5">
      <Label for="wizard-endpoint" class="text-xs">
        {i18nStore.t('machines.machineCreateWizard.endpoint.label')}
      </Label>
      <Input
        id="wizard-endpoint"
        bind:value={advertisedEndpoint}
        placeholder="wss://machine.example.com/openase/runtime"
        autocomplete="off"
        data-testid="machine-wizard-endpoint"
      />
    </div>
  </div>
{/if}
