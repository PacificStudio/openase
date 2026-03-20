<script lang="ts">
  import { Badge } from '$ui/badge'
  import GitBranch from '@lucide/svelte/icons/git-branch'
  import GitPullRequest from '@lucide/svelte/icons/git-pull-request'
  import CircleCheck from '@lucide/svelte/icons/circle-check'
  import CircleX from '@lucide/svelte/icons/circle-x'
  import Loader from '@lucide/svelte/icons/loader'
  import Circle from '@lucide/svelte/icons/circle'
  import ExternalLink from '@lucide/svelte/icons/external-link'
  import { cn } from '$lib/utils'
  import type { TicketDetail } from '../types'

  let { ticket }: { ticket: TicketDetail } = $props()

  const prStatusConfig: Record<string, { class: string; label: string }> = {
    open: { class: 'text-green-400', label: 'Open' },
    merged: { class: 'text-purple-400', label: 'Merged' },
    closed: { class: 'text-red-400', label: 'Closed' },
    draft: { class: 'text-muted-foreground', label: 'Draft' },
  }

  const ciStatusConfig: Record<string, { icon: typeof CircleCheck; class: string }> = {
    pass: { icon: CircleCheck, class: 'text-green-400' },
    fail: { icon: CircleX, class: 'text-red-400' },
    running: { icon: Loader, class: 'text-yellow-400 animate-spin' },
    pending: { icon: Circle, class: 'text-muted-foreground' },
  }
</script>

<div class="flex flex-col gap-2 px-5 py-3">
  <span class="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
    Repositories & PRs
  </span>

  {#if ticket.repoScopes.length === 0}
    <p class="py-4 text-center text-xs text-muted-foreground">No repositories linked</p>
  {/if}

  {#each ticket.repoScopes as scope}
    <div class="flex flex-col gap-1.5 rounded-md border border-border bg-muted/30 p-3">
      <div class="flex items-center gap-2 text-xs font-medium text-foreground">
        <GitBranch class="size-3.5 text-muted-foreground" />
        <span class="truncate">{scope.repoName}</span>
      </div>

      <div class="flex items-center gap-2 pl-5 text-[11px] text-muted-foreground">
        <code class="rounded bg-muted px-1 py-0.5 font-mono text-[10px]">
          {scope.branchName}
        </code>
      </div>

      {#if scope.prUrl}
        <div class="flex items-center gap-2 pl-5">
          <GitPullRequest class={cn(
            'size-3.5',
            prStatusConfig[scope.prStatus ?? 'open']?.class ?? 'text-muted-foreground',
          )} />
          <a
            href={scope.prUrl}
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center gap-1 text-[11px] text-blue-400 hover:underline"
          >
            Pull Request
            <ExternalLink class="size-2.5" />
          </a>
          {#if scope.prStatus}
            <Badge
              variant="outline"
              class={cn(
                'text-[10px] py-0 h-4',
                prStatusConfig[scope.prStatus]?.class,
              )}
            >
              {prStatusConfig[scope.prStatus]?.label ?? scope.prStatus}
            </Badge>
          {/if}
          {#if scope.ciStatus}
            {@const ci = ciStatusConfig[scope.ciStatus]}
            {#if ci}
              <svelte:component this={ci.icon} class={cn('size-3.5 ml-auto', ci.class)} />
            {/if}
          {/if}
        </div>
      {/if}
    </div>
  {/each}
</div>
