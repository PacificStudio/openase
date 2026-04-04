<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { adapterDisplayName, adapterIconPath } from '$lib/features/providers'
  import { providerAvailabilityCheckedAtText } from '$lib/features/providers'
  import {
    providerAvailabilityDescription,
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
    'border-border bg-card rounded-xl border p-4 transition-all',
    isSelected ? 'border-primary bg-primary/5 ring-primary/20 ring-1' : '',
  )}
>
  <div class="mb-3 flex items-start gap-3">
    <div class="bg-muted flex size-10 shrink-0 items-center justify-center rounded-xl">
      {#if iconPath}
        <img src={iconPath} alt="" class="size-6" />
      {:else}
        <Zap class="text-foreground size-5" />
      {/if}
    </div>

    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-2">
        <p class="text-foreground truncate text-sm font-semibold">{provider.name}</p>
        <span class={cn('text-xs', status.className)}>{status.text}</span>
      </div>
      <p class="text-muted-foreground mt-1 text-xs">
        {adapterDisplayName(provider.adapter_type)} · {provider.model_name || '未设置模型'}
      </p>
    </div>

    {#if isSelected}
      <div
        class="bg-primary text-primary-foreground flex size-5 items-center justify-center rounded-full"
      >
        <CheckCircle2 class="size-3.5" />
      </div>
    {/if}
  </div>

  <div class="mb-3 space-y-1 text-xs">
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted-foreground">机器</span>
      <span class="text-foreground text-right">{provider.machine_name || '—'}</span>
    </div>
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted-foreground">CLI</span>
      <span class="text-foreground text-right">{provider.cli_command || '—'}</span>
    </div>
    {#if checkedAt}
      <div class="flex items-center justify-between gap-3">
        <span class="text-muted-foreground">最近检测</span>
        <span class="text-foreground text-right">{checkedAt}</span>
      </div>
    {/if}
  </div>

  <div class="border-border bg-muted/30 mb-3 rounded-lg border px-3 py-2 text-xs">
    <p class="text-foreground font-medium">
      {providerAvailabilityHeadline(provider.availability_state, provider.availability_reason)}
    </p>
    <p class="text-muted-foreground mt-1">
      {providerAvailabilityDescription(provider.availability_state, provider.availability_reason)}
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
          设置中...
        {:else if isSelected}
          已设为默认
        {:else}
          使用这个 Provider
        {/if}
      </Button>
    {:else}
      {#if guide}
        <Button size="sm" variant="outline" onclick={() => onOpenGuide(provider.id)}>
          查看 {guide.title} 指南
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
          检测中...
        {:else}
          <RefreshCcw class="mr-1.5 size-3.5" />
          重新检测
        {/if}
      </Button>
    {/if}
  </div>
</div>
