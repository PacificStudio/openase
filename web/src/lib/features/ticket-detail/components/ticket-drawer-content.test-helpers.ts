import { render } from '@testing-library/svelte'
import type { AgentProvider } from '$lib/api/contracts'
import { appStore } from '$lib/stores/app.svelte'
import type {
  HookExecution,
  TicketDetail,
  TicketRun,
  TicketStatusOption,
  TicketTimelineItem,
} from '../types'
import TicketDrawerContent from './ticket-drawer-content.svelte'

export const providerFixtures: AgentProvider[] = [
  {
    id: 'provider-1',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Codex',
    adapter_type: 'codex-app-server',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
    cli_command: 'codex',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'gpt-5.4',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
  {
    id: 'provider-2',
    organization_id: 'org-1',
    machine_id: 'machine-1',
    machine_name: 'Localhost',
    machine_host: '127.0.0.1',
    machine_status: 'online',
    machine_ssh_user: null,
    machine_workspace_root: '/workspace',
    name: 'Claude',
    adapter_type: 'claude-code-cli',
    permission_profile: 'unrestricted',
    availability_state: 'available',
    available: true,
    availability_checked_at: '2026-03-28T12:00:00Z',
    availability_reason: null,
    capabilities: {
      ephemeral_chat: {
        state: 'available',
        reason: null,
      },
    },
    cli_command: 'claude',
    cli_args: [],
    auth_config: {},
    cli_rate_limit: null,
    cli_rate_limit_updated_at: null,
    model_name: 'claude-sonnet-4',
    model_temperature: 0,
    model_max_tokens: 4096,
    max_parallel_runs: 2,
    cost_per_input_token: 0,
    cost_per_output_token: 0,
    pricing_config: {},
  },
]

export const statusesFixture: TicketStatusOption[] = [
  { id: 'status-1', name: 'In Review', color: '#f59e0b' },
  { id: 'status-2', name: 'Done', color: '#10b981' },
]

export const ticketFixture: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-470',
  title: 'Replace Ticket AI with ticket-focused Project AI',
  description: 'Route ticket drawer AI through the durable project conversation runtime.',
  status: statusesFixture[0],
  priority: 'high',
  type: 'feature',
  archived: false,
  workflow: { id: 'workflow-1', name: 'coding', type: 'coding' },
  assignedAgent: {
    id: 'agent-1',
    name: 'todo-app-coding-01',
    provider: 'codex-cloud',
    runtimeControlState: 'active',
    runtimePhase: 'executing',
  },
  repoScopes: [
    {
      id: 'scope-1',
      repoId: 'repo-1',
      repoName: 'openase',
      branchName: 'feat/openase-470-project-ai',
      defaultBranch: 'main',
      effectiveBranchName: 'feat/openase-470-project-ai',
      branchSource: 'override',
      prUrl: 'https://github.com/PacificStudio/openase/pull/999',
    },
  ],
  attemptCount: 3,
  consecutiveErrors: 2,
  retryPaused: true,
  pauseReason: 'Repeated hook failures',
  currentRunId: 'run-1',
  targetMachineId: 'machine-1',
  nextRetryAt: '2026-04-02T10:00:00Z',
  costTokensInput: 1200,
  costTokensOutput: 340,
  costAmount: 1.23,
  budgetUsd: 10,
  dependencies: [
    {
      id: 'dep-1',
      targetId: 'ticket-2',
      identifier: 'ASE-100',
      title: 'Finish durable conversation restore',
      relation: 'blocked_by',
      stage: 'started',
    },
  ],
  externalLinks: [],
  children: [],
  createdBy: 'user:test',
  createdAt: '2026-04-01T09:00:00Z',
  updatedAt: '2026-04-01T09:30:00Z',
}

export const hooksFixture: HookExecution[] = [
  {
    id: 'hook-1',
    hookName: 'ticket.on_complete',
    status: 'fail',
    output: 'go test ./... failed in internal/chat',
    timestamp: '2026-04-02T08:15:00Z',
  },
]

export const timelineFixture: TicketTimelineItem[] = [
  {
    id: 'activity-1',
    kind: 'activity',
    ticketId: 'ticket-1',
    actor: { name: 'dispatcher', type: 'system' },
    createdAt: '2026-04-02T08:10:00Z',
    updatedAt: '2026-04-02T08:10:00Z',
    isCollapsible: true,
    isDeleted: false,
    eventType: 'ticket.retry_paused',
    title: 'ticket.retry_paused',
    bodyText: 'Paused retries after repeated failures.',
    metadata: {},
  },
]

export const currentRunFixture: TicketRun = {
  id: 'run-1',
  attemptNumber: 3,
  agentId: 'agent-1',
  agentName: 'todo-app-coding-01',
  provider: 'codex-cloud',
  status: 'failed',
  currentStepStatus: 'failed',
  currentStepSummary: 'openase test ./internal/chat',
  createdAt: '2026-04-02T08:00:00Z',
  lastError: 'ticket.on_complete hook failed',
}

export function resetTicketDrawerTestAppStore() {
  appStore.currentOrg = null
  appStore.currentProject = null
}

export function createWorkspaceDiff(conversationId: string, dirty = false) {
  return {
    workspaceDiff: {
      conversationId,
      workspacePath: `/tmp/${conversationId}`,
      dirty,
      reposChanged: dirty ? 1 : 0,
      filesChanged: dirty ? 1 : 0,
      added: dirty ? 4 : 0,
      removed: dirty ? 1 : 0,
      repos: dirty
        ? [
            {
              name: 'openase',
              path: 'services/openase',
              branch: 'agent/conv-123',
              dirty: true,
              filesChanged: 1,
              added: 4,
              removed: 1,
              files: [
                {
                  path: 'web/src/app.ts',
                  status: 'modified',
                  added: 4,
                  removed: 1,
                },
              ],
            },
          ]
        : [],
    },
  }
}

export function renderTicketDrawerContent() {
  appStore.currentOrg = {
    id: 'org-1',
    name: 'Org',
    slug: 'org',
    default_agent_provider_id: 'provider-1',
    status: 'active',
  }
  appStore.currentProject = {
    id: 'project-1',
    organization_id: 'org-1',
    name: 'OpenASE',
    slug: 'openase',
    description: '',
    status: 'active',
    default_agent_provider_id: 'provider-1',
    max_concurrent_agents: 1,
    accessible_machine_ids: [],
  }

  return render(TicketDrawerContent, {
    props: {
      ticket: ticketFixture,
      projectId: 'project-1',
      hooks: hooksFixture,
      timeline: timelineFixture,
      runs: [currentRunFixture],
      currentRun: currentRunFixture,
      runBlocks: [],
      statuses: statusesFixture,
      dependencyCandidates: [],
      repoOptions: [],
    },
  })
}
