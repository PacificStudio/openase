<script lang="ts">
  import { untrack } from 'svelte'
  import type { AgentProvider, SkillFile } from '$lib/api/contracts'
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
  import {
    buildDiffPreview,
    findLatestSkillSuggestion,
    fingerprintSuggestion,
  } from '$lib/features/skills/assistant'
  import SkillChatEmptyState from './skill-chat-empty-state.svelte'
  import SkillSuggestionCard from './skill-suggestion-card.svelte'

  let {
    projectId,
    providers = [],
    skillId,
    files = [],
    selectedFilePath,
    selectedFileIsText = true,
    onApplySuggestion,
  }: {
    projectId?: string
    providers?: AgentProvider[]
    skillId?: string
    files?: SkillFile[]
    selectedFilePath?: string | null
    selectedFileIsText?: boolean
    onApplySuggestion?: (
      files: Array<{ path: string; content: string }>,
      focusPath?: string,
    ) => void
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
  const selectedFileContent = $derived(
    files.find((file) => file.path === normalizedSelectedPath)?.content ?? '',
  )
  const suggestion = $derived(
    normalizedSelectedPath && selectedFileIsText
      ? findLatestSkillSuggestion(entries, {
          selectedFilePath: normalizedSelectedPath,
          files,
        })
      : null,
  )
  let selectedSuggestionPath = $state('')
  const previewTarget = $derived(
    suggestion?.files.find((file) => file.path === selectedSuggestionPath) ??
      suggestion?.files[0] ??
      null,
  )
  const preview = $derived(
    previewTarget
      ? buildDiffPreview(
          files.find((file) => file.path === previewTarget.path)?.content ?? '',
          previewTarget.content,
        )
      : null,
  )
  const currentFingerprint = $derived(
    suggestion
      ? fingerprintSuggestion(
          suggestion.files.map((file) => `${file.path}\n${file.content}`).join('\n\n'),
        )
      : '',
  )
  const previewList = $derived(
    suggestion?.files.map((file) => ({
      path: file.path,
      preview: buildDiffPreview(
        files.find((current) => current.path === file.path)?.content ?? '',
        file.content,
      ),
    })) ?? [],
  )
  const suggestionAlreadyApplied = $derived(
    (previewList.length > 0 && previewList.every((item) => !item.preview.hasChanges)) ||
      appliedFingerprint === currentFingerprint,
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
    selectedSuggestionPath = ''
    void chatController.resetConversation()
  })

  $effect(() => {
    if (!suggestion || suggestion.files.length === 0) {
      selectedSuggestionPath = ''
      return
    }
    const stillExists = suggestion.files.some((file) => file.path === selectedSuggestionPath)
    if (stillExists) {
      return
    }
    selectedSuggestionPath =
      suggestion.files.find((file) => file.path === normalizedSelectedPath)?.path ??
      suggestion.files[0]?.path ??
      ''
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
    onApplySuggestion?.(suggestion.files, selectedSuggestionPath || suggestion.files[0]?.path)
    appliedFingerprint = currentFingerprint
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
          selectedPath={selectedSuggestionPath}
          {preview}
          {suggestionAlreadyApplied}
          onSelectPath={(path) => (selectedSuggestionPath = path)}
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
