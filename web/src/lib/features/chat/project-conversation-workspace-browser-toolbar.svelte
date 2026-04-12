<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    ProjectConversationWorkspaceBranchRef,
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceCurrentRef,
  } from '$lib/api/chat'
  import { Button } from '$ui/button'
  import * as Dialog from '$ui/dialog'
  import { cn } from '$lib/utils'
  import { Check, Copy, FolderTree, GitBranch, RefreshCcw, SquareTerminal, X } from '@lucide/svelte'

  let {
    workspacePath = '',
    pathCopied = false,
    showTerminalButton = false,
    terminalPanelOpen = false,
    conversationId = '',
    metadataLoading = false,
    selectedRepoName = '',
    currentRef = null,
    localBranches = [],
    remoteBranches = [],
    repoRefsLoading = false,
    repoRefsError = '',
    checkoutBlockers = [],
    onCopyWorkspacePath,
    onToggleTerminal,
    onRefreshWorkspace,
    onCheckoutBranch,
    onClose,
  }: {
    workspacePath?: string
    pathCopied?: boolean
    showTerminalButton?: boolean
    terminalPanelOpen?: boolean
    conversationId?: string
    metadataLoading?: boolean
    selectedRepoName?: string
    currentRef?: ProjectConversationWorkspaceCurrentRef | null
    localBranches?: ProjectConversationWorkspaceBranchRef[]
    remoteBranches?: ProjectConversationWorkspaceBranchRef[]
    repoRefsLoading?: boolean
    repoRefsError?: string
    checkoutBlockers?: string[]
    onCopyWorkspacePath?: () => void
    onToggleTerminal?: () => void
    onRefreshWorkspace?: () => void
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
    onClose?: () => void
  } = $props()

  let branchDialogOpen = $state(false)
  let branchActionError = $state('')
  let branchActionPending = $state('')

  const currentRefLabel = $derived(currentRef?.displayName ?? '')
  const hasBranchDialog = $derived(Boolean(currentRef) || repoRefsLoading || repoRefsError)

  async function handleCheckoutBranch(request: {
    targetKind: ProjectConversationWorkspaceBranchScope
    targetName: string
    createTrackingBranch: boolean
    localBranchName?: string
  }) {
    if (!onCheckoutBranch) {
      return
    }
    branchActionPending = request.targetName
    branchActionError = ''
    try {
      const result = await onCheckoutBranch(request)
      if (result.ok) {
        branchDialogOpen = false
        return
      }
      branchActionError = result.blockers.join(' ')
    } catch (error) {
      branchActionError =
        error instanceof ApiError
          ? error.detail
          : error instanceof Error
            ? error.message
            : 'Failed to switch branches.'
    } finally {
      branchActionPending = ''
    }
  }
</script>

