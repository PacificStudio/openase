<script lang="ts">
  import { connectEventStream } from '$lib/api/sse'
  import { ApiError } from '$lib/api/client'
  import { capabilityCatalog } from '$lib/features/capabilities'
  import { listTickets, listWorkflows } from '$lib/api/openase'
  import PageHeader from '$lib/components/layout/page-header.svelte'
  import { appStore } from '$lib/stores/app.svelte'
  import { cn, formatRelativeTime } from '$lib/utils'
  import { Badge } from '$ui/badge'
  import { Button } from '$ui/button'
  import { Input } from '$ui/input'
  import * as Select from '$ui/select'
  import { ArrowDownAZ, Search } from '@lucide/svelte'

  const priorityColors: Record<string, string> = {
    urgent: 'bg-destructive',
    high: 'bg-warning',
    medium: 'bg-info',
    low: 'bg-muted-foreground',
  }

  type TicketRow = {
    id: string
    identifier: string
    title: string
    status: string
    priority: 'urgent' | 'high' | 'medium' | 'low'
    workflow: string
    updatedAt: string
  }

  let loading = $state(false)
  let error = $state('')
  let tickets = $state<TicketRow[]>([])
  let searchQuery = $state('')
  let priorityFilter = $state('all')
  let sortOrder = $state<'updated' | 'priority'>('updated')
  const newTicketCapability = capabilityCatalog.newTicket

  const filteredTickets = $derived.by(() => {
    const query = searchQuery.toLowerCase()
    const filtered = tickets.filter((ticket) => {
      if (priorityFilter !== 'all' && ticket.priority !== priorityFilter) return false
      if (!query) return true
      return (
        ticket.identifier.toLowerCase().includes(query) ||
        ticket.title.toLowerCase().includes(query) ||
        ticket.status.toLowerCase().includes(query)
      )
    })

    return filtered.sort((left, right) => {
      if (sortOrder === 'priority') {
        return priorityWeight(right.priority) - priorityWeight(left.priority)
      }

      return new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime()
    })
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      tickets = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [ticketPayload, workflowPayload] = await Promise.all([
          listTickets(projectId),
          listWorkflows(projectId),
        ])
        if (cancelled) return

        const workflowMap = new Map(
          workflowPayload.workflows.map((workflow) => [workflow.id, workflow.type]),
        )

        tickets = ticketPayload.tickets.map((ticket) => ({
          id: ticket.id,
          identifier: ticket.identifier,
          title: ticket.title,
          status: ticket.status_name,
          priority: normalizePriority(ticket.priority),
          workflow: ticket.workflow_id
            ? (workflowMap.get(ticket.workflow_id) ?? 'Unassigned')
            : 'Unassigned',
          updatedAt: ticket.created_at,
        }))
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load tickets.'
      } finally {
        if (!cancelled) {
          loading = false
        }
      }
    }

    void load()

    const disconnect = connectEventStream(`/api/v1/projects/${projectId}/tickets/stream`, {
      onEvent: () => {
        void load()
      },
      onError: (streamError) => {
        console.error('Tickets stream error:', streamError)
      },
    })

    return () => {
      cancelled = true
      disconnect()
    }
  })

  function normalizePriority(priority: string): TicketRow['priority'] {
    if (
      priority === 'urgent' ||
      priority === 'high' ||
      priority === 'medium' ||
      priority === 'low'
    ) {
      return priority
    }

    return 'medium'
  }

  function priorityWeight(priority: TicketRow['priority']) {
    if (priority === 'urgent') return 4
    if (priority === 'high') return 3
    if (priority === 'medium') return 2
    return 1
  }
</script>

{#snippet actions()}
  <Button size="sm" disabled title={newTicketCapability.summary}>New Ticket</Button>
{/snippet}

<PageHeader title="Tickets" description="All tickets in this project" {actions} />

<div class="px-6">
  <div class="mb-4 flex items-center gap-3">
    <div class="relative max-w-sm flex-1">
      <Search class="text-muted-foreground absolute top-2.5 left-2.5 size-3.5" />
      <Input placeholder="Search tickets..." class="h-9 pl-8 text-sm" bind:value={searchQuery} />
    </div>
    <Select.Root
      type="single"
      onValueChange={(value) => {
        priorityFilter = value || 'all'
      }}
    >
      <Select.Trigger class="w-40">
        {priorityFilter === 'all' ? 'All priorities' : priorityFilter}
      </Select.Trigger>
      <Select.Content>
        <Select.Item value="all">All priorities</Select.Item>
        <Select.Item value="urgent">Urgent</Select.Item>
        <Select.Item value="high">High</Select.Item>
        <Select.Item value="medium">Medium</Select.Item>
        <Select.Item value="low">Low</Select.Item>
      </Select.Content>
    </Select.Root>
    <Button
      variant="outline"
      size="sm"
      class="gap-1.5"
      onclick={() => {
        sortOrder = sortOrder === 'updated' ? 'priority' : 'updated'
      }}
    >
      <ArrowDownAZ class="size-3.5" />
      {sortOrder === 'updated' ? 'Sort: Updated' : 'Sort: Priority'}
    </Button>
  </div>

  {#if loading}
    <div
      class="border-border bg-card text-muted-foreground rounded-md border px-4 py-10 text-center text-sm"
    >
      Loading tickets…
    </div>
  {:else if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    <div class="border-border rounded-md border">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-border text-muted-foreground border-b text-left text-xs">
            <th class="px-4 py-2.5 font-medium">Ticket</th>
            <th class="px-4 py-2.5 font-medium">Status</th>
            <th class="px-4 py-2.5 font-medium">Priority</th>
            <th class="px-4 py-2.5 font-medium">Workflow</th>
            <th class="px-4 py-2.5 text-right font-medium">Updated</th>
          </tr>
        </thead>
        <tbody>
          {#each filteredTickets as ticket (ticket.id)}
            <tr
              class="border-border hover:bg-muted/50 cursor-pointer border-b transition-colors last:border-0"
              onclick={() => appStore.openRightPanel({ type: 'ticket', id: ticket.id })}
            >
              <td class="px-4 py-3">
                <div class="flex items-center gap-2">
                  <span class="text-muted-foreground font-mono text-xs">{ticket.identifier}</span>
                  <span class="text-foreground">{ticket.title}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <Badge variant="outline" class="text-xs">{ticket.status}</Badge>
              </td>
              <td class="px-4 py-3">
                <div class="flex items-center gap-1.5">
                  <span class={cn('size-2 rounded-full', priorityColors[ticket.priority])}></span>
                  <span class="text-muted-foreground text-xs capitalize">{ticket.priority}</span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span class="text-muted-foreground text-xs">{ticket.workflow}</span>
              </td>
              <td class="px-4 py-3 text-right">
                <span class="text-muted-foreground text-xs"
                  >{formatRelativeTime(ticket.updatedAt)}</span
                >
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
