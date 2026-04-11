<script lang="ts">
  /* eslint-disable max-lines */
  import { untrack } from 'svelte'
  import { Button } from '$ui/button'
  import {
    syncProjectConversationWorkspace,
    type ProjectConversationWorkspaceSyncPrompt,
  } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import {
    AlertCircle,
    Check,
    Copy,
    FolderTree,
    RefreshCcw,
    SquareTerminal,
    X,
  } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import WorkspaceTerminalPanel from './workspace-terminal-panel.svelte'
  import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'
  import ProjectConversationWorkspaceBrowserSidebar from './project-conversation-workspace-browser-sidebar.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
  import { createTerminalManager } from './terminal-manager.svelte'
  import { workspaceBrowserPortal } from './workspace-browser-portal.svelte'
  import { onDestroy } from 'svelte'

  let {
    conversationId = '',
    workspaceDiff = null,
    workspaceDiffLoading = false,
    syncGeneration = 0,
    pendingFilePath = '',
    onClose,
    onPendingFileConsumed,
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    workspaceDiffLoading?: boolean
    syncGeneration?: number
    /** File path to navigate to (consumed once on change). */
    pendingFilePath?: string
    onClose?: () => void
    onPendingFileConsumed?: () => void
  } = $props()

  const browser = createProjectConversationWorkspaceBrowserState({
    getConversationId: () => conversationId,
  })

  const terminalManager = createTerminalManager({
    getConversationId: () => conversationId,
    getWorkspacePath: () => browser.metadata?.workspacePath ?? '',
  })

  onDestroy(() => {
    terminalManager.disposeAll()
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

  // -- Terminal panel vertical resize --
  const MIN_TERMINAL_HEIGHT = 120
  const DEFAULT_TERMINAL_HEIGHT = 260
  let terminalHeight = $state(DEFAULT_TERMINAL_HEIGHT)
  let terminalResizing = $state(false)
  let containerElement: HTMLDivElement | null = null

  function handleTerminalResizeStart(event: PointerEvent) {
    event.preventDefault()
    terminalResizing = true
    const startY = event.clientY
    const startHeight = terminalHeight

    function onMove(e: PointerEvent) {
      const maxHeight = containerElement ? containerElement.clientHeight - 100 : 600
      terminalHeight = Math.min(
        maxHeight,
        Math.max(MIN_TERMINAL_HEIGHT, startHeight - (e.clientY - startY)),
      )
    }

    function onUp() {
      terminalResizing = false
      window.removeEventListener('pointermove', onMove)
      window.removeEventListener('pointerup', onUp)
      terminalManager.refitAll()
    }

    window.addEventListener('pointermove', onMove)
    window.addEventListener('pointerup', onUp)
  }

  let refreshGeneration = $state(0)
  let lastRefreshKey = $state('')
  let lastWorkspaceDiffLoading = $state(false)
  let lastConversationId = $state('')
  let lastSyncGeneration = $state(0)
  let syncInFlight = $state(false)
  let syncError = $state('')

  const selectedRepo = $derived(
    browser.metadata?.repos.find((repo) => repo.path === browser.selectedRepoPath) ??
      browser.metadata?.repos[0] ??
      null,
  )
  const selectedRepoDiff = $derived(
    workspaceDiff?.repos.find((repo) => repo.path === browser.selectedRepoPath) ?? null,
  )
  const syncPrompt = $derived(workspaceDiff?.syncPrompt ?? browser.metadata?.syncPrompt ?? null)

  function syncPromptTitle(prompt: ProjectConversationWorkspaceSyncPrompt | null) {
    if (!prompt) return ''
    return prompt.reason === 'repo_binding_changed'
      ? 'Workspace sync required'
      : 'Some project repos are missing from this workspace'
  }

  function syncPromptDescription(prompt: ProjectConversationWorkspaceSyncPrompt | null) {
    if (!prompt) return ''
    if (prompt.reason === 'repo_binding_changed') {
      return 'This conversation workspace was prepared before the latest project repo binding changes. Newly bound repos have not been cloned into this workspace yet, so browse and diff can be incomplete until you sync.'
    }
    return 'One or more repos are bound to this project but are still missing from the current conversation workspace. Sync the workspace to clone them before browsing or diffing.'
  }

  async function handleSyncWorkspace() {
    if (!conversationId || syncInFlight) {
      return
    }
    syncInFlight = true
    syncError = ''
    try {
      await syncProjectConversationWorkspace(conversationId)
      workspaceBrowserPortal.markWorkspaceSynced()
      await Promise.resolve(workspaceBrowserPortal.onSyncWorkspace?.())
      await browser.refreshWorkspace(true)
    } catch (error) {
      syncError =
        error instanceof Error ? error.message : 'Failed to sync the Project AI workspace.'
    } finally {
      syncInFlight = false
    }
  }

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
    if (lastConversationId && lastConversationId !== conversationId) {
      terminalManager.disposeAll()
    }
    lastConversationId = conversationId
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
      lastSyncGeneration = syncGeneration
      return
    }
    if (syncGeneration !== lastSyncGeneration) {
      lastSyncGeneration = syncGeneration
      untrack(() => {
        void browser.refreshWorkspace(true)
      })
    }
  })

  $effect(() => {
    if (!conversationId) {
      lastRefreshKey = ''
      browser.reset()
      terminalManager.disposeAll()
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
  bind:this={containerElement}
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
    {#if browser.metadata?.available}
      <Button
        variant={terminalManager.panelOpen ? 'secondary' : 'ghost'}
        size="icon-xs"
        class={cn('text-muted-foreground size-6', terminalManager.panelOpen && 'text-foreground')}
        aria-label="Toggle terminal"
        onclick={() => terminalManager.togglePanel()}
        disabled={!conversationId}
      >
        <SquareTerminal class="size-3" />
      </Button>
    {/if}
    <Button
      variant="ghost"
      size="icon-xs"
      class="text-muted-foreground size-6"
      aria-label="Refresh workspace browser"
      onclick={() => void browser.refreshWorkspace(true)}
      disabled={!conversationId || browser.metadataLoading}
    >
      <RefreshCcw class={cn('size-3', browser.metadataLoading && 'animate-spin')} />
    </Button>
    {#if onClose}
      <Button
        variant="ghost"
        size="icon-xs"
        class="text-muted-foreground size-6"
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
      Loading workspace...
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
  {:else if syncPrompt && (browser.metadata?.repos.length ?? 0) === 0}
    <div class="flex flex-1 items-center justify-center px-6">
      <div class="border-border bg-muted/20 max-w-lg rounded-xl border p-5 text-left">
        <p class="text-sm font-medium">{syncPromptTitle(syncPrompt)}</p>
        <p class="text-muted-foreground mt-2 text-sm">{syncPromptDescription(syncPrompt)}</p>
        <p class="text-muted-foreground mt-3 text-xs">
          Missing repos:
          {syncPrompt.missingRepos.map((repo) => repo.path).join(', ')}
        </p>
        {#if syncError}
          <p class="text-destructive mt-3 text-xs">{syncError}</p>
        {/if}
        <div class="mt-4 flex gap-2">
          <Button size="sm" onclick={() => void handleSyncWorkspace()} disabled={syncInFlight}>
            {syncInFlight ? 'Syncing repos...' : 'Sync repos'}
          </Button>
        </div>
      </div>
    </div>
  {:else}
    <div
      class={cn(
        'flex min-h-0 flex-1 flex-col',
        (sidebarResizing || terminalResizing) && 'select-none',
      )}
    >
      {#if syncPrompt}
        <div
          class="border-border border-b bg-amber-50/80 px-3 py-2 text-amber-950 dark:bg-amber-500/10 dark:text-amber-100"
        >
          <div class="flex items-start gap-3">
            <AlertCircle class="mt-0.5 size-4 shrink-0" />
            <div class="min-w-0 flex-1">
              <p class="text-sm font-medium">{syncPromptTitle(syncPrompt)}</p>
              <p class="mt-1 text-xs leading-5">{syncPromptDescription(syncPrompt)}</p>
              <p class="mt-2 text-xs">
                Missing repos: {syncPrompt.missingRepos.map((repo) => repo.path).join(', ')}
              </p>
              {#if syncError}
                <p class="text-destructive mt-2 text-xs">{syncError}</p>
              {/if}
            </div>
            <Button
              size="sm"
              variant="secondary"
              class="shrink-0"
              onclick={() => void handleSyncWorkspace()}
              disabled={syncInFlight}
            >
              {syncInFlight ? 'Syncing...' : 'Sync repos'}
            </Button>
          </div>
        </div>
      {/if}
      <!-- Files area -->
      <div class="flex min-h-0 flex-1">
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

      <!-- Terminal panel (bottom, like VSCode) -->
      {#if terminalManager.panelOpen}
        <!-- Resize handle -->
        <div
          class={cn(
            'h-[3px] shrink-0 cursor-row-resize transition-colors',
            terminalResizing ? 'bg-primary' : 'bg-border hover:bg-primary/50',
          )}
          role="separator"
          aria-orientation="horizontal"
          onpointerdown={handleTerminalResizeStart}
        ></div>
        <div class="flex min-h-0 shrink-0 overflow-hidden" style="height: {terminalHeight}px">
          <WorkspaceTerminalPanel manager={terminalManager} />
        </div>
      {/if}
    </div>
  {/if}
</div>
