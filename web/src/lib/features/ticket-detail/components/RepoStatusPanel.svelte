<script lang="ts">
  import { ExternalLink, GitBranch } from '@lucide/svelte'
  import ScrollPane from '$lib/components/layout/ScrollPane.svelte'
  import SurfacePanel from '$lib/components/layout/SurfacePanel.svelte'
  import { Badge } from '$lib/components/ui/badge'
  import type { TicketRepoScope } from '../types'

  let {
    repoScopes = [],
  }: {
    repoScopes?: TicketRepoScope[]
  } = $props()

  function statusBadgeClass(status: string) {
    switch (status) {
      case 'merged':
      case 'passing':
      case 'done':
        return 'border-emerald-500/25 bg-emerald-500/10 text-emerald-700'
      case 'open':
      case 'pending':
        return 'border-sky-500/25 bg-sky-500/10 text-sky-700'
      case 'changes_requested':
      case 'failing':
        return 'border-amber-500/25 bg-amber-500/10 text-amber-700'
      case 'closed':
      case 'failed':
        return 'border-rose-500/25 bg-rose-500/10 text-rose-700'
      default:
        return 'border-border/80 bg-background text-muted-foreground'
    }
  }
</script>

<SurfacePanel class="h-full">
  {#snippet header()}
    <div>
      <div class="flex items-center gap-2 text-sm font-semibold">
      <GitBranch class="size-4" />
      <span>Repo / PR Status</span>
      </div>
      <p class="text-muted-foreground mt-1 text-xs leading-5">
        Each repo scope tracks branch, PR URL, review status, and CI health for this ticket.
      </p>
    </div>
  {/snippet}

  <ScrollPane class="max-h-[22rem] space-y-3 px-4 py-4">
    {#if repoScopes.length === 0}
      <div
        class="border-border/80 bg-muted/30 text-muted-foreground rounded-3xl border border-dashed px-4 py-6 text-sm"
      >
        No repo scopes are linked to this ticket yet.
      </div>
    {:else}
      {#each repoScopes as scope}
        <div class="border-border/70 bg-background/75 rounded-3xl border p-4">
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div class="space-y-2">
              <div class="flex flex-wrap items-center gap-2">
                <p class="text-foreground text-sm font-semibold">
                  {scope.repo?.name ?? scope.repo_id}
                </p>
                {#if scope.is_primary_scope}
                  <Badge variant="secondary">primary</Badge>
                {/if}
                <span
                  class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${statusBadgeClass(scope.pr_status)}`}
                >
                  PR {scope.pr_status}
                </span>
                <span
                  class={`inline-flex rounded-full border px-2.5 py-1 text-[11px] font-medium ${statusBadgeClass(scope.ci_status)}`}
                >
                  CI {scope.ci_status}
                </span>
              </div>
              <p class="text-muted-foreground text-xs">
                Branch `{scope.branch_name}` on {scope.repo?.default_branch ??
                  'unknown default branch'}
              </p>
            </div>

            {#if scope.pull_request_url}
              <a
                href={scope.pull_request_url}
                target="_blank"
                rel="noreferrer"
                class="border-border/70 bg-background text-foreground hover:border-foreground/20 inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium transition"
              >
                <ExternalLink class="size-3.5" />
                Open PR
              </a>
            {:else}
              <span class="text-muted-foreground text-xs">No PR URL yet</span>
            {/if}
          </div>

          {#if scope.repo?.repository_url}
            <p class="text-muted-foreground mt-3 text-xs">{scope.repo.repository_url}</p>
          {/if}
        </div>
      {/each}
    {/if}
  </ScrollPane>
</SurfacePanel>
