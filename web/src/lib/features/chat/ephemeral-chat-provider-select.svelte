<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { providerAvailabilityHeadline, providerAvailabilityLabel } from '$lib/features/providers'
  import { Label } from '$ui/label'
  import * as Select from '$ui/select'

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
    return `${provider.name} · ${provider.adapter_type} · ${provider.model_name} · ${providerAvailabilityLabel(provider.availability_state)}`
  }

  function providerReason(provider: AgentProvider) {
    return provider.available
      ? null
      : providerAvailabilityHeadline(provider.availability_state, provider.availability_reason)
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
          disabled={!provider.available}
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
              {providerAvailabilityLabel(provider.availability_state)}
            </div>
          </div>
        </Select.Item>
      {/each}
    </Select.Content>
  </Select.Root>
</div>
