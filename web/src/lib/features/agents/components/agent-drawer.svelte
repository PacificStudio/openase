<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { deleteAgent, pauseAgent, retireAgent, resumeAgent, updateAgent } from '$lib/api/openase'
  import type { AgentProvider } from '$lib/api/contracts'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from '$ui/sheet'
  import AgentDrawerContent from './agent-drawer-content.svelte'
  import AgentDrawerHeader from './agent-drawer-header.svelte'
  import { canPauseAgent, canRetireAgent, canResumeAgent } from './agent-drawer-state'
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

  function showApiError(err: unknown, fallback: string) {
    toastStore.error(err instanceof ApiError ? err.detail : fallback)
  }

  async function runAgentAction(
    action: () => Promise<unknown>,
    successMessage: string,
    fallbackMessage: string,
    afterSuccess?: () => void,
  ) {
    actionBusy = true
    try {
      await action()
      toastStore.success(successMessage)
      afterSuccess?.()
    } catch (err) {
      showApiError(err, fallbackMessage)
    } finally {
      actionBusy = false
    }
  }

  async function handlePause() {
    if (!agent) return
    await runAgentAction(
      () => pauseAgent(agent.id),
      `Pause requested for "${agent.name}".`,
      'Failed to pause agent.',
    )
  }

  async function handleResume() {
    if (!agent) return
    await runAgentAction(
      () => resumeAgent(agent.id),
      `Resumed "${agent.name}".`,
      'Failed to resume agent.',
    )
  }

  async function handleDelete() {
    if (!agent) return
    const confirmed = window.confirm(
      `Delete "${agent.name}"? This agent definition will be permanently removed.`,
    )
    if (!confirmed) return

    await runAgentAction(
      () => deleteAgent(agent.id),
      `Deleted agent "${agent.name}".`,
      'Failed to delete agent.',
      () => {
        onDeleted?.(agent.id)
        onOpenChange?.(false)
      },
    )
  }

  async function handleRetire() {
    if (!agent) return
    const confirmed = window.confirm(
      `Retire "${agent.name}"? It will stop appearing in workflow assignment and creation paths, but historical runs will remain.`,
    )
    if (!confirmed) return

    await runAgentAction(
      () => retireAgent(agent.id),
      `Retired agent "${agent.name}".`,
      'Failed to retire agent.',
      () => onUpdated?.(),
    )
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
        bind:editNameValue
        {savingName}
        {savingProvider}
        onStartEditingName={startEditingName}
        onCancelEditingName={cancelEditingName}
        onSaveName={handleSaveName}
        onNameKeydown={handleNameKeydown}
        onProviderChange={handleProviderChange}
      />
      <AgentDrawerContent
        {agent}
        {actionBusy}
        canPause={canPauseAgent(agent)}
        canResume={canResumeAgent(agent)}
        canRetire={canRetireAgent(agent)}
        onPause={handlePause}
        onResume={handleResume}
        onRetire={handleRetire}
        onDelete={handleDelete}
      />
    {/if}
  </SheetContent>
</Sheet>
