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
  <ScrollArea class="max-h-72">
    <div class="flex flex-col gap-0.5 py-1">
      {#each conversations as conversation (conversation.id)}
        {@const isOpen = openSet.has(conversation.id)}
        <button
          type="button"
          class="hover:bg-muted/60 flex w-full items-start gap-2.5 rounded-md px-2.5 py-2 text-left transition-colors"
          onclick={() => onSelect?.(conversation.id)}
        >
          <span class={`mt-1.5 size-1.5 shrink-0 rounded-full ${statusDot(conversation)}`}></span>
          <div class="min-w-0 flex-1">
            <div class="text-foreground flex items-center gap-1.5 text-xs font-medium">
              <span class="truncate">{summaryText(conversation)}</span>
              {#if isOpen}
                <span
                  class="bg-muted text-muted-foreground shrink-0 rounded px-1 py-0.5 text-[9px] leading-none"
                >
                  open
                </span>
              {/if}
            </div>
            <div class="text-muted-foreground mt-0.5 text-[10px]">
              {formatRelativeTime(conversation.lastActivityAt || conversation.createdAt)}
            </div>
          </div>
        </button>
      {/each}
    </div>
  </ScrollArea>
{/if}
