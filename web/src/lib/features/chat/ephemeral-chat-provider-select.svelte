<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { providerAvailabilityHeadline } from '$lib/features/providers'
  import * as Select from '$ui/select'
  import * as Tooltip from '$ui/tooltip'
  import {
    hasAvailableProviderCapability,
    providerCapabilityLabel,
    providerCapabilityReason,
    providerCapabilityState,
    type ProviderCapabilityName,
  } from './provider-options'

  let {
    providers,
    providerId,
    capability = 'ephemeral_chat',
    disabled = false,
    switchHint = '',
    onProviderChange,
  }: {
    providers: AgentProvider[]
    providerId: string
    capability?: ProviderCapabilityName
    disabled?: boolean
    switchHint?: string
    onProviderChange?: (providerId: string) => void
  } = $props()

  function providerReason(provider: AgentProvider) {
    return hasAvailableProviderCapability(provider, capability)
      ? null
      : providerAvailabilityHeadline(
          providerCapabilityState(provider, capability),
          providerCapabilityReason(provider, capability),
        )
  }

  function selectedModelLabel() {
    const provider = providers.find((item) => item.id === providerId)
    if (provider) {
      return provider.model_name
    }

    return 'No model'
  }
</script>

<Select.Root
  type="single"
  value={providerId}
  disabled={disabled || providers.length === 0}
  onValueChange={(value) => onProviderChange?.(value || '')}
>
  {#if switchHint}
    <Tooltip.Root>
      <Tooltip.Trigger>
        {#snippet child({ props })}
          <span {...props} class="inline-flex">
            <Select.Trigger
              aria-label="Chat model"
              class="text-muted-foreground hover:bg-muted hover:text-foreground h-7 w-auto gap-1 rounded-md border-none bg-transparent px-2 text-[11px] shadow-none"
            >
              {selectedModelLabel()}
            </Select.Trigger>
          </span>
        {/snippet}
      </Tooltip.Trigger>
      <Tooltip.Content side="bottom" class="max-w-52 text-xs">{switchHint}</Tooltip.Content>
    </Tooltip.Root>
  {:else}
    <Select.Trigger
      aria-label="Chat model"
      class="text-muted-foreground hover:bg-muted hover:text-foreground h-7 w-auto gap-1 rounded-md border-none bg-transparent px-2 text-[11px] shadow-none"
    >
      {selectedModelLabel()}
    </Select.Trigger>
  {/if}
  <Select.Content align="start" class="min-w-48">
    {#each providers as provider (provider.id)}
      <Select.Item
        value={provider.id}
        disabled={!hasAvailableProviderCapability(provider, capability)}
        label={provider.model_name}
      >
        <div class="flex w-full items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="truncate text-sm">{provider.model_name}</div>
            <div class="text-muted-foreground text-[11px]">
              {provider.name} · {provider.adapter_type}
            </div>
            {#if providerReason(provider)}
              <div class="mt-0.5 text-[10px] text-amber-700">{providerReason(provider)}</div>
            {/if}
          </div>
          <div class="text-muted-foreground shrink-0 text-[10px]">
            {providerCapabilityLabel(provider, capability)}
          </div>
        </div>
      </Select.Item>
    {/each}
  </Select.Content>
</Select.Root>
