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
      if (!silent) toastStore.success('Provider availability rechecked.')
    } catch (caughtError) {
      if (!silent) {
        toastStore.error(
          caughtError instanceof ApiError
            ? caughtError.detail
            : 'Failed to recheck provider availability.',
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
      toastStore.success('Set as the default provider.')
      onComplete(providerId)
    } catch (caughtError) {
      selectedId = previousSelectedId
      toastStore.error(
        caughtError instanceof ApiError
          ? caughtError.detail
          : 'Failed to set the default provider.',
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
      toastStore.success('Command copied.')
    } catch {
      toastStore.error('Failed to copy command.')
    }
  }
</script>

<div class="space-y-8">
  <section>
    <div class="mb-3 flex items-center justify-between gap-3">
      <h4 class="text-foreground text-sm font-semibold">CLI setup guide</h4>
      <Button
        variant="outline"
        size="sm"
        class="shrink-0"
        disabled={isRefreshing(uniqueMachineIds(providers))}
        onclick={() => void refreshProviders(uniqueMachineIds(providers))}
      >
        {#if isRefreshing(uniqueMachineIds(providers))}
          <Loader2 class="mr-1.5 size-3.5 animate-spin" />
          Checking…
        {:else}
          <RefreshCcw class="mr-1.5 size-3.5" />
          Recheck all
        {/if}
      </Button>
    </div>

    <div class="flex flex-col gap-2">
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
  </section>

  <section>
    <h4 class="text-foreground mb-3 text-sm font-semibold">Registered providers</h4>

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
  </section>
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
