<script lang="ts">
  import { adapterIconPath } from '$lib/features/providers'
  import { cn } from '$lib/utils'
  import { ApiError } from '$lib/api/client'
  import { deleteAgent, pauseAgent, resumeAgent, updateAgent } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import { Check, Pencil, Wrench, X } from '@lucide/svelte'
  import AgentDrawerContent from './agent-drawer-content.svelte'
  import {
    agentStatusDot,
    agentStatusLabel,
    agentStatusVariant,
    canPauseAgent,
    canResumeAgent,
  } from './agent-drawer-state'
  import type { AgentInstance } from '../types'

  let {
    open = $bindable(false),
    agent,
    providers = [],
    onOpenChange,
    onDeleted,
    onUpdated,
  }: {
    open?: boolean
    agent: AgentInstance | null
    providers?: AgentProvider[]
    onOpenChange?: (open: boolean) => void
    onDeleted?: (agentId: string) => void
    onUpdated?: () => void
  } = $props()

  let actionBusy = $state(false)
  let editingName = $state(false)
  let editNameValue = $state('')
  let savingName = $state(false)
  let savingProvider = $state(false)

  const selectedProvider = $derived(
    agent ? (providers.find((provider) => provider.id === agent.providerId) ?? null) : null,
  )

  function startEditingName() {
    if (!agent) return
    editNameValue = agent.name
    editingName = true
  }

  function cancelEditingName() {
    editingName = false
  }

  async function handleSaveName() {
    if (!agent) return
    const trimmed = editNameValue.trim()
    if (!trimmed || trimmed === agent.name) {
      editingName = false
      return
    }

    savingName = true
    try {
      await updateAgent(agent.id, { name: trimmed })
      editingName = false
      toastStore.success('Agent name updated.')
      onUpdated?.()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update agent name.')
    } finally {
      savingName = false
    }
  }

  function handleNameKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !event.isComposing) {
      event.preventDefault()
      void handleSaveName()
    } else if (event.key === 'Escape') {
      cancelEditingName()
    }
  }

  async function handleProviderChange(nextProviderId: string) {
    if (!agent || nextProviderId === agent.providerId) return

    savingProvider = true
    try {
      await updateAgent(agent.id, { provider_id: nextProviderId })
      toastStore.success('Agent provider updated.')
      onUpdated?.()
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to update agent provider.')
    } finally {
      savingProvider = false
    }
  }

  async function handlePause() {
    if (!agent) return
    actionBusy = true
    try {
      await pauseAgent(agent.id)
      toastStore.success(`Pause requested for "${agent.name}".`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to pause agent.')
    } finally {
      actionBusy = false
    }
  }

  async function handleResume() {
    if (!agent) return
    actionBusy = true
    try {
      await resumeAgent(agent.id)
      toastStore.success(`Resumed "${agent.name}".`)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to resume agent.')
    } finally {
      actionBusy = false
    }
  }

  async function handleDelete() {
    if (!agent) return
    const confirmed = window.confirm(
      `Delete "${agent.name}"? This agent definition will be permanently removed.`,
    )
    if (!confirmed) return

    actionBusy = true
    try {
      await deleteAgent(agent.id)
      toastStore.success(`Deleted agent "${agent.name}".`)
      onDeleted?.(agent.id)
      onOpenChange?.(false)
    } catch (err) {
      toastStore.error(err instanceof ApiError ? err.detail : 'Failed to delete agent.')
    } finally {
      actionBusy = false
    }
  }
</script>

<Sheet
  bind:open
  onOpenChange={(value) => {
    open = value
    editingName = false
    onOpenChange?.(value)
  }}
>
  <SheetContent class="flex w-full flex-col gap-0 overflow-y-auto p-0 sm:max-w-md">
    {#if !agent}
      <SheetHeader class="p-6">
        <SheetTitle>Agent</SheetTitle>
        <SheetDescription>No agent selected.</SheetDescription>
      </SheetHeader>
    {:else}
      <!-- Header -->
      <SheetHeader class="border-border border-b px-6 py-5">
        <div class="flex items-start gap-3">
          <span class={cn('mt-2 size-2.5 shrink-0 rounded-full', agentStatusDot[agent.status])}
          ></span>
          <div class="min-w-0 flex-1">
            {#if editingName}
              <div class="flex items-center gap-1.5">
                <Input
                  bind:value={editNameValue}
                  class="h-8 text-base font-semibold"
                  onkeydown={handleNameKeydown}
                  disabled={savingName}
                />
                <Button
                  variant="ghost"
                  size="sm"
                  class="size-7 shrink-0 p-0"
                  disabled={savingName}
                  aria-label="Save agent name"
                  onclick={() => void handleSaveName()}
                >
                  <Check class="size-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  class="size-7 shrink-0 p-0"
                  aria-label="Cancel agent rename"
                  onclick={cancelEditingName}
                >
                  <X class="size-3.5" />
                </Button>
              </div>
            {:else}
              <div class="group flex items-center gap-1.5">
                <SheetTitle class="truncate text-base">{agent.name}</SheetTitle>
                <button
                  type="button"
                  class="text-muted-foreground hover:text-foreground opacity-0 transition-opacity group-hover:opacity-100"
                  aria-label="Rename agent"
                  onclick={startEditingName}
                  title="Rename agent"
                >
                  <Pencil class="size-3" />
                </button>
              </div>
            {/if}
            <SheetDescription class="mt-1">
              <Select.Root
                type="single"
                value={agent.providerId}
                disabled={savingProvider || providers.length === 0}
                onValueChange={(v) => {
                  if (v) void handleProviderChange(v)
                }}
              >
                <Select.Trigger
                  aria-label="Agent provider"
                  class="text-muted-foreground hover:text-foreground h-auto w-auto gap-2 border-none bg-transparent p-0 text-[13px] shadow-none"
                >
                  {#if selectedProvider}
                    {@const iconPath = adapterIconPath(selectedProvider.adapter_type)}
                    <div class="flex min-w-0 items-center gap-2">
                      {#if iconPath}
                        <img src={iconPath} alt="" class="size-4 shrink-0" />
                      {:else}
                        <Wrench class="text-muted-foreground size-4 shrink-0" />
                      {/if}
                      <span class="truncate"
                        >{selectedProvider.name} · {selectedProvider.model_name}</span
                      >
                    </div>
                  {:else}
                    {agent.providerName} · {agent.modelName}
                  {/if}
                </Select.Trigger>
                <Select.Content align="start" class="min-w-56">
                  {#each providers as provider (provider.id)}
                    <Select.Item value={provider.id} label={provider.name}>
                      {@const iconPath = adapterIconPath(provider.adapter_type)}
                      <div class="flex min-w-0 items-center gap-2.5 py-0.5">
                        {#if iconPath}
                          <img src={iconPath} alt="" class="size-4 shrink-0" />
                        {:else}
                          <Wrench class="text-muted-foreground size-4 shrink-0" />
                        {/if}
                        <div class="min-w-0">
                          <div class="truncate text-sm">{provider.name}</div>
                          <div class="text-muted-foreground text-[11px]">
                            {provider.model_name} · {provider.adapter_type}
                          </div>
                        </div>
                      </div>
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </SheetDescription>
          </div>
          <span
            class={cn(
              'mt-0.5 inline-flex shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-medium',
              agentStatusVariant[agent.status],
            )}
          >
            {agentStatusLabel[agent.status]}
            {#if agent.runtimeControlState !== 'active'}
              <span class="ml-1 opacity-70">
                · {agent.runtimeControlState === 'pause_requested' ? 'Pausing' : 'Paused'}
              </span>
            {/if}
          </span>
        </div>
      </SheetHeader>
      <AgentDrawerContent
        {agent}
        {actionBusy}
        canPause={canPauseAgent(agent)}
        canResume={canResumeAgent(agent)}
        onPause={handlePause}
        onResume={handleResume}
        onDelete={handleDelete}
      />
    {/if}
  </SheetContent>
</Sheet>
