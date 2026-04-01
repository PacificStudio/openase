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

<div class="border-border bg-background flex h-full flex-col rounded-xl border p-4">
  <div class="mb-3 flex items-start gap-3">
    <div class="bg-muted flex size-11 shrink-0 items-center justify-center rounded-xl">
      {#if iconPath}
        <img src={iconPath} alt="" class="size-6" />
      {:else}
        <Zap class="text-foreground size-5" />
      {/if}
    </div>
    <div class="min-w-0 flex-1 space-y-1">
      <div class="flex items-center gap-2">
        <p class="text-foreground text-sm font-semibold">{guide.title}</p>
        <span class={cn('text-xs', status.className)}>{status.text}</span>
      </div>
      <p class="text-muted-foreground text-xs">
        {providers.length > 0 ? `已注册 ${providers.length} 个 Provider` : '尚未注册 Provider'}
      </p>
    </div>
  </div>

  <div class="mb-3 grid grid-cols-2 gap-2 text-xs">
    <div class="bg-muted/50 rounded-lg px-3 py-2">
      <p class="text-muted-foreground">CLI</p>
      <p class="text-foreground mt-1 font-medium">
        {cliDetectionLabel[cliDetectionState(providers)]}
      </p>
    </div>
    <div class="bg-muted/50 rounded-lg px-3 py-2">
      <p class="text-muted-foreground">登录</p>
      <p class="text-foreground mt-1 font-medium">
        {authDetectionLabel[authDetectionState(providers)]}
      </p>
    </div>
    <div class="bg-muted/50 rounded-lg px-3 py-2">
      <p class="text-muted-foreground">推荐模型</p>
      <p class="text-foreground mt-1 font-medium">{guide.recommendedModel}</p>
    </div>
    <div class="bg-muted/50 rounded-lg px-3 py-2">
      <p class="text-muted-foreground">默认入口</p>
      <p class="text-foreground mt-1 font-medium">{guide.title}</p>
    </div>
  </div>

  {#if primaryProvider}
    <div class="border-border bg-muted/30 mb-3 rounded-lg border px-3 py-2 text-xs">
      <p class="text-foreground font-medium">
        {providerAvailabilityHeadline(
          primaryProvider.availability_state,
          primaryProvider.availability_reason,
        )}
      </p>
      <p class="text-muted-foreground mt-1">
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
          设置中...
        {:else if selectedId === availableProviders[0]?.id}
          已设为默认
        {:else}
          使用这个 Provider
        {/if}
      </Button>
    {:else}
      <Button size="sm" class="flex-1" variant="outline" onclick={() => onOpenGuide(guide.key)}>
        {availableProviders.length > 1 ? `查看 ${availableProviders.length} 个实例` : '继续配置'}
      </Button>
    {/if}

    <Button size="sm" variant="ghost" onclick={() => onOpenGuide(guide.key)}>指南</Button>
  </div>
</div>
