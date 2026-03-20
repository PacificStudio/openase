<script lang="ts">
  import type { BoardColumn, BoardFilter, BoardTicket } from '../types'
  import BoardToolbar from './board-toolbar.svelte'
  import BoardView from './board-view.svelte'

  let filter = $state<BoardFilter>({ search: '' })
  let view = $state<'board' | 'list'>('board')

  const now = Date.now()
  const h = (hours: number) => new Date(now - hours * 3_600_000).toISOString()

  const allColumns: BoardColumn[] = [
    {
      id: 'backlog',
      name: 'Backlog',
      color: '#6b7280',
      tickets: [
        {
          id: '1',
          identifier: 'ASE-38',
          title: 'Add rate limiting to public API endpoints',
          priority: 'medium',
          workflowType: 'coding',
          updatedAt: h(72),
          labels: ['api'],
        },
        {
          id: '2',
          identifier: 'ASE-39',
          title: 'Write integration tests for webhook delivery',
          priority: 'low',
          workflowType: 'test',
          updatedAt: h(48),
        },
        {
          id: '3',
          identifier: 'ASE-40',
          title: 'Audit dependency versions for CVEs',
          priority: 'high',
          workflowType: 'security',
          updatedAt: h(24),
        },
      ],
    },
    {
      id: 'todo',
      name: 'Todo',
      color: '#a855f7',
      tickets: [
        {
          id: '4',
          identifier: 'ASE-41',
          title: 'Implement SSO login with SAML provider',
          priority: 'high',
          workflowType: 'coding',
          agentName: 'Coder-1',
          updatedAt: h(6),
        },
        {
          id: '5',
          identifier: 'ASE-42',
          title: 'Add pagination to workflow history endpoint',
          priority: 'medium',
          workflowType: 'coding',
          updatedAt: h(12),
        },
        {
          id: '6',
          identifier: 'ASE-43',
          title: 'Create E2E test suite for onboarding flow',
          priority: 'medium',
          workflowType: 'test',
          updatedAt: h(8),
        },
        {
          id: '7',
          identifier: 'ASE-44',
          title: 'Migrate legacy cron jobs to event-driven triggers',
          priority: 'low',
          workflowType: 'coding',
          updatedAt: h(30),
        },
      ],
    },
    {
      id: 'in_progress',
      name: 'In Progress',
      color: '#f59e0b',
      wipInfo: '3 / 5 WIP',
      tickets: [
        {
          id: '8',
          identifier: 'ASE-45',
          title: 'Refactor notification service to use message queue',
          priority: 'high',
          workflowType: 'coding',
          agentName: 'Coder-2',
          prCount: 1,
          prStatus: 'draft',
          updatedAt: h(1),
        },
        {
          id: '9',
          identifier: 'ASE-46',
          title: 'Fix flaky test in CI for auth module',
          priority: 'urgent',
          workflowType: 'test',
          agentName: 'Tester-1',
          anomaly: 'retry',
          updatedAt: h(0.5),
        },
        {
          id: '10',
          identifier: 'ASE-47',
          title: 'Implement role-based access control for admin panel',
          priority: 'high',
          workflowType: 'coding',
          agentName: 'Coder-1',
          prCount: 2,
          prStatus: 'open',
          updatedAt: h(2),
        },
      ],
    },
    {
      id: 'in_review',
      name: 'In Review',
      color: '#3b82f6',
      tickets: [
        {
          id: '11',
          identifier: 'ASE-48',
          title: 'Add OpenTelemetry tracing to core services',
          priority: 'medium',
          workflowType: 'coding',
          agentName: 'Coder-2',
          prCount: 1,
          prStatus: 'approved',
          updatedAt: h(3),
        },
        {
          id: '12',
          identifier: 'ASE-49',
          title: 'Security scan: validate input sanitization',
          priority: 'high',
          workflowType: 'security',
          agentName: 'SecBot',
          anomaly: 'hook_failed',
          prCount: 1,
          prStatus: 'changes requested',
          updatedAt: h(1.5),
        },
        {
          id: '13',
          identifier: 'ASE-50',
          title: 'Update API docs for v2 billing endpoints',
          priority: 'low',
          workflowType: 'review',
          agentName: 'Coder-3',
          prCount: 1,
          prStatus: 'open',
          updatedAt: h(5),
        },
        {
          id: '14',
          identifier: 'ASE-51',
          title: 'Add budget guardrails to agent orchestrator',
          priority: 'urgent',
          workflowType: 'coding',
          agentName: 'Coder-1',
          anomaly: 'awaiting_approval',
          prCount: 1,
          prStatus: 'open',
          updatedAt: h(0.3),
        },
      ],
    },
    {
      id: 'done',
      name: 'Done',
      color: '#22c55e',
      tickets: [
        {
          id: '15',
          identifier: 'ASE-33',
          title: 'Set up GitHub Actions for automated releases',
          priority: 'medium',
          workflowType: 'deploy',
          agentName: 'Coder-2',
          prCount: 1,
          prStatus: 'merged',
          updatedAt: h(10),
        },
        {
          id: '16',
          identifier: 'ASE-34',
          title: 'Fix memory leak in WebSocket connection handler',
          priority: 'urgent',
          workflowType: 'coding',
          agentName: 'Coder-1',
          prCount: 1,
          prStatus: 'merged',
          updatedAt: h(14),
        },
        {
          id: '17',
          identifier: 'ASE-35',
          title: 'Add snapshot tests for dashboard components',
          priority: 'low',
          workflowType: 'test',
          agentName: 'Tester-1',
          prCount: 1,
          prStatus: 'merged',
          updatedAt: h(20),
        },
        {
          id: '18',
          identifier: 'ASE-36',
          title: 'Implement retry logic for external API calls',
          priority: 'medium',
          workflowType: 'coding',
          agentName: 'Coder-3',
          prCount: 2,
          prStatus: 'merged',
          updatedAt: h(18),
        },
        {
          id: '19',
          identifier: 'ASE-37',
          title: 'Upgrade Node.js runtime to v22 LTS',
          priority: 'low',
          workflowType: 'deploy',
          agentName: 'Coder-2',
          prCount: 1,
          prStatus: 'merged',
          updatedAt: h(36),
        },
      ],
    },
    {
      id: 'cancelled',
      name: 'Cancelled',
      color: '#ef4444',
      tickets: [
        {
          id: '20',
          identifier: 'ASE-30',
          title: 'Evaluate alternative vector DB for embeddings',
          priority: 'low',
          workflowType: 'review',
          anomaly: 'budget_exhausted',
          updatedAt: h(96),
        },
      ],
    },
  ]

  const workflows = ['coding', 'test', 'security', 'review', 'deploy']
  const agents = ['Coder-1', 'Coder-2', 'Coder-3', 'Tester-1', 'SecBot']

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

  function handleTicketClick(ticket: BoardTicket) {
    console.log('Ticket clicked:', ticket.identifier, ticket.title)
  }
</script>

<div class="flex h-full flex-col gap-4">
  <BoardToolbar bind:filter bind:view {workflows} {agents} />
  <BoardView columns={filteredColumns} onticketclick={handleTicketClick} />
</div>
