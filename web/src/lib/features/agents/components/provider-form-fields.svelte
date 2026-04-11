<script lang="ts">
  import type { AgentProviderModelCatalogEntry, Machine } from '$lib/api/contracts'
  import { Input } from '$ui/input'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import { Textarea } from '$ui/textarea'
  import { ChevronDown, ChevronUp } from '@lucide/svelte'
  import {
    createCustomFlatPricingConfig,
    isRoutedOfficialPricingConfig,
    parseProviderPricingConfig,
    providerPricingDetailRows,
    providerPricingStatusText,
    stringifyProviderPricingConfig,
    suggestPricingDraftValues,
  } from '../provider-pricing'
  import { providerAdapterOptions, providerPermissionProfileOptions } from '../provider-draft'
  import type { ProviderDraft, ProviderDraftField } from '../types'
  import ProviderAuthConfigField from './provider-auth-config-field.svelte'
  import ProviderPricingFields from './provider-pricing-fields.svelte'
  import ProviderModelPicker from './provider-model-picker.svelte'
  import ProviderReasoningFields from './provider-reasoning-fields.svelte'
  import ProviderSecretBindingsFields from './provider-secret-bindings-fields.svelte'
  import { providerModelReasoningCapability } from '../provider-model-options'

  let {
    draft,
    modelCatalog = [],
    machines = [],
    showCostFields = false,
    onFieldChange,
  }: {
    draft: ProviderDraft
    modelCatalog?: AgentProviderModelCatalogEntry[]
    machines?: Machine[]
    showCostFields?: boolean
    onFieldChange?: (field: ProviderDraftField, value: string) => void
  } = $props()

  let advancedOpen = $state(false)
  let syncedPricingModelKey = $state('')

  const pricingConfig = $derived(parseProviderPricingConfig(draft.pricingConfig))
  const pricingStatus = $derived(providerPricingStatusText(pricingConfig))
  const pricingRows = $derived(providerPricingDetailRows(pricingConfig))
  const routedOfficialPricing = $derived(isRoutedOfficialPricingConfig(pricingConfig))
  const reasoningCapability = $derived(
    providerModelReasoningCapability(modelCatalog, draft.adapterType, draft.modelName),
  )
  const effectiveReasoningEffort = $derived(
    draft.reasoningEffort.trim() || reasoningCapability?.defaultEffort || '',
  )

  const fieldValue = (event: Event) =>
    (event.currentTarget as HTMLInputElement | HTMLTextAreaElement).value

  $effect(() => {
    const modelKey = `${draft.adapterType}:${draft.modelName}`
    const suggested = suggestPricingDraftValues({
      modelCatalog,
      adapterType: draft.adapterType,
      modelName: draft.modelName,
      currentPricingConfig: pricingConfig,
      currentCostPerInputToken: draft.costPerInputToken,
      currentCostPerOutputToken: draft.costPerOutputToken,
    })
    if (!suggested) {
      syncedPricingModelKey = modelKey
      return
    }

    const nextSerializedPricing = stringifyProviderPricingConfig(suggested.pricingConfig)
    if (
      syncedPricingModelKey === modelKey &&
      draft.pricingConfig === nextSerializedPricing &&
      draft.costPerInputToken === suggested.costPerInputToken &&
      draft.costPerOutputToken === suggested.costPerOutputToken
    ) {
      return
    }

    syncedPricingModelKey = modelKey
    onFieldChange?.('pricingConfig', nextSerializedPricing)
    onFieldChange?.('costPerInputToken', suggested.costPerInputToken)
    onFieldChange?.('costPerOutputToken', suggested.costPerOutputToken)
  })

  function handlePricingFieldChange(
    field: 'costPerInputToken' | 'costPerOutputToken',
    value: string,
  ) {
    onFieldChange?.(field, value)

    const nextInput = field === 'costPerInputToken' ? value : draft.costPerInputToken
    const nextOutput = field === 'costPerOutputToken' ? value : draft.costPerOutputToken

    const nextPricing = createCustomFlatPricingConfig(
      nextInput.trim() ? Number(nextInput) / 1_000_000 : 0,
      nextOutput.trim() ? Number(nextOutput) / 1_000_000 : 0,
    )
    onFieldChange?.('pricingConfig', stringifyProviderPricingConfig(nextPricing))
  }

  $effect(() => {
    const capability = reasoningCapability
    const selectedReasoning = draft.reasoningEffort.trim()
    if (!selectedReasoning) {
      return
    }
    if (capability?.state !== 'available') {
      onFieldChange?.('reasoningEffort', '')
      return
    }
    if (!(capability.supportedEfforts ?? []).some((effort) => effort === selectedReasoning)) {
      onFieldChange?.('reasoningEffort', '')
    }
  })
</script>

