<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronRight, Folder, FolderOpen, GitBranch, Loader2 } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceDiffRepo,
    ProjectConversationWorkspaceFileStatus,
    ProjectConversationWorkspaceRepoMetadata,
    ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'
  import {
    fileIcon,
    formatTotals,
    statusClass,
    statusLabel,
  } from './project-conversation-workspace-browser-helpers'

  let {
    repos = [],
    selectedRepoPath = '',
    selectedRepo = null,
    selectedRepoDiff = null,
    treeNodes = new Map(),
    expandedDirs = new Set(),
    loadingDirs = new Set(),
    selectedFilePath = '',
    onOpenRepo,
    onToggleDir,
    onSelectFile,
  }: {
    repos?: ProjectConversationWorkspaceRepoMetadata[]
    selectedRepoPath?: string
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    treeNodes?: Map<string, ProjectConversationWorkspaceTreeEntry[]>
    expandedDirs?: Set<string>
    loadingDirs?: Set<string>
    selectedFilePath?: string
    onOpenRepo?: (repoPath: string) => void
    onToggleDir?: (path: string) => void
    onSelectFile?: (path: string) => void
  } = $props()

  const rootEntries = $derived(treeNodes.get('') ?? [])
  const dirtyFiles = $derived(selectedRepoDiff?.files ?? [])

  /** Map of dirty file path → status */
  const dirtyFileStatus = $derived(
    new Map<string, ProjectConversationWorkspaceFileStatus>(
      dirtyFiles.map((f) => [f.path, f.status]),
    ),
  )

  /** Set of all parent directory paths that contain dirty files */
  const dirtyParentDirs = $derived(() => {
    const dirs = new Set<string>()
    for (const file of dirtyFiles) {
      const parts = file.path.split('/')
      for (let i = 1; i < parts.length; i++) {
        dirs.add(parts.slice(0, i).join('/'))
      }
    }
    return dirs
  })

  /** Color class for a dirty file based on its git status */
  function dirtyFileColorClass(status: ProjectConversationWorkspaceFileStatus): string {
    switch (status) {
      case 'added':
      case 'untracked':
        return 'text-emerald-600 dark:text-emerald-400'
      case 'deleted':
        return 'text-rose-600 dark:text-rose-400'
      default:
        return 'text-amber-600 dark:text-amber-400'
    }
  }

  let explorerExpanded = $state(true)
  let changesExpanded = $state(true)

  function filenameFromPath(path: string): string {
    return path.split('/').pop() ?? ''
  }
</script>

{#snippet treeLevel(entries: ProjectConversationWorkspaceTreeEntry[], depth: number)}
  {#each entries as entry (entry.path)}
    {#if entry.kind === 'directory'}
      {@const isExpanded = expandedDirs.has(entry.path)}
      {@const isLoading = loadingDirs.has(entry.path)}
      {@const isDirtyDir = dirtyParentDirs().has(entry.path)}
      <button
        type="button"
        class="hover:bg-muted/50 flex w-full items-center gap-1 py-[3px] text-left text-[13px] transition-colors"
        style="padding-left: {depth * 16 + 8}px"
        onclick={() => onToggleDir?.(entry.path)}
      >
        <ChevronRight
          class={cn(
            'text-muted-foreground size-3 shrink-0 transition-transform duration-100',
            isExpanded && 'rotate-90',
          )}
        />
        {#if isExpanded}
          <FolderOpen
            class={cn(
              'size-3.5 shrink-0',
              isDirtyDir ? 'text-amber-600 dark:text-amber-400' : 'text-muted-foreground',
            )}
          />
        {:else}
          <Folder
            class={cn(
              'size-3.5 shrink-0',
              isDirtyDir ? 'text-amber-600 dark:text-amber-400' : 'text-muted-foreground',
            )}
          />
        {/if}
        <span
          class={cn('min-w-0 flex-1 truncate', isDirtyDir && 'text-amber-600 dark:text-amber-400')}
          >{entry.name}</span
        >
        {#if isLoading}
          <Loader2 class="text-muted-foreground size-3 shrink-0 animate-spin" />
        {/if}
      </button>
      {#if isExpanded}
        {@const children = treeNodes.get(entry.path) ?? []}
        {#if children.length > 0}
          {@render treeLevel(children, depth + 1)}
        {:else if isLoading}
          <div
            class="text-muted-foreground/60 py-1 text-[11px]"
            style="padding-left: {(depth + 1) * 16 + 28}px"
          >
            Loading…
          </div>
        {/if}
      {/if}
    {:else}
      {@const fileStatus = dirtyFileStatus.get(entry.path)}
      <button
        type="button"
        class={cn(
          'hover:bg-muted/50 flex w-full items-center gap-1 py-[3px] text-left text-[13px] transition-colors',
          entry.path === selectedFilePath && 'bg-primary/10 text-primary',
        )}
        style="padding-left: {depth * 16 + 24}px"
        onclick={() => onSelectFile?.(entry.path)}
      >
        {#each [fileIcon(entry.name)] as fi}
          <fi.icon class={cn('size-3.5 shrink-0', fi.colorClass)} />
        {/each}
        <span class={cn('min-w-0 flex-1 truncate', fileStatus && dirtyFileColorClass(fileStatus))}>
          {entry.name}
        </span>
        {#if fileStatus}
          <span
            class={cn(
              'w-3.5 shrink-0 text-right font-mono text-[10px] font-bold',
              dirtyFileColorClass(fileStatus),
            )}
          >
            {statusLabel(fileStatus)}
          </span>
        {/if}
      </button>
    {/if}
  {/each}
{/snippet}

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

  <!-- Explorer panel — fills remaining space, scrolls independently -->
  <div class="flex min-h-0 flex-1 flex-col" data-testid="workspace-browser-explorer-panel">
    <button
      type="button"
      class="text-muted-foreground hover:bg-muted/30 flex shrink-0 items-center gap-1 px-2 py-1 text-[10px] font-semibold tracking-wider uppercase transition-colors"
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
    {#if explorerExpanded}
      <div
        class="min-h-0 flex-1 overflow-y-auto pb-1"
        data-testid="workspace-browser-explorer-list"
      >
        {#if rootEntries.length > 0}
          {@render treeLevel(rootEntries, 0)}
        {:else if loadingDirs.has('')}
          <div class="text-muted-foreground/60 px-4 py-2 text-[11px]">Loading files…</div>
        {:else}
          <div class="text-muted-foreground/60 px-4 py-2 text-[11px]">Empty directory</div>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Changes panel — pinned to bottom, max 30% height, scrolls independently -->
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
            >
              {#each [fileIcon(filenameFromPath(file.path))] as fi}
                <fi.icon class={cn('size-3.5 shrink-0', fi.colorClass)} />
              {/each}
              <span class="min-w-0 truncate">{file.path.split('/').pop()}</span>
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

  <!-- Branch status bar (bottom) -->
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
