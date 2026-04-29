<script lang="ts">
  import { ArrowLeft, ArrowRight } from '@lucide/svelte'
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    currentStepIndex,
    canAdvance,
    isLastStep,
    saving = false,
    bootstrapping = false,
    hasTerminalState = false,
    onBack,
    onNext,
    onCreate,
  }: {
    currentStepIndex: number
    canAdvance: boolean
    isLastStep: boolean
    saving?: boolean
    bootstrapping?: boolean
    hasTerminalState?: boolean
    onBack?: () => void
    onNext?: () => void
    onCreate?: () => void
  } = $props()
</script>

<div class="flex items-center justify-between gap-2">
  <Button
    variant="ghost"
    size="sm"
    onclick={onBack}
    disabled={currentStepIndex === 0 || saving || bootstrapping}
    data-testid="machine-wizard-back"
  >
    <ArrowLeft class="size-3.5" />
    {i18nStore.t('machines.machineCreateWizard.actions.back')}
  </Button>

  {#if isLastStep}
    {#if !hasTerminalState}
      <Button
        size="sm"
        onclick={onCreate}
        disabled={saving || bootstrapping}
        data-testid="machine-wizard-create"
      >
        {saving || bootstrapping
          ? i18nStore.t('machines.machineCreateWizard.actions.creating')
          : i18nStore.t('machines.machineCreateWizard.actions.create')}
      </Button>
    {/if}
  {:else}
    <Button size="sm" onclick={onNext} disabled={!canAdvance} data-testid="machine-wizard-next">
      {i18nStore.t('machines.machineCreateWizard.actions.next')}
      <ArrowRight class="size-3.5" />
    </Button>
  {/if}
</div>
