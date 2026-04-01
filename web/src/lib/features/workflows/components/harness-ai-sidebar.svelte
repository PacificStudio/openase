<script lang="ts">
  import { untrack } from 'svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    createEphemeralChatSessionController,
    EphemeralChatTranscript,
    EphemeralChatProviderSelect,
  } from '$lib/features/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { RefreshCcw, Send } from '@lucide/svelte'
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
    draftContent,
    onApplySuggestion,
  }: {
    projectId?: string
    providers?: AgentProvider[]
    workflowId?: string
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
  const pending = $derived(chatController.pending)
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

  let sending = $state(false)
  const sendDisabled = $derived(!projectId || !workflowId || !providerId || pending || sending)

  async function handleSend() {
    const message = prompt.trim()
    if (!message || sendDisabled) {
      return
    }

    sending = true
    prompt = ''

    try {
      await chatController.sendTurn({
        message,
        context: {
          projectId: projectId!,
          workflowId: workflowId!,
          harnessDraft: draftContent,
        },
      })
    } finally {
      sending = false
    }
  }

  function handleApply() {
    if (!suggestion) return
    onApplySuggestion?.(suggestion.content)
    appliedFingerprint = fingerprintSuggestion(suggestion.content)
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) {
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
  <div class="border-border flex items-center justify-between gap-2 border-b px-4 py-2">
    <div class="flex items-center gap-2">
      <h2 class="text-sm font-semibold">Harness AI</h2>
      <EphemeralChatProviderSelect
        providers={chatProviders}
        {providerId}
        onProviderChange={(nextProviderId) => void handleProviderChange(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-7 p-0"
      aria-label="Reset conversation"
      onclick={() => void resetConversation()}
      disabled={entries.length === 0 && !pending}
    >
      <RefreshCcw class="size-3.5" />
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
    <div
      class="border-input focus-within:ring-ring flex items-center gap-2 rounded-lg border px-3 py-1.5 focus-within:ring-1"
    >
      <Textarea
        bind:value={prompt}
        rows={1}
        class="min-h-0 flex-1 resize-none border-0 px-0 py-1.5 text-sm shadow-none focus-visible:ring-0"
        placeholder="Ask AI to refine this harness…"
        disabled={!projectId || !workflowId || !providerId}
        onkeydown={handlePromptKeydown}
      />
      <Button
        variant="ghost"
        size="sm"
        class="size-7 shrink-0 p-0"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || sendDisabled}
      >
        <Send class="size-4" />
      </Button>
    </div>
  </div>
</div>
