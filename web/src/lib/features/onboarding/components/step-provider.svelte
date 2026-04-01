<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import { listProviders, refreshMachineHealth, updateProject } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Loader2, RefreshCcw } from '@lucide/svelte'
  import type { ProviderState } from '../types'
  import ProviderEmptyState from './provider-empty-state.svelte'
  import ProviderGuideCard from './provider-guide-card.svelte'
  import RegisteredProviderCard from './registered-provider-card.svelte'
  import ProviderGuideSheet from './provider-guide-sheet.svelte'
  import {
    guideForProvider,
    guideProviders,
    providerGuides,
    uniqueMachineIds,
  } from '../provider-guides'

  let {
    projectId,
    orgId,
    initialState,
    onComplete,
    onStateChange = () => {},
  }: {
    projectId: string
    orgId: string
    initialState: ProviderState
    onComplete: (providerId: string) => void
    onStateChange?: (nextState: ProviderState) => void
  } = $props()

  let selecting = $state(false)
  let selectedId = $state(untrack(() => initialState.selectedProviderId))
  let providers = $state<AgentProvider[]>(untrack(() => [...initialState.providers]))
  let activeGuideKey = $state<(typeof providerGuides)[number]['key'] | null>(null)
  let guideOpen = $state(false)
  let refreshingTokens = $state<string[]>([])
  let autoRefreshStarted = $state(false)

  const activeGuide = $derived(providerGuides.find((guide) => guide.key === activeGuideKey) ?? null)
  const activeGuideProviders = $derived(activeGuide ? guideProviders(providers, activeGuide) : [])

  $effect(() => {
    providers = [...initialState.providers]
    selectedId = initialState.selectedProviderId
  })

  $effect(() => {
    if (autoRefreshStarted) return
    autoRefreshStarted = true
    void refreshProviders(uniqueMachineIds(providers), true)
  })

  function isRefreshing(machineIds: string[]): boolean {
    const token = machineIds.length > 0 ? machineIds.slice().sort().join('|') : 'provider-list'
    return refreshingTokens.includes(token)
  }

  function syncProviderState(nextProviders: AgentProvider[], nextSelectedId = selectedId) {
    providers = nextProviders
    onStateChange({ providers: nextProviders, selectedProviderId: nextSelectedId })
  }

  async function refreshProviders(machineIds: string[], silent = false) {
    const token = machineIds.length > 0 ? machineIds.slice().sort().join('|') : 'provider-list'
    if (refreshingTokens.includes(token)) return

    refreshingTokens = [...refreshingTokens, token]
    try {
      await Promise.allSettled(machineIds.map((machineId) => refreshMachineHealth(machineId)))
      const payload = await listProviders(orgId)
      syncProviderState(payload.providers)
      if (!silent) toastStore.success('已重新检测 Provider 可用性。')
    } catch (caughtError) {
      if (!silent) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : '重新检测 Provider 失败。',
        )
      }
    } finally {
      refreshingTokens = refreshingTokens.filter((value) => value !== token)
    }
  }

  async function handleSelectProvider(providerId: string) {
    if (selecting) return
    const previousSelectedId = selectedId
    selectedId = providerId
    selecting = true
    try {
      const payload = await updateProject(projectId, { default_agent_provider_id: providerId })
      appStore.currentProject = payload.project
      syncProviderState(providers, providerId)
      toastStore.success('已设为默认 Provider。')
      onComplete(providerId)
    } catch (caughtError) {
      selectedId = previousSelectedId
      toastStore.error(
        caughtError instanceof ApiError ? caughtError.detail : '设置默认 Provider 失败。',
      )
    } finally {
      selecting = false
    }
  }

  function openGuideForKey(key: (typeof providerGuides)[number]['key']) {
    activeGuideKey = key
    guideOpen = true
  }

  function openGuideForProvider(providerId: string) {
    const provider = providers.find((item) => item.id === providerId)
    const guide = provider ? guideForProvider(provider) : null
    if (guide) openGuideForKey(guide.key)
  }

  async function copyCommand(command: string) {
    try {
      await navigator.clipboard.writeText(command)
      toastStore.success('命令已复制。')
    } catch {
      toastStore.error('复制命令失败。')
    }
  }
</script>

<div class="space-y-6">
  <div class="border-border bg-card space-y-4 rounded-xl border p-4">
    <div class="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
      <div class="space-y-1">
        <h4 class="text-foreground text-sm font-semibold">内置 CLI 配置向导</h4>
        <p class="text-muted-foreground text-xs">
          进入此步骤时会基于已注册 Provider 的绑定机器重新拉取可用性。即使你还没注册
          Provider，也可以先按下面的官方指引完成安装、登录与验证。
        </p>
      </div>
      <Button
        variant="outline"
        size="sm"
        disabled={isRefreshing(uniqueMachineIds(providers))}
        onclick={() => void refreshProviders(uniqueMachineIds(providers))}
      >
        {#if isRefreshing(uniqueMachineIds(providers))}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          检测中...
        {:else}
          <RefreshCcw class="mr-1.5 size-3.5" />
          重新检测全部 Provider
        {/if}
      </Button>
    </div>

    <div class="grid grid-cols-1 gap-3 xl:grid-cols-3">
      {#each providerGuides as guide (guide.key)}
        <ProviderGuideCard
          {guide}
          providers={guideProviders(providers, guide)}
          {selectedId}
          {selecting}
          onSelectProvider={handleSelectProvider}
          onOpenGuide={openGuideForKey}
        />
      {/each}
    </div>
  </div>

  <div class="space-y-3">
    <div class="space-y-1">
      <h4 class="text-foreground text-sm font-semibold">已注册 Provider</h4>
      <p class="text-muted-foreground text-xs">
        已注册的 Provider 会继续保留精确的机器、模型与可用性信息。不可用实例可以直接回到对应 CLI
        指南，并在完成配置后重新检测。
      </p>
    </div>

    {#if providers.length === 0}
      <ProviderEmptyState {orgId} />
    {:else}
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {#each providers as provider (provider.id)}
          <RegisteredProviderCard
            {provider}
            {selectedId}
            {selecting}
            refreshing={isRefreshing(provider.machine_id ? [provider.machine_id] : [])}
            onSelectProvider={handleSelectProvider}
            onOpenGuide={openGuideForProvider}
            onRefresh={async (providerId) => {
              const target = providers.find((item) => item.id === providerId)
              await refreshProviders(target?.machine_id ? [target.machine_id] : [])
            }}
          />
        {/each}
      </div>
    {/if}
  </div>
</div>

<ProviderGuideSheet
  bind:open={guideOpen}
  {orgId}
  {activeGuide}
  matchingProviders={activeGuideProviders}
  {selectedId}
  {selecting}
  {isRefreshing}
  onClose={() => {
    guideOpen = false
  }}
  onCopyCommand={copyCommand}
  onRefresh={refreshProviders}
  onSelectProvider={handleSelectProvider}
/>
