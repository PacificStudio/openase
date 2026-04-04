<script lang="ts">
  import { Button } from '$ui/button'
  import { Send } from '@lucide/svelte'
  import ProjectConversationFocusCard from './project-conversation-focus-card.svelte'

  type FocusCard = {
    label: string
    title: string
    detail?: string
  }

  type QueuedTurn = {
    id: string
    message: string
  }

  let {
    loadingProviders = false,
    providerError = '',
    providerCount = 0,
    statusMessage = '',
    focusCard = null,
    queuedTurns = [],
    hasPendingInterrupt = false,
    draft = '',
    placeholder = 'Ask anything about this project…',
    inputDisabled = false,
    sendDisabled = false,
    canQueueTurn = false,
    onDismissFocus,
    onCancelQueuedTurn,
    onDraftChange,
    onSend,
  }: {
    loadingProviders?: boolean
    providerError?: string
    providerCount?: number
    statusMessage?: string
    focusCard?: FocusCard | null
    queuedTurns?: QueuedTurn[]
    hasPendingInterrupt?: boolean
    draft?: string
    placeholder?: string
    inputDisabled?: boolean
    sendDisabled?: boolean
    canQueueTurn?: boolean
    onDismissFocus: () => void
    onCancelQueuedTurn?: (queuedTurnId: string) => void
    onDraftChange?: (value: string) => void
    onSend?: () => void
  } = $props()

  let textareaEl = $state<HTMLTextAreaElement | null>(null)

  function autoResize(el: HTMLTextAreaElement) {
    el.style.height = 'auto'
    el.style.height = `${el.scrollHeight}px`
  }

  $effect(() => {
    // re-run whenever draft changes
    void draft
    if (textareaEl) autoResize(textareaEl)
  })

  function truncateQueuedMessage(value: string) {
    return value.length > 80 ? `${value.slice(0, 80)}…` : value
  }
</script>

<div class="border-border border-t px-3 py-2">
  {#if loadingProviders}
    <div class="text-muted-foreground mb-1.5 text-[11px]">Loading providers...</div>
  {:else if providerError}
    <div class="text-destructive mb-1.5 text-[11px]">{providerError}</div>
  {:else if providerCount === 0}
    <div class="text-muted-foreground mb-1.5 text-[11px]">No chat provider available.</div>
  {:else if statusMessage}
    <div class="text-muted-foreground mb-1.5 text-[11px]">{statusMessage}</div>
  {/if}

  {#if focusCard}
    <ProjectConversationFocusCard
      label={focusCard.label}
      title={focusCard.title}
      detail={focusCard.detail}
      onDismiss={onDismissFocus}
    />
  {/if}

  {#if queuedTurns.length > 0}
    <div class="mb-1.5 space-y-1">
      {#each queuedTurns as queuedTurn, index (queuedTurn.id)}
        <div class="flex items-center gap-1.5 text-[11px]">
          <span class="text-muted-foreground shrink-0">
            {#if hasPendingInterrupt}Paused{:else}Queued{/if}
          </span>
          <span class="text-foreground min-w-0 flex-1 truncate">
            {truncateQueuedMessage(queuedTurn.message)}
          </span>
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground shrink-0 underline-offset-2 hover:underline"
            aria-label={`Cancel queued message ${index + 1}`}
            onclick={() => {
              onCancelQueuedTurn?.(queuedTurn.id)
            }}
          >
            Cancel
          </button>
        </div>
      {/each}
    </div>
  {/if}

  <div class="flex items-end gap-1.5 px-0.5">
    <textarea
      bind:this={textareaEl}
      value={draft}
      rows={1}
      class="placeholder:text-muted-foreground max-h-[calc(4*1.5em+0.5rem)] min-h-0 flex-1 resize-none overflow-y-auto border-0 bg-transparent px-0 py-1 text-sm leading-[1.5] shadow-none outline-none focus-visible:ring-0 disabled:cursor-not-allowed disabled:opacity-50"
      {placeholder}
      disabled={inputDisabled}
      oninput={(event) => {
        const el = event.currentTarget as HTMLTextAreaElement
        onDraftChange?.(el.value)
        autoResize(el)
      }}
      onkeydown={(event) => {
        if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
          event.preventDefault()
          onSend?.()
        }
      }}
    ></textarea>
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground size-6 shrink-0 p-0"
      aria-label="Send message"
      onclick={() => onSend?.()}
      disabled={!draft.trim() || (sendDisabled && !canQueueTurn)}
    >
      <Send class="size-3.5" />
    </Button>
  </div>
</div>
