<script lang="ts">
  import { ApiError } from '$lib/api/client'
  import type {
    ProjectConversationWorkspaceBranchRef,
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceCurrentRef,
  } from '$lib/api/chat'
  import { cn } from '$lib/utils'
  import * as Popover from '$ui/popover'
  import * as Command from '$ui/command'
  import { ChevronDown, GitBranch, Globe, Laptop } from '@lucide/svelte'

  let {
    currentRef = null,
    localBranches = [],
    remoteBranches = [],
    repoRefsLoading = false,
    repoRefsError = '',
    checkoutBlockers = [],
    selectedRepo = null,
    onCheckoutBranch,
  }: {
    currentRef?: ProjectConversationWorkspaceCurrentRef | null
    localBranches?: ProjectConversationWorkspaceBranchRef[]
    remoteBranches?: ProjectConversationWorkspaceBranchRef[]
    repoRefsLoading?: boolean
    repoRefsError?: string
    checkoutBlockers?: string[]
    selectedRepo?: { branch: string; headCommit: string } | null
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => Promise<{ ok: boolean; blockers: string[] }>
  } = $props()

  let open = $state(false)
  let searchValue = $state('')
  let actionPending = $state('')
  let actionError = $state('')

  const currentRefLabel = $derived(currentRef?.displayName ?? selectedRepo?.branch ?? '')
  const commitHash = $derived(selectedRepo?.headCommit ?? '')
  const blocked = $derived(checkoutBlockers.length > 0)

  async function handleSelect(branch: ProjectConversationWorkspaceBranchRef) {
    if (!onCheckoutBranch || blocked) return
    const isRemote = branch.scope === 'remote_tracking_branch'
    actionPending = branch.fullName
    actionError = ''
    try {
      const result = await onCheckoutBranch({
        targetKind: branch.scope,
        targetName: branch.name,
        createTrackingBranch: isRemote,
        localBranchName: isRemote ? branch.suggestedLocalBranchName || undefined : undefined,
      })
      if (result.ok) {
        open = false
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
      actionPending = ''
    }
  }
</script>

{#if selectedRepo || currentRef}
  <Popover.Root
    bind:open
    onOpenChange={(next) => {
      if (!next) {
        searchValue = ''
        actionError = ''
      }
    }}
  >
    <Popover.Trigger
      class="border-border bg-muted/30 hover:bg-muted/60 flex w-full items-center gap-1.5 border-t px-3 py-1.5 text-left text-[11px] transition-colors"
    >
      <GitBranch class="text-muted-foreground size-3 shrink-0" />
      <span class="min-w-0 truncate font-medium">{currentRefLabel}</span>
      {#if commitHash}
        <span class="text-muted-foreground/60 min-w-0 truncate font-mono text-[10px]">
          {commitHash}
        </span>
      {/if}
      <ChevronDown class="text-muted-foreground ml-auto size-3 shrink-0" />
    </Popover.Trigger>

    <Popover.Content
      class="w-[320px] p-0!"
      side="top"
      align="start"
      sideOffset={2}
    >
      <Command.Root shouldFilter={true} bind:value={searchValue}>
        <Command.Input placeholder="Switch branch…" bind:value={searchValue} />

        {#if actionError}
          <div class="border-destructive/20 bg-destructive/5 border-b px-3 py-1.5 text-[11px]">
            <span class="text-destructive">{actionError}</span>
          </div>
        {/if}

        {#if blocked}
          <div class="border-b border-amber-500/20 bg-amber-500/10 px-3 py-1.5 text-[11px]">
            <span class="text-amber-900">Checkout blocked: {checkoutBlockers[0]}</span>
          </div>
        {/if}

        <Command.List class="max-h-64!">
          {#if repoRefsLoading}
            <Command.Empty>Loading branches…</Command.Empty>
          {:else if repoRefsError}
            <Command.Empty>{repoRefsError}</Command.Empty>
          {:else}
            {#if localBranches.length > 0}
              <Command.Group heading="Local branches">
                {#each localBranches as branch (branch.fullName)}
                  <Command.Item
                    value={branch.name}
                    onSelect={() => handleSelect(branch)}
                    disabled={branch.current || blocked || actionPending !== ''}
                    class={cn(branch.current && 'bg-primary/5')}
                  >
                    <Laptop class="text-muted-foreground size-3.5 shrink-0" />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <span class="truncate text-[12px] font-medium">{branch.name}</span>
                        {#if branch.current}
                          <span
                            class="bg-primary/15 text-primary rounded-full px-1.5 text-[9px] font-bold"
                          >
                            current
                          </span>
                        {/if}
                      </div>
                      <div class="text-muted-foreground truncate text-[10px]">
                        {branch.shortCommitId}
                        {#if branch.upstreamName}
                          · ↑ {branch.upstreamName}
                          {#if branch.ahead || branch.behind}
                            +{branch.ahead}/-{branch.behind}
                          {/if}
                        {/if}
                      </div>
                    </div>
                  </Command.Item>
                {/each}
              </Command.Group>
            {/if}

            {#if remoteBranches.length > 0}
              <Command.Group heading="Remote branches">
                {#each remoteBranches as branch (branch.fullName)}
                  <Command.Item
                    value={branch.name}
                    onSelect={() => handleSelect(branch)}
                    disabled={blocked || actionPending !== ''}
                  >
                    <Globe class="text-muted-foreground size-3.5 shrink-0" />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-1.5">
                        <span class="truncate text-[12px] font-medium">{branch.name}</span>
                      </div>
                      <div class="text-muted-foreground truncate text-[10px]">
                        {branch.shortCommitId}
                        {#if branch.subject}
                          · {branch.subject}
                        {/if}
                      </div>
                    </div>
                  </Command.Item>
                {/each}
              </Command.Group>
            {/if}

            {#if localBranches.length === 0 && remoteBranches.length === 0}
              <Command.Empty>No branches found.</Command.Empty>
            {/if}
          {/if}
        </Command.List>
      </Command.Root>
    </Popover.Content>
  </Popover.Root>
{/if}
