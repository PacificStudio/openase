<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { adapterDisplayName, adapterIconPath } from '$lib/features/providers'
  import {
    providerAvailabilityCheckedAtText,
    providerAvailabilityHeadline,
  } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { CheckCircle2, Loader2, RefreshCcw, Zap } from '@lucide/svelte'
  import { guideForProvider, isProviderAvailable, providerStatus } from '../provider-guides'

  let {
    provider,
    selectedId,
    selecting,
    refreshing,
    onSelectProvider,
    onOpenGuide,
    onRefresh,
  }: {
    provider: AgentProvider
    selectedId: string
    selecting: boolean
    refreshing: boolean
    onSelectProvider: (providerId: string) => Promise<void>
    onOpenGuide: (providerId: string) => void
    onRefresh: (providerId: string) => Promise<void>
  } = $props()

  const status = $derived(providerStatus(provider))
  const checkedAt = $derived(providerAvailabilityCheckedAtText(provider.availability_checked_at))
  const guide = $derived(guideForProvider(provider))
  const available = $derived(isProviderAvailable(provider))
  const isSelected = $derived(selectedId === provider.id)
  const iconPath = $derived(adapterIconPath(provider.adapter_type))
</script>

<div
  class={cn(
    'border-border bg-card rounded-lg border p-4 transition-colors',
    isSelected ? 'border-primary bg-primary/5 ring-primary/20 ring-1' : '',
  )}
>
  <div class="mb-3 flex items-start gap-3">
    <div class="bg-muted flex size-9 shrink-0 items-center justify-center rounded-lg">
      {#if iconPath}
        <img src={iconPath} alt="" class="size-5" />
      {:else}
        <Zap class="text-foreground size-4" />
      {/if}
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <p class="text-foreground truncate text-sm leading-tight font-semibold">{provider.name}</p>
        <span
          class={cn(
            'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] leading-none font-medium',
            status.className,
          )}
        >
          {status.text}
        </span>
        {#if isSelected}
          <CheckCircle2 class="text-primary ml-auto size-4 shrink-0" />
        {/if}
      </div>
      <p class="text-muted-foreground mt-0.5 text-xs">
        {adapterDisplayName(provider.adapter_type)} · {provider.model_name || 'Model not set'}
      </p>
    </div>
  </div>

  <div class="mb-3 space-y-1.5 text-xs">
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted-foreground">Machine</span>
      <span class="text-foreground truncate text-right font-medium"
        >{provider.machine_name || '—'}</span
      >
    </div>
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted-foreground">CLI</span>
      <span class="text-foreground truncate text-right font-mono text-[11px]"
        >{provider.cli_command || '—'}</span
      >
    </div>
    {#if checkedAt}
      <div class="flex items-center justify-between gap-3">
        <span class="text-muted-foreground">Checked</span>
        <span class="text-muted-foreground text-right">{checkedAt}</span>
      </div>
    {/if}
  </div>

  <div class="bg-muted/40 mb-3 rounded-md px-3 py-2 text-xs">
    <p class="text-foreground leading-snug font-medium">
      {providerAvailabilityHeadline(provider.availability_state, provider.availability_reason)}
    </p>
  </div>

  <div class="flex flex-wrap gap-2">
    {#if available}
      <Button
        size="sm"
        variant={isSelected ? 'default' : 'outline'}
        disabled={selecting}
        onclick={() => void onSelectProvider(provider.id)}
      >
        {#if selecting && isSelected}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          Setting…
        {:else if isSelected}
          Default
        {:else}
          Use this provider
        {/if}
      </Button>
    {:else}
      {#if guide}
        <Button size="sm" variant="outline" onclick={() => onOpenGuide(provider.id)}>
          {guide.title} guide
        </Button>
      {/if}
      <Button
        size="sm"
        variant="ghost"
        disabled={refreshing}
        onclick={() => void onRefresh(provider.id)}
      >
        {#if refreshing}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          Checking…
        {:else}
          <RefreshCcw class="mr-1.5 size-3.5" />
          Recheck
        {/if}
      </Button>
    {/if}
  </div>
</div>
