<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { adapterIconPath } from '$lib/features/providers'
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

<div class="border-border bg-background flex items-center gap-3 rounded-lg border px-4 py-3">
  <div class="bg-muted flex size-8 shrink-0 items-center justify-center rounded-lg">
    {#if iconPath}
      <img src={iconPath} alt="" class="size-4" />
    {:else}
      <Zap class="text-foreground size-3.5" />
    {/if}
  </div>

  <div class="min-w-0 flex-1">
    <div class="flex items-center gap-2">
      <p class="text-foreground text-sm font-medium">{guide.title}</p>
      <span
        class={cn(
          'inline-flex items-center rounded px-1.5 py-0.5 text-[10px] leading-none font-medium',
          status.className,
        )}
      >
        {status.text}
      </span>
    </div>
    <p class="text-muted-foreground text-xs">
      CLI: {cliDetectionLabel[cliDetectionState(providers)]} · Auth: {authDetectionLabel[
        authDetectionState(providers)
      ]}
    </p>
  </div>

  <div class="flex shrink-0 gap-2">
    {#if availableProviders.length === 1}
      <Button
        size="sm"
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
          Use this
        {/if}
      </Button>
    {/if}
    <Button size="sm" variant="ghost" onclick={() => onOpenGuide(guide.key)}>Guide</Button>
  </div>
</div>
