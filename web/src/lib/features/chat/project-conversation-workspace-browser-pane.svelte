<script lang="ts">
  import { cn } from '$lib/utils'
  import type {
    ProjectConversationWorkspaceDiffRepo,
    ProjectConversationWorkspaceRepoMetadata,
  } from '$lib/api/chat'
  import ProjectConversationWorkspaceBrowserDetail from './project-conversation-workspace-browser-detail.svelte'
  import ProjectConversationWorkspaceBrowserSidebar from './project-conversation-workspace-browser-sidebar.svelte'
  import type { ProjectConversationWorkspaceBrowserState } from './project-conversation-workspace-browser-state.svelte'
  import { createTerminalManager } from './terminal-manager.svelte'
  import WorkspaceTerminalPanel from './workspace-terminal-panel.svelte'

  let {
    browser,
    selectedRepo = null,
    selectedRepoDiff = null,
    runtimeActive = false,
    terminalManager,
  }: {
    browser: ProjectConversationWorkspaceBrowserState
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    runtimeActive?: boolean
    terminalManager: ReturnType<typeof createTerminalManager>
  } = $props()

  const MIN_SIDEBAR_WIDTH = 180
  const MAX_SIDEBAR_WIDTH = 480
  let sidebarWidth = $state(240)
  let sidebarResizing = $state(false)

  const MIN_TERMINAL_HEIGHT = 120
  const DEFAULT_TERMINAL_HEIGHT = 260
  let terminalHeight = $state(DEFAULT_TERMINAL_HEIGHT)
  let terminalResizing = $state(false)
  let containerElement: HTMLDivElement | null = null

  function joinPath(parent: string, child: string): string {
    return parent ? `${parent}/${child}` : child
  }

  function parentDirOf(path: string): string {
    const idx = path.lastIndexOf('/')
    return idx === -1 ? '' : path.slice(0, idx)
  }

  async function handleCreateEntry(parentPath: string, name: string, kind: 'file' | 'folder') {
    // The backend createFile endpoint auto-creates any missing parent
    // directories. For folders we create a sentinel `.gitkeep` inside so
    // the directory exists even when empty.
    const leafPath =
      kind === 'folder'
        ? joinPath(joinPath(parentPath, name), '.gitkeep')
        : joinPath(parentPath, name)
    await browser.createFile(leafPath)
  }

  async function handleRenameEntry(fromPath: string, newName: string) {
    const parent = parentDirOf(fromPath)
    const toPath = joinPath(parent, newName)
    if (toPath === fromPath) {
      return
    }
    await browser.renameFile(fromPath, toPath)
  }

  async function handleDeleteEntry(path: string) {
    const editor = browser.selectedEditorState
    const confirmed = window.confirm(
      editor?.dirty && browser.selectedFilePath === path
        ? `Delete ${path}? Unsaved local draft changes will be discarded.`
        : `Delete ${path}?`,
    )
    if (!confirmed) {
      return
    }
    await browser.deleteFile(path)
  }

  async function copyToClipboard(value: string) {
    if (!value) return
    try {
      await navigator.clipboard.writeText(value)
    } catch {
      // Clipboard permission can be denied in some contexts; silently ignore.
    }
  }

  function buildAbsolutePath(filePath: string): string {
    const workspaceRoot = browser.metadata?.workspacePath ?? ''
    const repoPath = browser.selectedRepoPath
    return [workspaceRoot, repoPath, filePath].filter(Boolean).join('/')
  }

  function buildRelativePath(filePath: string): string {
    const repoPath = browser.selectedRepoPath
    return repoPath ? `${repoPath}/${filePath}` : filePath
  }

  function handleCopyAbsolutePath(filePath: string) {
    void copyToClipboard(buildAbsolutePath(filePath))
  }

  function handleCopyRelativePath(filePath: string) {
    void copyToClipboard(buildRelativePath(filePath))
  }

  function handleSidebarResizeStart(event: PointerEvent) {
    event.preventDefault()
    sidebarResizing = true
    const startX = event.clientX
    const startWidth = sidebarWidth

    function onMove(nextEvent: PointerEvent) {
      sidebarWidth = Math.min(
        MAX_SIDEBAR_WIDTH,
        Math.max(MIN_SIDEBAR_WIDTH, startWidth + (nextEvent.clientX - startX)),
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

  function handleTerminalResizeStart(event: PointerEvent) {
    event.preventDefault()
    terminalResizing = true
    const startY = event.clientY
    const startHeight = terminalHeight

    function onMove(nextEvent: PointerEvent) {
      const maxHeight = containerElement ? containerElement.clientHeight - 100 : 600
      terminalHeight = Math.min(
        maxHeight,
        Math.max(MIN_TERMINAL_HEIGHT, startHeight - (nextEvent.clientY - startY)),
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
</script>

<div
  class={cn('flex min-h-0 flex-1 flex-col', (sidebarResizing || terminalResizing) && 'select-none')}
  bind:this={containerElement}
>
  <div class="flex min-h-0 flex-1">
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
        recentFiles={browser.recentFiles}
        onSearchPaths={browser.searchPaths}
        onOpenRepo={browser.openRepo}
        onToggleDir={browser.toggleDir}
        onSelectFile={browser.selectFile}
        onCreateEntry={handleCreateEntry}
        onRenameEntry={handleRenameEntry}
        onDeleteEntry={handleDeleteEntry}
        onCopyAbsolutePath={handleCopyAbsolutePath}
        onCopyRelativePath={handleCopyRelativePath}
      />
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

    <div
      class="min-h-0 min-w-0 flex-1 overflow-hidden"
      data-testid="workspace-browser-detail-panel"
    >
      <ProjectConversationWorkspaceBrowserDetail {browser} {selectedRepo} {runtimeActive} />
    </div>
  </div>

  {#if terminalManager.panelOpen}
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
