<script lang="ts">
  import { formatRelativeTime } from '$lib/utils'
  import { ScrollArea } from '$ui/scroll-area'
  import { MessageSquare } from '@lucide/svelte'
  import type { ProjectConversation } from '$lib/api/chat'

  let {
    conversations = [],
    openConversationIds = [],
    onSelect,
  }: {
    conversations?: ProjectConversation[]
    openConversationIds?: string[]
    onSelect?: (conversationId: string) => void
  } = $props()

  const openSet = $derived(new Set(openConversationIds))

  function summaryText(conversation: ProjectConversation) {
    if (conversation.rollingSummary?.trim()) {
      return conversation.rollingSummary.trim()
    }
    return 'New conversation'
  }

  function statusDot(conversation: ProjectConversation) {
    switch (conversation.status) {
      case 'active':
        return 'bg-emerald-400'
      case 'idle':
        return 'bg-muted-foreground/30'
      default:
        return 'bg-muted-foreground/20'
    }
  }
</script>

{#if conversations.length === 0}
  <div class="text-muted-foreground flex flex-col items-center gap-2 py-6 text-xs">
    <MessageSquare class="size-5 opacity-40" />
    <span>No conversations yet</span>
  </div>
{:else}
  <ScrollArea class="max-h-80">
    <div class="flex flex-col py-0.5">
      {#each conversations as conversation (conversation.id)}
        {@const isOpen = openSet.has(conversation.id)}
        <button
          type="button"
          class="hover:bg-muted/60 flex w-full items-center gap-2 rounded px-2 py-1.5 text-left transition-colors"
          onclick={() => onSelect?.(conversation.id)}
        >
          <span class={`size-1.5 shrink-0 rounded-full ${statusDot(conversation)}`}></span>
          <span class="text-foreground min-w-0 flex-1 truncate text-[11px] leading-tight">
            {summaryText(conversation)}
          </span>
          {#if isOpen}
            <span
              class="bg-muted text-muted-foreground shrink-0 rounded px-1 py-0.5 text-[9px] leading-none"
            >
              open
            </span>
          {/if}
          <span class="text-muted-foreground shrink-0 text-[10px]">
            {formatRelativeTime(conversation.lastActivityAt || conversation.createdAt)}
          </span>
        </button>
      {/each}
    </div>
  </ScrollArea>
{/if}
