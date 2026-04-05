<script lang="ts">
  import type { ProjectConversation } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import * as Tooltip from '$ui/tooltip'
  import { LoaderCircle, X } from '@lucide/svelte'
  import {
    formatProjectConversationLabel,
    formatProjectConversationTabStatus,
    type ProjectConversationTabView,
  } from './project-conversation-panel-labels'
  import { getProjectColor } from './project-conversation-tab-colors'

  let {
    tabs,
    activeTabId,
    conversations,
    currentProjectId = '',
    onSelectTab,
    onCloseTab,
  }: {
    tabs: ProjectConversationTabView[]
    activeTabId: string
    conversations: ProjectConversation[]
    currentProjectId?: string
    onSelectTab: (tabId: string) => void
    onCloseTab: (tabId: string) => void
  } = $props()

  let confirmCloseTabId = $state('')
  let confirmOpen = $state(false)

  const canClose = $derived.by(() => {
    return (tab: ProjectConversationTabView) =>
      tabs.length > 1 || tab.conversationId || tab.entries.length > 0 || tab.draft.trim().length > 0
  })

  function handleCloseClick(tab: ProjectConversationTabView) {
    if (tab.pending) {
      confirmCloseTabId = tab.id
      confirmOpen = true
    } else {
      onCloseTab(tab.id)
    }
  }

  function handleConfirmClose() {
    const tabId = confirmCloseTabId
    confirmOpen = false
    confirmCloseTabId = ''
    onCloseTab(tabId)
  }

  function handleCancelClose() {
    confirmOpen = false
    confirmCloseTabId = ''
  }
</script>

{#if tabs.length > 0}
  <div class="border-border flex items-center gap-px overflow-x-auto border-b px-2">
    {#each tabs as tab (tab.id)}
      {@const label = formatProjectConversationLabel(tab, conversations)}
      {@const status = formatProjectConversationTabStatus(tab)}
      {@const isActive = tab.id === activeTabId}
      {@const isCrossProject =
        !!tab.projectId && !!currentProjectId && tab.projectId !== currentProjectId}
      {@const projectColor = isCrossProject ? getProjectColor(tab.projectId) : ''}
      {@const projectLabel = tab.projectName || 'Other project'}

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
        {#if isCrossProject}
          <Tooltip.Root>
            <Tooltip.Trigger>
              {#snippet child({ props })}
                <span
                  {...props}
                  class="absolute inset-x-0 top-0 h-[3px]"
                  style="background-color: {projectColor}"
                ></span>
              {/snippet}
            </Tooltip.Trigger>
            <Tooltip.Content side="bottom" class="text-xs">
              {projectLabel}
            </Tooltip.Content>
          </Tooltip.Root>
        {/if}
        {#if isActive}
          <span class="bg-foreground absolute inset-x-2.5 bottom-0 h-[1.5px] rounded-full"></span>
        {/if}
        {#if tab.pending}
          <LoaderCircle class="size-3 shrink-0 animate-spin opacity-50" />
        {/if}
        {#if isCrossProject && isActive}
          <span
            class="shrink-0 rounded-sm px-1 py-px text-[9px] leading-tight font-medium text-white"
            style="background-color: {projectColor}">{projectLabel}</span
          >
        {/if}
        <span class="max-w-[140px] truncate">{label}</span>
        {#if status && !tab.pending}
          <span class="shrink-0 text-[9px] uppercase opacity-50">{status}</span>
        {/if}
        {#if canClose(tab)}
          <button
            type="button"
            class="text-muted-foreground/40 hover:text-foreground -mr-1 ml-0.5 shrink-0 rounded-sm p-0.5 opacity-0 transition-opacity group-hover:opacity-100"
            aria-label={`Close ${label}`}
            onclick={(event) => {
              event.stopPropagation()
              handleCloseClick(tab)
            }}
          >
            <X class="size-2.5" />
          </button>
        {/if}
      </div>
    {/each}
  </div>
{/if}

<Dialog.Root bind:open={confirmOpen}>
  <Dialog.Content class="max-w-sm">
    <Dialog.Header>
      <Dialog.Title class="text-sm">Close active conversation?</Dialog.Title>
      <Dialog.Description class="text-muted-foreground text-xs">
        This conversation is still running. Closing the tab may interrupt the current operation.
      </Dialog.Description>
    </Dialog.Header>
    <Dialog.Footer class="gap-2">
      <Button variant="ghost" size="sm" class="text-xs" onclick={handleCancelClose}>Cancel</Button>
      <Button variant="destructive" size="sm" class="text-xs" onclick={handleConfirmClose}>
        Close anyway
      </Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
