<script lang="ts">
  import { cn } from '$lib/utils'
  import { LoaderCircle } from '@lucide/svelte'
  import EphemeralChatActionProposalCard from './ephemeral-chat-action-proposal-card.svelte'
  import EphemeralChatDiffCard from './ephemeral-chat-diff-card.svelte'
  import ChatMarkdownContent from './chat-markdown-content.svelte'
  import ProjectConversationCommandOutputCard from './project-conversation-command-output-card.svelte'
  import ProjectConversationInterruptCard from './project-conversation-interrupt-card.svelte'
  import ProjectConversationToolCallCard from './project-conversation-tool-call-card.svelte'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-state'

  let {
    entries,
    pending = false,
    onConfirmActionProposal,
    onRespondInterrupt,
  }: {
    entries: ProjectConversationTranscriptEntry[]
    pending?: boolean
    onConfirmActionProposal?: (entryId: string) => Promise<void> | void
    onRespondInterrupt?: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => Promise<void> | void
  } = $props()
</script>

<div class="space-y-3">
  {#each entries as entry (entry.id)}
    {#if entry.kind === 'action_proposal'}
      <EphemeralChatActionProposalCard
        {entry}
        onConfirm={onConfirmActionProposal}
        onCancel={undefined}
      />
    {:else if entry.kind === 'diff'}
      <EphemeralChatDiffCard {entry} />
    {:else if entry.kind === 'tool_call'}
      <ProjectConversationToolCallCard {entry} />
    {:else if entry.kind === 'command_output'}
      <ProjectConversationCommandOutputCard {entry} />
    {:else if entry.kind === 'task_status'}
      <div class="border-border/70 bg-muted/20 rounded-2xl border px-3 py-2.5 text-sm">
        <div class="mb-1 text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">
          status
        </div>
        <div class="font-medium">{entry.title}</div>
        {#if entry.detail}
          <div class="text-muted-foreground mt-1 text-xs leading-5 whitespace-pre-wrap">
            {entry.detail}
          </div>
        {/if}
      </div>
    {:else if entry.kind === 'interrupt'}
      <ProjectConversationInterruptCard {entry} {onRespondInterrupt} />
    {:else}
      <div
        class={cn(
          'rounded-2xl border px-3 py-2.5 text-sm leading-6',
          entry.role === 'user' && 'bg-primary text-primary-foreground',
          entry.role === 'assistant' && 'border-border bg-muted/40 text-foreground',
          entry.role === 'system' && 'border-border bg-muted/20 text-foreground',
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

  {#if pending}
    <div
      class="border-border bg-muted/30 flex items-center gap-2 rounded-2xl border px-3 py-2.5 text-sm"
    >
      <LoaderCircle class="size-4 animate-spin" />
      Thinking…
    </div>
  {/if}
</div>
