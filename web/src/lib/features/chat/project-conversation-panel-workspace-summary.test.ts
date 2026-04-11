import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  closeProjectConversationRuntime,
  createProjectConversationTerminalSession,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  getProjectConversationWorkspaceDiff,
  syncProjectConversationWorkspace,
  listProjectConversationEntries,
  listProjectConversations,
  listProjectConversationWorkspaceTree,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
} = vi.hoisted(() => ({
  closeProjectConversationRuntime: vi.fn(),
  createProjectConversationTerminalSession: vi.fn(),
  createProjectConversation: vi.fn(),
  executeProjectConversationActionProposal: vi.fn(),
  getProjectConversation: vi.fn(),
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  getProjectConversationWorkspaceDiff: vi.fn(),
  syncProjectConversationWorkspace: vi.fn(),
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  closeProjectConversationRuntime,
  createProjectConversationTerminalSession,
  createProjectConversation,
  executeProjectConversationActionProposal,
  getProjectConversation,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  getProjectConversationWorkspaceDiff,
  syncProjectConversationWorkspace,
  listProjectConversationEntries,
  listProjectConversations,
  listProjectConversationWorkspaceTree,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream,
}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class {
    cols = 96
    rows = 28
    loadAddon() {}
    open() {}
    focus() {}
    clear() {}
    reset() {}
    dispose() {}
    write() {}
    onData() {
      return { dispose() {} }
    }
    onResize() {
      return { dispose() {} }
    }
  },
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: class {
    fit() {}
  },
}))

vi.mock('@xterm/xterm/css/xterm.css', () => ({}))

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'
import { createWorkspaceDiff } from './project-conversation-panel.test-helpers'
import { workspaceBrowserPortal } from './workspace-browser-portal.svelte'

function mockLiveMuxStream() {
  let handlers:
    | {
        signal?: AbortSignal
        onOpen?: () => void
        onFrame: (frame: {
          conversationId: string
          sentAt: string
          event: { kind: string; payload: Record<string, unknown> }
        }) => void
      }
    | undefined

  watchProjectConversationMuxStream.mockImplementation(async (_projectId, nextHandlers) => {
    handlers = nextHandlers
    nextHandlers.onOpen?.()
    await new Promise<void>((resolve) => {
      nextHandlers.signal?.addEventListener('abort', () => resolve(), { once: true })
    })
  })

  return {
    emit(conversationId: string, event: { kind: string; payload: Record<string, unknown> }) {
      handlers?.onFrame({
        conversationId,
        sentAt: '2026-04-01T10:00:00Z',
        event,
      })
    },
  }
}

describe('ProjectConversationPanel workspace summary', () => {
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
    workspaceBrowserPortal.close()
    workspaceBrowserPortal.conversationId = ''
    workspaceBrowserPortal.workspaceDiff = null
    workspaceBrowserPortal.workspaceDiffLoading = false
  })

  it('renders compact workspace bar and expands to show repo details', async () => {
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-1', true),
    )
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Current conversation' },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    watchProjectConversation.mockResolvedValue(undefined)

    const { findByText, queryByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await findByText('Workspace changes')
    await findByText('1 repo changed · +4 -1')
    expect(queryByText('web/src/app.ts')).toBeNull()

    const bar = await findByText('Workspace changes')
    await fireEvent.click(bar.closest('button')!)

    await findByText('web/src/app.ts')
    await findByText('M')
  })

  it('syncs workspace state into the shared browser portal when browse is toggled', async () => {
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-1', true),
    )
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Current conversation' },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    watchProjectConversation.mockResolvedValue(undefined)

    const diff = createWorkspaceDiff('conversation-1', true)

    const { findByText, getByText, queryByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
      },
    })

    await findByText('Browse')
    await waitFor(() => expect(workspaceBrowserPortal.conversationId).toBe('conversation-1'))
    expect(workspaceBrowserPortal.open).toBe(false)
    expect(workspaceBrowserPortal.workspaceDiff).toEqual(diff.workspaceDiff)

    await fireEvent.click(getByText('Browse'))

    await waitFor(() => expect(workspaceBrowserPortal.open).toBe(true))
    expect(workspaceBrowserPortal.conversationId).toBe('conversation-1')
    expect(workspaceBrowserPortal.workspaceDiff).toEqual(diff.workspaceDiff)
    expect(queryByText('Browse')).toBeNull()
    expect(getByText('Hide browser')).toBeTruthy()

    await fireEvent.click(getByText('Hide browser'))
    await waitFor(() => expect(workspaceBrowserPortal.open).toBe(false))
    expect(getByText('Browse')).toBeTruthy()
  })

  it('updates the workspace summary in real time after a turn completes', async () => {
    const mux = mockLiveMuxStream()

    listProjectConversations.mockResolvedValue({ conversations: [] })
    createProjectConversation.mockResolvedValue({
      conversation: {
        id: 'conversation-1',
        providerId: 'provider-1',
        lastActivityAt: '2026-04-01T10:00:00Z',
      },
    })
    getProjectConversationWorkspaceDiff.mockResolvedValue(
      createWorkspaceDiff('conversation-1', true),
    )
    startProjectConversationTurn.mockResolvedValue({
      turn: { id: 'turn-1', turn_index: 1, status: 'started' },
    })

    const { container, getByPlaceholderText, getByRole } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    await fireEvent.input(getByPlaceholderText('Ask anything about this project…'), {
      target: { value: 'Check the repo state' },
    })
    await fireEvent.click(getByRole('button', { name: 'Send message' }))

    expect(container.textContent).not.toContain('1 repo changed · +4 -1')

    mux.emit('conversation-1', {
      kind: 'turn_done',
      payload: {
        conversationId: 'conversation-1',
        turnId: 'turn-1',
      },
    })

    await waitFor(() => expect(container.textContent).toContain('1 repo changed · +4 -1'))
    expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(1)
  })

  it('surfaces repo sync guidance and refreshes workspace diff after syncing', async () => {
    listProjectConversations.mockResolvedValue({
      conversations: [
        {
          id: 'conversation-1',
          rollingSummary: 'Current conversation',
          lastActivityAt: '2026-04-01T10:00:00Z',
          providerId: 'provider-1',
        },
      ],
    })
    getProjectConversationWorkspaceDiff
      .mockResolvedValueOnce({
        workspaceDiff: {
          conversationId: 'conversation-1',
          workspacePath: '/tmp/conversation-1',
          dirty: false,
          reposChanged: 0,
          filesChanged: 0,
          added: 0,
          removed: 0,
          repos: [],
          syncPrompt: {
            reason: 'repo_binding_changed',
            missingRepos: [{ name: 'docs', path: 'docs' }],
          },
        },
      })
      .mockResolvedValueOnce(createWorkspaceDiff('conversation-1'))
    syncProjectConversationWorkspace.mockResolvedValue({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [],
      },
    })
    listProjectConversationEntries.mockResolvedValue({
      entries: [
        {
          id: 'entry-1',
          conversationId: 'conversation-1',
          turnId: 'turn-1',
          seq: 1,
          kind: 'user_message',
          payload: { content: 'Current conversation' },
          createdAt: '2026-04-01T10:00:00Z',
        },
      ],
    })
    watchProjectConversation.mockResolvedValue(undefined)

    const { findByText, getByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
      },
    })

    await findByText('1 repo needs sync')
    await fireEvent.click(getByText('Sync repos'))

    await waitFor(() => {
      expect(syncProjectConversationWorkspace).toHaveBeenCalledWith('conversation-1')
      expect(getProjectConversationWorkspaceDiff).toHaveBeenCalledTimes(2)
    })
  })
})
