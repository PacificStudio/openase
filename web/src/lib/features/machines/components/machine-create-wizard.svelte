<script lang="ts">
  import type { Machine, MachineSSHBootstrapResult } from '$lib/api/contracts'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import * as Dialog from '$ui/dialog'
  import MachineCreateWizardFooter from './machine-create-wizard-footer.svelte'
  import MachineCreateWizardProgress from './machine-create-wizard-progress.svelte'
  import {
    applyMachineWizardLocationChoice,
    canAdvanceMachineWizardStep,
    computeMachineWizardStepOrder,
    createMachineCreateWizardDraft,
  } from './machine-create-wizard-flow'
  import {
    machineWizardLocationOptions,
    machineWizardStrategyOptions,
  } from './machine-create-wizard-options'
  import MachineCreateWizardReview from './machine-create-wizard-review.svelte'
  import {
    MachineCreateWizardSubmitError,
    submitMachineCreateWizard,
  } from './machine-create-wizard-submit'
  import MachineCreateWizardSteps from './machine-create-wizard-steps.svelte'
  import { machineErrorMessage } from './machines-page-api'
  import type {
    MachineWizardLocationAnswer,
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

  let wizard = $state(createMachineCreateWizardDraft())
  let saving = $state(false)
  let bootstrapping = $state(false)
  let bootstrapResult = $state<MachineSSHBootstrapResult | null>(null)
  let errorMessage = $state('')

  $effect(() => {
    if (!open) return
    wizard = createMachineCreateWizardDraft()
    saving = false
    bootstrapping = false
    bootstrapResult = null
    errorMessage = ''
  })

  const strategyNeedsSSH = $derived(
    wizard.strategy === 'ssh-install-listener' || wizard.strategy === 'reverse',
  )
  const stepOrder = $derived(computeMachineWizardStepOrder(wizard.location, wizard.strategy))
  const currentStepIndex = $derived(Math.max(0, stepOrder.indexOf(wizard.step)))
  const totalSteps = $derived(stepOrder.length)
  const isLastStep = $derived(wizard.step === 'review')
  const canAdvanceCurrentStep = $derived(canAdvanceMachineWizardStep(wizard))

  function goNext() {
    if (!canAdvanceCurrentStep) return
    const nextIndex = stepOrder.indexOf(wizard.step) + 1
    if (nextIndex < stepOrder.length) {
      wizard.step = stepOrder[nextIndex]
    }
  }

  function goBack() {
    const previousIndex = stepOrder.indexOf(wizard.step) - 1
    if (previousIndex >= 0) {
      wizard.step = stepOrder[previousIndex]
    }
  }

  function pickLocation(value: MachineWizardLocationAnswer) {
    wizard = applyMachineWizardLocationChoice(wizard, value)
    setTimeout(() => goNext(), 120)
  }

  function pickStrategy(value: MachineWizardStrategy) {
    wizard.strategy = value
    setTimeout(() => goNext(), 120)
  }

  async function handleCreate() {
    if (!organizationId) {
      toastStore.error(i18nStore.t('machines.machineCreateWizard.errors.noOrg'))
      return
    }

    errorMessage = ''
    saving = true
    try {
      const created = await submitMachineCreateWizard({
        organizationId,
        draft: wizard,
        setBootstrapping: (value) => (bootstrapping = value),
      })
      bootstrapResult = created.bootstrapResult
      onCreated?.(created.machine)
      toastStore.success(
        created.bootstrapResult?.summary ||
          i18nStore.t(
            created.bootstrapResult
              ? 'machines.machineCreateWizard.successes.bootstrap'
              : 'machines.machineCreateWizard.successes.created',
          ),
      )
      if (!created.bootstrapResult) {
        open = false
      }
    } catch (caughtError) {
      const stage =
        caughtError instanceof MachineCreateWizardSubmitError ? caughtError.stage : 'create'
      const fallbackKey =
        stage === 'bootstrap'
          ? 'machines.machineCreateWizard.errors.bootstrap'
          : 'machines.machineCreateWizard.errors.create'
      const cause =
        caughtError instanceof MachineCreateWizardSubmitError ? caughtError.cause : caughtError
      errorMessage = machineErrorMessage(cause, i18nStore.t(fallbackKey))
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

    <MachineCreateWizardProgress {currentStepIndex} {totalSteps} />

    <Dialog.Body class="min-h-[220px] py-4">
      {#if wizard.step === 'review'}
        <MachineCreateWizardReview
          location={wizard.location}
          name={wizard.name}
          host={wizard.host}
          strategy={wizard.strategy}
          sshUser={wizard.sshUser}
          advertisedEndpoint={wizard.advertisedEndpoint}
          {strategyNeedsSSH}
          {saving}
          {bootstrapping}
          {bootstrapResult}
          {errorMessage}
          onClose={closeWizard}
        />
      {:else}
        <MachineCreateWizardSteps
          step={wizard.step}
          location={wizard.location}
          bind:name={wizard.name}
          bind:host={wizard.host}
          strategy={wizard.strategy}
          bind:sshUser={wizard.sshUser}
          bind:sshKeyPath={wizard.sshKeyPath}
          bind:advertisedEndpoint={wizard.advertisedEndpoint}
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
        canAdvance={canAdvanceCurrentStep}
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
