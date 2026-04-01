<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider } from '$lib/api/contracts'
  import { listProviders } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Plus, RefreshCcw, Send, X } from '@lucide/svelte'
  import { createProjectConversationController } from './project-conversation-controller.svelte'
  import type { ProjectConversationPhase } from './project-conversation-controller-helpers'
  import type {
    ProjectConversationTextEntry,
    ProjectConversationTranscriptEntry,
  } from './project-conversation-transcript-types'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'
  import ProjectConversationTranscript from './project-conversation-transcript.svelte'

  let {
    context,
    organizationId = '',
    providers = [],
    defaultProviderId = null,
    title = 'Project AI',
    placeholder = 'Ask anything about this project…',
    initialPrompt = '',
  }: {
    context: { projectId: string }
    organizationId?: string
    providers?: AgentProvider[]
    defaultProviderId?: string | null
    title?: string
    placeholder?: string
    initialPrompt?: string
  } = $props()

  let prompt = $state('')
  let loadingProviders = $state(false)
  let providerError = $state('')
  let loadedProviders = $state<AgentProvider[]>([])
  let previousRestoreKey = ''
  let openConversationId = $state('')

  const controller = createProjectConversationController({
    getProjectId: () => context.projectId,
    onError: (message) => toastStore.error(message),
  })

  const activeProviders = $derived(providers.length > 0 ? providers : loadedProviders)
  const chatProviders = $derived(controller.providers)
  const providerId = $derived(controller.providerId)
  const conversations = $derived(controller.conversations)
  const tabs = $derived(controller.tabs)
  const activeTabId = $derived(controller.activeTabId)
  const activeTab = $derived(tabs.find((tab) => tab.id === activeTabId) ?? tabs[0] ?? null)
  const entries = $derived(controller.entries)
  const pending = $derived(controller.pending)
  const phase = $derived(controller.phase)
  const inputDisabled = $derived(controller.inputDisabled)
  const providerSelectionDisabled = $derived(controller.providerSelectionDisabled)
  const statusMessage = $derived(getStatusMessage(phase, controller.hasPendingInterrupt))
  const openConversationIds = $derived(
    new Set(tabs.map((tab) => tab.conversationId).filter((conversationId) => conversationId)),
  )
  const historicalConversations = $derived(
    conversations.filter((conversation) => !openConversationIds.has(conversation.id)),
  )

  $effect(() => {
    if (providers.length > 0 || !organizationId) {
      loadingProviders = false
      providerError = ''
      loadedProviders = []
      return
    }

    let cancelled = false

    const load = async () => {
      loadingProviders = true
      providerError = ''

      try {
        const payload = await listProviders(organizationId)
        if (!cancelled) {
          loadedProviders = payload.providers
        }
      } catch (caughtError) {
        if (!cancelled) {
          providerError =
            caughtError instanceof ApiError ? caughtError.detail : 'Failed to load chat providers.'
        }
      } finally {
        if (!cancelled) {
          loadingProviders = false
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  })

  $effect(() => {
    const nextProviders = activeProviders
    const nextDefaultProviderId = defaultProviderId
    untrack(() => {
      controller.syncProviders(nextProviders, nextDefaultProviderId)
    })
  })

  $effect(() => {
    const restoreKey = `${context.projectId}:${providerId}`
    if (!context.projectId || !providerId || restoreKey === previousRestoreKey) {
      return
    }
    previousRestoreKey = restoreKey
    prompt = initialPrompt
    void controller.restore()
  })

  $effect(() => {
    return () => {
      controller.dispose()
    }
  })

  async function handleSend() {
    const message = prompt.trim()
    if (!message || !context.projectId || !providerId || pending) {
      return
    }

    prompt = ''
    await controller.sendTurn(message)
  }

  async function handleOpenConversation(nextConversationId: string) {
    if (!nextConversationId) {
      return
    }
    openConversationId = ''
    await controller.openConversation(nextConversationId)
  }

  function formatConversationLabel(tab: {
    conversationId: string
    entries: ProjectConversationTranscriptEntry[]
  }) {
    const conversation = conversations.find((item) => item.id === tab.conversationId)
    const summary = (conversation?.rollingSummary ?? '').trim()
    if (summary) {
      return summary.length > 32 ? `${summary.slice(0, 32)}…` : summary
    }

    const recentUserMessage = [...tab.entries]
      .reverse()
      .find(
        (entry): entry is ProjectConversationTextEntry =>
          entry.kind === 'text' && entry.role === 'user' && entry.content.trim().length > 0,
      )
    if (recentUserMessage?.content) {
      const content = recentUserMessage.content.trim()
      return content.length > 32 ? `${content.slice(0, 32)}…` : content
    }

    if (!tab.conversationId) {
      return 'New tab'
    }

    const timestamp = new Date(conversation?.lastActivityAt ?? '')
    if (Number.isNaN(timestamp.getTime())) {
      return 'Conversation'
    }

    return `Conversation · ${timestamp.toLocaleString([], {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })}`
  }

  function formatTabStatus(tab: {
    pending: boolean
    hasPendingInterrupt: boolean
    restored: boolean
  }) {
    if (tab.hasPendingInterrupt) {
      return 'Input required'
    }
    if (tab.pending) {
      return 'Running'
    }
    if (tab.restored) {
      return 'Restored'
    }
    return ''
  }

  function getStatusMessage(
    currentPhase: ProjectConversationPhase,
    hasPendingInterrupt: boolean,
  ): string | null {
    if (hasPendingInterrupt) {
      return 'Additional input is required before the conversation can continue.'
    }

    switch (currentPhase) {
      case 'restoring':
        return 'Restoring this project conversation…'
      case 'creating_conversation':
        return 'Creating a fresh project conversation…'
      case 'connecting_stream':
        return 'Connecting the live conversation stream…'
      case 'submitting_turn':
        return 'Sending your message…'
      case 'awaiting_reply':
        return 'Waiting for the assistant reply…'
      case 'resetting':
        return 'Resetting the current conversation…'
      default:
        return null
    }
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-4 py-2">
    <div class="flex items-center gap-2">
      <h2 class="text-sm font-semibold">{title}</h2>
      <EphemeralChatProviderSelect
        providers={chatProviders}
        {providerId}
        disabled={providerSelectionDisabled}
        onProviderChange={(nextProviderId) => void controller.selectProvider(nextProviderId)}
      />
    </div>

    <div class="flex items-center gap-1">
      <Button
        variant="ghost"
        size="sm"
        class="h-7 gap-1 px-2 text-xs"
        onclick={() => controller.createTab()}
        disabled={!providerId}
      >
        <Plus class="size-3.5" />
        New Tab
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="size-7 p-0"
        aria-label="Reset conversation"
        onclick={() => void controller.resetConversation()}
        disabled={entries.length === 0 && !pending}
      >
        <RefreshCcw class="size-3.5" />
      </Button>
    </div>
  </div>

  <div class="border-border border-b px-4 py-2">
    <div class="flex flex-wrap gap-2">
      {#each tabs as tab (tab.id)}
        <div
          class:bg-accent={tab.id === activeTabId}
          class="border-input flex max-w-full items-center gap-1 rounded-md border px-2 py-1"
        >
          <button
            type="button"
            class="min-w-0 text-left"
            onclick={() => controller.selectTab(tab.id)}
          >
            <div class="truncate text-sm font-medium">{formatConversationLabel(tab)}</div>
            {#if formatTabStatus(tab)}
              <div class="text-muted-foreground text-[10px] uppercase">{formatTabStatus(tab)}</div>
            {/if}
          </button>
          {#if tabs.length > 1 || tab.conversationId || tab.entries.length > 0}
            <button
              type="button"
              class="text-muted-foreground hover:text-foreground rounded p-0.5"
              aria-label={`Close ${formatConversationLabel(tab)}`}
              onclick={(event) => {
                event.stopPropagation()
                controller.closeTab(tab.id)
              }}
            >
              <X class="size-3" />
            </button>
          {/if}
        </div>
      {/each}
    </div>

    {#if historicalConversations.length > 0}
      <div class="mt-2">
        <label
          class="text-muted-foreground mb-1 block text-[10px] font-semibold tracking-[0.16em] uppercase"
          for="project-conversation-open"
        >
          Open Existing Conversation
        </label>
        <select
          id="project-conversation-open"
          class="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
          bind:value={openConversationId}
          disabled={!providerId}
          onchange={(event) =>
            void handleOpenConversation((event.currentTarget as HTMLSelectElement).value)}
        >
          <option value="">Select a conversation</option>
          {#each historicalConversations as conversation (conversation.id)}
            <option value={conversation.id}>
              {conversation.rollingSummary?.trim() ||
                formatConversationLabel({
                  conversationId: conversation.id,
                  entries: [],
                })}
            </option>
          {/each}
        </select>
      </div>
    {/if}
  </div>

  <ScrollArea class="min-h-0 flex-1 px-4 py-4">
    <ProjectConversationTranscript
      {entries}
      {pending}
      onConfirmActionProposal={(entryId) => controller.confirmActionProposal(entryId)}
      onRespondInterrupt={(payload) => controller.respondInterrupt(payload)}
    />
  </ScrollArea>

  <div class="border-border border-t px-4 py-3">
    {#if loadingProviders}
      <div class="text-muted-foreground mb-2 text-xs">Loading providers…</div>
    {:else if providerError}
      <div class="text-destructive mb-2 text-xs">{providerError}</div>
    {:else if chatProviders.length === 0}
      <div class="text-muted-foreground mb-2 text-xs">No chat provider available.</div>
    {:else if statusMessage}
      <div class="text-muted-foreground mb-2 text-xs">{statusMessage}</div>
    {:else if activeTab?.restored}
      <div class="text-muted-foreground mb-2 text-xs">
        This tab was restored from your last session.
      </div>
    {/if}

    <div
      class="border-input focus-within:ring-ring flex items-center gap-2 rounded-lg border px-3 py-1.5 focus-within:ring-1"
    >
      <Textarea
        bind:value={prompt}
        rows={1}
        class="min-h-0 flex-1 resize-none border-0 px-0 py-1.5 text-sm shadow-none focus-visible:ring-0"
        {placeholder}
        disabled={inputDisabled}
        onkeydown={(event) => {
          if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
            event.preventDefault()
            void handleSend()
          }
        }}
      />
      <Button
        variant="ghost"
        size="sm"
        class="size-7 shrink-0 p-0"
        aria-label="Send message"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || inputDisabled}
      >
        <Send class="size-4" />
      </Button>
    </div>
  </div>
</div>
