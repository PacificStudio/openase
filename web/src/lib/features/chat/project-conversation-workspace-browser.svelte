<script lang="ts">
  import { untrack } from 'svelte'
  import { Button } from '$ui/button'
  import { cn } from '$lib/utils'
  import { AlertCircle, Check, Copy, FolderTree, RefreshCcw, X } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'
  import ProjectConversationWorkspaceBrowserSidebar from './project-conversation-workspace-browser-sidebar.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'

  let {
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    pendingFilePath = '',
    onClose,
    onPendingFileConsumed,
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    /** File path to navigate to (consumed once on change). */
    pendingFilePath?: string
    onClose?: () => void
    onPendingFileConsumed?: () => void
  } = $props()

  const browser = createProjectConversationWorkspaceBrowserState({
    getConversationId: () => conversationId,
  })

  let pathCopied = $state(false)

  function copyWorkspacePath() {
    const path = browser.metadata?.workspacePath
    if (!path) return
    navigator.clipboard.writeText(path)
    pathCopied = true
    setTimeout(() => (pathCopied = false), 1500)
  }

  // -- Sidebar resize --
  const MIN_SIDEBAR_WIDTH = 180
  const MAX_SIDEBAR_WIDTH = 480
  let sidebarWidth = $state(240)
  let sidebarResizing = $state(false)

  function handleSidebarResizeStart(event: PointerEvent) {
    event.preventDefault()
    sidebarResizing = true
    const startX = event.clientX
    const startWidth = sidebarWidth

    function onMove(e: PointerEvent) {
      sidebarWidth = Math.min(
        MAX_SIDEBAR_WIDTH,
        Math.max(MIN_SIDEBAR_WIDTH, startWidth + (e.clientX - startX)),
      )
    }

    function onUp() {
      sidebarResizing = false
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

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

  // Navigate to a pending file when set
  $effect(() => {
    if (pendingFilePath && browser.metadata?.available) {
      untrack(() => {
        browser.selectFile(pendingFilePath)
        onPendingFileConsumed?.()
      })
    }
  })

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
  <!-- Compact toolbar -->
  <div class="border-border flex h-9 items-center gap-1.5 border-b px-3">
    <FolderTree class="text-muted-foreground size-3 shrink-0" />
    <span class="text-[12px] font-semibold">Workspace</span>
    {#if browser.metadata?.workspacePath}
      <button
        type="button"
        class="text-muted-foreground/50 hover:text-muted-foreground group flex min-w-0 items-center gap-1 truncate text-[11px] transition-colors"
        title="Click to copy path"
        onclick={copyWorkspacePath}
      >
        <span class="min-w-0 truncate">{browser.metadata.workspacePath}</span>
        {#if pathCopied}
          <Check class="size-2.5 shrink-0 text-emerald-500" />
        {:else}
          <Copy class="size-2.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
        {/if}
      </button>
    {/if}
    <div class="flex-1"></div>
    <Button
      variant="ghost"
      size="sm"
      class="text-muted-foreground size-6 p-0"
      aria-label="Refresh workspace browser"
      onclick={() => void browser.refreshWorkspace(true)}
      disabled={!conversationId || browser.metadataLoading}
    >
      <RefreshCcw class={cn('size-3', browser.metadataLoading && 'animate-spin')} />
    </Button>
    {#if onClose}
      <Button
        variant="ghost"
        size="sm"
        class="text-muted-foreground size-6 p-0"
        aria-label="Close workspace browser"
        onclick={onClose}
      >
        <X class="size-3" />
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
    <div class={cn('flex min-h-0 flex-1', sidebarResizing && 'select-none')}>
      <!-- Sidebar (resizable) -->
      <div
        class="relative min-h-0 shrink-0 overflow-hidden"
        style="width: {sidebarWidth}px"
        data-testid="workspace-browser-sidebar-panel"
      >
        <ProjectConversationWorkspaceBrowserSidebar
          repos={browser.metadata?.repos ?? []}
          selectedRepoPath={browser.selectedRepoPath}
          {selectedRepo}
          {selectedRepoDiff}
          treeNodes={browser.treeNodes}
          expandedDirs={browser.expandedDirs}
          loadingDirs={browser.loadingDirs}
          selectedFilePath={browser.selectedFilePath}
          onOpenRepo={browser.openRepo}
          onToggleDir={browser.toggleDir}
          onSelectFile={browser.selectFile}
        />
        <!-- Resize handle -->
        <div
          class={cn(
            'absolute inset-y-0 right-0 z-10 w-1 cursor-col-resize transition-colors',
            sidebarResizing ? 'bg-primary' : 'bg-border hover:bg-primary/50',
          )}
          role="separator"
          aria-orientation="vertical"
          onpointerdown={handleSidebarResizeStart}
        ></div>
      </div>
      <!-- Detail -->
      <div
        class="min-h-0 min-w-0 flex-1 overflow-hidden"
        data-testid="workspace-browser-detail-panel"
      >
        <ProjectConversationWorkspaceBrowserDetail
          {selectedRepo}
          selectedFilePath={browser.selectedFilePath}
          preview={browser.preview}
          patch={browser.patch}
          fileLoading={browser.fileLoading}
          fileError={browser.fileError}
        />
      </div>
    </div>
  {/if}
</div>
