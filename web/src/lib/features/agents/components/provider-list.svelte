<script lang="ts">
  import {
    adapterIconPath,
    summarizeProviderRateLimit,
    ProviderAvailabilityBadge,
    ProviderRateLimitDisplay,
  } from '$lib/features/providers'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import * as Card from '$ui/card'
  import { Settings, Wrench } from '@lucide/svelte'
  import type { ProviderConfig } from '../types'

  let {
    providers,
    onConfigure,
  }: {
    providers: ProviderConfig[]
    onConfigure?: (provider: ProviderConfig) => void
  } = $props()
</script>

<div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
  {#each providers as provider (provider.id)}
    {@const iconPath = adapterIconPath(provider.adapterType)}
    {@const rateLimit = summarizeProviderRateLimit(provider)}
    <Card.Root class="hover:border-border/80 transition-colors">
      <Card.Header class="flex-row items-start justify-between gap-3 space-y-0 pb-3">
        <div class="flex items-center gap-2.5">
          <div class="bg-muted flex size-8 items-center justify-center rounded-md">
            {#if iconPath}
              <img src={iconPath} alt="" class="size-5" />
            {:else}
              <Wrench class="text-muted-foreground size-4" />
            {/if}
          </div>
          <div>
            <div class="flex items-center gap-2">
              <Card.Title class="text-sm">{provider.name}</Card.Title>
              {#if provider.isDefault}
                <Badge variant="outline" class="text-[10px]">Default</Badge>
              {/if}
              <ProviderAvailabilityBadge
                availabilityState={provider.availabilityState}
                availabilityReason={provider.availabilityReason}
                availabilityCheckedAt={provider.availabilityCheckedAt}
                class="text-[10px]"
              />
            </div>
            <Card.Description class="text-xs">{provider.adapterType}</Card.Description>
          </div>
        </div>
      </Card.Header>
      <Card.Content class="space-y-2 pt-0">
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Model</span>
          <span class="text-foreground font-mono">{provider.modelName}</span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Permission</span>
          <span
            class="text-foreground text-[11px] font-medium capitalize {provider.permissionProfile ===
            'unrestricted'
              ? 'text-amber-600 dark:text-amber-400'
              : ''}"
          >
            {provider.permissionProfile}
          </span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Machine</span>
          <span class="text-foreground">{provider.machineName}</span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Agents</span>
          <span class="text-foreground tabular-nums">{provider.agentCount}</span>
        </div>
        {#if rateLimit}
          <ProviderRateLimitDisplay {rateLimit} />
        {/if}
      </Card.Content>
      <Card.Footer class="pt-2">
        <Button
          variant="outline"
          size="sm"
          class="w-full"
          onclick={() => onConfigure?.(provider)}
          title="Configure provider"
          aria-label="Configure provider"
        >
          <Settings class="size-3.5" />
          Configure
        </Button>
      </Card.Footer>
    </Card.Root>
  {/each}
</div>
