<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityLabel,
  } from '$lib/features/providers'
  import { Badge } from '$ui/badge'

  let {
    providers,
    defaultProviderId,
  }: {
    providers: AgentProvider[]
    defaultProviderId: string | null | undefined
  } = $props()
</script>

{#if providers.length > 0}
  <div class="border-border divide-border divide-y rounded-lg border">
    {#each providers as provider (provider.id)}
      <div class="flex items-center justify-between gap-4 px-4 py-3">
        <div class="min-w-0">
          <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
          <p class="text-muted-foreground truncate text-xs">{provider.model_name}</p>
        </div>
        <div class="flex shrink-0 items-center gap-2">
          <Badge variant={providerAvailabilityBadgeVariant(provider.availability_state)}>
            {providerAvailabilityLabel(provider.availability_state)}
          </Badge>
          {#if defaultProviderId === provider.id}
            <Badge variant="secondary">Default</Badge>
          {/if}
        </div>
      </div>
    {/each}
  </div>
{/if}
