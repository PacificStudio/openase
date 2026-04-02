<script lang="ts">
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
        return 'border-emerald-200 bg-emerald-50 text-emerald-700'
      case 'deleted':
        return 'border-rose-200 bg-rose-50 text-rose-700'
      case 'renamed':
        return 'border-amber-200 bg-amber-50 text-amber-700'
      default:
        return 'border-sky-200 bg-sky-50 text-sky-700'
    }
  }
</script>

<section class="border-border bg-muted/20 border-b px-4 py-3">
  <div class="flex flex-wrap items-center justify-between gap-2">
    <div>
      <h3 class="text-sm font-semibold">Workspace changes</h3>
      <p class="text-muted-foreground text-xs">
        These changes live inside the OpenASE-managed Project AI workspace for this conversation.
      </p>
    </div>

    {#if workspaceDiff}
      <div class="text-xs font-medium">
        {#if workspaceDiff.dirty}
          {formatRepoSummary(workspaceDiff)}
        {:else}
          Clean workspace
        {/if}
      </div>
    {/if}
  </div>

  {#if !conversationId && !loading && !error}
    <p class="text-muted-foreground mt-3 text-xs">
      Start or restore a Project AI conversation to inspect its isolated workspace.
    </p>
  {:else if loading}
    <p class="text-muted-foreground mt-3 text-xs">Loading Project AI workspace changes…</p>
  {:else if error}
    <p class="text-destructive mt-3 text-xs">{error}</p>
  {:else if workspaceDiff}
    <div class="mt-3 space-y-3">
      <div class="rounded-lg border px-3 py-2">
        <p
          class={`text-xs font-medium ${workspaceDiff.dirty ? 'text-amber-700' : 'text-emerald-700'}`}
        >
          {#if workspaceDiff.dirty}
            Uncommitted changes in this Project AI workspace
          {:else}
            No uncommitted changes in this Project AI workspace
          {/if}
        </p>
        <p class="text-muted-foreground mt-1 font-mono text-[11px] break-all">
          {workspaceDiff.workspacePath}
        </p>
      </div>

      {#if workspaceDiff.repos.length === 0}
        <p class="text-muted-foreground text-xs">
          No repo changes are currently detected in this conversation workspace.
        </p>
      {:else}
        <div class="space-y-2">
          {#each workspaceDiff.repos as repo}
            <details class="rounded-lg border px-3 py-2" open={workspaceDiff.repos.length === 1}>
              <summary class="cursor-pointer list-none">
                <div class="flex flex-wrap items-start justify-between gap-2">
                  <div>
                    <div class="text-sm font-medium">{repo.name}</div>
                    <div class="text-muted-foreground font-mono text-[11px] break-all">
                      {repo.path} · {repo.branch}
                    </div>
                  </div>
                  <div class="text-right text-xs">
                    <div class="font-medium">{formatTotals(repo.added, repo.removed)}</div>
                    <div class="text-muted-foreground">{formatFileCount(repo.filesChanged)}</div>
                  </div>
                </div>
              </summary>

              <div class="mt-3 space-y-2">
                {#each repo.files as file}
                  <div
                    class="flex flex-wrap items-start justify-between gap-2 rounded-md border px-2 py-2"
                  >
                    <div class="min-w-0 flex-1">
                      <div class="font-mono text-[11px] break-all">{file.path}</div>
                      <span
                        class={`mt-1 inline-flex rounded-full border px-2 py-0.5 text-[10px] font-medium tracking-wide uppercase ${statusBadgeClass(file.status)}`}
                      >
                        {file.status}
                      </span>
                    </div>
                    <div class="text-xs font-medium">{formatTotals(file.added, file.removed)}</div>
                  </div>
                {/each}
              </div>
            </details>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</section>
