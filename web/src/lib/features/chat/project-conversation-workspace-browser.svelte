<script lang="ts">
  import { onDestroy, untrack } from 'svelte'
  import { syncProjectConversationWorkspace } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import { AlertCircle } from '@lucide/svelte'
  import type { ProjectConversationWorkspaceDiff } from '$lib/api/chat'
  import WorkspaceTerminalPanel from './workspace-terminal-panel.svelte'
  import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'
  import ProjectConversationWorkspaceBrowserSidebar from './project-conversation-workspace-browser-sidebar.svelte'
  import ProjectConversationWorkspaceSyncBanner from './project-conversation-workspace-sync-banner.svelte'
  import ProjectConversationWorkspaceBrowserToolbar from './project-conversation-workspace-browser-toolbar.svelte'
  import { createProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
  import { createTerminalManager } from './terminal-manager.svelte'
  import { workspaceBrowserPortal } from './workspace-browser-portal.svelte'

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
  <ProjectConversationWorkspaceBrowserToolbar
    workspacePath={browser.metadata?.workspacePath ?? ''}
    {pathCopied}
    showTerminalButton={Boolean(browser.metadata?.available)}
    terminalPanelOpen={terminalManager.panelOpen}
    {conversationId}
    metadataLoading={browser.metadataLoading}
    onCopyWorkspacePath={copyWorkspacePath}
    onToggleTerminal={() => terminalManager.togglePanel()}
    onRefreshWorkspace={() => void browser.refreshWorkspace(true)}
    {onClose}
  />

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
    <ProjectConversationWorkspaceSyncBanner
      prompt={syncPrompt}
      {syncError}
      {syncInFlight}
      centered
      onSync={handleSyncWorkspace}
    />
  {:else}
    <div
      class={cn(
        'flex min-h-0 flex-1 flex-col',
        (sidebarResizing || terminalResizing) && 'select-none',
      )}
    >
      {#if syncPrompt}
        <ProjectConversationWorkspaceSyncBanner
          prompt={syncPrompt}
          {syncError}
          {syncInFlight}
          onSync={handleSyncWorkspace}
        />
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
