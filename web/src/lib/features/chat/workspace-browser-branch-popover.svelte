<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    ProjectConversationWorkspaceBranchRef,
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceCurrentRef,
  } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import { chatT } from './i18n'
  import * as Command from '$ui/command'
  import * as Popover from '$ui/popover'
  import { ChevronDown, Plus, GitBranch } from '@lucide/svelte'

  let {
    currentRef = null,
    selectedBranch = '',
    localBranches = [],
    remoteBranches = [],
    repoRefsLoading = false,
    repoRefsError = '',
    checkoutBlockers = [],
    onCheckoutBranch,
    onCreateBranchName,
  }: {
    currentRef?: ProjectConversationWorkspaceCurrentRef | null
    selectedBranch?: string
    localBranches?: ProjectConversationWorkspaceBranchRef[]
    remoteBranches?: ProjectConversationWorkspaceBranchRef[]
    repoRefsLoading?: boolean
    repoRefsError?: string
    checkoutBlockers?: string[]
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
    onCreateBranchName?: (branchName: string) => Promise<void>
  } = $props()

  let popoverOpen = $state(false)
  let searchValue = $state('')
  let checkoutPending = $state('')
  let actionError = $state('')

  const currentRefLabel = $derived(currentRef?.displayName ?? selectedBranch)
  const blocked = $derived(checkoutBlockers.length > 0)
  const trimmedSearchValue = $derived(searchValue.trim())
  const branchNameTaken = $derived(
    trimmedSearchValue !== '' &&
      [...localBranches, ...remoteBranches].some((branch) => branch.name === trimmedSearchValue),
  )
  const showCreateBranchAction = $derived(
    trimmedSearchValue !== '' && !branchNameTaken && !repoRefsLoading && !repoRefsError,
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
</script>

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
    onclick={(event: MouseEvent) => event.stopPropagation()}
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
        placeholder={chatT('chat.branchPicker.placeholder')}
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
          <Command.Empty>{chatT('chat.loadingEllipsis')}</Command.Empty>
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
                  +{branch.ahead} -{branch.behind}
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
              <span class="text-muted-foreground min-w-0 truncate text-[11px]">{branch.name}</span>
            </Command.Item>
          {/each}

          {#if localBranches.length === 0 && remoteBranches.length === 0}
            <Command.Empty>{chatT('chat.branchPicker.noBranches')}</Command.Empty>
          {/if}
        {/if}
      </Command.List>
    </Command.Root>
  </Popover.Content>
</Popover.Root>

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
