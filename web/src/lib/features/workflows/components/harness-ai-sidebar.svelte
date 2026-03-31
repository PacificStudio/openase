<script lang="ts">
  import { untrack } from 'svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    createEphemeralChatSessionController,
    EPHEMERAL_CHAT_MAX_BUDGET_USD,
    EPHEMERAL_CHAT_MAX_TURNS,
    EphemeralChatTranscript,
    EphemeralChatProviderSelect,
  } from '$lib/features/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Bot, RefreshCcw, Send } from '@lucide/svelte'
  import {
    buildDiffPreview,
    findLatestHarnessSuggestion,
    fingerprintSuggestion,
  } from '../assistant'
  import HarnessChatEmptyState from './harness-chat-empty-state.svelte'
  import HarnessSuggestionCard from './harness-suggestion-card.svelte'

  let {
    projectId,
    providers = [],
    workflowId,
    workflowName,
    draftContent,
    onApplySuggestion,
  }: {
    projectId?: string
    providers?: AgentProvider[]
    workflowId?: string
    workflowName?: string
    draftContent: string
    onApplySuggestion?: (content: string) => void
  } = $props()

  let prompt = $state('')
  let appliedFingerprint = $state('')
  let previousContextKey = ''
  const chatController = createEphemeralChatSessionController({
    getSource: () => 'harness_editor',
    onError: (message) => toastStore.error(message),
  })

  const chatProviders = $derived(chatController.providers)
  const providerId = $derived(chatController.providerId)
  const selectedProvider = $derived(chatController.selectedProvider)
  const pending = $derived(chatController.pending)
  const sessionId = $derived(chatController.sessionId)
  const entries = $derived(chatController.entries)
  const suggestion = $derived(findLatestHarnessSuggestion(entries, draftContent))
  const preview = $derived(suggestion ? buildDiffPreview(draftContent, suggestion.content) : null)
  const currentFingerprint = $derived(suggestion ? fingerprintSuggestion(suggestion.content) : '')
  const suggestionAlreadyApplied = $derived(
    Boolean(preview && !preview.hasChanges) || appliedFingerprint === currentFingerprint,
  )

  $effect(() => {
    const contextKey = projectId && workflowId ? `${projectId}:${workflowId}` : ''
    if (contextKey === previousContextKey) {
      return
    }
    previousContextKey = contextKey
    prompt = ''
    appliedFingerprint = ''
    void chatController.resetConversation()
  })

  $effect(() => {
    const nextProviders = providers
    const nextDefaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''
    untrack(() => {
      chatController.syncProviders(nextProviders, nextDefaultProviderId)
    })
  })

  $effect(() => {
    return () => {
      void chatController.dispose()
    }
  })

  async function handleSend() {
    if (!projectId || !workflowId) {
      return
    }

    const message = prompt.trim()
    if (!message) {
      return
    }

    prompt = ''

    await chatController.sendTurn({
      message,
      context: {
        projectId,
        workflowId,
      },
    })
  }

  function handleApply() {
    if (!suggestion) return
    onApplySuggestion?.(suggestion.content)
    appliedFingerprint = fingerprintSuggestion(suggestion.content)
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey) {
      return
    }
    event.preventDefault()
    void handleSend()
  }

  async function resetConversation() {
    prompt = ''
    appliedFingerprint = ''
    await chatController.resetConversation()
  }

  async function handleProviderChange(nextProviderId: string) {
    prompt = ''
    appliedFingerprint = ''
    await chatController.selectProvider(nextProviderId)
  }

  async function handleConfirmActionProposal(entryId: string) {
    await chatController.confirmActionProposal(entryId)
  }

  function handleCancelActionProposal(entryId: string) {
    chatController.cancelActionProposal(entryId)
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <Bot class="text-primary size-4" />
        <h2 class="text-sm font-semibold">Harness AI</h2>
        {#if selectedProvider}
          <Badge variant="outline" class="text-[10px]">{selectedProvider.name}</Badge>
        {/if}
        {#if sessionId}
          <Badge variant="outline" class="text-[10px]">Context kept</Badge>
        {/if}
      </div>
      <p class="text-muted-foreground mt-1 truncate text-xs">
        {workflowName ? `Editing ${workflowName}` : 'Select a workflow to start chatting.'}
      </p>
    </div>

    <Button
      variant="ghost"
      size="sm"
      onclick={() => void resetConversation()}
      disabled={entries.length === 0 && !pending}
    >
      <RefreshCcw class="size-4" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-4 py-4">
    <div class="space-y-3">
      {#if entries.length === 0}
        <HarnessChatEmptyState />
      {/if}
      <EphemeralChatTranscript
        {entries}
        {pending}
        onConfirmActionProposal={handleConfirmActionProposal}
        onCancelActionProposal={handleCancelActionProposal}
      />

      {#if suggestion && preview}
        <HarnessSuggestionCard
          {suggestion}
          {preview}
          {suggestionAlreadyApplied}
          onApply={handleApply}
        />
      {/if}
    </div>
  </ScrollArea>

  <div class="border-border border-t px-4 py-3">
    <EphemeralChatProviderSelect
      providers={chatProviders}
      {providerId}
      onProviderChange={(nextProviderId) => void handleProviderChange(nextProviderId)}
    />

    <Textarea
      bind:value={prompt}
      rows={4}
      class="text-sm"
      placeholder="Ask the assistant to refine this harness. Shift+Enter for newline."
      disabled={!projectId || !workflowId || !providerId || pending}
      onkeydown={handlePromptKeydown}
    />

    <div class="mt-3 flex items-center justify-between gap-3">
      <p class="text-muted-foreground text-[11px] leading-4">
        {sessionId
          ? `Session cap: ${EPHEMERAL_CHAT_MAX_TURNS} turns / $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}. Follow-up prompts reuse the current chat context until you close the panel, switch providers, or hit the limit.`
          : `Session cap: ${EPHEMERAL_CHAT_MAX_TURNS} turns / $${EPHEMERAL_CHAT_MAX_BUDGET_USD.toFixed(2)}. The first reply starts a new ephemeral chat session.`}
      </p>
      <Button
        size="sm"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || !projectId || !workflowId || !providerId || pending}
      >
        <Send class="size-4" />
        Send
      </Button>
    </div>
  </div>
</div>
