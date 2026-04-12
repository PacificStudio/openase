<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import { ChevronRight, FilePlus2, FolderPlus, GitBranch } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceDiffRepo,
    ProjectConversationWorkspaceRepoMetadata,
    ProjectConversationWorkspaceSearchResult,
    ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'
  import {
    buildDirtyFileStatusMap,
    buildDirtyParentDirs,
    buildTreeMenuItems,
    dirtyFileColorClass,
    filenameFromPath,
    parentOf,
  } from './project-conversation-workspace-browser-sidebar-helpers'
  import {
    fileIcon,
    formatTotals,
    statusClass,
    statusLabel,
  } from './project-conversation-workspace-browser-helpers'
  import WorkspaceBrowserSearch from './project-conversation-workspace-browser-search.svelte'
  import WorkspaceBrowserTree, {
    type PendingCreate,
    type TreeMenuTarget,
  } from './project-conversation-workspace-browser-tree.svelte'
  import WorkspaceBrowserTreeMenu from './project-conversation-workspace-browser-tree-menu.svelte'

  let {
    repos = [],
    selectedRepoPath = '',
    selectedRepo = null,
    selectedRepoDiff = null,
    treeNodes = new Map(),
    expandedDirs = new Set(),
    loadingDirs = new Set(),
    selectedFilePath = '',
    recentFiles = [],
    onSearchPaths,
    onOpenRepo,
    onToggleDir,
    onSelectFile,
    onCreateEntry,
    onRenameEntry,
    onDeleteEntry,
    onCopyAbsolutePath,
    onCopyRelativePath,
  }: {
    repos?: ProjectConversationWorkspaceRepoMetadata[]
    selectedRepoPath?: string
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    treeNodes?: Map<string, ProjectConversationWorkspaceTreeEntry[]>
    expandedDirs?: Set<string>
    loadingDirs?: Set<string>
    selectedFilePath?: string
    recentFiles?: Array<{ repoPath: string; filePath: string }>
    onSearchPaths?: (
      query: string,
      limit?: number,
    ) => Promise<ProjectConversationWorkspaceSearchResult[]>
    onOpenRepo?: (repoPath: string) => void
    onToggleDir?: (path: string) => void
    onSelectFile?: (path: string) => void
    onCreateEntry?: (parentPath: string, name: string, kind: 'file' | 'folder') => void
    onRenameEntry?: (path: string, newName: string) => void
    onDeleteEntry?: (path: string) => void
    onCopyAbsolutePath?: (path: string) => void
    onCopyRelativePath?: (path: string) => void
  } = $props()

  const dirtyFiles = $derived(selectedRepoDiff?.files ?? [])
  const dirtyFileStatus = $derived(buildDirtyFileStatusMap(dirtyFiles))
  const dirtyParentDirs = $derived.by(() => buildDirtyParentDirs(dirtyFiles))

  let explorerExpanded = $state(true)
  let changesExpanded = $state(true)

  let pendingCreate = $state<PendingCreate | null>(null)
  let renameTarget = $state<{ path: string } | null>(null)
  let contextMenu = $state<{ x: number; y: number; entry: TreeMenuTarget } | null>(null)

  function ensureExpanded(path: string) {
    if (!path) return
    if (!expandedDirs.has(path)) {
      onToggleDir?.(path)
    }
  }

  function startCreate(kind: 'file' | 'folder', parentPath?: string) {
    const target = parentPath ?? (selectedFilePath ? parentOf(selectedFilePath) : '')
    renameTarget = null
    pendingCreate = { parentPath: target, kind }
    ensureExpanded(target)
    explorerExpanded = true
  }

  function startRename(path: string) {
    pendingCreate = null
    renameTarget = { path }
  }

  function commitCreate(rawName: string) {
    const pending = pendingCreate
    pendingCreate = null
    const name = rawName.trim()
    if (!pending || !name) return
    onCreateEntry?.(pending.parentPath, name, pending.kind)
  }

  function commitRename(rawName: string) {
    const target = renameTarget
    renameTarget = null
    const name = rawName.trim()
    if (!target || !name) return
    if (name === filenameFromPath(target.path)) return
    onRenameEntry?.(target.path, name)
  }

  function openMenu(event: MouseEvent, entry: TreeMenuTarget) {
    event.preventDefault()
    event.stopPropagation()
    contextMenu = { x: event.clientX, y: event.clientY, entry }
  }
</script>

