<script lang="ts">
  import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetDescription } from '$ui/sheet'
  import { Tabs, TabsContent, TabsList, TabsTrigger } from '$ui/tabs'
  import TicketHeader from './ticket-header.svelte'
  import TicketSummary from './ticket-summary.svelte'
  import TicketRepos from './ticket-repos.svelte'
  import TicketHooks from './ticket-hooks.svelte'
  import TicketActivityList from './ticket-activity.svelte'
  import type { TicketDetail, HookExecution, TicketActivity } from '../types'

  const MOCK_TICKET: TicketDetail = {
    id: 'tk_01',
    identifier: 'ASE-42',
    title: 'Implement OAuth2 token refresh flow for GitHub connector',
    description:
      'The current GitHub connector does not handle token expiry gracefully. We need to implement automatic token refresh using the refresh_token grant, with retry logic and proper error surfacing when refresh fails.',
    status: { id: 'st_3', name: 'In Progress', color: '#f59e0b' },
    priority: 'high',
    type: 'feature',
    workflow: { id: 'wf_1', name: 'Standard Dev', type: 'development' },
    assignedAgent: { id: 'ag_1', name: 'Claude Opus', provider: 'Anthropic' },
    repoScopes: [
      {
        repoName: 'openase/core',
        branchName: 'feat/oauth2-refresh',
        prUrl: 'https://github.com/openase/core/pull/87',
        prStatus: 'open',
        ciStatus: 'pass',
      },
      {
        repoName: 'openase/connectors',
        branchName: 'feat/github-token-refresh',
        prUrl: 'https://github.com/openase/connectors/pull/23',
        prStatus: 'draft',
        ciStatus: 'running',
      },
    ],
    attemptCount: 2,
    costAmount: 1.47,
    budgetUsd: 5.0,
    dependencies: [
      {
        id: 'tk_38',
        identifier: 'ASE-38',
        title: 'Add secrets store abstraction',
        relation: 'blocked_by',
      },
    ],
    children: [
      {
        id: 'tk_43',
        identifier: 'ASE-43',
        title: 'Write refresh token unit tests',
        status: 'done',
      },
    ],
    createdBy: 'yuzhong',
    createdAt: '2026-03-19T10:30:00Z',
    updatedAt: '2026-03-20T08:15:00Z',
    startedAt: '2026-03-19T11:00:00Z',
  }

  const MOCK_HOOKS: HookExecution[] = [
    {
      id: 'hk_1',
      hookName: 'pre-implementation',
      status: 'pass',
      duration: 3200,
      output: 'All pre-checks passed. Repository cloned. Branch created.',
      timestamp: '2026-03-19T11:00:00Z',
    },
    {
      id: 'hk_2',
      hookName: 'lint-check',
      status: 'pass',
      duration: 1450,
      output: 'No linting errors found.',
      timestamp: '2026-03-19T14:30:00Z',
    },
    {
      id: 'hk_3',
      hookName: 'test-suite',
      status: 'fail',
      duration: 8700,
      output:
        'FAIL: TestOAuth2Refresh/token_expired_scenario\n  Expected status 200, got 401\n  at oauth2_test.go:142',
      timestamp: '2026-03-19T15:00:00Z',
    },
    {
      id: 'hk_4',
      hookName: 'test-suite',
      status: 'pass',
      duration: 7300,
      output: 'All 24 tests passed.',
      timestamp: '2026-03-20T08:10:00Z',
    },
  ]

  const MOCK_ACTIVITIES: TicketActivity[] = [
    {
      id: 'act_1',
      type: 'started',
      message: 'Ticket moved to In Progress',
      timestamp: '2026-03-19T11:00:00Z',
    },
    {
      id: 'act_2',
      type: 'agent_assigned',
      message: 'Claude Opus assigned to this ticket',
      agentName: 'System',
      timestamp: '2026-03-19T11:00:30Z',
    },
    {
      id: 'act_3',
      type: 'pr_opened',
      message: 'Opened PR #87 on openase/core',
      agentName: 'Claude Opus',
      timestamp: '2026-03-19T14:20:00Z',
    },
    {
      id: 'act_4',
      type: 'failed',
      message: 'Test suite failed — retrying with fix',
      agentName: 'Claude Opus',
      timestamp: '2026-03-19T15:00:00Z',
    },
    {
      id: 'act_5',
      type: 'pr_opened',
      message: 'Opened PR #23 on openase/connectors',
      agentName: 'Claude Opus',
      timestamp: '2026-03-20T07:45:00Z',
    },
    {
      id: 'act_6',
      type: 'status_change',
      message: 'All tests passing on attempt 2',
      agentName: 'Claude Opus',
      timestamp: '2026-03-20T08:15:00Z',
    },
  ]

  let {
    open = $bindable(false),
    ticket = MOCK_TICKET,
    hooks = MOCK_HOOKS,
    activities = MOCK_ACTIVITIES,
  }: {
    open?: boolean
    ticket?: TicketDetail
    hooks?: HookExecution[]
    activities?: TicketActivity[]
  } = $props()
</script>

<Sheet bind:open>
  <SheetContent side="right" class="flex w-full flex-col p-0 sm:max-w-lg" showCloseButton={false}>
    <SheetHeader class="sr-only">
      <SheetTitle>{ticket.identifier}: {ticket.title}</SheetTitle>
      <SheetDescription>Ticket detail drawer</SheetDescription>
    </SheetHeader>

    <TicketHeader {ticket} />

    <Tabs value="summary" class="flex flex-1 flex-col overflow-hidden">
      <TabsList class="mx-5 shrink-0">
        <TabsTrigger value="summary">Summary</TabsTrigger>
        <TabsTrigger value="code">Code</TabsTrigger>
        <TabsTrigger value="hooks">Hooks</TabsTrigger>
        <TabsTrigger value="activity">Activity</TabsTrigger>
      </TabsList>

      <div class="flex-1 overflow-y-auto">
        <TabsContent value="summary" class="mt-0">
          <TicketSummary {ticket} />
        </TabsContent>

        <TabsContent value="code" class="mt-0">
          <TicketRepos {ticket} />
        </TabsContent>

        <TabsContent value="hooks" class="mt-0">
          <TicketHooks {hooks} />
        </TabsContent>

        <TabsContent value="activity" class="mt-0">
          <TicketActivityList {activities} />
        </TabsContent>
      </div>
    </Tabs>
  </SheetContent>
</Sheet>
