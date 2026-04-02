<script lang="ts">
  import type { ProjectConversation } from '$lib/api/chat'
  import { cn } from '$lib/utils'
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
    onSelectTab,
    onCloseTab,
  }: {
    tabs: ProjectConversationTabView[]
    activeTabId: string
    conversations: ProjectConversation[]
    onSelectTab: (tabId: string) => void
    onCloseTab: (tabId: string) => void
  } = $props()

  const canClose = $derived.by(() => {
    return (tab: ProjectConversationTabView) =>
      tabs.length > 1 || tab.conversationId || tab.entries.length > 0
  })
</script>

{#if tabs.length > 0}
  <div class="border-border flex items-center gap-px overflow-x-auto border-b px-2">
    {#each tabs as tab (tab.id)}
      {@const label = formatProjectConversationLabel(tab, conversations)}
      {@const status = formatProjectConversationTabStatus(tab)}
      {@const isActive = tab.id === activeTabId}

      <div
        role="tab"
        tabindex="0"
        class={cn(
          'group relative flex max-w-[180px] shrink-0 cursor-pointer items-center gap-1 px-2.5 py-1.5 text-xs transition-colors',
          isActive ? 'text-foreground' : 'text-muted-foreground hover:text-foreground',
        )}
        onclick={() => onSelectTab(tab.id)}
        onkeydown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault()
            onSelectTab(tab.id)
          }
        }}
      >
        {#if isActive}
          <span class="bg-foreground absolute inset-x-2.5 bottom-0 h-[1.5px] rounded-full"></span>
        {/if}
        <span class="max-w-[140px] truncate">{label}</span>
        {#if status}
          <span class="shrink-0 text-[9px] uppercase opacity-50">{status}</span>
        {/if}
        {#if canClose(tab)}
          <button
            type="button"
            class="text-muted-foreground/40 hover:text-foreground -mr-1 ml-0.5 shrink-0 rounded-sm p-0.5 opacity-0 transition-opacity group-hover:opacity-100"
            aria-label={`Close ${label}`}
            onclick={(event) => {
              event.stopPropagation()
              onCloseTab(tab.id)
            }}
          >
            <X class="size-2.5" />
          </button>
        {/if}
      </div>
    {/each}
  </div>
{/if}
