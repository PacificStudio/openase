<script lang="ts">
  import { appStore } from '$lib/stores/app.svelte'
  import { connectEventStream } from '$lib/api/sse'
  import { listStatuses, listTickets, listWorkflows } from '$lib/api/openase'
  import { ApiError } from '$lib/api/client'
  import type { BoardColumn, BoardFilter, BoardTicket } from '../types'
  import BoardToolbar from './board-toolbar.svelte'
  import BoardView from './board-view.svelte'

  let filter = $state<BoardFilter>({ search: '' })
  let view = $state<'board' | 'list'>('board')
  let loading = $state(false)
  let error = $state('')
  let allColumns = $state<BoardColumn[]>([])
  let workflows = $state<string[]>([])

  let filteredColumns = $derived.by(() => {
    return allColumns.map((col) => {
      const filtered = col.tickets.filter((t) => {
        if (
          filter.search &&
          !t.title.toLowerCase().includes(filter.search.toLowerCase()) &&
          !t.identifier.toLowerCase().includes(filter.search.toLowerCase())
        )
          return false
        if (filter.workflow && t.workflowType !== filter.workflow) return false
        if (filter.agent && t.agentName !== filter.agent) return false
        if (filter.priority && t.priority !== filter.priority) return false
        if (filter.anomalyOnly && !t.anomaly) return false
        return true
      })
      return { ...col, tickets: filtered }
    })
  })

  $effect(() => {
    const projectId = appStore.currentProject?.id
    if (!projectId) {
      allColumns = []
      workflows = []
      return
    }

    let cancelled = false

    const load = async () => {
      loading = true
      error = ''

      try {
        const [statusPayload, ticketPayload, workflowPayload] = await Promise.all([
          listStatuses(projectId),
          listTickets(projectId),
          listWorkflows(projectId),
        ])
        if (cancelled) return

        const workflowTypeById = new Map(
          workflowPayload.workflows.map((workflow) => [workflow.id, workflow.type]),
        )

        workflows = Array.from(new Set(workflowPayload.workflows.map((workflow) => workflow.type)))

        allColumns = statusPayload.statuses
          .slice()
          .sort((left, right) => left.position - right.position)
          .map((status) => ({
            id: status.id,
            name: status.name,
            color: status.color || '#94a3b8',
            tickets: ticketPayload.tickets
              .filter((ticket) => ticket.status_id === status.id)
              .map((ticket) => ({
                id: ticket.id,
                identifier: ticket.identifier,
                title: ticket.title,
                priority: normalizePriority(ticket.priority),
                workflowType: ticket.workflow_id
                  ? (workflowTypeById.get(ticket.workflow_id) ?? undefined)
                  : undefined,
                updatedAt: ticket.created_at,
                labels: [],
                anomaly: inferAnomaly(ticket),
              })),
          }))
      } catch (caughtError) {
        if (cancelled) return
        error = caughtError instanceof ApiError ? caughtError.detail : 'Failed to load board data.'
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
        console.error('Board tickets stream error:', streamError)
      },
    })

    return () => {
      cancelled = true
      disconnect()
    }
  })

  function handleTicketClick(ticket: BoardTicket) {
    appStore.openRightPanel({ type: 'ticket', id: ticket.id })
  }

  function normalizePriority(priority: string): BoardTicket['priority'] {
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

  function inferAnomaly(ticket: {
    budget_usd: number
    cost_amount: number
    consecutive_errors: number
    retry_paused: boolean
  }): BoardTicket['anomaly'] | undefined {
    if (ticket.retry_paused) return 'retry'
    if (ticket.consecutive_errors > 0) return 'hook_failed'
    if (ticket.budget_usd > 0 && ticket.cost_amount >= ticket.budget_usd) return 'budget_exhausted'
    return undefined
  }
</script>

<div class="flex h-full flex-col gap-4">
  <BoardToolbar bind:filter bind:view {workflows} agents={[]} listEnabled={false} />
  {#if loading}
    <div class="text-muted-foreground flex flex-1 items-center justify-center text-sm">
      Loading board…
    </div>
  {:else if error}
    <div
      class="border-destructive/40 bg-destructive/10 text-destructive rounded-md border px-4 py-3 text-sm"
    >
      {error}
    </div>
  {:else}
    <BoardView columns={filteredColumns} onticketclick={handleTicketClick} />
  {/if}
</div>
