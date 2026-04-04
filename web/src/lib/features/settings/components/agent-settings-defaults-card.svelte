<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import {
    adapterDisplayName,
    adapterIconPath,
    ProviderAvailabilityBadge,
    ProviderRateLimitDisplay,
    summarizeProviderRateLimit,
  } from '$lib/features/providers'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Settings, Star, Wrench } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import type { ProviderOption } from './agent-settings-model'

  let {
    providers,
    selectedDefaultProviderId,
    orgDefaultProviderId = null,
    orgDefaultProviderName = null,
    machineCount = 0,
    saving = false,
    onSelectionChange,
    onSave,
    onConfigure,
    onAddProvider,
  }: {
    providers: ProviderOption[]
    selectedDefaultProviderId: string
    orgDefaultProviderId?: string | null
    orgDefaultProviderName?: string | null
    machineCount?: number
    saving?: boolean
    onSelectionChange?: (value: string) => void
    onSave?: () => void
    onConfigure?: (providerId: string) => void
    onAddProvider?: () => void
  } = $props()

  const effectiveDefaultId = $derived(selectedDefaultProviderId || orgDefaultProviderId || '')
  const hasUnsavedChange = $derived(
    selectedDefaultProviderId !== (appStore.currentProject?.default_agent_provider_id ?? ''),
  )
  const canAddProvider = $derived(machineCount > 0)
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between gap-3">
    <div>
      <h3 class="text-foreground text-sm font-semibold">Agent Providers</h3>
      <p class="text-muted-foreground mt-0.5 text-xs">
        Click the star to set the project default. Configure adapter and model settings per
        provider.
      </p>
    </div>
    <div class="flex items-center gap-2">
      <Button variant="outline" size="sm" onclick={onAddProvider} disabled={!canAddProvider}>
        Add provider
      </Button>
      {#if hasUnsavedChange}
        <Button size="sm" onclick={onSave} disabled={saving}>
          {saving ? 'Saving…' : 'Save default'}
        </Button>
      {/if}
    </div>
  </div>

  {#if !canAddProvider}
    <p class="text-muted-foreground text-xs">
      Register an execution machine in this organization before creating a provider from project
      settings.
    </p>
  {/if}

  {#if providers.length === 0}
    <div
      class="border-border bg-card text-muted-foreground rounded-xl border border-dashed px-4 py-10 text-center text-sm"
    >
      {#if canAddProvider}
        No providers registered for this organization yet.
      {:else}
        No providers or execution machines are registered for this organization yet.
      {/if}
    </div>
  {:else}
    <div class="space-y-2">
      {#each providers as provider (provider.id)}
        {@const iconPath = adapterIconPath(provider.adapterType)}
        {@const isDefault = provider.id === effectiveDefaultId}
        {@const isProjectDefault = provider.id === selectedDefaultProviderId}
        {@const isOrgDefault = provider.id === orgDefaultProviderId}
        {@const rateLimit = summarizeProviderRateLimit(provider)}
        <div
          class={cn(
            'border-border/60 bg-card/60 flex items-center gap-3 rounded-xl border px-4 py-3',
            isDefault && 'border-primary/30 bg-primary/5',
          )}
        >
          <button
            type="button"
            class={cn(
              'shrink-0 transition-colors',
              isProjectDefault ? 'text-amber-500' : 'text-muted-foreground/40 hover:text-amber-400',
            )}
            title={isProjectDefault
              ? 'Clear project default (inherit org)'
              : `Set ${provider.name} as project default`}
            onclick={() => {
              onSelectionChange?.(isProjectDefault ? '' : provider.id)
            }}
          >
            <Star class={cn('size-4', isProjectDefault && 'fill-current')} />
          </button>

          <div class="bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg">
            {#if iconPath}
              <img src={iconPath} alt="" class="size-5" />
            {:else}
              <Wrench class="text-muted-foreground size-4" />
            {/if}
          </div>

          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-1.5">
              <span class="text-foreground text-sm font-semibold">{provider.name}</span>
              <ProviderAvailabilityBadge
                availabilityState={provider.availabilityState}
                availabilityReason={provider.availabilityReason}
                availabilityCheckedAt={provider.availabilityCheckedAt}
                class="text-[10px]"
              />
              {#if isProjectDefault}
                <Badge variant="outline" class="text-[10px]">Project default</Badge>
              {:else if isOrgDefault && !selectedDefaultProviderId}
                <Badge variant="secondary" class="text-[10px]">Org default</Badge>
              {/if}
            </div>
            <div
              class="text-muted-foreground mt-0.5 flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs"
            >
              <span>{adapterDisplayName(provider.adapterType)}</span>
              <span class="font-mono">{provider.modelName}</span>
              <span>{provider.machineName}</span>
              <span>{provider.agentCount} agent{provider.agentCount !== 1 ? 's' : ''}</span>
            </div>
            {#if rateLimit}
              <div class="mt-2">
                <ProviderRateLimitDisplay {rateLimit} />
              </div>
            {/if}
          </div>

          <Button
            variant="ghost"
            size="icon-xs"
            title="Configure provider"
            aria-label="Configure provider"
            onclick={() => onConfigure?.(provider.id)}
          >
            <Settings class="size-3.5" />
          </Button>
        </div>
      {/each}
    </div>

    {#if !selectedDefaultProviderId && orgDefaultProviderName}
      <p class="text-muted-foreground text-xs">
        Inheriting org default: {orgDefaultProviderName}. Star a provider to override.
      </p>
    {/if}
  {/if}
</div>
