<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    ProjectConversationWorkspaceBranchRef,
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceCurrentRef,
    ProjectConversationWorkspaceDiffRepo,
  } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import * as Popover from '$ui/popover'
  import * as Command from '$ui/command'
  import { ChevronRight, ChevronDown, GitBranch, Minus, Plus, Undo2 } from '@lucide/svelte'
  import { fileIcon, formatTotals } from './project-conversation-workspace-browser-helpers'
  import {
    dirtyFileColorClass,
    filenameFromPath,
  } from './project-conversation-workspace-browser-sidebar-helpers'

  let {
    currentRef = null,
    localBranches = [],
    remoteBranches = [],
    repoRefsLoading = false,
    repoRefsError = '',
    checkoutBlockers = [],
    selectedRepo = null,
    selectedRepoDiff = null,
    selectedFilePath = '',
    onCheckoutBranch,
    onCreateBranchName,
    onStageFile,
    onStageAll,
    onUnstage,
    onCommitRepo,
    onDiscardFile,
    onSelectFile,
  }: {
    currentRef?: ProjectConversationWorkspaceCurrentRef | null
    localBranches?: ProjectConversationWorkspaceBranchRef[]
    remoteBranches?: ProjectConversationWorkspaceBranchRef[]
    repoRefsLoading?: boolean
    repoRefsError?: string
    checkoutBlockers?: string[]
    selectedRepo?: { branch: string; headCommit: string } | null
    selectedRepoDiff?: ProjectConversationWorkspaceDiffRepo | null
    selectedFilePath?: string
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
    onCreateBranchName?: (branchName: string) => Promise<void>
    onStageFile?: (path: string) => Promise<void>
    onStageAll?: () => Promise<void>
    onUnstage?: (path?: string) => Promise<void>
    onCommitRepo?: (message: string) => Promise<void>
    onDiscardFile?: (path: string) => Promise<void>
    onSelectFile?: (path: string) => void
  } = $props()

  let popoverOpen = $state(false)
  let searchValue = $state('')
  let checkoutPending = $state('')
  let actionPendingPath = $state('')
  let actionError = $state('')
  let commitMessage = $state('')
  let commitPending = $state(false)
  let changesExpanded = $state(false)
  let stagedSectionOpen = $state(true)
  let unstagedSectionOpen = $state(true)

  const currentRefLabel = $derived(currentRef?.displayName ?? selectedRepo?.branch ?? '')
  const commitHash = $derived(selectedRepo?.headCommit ?? '')
  const blocked = $derived(checkoutBlockers.length > 0)
  const trimmedSearchValue = $derived(searchValue.trim())
  const branchNameTaken = $derived(
    trimmedSearchValue !== '' &&
      [...localBranches, ...remoteBranches].some((branch) => branch.name === trimmedSearchValue),
  )
  const showCreateBranchAction = $derived(
    trimmedSearchValue !== '' && !branchNameTaken && !repoRefsLoading && !repoRefsError,
  )
  const diffFiles = $derived(selectedRepoDiff?.files ?? [])
  const hasDiff = $derived(selectedRepoDiff != null && selectedRepoDiff.filesChanged > 0)
  const stagedFiles = $derived(diffFiles.filter((file) => file.staged))
  const unstagedFiles = $derived(diffFiles.filter((file) => file.unstaged))
  const stagedFileCount = $derived(stagedFiles.length)
  const unstagedFileCount = $derived(unstagedFiles.length)
  const canCommit = $derived(
    stagedFileCount > 0 && commitMessage.trim().length > 0 && !commitPending,
  )

  async function handleSelect(branch: ProjectConversationWorkspaceBranchRef) {
    if (!onCheckoutBranch || blocked) return
    const isRemote = branch.scope === 'remote_tracking_branch'
    checkoutPending = branch.fullName
    actionError = ''
    try {
      const result = await onCheckoutBranch({
        targetKind: branch.scope,
        targetName: branch.name,
        createTrackingBranch: isRemote,
        localBranchName: isRemote ? branch.suggestedLocalBranchName || undefined : undefined,
      })
      if (result.ok) {
        popoverOpen = false
        searchValue = ''
      } else {
        actionError = result.blockers.join(' ')
      }
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to switch branches.'
    } finally {
      checkoutPending = ''
    }
  }

  async function handleStage(path: string) {
    if (!onStageFile) return
    actionPendingPath = path
    actionError = ''
    try {
      await onStageFile(path)
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to stage the file.'
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleCreateBranch() {
    if (!onCreateBranchName || !showCreateBranchAction) return
    checkoutPending = '__create__'
    actionError = ''
    try {
      await onCreateBranchName(trimmedSearchValue)
      searchValue = trimmedSearchValue
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to create the branch.'
    } finally {
      checkoutPending = ''
    }
  }

  async function handleStageAll() {
    if (!onStageAll) return
    actionPendingPath = '*'
    actionError = ''
    try {
      await onStageAll()
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to stage all files.'
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleUnstage(path = '') {
    if (!onUnstage) return
    actionPendingPath = path || '*'
    actionError = ''
    try {
      await onUnstage(path || undefined)
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to unstage changes.'
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleDiscard(path: string) {
    if (!onDiscardFile) return
    actionPendingPath = path
    actionError = ''
    try {
      await onDiscardFile(path)
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to discard the file changes.'
    } finally {
      actionPendingPath = ''
    }
  }

  async function handleCommit() {
    if (!onCommitRepo || !canCommit) return
    commitPending = true
    actionError = ''
    try {
      await onCommitRepo(commitMessage.trim())
      commitMessage = ''
    } catch (error) {
      actionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to create the commit.'
    } finally {
      commitPending = false
    }
  }
</script>

{#if selectedRepo || currentRef}
  <div class="border-border shrink-0 border-t">
    <div class="bg-muted/30 flex items-center text-[11px]">
      <Popover.Root
        bind:open={popoverOpen}
        onOpenChange={(next) => {
          if (!next) {
            searchValue = ''
            actionError = ''
          }
        }}
      >
        <Popover.Trigger
          class="hover:bg-muted/60 flex shrink-0 items-center gap-1.5 px-3 py-1.5 transition-colors"
          onclick={(e: MouseEvent) => e.stopPropagation()}
        >
          <GitBranch class="text-muted-foreground size-3 shrink-0" />
          <span class="max-w-[120px] truncate font-medium">{currentRefLabel}</span>
          <ChevronDown class="text-muted-foreground size-2.5 shrink-0" />
        </Popover.Trigger>

        <Popover.Content
          class="branch-picker-popover w-56 gap-0! p-0!"
          side="top"
          align="start"
          sideOffset={2}
        >
          <Command.Root shouldFilter={true} class="p-0!">
            <Command.Input
              placeholder="Branch…"
              bind:value={searchValue}
              onkeydown={(event) => {
                if (event.key === 'Enter' && showCreateBranchAction) {
                  event.preventDefault()
                  void handleCreateBranch()
                }
              }}
            />

            {#if actionError || blocked}
              <div
                class={cn(
                  'border-b px-2 py-1 text-[10px]',
                  actionError
                    ? 'border-destructive/20 bg-destructive/5 text-destructive'
                    : 'border-amber-500/20 bg-amber-500/10 text-amber-900',
                )}
              >
                {actionError || checkoutBlockers[0]}
              </div>
            {/if}

            <Command.List class="max-h-56!">
              {#if repoRefsLoading}
                <Command.Empty>Loading…</Command.Empty>
              {:else if repoRefsError}
                <Command.Empty>{repoRefsError}</Command.Empty>
              {:else}
                {#if showCreateBranchAction}
                  <Command.Item
                    value={`create ${trimmedSearchValue}`}
                    onSelect={() => void handleCreateBranch()}
                    disabled={checkoutPending !== ''}
                    class="gap-1! px-2! py-0.5!"
                    data-testid="workspace-branch-create"
                  >
                    <Plus class="size-3" />
                    <span class="min-w-0 truncate text-[11px]">
                      Create branch "{trimmedSearchValue}"
                    </span>
                  </Command.Item>
                  {#if localBranches.length > 0 || remoteBranches.length > 0}
                    <Command.Separator />
                  {/if}
                {/if}

                {#each localBranches as branch (branch.fullName)}
                  <Command.Item
                    value={branch.name}
                    onSelect={() => handleSelect(branch)}
                    disabled={branch.current || blocked || checkoutPending !== ''}
                    class={cn('gap-1! px-2! py-0.5!', branch.current && 'bg-primary/5')}
                  >
                    <span class="min-w-0 truncate text-[11px]">{branch.name}</span>
                    {#if branch.current}
                      <span class="text-primary ml-auto text-[9px]">✓</span>
                    {:else if branch.ahead || branch.behind}
                      <span class="text-muted-foreground/50 ml-auto text-[9px]">
                        +{branch.ahead} −{branch.behind}
                      </span>
                    {/if}
                  </Command.Item>
                {/each}

                {#if localBranches.length > 0 && remoteBranches.length > 0}
                  <Command.Separator />
                {/if}

                {#each remoteBranches as branch (branch.fullName)}
                  <Command.Item
                    value={branch.name}
                    onSelect={() => handleSelect(branch)}
                    disabled={blocked || checkoutPending !== ''}
                    class="gap-1! px-2! py-0.5!"
                  >
                    <span class="text-muted-foreground min-w-0 truncate text-[11px]">
                      {branch.name}
                    </span>
                  </Command.Item>
                {/each}

                {#if localBranches.length === 0 && remoteBranches.length === 0}
                  <Command.Empty>No branches.</Command.Empty>
                {/if}
              {/if}
            </Command.List>
          </Command.Root>
        </Popover.Content>
      </Popover.Root>

      <button
        type="button"
        class="hover:bg-muted/60 flex min-w-0 flex-1 items-center gap-1.5 px-2 py-1.5 transition-colors"
        onclick={() => (changesExpanded = !changesExpanded)}
      >
        {#if commitHash}
          <span class="text-muted-foreground/50 shrink-0 font-mono text-[10px]">{commitHash}</span>
        {/if}
        {#if hasDiff}
          <span class="text-muted-foreground/50 shrink-0">·</span>
          <span class="shrink-0 text-[10px] font-medium">
            {selectedRepoDiff?.filesChanged} file{selectedRepoDiff?.filesChanged === 1 ? '' : 's'}
          </span>
          {#if stagedFileCount > 0}
            <span class="shrink-0 text-[9px] text-emerald-600">{stagedFileCount}✓</span>
          {/if}
          <span class="shrink-0 text-[10px] text-emerald-600">+{selectedRepoDiff?.added ?? 0}</span>
          <span class="shrink-0 text-[10px] text-red-500">-{selectedRepoDiff?.removed ?? 0}</span>
          <ChevronRight
            class={cn(
              'text-muted-foreground ml-auto size-2.5 shrink-0 transition-transform duration-100',
              changesExpanded && 'rotate-90',
            )}
          />
        {:else}
          <span class="text-muted-foreground/40 text-[10px]">clean</span>
        {/if}
      </button>
    </div>

    {#if changesExpanded && hasDiff}
      <div class="border-border max-h-64 overflow-y-auto border-t">
        <div class="bg-muted/20 flex items-end gap-1 px-2 py-1.5">
          <textarea
            class="border-input bg-background placeholder:text-muted-foreground/50 focus-visible:ring-ring min-w-0 flex-1 resize-none rounded border px-2 py-1 text-[11px] leading-snug outline-none focus-visible:ring-1"
            placeholder="Commit message… (Ctrl+Enter to commit)"
            rows="2"
            bind:value={commitMessage}
            data-testid="workspace-branch-commit-message"
            onkeydown={(event) => {
              if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
                event.preventDefault()
                void handleCommit()
              }
            }}
          ></textarea>
          <button
            type="button"
            class={cn(
              'shrink-0 rounded px-2 py-1 text-[10px] font-semibold transition-colors',
              canCommit
                ? 'bg-primary text-primary-foreground hover:bg-primary/90'
                : 'bg-muted text-muted-foreground cursor-not-allowed',
            )}
            data-testid="workspace-branch-commit-button"
            disabled={!canCommit}
            onclick={() => void handleCommit()}
          >
            {commitPending ? 'Committing…' : 'Commit'}
          </button>
        </div>

        {#if actionError}
          <div class="border-destructive/20 bg-destructive/5 border-t px-3 py-1 text-[11px]">
            <span class="text-destructive">{actionError}</span>
          </div>
        {/if}

        {#if stagedFileCount > 0}
          <div class="border-border border-t">
            <div class="flex items-center gap-0.5 pr-1">
              <button
                type="button"
                class="text-muted-foreground hover:bg-muted/30 flex flex-1 items-center gap-1 px-2 py-0.5 text-[10px] font-semibold tracking-wider uppercase transition-colors"
                onclick={() => (stagedSectionOpen = !stagedSectionOpen)}
              >
                <ChevronRight
                  class={cn(
                    'size-2.5 shrink-0 transition-transform duration-100',
                    stagedSectionOpen && 'rotate-90',
                  )}
                />
                Staged
                <span class="sr-only">{stagedFileCount} staged</span>
                <span
                  class="text-muted-foreground/50 ml-auto text-[9px] font-normal tracking-normal normal-case"
                >
                  {stagedFileCount}
                </span>
              </button>
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded p-0.5 transition-colors"
                title="Unstage all"
                data-testid="workspace-branch-unstage-all"
                disabled={actionPendingPath !== '' || commitPending}
                onclick={() => void handleUnstage()}
              >
                <Minus class="size-3" />
              </button>
            </div>
            {#if stagedSectionOpen}
              {#each stagedFiles as file (file.path)}
                <div
                  class={cn(
                    'hover:bg-muted/40 group flex items-center gap-1 py-[3px] pr-2 pl-5 text-[11px] transition-colors',
                    file.path === selectedFilePath && 'bg-primary/10',
                  )}
                >
                  <button
                    type="button"
                    class="flex min-w-0 flex-1 items-center gap-1.5 text-left"
                    title="{file.path}\n{file.status} · +{file.added} -{file.removed}"
                    data-testid={`workspace-branch-file-${file.path}`}
                    onclick={() => onSelectFile?.(file.path)}
                  >
                    {#each [fileIcon(filenameFromPath(file.path))] as fi}
                      <fi.icon class={cn('size-3 shrink-0', fi.colorClass)} />
                    {/each}
                    <span class={cn('min-w-0 truncate', dirtyFileColorClass(file.status))}>
                      {filenameFromPath(file.path)}
                    </span>
                    {#if file.path.includes('/')}
                      <span class="text-muted-foreground/30 min-w-0 shrink truncate text-[9px]">
                        {file.path.slice(0, file.path.lastIndexOf('/'))}
                      </span>
                    {/if}
                  </button>
                  <span class="text-muted-foreground/40 shrink-0 text-[9px]">
                    {formatTotals(file.added, file.removed)}
                  </span>
                  {#if file.status === 'deleted'}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-red-500">D</span>
                  {:else if file.status === 'added'}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-emerald-600">A</span>
                  {:else if file.status === 'renamed'}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-sky-500">R</span>
                  {:else}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-amber-500">M</span>
                  {/if}
                  <button
                    type="button"
                    class="text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded p-0.5 opacity-0 transition-all group-hover:opacity-100"
                    title="Unstage"
                    data-testid={`workspace-branch-unstage-${file.path}`}
                    disabled={actionPendingPath !== '' || commitPending}
                    onclick={(event) => {
                      event.stopPropagation()
                      void handleUnstage(file.path)
                    }}
                  >
                    <Minus class="size-3" />
                  </button>
                </div>
              {/each}
            {/if}
          </div>
        {/if}

        {#if unstagedFileCount > 0}
          <div class="border-border border-t">
            <div class="flex items-center gap-0.5 pr-1">
              <button
                type="button"
                class="text-muted-foreground hover:bg-muted/30 flex flex-1 items-center gap-1 px-2 py-0.5 text-[10px] font-semibold tracking-wider uppercase transition-colors"
                onclick={() => (unstagedSectionOpen = !unstagedSectionOpen)}
              >
                <ChevronRight
                  class={cn(
                    'size-2.5 shrink-0 transition-transform duration-100',
                    unstagedSectionOpen && 'rotate-90',
                  )}
                />
                Changes
                <span
                  class="text-muted-foreground/50 ml-auto text-[9px] font-normal tracking-normal normal-case"
                >
                  {unstagedFileCount}
                </span>
              </button>
              <button
                type="button"
                class="text-muted-foreground hover:text-foreground hover:bg-muted shrink-0 rounded p-0.5 transition-colors"
                title="Stage all"
                data-testid="workspace-branch-stage-all"
                disabled={actionPendingPath !== '' || commitPending}
                onclick={() => void handleStageAll()}
              >
                <Plus class="size-3" />
              </button>
            </div>
            {#if unstagedSectionOpen}
              {#each unstagedFiles as file (file.path)}
                <div
                  class={cn(
                    'hover:bg-muted/40 group flex items-center gap-1 py-[3px] pr-2 pl-5 text-[11px] transition-colors',
                    file.path === selectedFilePath && 'bg-primary/10',
                  )}
                >
                  <button
                    type="button"
                    class="flex min-w-0 flex-1 items-center gap-1.5 text-left"
                    title="{file.path}\n{file.status} · +{file.added} -{file.removed}"
                    data-testid={`workspace-branch-file-${file.path}`}
                    onclick={() => onSelectFile?.(file.path)}
                  >
                    {#each [fileIcon(filenameFromPath(file.path))] as fi}
                      <fi.icon class={cn('size-3 shrink-0', fi.colorClass)} />
                    {/each}
                    <span class={cn('min-w-0 truncate', dirtyFileColorClass(file.status))}>
                      {filenameFromPath(file.path)}
                    </span>
                    {#if file.path.includes('/')}
                      <span class="text-muted-foreground/30 min-w-0 shrink truncate text-[9px]">
                        {file.path.slice(0, file.path.lastIndexOf('/'))}
                      </span>
                    {/if}
                  </button>
                  <span class="text-muted-foreground/40 shrink-0 text-[9px]">
                    {formatTotals(file.added, file.removed)}
                  </span>
                  {#if file.status === 'deleted'}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-red-500">D</span>
                  {:else if file.status === 'added' || file.status === 'untracked'}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-emerald-600">
                      {file.status === 'untracked' ? 'U' : 'A'}
                    </span>
                  {:else if file.status === 'renamed'}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-sky-500">R</span>
                  {:else}
                    <span class="shrink-0 font-mono text-[9px] font-bold text-amber-500">M</span>
                  {/if}
                  <div
                    class="flex shrink-0 items-center gap-0.5 opacity-0 transition-opacity group-hover:opacity-100"
                  >
                    <button
                      type="button"
                      class="text-muted-foreground hover:text-foreground hover:bg-muted rounded p-0.5 transition-colors"
                      title="Stage"
                      data-testid={`workspace-branch-stage-${file.path}`}
                      disabled={actionPendingPath !== '' || commitPending}
                      onclick={(event) => {
                        event.stopPropagation()
                        void handleStage(file.path)
                      }}
                    >
                      <Plus class="size-3" />
                    </button>
                    <button
                      type="button"
                      class="text-muted-foreground hover:text-foreground hover:bg-muted rounded p-0.5 transition-colors"
                      title="Discard"
                      data-testid={`workspace-branch-discard-${file.path}`}
                      disabled={actionPendingPath !== '' || commitPending}
                      onclick={(event) => {
                        event.stopPropagation()
                        void handleDiscard(file.path)
                      }}
                    >
                      <Undo2 class="size-3" />
                    </button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}

<style>
  :global(.branch-picker-popover [data-slot='command-input-wrapper']) {
    padding: 0;
  }
  :global(.branch-picker-popover [data-slot='command-input-wrapper'] [data-slot='input-group']) {
    border: none !important;
    background: transparent !important;
    box-shadow: none !important;
    height: 28px !important;
    border-radius: 0 !important;
    border-bottom: 1px solid var(--color-border) !important;
  }
  :global(
    .branch-picker-popover [data-slot='command-input-wrapper'] [data-slot='input-group-addon']
  ) {
    display: none;
  }
  :global(.branch-picker-popover [data-slot='command-input']) {
    font-size: 11px !important;
    padding-left: 8px !important;
  }
</style>
