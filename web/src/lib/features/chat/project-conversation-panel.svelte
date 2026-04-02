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
  import {
    describeProjectAIFocus,
    projectAIFocusKey,
    type ProjectAIFocus,
  } from './project-ai-focus'
  import { getProjectConversationStatusMessage } from './project-conversation-panel-labels'
  import ProjectConversationTabStrip from './project-conversation-tab-strip.svelte'
  import ProjectConversationWorkspaceSummary from './project-conversation-workspace-summary.svelte'
  import EphemeralChatProviderSelect from './ephemeral-chat-provider-select.svelte'
  import ProjectConversationTranscript from './project-conversation-transcript.svelte'

  let {
    context,
    organizationId = '',
    providers = [],
    defaultProviderId = null,
    focus = null,
    title = 'Project AI',
    placeholder = 'Ask anything about this project…',
    initialPrompt = '',
  }: {
    context: { projectId: string }
    organizationId?: string
    providers?: AgentProvider[]
    defaultProviderId?: string | null
    focus?: ProjectAIFocus | null
    title?: string
    placeholder?: string
    initialPrompt?: string
  } = $props()

  let prompt = $state('')
  let suppressedFocusKey = $state('')
  let loadingProviders = $state(false)
  let providerError = $state('')
  let loadedProviders = $state<AgentProvider[]>([])
  let previousRestoreKey = ''

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
  const workspaceDiff = $derived(controller.workspaceDiff)
  const workspaceDiffLoading = $derived(controller.workspaceDiffLoading)
  const workspaceDiffError = $derived(controller.workspaceDiffError)
  const pending = $derived(controller.pending)
  const phase = $derived(controller.phase)
  const inputDisabled = $derived(controller.inputDisabled)
  const providerSelectionDisabled = $derived(controller.providerSelectionDisabled)
  const statusMessage = $derived(
    getProjectConversationStatusMessage(phase, controller.hasPendingInterrupt),
  )
  const effectiveFocus = $derived(focus && focus.projectId === context.projectId ? focus : null)
  const effectiveFocusKey = $derived(projectAIFocusKey(effectiveFocus))
  const focusForSend = $derived(
    effectiveFocus && suppressedFocusKey !== effectiveFocusKey ? effectiveFocus : null,
  )
  const focusCard = $derived(focusForSend ? describeProjectAIFocus(focusForSend) : null)
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

  $effect(() => {
    if (!effectiveFocusKey) {
      suppressedFocusKey = ''
    }
  })

  async function handleSend() {
    const message = prompt.trim()
    if (!message || !context.projectId || !providerId || pending) {
      return
    }

    const nextFocus = suppressedFocusKey === effectiveFocusKey ? null : effectiveFocus
    prompt = ''
    await controller.sendTurn(message, nextFocus)
    suppressedFocusKey = ''
  }

  async function handleOpenConversation(nextConversationId: string) {
    if (!nextConversationId) {
      return
    }
    await controller.openConversation(nextConversationId)
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

  <ProjectConversationTabStrip
    {tabs}
    {activeTabId}
    {conversations}
    {historicalConversations}
    {providerId}
    onSelectTab={(tabId) => controller.selectTab(tabId)}
    onCloseTab={(tabId) => controller.closeTab(tabId)}
    onOpenConversation={(conversationId) => void handleOpenConversation(conversationId)}
  />

  <ProjectConversationWorkspaceSummary
    conversationId={activeTab?.conversationId ?? ''}
    {workspaceDiff}
    loading={workspaceDiffLoading}
    error={workspaceDiffError}
  />

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

    {#if focusCard}
      <div
        class="bg-muted/40 mb-2 flex items-start justify-between gap-3 rounded-lg border px-3 py-2"
      >
        <div class="min-w-0">
          <div class="text-muted-foreground text-[11px] font-medium tracking-[0.16em] uppercase">
            Current focus
          </div>
          <div class="truncate text-sm font-medium">{focusCard.label}: {focusCard.title}</div>
          {#if focusCard.detail}
            <div class="text-muted-foreground truncate text-xs">{focusCard.detail}</div>
          {/if}
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="text-muted-foreground hover:text-foreground size-7 shrink-0 p-0"
          aria-label="Remove focus for this send"
          onclick={() => {
            suppressedFocusKey = effectiveFocusKey
          }}
        >
          <X class="size-4" />
        </Button>
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
