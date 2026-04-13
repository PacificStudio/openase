<script lang="ts">
  import { cn } from '$lib/utils'
  import * as DropdownMenu from '$ui/dropdown-menu'
  import type {
    ProjectConversationWorkspaceBranchScope,
    ProjectConversationWorkspaceGitGraph,
    ProjectConversationWorkspaceGitGraphCommit,
    ProjectConversationWorkspaceGitRefLabel,
  } from '$lib/api/chat'
  import { buildWorkspaceGitGraphRows } from './project-conversation-workspace-git-graph'
  import { Copy, GitBranchPlus } from '@lucide/svelte'

  let {
    gitGraph = null,
    loading = false,
    error = '',
    onCheckoutBranch,
    onCreateBranch,
  }: {
    gitGraph?: ProjectConversationWorkspaceGitGraph | null
    loading?: boolean
    error?: string
    onCheckoutBranch?: (request: {
      targetKind: ProjectConversationWorkspaceBranchScope
      targetName: string
      createTrackingBranch: boolean
      localBranchName?: string
    }) => void
    onCreateBranch?: (commitId: string) => void
  } = $props()

  const rows = $derived(gitGraph ? buildWorkspaceGitGraphRows(gitGraph.commits) : [])

  let contextMenu = $state<{
    commit: ProjectConversationWorkspaceGitGraphCommit
    x: number
    y: number
  } | null>(null)

  function labelColor(label: ProjectConversationWorkspaceGitRefLabel) {
    if (label.scope === 'head') return 'text-emerald-600'
    if (label.scope === 'remote_tracking_branch') return 'text-sky-600'
    return label.current ? 'text-primary' : 'text-foreground/70'
  }

  function formatTime(iso: string): string {
    const d = new Date(iso)
    const now = new Date()
    const diffMs = now.getTime() - d.getTime()
    const diffMin = Math.floor(diffMs / 60000)
    if (diffMin < 1) return 'now'
    if (diffMin < 60) return `${diffMin}m`
    const diffHr = Math.floor(diffMin / 60)
    if (diffHr < 24) return `${diffHr}h`
    const diffDay = Math.floor(diffHr / 24)
    if (diffDay < 30) return `${diffDay}d`
    return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  }

  async function copyText(text: string) {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      // ignore
    }
  }

  function handleContextMenu(
    event: MouseEvent,
    commit: ProjectConversationWorkspaceGitGraphCommit,
  ) {
    event.preventDefault()
    event.stopPropagation()
    contextMenu = { commit, x: event.clientX, y: event.clientY }
  }
</script>

{#if loading}
  <div class="text-muted-foreground px-4 py-3 text-[11px]">Loading…</div>
{:else if error}
  <div class="text-destructive px-4 py-2 text-[11px]">{error}</div>
{:else if !gitGraph || rows.length === 0}
  <div class="text-muted-foreground px-4 py-3 text-[11px]">No commits.</div>
{:else}
  {#each rows as row (row.commit.commitId)}
    {@const laneW = 14}
    {@const graphW = Math.max(1, row.laneCount) * laneW}
    <div
      class="hover:bg-muted/40 flex min-h-7 items-stretch gap-2 px-2"
      title="{row.commit.authorName} · {new Date(row.commit.authoredAt).toLocaleString()}\n{row
        .commit.commitId}"
      oncontextmenu={(e) => handleContextMenu(e, row.commit)}
      role="button"
      tabindex="0"
    >
      <div class="relative shrink-0" style="width: {graphW}px">
        {#each Array.from({ length: row.laneCount }, (_, i) => i) as lane}
          {#if row.activeBefore.includes(lane) || row.activeAfter.includes(lane)}
            <div
              class="bg-foreground/25 absolute top-0 bottom-0 w-px"
              style="left: {lane * laneW + laneW / 2}px"
            ></div>
          {/if}
        {/each}
        {#if row.parentColumns.length > 0}
          <svg
            class="pointer-events-none absolute inset-0 h-full w-full overflow-visible"
            viewBox={`0 0 ${graphW} 100`}
            preserveAspectRatio="none"
          >
            {#each row.parentColumns as pc}
              <line
                x1={row.column * laneW + laneW / 2}
                y1="50"
                x2={pc * laneW + laneW / 2}
                y2="100"
                stroke="currentColor"
                stroke-opacity="0.25"
                vector-effect="non-scaling-stroke"
              />
            {/each}
          </svg>
        {/if}
        <div
          class={cn(
            'absolute top-1/2 -translate-y-1/2 rounded-full',
            row.commit.head ? 'bg-primary size-2' : 'bg-foreground/60 size-[7px]',
          )}
          style="left: {row.column * laneW + laneW / 2 - (row.commit.head ? 4 : 3.5)}px"
        ></div>
      </div>

      <div class="flex min-w-0 flex-1 flex-col justify-center py-1">
        <div class="flex items-center gap-1.5">
          <span class="min-w-0 truncate text-[11px]">{row.commit.subject}</span>
          <span class="text-muted-foreground/50 shrink-0 text-[9px]">
            {formatTime(row.commit.authoredAt)}
          </span>
        </div>
        {#if row.commit.labels.length > 0}
          <div class="flex flex-wrap gap-1 pt-0.5">
            {#each row.commit.labels as label (label.scope + label.name)}
              <button
                type="button"
                class={cn(
                  'truncate rounded px-1 text-[9px] font-medium',
                  label.scope === 'head'
                    ? 'bg-emerald-500/10'
                    : label.scope === 'remote_tracking_branch'
                      ? 'bg-sky-500/10'
                      : label.current
                        ? 'bg-primary/10'
                        : 'bg-muted',
                  labelColor(label),
                )}
                disabled={label.scope === 'head'}
                onclick={(e) => {
                  e.stopPropagation()
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
              >
                {label.name}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>
  {/each}
{/if}

{#if contextMenu}
  {@const commit = contextMenu.commit}
  <DropdownMenu.Root
    open={true}
    onOpenChange={(next) => {
      if (!next) contextMenu = null
    }}
  >
    <DropdownMenu.Content
      class="w-52"
      style="position: fixed; left: {contextMenu.x}px; top: {contextMenu.y}px;"
    >
      <DropdownMenu.Item
        onclick={() => {
          copyText(commit.commitId)
          contextMenu = null
        }}
      >
        <Copy class="size-3.5" />
        <span>Copy commit hash</span>
      </DropdownMenu.Item>
      <DropdownMenu.Item
        onclick={() => {
          copyText(commit.shortCommitId)
          contextMenu = null
        }}
      >
        <Copy class="size-3.5" />
        <span>Copy short hash</span>
      </DropdownMenu.Item>
      <DropdownMenu.Item
        onclick={() => {
          copyText(commit.subject)
          contextMenu = null
        }}
      >
        <Copy class="size-3.5" />
        <span>Copy commit message</span>
      </DropdownMenu.Item>
      <DropdownMenu.Separator />
      <DropdownMenu.Item
        onclick={() => {
          onCreateBranch?.(commit.commitId)
          contextMenu = null
        }}
      >
        <GitBranchPlus class="size-3.5" />
        <span>Create branch here…</span>
      </DropdownMenu.Item>
    </DropdownMenu.Content>
  </DropdownMenu.Root>
{/if}
