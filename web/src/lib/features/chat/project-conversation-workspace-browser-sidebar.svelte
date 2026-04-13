<script lang="ts">
  import { cn } from '$lib/utils'
  import { Button } from '$ui/button'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import {
    ChevronRight,
    CloudDownload,
    CloudUpload,
    Ellipsis,
    FilePlus2,
    FolderPlus,
    GitGraph,
    LoaderCircle,
    RefreshCcw,
  } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceBranchRef,
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceCurrentRef,
    ProjectConversationWorkspaceDiffRepo,
    ProjectConversationWorkspaceGitGraph,
    ProjectConversationWorkspaceGitRemoteOp,
    ProjectConversationWorkspaceRepoMetadata,
    ProjectConversationWorkspaceSearchResult,
    ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'
  import WorkspaceBrowserBranchPicker from './project-conversation-workspace-browser-branch-picker.svelte'
  import WorkspaceBrowserSidebarGitGraph from './project-conversation-workspace-browser-sidebar-git-graph.svelte'
  import {
    buildDirtyFileStatusMap,
    buildDirtyParentDirs,
    buildTreeMenuItems,
    filenameFromPath,
    parentOf,
  } from './project-conversation-workspace-browser-sidebar-helpers'
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
    currentRef = null,
    localBranches = [],
    remoteBranches = [],
    repoRefsLoading = false,
    repoRefsError = '',
    checkoutBlockers = [],
    gitGraph = null,
    gitGraphLoading = false,
    gitGraphError = '',
    onCheckoutBranch,
    onCreateBranchName,
    onGraphCheckoutBranch,
    onGitRemoteOp,
    onStageFile,
    onStageAll,
    onUnstage,
    onCommitRepo,
    onDiscardFile,
    onCreateBranch,
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
    currentRef?: ProjectConversationWorkspaceCurrentRef | null
    localBranches?: ProjectConversationWorkspaceBranchRef[]
    remoteBranches?: ProjectConversationWorkspaceBranchRef[]
    repoRefsLoading?: boolean
    repoRefsError?: string
    checkoutBlockers?: string[]
    gitGraph?: ProjectConversationWorkspaceGitGraph | null
    gitGraphLoading?: boolean
    gitGraphError?: string
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
    onCreateBranchName?: (branchName: string) => Promise<void>
    onGraphCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => void
    onGitRemoteOp?: (op: ProjectConversationWorkspaceGitRemoteOp) => Promise<void>
    onStageFile?: (path: string) => Promise<void>
    onStageAll?: () => Promise<void>
    onUnstage?: (path?: string) => Promise<void>
    onCommitRepo?: (message: string) => Promise<void>
    onDiscardFile?: (path: string) => Promise<void>
    onCreateBranch?: (commitId: string) => void
    onDeleteEntry?: (path: string) => void
    onCopyAbsolutePath?: (path: string) => void
    onCopyRelativePath?: (path: string) => void
  } = $props()

  const dirtyFiles = $derived(selectedRepoDiff?.files ?? [])
  const dirtyFileStatus = $derived(buildDirtyFileStatusMap(dirtyFiles))
  const dirtyParentDirs = $derived.by(() => buildDirtyParentDirs(dirtyFiles))

  let explorerExpanded = $state(true)
  let gitGraphExpanded = $state(false)
  let gitRemoteOpLoading = $state(false)

  async function handleGitRemoteOp(op: ProjectConversationWorkspaceGitRemoteOp) {
    if (gitRemoteOpLoading) return
    gitRemoteOpLoading = true
    try {
      await onGitRemoteOp?.(op)
    } finally {
      gitRemoteOpLoading = false
    }
  }

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

  {#if gitGraph || gitGraphLoading}
    <div
      class="border-border flex max-h-[40%] min-h-0 shrink-0 flex-col border-t"
      data-testid="workspace-browser-git-graph-panel"
    >
      <div class="flex shrink-0 items-center gap-0.5 pr-1">
        <button
          type="button"
          class="text-muted-foreground hover:bg-muted/30 flex flex-1 items-center gap-1 px-2 py-1 text-[10px] font-semibold tracking-wider uppercase transition-colors"
          onclick={() => (gitGraphExpanded = !gitGraphExpanded)}
        >
          <ChevronRight
            class={cn(
              'size-2.5 shrink-0 transition-transform duration-100',
              gitGraphExpanded && 'rotate-90',
            )}
          />
          <GitGraph class="size-2.5 shrink-0" />
          Git Graph
          {#if gitGraph && gitGraph.commits.length > 0}
            <span
              class="text-muted-foreground/60 ml-auto text-[9px] font-normal tracking-normal normal-case"
            >
              {gitGraph.commits.length}
            </span>
          {/if}
        </button>
        {#if gitRemoteOpLoading}
          <div class="flex size-5 items-center justify-center">
            <LoaderCircle class="text-muted-foreground size-3 animate-spin" />
          </div>
        {:else}
          <DropdownMenu.Root>
            <DropdownMenu.Trigger
              class="text-muted-foreground hover:bg-muted/40 flex size-5 items-center justify-center rounded transition-colors"
              onclick={(e: MouseEvent) => e.stopPropagation()}
            >
              <Ellipsis class="size-3" />
            </DropdownMenu.Trigger>
            <DropdownMenu.Content align="end" class="w-40">
              <DropdownMenu.Item onclick={() => handleGitRemoteOp('fetch')}>
                <RefreshCcw class="size-3.5" />
                <span>Fetch</span>
              </DropdownMenu.Item>
              <DropdownMenu.Item onclick={() => handleGitRemoteOp('pull')}>
                <CloudDownload class="size-3.5" />
                <span>Pull</span>
              </DropdownMenu.Item>
              <DropdownMenu.Item onclick={() => handleGitRemoteOp('push')}>
                <CloudUpload class="size-3.5" />
                <span>Push</span>
              </DropdownMenu.Item>
            </DropdownMenu.Content>
          </DropdownMenu.Root>
        {/if}
      </div>
      {#if gitGraphExpanded}
        <div class="min-h-0 flex-1 overflow-y-auto pb-1">
          <WorkspaceBrowserSidebarGitGraph
            {gitGraph}
            loading={gitGraphLoading}
            error={gitGraphError}
            onCheckoutBranch={onGraphCheckoutBranch}
            {onCreateBranch}
          />
        </div>
      {/if}
    </div>
  {/if}

  <WorkspaceBrowserBranchPicker
    {currentRef}
    {localBranches}
    {remoteBranches}
    {repoRefsLoading}
    {repoRefsError}
    {checkoutBlockers}
    {selectedRepo}
    {selectedRepoDiff}
    {selectedFilePath}
    {onCheckoutBranch}
    {onCreateBranchName}
    {onStageFile}
    {onStageAll}
    {onUnstage}
    {onCommitRepo}
    {onDiscardFile}
    onSelectFile={(path) => onSelectFile?.(path)}
  />
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
