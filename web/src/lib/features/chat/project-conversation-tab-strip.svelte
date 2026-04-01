<script lang="ts">
  import type { ProjectConversation } from '$lib/api/chat'
  import { X } from '@lucide/svelte'
  import {
    formatProjectConversationLabel,
    formatProjectConversationTabStatus,
    type ProjectConversationTabView,
  } from './project-conversation-panel-labels'

  let {
    tabs,
    activeTabId,
    conversations,
    historicalConversations,
    providerId = '',
    onSelectTab,
    onCloseTab,
    onOpenConversation,
  }: {
    tabs: ProjectConversationTabView[]
    activeTabId: string
    conversations: ProjectConversation[]
    historicalConversations: ProjectConversation[]
    providerId?: string
    onSelectTab: (tabId: string) => void
    onCloseTab: (tabId: string) => void
    onOpenConversation: (conversationId: string) => Promise<void> | void
  } = $props()

  let openConversationId = $state('')

  async function handleOpenConversation(nextConversationId: string) {
    if (!nextConversationId) {
      return
    }
    openConversationId = ''
    await onOpenConversation(nextConversationId)
  }
</script>

<div class="border-border border-b px-4 py-2">
  <div class="flex flex-wrap gap-2">
    {#each tabs as tab (tab.id)}
      {@const label = formatProjectConversationLabel(tab, conversations)}
      {@const status = formatProjectConversationTabStatus(tab)}
      <div
        class:bg-accent={tab.id === activeTabId}
        class="border-input flex max-w-full items-center gap-1 rounded-md border px-2 py-1"
      >
        <button type="button" class="min-w-0 text-left" onclick={() => onSelectTab(tab.id)}>
          <div class="truncate text-sm font-medium">{label}</div>
          {#if status}
            <div class="text-muted-foreground text-[10px] uppercase">{status}</div>
          {/if}
        </button>
        {#if tabs.length > 1 || tab.conversationId || tab.entries.length > 0}
          <button
            type="button"
            class="text-muted-foreground hover:text-foreground rounded p-0.5"
            aria-label={`Close ${label}`}
            onclick={(event) => {
              event.stopPropagation()
              onCloseTab(tab.id)
            }}
          >
            <X class="size-3" />
          </button>
        {/if}
      </div>
    {/each}
  </div>

  {#if historicalConversations.length > 0}
    <div class="mt-2">
      <label
        class="text-muted-foreground mb-1 block text-[10px] font-semibold tracking-[0.16em] uppercase"
        for="project-conversation-open"
      >
        Open Existing Conversation
      </label>
      <select
        id="project-conversation-open"
        class="border-input bg-background h-9 w-full rounded-md border px-3 text-sm"
        bind:value={openConversationId}
        disabled={!providerId}
        onchange={(event) =>
          void handleOpenConversation((event.currentTarget as HTMLSelectElement).value)}
      >
        <option value="">Select a conversation</option>
        {#each historicalConversations as conversation (conversation.id)}
          <option value={conversation.id}>
            {conversation.rollingSummary?.trim() ||
              formatProjectConversationLabel(
                {
                  conversationId: conversation.id,
                  entries: [],
                },
                conversations,
              )}
          </option>
        {/each}
      </select>
    </div>
  {/if}
</div>
