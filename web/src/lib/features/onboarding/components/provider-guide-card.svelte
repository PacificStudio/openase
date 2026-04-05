<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    adapterIconPath,
    providerAvailabilityDescription,
    providerAvailabilityHeadline,
  } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { Loader2, Zap } from '@lucide/svelte'
  import type { ProviderGuide } from '../provider-guides'
  import {
    authDetectionLabel,
    authDetectionState,
    cliDetectionLabel,
    cliDetectionState,
    primaryGuideProvider,
    providerStatus,
  } from '../provider-guides'

  let {
    guide,
    providers,
    selectedId,
    selecting,
    onSelectProvider,
    onOpenGuide,
  }: {
    guide: ProviderGuide
    providers: AgentProvider[]
    selectedId: string
    selecting: boolean
    onSelectProvider: (providerId: string) => Promise<void>
    onOpenGuide: (key: ProviderGuide['key']) => void
  } = $props()

  const availableProviders = $derived(
    providers.filter(
      (provider) =>
        provider.availability_state === 'available' || provider.availability_state === 'ready',
    ),
  )
  const primaryProvider = $derived(primaryGuideProvider(providers, selectedId))
  const status = $derived(providerStatus(primaryProvider))
  const iconPath = $derived(adapterIconPath(guide.adapterTypes[0] ?? ''))
</script>

<div class="border-border bg-background flex h-full flex-col rounded-lg border p-4">
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
        <p class="text-foreground text-sm font-semibold leading-tight">{guide.title}</p>
        <span class={cn('inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium leading-none', status.className)}>
          {status.text}
        </span>
      </div>
      <p class="text-muted-foreground mt-0.5 text-xs">{guide.recommendedModel}</p>
    </div>
  </div>

  <div class="mb-3 space-y-1.5 text-xs">
    <div class="flex items-center justify-between">
      <span class="text-muted-foreground">CLI</span>
      <span class="text-foreground font-medium">{cliDetectionLabel[cliDetectionState(providers)]}</span>
    </div>
    <div class="flex items-center justify-between">
      <span class="text-muted-foreground">Auth</span>
      <span class="text-foreground font-medium">{authDetectionLabel[authDetectionState(providers)]}</span>
    </div>
    <div class="flex items-center justify-between">
      <span class="text-muted-foreground">Instances</span>
      <span class="text-foreground font-medium">{providers.length}</span>
    </div>
  </div>

  {#if primaryProvider}
    <div class="bg-muted/40 mb-3 rounded-md px-3 py-2 text-xs">
      <p class="text-foreground font-medium leading-snug">
        {providerAvailabilityHeadline(
          primaryProvider.availability_state,
          primaryProvider.availability_reason,
        )}
      </p>
      <p class="text-muted-foreground mt-0.5 leading-snug">
        {providerAvailabilityDescription(
          primaryProvider.availability_state,
          primaryProvider.availability_reason,
        )}
      </p>
    </div>
  {/if}

  <div class="mt-auto flex gap-2">
    {#if availableProviders.length === 1}
      <Button
        size="sm"
        class="flex-1"
        variant={selectedId === availableProviders[0]?.id ? 'default' : 'outline'}
        disabled={selecting}
        onclick={() => void onSelectProvider(availableProviders[0]!.id)}
      >
        {#if selecting && selectedId === availableProviders[0]?.id}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          Setting…
        {:else if selectedId === availableProviders[0]?.id}
          Default
        {:else}
          Use this provider
        {/if}
      </Button>
    {:else}
      <Button size="sm" class="flex-1" variant="outline" onclick={() => onOpenGuide(guide.key)}>
        {availableProviders.length > 1
          ? `${availableProviders.length} instances`
          : 'Setup guide'}
      </Button>
    {/if}

    <Button size="sm" variant="ghost" onclick={() => onOpenGuide(guide.key)}>Guide</Button>
  </div>
</div>
