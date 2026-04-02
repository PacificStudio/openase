<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronRight, LoaderCircle, Layers } from '@lucide/svelte'
  import EphemeralChatActionProposalCard from './ephemeral-chat-action-proposal-card.svelte'
  import EphemeralChatDiffCard from './ephemeral-chat-diff-card.svelte'
  import ChatMarkdownContent from './chat-markdown-content.svelte'
  import ProjectConversationCommandOutputCard from './project-conversation-command-output-card.svelte'
  import ProjectConversationInterruptCard from './project-conversation-interrupt-card.svelte'
  import ProjectConversationTaskStatusCard from './project-conversation-task-status-card.svelte'
  import ProjectConversationToolCallCard from './project-conversation-tool-call-card.svelte'
  import type { ProjectConversationTranscriptEntry } from './project-conversation-transcript-types'
  import {
    groupTranscriptEntries,
    type OperationGroup,
  } from './project-conversation-transcript-grouping'

  let {
    entries,
    pending = false,
    onConfirmActionProposal,
    onCancelActionProposal,
    onRespondInterrupt,
  }: {
    entries: ProjectConversationTranscriptEntry[]
    pending?: boolean
    onConfirmActionProposal?: (entryId: string) => Promise<void> | void
    onCancelActionProposal?: (entryId: string) => void
    onRespondInterrupt?: (input: {
      interruptId: string
      decision?: string
      answer?: Record<string, unknown>
    }) => Promise<void> | void
  } = $props()

  const displayItems = $derived(groupTranscriptEntries(entries))

  let expandedGroups = $state(new Set<string>())

  function toggleGroup(groupId: string) {
    const next = new Set(expandedGroups)
    if (next.has(groupId)) {
      next.delete(groupId)
    } else {
      next.add(groupId)
    }
    expandedGroups = next
  }
</script>

<div class="space-y-2">
  {#each displayItems as item (item.type === 'standalone' ? item.entry.id : item.id)}
    {#if item.type === 'standalone'}
      {@const entry = item.entry}

      {#if entry.kind === 'action_proposal'}
        <EphemeralChatActionProposalCard
          {entry}
          onConfirm={onConfirmActionProposal}
          onCancel={onCancelActionProposal}
        />
      {:else if entry.kind === 'diff'}
        <EphemeralChatDiffCard {entry} />
      {:else if entry.kind === 'interrupt'}
        <ProjectConversationInterruptCard {entry} {onRespondInterrupt} />
      {:else if entry.kind === 'text'}
        {#if entry.role === 'user'}
          <div class="flex justify-end">
            <div
              class="bg-foreground/5 text-foreground max-w-[85%] rounded-2xl rounded-br-md px-3 py-1.5 text-sm"
            >
              <div class="break-words whitespace-pre-wrap">{entry.content}</div>
            </div>
          </div>
        {:else if entry.role === 'assistant'}
          <div class="mx-auto max-w-full">
            <ChatMarkdownContent source={entry.content} />
          </div>
        {:else}
          <div class="text-muted-foreground text-xs break-words whitespace-pre-wrap">
            {entry.content}
          </div>
        {/if}
      {/if}
    {:else}
      <!-- Operation Group: collapsible block of system entries -->
      {@const group = item as OperationGroup}
      {@const isExpanded = expandedGroups.has(group.id)}

      <div class="border-border/50 bg-muted/10 rounded-lg border">
        <button
          type="button"
          class="hover:bg-muted/30 flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-xs transition-colors"
          onclick={() => toggleGroup(group.id)}
        >
          <ChevronRight
            class={cn(
              'text-muted-foreground size-3.5 shrink-0 transition-transform duration-150',
              isExpanded && 'rotate-90',
            )}
          />
          <Layers class="text-muted-foreground/70 size-3.5 shrink-0" />
          <span class="text-foreground min-w-0 flex-1 truncate font-medium">
            {group.summary}
          </span>
          {#if group.detail}
            <span class="text-muted-foreground/60 shrink-0 text-[10px]">{group.detail}</span>
          {/if}
          <span class="text-muted-foreground/40 shrink-0 text-[10px]">
            {group.entries.length} item{group.entries.length === 1 ? '' : 's'}
          </span>
        </button>

        {#if isExpanded}
          <div class="border-border/30 border-t px-1 py-1">
            {#each group.entries as groupEntry (groupEntry.id)}
              {#if groupEntry.kind === 'tool_call'}
                <ProjectConversationToolCallCard entry={groupEntry} />
              {:else if groupEntry.kind === 'command_output'}
                <ProjectConversationCommandOutputCard entry={groupEntry} />
              {:else if groupEntry.kind === 'task_status'}
                <ProjectConversationTaskStatusCard entry={groupEntry} />
              {/if}
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/each}

  {#if pending}
    <div class="text-muted-foreground flex items-center gap-1.5 py-1 text-xs">
      <LoaderCircle class="size-3 animate-spin" />
      <span>Thinking...</span>
    </div>
  {/if}
</div>
