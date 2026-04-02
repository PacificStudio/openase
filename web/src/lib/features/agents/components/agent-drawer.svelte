<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { deleteAgent, pauseAgent, resumeAgent, updateAgent } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import AgentDrawerHeader from './agent-drawer-header.svelte'
  import AgentDrawerContent from './agent-drawer-content.svelte'
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

  function canPause(a: AgentInstance) {
    return (
      a.runtimeControlState === 'active' &&
      a.activeRunCount > 0 &&
      (a.status === 'claimed' || a.status === 'running')
    )
  }

  function canResume(a: AgentInstance) {
    return a.runtimeControlState === 'paused'
  }
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

  function handleEditNameValueChange(value: string) {
    editNameValue = value
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
      <AgentDrawerHeader
        {agent}
        {providers}
        {editingName}
        {editNameValue}
        {savingName}
        {savingProvider}
        onStartEditingName={startEditingName}
        onCancelEditingName={cancelEditingName}
        onSaveName={handleSaveName}
        onProviderChange={handleProviderChange}
        onEditNameValueChange={handleEditNameValueChange}
        onNameKeydown={handleNameKeydown}
      />
      <AgentDrawerContent
        {agent}
        {actionBusy}
        canPause={canPause(agent)}
        canResume={canResume(agent)}
        onPause={handlePause}
        onResume={handleResume}
        onDelete={handleDelete}
      />
    {/if}
  </SheetContent>
</Sheet>
