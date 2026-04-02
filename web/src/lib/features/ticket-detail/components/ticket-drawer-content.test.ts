import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  closeChatSession,
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  streamChatTurn,
  watchProjectConversation,
} = vi.hoisted(() => ({
  closeChatSession: vi.fn(),
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversation: vi.fn(),
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  streamChatTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
}))

const { listProviders } = vi.hoisted(() => ({
  listProviders: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeChatSession,
  closeProjectConversationRuntime,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspaceDiff,
  listProjectConversationEntries,
  listProjectConversations,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  streamChatTurn,
  watchProjectConversation,
}))

vi.mock('$lib/api/openase', async () => {
  const actual = await vi.importActual<typeof import('$lib/api/openase')>('$lib/api/openase')
  return {
    ...actual,
    listProviders,
  }
})

import { appStore } from '$lib/stores/app.svelte'
import { providerFixtures } from '$lib/features/chat/ephemeral-chat-session-controller.test-helpers'
import { createWorkspaceDiff } from '$lib/features/chat/project-conversation-panel.test-helpers'
import type {
  HookExecution,
  TicketDetail,
  TicketRun,
  TicketStatusOption,
  TicketTimelineItem,
} from '../types'
import TicketDrawerContent from './ticket-drawer-content.svelte'

const statusesFixture: TicketStatusOption[] = [
  { id: 'status-1', name: 'In Review', color: '#f59e0b' },
  { id: 'status-2', name: 'Done', color: '#10b981' },
]

const ticketFixture: TicketDetail = {
  id: 'ticket-1',
  identifier: 'ASE-470',
  title: 'Replace Ticket AI with ticket-focused Project AI',
  description: 'Route ticket drawer AI through the durable project conversation runtime.',
  status: statusesFixture[0],
  priority: 'high',
  type: 'feature',
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
      prUrl: 'https://github.com/GrandCX/openase/pull/999',
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

const hooksFixture: HookExecution[] = [
  {
    id: 'hook-1',
    hookName: 'ticket.on_complete',
    status: 'fail',
    output: 'go test ./... failed in internal/chat',
    timestamp: '2026-04-02T08:15:00Z',
  },
]

const timelineFixture: TicketTimelineItem[] = [
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

const currentRunFixture: TicketRun = {
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

describe('TicketDrawerContent project AI integration', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    window.localStorage.clear()
    appStore.currentOrg = null
    appStore.currentProject = null
  })

  function renderDrawer() {
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

  it('opens Project AI from the ticket drawer and sends the full ticket focus through project conversations', async () => {
    listProviders.mockResolvedValue({ providers: providerFixtures })
    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-02T09:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { getByLabelText, getByPlaceholderText, getByRole } = renderDrawer()

    await fireEvent.click(getByLabelText('AI 分析'))
    await waitFor(() => {
      expect(listProjectConversations).toHaveBeenCalled()
    })

    const prompt = getByPlaceholderText(
      'Ask about this ticket without restating the basics…',
    ) as HTMLTextAreaElement
    await fireEvent.input(prompt, { target: { value: 'Why is this ticket not running?' } })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

    await waitFor(() => {
      expect(startProjectConversationTurn).toHaveBeenCalledWith(
        'conversation-1',
        expect.objectContaining({
          message: 'Why is this ticket not running?',
          focus: expect.objectContaining({
            kind: 'ticket',
            projectId: 'project-1',
            ticketId: 'ticket-1',
            ticketIdentifier: 'ASE-470',
            ticketDescription:
              'Route ticket drawer AI through the durable project conversation runtime.',
            ticketPriority: 'high',
            ticketAttemptCount: 3,
            ticketRetryPaused: true,
            ticketPauseReason: 'Repeated hook failures',
            selectedArea: 'comments',
            ticketDependencies: [
              expect.objectContaining({
                identifier: 'ASE-100',
                relation: 'blocked_by',
              }),
            ],
            ticketRepoScopes: [
              expect.objectContaining({
                repoId: 'repo-1',
                pullRequestUrl: 'https://github.com/GrandCX/openase/pull/999',
              }),
            ],
            ticketHookHistory: [
              expect.objectContaining({
                hookName: 'ticket.on_complete',
                status: 'fail',
              }),
            ],
            ticketCurrentRun: expect.objectContaining({
              id: 'run-1',
              status: 'failed',
              currentStepSummary: 'openase test ./internal/chat',
            }),
          }),
        }),
      )
    })

    expect(streamChatTurn).not.toHaveBeenCalled()
    expect(closeChatSession).not.toHaveBeenCalled()
  })

  it('keeps action proposal cards working from the ticket drawer entry point', async () => {
    window.localStorage.setItem(
      'openase.project-conversation.project-1.provider-1',
      JSON.stringify({
        conversationIds: ['conversation-1'],
        activeConversationId: 'conversation-1',
      }),
    )

    listProviders.mockResolvedValue({ providers: providerFixtures })
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'ticket-focused project ai',
          lastActivityAt: '2026-04-02T09:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-proposal',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'action_proposal',
          payload: {
            summary: 'Create a retry investigation child ticket',
            actions: [
              {
                method: 'POST',
                path: '/api/v1/projects/project-1/tickets',
                body: { title: 'Investigate repeated retry pause' },
              },
            ],
          },
          createdAt: '2026-04-02T09:00:00Z',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(createWorkspaceDiff('conversation-1'))
    watchProjectConversation.mockResolvedValue(undefined)
    executeProjectConversationActionProposal.mockResolvedValue({
      results: [{ ok: true, summary: 'POST /api/v1/projects/project-1/tickets succeeded.' }],
    })

    const { findByText, getByLabelText, getByRole } = renderDrawer()

    await fireEvent.click(getByLabelText('AI 分析'))
    await findByText('Create a retry investigation child ticket')
    await fireEvent.click(getByRole('button', { name: 'Confirm' }))

    await waitFor(() => {
      expect(executeProjectConversationActionProposal).toHaveBeenCalledWith(
        'conversation-1',
        'entry-proposal',
      )
    })
    expect(await findByText('Executed')).toBeTruthy()
  })
})
