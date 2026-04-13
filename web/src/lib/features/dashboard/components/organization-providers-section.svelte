<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    providerAvailabilityBadgeVariant,
    providerAvailabilityLabel,
    summarizeAgentProviderRateLimit,
    ProviderRateLimitDisplay,
  } from '$lib/features/providers'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { i18nStore } from '$lib/i18n/store.svelte'

  let {
    providers = [],
    defaultProviderId = null,
    onAddProvider,
  }: {
    providers?: AgentProvider[]
    defaultProviderId?: string | null
    onAddProvider?: () => void
  } = $props()
</script>

<div class="space-y-3">
  <div class="flex items-center justify-between gap-4">
    <div>
      <h3 class="text-foreground text-sm font-semibold">
        {i18nStore.t('dashboard.organizationProviders.labels.heading')}
      </h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        {i18nStore.t('dashboard.organizationProviders.labels.description')}
      </p>
    </div>
    {#if onAddProvider}
      <Button variant="outline" size="sm" onclick={onAddProvider}>
        {i18nStore.t('dashboard.organizationProviders.actions.addProvider')}
      </Button>
    {/if}
  </div>

  {#if providers.length > 0}
    <div class="border-border divide-border divide-y rounded-md border">
      {#each providers as provider (provider.id)}
        {@const rateLimit = summarizeAgentProviderRateLimit(provider)}
        <div class="flex items-center justify-between gap-4 px-4 py-3">
          <div class="min-w-0 flex-1">
            <p class="text-foreground truncate text-sm font-medium">{provider.name}</p>
            <p class="text-muted-foreground truncate text-xs">{provider.model_name}</p>
            {#if rateLimit}
              <div class="mt-2">
                <ProviderRateLimitDisplay {rateLimit} />
              </div>
            {/if}
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <Badge variant={providerAvailabilityBadgeVariant(provider.availability_state)}>
              {providerAvailabilityLabel(provider.availability_state)}
            </Badge>
            {#if defaultProviderId === provider.id}
              <Badge variant="secondary">
                {i18nStore.t('dashboard.organizationProviders.badge.default')}
              </Badge>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <button
      type="button"
      class="border-border hover:border-border/80 hover:bg-muted/30 w-full rounded-md border border-dashed px-4 py-8 text-center transition-colors"
      onclick={onAddProvider}
    >
      <p class="text-muted-foreground text-sm">
        {i18nStore.t('dashboard.organizationProviders.messages.noProviders')}
      </p>
      <p class="text-foreground mt-1 text-sm font-medium">
        {i18nStore.t('dashboard.organizationProviders.messages.addProviderHint')}
      </p>
    </button>
  {/if}
</div>
