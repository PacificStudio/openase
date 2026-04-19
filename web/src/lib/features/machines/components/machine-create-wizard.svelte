<script lang="ts">
  import * as Dialog from '$ui/dialog'
  import { cn } from '$lib/utils'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import type { Machine, MachineSSHBootstrapResult } from '$lib/api/contracts'
  import MachineCreateWizardFooter from './machine-create-wizard-footer.svelte'
  import {
    machineWizardLocationOptions,
    machineWizardStrategyOptions,
  } from './machine-create-wizard-options'
  import MachineCreateWizardReview from './machine-create-wizard-review.svelte'
  import {
    machineErrorMessage,
    runMachineHealthRefresh,
    runMachineSSHBootstrap,
    saveMachine,
  } from './machines-page-api'
  import MachineCreateWizardSteps from './machine-create-wizard-steps.svelte'
  import type { MachineMutationInput } from '../types'
  import type {
    MachineWizardLocationAnswer,
    MachineWizardStep,
    MachineWizardStrategy,
  } from './machine-create-wizard-types'

  let {
    open = $bindable(false),
    organizationId,
    onCreated,
  }: {
    open?: boolean
    organizationId: string | null
    onCreated?: (machine: Machine) => void
  } = $props()

  let step = $state<MachineWizardStep>('location')
  let location = $state<MachineWizardLocationAnswer | null>(null)
  let name = $state('')
  let host = $state('')
  let strategy = $state<MachineWizardStrategy | null>(null)
  let sshUser = $state('')
  let sshKeyPath = $state('~/.ssh/id_ed25519')
  let advertisedEndpoint = $state('')

  let saving = $state(false)
  let bootstrapping = $state(false)
  let bootstrapResult = $state<MachineSSHBootstrapResult | null>(null)
  let errorMessage = $state('')

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

  const strategyNeedsSSH = $derived(strategy === 'ssh-install-listener' || strategy === 'reverse')
  const stepOrder = $derived(computeStepOrder(location, strategy))
  const currentStepIndex = $derived(Math.max(0, stepOrder.indexOf(step)))
  const totalSteps = $derived(stepOrder.length)
  const isLastStep = $derived(step === 'review')

  function computeStepOrder(
    loc: MachineWizardLocationAnswer | null,
    strat: MachineWizardStrategy | null,
  ): MachineWizardStep[] {
    if (loc === 'local') return ['location', 'identity', 'review']
    const order: MachineWizardStep[] = ['location', 'identity', 'strategy']
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

  function pickLocation(value: MachineWizardLocationAnswer) {
    location = value
    if (value === 'local') {
      strategy = null
      name = name || 'local'
    } else if (!strategy) {
      strategy = 'ssh-install-listener'
    }
    setTimeout(() => goNext(), 120)
  }

  function pickStrategy(value: MachineWizardStrategy) {
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
        status: 'online',
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
        status: 'offline',
        workspace_root: '',
        agent_cli_path: '',
        env_vars: [],
      }
    }

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
      status: 'offline',
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
    let createdMachine: Machine | null = null
    try {
      const mutation = buildMutationInput()
      const created = await saveMachine(organizationId, null, 'create', mutation)
      createdMachine = created.machine

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
          const refreshed = await runMachineHealthRefresh(created.machine.id)
          createdMachine = refreshed.machine
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

      if (createdMachine) {
        onCreated?.(createdMachine)
      }

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
</script>

<Dialog.Root bind:open>
  <Dialog.Content class="sm:max-w-lg" data-testid="machine-create-wizard">
    <Dialog.Header>
      <Dialog.Title>{i18nStore.t('machines.machineCreateWizard.title')}</Dialog.Title>
      <Dialog.Description class="text-xs">
        {i18nStore.t('machines.machineCreateWizard.description')}
      </Dialog.Description>
    </Dialog.Header>

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
      {#if step === 'review'}
        <MachineCreateWizardReview
          {location}
          {name}
          {host}
          {strategy}
          {sshUser}
          {advertisedEndpoint}
          {strategyNeedsSSH}
          {saving}
          {bootstrapping}
          {bootstrapResult}
          {errorMessage}
          onClose={closeWizard}
        />
      {:else}
        <MachineCreateWizardSteps
          {step}
          {location}
          bind:name
          bind:host
          {strategy}
          bind:sshUser
          bind:sshKeyPath
          bind:advertisedEndpoint
          locationOptions={machineWizardLocationOptions}
          strategyOptions={machineWizardStrategyOptions}
          onPickLocation={pickLocation}
          onPickStrategy={pickStrategy}
        />
      {/if}
    </Dialog.Body>

    <Dialog.Footer>
      <MachineCreateWizardFooter
        {currentStepIndex}
        canAdvance={canAdvance()}
        {isLastStep}
        {saving}
        {bootstrapping}
        hasTerminalState={Boolean(bootstrapResult || (!saving && !bootstrapping && errorMessage))}
        onBack={goBack}
        onNext={goNext}
        onCreate={handleCreate}
      />
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
