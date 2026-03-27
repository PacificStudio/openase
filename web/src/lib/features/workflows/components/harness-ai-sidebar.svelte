<script lang="ts">
  import type { AgentProvider } from '$lib/api/contracts'
  import { ApiError } from '$lib/api/client'
  import { streamChatTurn, type ChatStreamEvent } from '$lib/api/chat'
  import { appStore } from '$lib/stores/app.svelte'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Bot, LoaderCircle, RefreshCcw, Send } from '@lucide/svelte'
  import {
    buildDiffPreview,
    findLatestHarnessSuggestion,
    fingerprintSuggestion,
    isAbortError,
    mapChatPayloadToTranscriptEntry,
    type AssistantTranscriptEntry,
  } from '../assistant'
  import HarnessChatEmptyState from './harness-chat-empty-state.svelte'
  import HarnessChatProviderSelect from './harness-chat-provider-select.svelte'
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
  let pending = $state(false)
  let sessionId = $state('')
  let providerId = $state('')
  let entries = $state<AssistantTranscriptEntry[]>([])
  let appliedFingerprint = $state('')
  let entryCounter = 0
  let previousContextKey = ''
  let abortController: AbortController | null = null

  const chatProviders = $derived(
    providers.filter(
      (provider) =>
        provider.available &&
        ['claude-code-cli', 'codex-app-server', 'gemini-cli'].includes(provider.adapter_type),
    ),
  )
  const selectedProvider = $derived(
    chatProviders.find((provider) => provider.id === providerId) ?? null,
  )
  const suggestion = $derived(findLatestHarnessSuggestion(entries))
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
    resetConversation()
  })

  $effect(() => {
    if (providerId && chatProviders.some((provider) => provider.id === providerId)) {
      return
    }

    const defaultProviderId = appStore.currentProject?.default_agent_provider_id ?? ''
    if (defaultProviderId && chatProviders.some((provider) => provider.id === defaultProviderId)) {
      providerId = defaultProviderId
      return
    }

    providerId = chatProviders[0]?.id ?? ''
  })

  $effect(() => {
    return () => {
      abortController?.abort()
    }
  })

  async function handleSend() {
    const message = prompt.trim()
    if (!message || !projectId || !workflowId || !providerId || pending) {
      return
    }

    appendEntry('user', message)
    prompt = ''
    pending = true

    const controller = new AbortController()
    abortController = controller

    try {
      await streamChatTurn(
        {
          message,
          source: 'harness_editor',
          providerId: providerId || undefined,
          sessionId: sessionId || undefined,
          context: {
            projectId,
            workflowId,
          },
        },
        {
          signal: controller.signal,
          onEvent: handleStreamEvent,
        },
      )
    } catch (caughtError) {
      if (!isAbortError(caughtError)) {
        toastStore.error(
          caughtError instanceof ApiError ? caughtError.detail : 'Harness AI request failed.',
        )
      }
    } finally {
      if (abortController === controller) {
        abortController = null
        pending = false
      }
    }
  }

  function handleStreamEvent(event: ChatStreamEvent) {
    if (event.kind === 'done') {
      sessionId = event.payload.sessionId
      pending = false
      return
    }

    if (event.kind === 'error') {
      toastStore.error(event.payload.message)
      pending = false
      return
    }

    const entry = mapChatPayloadToTranscriptEntry(event.payload)
    appendEntry(entry.role, entry.content)
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

  function resetConversation() {
    abortController?.abort()
    abortController = null
    prompt = ''
    pending = false
    sessionId = ''
    entries = []
    appliedFingerprint = ''
  }

  function appendEntry(role: AssistantTranscriptEntry['role'], content: string) {
    entryCounter += 1
    entries = [...entries, { id: `entry-${entryCounter}`, role, content }]
  }

  function handleProviderChange(nextProviderId: string) {
    if (nextProviderId === providerId) return
    providerId = nextProviderId
    resetConversation()
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
      onclick={resetConversation}
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

      {#each entries as entry (entry.id)}
        <div
          class={cn(
            'rounded-2xl border px-3 py-2.5 text-sm leading-6',
            entry.role === 'user' && 'bg-primary text-primary-foreground',
            entry.role === 'assistant' && 'border-border bg-muted/40 text-foreground',
            entry.role === 'system' && 'border-border text-foreground bg-amber-500/10',
          )}
        >
          <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">
            {entry.role}
          </div>
          <div class="break-words whitespace-pre-wrap">{entry.content}</div>
        </div>
      {/each}

      {#if pending}
        <div
          class="border-border bg-muted/30 flex items-center gap-2 rounded-2xl border px-3 py-2.5 text-sm"
        >
          <LoaderCircle class="size-4 animate-spin" />
          Thinking…
        </div>
      {/if}

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
    <HarnessChatProviderSelect
      providers={chatProviders}
      {providerId}
      onProviderChange={handleProviderChange}
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
          ? 'Follow-up prompts reuse the current chat context.'
          : 'The first reply starts a new ephemeral chat session.'}
      </p>
      <Button
        size="sm"
        onclick={handleSend}
        disabled={!prompt.trim() || !projectId || !workflowId || !providerId || pending}
      >
        <Send class="size-4" />
        Send
      </Button>
    </div>
  </div>
</div>
