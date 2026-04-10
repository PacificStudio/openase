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
  listProjectConversationEntries,
  listProjectConversations,
  listProjectConversationWorkspaceTree,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
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
  listProjectConversationEntries: vi.fn(),
  listProjectConversations: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  respondProjectConversationInterrupt: vi.fn(),
  startProjectConversationTurn: vi.fn(),
  watchProjectConversation: vi.fn(),
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
  listProjectConversationEntries,
  listProjectConversations,
  listProjectConversationWorkspaceTree,
  respondProjectConversationInterrupt,
  startProjectConversationTurn,
  watchProjectConversation,
  watchProjectConversationMuxStream: vi.fn(),
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

    // Compact bar shows summary
    await findByText('Workspace changes')
    await findByText('1 repo changed · +4 -1')

    // Details are hidden by default
    expect(queryByText('web/src/app.ts')).toBeNull()

    // Click to expand
    const bar = await findByText('Workspace changes')
    await fireEvent.click(bar.closest('button')!)

    // Now file list is visible (single repo omits repo header)
    await findByText('web/src/app.ts')
    await findByText('M')
  })

  it('opens the read-only workspace browser and loads file preview plus diff', async () => {
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
    getProjectConversationWorkspace.mockResolvedValue({
      workspace: {
        conversationId: 'conversation-1',
        available: true,
        workspacePath: '/tmp/conversation-1',
        repos: [
          {
            name: 'openase',
            path: 'services/openase',
            branch: 'agent/conv-123',
            headCommit: '123456789abc',
            headSummary: 'Add workspace browser scaffolding',
            dirty: true,
            filesChanged: 1,
            added: 4,
            removed: 1,
          },
        ],
      },
    })
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [
          { path: 'src', name: 'src', kind: 'directory', sizeBytes: 0 },
          { path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 },
        ],
      },
    })
    getProjectConversationWorkspaceFilePreview.mockResolvedValue({
      filePreview: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        sizeBytes: 64,
        mediaType: 'text/plain',
        previewKind: 'text',
        truncated: false,
        content: 'line one\nline two\nline three\n',
      },
    })
    getProjectConversationWorkspaceFilePatch.mockResolvedValue({
      filePatch: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: 'README.md',
        status: 'modified',
        diffKind: 'text',
        truncated: false,
        diff: '@@ -1,2 +1,3 @@\n line one\n line two\n+line three\n',
      },
    })
    watchProjectConversation.mockResolvedValue(undefined)

    const { container, findByText, getByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        providers: providerFixtures,
        defaultProviderId: 'provider-1',
      },
    })

    await findByText('Browse')
    await fireEvent.click(getByText('Browse'))
    await findByText('Workspace browser')
    await findByText(/Add workspace browser scaffolding/)
    await fireEvent.click(getByText('README.md'))
    await waitFor(() => {
      expect(container.textContent).toContain('README.md')
      expect(container.textContent).toContain('line three')
      expect(container.textContent).toContain('+line three')
    })

    expect(getProjectConversationWorkspace).toHaveBeenCalledWith('conversation-1')
    expect(listProjectConversationWorkspaceTree).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: '',
    })
    expect(getProjectConversationWorkspaceFilePreview).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: 'README.md',
    })
    expect(getProjectConversationWorkspaceFilePatch).toHaveBeenCalledWith('conversation-1', {
      repoPath: 'services/openase',
      path: 'README.md',
    })
  })
})
