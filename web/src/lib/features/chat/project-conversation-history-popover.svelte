<script lang="ts">
  import { chatT } from './i18n'
  import { formatRelativeTime } from '$lib/utils'
  import { MessageSquare, Trash2 } from '@lucide/svelte'
  import type { ProjectConversation } from '$lib/api/chat'
  import {
    getProjectConversationDisplayTitle,
    getProjectConversationSummary,
  } from './project-conversation-panel-labels'

  let {
    conversations = [],
    openConversationIds = [],
    onSelect,
    onDelete,
  }: {
    conversations?: ProjectConversation[]
    openConversationIds?: string[]
    onSelect?: (conversationId: string) => void
    onDelete?: (conversationId: string) => void
  } = $props()

  const openSet = $derived(new Set(openConversationIds))

  function titleText(conversation: ProjectConversation) {
    const title = getProjectConversationDisplayTitle(conversation)
    if (title) {
      return title
    }
    return chatT('chat.newConversation')
  }

  function secondarySummaryText(conversation: ProjectConversation) {
    const summary = getProjectConversationSummary(conversation)
    if (!summary) {
      return ''
    }
    if (summary === titleText(conversation)) {
      return ''
    }
    return summary
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
    <span>{chatT('chat.noConversationsYet')}</span>
  </div>
{:else}
  <div
    class="max-h-80 overflow-y-auto overscroll-contain"
    data-testid="conversation-history-scroll"
  >
    <div class="flex flex-col py-0.5">
      {#each conversations as conversation (conversation.id)}
        {@const isOpen = openSet.has(conversation.id)}
        <div class="hover:bg-muted/60 flex items-center gap-1 rounded px-1 py-1 transition-colors">
          <button
            type="button"
            class="flex min-w-0 flex-1 items-center gap-2 rounded px-1 py-0.5 text-left"
            aria-label={chatT('chat.openConversationLabel', { title: titleText(conversation) })}
            onclick={() => onSelect?.(conversation.id)}
          >
            <span class={`size-1.5 shrink-0 rounded-full ${statusDot(conversation)}`}></span>
            <span class="min-w-0 flex-1">
              <span class="text-foreground block truncate text-[11px] leading-tight">
                {titleText(conversation)}
              </span>
              {#if secondarySummaryText(conversation)}
                <span class="text-muted-foreground block truncate text-[10px] leading-tight">
                  {secondarySummaryText(conversation)}
                </span>
              {/if}
            </span>
            {#if isOpen}
              <span
                class="bg-muted text-muted-foreground shrink-0 rounded px-1 py-0.5 text-[9px] leading-none"
              >
                {chatT('chat.statusOpen')}
              </span>
            {/if}
            <span class="text-muted-foreground shrink-0 text-[10px]">
              {formatRelativeTime(conversation.lastActivityAt || conversation.createdAt)}
            </span>
          </button>
          <button
            type="button"
            class="text-muted-foreground hover:text-destructive inline-flex size-6 shrink-0 items-center justify-center rounded transition-colors"
            aria-label={chatT('chat.deleteConversationLabel', { title: titleText(conversation) })}
            title={chatT('chat.deleteConversationTitle')}
            onclick={(event) => {
              event.stopPropagation()
              onDelete?.(conversation.id)
            }}
          >
            <Trash2 class="size-3.5" />
          </button>
        </div>
      {/each}
    </div>
  </div>
{/if}