<div class="border-border flex h-full min-h-0 flex-col overflow-hidden border-r">
  {#if repos.length > 1}
    <div class="border-border flex gap-1 border-b px-2 py-1.5">
      {#each repos as repo (repo.path)}
        <button
          type="button"
          class={cn(
            'hover:bg-muted/40 truncate rounded px-2 py-0.5 text-[11px] font-medium transition-colors',
            repo.path === selectedRepoPath ? 'bg-primary/10 text-primary' : 'text-muted-foreground',
          )}
          onclick={() => onOpenRepo?.(repo.path)}
        >
          {repo.name}
        </button>
      {/each}
    </div>
  {/if}

  <WorkspaceBrowserSearch
    {selectedRepoPath}
    {recentFiles}
    {treeNodes}
    {onSearchPaths}
    onSelectFile={(path) => onSelectFile?.(path)}
  />

  <div class="flex min-h-0 flex-1 flex-col" data-testid="workspace-browser-explorer-panel">
    <div class="flex shrink-0 items-center gap-0.5 pr-1">
      <button
        type="button"
        class="text-muted-foreground hover:bg-muted/30 flex flex-1 items-center gap-1 px-2 py-1 text-[10px] font-semibold tracking-wider uppercase transition-colors"
        onclick={() => (explorerExpanded = !explorerExpanded)}
      >
        <ChevronRight
          class={cn(
            'size-2.5 shrink-0 transition-transform duration-100',
            explorerExpanded && 'rotate-90',
          )}
        />
        Explorer
      </button>
      <Button
        size="icon-xs"
        variant="ghost"
        class="size-5"
        title="New File"
        data-testid="workspace-browser-new-file"
        onclick={() => startCreate('file')}
      >
        <FilePlus2 class="size-3" />
      </Button>
      <Button
        size="icon-xs"
        variant="ghost"
        class="size-5"
        title="New Folder"
        data-testid="workspace-browser-new-folder"
        onclick={() => startCreate('folder')}
      >
        <FolderPlus class="size-3" />
      </Button>
    </div>
    {#if explorerExpanded}
      <div
        class="min-h-0 flex-1 overflow-y-auto pb-1"
        data-testid="workspace-browser-explorer-list"
      >
        <WorkspaceBrowserTree
          {treeNodes}
          {expandedDirs}
          {loadingDirs}
          {selectedFilePath}
          {dirtyFileStatus}
          {dirtyParentDirs}
          {pendingCreate}
          {renameTarget}
          {onToggleDir}
          onSelectFile={(path) => onSelectFile?.(path)}
          onOpenMenu={openMenu}
          onCommitCreate={commitCreate}
          onCancelCreate={() => (pendingCreate = null)}
          onCommitRename={commitRename}
          onCancelRename={() => (renameTarget = null)}
        />
      </div>
    {/if}
  </div>

  {#if dirtyFiles.length > 0}
    <div
      class="border-border flex max-h-[30%] min-h-0 shrink-0 flex-col border-t"
      data-testid="workspace-browser-changes-panel"
    >
      <button
        type="button"
        class="text-muted-foreground hover:bg-muted/30 flex shrink-0 items-center gap-1 px-2 py-1 text-[10px] font-semibold tracking-wider uppercase transition-colors"
        onclick={() => (changesExpanded = !changesExpanded)}
      >
        <ChevronRight
          class={cn(
            'size-2.5 shrink-0 transition-transform duration-100',
            changesExpanded && 'rotate-90',
          )}
        />
        Changes
        <span class="bg-primary/15 text-primary ml-auto rounded-full px-1.5 text-[9px] font-bold">
          {dirtyFiles.length}
        </span>
      </button>
      {#if changesExpanded}
        <div
          class="min-h-0 flex-1 overflow-y-auto pb-1"
          data-testid="workspace-browser-changes-list"
        >
          {#each dirtyFiles as file (file.path)}
            <button
              type="button"
              class={cn(
                'hover:bg-muted/50 flex w-full items-center gap-1.5 py-[3px] pr-2 pl-4 text-left text-[13px] transition-colors',
                file.path === selectedFilePath && 'bg-primary/10 text-primary',
              )}
              onclick={() => onSelectFile?.(file.path)}
              oncontextmenu={(event) =>
                openMenu(event, {
                  kind: 'file',
                  path: file.path,
                  name: filenameFromPath(file.path),
                })}
            >
              {#each [fileIcon(filenameFromPath(file.path))] as fi}
                <fi.icon class={cn('size-3.5 shrink-0', fi.colorClass)} />
              {/each}
              <span class={cn('min-w-0 truncate', dirtyFileColorClass(file.status))}
                >{filenameFromPath(file.path)}</span
              >
              {#if file.path.includes('/')}
                <span class="text-muted-foreground/40 min-w-0 shrink truncate text-[10px]">
                  {file.path.slice(0, file.path.lastIndexOf('/'))}
                </span>
              {/if}
              <span class="flex-1"></span>
              <span class="text-muted-foreground/60 mr-1 hidden shrink-0 text-[10px] sm:inline">
                {formatTotals(file.added, file.removed)}
              </span>
              <span
                class={cn(
                  'w-3.5 shrink-0 text-center font-mono text-[10px] font-bold',
                  statusClass(file.status),
                )}
              >
                {statusLabel(file.status)}
              </span>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  {/if}

  {#if selectedRepo}
    <div class="border-border bg-muted/30 flex items-center gap-1.5 border-t px-3 py-1 text-[11px]">
      <GitBranch class="text-muted-foreground size-3 shrink-0" />
      <span class="font-medium">{selectedRepo.branch}</span>
      <span class="text-muted-foreground/60 min-w-0 truncate font-mono text-[10px]">
        {selectedRepo.headCommit}
      </span>
    </div>
  {/if}
</div>

{#if contextMenu}
  <WorkspaceBrowserTreeMenu
    x={contextMenu.x}
    y={contextMenu.y}
    items={buildTreeMenuItems(contextMenu.entry, {
      onStartCreate: startCreate,
      onStartRename: startRename,
      onDeleteEntry,
      onCopyAbsolutePath,
      onCopyRelativePath,
    })}
    onClose={() => (contextMenu = null)}
  />
{/if}
