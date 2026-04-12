<script lang="ts">
  import type { AgentProviderModelCatalogEntry } from '$lib/api/contracts'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import {
    customProviderModelOptionValue,
    providerModelOptionsForAdapter,
    splitProviderModelSelection,
  } from '../provider-model-options'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    adapterType,
    modelName,
    modelCatalog = [],
    inputId = 'provider-model-name',
    onModelNameChange,
  }: {
    adapterType: string
    modelName: string
    modelCatalog?: AgentProviderModelCatalogEntry[]
    inputId?: string
    onModelNameChange?: (value: string) => void
  } = $props()

  let baseModelId = $state('')
  let customModelId = $state('')
  let syncedAdapterType = $state('')
  let syncedOptionSignature = $state('')

  const options = $derived(providerModelOptionsForAdapter(modelCatalog, adapterType))
  const optionSignature = $derived(options.map((option) => option.id).join('\n'))

  function effectiveModelName() {
    return customModelId.trim() || baseModelId
  }

  $effect(() => {
    const adapterChanged = adapterType !== syncedAdapterType
    const catalogChanged = optionSignature !== syncedOptionSignature

    if (!adapterChanged && !catalogChanged && modelName === effectiveModelName()) {
      return
    }

    const selection = splitProviderModelSelection(
      modelCatalog,
      adapterType,
      modelName,
      !adapterChanged,
    )

    baseModelId = selection.baseModelId
    customModelId = selection.customModelId
    syncedAdapterType = adapterType
    syncedOptionSignature = optionSignature

    const nextEffectiveModelName = selection.customModelId.trim() || selection.baseModelId
    if (nextEffectiveModelName && nextEffectiveModelName !== modelName) {
      onModelNameChange?.(nextEffectiveModelName)
    }
  })

  function handleSuggestedModelChange(value: string) {
    if (!value || value === customProviderModelOptionValue) {
      return
    }

    baseModelId = value
    onModelNameChange?.(customModelId.trim() || value)
  }

  function handleCustomModelInput(event: Event) {
    const value = (event.currentTarget as HTMLInputElement).value
    customModelId = value
    onModelNameChange?.(value.trim() || baseModelId)
  }
</script>

{#if options.length > 0}
  <div class="space-y-4">
    <div class="space-y-2">
      <Label>{i18nStore.t('agents.providerModelPicker.labels.suggested')}</Label>
      <Select.Root
        type="single"
        value={customModelId.trim() ? customProviderModelOptionValue : baseModelId}
        onValueChange={handleSuggestedModelChange}
      >
        <Select.Trigger class="w-full">
          {#if customModelId.trim()}
            {i18nStore.t('agents.providerModelPicker.labels.customOverride')}
          {:else}
            {options.find((option) => option.id === baseModelId)?.label ??
              i18nStore.t('agents.providerModelPicker.placeholders.selectModel')}
          {/if}
        </Select.Trigger>
        <Select.Content>
          {#each options as option (option.id)}
            <Select.Item value={option.id}>
              {option.label}
              {option.recommended
                ? i18nStore.t('agents.providerModelPicker.hints.recommendedSuffix')
                : ''}
              {option.preview ? i18nStore.t('agents.providerModelPicker.hints.previewSuffix') : ''}
            </Select.Item>
          {/each}
          <Select.Item value={customProviderModelOptionValue}>
            {i18nStore.t('agents.providerModelPicker.labels.customOverride')}
          </Select.Item>
        </Select.Content>
      </Select.Root>
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('agents.providerModelPicker.hints.pickAndOverride')}
      </p>
    </div>

    <div class="space-y-2">
      <Label for={inputId}>
        {i18nStore.t('agents.providerModelPicker.labels.customOverride')}
      </Label>
      <Input
        id={inputId}
        value={customModelId}
        placeholder={baseModelId
          ? baseModelId
          : i18nStore.t('agents.providerModelPicker.placeholders.enterModelId')}
        oninput={handleCustomModelInput}
      />
      <p class="text-muted-foreground text-xs">
        {i18nStore.t('agents.providerModelPicker.hints.leaveBlank')}
      </p>
    </div>
  </div>
{:else}
  <div class="space-y-2">
    <Label for={inputId}>{i18nStore.t('agents.providerModelPicker.labels.modelName')}</Label>
    <Input id={inputId} value={modelName} oninput={handleCustomModelInput} />
  </div>
{/if}
