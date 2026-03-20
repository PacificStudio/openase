<script lang="ts">
  import PageHeader from '$lib/components/layout/page-header.svelte'
  import { Button } from '$ui/button'
  import { Badge } from '$ui/badge'
  import { cn, formatRelativeTime } from '$lib/utils'
  import {
    Search,
    Filter,
    ArrowUpDown,
    Circle,
    CircleDot,
  } from '@lucide/svelte'
  import { Input } from '$ui/input'

  const mockTickets = [
    { id: '1', identifier: 'ASE-42', title: 'Fix login validation edge case', status: 'In Progress', priority: 'high' as const, type: 'bugfix', workflow: 'coding', agent: 'claude-01', updatedAt: '2026-03-20T08:30:00Z' },
    { id: '2', identifier: 'ASE-43', title: 'Add audit logging to API endpoints', status: 'Todo', priority: 'medium' as const, type: 'feature', workflow: 'coding', agent: '', updatedAt: '2026-03-20T07:15:00Z' },
    { id: '3', identifier: 'ASE-44', title: 'Update deployment pipeline for v2', status: 'In Review', priority: 'high' as const, type: 'chore', workflow: 'deploy', agent: 'codex-01', updatedAt: '2026-03-20T06:00:00Z' },
    { id: '4', identifier: 'ASE-45', title: 'Refactor user service to use repository pattern', status: 'Done', priority: 'low' as const, type: 'refactor', workflow: 'coding', agent: 'claude-02', updatedAt: '2026-03-19T22:00:00Z' },
    { id: '5', identifier: 'ASE-46', title: 'Write integration tests for payment module', status: 'Todo', priority: 'urgent' as const, type: 'feature', workflow: 'test', agent: '', updatedAt: '2026-03-20T09:00:00Z' },
    { id: '6', identifier: 'ASE-47', title: 'Security scan for dependency vulnerabilities', status: 'Backlog', priority: 'medium' as const, type: 'chore', workflow: 'security', agent: '', updatedAt: '2026-03-19T14:00:00Z' },
  ]

  const priorityColors: Record<string, string> = {
    urgent: 'bg-destructive',
    high: 'bg-warning',
    medium: 'bg-info',
    low: 'bg-muted-foreground',
  }
</script>

<svelte:head>
  <title>Tickets - OpenASE</title>
</svelte:head>

{#snippet actions()}
  <Button size="sm">New Ticket</Button>
{/snippet}

<PageHeader title="Tickets" description="All tickets in this project" {actions} />

<div class="px-6">
  <div class="mb-4 flex items-center gap-3">
    <div class="relative flex-1 max-w-sm">
      <Search class="absolute left-2.5 top-2.5 size-3.5 text-muted-foreground" />
      <Input placeholder="Search tickets..." class="pl-8 h-9 text-sm" />
    </div>
    <Button variant="outline" size="sm" class="gap-1.5">
      <Filter class="size-3.5" />
      Filter
    </Button>
    <Button variant="outline" size="sm" class="gap-1.5">
      <ArrowUpDown class="size-3.5" />
      Sort
    </Button>
  </div>

  <div class="rounded-md border border-border">
    <table class="w-full text-sm">
      <thead>
        <tr class="border-b border-border text-left text-xs text-muted-foreground">
          <th class="px-4 py-2.5 font-medium">Ticket</th>
          <th class="px-4 py-2.5 font-medium">Status</th>
          <th class="px-4 py-2.5 font-medium">Priority</th>
          <th class="px-4 py-2.5 font-medium">Workflow</th>
          <th class="px-4 py-2.5 font-medium">Agent</th>
          <th class="px-4 py-2.5 font-medium text-right">Updated</th>
        </tr>
      </thead>
      <tbody>
        {#each mockTickets as ticket (ticket.id)}
          <tr class="border-b border-border last:border-0 hover:bg-muted/50 cursor-pointer transition-colors">
            <td class="px-4 py-3">
              <div class="flex items-center gap-2">
                <span class="text-xs text-muted-foreground font-mono">{ticket.identifier}</span>
                <span class="text-foreground">{ticket.title}</span>
              </div>
            </td>
            <td class="px-4 py-3">
              <Badge variant="outline" class="text-xs">{ticket.status}</Badge>
            </td>
            <td class="px-4 py-3">
              <div class="flex items-center gap-1.5">
                <span class={cn('size-2 rounded-full', priorityColors[ticket.priority])} />
                <span class="text-xs text-muted-foreground capitalize">{ticket.priority}</span>
              </div>
            </td>
            <td class="px-4 py-3">
              <span class="text-xs text-muted-foreground">{ticket.workflow}</span>
            </td>
            <td class="px-4 py-3">
              {#if ticket.agent}
                <span class="text-xs text-foreground">{ticket.agent}</span>
              {:else}
                <span class="text-xs text-muted-foreground/50">—</span>
              {/if}
            </td>
            <td class="px-4 py-3 text-right">
              <span class="text-xs text-muted-foreground">{formatRelativeTime(ticket.updatedAt)}</span>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
