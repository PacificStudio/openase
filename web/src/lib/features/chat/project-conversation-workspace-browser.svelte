<script lang="ts">
  import { untrack } from 'svelte'
  import { Button } from '$ui/button'
  import { cn } from '$lib/utils'
  import { AlertCircle, FolderTree, RefreshCcw, X } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'
  import ProjectConversationWorkspaceBrowserSidebar from './project-conversation-workspace-browser-sidebar.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'

  let {
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    onClose,
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    onClose?: () => void
  } = $props()

  const browser = createProjectConversationWorkspaceBrowserState({
    getConversationId: () => conversationId,
  })

  let refreshGeneration = $state(0)
  let lastRefreshKey = $state('')
  let lastWorkspaceDiffLoading = $state(false)

  const selectedRepo = $derived(
    browser.metadata?.repos.find((repo) => repo.path === browser.selectedRepoPath) ??
      browser.metadata?.repos[0] ??
      null,
  )
  const selectedRepoDiff = $derived(
    workspaceDiff?.repos.find((repo) => repo.path === browser.selectedRepoPath) ?? null,
  )

  $effect(() => {
    const nextLoading = workspaceDiffLoading
    if (lastWorkspaceDiffLoading && !nextLoading && conversationId) {
      refreshGeneration += 1
    }
    lastWorkspaceDiffLoading = nextLoading
  })

  $effect(() => {
    if (!conversationId) {
      lastRefreshKey = ''
      browser.reset()
      return
    }

    const refreshKey = `${conversationId}:${refreshGeneration}`
    if (lastRefreshKey === refreshKey) {
      return
    }
    lastRefreshKey = refreshKey

    untrack(() => {
      void browser.refreshWorkspace(true)
    })
  })
</script>

<div
  class="bg-background flex h-full min-h-0 w-full flex-col"
  data-testid="project-conversation-workspace-browser"
>
  <div class="border-border flex items-center gap-2 border-b px-3 py-2">
    <div class="min-w-0 flex-1">
      <div class="flex items-center gap-1.5">
        <FolderTree class="text-muted-foreground size-3.5 shrink-0" />
        <h3 class="text-sm font-semibold">Workspace browser</h3>
      </div>
      <p class="text-muted-foreground truncate text-[11px]">
        {browser.metadata?.workspacePath || 'Read-only view of the active Project AI workdir'}
      </p>
    </div>
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground size-7 p-0"
      aria-label="Refresh workspace browser"
      onclick={() => void browser.refreshWorkspace(true)}
      disabled={!conversationId || browser.metadataLoading}
    >
      <RefreshCcw class={cn('size-3.5', browser.metadataLoading && 'animate-spin')} />
    </Button>
    {#if onClose}
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground size-7 p-0"
        aria-label="Close workspace browser"
        onclick={onClose}
      >
        <X class="size-3.5" />
      </Button>
    {/if}
  </div>

  {#if !conversationId}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Start or reopen a Project AI conversation to inspect its workspace.
    </div>
  {:else if browser.metadataLoading && !browser.metadata}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      Loading workspace…
    </div>
  {:else if browser.metadataError}
    <div class="flex flex-1 items-center justify-center px-6">
      <div class="max-w-sm space-y-3 text-center">
        <div
          class="bg-destructive/10 text-destructive mx-auto flex size-10 items-center justify-center rounded-full"
        >
          <AlertCircle class="size-4" />
        </div>
        <p class="text-sm font-medium">Workspace browser unavailable</p>
        <p class="text-muted-foreground text-sm">{browser.metadataError}</p>
      </div>
    </div>
  {:else if !browser.metadata?.available}
    <div
      class="text-muted-foreground flex flex-1 items-center justify-center px-6 text-center text-sm"
    >
      The workspace will appear after Project AI provisions the conversation workdir.
    </div>
  {:else}
    <div class="grid min-h-0 flex-1 grid-cols-[18rem_minmax(0,1fr)]">
      <ProjectConversationWorkspaceBrowserSidebar
        repos={browser.metadata?.repos ?? []}
        selectedRepoPath={browser.selectedRepoPath}
        {selectedRepo}
        {selectedRepoDiff}
        tree={browser.tree}
        treeLoading={browser.treeLoading}
        treeError={browser.treeError}
        selectedFilePath={browser.selectedFilePath}
        currentTreePath={browser.currentTreePath}
        onOpenRepo={browser.openRepo}
        onOpenTreePath={browser.openTreePath}
        onOpenEntry={browser.openEntry}
        onOpenDirtyFile={browser.openDirtyFile}
      />
      <ProjectConversationWorkspaceBrowserDetail
        {selectedRepo}
        selectedFilePath={browser.selectedFilePath}
        currentTreePath={browser.currentTreePath}
        preview={browser.preview}
        patch={browser.patch}
        fileLoading={browser.fileLoading}
        fileError={browser.fileError}
      />
    </div>
  {/if}
</div>
