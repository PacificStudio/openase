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
  import { LoaderCircle, Plus, Send } from '@lucide/svelte'
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
  let dismissed = $state(false)
  let previousContextKey = ''
  const chatController = createEphemeralChatSessionController({
    getSource: () => 'harness_editor',
    capability: 'harness_ai',
    onError: (message) => toastStore.error(message),
  })

  const chatProviders = $derived(chatController.providers)
  const providerId = $derived(chatController.providerId)
  const pending = $derived(chatController.pending)
  const entries = $derived(chatController.entries)
  const suggestion = $derived(findLatestHarnessSuggestion(entries, draftContent))
  const preview = $derived(suggestion ? buildDiffPreview(draftContent, suggestion.content) : null)
  const currentFingerprint = $derived(suggestion ? fingerprintSuggestion(suggestion.content) : '')
  const streamingDiff = $derived(
    pending &&
      !suggestion &&
      entries.some(
        (entry) => entry.kind === 'text' && entry.role === 'assistant' && entry.streaming,
      ),
  )
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
    dismissed = false
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
    dismissed = false

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
    const currentSuggestion = suggestion
    if (!currentSuggestion) return
    onApplySuggestion?.(currentSuggestion.content)
    appliedFingerprint = fingerprintSuggestion(currentSuggestion.content)
  }

  function handlePromptKeydown(event: KeyboardEvent) {
    if (event.key !== 'Enter' || event.shiftKey || event.isComposing) {
      return
    }
    event.preventDefault()
    void handleSend()
  }

  function handleDismiss() {
    dismissed = true
  }

  async function resetConversation() {
    prompt = ''
    appliedFingerprint = ''
    dismissed = false
    await chatController.resetConversation()
  }

  async function handleProviderChange(nextProviderId: string) {
    prompt = ''
    appliedFingerprint = ''
    await chatController.selectProvider(nextProviderId)
  }
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between gap-2 border-b px-3 py-1">
    <div class="flex items-center gap-1.5">
      <span class="text-muted-foreground text-[11px] font-medium">AI</span>
      <EphemeralChatProviderSelect
        providers={chatProviders}
        capability="harness_ai"
        {providerId}
        onProviderChange={(nextProviderId) => void handleProviderChange(nextProviderId)}
      />
    </div>

    <Button
      variant="ghost"
      size="sm"
      class="size-6 p-0"
      aria-label="Reset conversation"
      onclick={() => void resetConversation()}
      disabled={entries.length === 0 && !pending}
    >
      <Plus class="size-3" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-4 py-4">
    <div class="space-y-3">
      {#if entries.length === 0}
        <HarnessChatEmptyState />
      {/if}
      <EphemeralChatTranscript {entries} {pending} variant="minimal" />

      {#if streamingDiff}
        <div class="text-muted-foreground flex items-center gap-1.5 py-1 text-xs">
          <LoaderCircle class="size-3 shrink-0 animate-spin" />
          Suggesting diff...
        </div>
      {/if}

      {#if suggestion && preview && !dismissed}
        <HarnessSuggestionCard
          {suggestion}
          {preview}
          {suggestionAlreadyApplied}
          onApply={handleApply}
          onDismiss={handleDismiss}
        />
      {/if}
    </div>
  </ScrollArea>

  <div class="border-border border-t px-3 py-1.5">
    <div
      class="border-input focus-within:ring-ring flex items-center gap-1.5 rounded-md border px-2.5 py-1 focus-within:ring-1"
    >
      <Textarea
        bind:value={prompt}
        rows={1}
        class="min-h-0 flex-1 resize-none border-0 px-0 py-1 text-xs shadow-none focus-visible:ring-0"
        placeholder="Ask AI to refine this harness…"
        disabled={!projectId || !workflowId || !providerId}
        onkeydown={handlePromptKeydown}
      />
      <Button
        variant="ghost"
        size="sm"
        class="size-6 shrink-0 p-0"
        onclick={() => void handleSend()}
        disabled={!prompt.trim() || sendDisabled}
      >
        <Send class="size-3.5" />
      </Button>
    </div>
  </div>
</div>
