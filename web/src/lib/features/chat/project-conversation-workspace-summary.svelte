<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronRight, GitBranch } from '@lucide/svelte'
  import type {
    ProjectConversationWorkspaceDiff,
    ProjectConversationWorkspaceFileStatus,
  } from '$lib/api/chat'

  let {
    conversationId = '',
    workspaceDiff = null,
    loading = false,
    error = '',
  }: {
    conversationId?: string
    workspaceDiff?: ProjectConversationWorkspaceDiff | null
    loading?: boolean
    error?: string
  } = $props()

  let expanded = $state(false)

  function formatTotals(added: number, removed: number) {
    return `+${added} -${removed}`
  }

  function formatRepoSummary(diff: ProjectConversationWorkspaceDiff) {
    const repoLabel = diff.reposChanged === 1 ? 'repo changed' : 'repos changed'
    return `${diff.reposChanged} ${repoLabel} · ${formatTotals(diff.added, diff.removed)}`
  }

  function statusLabel(status: ProjectConversationWorkspaceFileStatus) {
    switch (status) {
      case 'added':
        return 'A'
      case 'deleted':
        return 'D'
      case 'renamed':
        return 'R'
      case 'untracked':
        return 'U'
      default:
        return 'M'
    }
  }

  function statusClass(status: ProjectConversationWorkspaceFileStatus) {
    switch (status) {
      case 'added':
      case 'untracked':
        return 'text-emerald-600'
      case 'deleted':
        return 'text-rose-600'
      case 'renamed':
        return 'text-amber-600'
      default:
        return 'text-sky-600'
    }
  }

  const hasContent = $derived(!!conversationId && !loading && !error && !!workspaceDiff)
  const isDirty = $derived(workspaceDiff?.dirty ?? false)
</script>

{#if !conversationId && !loading && !error}
  <!-- No conversation — hide entirely -->
{:else}
  <div class="border-border border-b">
    <button
      type="button"
      class="hover:bg-muted/30 flex w-full items-center gap-2 px-3 py-1 text-left text-[11px] transition-colors"
      onclick={() => {
        if (hasContent) expanded = !expanded
      }}
      disabled={!hasContent}
    >
      {#if hasContent}
        <ChevronRight
          class={cn(
            'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
            expanded && 'rotate-90',
          )}
        />
      {/if}

      <span class="text-muted-foreground">Workspace changes</span>

      {#if loading}
        <span class="text-muted-foreground/60">Loading...</span>
      {:else if error}
        <span class="text-destructive truncate">{error}</span>
      {:else if workspaceDiff}
        {#if isDirty}
          <span class="font-medium">{formatRepoSummary(workspaceDiff)}</span>
        {:else}
          <span class="text-muted-foreground/60">Clean workspace</span>
        {/if}
      {/if}
    </button>

    {#if expanded && workspaceDiff}
      <div class="border-border border-t text-[11px]">
        {#if workspaceDiff.repos.length === 0}
          <p class="text-muted-foreground px-3 py-1.5">No repo changes detected.</p>
        {:else}
          {#each workspaceDiff.repos as repo, repoIndex}
            {#if workspaceDiff.repos.length > 1}
              <div
                class={cn(
                  'text-muted-foreground flex items-center gap-1.5 px-3 py-1',
                  repoIndex > 0 && 'border-border border-t',
                )}
              >
                <GitBranch class="size-3 shrink-0" />
                <span class="font-medium">{repo.name}</span>
                <span class="text-muted-foreground/60 font-mono text-[10px]">
                  {repo.path} · {repo.branch}
                </span>
                <span class="ml-auto shrink-0 font-mono text-[10px]">
                  {formatTotals(repo.added, repo.removed)}
                </span>
              </div>
            {/if}
            {#each repo.files as file}
              <div class="flex items-center gap-1.5 px-3 py-0.5">
                <span class={cn('w-3 shrink-0 font-mono font-bold', statusClass(file.status))}>
                  {statusLabel(file.status)}
                </span>
                <span class="text-foreground/80 min-w-0 flex-1 truncate font-mono text-[10px]">
                  {file.path}
                </span>
                <span class="text-muted-foreground/60 shrink-0 font-mono text-[10px]">
                  {formatTotals(file.added, file.removed)}
                </span>
              </div>
            {/each}
          {/each}
          <div class="h-1"></div>
        {/if}
      </div>
    {/if}
  </div>
{/if}
