<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import { streamChatTurn, type ChatStreamEvent } from '$lib/api/chat'
  import { toastStore } from '$lib/stores/toast.svelte'
  import { cn } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { ScrollArea } from '$ui/scroll-area'
  import Textarea from '$ui/textarea/textarea.svelte'
  import { Bot, LoaderCircle, RefreshCcw, Send, WandSparkles } from '@lucide/svelte'
  import {
    buildDiffPreview,
    findLatestHarnessSuggestion,
    fingerprintSuggestion,
    isAbortError,
    mapChatPayloadToTranscriptEntry,
    type AssistantTranscriptEntry,
  } from '../assistant'
  import HarnessDiffPreview from './harness-diff-preview.svelte'

  let {
    projectId,
    workflowId,
    workflowName,
    draftContent,
    onApplySuggestion,
  }: {
    projectId?: string
    workflowId?: string
    workflowName?: string
    draftContent: string
    onApplySuggestion?: (content: string) => void
  } = $props()

  let prompt = $state('')
  let pending = $state(false)
  let sessionId = $state('')
  let entries = $state<AssistantTranscriptEntry[]>([])
  let appliedFingerprint = $state('')
  let entryCounter = 0
  let previousContextKey = ''
  let abortController: AbortController | null = null

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
    return () => {
      abortController?.abort()
    }
  })

  async function handleSend() {
    const message = prompt.trim()
    if (!message || !projectId || !workflowId || pending) {
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
    if (!suggestion) {
      return
    }

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
</script>

<div class="bg-background flex h-full min-h-0 flex-col">
  <div class="border-border flex items-center justify-between border-b px-4 py-3">
    <div class="min-w-0">
      <div class="flex items-center gap-2">
        <Bot class="text-primary size-4" />
        <h2 class="text-sm font-semibold">Harness AI</h2>
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
        <div class="border-border bg-muted/30 rounded-xl border border-dashed px-4 py-4 text-sm">
          <div class="flex items-center gap-2 font-medium">
            <WandSparkles class="text-primary size-4" />
            Ask for a harness rewrite, guardrail tweak, or a new workflow handoff.
          </div>
          <p class="text-muted-foreground mt-2 text-xs leading-5">
            When the assistant returns a full harness draft, you can preview the diff here and apply
            it into the editor without leaving the page.
          </p>
        </div>
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
        <div class="space-y-3 rounded-2xl border border-sky-500/30 bg-sky-500/8 p-3">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="text-sm font-medium">Suggested Harness Update</div>
              <p class="text-muted-foreground mt-1 text-xs leading-5">{suggestion.summary}</p>
            </div>
            {#if suggestionAlreadyApplied}
              <Badge variant="outline" class="text-[10px]">Applied</Badge>
            {/if}
          </div>

          <HarnessDiffPreview {preview} />

          <Button
            size="sm"
            class="w-full"
            onclick={handleApply}
            disabled={suggestionAlreadyApplied}
          >
            Apply to Editor
          </Button>
        </div>
      {/if}
    </div>
  </ScrollArea>

  <div class="border-border border-t px-4 py-3">
    <Textarea
      bind:value={prompt}
      rows={4}
      class="text-sm"
      placeholder="Ask the assistant to refine this harness. Shift+Enter for newline."
      disabled={!projectId || !workflowId || pending}
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
        disabled={!prompt.trim() || !projectId || !workflowId || pending}
      >
        <Send class="size-4" />
        Send
      </Button>
    </div>
  </div>
</div>
