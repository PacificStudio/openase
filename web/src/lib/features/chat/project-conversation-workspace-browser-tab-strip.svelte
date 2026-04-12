<script lang="ts">
  import { FileCode2, X } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import {
    workspaceTabKey,
    type WorkspaceTab,
  } from './project-conversation-workspace-browser-state.svelte'

  let {
    openTabs,
    activeTabKey,
    isTabDirty,
    onActivateTab,
    onRequestCloseTab,
  }: {
    openTabs: WorkspaceTab[]
    activeTabKey: string
    isTabDirty: (repoPath: string, filePath: string) => boolean
    onActivateTab: (repoPath: string, filePath: string) => void
    onRequestCloseTab: (event: MouseEvent, repoPath: string, filePath: string) => void
  } = $props()
</script>

<div
  class="border-border bg-muted/20 flex min-h-9 shrink-0 items-stretch overflow-x-auto border-b"
  data-testid="workspace-browser-detail-tab-bar"
  role="tablist"
>
  {#each openTabs as tab (workspaceTabKey(tab))}
    {@const tabKey = workspaceTabKey(tab)}
    {@const isActive = tabKey === activeTabKey}
    {@const dirty = isTabDirty(tab.repoPath, tab.filePath)}
    {@const tabName = tab.filePath.split('/').pop() ?? tab.filePath}
    <button
      type="button"
      role="tab"
      aria-selected={isActive}
      class={cn(
        'border-border flex max-w-[220px] min-w-0 shrink-0 items-center gap-1.5 border-r px-2.5 py-1.5 text-[12px] transition-colors',
        isActive
          ? 'bg-background text-foreground'
          : 'text-muted-foreground hover:bg-muted/40 hover:text-foreground',
      )}
      onclick={() => onActivateTab(tab.repoPath, tab.filePath)}
      data-testid={`workspace-browser-detail-tab-${tabName}`}
    >
      {#if dirty}
        <span
          class="size-1.5 shrink-0 rounded-full bg-orange-500"
          aria-label="Unsaved changes"
          data-testid="workspace-browser-detail-tab-dirty-dot"
        ></span>
      {:else}
        <FileCode2 class="size-3 shrink-0 opacity-60" />
      {/if}
      <span class="min-w-0 truncate">{tabName}</span>
      <span
        role="button"
        tabindex="0"
        class="hover:bg-muted/80 ml-0.5 inline-flex size-4 shrink-0 items-center justify-center rounded transition-colors"
        aria-label={`Close ${tabName}`}
        onclick={(event) => onRequestCloseTab(event, tab.repoPath, tab.filePath)}
        onkeydown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault()
            event.stopPropagation()
            onRequestCloseTab(event as unknown as MouseEvent, tab.repoPath, tab.filePath)
          }
        }}
      >
        <X class="size-3" />
      </span>
    </button>
  {/each}
</div>
