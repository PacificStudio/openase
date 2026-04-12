<script lang="ts" module>
  export type TreeMenuTarget = { kind: 'file' | 'directory'; path: string; name: string }
  export type PendingCreate = { parentPath: string; kind: 'file' | 'folder' }
</script>

<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronRight, Folder, FolderOpen, Loader2 } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceFileStatus,
    ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'
  import { fileIcon, statusLabel } from './project-conversation-workspace-browser-helpers'

  type TreeEntry = ProjectConversationWorkspaceTreeEntry

  let {
    treeNodes = new Map(),
    expandedDirs = new Set(),
    loadingDirs = new Set(),
    selectedFilePath = '',
    dirtyFileStatus = new Map(),
    dirtyParentDirs = new Set(),
    pendingCreate = null,
    renameTarget = null,
    onToggleDir,
    onSelectFile,
    onOpenMenu,
    onCommitCreate,
    onCancelCreate,
    onCommitRename,
    onCancelRename,
  }: {
    treeNodes?: Map<string, TreeEntry[]>
    expandedDirs?: Set<string>
    loadingDirs?: Set<string>
    selectedFilePath?: string
    dirtyFileStatus?: Map<string, ProjectConversationWorkspaceFileStatus>
    dirtyParentDirs?: Set<string>
    pendingCreate?: PendingCreate | null
    renameTarget?: { path: string } | null
    onToggleDir?: (path: string) => void
    onSelectFile?: (path: string) => void
    onOpenMenu?: (event: MouseEvent, entry: TreeMenuTarget) => void
    onCommitCreate?: (name: string) => void
    onCancelCreate?: () => void
    onCommitRename?: (name: string) => void
    onCancelRename?: () => void
  } = $props()

  const rootEntries = $derived(treeNodes.get('') ?? [])

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

  function autofocusInput(node: HTMLInputElement, initial: string) {
    node.value = initial
    node.focus()
    if (initial) {
      const dot = initial.lastIndexOf('.')
      node.setSelectionRange(0, dot > 0 ? dot : initial.length)
    }
  }

  function onInlineKey(event: KeyboardEvent, commit: (value: string) => void, cancel: () => void) {
    if (event.key === 'Enter') {
      event.preventDefault()
      commit((event.currentTarget as HTMLInputElement).value)
    } else if (event.key === 'Escape') {
      event.preventDefault()
      cancel()
    }
  }
</script>

{#snippet inlineInput(
  depth: number,
  isFolder: boolean,
  initial: string,
  commit: (name: string) => void,
  cancel: () => void,
)}
  <div
    class="flex items-center gap-1 py-[3px] pr-3 text-[13px]"
    style="padding-left: {depth * 16 + 8}px"
  >
    {#if isFolder}
      <Folder class="text-muted-foreground size-3.5 shrink-0" />
    {:else}
      <span class="size-3.5 shrink-0"></span>
    {/if}
    <input
      type="text"
      class="border-primary/60 bg-background min-w-0 flex-1 rounded border px-1 py-[1px] text-[12px] outline-none"
      use:autofocusInput={initial}
      onkeydown={(event) => onInlineKey(event, commit, cancel)}
      onblur={(event) => commit((event.currentTarget as HTMLInputElement).value)}
      data-testid="workspace-browser-inline-input"
    />
  </div>
{/snippet}

{#snippet treeLevel(entries: TreeEntry[], depth: number)}
  {#each entries as entry (entry.path)}
    {#if renameTarget?.path === entry.path}
      {@render inlineInput(
        depth,
        entry.kind === 'directory',
        entry.name,
        (name) => onCommitRename?.(name),
        () => onCancelRename?.(),
      )}
    {:else if entry.kind === 'directory'}
      {@const isExpanded = expandedDirs.has(entry.path)}
      {@const isLoading = loadingDirs.has(entry.path)}
      {@const isDirtyDir = dirtyParentDirs.has(entry.path)}
      <button
        type="button"
        class="hover:bg-muted/50 flex w-full items-center gap-1 py-[3px] pr-3 text-left text-[13px] transition-colors"
        style="padding-left: {depth * 16 + 8}px"
        onclick={() => onToggleDir?.(entry.path)}
        oncontextmenu={(event) =>
          onOpenMenu?.(event, { kind: 'directory', path: entry.path, name: entry.name })}
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
        {#if pendingCreate?.parentPath === entry.path}
          {@render inlineInput(
            depth + 1,
            pendingCreate.kind === 'folder',
            '',
            (name) => onCommitCreate?.(name),
            () => onCancelCreate?.(),
          )}
        {/if}
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
          'hover:bg-muted/50 flex w-full items-center gap-1 py-[3px] pr-3 text-left text-[13px] transition-colors',
          entry.path === selectedFilePath && 'bg-primary/10 text-primary',
        )}
        style="padding-left: {depth * 16 + 24}px"
        onclick={() => onSelectFile?.(entry.path)}
        oncontextmenu={(event) =>
          onOpenMenu?.(event, { kind: 'file', path: entry.path, name: entry.name })}
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

{#if pendingCreate?.parentPath === ''}
  {@render inlineInput(
    0,
    pendingCreate.kind === 'folder',
    '',
    (name) => onCommitCreate?.(name),
    () => onCancelCreate?.(),
  )}
{/if}
{#if rootEntries.length > 0}
  {@render treeLevel(rootEntries, 0)}
{:else if loadingDirs.has('')}
  <div class="text-muted-foreground/60 px-4 py-2 text-[11px]">Loading files…</div>
{:else if !pendingCreate}
  <div class="text-muted-foreground/60 px-4 py-2 text-[11px]">Empty directory</div>
{/if}
