<script lang="ts">
  import * as Dialog from '$ui/dialog'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import { cn } from '$lib/utils'
  import { fly } from 'svelte/transition'
  import {
    ArrowLeftRight,
    ArrowRight,
    ArrowLeft,
    Check,
    KeyRound,
    Loader2,
    Monitor,
    Radio,
    Sparkles,
  } from '@lucide/svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { TranslationKey } from '$lib/i18n'
  import type {
    Machine,
    MachineSSHBootstrapResult,
  } from '$lib/api/contracts'
  import {
    machineErrorMessage,
    runMachineSSHBootstrap,
    saveMachine,
  } from './machines-page-api'
  import type { MachineMutationInput } from '../types'

  type Strategy = 'direct-open' | 'ssh-install-listener' | 'reverse'
  type LocationAnswer = 'local' | 'remote'
  type Step =
    | 'location'
    | 'identity'
    | 'strategy'
    | 'credentials'
    | 'advertised-endpoint'
    | 'review'

  let {
    open = $bindable(false),
    organizationId,
    onCreated,
  }: {
    open?: boolean
    organizationId: string | null
    onCreated?: (machine: Machine) => void
  } = $props()

  // Wizard state — only the fields the user can actually answer. Ports, topology
  // envs, and binary paths fall back to sensible server defaults.
  let step = $state<Step>('location')
  let location = $state<LocationAnswer | null>(null)
  let name = $state('')
  let host = $state('')
  let strategy = $state<Strategy | null>(null)
  let sshUser = $state('')
  let sshKeyPath = $state('~/.ssh/id_ed25519')
  let advertisedEndpoint = $state('')

  let saving = $state(false)
  let bootstrapping = $state(false)
  let bootstrapResult = $state<MachineSSHBootstrapResult | null>(null)
  let errorMessage = $state('')

  // Reset wizard every time the dialog opens.
  $effect(() => {
    if (open) {
      step = 'location'
      location = null
      name = ''
      host = ''
      strategy = null
      sshUser = ''
      sshKeyPath = '~/.ssh/id_ed25519'
      advertisedEndpoint = ''
      saving = false
      bootstrapping = false
      bootstrapResult = null
      errorMessage = ''
    }
  })

  const strategyNeedsSSH = $derived(
    strategy === 'ssh-install-listener' || strategy === 'reverse',
  )
  const stepOrder = $derived(computeStepOrder(location, strategy))
  const currentStepIndex = $derived(Math.max(0, stepOrder.indexOf(step)))
  const totalSteps = $derived(stepOrder.length)
  const isLastStep = $derived(step === 'review')

  function computeStepOrder(
    loc: LocationAnswer | null,
    strat: Strategy | null,
  ): Step[] {
    if (loc === 'local') return ['location', 'identity', 'review']
    const order: Step[] = ['location', 'identity', 'strategy']
    if (strat === 'direct-open') order.push('advertised-endpoint')
    if (strat === 'ssh-install-listener' || strat === 'reverse') {
      order.push('credentials')
    }
    order.push('review')
    return order
  }

  function canAdvance(): boolean {
    switch (step) {
      case 'location':
        return location !== null
      case 'identity':
        if (location === 'local') return name.trim().length > 0
        return name.trim().length > 0 && host.trim().length > 0
      case 'strategy':
        return strategy !== null
      case 'credentials':
        return sshUser.trim().length > 0 && sshKeyPath.trim().length > 0
      case 'advertised-endpoint':
        return advertisedEndpoint.trim().length > 0
      case 'review':
        return true
    }
  }

  function goNext() {
    if (!canAdvance()) return
    const idx = stepOrder.indexOf(step)
    if (idx < stepOrder.length - 1) {
      step = stepOrder[idx + 1]
    }
  }

  function goBack() {
    const idx = stepOrder.indexOf(step)
    if (idx > 0) {
      step = stepOrder[idx - 1]
    }
  }

  function pickLocation(value: LocationAnswer) {
    location = value
    if (value === 'local') {
      strategy = null
      name = name || 'local'
    } else if (!strategy) {
      strategy = 'ssh-install-listener'
    }
    // Auto-advance for a snappy feel.
    setTimeout(() => goNext(), 120)
  }

  function pickStrategy(value: Strategy) {
    strategy = value
    setTimeout(() => goNext(), 120)
  }

  function buildMutationInput(): MachineMutationInput {
    if (location === 'local') {
      return {
        name: name.trim() || 'local',
        host: 'local',
        port: 22,
        reachability_mode: 'local',
        execution_mode: 'local_process',
        ssh_user: '',
        ssh_key_path: '',
        advertised_endpoint: '',
        description: '',
        labels: [],
        status: 'maintenance',
        workspace_root: '',
        agent_cli_path: '',
        env_vars: [],
      }
    }
    const hostValue = host.trim()
    if (strategy === 'reverse') {
      return {
        name: name.trim(),
        host: hostValue,
        port: 22,
        reachability_mode: 'reverse_connect',
        execution_mode: 'websocket',
        ssh_user: sshUser.trim(),
        ssh_key_path: sshKeyPath.trim(),
        advertised_endpoint: '',
        description: '',
        labels: [],
        status: 'maintenance',
        workspace_root: '',
        agent_cli_path: '',
        env_vars: [],
      }
    }
    // direct-open and ssh-install-listener both target direct_connect / ws_listener.
    const endpoint =
      strategy === 'direct-open'
        ? advertisedEndpoint.trim()
        : `ws://${hostValue}:19837/openase/runtime`
    return {
      name: name.trim(),
      host: hostValue,
      port: 22,
      reachability_mode: 'direct_connect',
      execution_mode: 'websocket',
      ssh_user: strategy === 'ssh-install-listener' ? sshUser.trim() : '',
      ssh_key_path: strategy === 'ssh-install-listener' ? sshKeyPath.trim() : '',
      advertised_endpoint: endpoint,
      description: '',
      labels: [],
      status: 'maintenance',
      workspace_root: '',
      agent_cli_path: '',
      env_vars: [],
    }
  }

  async function handleCreate() {
    if (!organizationId) {
      toastStore.error(i18nStore.t('machines.machineCreateWizard.errors.noOrg'))
      return
    }
    errorMessage = ''
    saving = true
    try {
      const mutation = buildMutationInput()
      const created = await saveMachine(organizationId, null, 'create', mutation)
      onCreated?.(created.machine)

      // Auto-bootstrap when the strategy maps to a CLI topology the backend
      // can drive in-process. direct-open skips this because the user manages
      // the listener themselves.
      if (strategy === 'ssh-install-listener' || strategy === 'reverse') {
        bootstrapping = true
        try {
          const topology =
            strategy === 'ssh-install-listener' ? 'remote-listener' : 'reverse-connect'
          const listenerAddress = strategy === 'ssh-install-listener' ? '0.0.0.0:19837' : ''
          bootstrapResult = await runMachineSSHBootstrap(created.machine.id, {
            topology,
            ...(listenerAddress ? { listener_address: listenerAddress } : {}),
          })
          toastStore.success(
            bootstrapResult.summary ||
              i18nStore.t('machines.machineCreateWizard.successes.bootstrap'),
          )
        } catch (bootstrapError) {
          errorMessage = machineErrorMessage(
            bootstrapError,
            i18nStore.t('machines.machineCreateWizard.errors.bootstrap'),
          )
          toastStore.error(errorMessage)
        } finally {
          bootstrapping = false
        }
      } else {
        toastStore.success(i18nStore.t('machines.machineCreateWizard.successes.created'))
      }
      // Keep the dialog open only when the user can read the bootstrap result.
      if (!bootstrapResult && !errorMessage) {
        open = false
      }
    } catch (caughtError) {
      errorMessage = machineErrorMessage(
        caughtError,
        i18nStore.t('machines.machineCreateWizard.errors.create'),
      )
      toastStore.error(errorMessage)
    } finally {
      saving = false
    }
  }

  function closeWizard() {
    open = false
  }

  type OptionCard<T> = {
    value: T
    icon: typeof Monitor
    titleKey: TranslationKey
    descKey: TranslationKey
  }

  const locationOptions: OptionCard<LocationAnswer>[] = [
    {
      value: 'local',
      icon: Monitor,
      titleKey: 'machines.machineCreateWizard.location.local.title',
      descKey: 'machines.machineCreateWizard.location.local.desc',
    },
    {
      value: 'remote',
      icon: ArrowRight,
      titleKey: 'machines.machineCreateWizard.location.remote.title',
      descKey: 'machines.machineCreateWizard.location.remote.desc',
    },
  ]

  const strategyOptions: OptionCard<Strategy>[] = [
    {
      value: 'ssh-install-listener',
      icon: Sparkles,
      titleKey: 'machines.machineCreateWizard.strategy.sshInstall.title',
      descKey: 'machines.machineCreateWizard.strategy.sshInstall.desc',
    },
    {
      value: 'reverse',
      icon: ArrowLeftRight,
      titleKey: 'machines.machineCreateWizard.strategy.reverse.title',
      descKey: 'machines.machineCreateWizard.strategy.reverse.desc',
    },
    {
      value: 'direct-open',
      icon: Radio,
      titleKey: 'machines.machineCreateWizard.strategy.directOpen.title',
      descKey: 'machines.machineCreateWizard.strategy.directOpen.desc',
    },
  ]
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg" data-testid="machine-create-wizard">
    <Dialog.Header>
      <Dialog.Title>
        {i18nStore.t('machines.machineCreateWizard.title')}
      </Dialog.Title>
      <Dialog.Description class="text-xs">
        {i18nStore.t('machines.machineCreateWizard.description')}
      </Dialog.Description>
    </Dialog.Header>

    <!-- Progress bar: simple dots, no chapter labels so it stays out of the way. -->
    <div class="flex items-center justify-between gap-1 px-1 pt-1">
      <div class="flex flex-1 gap-1">
        {#each Array(totalSteps) as _, idx (idx)}
          <span
            class={cn(
              'h-1 flex-1 rounded-full transition-colors',
              idx <= currentStepIndex ? 'bg-primary' : 'bg-muted',
            )}
          ></span>
        {/each}
      </div>
      <span class="text-muted-foreground shrink-0 text-[10px] font-medium">
        {currentStepIndex + 1} / {totalSteps}
      </span>
    </div>

    <Dialog.Body class="min-h-[220px] py-4">
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
                onclick={() => pickLocation(option.value)}
                data-testid={`machine-wizard-location-${option.value}`}
              >
                <div class="flex items-center gap-2">
                  <Icon
                    class={cn('size-4', selected ? 'text-primary' : 'text-muted-foreground')}
                  />
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
                onclick={() => pickStrategy(option.value)}
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
                placeholder={i18nStore.t(
                  'machines.machineCreateWizard.credentials.userPlaceholder',
                )}
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
                placeholder="~/.ssh/id_ed25519"
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
      {:else if step === 'review'}
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
                  <dd class="text-foreground font-medium">
                    {#if strategy === 'ssh-install-listener'}
                      {i18nStore.t('machines.machineCreateWizard.strategy.sshInstall.title')}
                    {:else if strategy === 'reverse'}
                      {i18nStore.t('machines.machineCreateWizard.strategy.reverse.title')}
                    {:else if strategy === 'direct-open'}
                      {i18nStore.t('machines.machineCreateWizard.strategy.directOpen.title')}
                    {/if}
                  </dd>
                </div>
                {#if strategyNeedsSSH}
                  <div class="flex items-center justify-between gap-3 px-3 py-2">
                    <dt class="text-muted-foreground">
                      {i18nStore.t('machines.machineCreateWizard.review.sshLabel')}
                    </dt>
                    <dd class="text-foreground font-medium">
                      {sshUser || '—'}@{host || '—'}
                    </dd>
                  </div>
                {/if}
                {#if strategy === 'direct-open'}
                  <div class="flex items-center justify-between gap-3 px-3 py-2">
                    <dt class="text-muted-foreground">
                      {i18nStore.t('machines.machineCreateWizard.review.endpointLabel')}
                    </dt>
                    <dd class="text-foreground font-medium break-all">
                      {advertisedEndpoint || '—'}
                    </dd>
                  </div>
                {/if}
              {/if}
            </dl>
          {/if}

          {#if saving || bootstrapping}
            <div class="border-primary/30 bg-primary/5 flex items-center gap-2 rounded-lg border px-3 py-2.5 text-xs">
              <Loader2 class="text-primary size-3.5 animate-spin" />
              <span class="text-foreground">
                {bootstrapping
                  ? i18nStore.t('machines.machineCreateWizard.review.bootstrapProgress')
                  : i18nStore.t('machines.machineCreateWizard.review.saveProgress')}
              </span>
            </div>
          {/if}

          {#if bootstrapResult}
            <div
              class="border-primary/30 bg-primary/5 space-y-1 rounded-lg border px-3 py-2.5 text-[11px]"
            >
              <p class="text-foreground text-xs font-medium">
                {i18nStore.t('machines.machineCreateWizard.review.bootstrapDone')}
              </p>
              <p class="text-muted-foreground">
                {bootstrapResult.summary}
              </p>
              <div class="mt-1 flex flex-wrap gap-x-3 gap-y-0.5">
                <span>
                  <span class="text-muted-foreground">
                    {i18nStore.t('machines.machineCreateWizard.review.serviceStatus')}
                  </span>
                  <span class="text-foreground ml-1">{bootstrapResult.service_status}</span>
                </span>
                <span>
                  <span class="text-muted-foreground">
                    {i18nStore.t('machines.machineCreateWizard.review.connectionTarget')}
                  </span>
                  <span class="text-foreground ml-1 break-all"
                    >{bootstrapResult.connection_target}</span
                  >
                </span>
              </div>
            </div>
          {/if}

          {#if errorMessage}
            <div
              class="border-destructive/40 bg-destructive/10 rounded-lg border px-3 py-2.5 text-xs"
            >
              <p class="text-destructive">{errorMessage}</p>
            </div>
          {/if}
        </div>
      {/if}
    </Dialog.Body>

    <Dialog.Footer class="flex items-center justify-between gap-2">
      <Button
        variant="ghost"
        size="sm"
        onclick={goBack}
        disabled={currentStepIndex === 0 || saving || bootstrapping}
        data-testid="machine-wizard-back"
      >
        <ArrowLeft class="size-3.5" />
        {i18nStore.t('machines.machineCreateWizard.actions.back')}
      </Button>

      {#if isLastStep}
        {#if bootstrapResult || (!saving && !bootstrapping && errorMessage)}
          <Button size="sm" onclick={closeWizard} data-testid="machine-wizard-done">
            {i18nStore.t('machines.machineCreateWizard.actions.done')}
          </Button>
        {:else}
          <Button
            size="sm"
            onclick={handleCreate}
            disabled={saving || bootstrapping}
            data-testid="machine-wizard-create"
          >
            {saving || bootstrapping
              ? i18nStore.t('machines.machineCreateWizard.actions.creating')
              : i18nStore.t('machines.machineCreateWizard.actions.create')}
          </Button>
        {/if}
      {:else}
        <Button
          size="sm"
          onclick={goNext}
          disabled={!canAdvance()}
          data-testid="machine-wizard-next"
        >
          {i18nStore.t('machines.machineCreateWizard.actions.next')}
          <ArrowRight class="size-3.5" />
        </Button>
      {/if}
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
