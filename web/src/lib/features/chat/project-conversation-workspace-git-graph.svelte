<script lang="ts">
  import { cn } from '$lib/utils'
  import type {
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceGitGraph,
    ProjectConversationWorkspaceGitGraphCommit,
    ProjectConversationWorkspaceGitRefLabel,
  } from '$lib/api/chat'
  import { buildWorkspaceGitGraphRows } from './project-conversation-workspace-git-graph'

  let {
    gitGraph = null,
    loading = false,
    error = '',
    selectedCommit = null,
    onSelectCommit,
    onCheckoutBranch,
  }: {
    gitGraph?: ProjectConversationWorkspaceGitGraph | null
    loading?: boolean
    error?: string
    selectedCommit?: ProjectConversationWorkspaceGitGraphCommit | null
    onSelectCommit?: (commitId: string) => void
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => void
  } = $props()

  const rows = $derived(gitGraph ? buildWorkspaceGitGraphRows(gitGraph.commits) : [])

  function labelTone(label: ProjectConversationWorkspaceGitRefLabel) {
    switch (label.scope) {
      case 'head':
        return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700'
      case 'remote_tracking_branch':
        return 'border-sky-500/30 bg-sky-500/10 text-sky-700'
      default:
        return label.current
          ? 'border-primary/30 bg-primary/10 text-primary'
          : 'border-border bg-muted text-foreground/80'
    }
  }
</script>

{#if loading}
  <div class="text-muted-foreground flex h-full items-center justify-center px-6 text-sm">
    Loading git graph...
  </div>
{:else if error}
  <div class="flex h-full items-center justify-center px-6">
    <div class="max-w-sm space-y-2 text-center">
      <p class="text-sm font-medium">Git graph unavailable</p>
      <p class="text-muted-foreground text-sm">{error}</p>
    </div>
  </div>
{:else if !gitGraph || gitGraph.commits.length === 0}
  <div class="text-muted-foreground flex h-full items-center justify-center px-6 text-sm">
    No commits available for this repo yet.
  </div>
{:else}
  <div class="flex h-full min-h-0 flex-col">
    <div class="grid min-h-0 flex-1 grid-cols-[minmax(0,1fr)_minmax(240px,320px)] overflow-hidden">
      <div class="border-border/60 min-h-0 overflow-auto border-r">
        {#each rows as row (row.commit.commitId)}
          <div
            role="button"
            tabindex="0"
            class={cn(
              'hover:bg-muted/40 border-border/40 flex w-full items-stretch gap-3 border-b px-3 py-2 text-left transition-colors',
              selectedCommit?.commitId === row.commit.commitId && 'bg-primary/5',
            )}
            onclick={() => onSelectCommit?.(row.commit.commitId)}
            onkeydown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault()
                onSelectCommit?.(row.commit.commitId)
              }
            }}
            data-testid={`workspace-git-graph-commit-${row.commit.shortCommitId}`}
          >
            <div class="shrink-0 pt-1.5">
              <svg
                width={Math.max(1, row.laneCount) * 14}
                height="28"
                viewBox={`0 0 ${Math.max(1, row.laneCount) * 14} 28`}
              >
                {#each Array.from({ length: row.laneCount }, (_, index) => index) as lane}
                  {@const x = lane * 14 + 7}
                  {#if row.activeBefore.includes(lane) || row.activeAfter.includes(lane)}
                    <line
                      x1={x}
                      y1="0"
                      x2={x}
                      y2="28"
                      stroke="currentColor"
                      stroke-opacity="0.22"
                    />
                  {/if}
                {/each}
                {#each row.parentColumns as parentColumn}
                  <line
                    x1={row.column * 14 + 7}
                    y1="14"
                    x2={parentColumn * 14 + 7}
                    y2="28"
                    stroke="currentColor"
                    stroke-opacity="0.5"
                  />
                {/each}
                <circle
                  cx={row.column * 14 + 7}
                  cy="14"
                  r={row.commit.head ? 4 : 3.5}
                  class={row.commit.head ? 'fill-primary' : 'fill-foreground/70'}
                />
              </svg>
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium">{row.commit.subject}</span>
                <span class="text-muted-foreground font-mono text-[11px]">
                  {row.commit.shortCommitId}
                </span>
              </div>
              <div class="mt-1 flex flex-wrap gap-1.5">
                {#each row.commit.labels as label (label.scope + label.name)}
                  <button
                    type="button"
                    class={cn(
                      'rounded-full border px-2 py-0.5 text-[10px] font-medium',
                      labelTone(label),
                    )}
                    onclick={(event) => {
                      event.stopPropagation()
                      if (label.scope === 'local_branch') {
                        onCheckoutBranch?.({
                          targetKind: 'local_branch',
                          targetName: label.name,
                          createTrackingBranch: false,
                        })
                      } else if (label.scope === 'remote_tracking_branch') {
                        onCheckoutBranch?.({
                          targetKind: 'remote_tracking_branch',
                          targetName: label.name,
                          createTrackingBranch: true,
                        })
                      }
                    }}
                    disabled={label.scope === 'head'}
                  >
                    {label.name}
                  </button>
                {/each}
              </div>
              <div class="text-muted-foreground mt-1 text-[11px]">
                {row.commit.authorName} · {new Date(row.commit.authoredAt).toLocaleString()}
              </div>
            </div>
          </div>
        {/each}
      </div>

      <div class="min-h-0 overflow-auto px-4 py-3">
        {#if selectedCommit}
          <div class="space-y-3">
            <div>
              <p class="text-muted-foreground text-[11px] font-semibold tracking-wide uppercase">
                Commit
              </p>
              <p class="mt-1 text-sm font-medium">{selectedCommit.subject}</p>
            </div>
            <dl class="space-y-2 text-sm">
              <div>
                <dt class="text-muted-foreground text-[11px] tracking-wide uppercase">SHA</dt>
                <dd class="mt-1 font-mono text-[12px]">{selectedCommit.commitId}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground text-[11px] tracking-wide uppercase">Author</dt>
                <dd class="mt-1">{selectedCommit.authorName}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground text-[11px] tracking-wide uppercase">Authored</dt>
                <dd class="mt-1">{new Date(selectedCommit.authoredAt).toLocaleString()}</dd>
              </div>
              <div>
                <dt class="text-muted-foreground text-[11px] tracking-wide uppercase">Parents</dt>
                <dd class="mt-1 font-mono text-[12px]">
                  {selectedCommit.parentIds.length > 0
                    ? selectedCommit.parentIds.join(', ')
                    : 'Root commit'}
                </dd>
              </div>
              <div>
                <dt class="text-muted-foreground text-[11px] tracking-wide uppercase">Labels</dt>
                <dd class="mt-2 flex flex-wrap gap-1.5">
                  {#if selectedCommit.labels.length === 0}
                    <span class="text-muted-foreground">No labels</span>
                  {:else}
                    {#each selectedCommit.labels as label (label.scope + label.name)}
                      <span
                        class={cn(
                          'rounded-full border px-2 py-0.5 text-[10px] font-medium',
                          labelTone(label),
                        )}
                      >
                        {label.name}
                      </span>
                    {/each}
                  {/if}
                </dd>
              </div>
            </dl>
          </div>
        {:else}
          <div class="text-muted-foreground flex h-full items-center justify-center text-sm">
            Select a commit to inspect it.
          </div>
        {/if}
      </div>
    </div>
  </div>
{/if}