<div class="border-border flex h-11 items-center gap-1.5 border-b px-3">
  <FolderTree class="text-muted-foreground size-3 shrink-0" />
  <span class="text-[12px] font-semibold">Workspace</span>
  {#if workspacePath}
    <button
      type="button"
      class="text-muted-foreground/50 hover:text-muted-foreground group flex min-w-0 items-center gap-1 truncate text-[11px] transition-colors"
      title="Click to copy path"
      onclick={onCopyWorkspacePath}
    >
      <span class="min-w-0 truncate">{workspacePath}</span>
      {#if pathCopied}
        <Check class="size-2.5 shrink-0 text-emerald-500" />
      {:else}
        <Copy class="size-2.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100" />
      {/if}
    </button>
  {/if}

  {#if currentRefLabel}
    <button
      type="button"
      class="hover:bg-muted/50 ml-2 inline-flex items-center gap-1 rounded-full border border-border px-2.5 py-1 text-[11px] font-medium transition-colors"
      onclick={() => {
        if (hasBranchDialog) {
          branchDialogOpen = true
          branchActionError = ''
        }
      }}
      disabled={!hasBranchDialog}
      aria-label="Open branch switcher"
    >
      <GitBranch class="size-3 shrink-0" />
      {#if selectedRepoName}
        <span class="text-muted-foreground">{selectedRepoName}</span>
        <span class="text-muted-foreground/50">/</span>
      {/if}
      <span>{currentRefLabel}</span>
    </button>
  {/if}

  <div class="flex-1"></div>
  {#if showTerminalButton}
    <Button
      variant={terminalPanelOpen ? 'secondary' : 'ghost'}
      size="icon-xs"
      class={cn('text-muted-foreground size-6', terminalPanelOpen && 'text-foreground')}
      aria-label="Toggle terminal"
      onclick={onToggleTerminal}
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
    onclick={onRefreshWorkspace}
    disabled={!conversationId || metadataLoading}
  >
    <RefreshCcw class={cn('size-3', metadataLoading && 'animate-spin')} />
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

<Dialog.Root bind:open={branchDialogOpen}>
  <Dialog.Content class="sm:max-w-2xl">
    <Dialog.Header>
      <Dialog.Title>Switch branch</Dialog.Title>
      <Dialog.Description>
        Review local and remote-tracking branches for {selectedRepoName || 'the selected repo'}.
      </Dialog.Description>
    </Dialog.Header>

    {#if checkoutBlockers.length > 0}
      <div class="border-amber-500/20 bg-amber-500/10 rounded-md border px-3 py-2 text-sm">
        <p class="font-medium text-amber-900">Branch switching is blocked</p>
        <ul class="mt-2 list-disc space-y-1 pl-5 text-amber-900">
          {#each checkoutBlockers as blocker}
            <li>{blocker}</li>
          {/each}
        </ul>
      </div>
    {/if}

    {#if branchActionError}
      <div class="border-destructive/20 bg-destructive/5 rounded-md border px-3 py-2 text-sm">
        <p class="text-destructive">{branchActionError}</p>
      </div>
    {/if}

    {#if repoRefsLoading}
      <div class="text-muted-foreground py-8 text-center text-sm">Loading branches...</div>
    {:else if repoRefsError}
      <div class="text-destructive py-8 text-center text-sm">{repoRefsError}</div>
    {:else}
      <div class="grid gap-4 md:grid-cols-2">
        <section class="space-y-2">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold">Local branches</h3>
            <span class="text-muted-foreground text-[11px]">{localBranches.length}</span>
          </div>
          <div class="max-h-80 space-y-2 overflow-auto rounded-md border border-border/60 p-2">
            {#if localBranches.length === 0}
              <p class="text-muted-foreground px-2 py-4 text-sm">No local branches available.</p>
            {:else}
              {#each localBranches as branch (branch.fullName)}
                <button
                  type="button"
                  class={cn(
                    'hover:bg-muted/50 flex w-full flex-col rounded-md border px-3 py-2 text-left transition-colors',
                    branch.current && 'border-primary/40 bg-primary/5',
                  )}
                  onclick={() =>
                    handleCheckoutBranch({
                      targetKind: 'local_branch',
                      targetName: branch.name,
                      createTrackingBranch: false,
                    })}
                  disabled={branch.current || checkoutBlockers.length > 0 || branchActionPending !== ''}
                >
                  <span class="flex items-center gap-2">
                    <span class="font-medium">{branch.name}</span>
                    {#if branch.current}
                      <span class="text-primary text-[10px] font-semibold uppercase">Current</span>
                    {/if}
                  </span>
                  <span class="text-muted-foreground mt-1 font-mono text-[11px]">
                    {branch.shortCommitId}
                    {#if branch.upstreamName}
                      · {branch.upstreamName}
                      {#if branch.ahead || branch.behind}
                        · +{branch.ahead}/-{branch.behind}
                      {/if}
                    {/if}
                  </span>
                  {#if branch.subject}
                    <span class="text-muted-foreground mt-1 text-[11px]">{branch.subject}</span>
                  {/if}
                </button>
              {/each}
            {/if}
          </div>
        </section>

        <section class="space-y-2">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold">Remote branches</h3>
            <span class="text-muted-foreground text-[11px]">{remoteBranches.length}</span>
          </div>
          <div class="max-h-80 space-y-2 overflow-auto rounded-md border border-border/60 p-2">
            {#if remoteBranches.length === 0}
              <p class="text-muted-foreground px-2 py-4 text-sm">No remote branches available.</p>
            {:else}
              {#each remoteBranches as branch (branch.fullName)}
                <button
                  type="button"
                  class="hover:bg-muted/50 flex w-full flex-col rounded-md border px-3 py-2 text-left transition-colors"
                  onclick={() =>
                    handleCheckoutBranch({
                      targetKind: 'remote_tracking_branch',
                      targetName: branch.name,
                      createTrackingBranch: true,
                      localBranchName: branch.suggestedLocalBranchName || undefined,
                    })}
                  disabled={checkoutBlockers.length > 0 || branchActionPending !== ''}
                >
                  <span class="font-medium">{branch.name}</span>
                  <span class="text-muted-foreground mt-1 text-[11px]">
                    Create and switch to
                    <span class="font-mono">{branch.suggestedLocalBranchName || branch.name}</span>
                  </span>
                  <span class="text-muted-foreground mt-1 font-mono text-[11px]">
                    {branch.shortCommitId}
                  </span>
                  {#if branch.subject}
                    <span class="text-muted-foreground mt-1 text-[11px]">{branch.subject}</span>
                  {/if}
                </button>
              {/each}
            {/if}
          </div>
        </section>
      </div>
    {/if}
  </Dialog.Content>
</Dialog.Root>
