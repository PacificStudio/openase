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

  function formatFileCount(count: number) {
    return `${count} ${count === 1 ? 'file' : 'files'}`
  }

  function statusBadgeClass(status: ProjectConversationWorkspaceFileStatus) {
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
      <div class="border-border border-t px-3 py-2 text-xs">
        <div class="flex items-center gap-2">
          <p class={cn('text-[11px] font-medium', isDirty ? 'text-amber-600' : 'text-emerald-600')}>
            {#if isDirty}
              Uncommitted changes in this Project AI workspace
            {:else}
              No uncommitted changes in this Project AI workspace
            {/if}
          </p>
        </div>
        <p class="text-muted-foreground/60 mt-0.5 font-mono text-[10px] break-all">
          {workspaceDiff.workspacePath}
        </p>

        {#if workspaceDiff.repos.length === 0}
          <p class="text-muted-foreground mt-2 text-[11px]">
            No repo changes are currently detected in this conversation workspace.
          </p>
        {:else}
          <div class="mt-2 space-y-1.5">
            {#each workspaceDiff.repos as repo}
              <details
                class="rounded-md border px-2 py-1.5"
                open={workspaceDiff.repos.length === 1}
              >
                <summary class="flex cursor-pointer list-none items-center gap-2">
                  <GitBranch class="text-muted-foreground size-3 shrink-0" />
                  <span class="min-w-0 flex-1 truncate font-medium">{repo.name}</span>
                  <span class="text-muted-foreground/60 font-mono text-[10px]">
                    {repo.path} · {repo.branch}
                  </span>
                  <span class="shrink-0 font-medium">
                    {formatTotals(repo.added, repo.removed)}
                  </span>
                  <span class="text-muted-foreground/60 shrink-0">
                    {formatFileCount(repo.filesChanged)}
                  </span>
                </summary>

                <div class="mt-1.5 space-y-0.5">
                  {#each repo.files as file}
                    <div class="hover:bg-muted/30 flex items-center gap-2 rounded px-1.5 py-0.5">
                      <span
                        class={cn(
                          'w-12 shrink-0 text-[10px] font-medium',
                          statusBadgeClass(file.status),
                        )}
                      >
                        {file.status}
                      </span>
                      <span
                        class="text-foreground/80 min-w-0 flex-1 truncate font-mono text-[11px]"
                      >
                        {file.path}
                      </span>
                      <span class="text-muted-foreground shrink-0 font-mono text-[10px]">
                        {formatTotals(file.added, file.removed)}
                      </span>
                    </div>
                  {/each}
                </div>
              </details>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </div>
{/if}
