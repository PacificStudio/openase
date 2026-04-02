<script lang="ts">
  import { untrack } from 'svelte'
  import { ApiError } from '$lib/api/client'
  import type { AgentProvider } from '$lib/api/contracts'
  import { listProviders } from '$lib/api/openase'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Plus, RefreshCcw, Send } from '@lucide/svelte'
  import { createProjectConversationController } from './project-conversation-controller.svelte'
  import {
    describeProjectAIFocus,
    projectAIFocusKey,
    type ProjectAIFocus,
  } from './project-ai-focus'
  import ProjectConversationFocusCard from './project-conversation-focus-card.svelte'
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
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center gap-2 border-b px-3 py-1.5 pr-12">
    <h2 class="text-xs font-medium">{title}</h2>
    <EphemeralChatProviderSelect
      providers={chatProviders}
      {providerId}
      disabled={providerSelectionDisabled}
      onProviderChange={(nextProviderId) => void controller.selectProvider(nextProviderId)}
    />
    <div class="ml-auto flex items-center">
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground h-6 gap-1 px-1.5 text-[11px]"
        aria-label="New Tab"
        onclick={() => controller.createTab()}
        disabled={!providerId}
      >
        <Plus class="size-3" />
      </Button>
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground size-6 p-0"
        aria-label="Reset conversation"
        onclick={() => void controller.resetConversation()}
        disabled={entries.length === 0 && !pending}
      >
        <RefreshCcw class="size-3" />
      </Button>
    </div>
  </div>

  <ProjectConversationTabStrip
    {tabs}
    {activeTabId}
    {conversations}
    onSelectTab={(tabId) => controller.selectTab(tabId)}
    onCloseTab={(tabId) => controller.closeTab(tabId)}
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

  <div class="border-border border-t px-3 py-2">
    {#if loadingProviders}
      <div class="text-muted-foreground mb-1.5 text-[11px]">Loading providers...</div>
    {:else if providerError}
      <div class="text-destructive mb-1.5 text-[11px]">{providerError}</div>
    {:else if chatProviders.length === 0}
      <div class="text-muted-foreground mb-1.5 text-[11px]">No chat provider available.</div>
    {:else if statusMessage}
      <div class="text-muted-foreground mb-1.5 text-[11px]">{statusMessage}</div>
    {:else if activeTab?.restored}
      <div class="text-muted-foreground mb-1.5 text-[11px]">Restored from last session.</div>
    {/if}

    {#if focusCard}
      <ProjectConversationFocusCard
        label={focusCard.label}
        title={focusCard.title}
        detail={focusCard.detail}
        onDismiss={() => {
          suppressedFocusKey = effectiveFocusKey
        }}
      />
    {/if}

    <div
      class="border-input focus-within:ring-ring flex items-center gap-1.5 rounded-lg border px-2.5 py-1 focus-within:ring-1"
    >
      <Textarea
        bind:value={prompt}
        rows={1}
        class="min-h-0 flex-1 resize-none border-0 px-0 py-1 text-sm shadow-none focus-visible:ring-0"
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
        class="text-muted-foreground size-6 shrink-0 p-0"
        aria-label="Send message"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || inputDisabled}
      >
        <Send class="size-3.5" />
      </Button>
    </div>
  </div>
</div>
