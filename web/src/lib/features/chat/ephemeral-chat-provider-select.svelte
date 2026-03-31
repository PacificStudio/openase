<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { providerAvailabilityHeadline } from '$lib/features/providers'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'
  import {
    ephemeralChatCapabilityLabel,
    ephemeralChatCapabilityReason,
    hasAvailableEphemeralChat,
  } from './provider-options'

  let {
    providers,
    providerId,
    onProviderChange,
  }: {
    providers: AgentProvider[]
    providerId: string
    onProviderChange?: (providerId: string) => void
  } = $props()

  function buildProviderSummary(provider: AgentProvider) {
    return `${provider.name} · ${provider.adapter_type} · ${provider.model_name} · ${ephemeralChatCapabilityLabel(provider)}`
  }

  function providerReason(provider: AgentProvider) {
    return hasAvailableEphemeralChat(provider)
      ? null
      : providerAvailabilityHeadline(
          provider.capabilities.ephemeral_chat.state,
          ephemeralChatCapabilityReason(provider),
        )
  }

  function selectedProviderLabel() {
    const provider = providers.find((item) => item.id === providerId)
    if (provider) {
      return buildProviderSummary(provider)
    }

    return providers.length === 0 ? 'No chat provider available' : 'No available chat provider'
  }
</script>

<div class="mb-3 space-y-2">
  <Label class="text-[11px]">Provider</Label>
  <Select.Root
    type="single"
    value={providerId}
    disabled={providers.length === 0}
    onValueChange={(value) => onProviderChange?.(value || '')}
  >
    <Select.Trigger aria-label="Ephemeral Chat Provider" class="w-full text-left text-sm">
      {selectedProviderLabel()}
    </Select.Trigger>
    <Select.Content>
      {#each providers as provider (provider.id)}
        <Select.Item
          value={provider.id}
          disabled={!hasAvailableEphemeralChat(provider)}
          label={buildProviderSummary(provider)}
        >
          <div class="flex w-full items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="truncate font-medium">{provider.name}</div>
              <div class="text-muted-foreground text-xs">
                {provider.adapter_type} · {provider.model_name}
              </div>
              {#if providerReason(provider)}
                <div class="mt-1 text-[11px] text-amber-700">{providerReason(provider)}</div>
              {/if}
            </div>
            <div class="text-muted-foreground shrink-0 text-[11px]">
              {ephemeralChatCapabilityLabel(provider)}
            </div>
          </div>
        </Select.Item>
      {/each}
    </Select.Content>
  </Select.Root>
</div>
