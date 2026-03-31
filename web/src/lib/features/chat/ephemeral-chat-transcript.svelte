<script lang="ts">
  import { cn } from '$lib/utils'
  import { LoaderCircle } from '@lucide/svelte'
  import EphemeralChatActionProposalCard from './ephemeral-chat-action-proposal-card.svelte'
  import EphemeralChatDiffCard from './ephemeral-chat-diff-card.svelte'
  import ChatMarkdownContent from './chat-markdown-content.svelte'
  import type { EphemeralChatTranscriptEntry } from './transcript'

  let {
    entries,
    pending = false,
    emptyStateTitle = '',
    emptyStateDescription = '',
    onConfirmActionProposal,
    onCancelActionProposal,
  }: {
    entries: EphemeralChatTranscriptEntry[]
    pending?: boolean
    emptyStateTitle?: string
    emptyStateDescription?: string
    onConfirmActionProposal?: (entryId: string) => Promise<void> | void
    onCancelActionProposal?: (entryId: string) => void
  } = $props()

  const hasStreamingAssistantEntry = $derived(
    entries.some((entry) => entry.kind === 'text' && entry.role === 'assistant' && entry.streaming),
  )
</script>

<div class="space-y-3">
  {#if entries.length === 0 && emptyStateTitle}
    <div class="border-border bg-muted/20 rounded-2xl border px-4 py-3">
      <div class="text-sm font-medium">{emptyStateTitle}</div>
      {#if emptyStateDescription}
        <p class="text-muted-foreground mt-1 text-xs leading-5">{emptyStateDescription}</p>
      {/if}
    </div>
  {/if}

  {#each entries as entry (entry.id)}
    {#if entry.kind === 'action_proposal'}
      <EphemeralChatActionProposalCard
        {entry}
        onConfirm={onConfirmActionProposal}
        onCancel={onCancelActionProposal}
      />
    {:else if entry.kind === 'diff'}
      <EphemeralChatDiffCard {entry} />
    {:else}
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
        {#if entry.role === 'assistant'}
          <ChatMarkdownContent source={entry.content} />
        {:else}
          <div class="break-words whitespace-pre-wrap">{entry.content}</div>
        {/if}
      </div>
    {/if}
  {/each}

  {#if pending && !hasStreamingAssistantEntry}
    <div
      class="border-border bg-muted/30 flex items-center gap-2 rounded-2xl border px-3 py-2.5 text-sm"
    >
      <LoaderCircle class="size-4 animate-spin" />
      Thinking…
    </div>
  {/if}
</div>
