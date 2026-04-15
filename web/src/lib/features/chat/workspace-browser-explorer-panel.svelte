<script lang="ts">
  import type {
  ProjectConversationWorkspaceFileStatus,
  ProjectConversationWorkspaceTreeEntry,
} from '$lib/api/chat'
  import { cn } from '$lib/utils'
import { chatT } from './i18n'
  import { Button } from '$ui/button'
  import { ChevronRight, FilePlus2, FolderPlus } from '@lucide/svelte'
  import WorkspaceBrowserTree, {
    type PendingCreate,
    type TreeMenuTarget,
  } from './project-conversation-workspace-browser-tree.svelte'

  let {
    treeNodes,
    expandedDirs,
    loadingDirs,
    selectedFilePath = '',
    dirtyFileStatus,
    dirtyParentDirs,
    pendingCreate = null,
    renameTarget = null,
    onToggleDir,
    onSelectFile,
    onOpenMenu,
    onStartCreate,
    onCommitCreate,
    onCancelCreate,
    onCommitRename,
    onCancelRename,
  }: {
    treeNodes: Map<string, ProjectConversationWorkspaceTreeEntry[]>
    expandedDirs: Set<string>
    loadingDirs: Set<string>
    selectedFilePath?: string
    dirtyFileStatus: Map<string, ProjectConversationWorkspaceFileStatus>
    dirtyParentDirs: Set<string>
    pendingCreate?: PendingCreate | null
    renameTarget?: { path: string } | null
    onToggleDir?: (path: string) => void
    onSelectFile?: (path: string) => void
    onOpenMenu?: (event: MouseEvent, entry: TreeMenuTarget) => void
    onStartCreate?: (kind: 'file' | 'folder') => void
    onCommitCreate?: (name: string) => void
    onCancelCreate?: () => void
    onCommitRename?: (name: string) => void
    onCancelRename?: () => void
  } = $props()

  let explorerExpanded = $state(true)
</script>

<div class="flex min-h-0 flex-1 flex-col" data-testid="workspace-browser-explorer-panel">
  <div class="flex shrink-0 items-center gap-0.5 pr-1">
    <button
      type="button"
      class="text-muted-foreground hover:bg-muted/30 flex flex-1 items-center gap-1 px-2 py-1 text-[10px] font-semibold tracking-wider uppercase transition-colors"
      onclick={() => (explorerExpanded = !explorerExpanded)}
    >
      <ChevronRight
        class={cn('size-2.5 shrink-0 transition-transform duration-100', explorerExpanded && 'rotate-90')}
      />
      Explorer
    </button>
    <Button
      size="icon-xs"
      variant="ghost"
      class="size-5"
      title={chatT('chat.explorer.newFile')}
      data-testid="workspace-browser-new-file"
      onclick={() => onStartCreate?.('file')}
    >
      <FilePlus2 class="size-3" />
    </Button>
    <Button
      size="icon-xs"
      variant="ghost"
      class="size-5"
      title={chatT('chat.explorer.newFolder')}
      data-testid="workspace-browser-new-folder"
      onclick={() => onStartCreate?.('folder')}
    >
      <FolderPlus class="size-3" />
    </Button>
  </div>

  {#if explorerExpanded}
    <div class="min-h-0 flex-1 overflow-y-auto pb-1" data-testid="workspace-browser-explorer-list">
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
        {onOpenMenu}
        onSelectFile={(path) => onSelectFile?.(path)}
        onCommitCreate={(name) => onCommitCreate?.(name)}
        onCancelCreate={() => onCancelCreate?.()}
        onCommitRename={(name) => onCommitRename?.(name)}
        onCancelRename={() => onCancelRename?.()}
      />
    </div>
  {/if}
</div>
