<script lang="ts">
  import { untrack } from 'svelte'
  import type { AgentProvider } from '$lib/api/contracts'
  import {
    createEphemeralChatSessionController,
    EphemeralChatProviderSelect,
    EphemeralChatTranscript,
  } from '$lib/features/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { RefreshCcw, Send } from '@lucide/svelte'
  import { buildDiffPreview, findLatestSkillSuggestion, fingerprintSuggestion } from '../assistant'
  import SkillChatEmptyState from './skill-chat-empty-state.svelte'
  import SkillSuggestionCard from './skill-suggestion-card.svelte'

  let {
    projectId,
    providers = [],
    skillId,
    selectedFilePath,
    selectedFileContent,
    selectedFileIsText = true,
    onApplySuggestion,
  }: {
    projectId?: string
    providers?: AgentProvider[]
    skillId?: string
    selectedFilePath?: string | null
    selectedFileContent: string
    selectedFileIsText?: boolean
    onApplySuggestion?: (path: string, content: string) => void
  } = $props()

  let prompt = $state('')
  let appliedFingerprint = $state('')
  let previousContextKey = ''
  const chatController = createEphemeralChatSessionController({
    getSource: () => 'skill_editor',
    onError: (message) => toastStore.error(message),
  })

  const chatProviders = $derived(chatController.providers)
  const providerId = $derived(chatController.providerId)
  const pending = $derived(chatController.pending)
  const entries = $derived(chatController.entries)
  const normalizedSelectedPath = $derived(selectedFilePath?.trim() ?? '')
  const suggestion = $derived(
    normalizedSelectedPath && selectedFileIsText
      ? findLatestSkillSuggestion(entries, normalizedSelectedPath, selectedFileContent)
      : null,
  )
  const preview = $derived(
    suggestion ? buildDiffPreview(selectedFileContent, suggestion.content) : null,
  )
  const currentFingerprint = $derived(suggestion ? fingerprintSuggestion(suggestion.content) : '')
  const suggestionAlreadyApplied = $derived(
    Boolean(preview && !preview.hasChanges) || appliedFingerprint === currentFingerprint,
  )

  $effect(() => {
    const contextKey =
      projectId && skillId && normalizedSelectedPath
        ? `${projectId}:${skillId}:${normalizedSelectedPath}`
        : ''
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
  const sendDisabled = $derived(
    !projectId ||
      !skillId ||
      !normalizedSelectedPath ||
      !providerId ||
      pending ||
      sending ||
      !selectedFileIsText,
  )

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
          skillId: skillId!,
          skillFilePath: normalizedSelectedPath,
          skillFileDraft: selectedFileContent,
        },
      })
    } finally {
      sending = false
    }
  }

  function handleApply() {
    if (!suggestion) return
    onApplySuggestion?.(suggestion.path, suggestion.content)
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
  <div class="border-border flex items-center justify-between gap-2 border-b px-3 py-1">
    <div class="flex min-w-0 items-center gap-1.5">
      <span class="text-muted-foreground text-[11px] font-medium">AI</span>
      <EphemeralChatProviderSelect
        providers={chatProviders}
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
      <RefreshCcw class="size-3" />
    </Button>
  </div>

  <ScrollArea class="min-h-0 flex-1 px-3 py-2">
    <div class="space-y-2">
      {#if entries.length === 0}
        <SkillChatEmptyState />
      {/if}
      <EphemeralChatTranscript
        {entries}
        {pending}
        onConfirmActionProposal={handleConfirmActionProposal}
        onCancelActionProposal={handleCancelActionProposal}
      />

      {#if suggestion && preview}
        <SkillSuggestionCard
          {suggestion}
          {preview}
          {suggestionAlreadyApplied}
          onApply={handleApply}
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
        placeholder={selectedFileIsText
          ? `Ask AI to refine ${normalizedSelectedPath || 'this file'}…`
          : 'Select a UTF-8 text file to edit with AI…'}
        disabled={!projectId ||
          !skillId ||
          !providerId ||
          !normalizedSelectedPath ||
          !selectedFileIsText}
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
