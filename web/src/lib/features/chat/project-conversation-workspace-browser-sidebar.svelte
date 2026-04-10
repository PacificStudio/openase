<script lang="ts">
  import { ScrollArea } from '$ui/scroll-area'
  import { cn } from '$lib/utils'
  import { FileCode2, FolderOpen, GitBranch } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceDiffRepo,
    ProjectConversationWorkspaceRepoMetadata,
    ProjectConversationWorkspaceTree,
    ProjectConversationWorkspaceTreeEntry,
  } from '$lib/api/chat'
  import {
    directorySegments,
    formatTotals,
    joinSegments,
    repoDirtyLabel,
    statusClass,
    statusLabel,
  } from './project-conversation-workspace-browser-helpers'

  let {
    repos = [],
    selectedRepoPath = '',
    selectedRepo = null,
    selectedRepoDiff = null,
    tree = null,
    treeLoading = false,
    treeError = '',
    selectedFilePath = '',
    currentTreePath = '',
    onOpenRepo,
    onOpenTreePath,
    onOpenEntry,
    onOpenDirtyFile,
  }: {
    repos?: ProjectConversationWorkspaceRepoMetadata[]
    selectedRepoPath?: string
    selectedRepo?: ProjectConversationWorkspaceRepoMetadata | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    tree?: ProjectConversationWorkspaceTree | null
    treeLoading?: boolean
    treeError?: string
    selectedFilePath?: string
    currentTreePath?: string
    onOpenRepo?: (repoPath: string) => void
    onOpenTreePath?: (path: string) => void
    onOpenEntry?: (entry: ProjectConversationWorkspaceTreeEntry) => void
    onOpenDirtyFile?: (path: string) => void
  } = $props()

  const currentEntries = $derived(tree?.entries ?? [])
  const dirtyFiles = $derived(selectedRepoDiff?.files ?? [])
</script>

<div class="border-border flex min-h-0 flex-col border-r">
  <div class="border-border flex flex-wrap gap-2 border-b p-3">
    {#each repos as repo}
      <button
        type="button"
        class={cn(
          'border-border bg-background hover:bg-muted/40 flex min-w-0 flex-1 flex-col rounded-lg border px-3 py-2 text-left transition-colors',
          repo.path === selectedRepoPath && 'border-primary bg-primary/5',
        )}
        onclick={() => onOpenRepo?.(repo.path)}
      >
        <span class="truncate text-sm font-medium">{repo.name}</span>
        <span class="text-muted-foreground truncate font-mono text-[10px]">{repo.path}</span>
        <span class="mt-1 truncate font-mono text-[10px]">
          {repoDirtyLabel(repo)} · {formatTotals(repo.added, repo.removed)}
        </span>
      </button>
    {/each}
  </div>

  {#if selectedRepo}
    <div class="border-border space-y-2 border-b px-3 py-2 text-[11px]">
      <div class="flex items-center gap-1.5">
        <GitBranch class="text-muted-foreground size-3" />
        <span class="font-medium">{selectedRepo.branch}</span>
      </div>
      <div class="text-muted-foreground flex flex-wrap items-center gap-1 font-mono">
        <span>{selectedRepo.headCommit}</span>
        <span aria-hidden="true">·</span>
        <span class="min-w-0 truncate">{selectedRepo.headSummary}</span>
      </div>
    </div>
  {/if}

  <div class="min-h-0 flex-1">
    <ScrollArea class="h-full">
      <div class="space-y-3 p-3">
        <section class="space-y-2">
          <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
            Tree
          </div>
          <div class="flex flex-wrap gap-1">
            <button
              type="button"
              class={cn(
                'border-border hover:bg-muted/40 rounded-full border px-2 py-0.5 text-[11px]',
                currentTreePath === '' && 'border-primary bg-primary/5',
              )}
              onclick={() => onOpenTreePath?.('')}
            >
              repo root
            </button>
            {#each directorySegments(currentTreePath) as segment, index}
              <button
                type="button"
                class="border-border hover:bg-muted/40 rounded-full border px-2 py-0.5 text-[11px]"
                onclick={() =>
                  onOpenTreePath?.(
                    joinSegments(directorySegments(currentTreePath).slice(0, index + 1)),
                  )}
              >
                {segment}
              </button>
            {/each}
          </div>

          {#if treeLoading && !tree}
            <p class="text-muted-foreground text-sm">Loading files…</p>
          {:else if treeError}
            <p class="text-destructive text-sm">{treeError}</p>
          {:else if currentEntries.length === 0}
            <p class="text-muted-foreground text-sm">This directory is empty.</p>
          {:else}
            <div class="space-y-1">
              {#each currentEntries as entry}
                <button
                  type="button"
                  class={cn(
                    'hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-sm transition-colors',
                    entry.path === selectedFilePath && 'bg-primary/5 text-primary',
                  )}
                  onclick={() => onOpenEntry?.(entry)}
                >
                  {#if entry.kind === 'directory'}
                    <FolderOpen class="text-muted-foreground size-3.5 shrink-0" />
                  {:else}
                    <FileCode2 class="text-muted-foreground size-3.5 shrink-0" />
                  {/if}
                  <span class="min-w-0 flex-1 truncate font-mono text-[12px]">{entry.name}</span>
                  {#if entry.kind === 'file'}
                    <span class="text-muted-foreground shrink-0 text-[10px]"
                      >{entry.sizeBytes} B</span
                    >
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </section>

        <section class="space-y-2">
          <div class="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
            Git status
          </div>
          {#if dirtyFiles.length === 0}
            <p class="text-muted-foreground text-sm">No changed files in this repo.</p>
          {:else}
            <div class="space-y-1">
              {#each dirtyFiles as file}
                <button
                  type="button"
                  class={cn(
                    'hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-sm transition-colors',
                    file.path === selectedFilePath && 'bg-primary/5 text-primary',
                  )}
                  onclick={() => onOpenDirtyFile?.(file.path)}
                >
                  <span
                    class={cn(
                      'w-3 shrink-0 font-mono text-[10px] font-bold',
                      statusClass(file.status),
                    )}
                  >
                    {statusLabel(file.status)}
                  </span>
                  <span class="min-w-0 flex-1 truncate font-mono text-[12px]">{file.path}</span>
                  <span class="text-muted-foreground shrink-0 font-mono text-[10px]">
                    {formatTotals(file.added, file.removed)}
                  </span>
                </button>
              {/each}
            </div>
          {/if}
        </section>
      </div>
    </ScrollArea>
  </div>
</div>
