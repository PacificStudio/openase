<script lang="ts">
  import { cn } from '$lib/utils'
  import { LoaderCircle } from '@lucide/svelte'
  import EphemeralChatActionProposalCard from './ephemeral-chat-action-proposal-card.svelte'
  import EphemeralChatBundleDiffCard from './ephemeral-chat-bundle-diff-card.svelte'
  import EphemeralChatDiffCard from './ephemeral-chat-diff-card.svelte'
  import ChatMarkdownContent from './chat-markdown-content.svelte'
  import type { EphemeralChatTranscriptEntry } from './transcript'

  let {
    entries,
    pending = false,
    onConfirmActionProposal,
    onCancelActionProposal,
  }: {
    entries: EphemeralChatTranscriptEntry[]
    pending?: boolean
    onConfirmActionProposal?: (entryId: string) => Promise<void> | void
    onCancelActionProposal?: (entryId: string) => void
  } = $props()

  const visibleEntries = $derived(
    entries.filter((entry) => !(entry.kind === 'text' && entry.role === 'system')),
  )

  const hasStreamingAssistantEntry = $derived(
    entries.some((entry) => entry.kind === 'text' && entry.role === 'assistant' && entry.streaming),
  )
</script>

<div class="space-y-1.5">
  {#each visibleEntries as entry (entry.id)}
    {#if entry.kind === 'action_proposal'}
      <EphemeralChatActionProposalCard
        {entry}
        onConfirm={onConfirmActionProposal}
        onCancel={onCancelActionProposal}
      />
    {:else if entry.kind === 'bundle_diff'}
      <EphemeralChatBundleDiffCard {entry} />
    {:else if entry.kind === 'diff'}
      <EphemeralChatDiffCard {entry} />
    {:else}
      <div
        class={cn(
          'rounded-lg px-2.5 py-1.5 text-xs leading-5',
          entry.role === 'user' && 'bg-primary text-primary-foreground ml-6',
          entry.role === 'assistant' && 'border-border bg-muted/40 text-foreground mr-6 border',
        )}
      >
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
      class="border-border bg-muted/30 flex items-center gap-1.5 rounded-lg border px-2.5 py-1.5 text-xs"
    >
      <LoaderCircle class="size-3 animate-spin" />
      Thinking…
    </div>
  {/if}
</div>