<div class="space-y-4">
  <div class="space-y-2">
    <Label for="provider-name">Provider name</Label>
    <Input
      id="provider-name"
      value={draft.name}
      placeholder="Codex primary"
      oninput={(event) => onFieldChange?.('name', fieldValue(event))}
    />
  </div>

  <div class="grid gap-4 md:grid-cols-2">
    <div class="space-y-2">
      <Label>Adapter</Label>
      <Select.Root
        type="single"
        value={draft.adapterType}
        onValueChange={(value) => onFieldChange?.('adapterType', value || 'custom')}
      >
        <Select.Trigger class="w-full">
          {providerAdapterOptions.find((option) => option.value === draft.adapterType)?.label ??
            'Select adapter'}
        </Select.Trigger>
        <Select.Content>
          {#each providerAdapterOptions as option (option.value)}
            <Select.Item value={option.value}>{option.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>

    <div class="space-y-2">
      <Label>Permission mode</Label>
      <Select.Root
        type="single"
        value={draft.permissionProfile}
        onValueChange={(value) => onFieldChange?.('permissionProfile', value || 'unrestricted')}
      >
        <Select.Trigger class="w-full">
          {providerPermissionProfileOptions.find(
            (option) => option.value === draft.permissionProfile,
          )?.label ?? 'Select permission mode'}
        </Select.Trigger>
        <Select.Content>
          {#each providerPermissionProfileOptions as option (option.value)}
            <Select.Item value={option.value}>{option.label}</Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
      <p class="text-muted-foreground text-xs">
        {providerPermissionProfileOptions.find((option) => option.value === draft.permissionProfile)
          ?.description}
      </p>
    </div>

    <div class="space-y-2">
      <Label>Execution machine</Label>
      <Select.Root
        type="single"
        value={draft.machineId}
        onValueChange={(value) => onFieldChange?.('machineId', value || '')}
      >
        <Select.Trigger class="w-full">
          {machines.find((machine) => machine.id === draft.machineId)?.name ?? 'Select machine'}
        </Select.Trigger>
        <Select.Content>
          {#each machines as machine (machine.id)}
            <Select.Item value={machine.id}>
              {machine.name} · {machine.status} · {machine.host}
            </Select.Item>
          {/each}
        </Select.Content>
      </Select.Root>
    </div>
  </div>

  <div class="space-y-2">
    <ProviderModelPicker
      adapterType={draft.adapterType}
      modelName={draft.modelName}
      {modelCatalog}
      inputId="provider-model"
      onModelNameChange={(value) => onFieldChange?.('modelName', value)}
    />
  </div>

  <ProviderReasoningFields
    capability={reasoningCapability}
    selectedEffort={draft.reasoningEffort}
    effectiveEffort={effectiveReasoningEffort}
    onValueChange={(value) => onFieldChange?.('reasoningEffort', value)}
  />

  <div class="border-border border-t pt-2">
    <button
      type="button"
      class="text-muted-foreground hover:text-foreground flex w-full items-center gap-1.5 py-1 text-xs transition-colors"
      onclick={() => (advancedOpen = !advancedOpen)}
    >
      {#if advancedOpen}
        <ChevronUp class="size-3.5" />
      {:else}
        <ChevronDown class="size-3.5" />
      {/if}
      Advanced settings
    </button>

    {#if advancedOpen}
      <div class="mt-3 space-y-4">
        <div class="grid gap-4 md:grid-cols-2">
          <div class="space-y-2">
            <Label for="provider-cli-command">CLI command</Label>
            <Input
              id="provider-cli-command"
              value={draft.cliCommand}
              placeholder="codex"
              oninput={(event) => onFieldChange?.('cliCommand', fieldValue(event))}
            />
            <p class="text-muted-foreground text-xs">Leave empty to use the adapter default.</p>
          </div>

          <div class="space-y-2">
            <Label for="provider-model-temperature">Temperature</Label>
            <Input
              id="provider-model-temperature"
              type="number"
              min="0"
              step="0.01"
              value={draft.modelTemperature}
              oninput={(event) => onFieldChange?.('modelTemperature', fieldValue(event))}
            />
          </div>
        </div>

        <div class="space-y-2">
          <Label for="provider-cli-args">CLI args</Label>
          <Textarea
            id="provider-cli-args"
            rows={3}
            value={draft.cliArgs}
            placeholder={`app-server\n--listen\nstdio://`}
            oninput={(event) => onFieldChange?.('cliArgs', fieldValue(event))}
          />
          <p class="text-muted-foreground text-xs">One argument per line.</p>
          <p class="text-muted-foreground text-xs">
            OpenASE injects adapter-managed permission flags from Permission mode.
          </p>
        </div>

        <ProviderAuthConfigField
          value={draft.authConfig}
          onValueChange={(value) => onFieldChange?.('authConfig', value)}
        />

        <ProviderSecretBindingsFields
          adapterType={draft.adapterType}
          value={draft.secretBindings}
          onValueChange={(value) => onFieldChange?.('secretBindings', value)}
        />

        <div class="grid gap-4 md:grid-cols-2">
          <div class="space-y-2">
            <Label for="provider-model-max-tokens">Max tokens</Label>
            <Input
              id="provider-model-max-tokens"
              type="number"
              min="1"
              step="1"
              value={draft.modelMaxTokens}
              oninput={(event) => onFieldChange?.('modelMaxTokens', fieldValue(event))}
            />
          </div>

          <div class="space-y-2">
            <Label for="provider-max-parallel-runs">Max parallel runs</Label>
            <Input
              id="provider-max-parallel-runs"
              type="number"
              min="1"
              step="1"
              value={draft.maxParallelRuns}
              placeholder="Unlimited"
              oninput={(event) => onFieldChange?.('maxParallelRuns', fieldValue(event))}
            />
            <p class="text-muted-foreground text-xs">Leave blank for unlimited.</p>
          </div>

          {#if showCostFields}
            <ProviderPricingFields
              costPerInputToken={draft.costPerInputToken}
              costPerOutputToken={draft.costPerOutputToken}
              {routedOfficialPricing}
              {pricingStatus}
              {pricingRows}
              pricingNotes={pricingConfig?.notes ?? []}
              onPricingFieldChange={handlePricingFieldChange}
            />
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>
