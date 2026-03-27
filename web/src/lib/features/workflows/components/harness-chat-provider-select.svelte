<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
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

  function providerLabel(provider: AgentProvider) {
    return provider.model_name ? `${provider.name} · ${provider.model_name}` : provider.name
  }

  function selectedProviderLabel() {
    if (!providerId) {
      return 'No chat provider available'
    }

    const provider = providers.find((item) => item.id === providerId)
    return provider ? providerLabel(provider) : 'No chat provider available'
  }
</script>

<div class="mb-3 space-y-2">
  <Label class="text-[11px]">Provider</Label>
  <Select.Root
    type="single"
    value={providerId}
    onValueChange={(value) => onProviderChange?.(value || '')}
  >
    <Select.Trigger class="w-full text-left text-sm">{selectedProviderLabel()}</Select.Trigger>
    <Select.Content>
      {#each providers as provider (provider.id)}
        <Select.Item value={provider.id}>{providerLabel(provider)}</Select.Item>
      {/each}
    </Select.Content>
  </Select.Root>
</div>
