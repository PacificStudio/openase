import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'

const {
  checkoutProjectConversationWorkspaceBranch,
  commitProjectConversationWorkspace,
  createProjectConversationTerminalSession,
  createProjectConversationWorkspaceFile,
  discardProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  stageProjectConversationWorkspaceFile,
} = vi.hoisted(() => ({
  checkoutProjectConversationWorkspaceBranch: vi.fn(),
  commitProjectConversationWorkspace: vi.fn(),
  createProjectConversationTerminalSession: vi.fn(),
  createProjectConversationWorkspaceFile: vi.fn(),
  discardProjectConversationWorkspaceFile: vi.fn(),
  deleteProjectConversationWorkspaceFile: vi.fn(),
  getProjectConversationWorkspaceGitGraph: vi.fn(),
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceRepoRefs: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  renameProjectConversationWorkspaceFile: vi.fn(),
  searchProjectConversationWorkspacePaths: vi.fn(),
  stageProjectConversationWorkspaceFile: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  checkoutProjectConversationWorkspaceBranch,
  commitProjectConversationWorkspace,
  createProjectConversationTerminalSession,
  createProjectConversationWorkspaceFile,
  discardProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspaceGitGraph,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceRepoRefs,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
  stageProjectConversationWorkspaceFile,
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

import ProjectConversationWorkspaceBrowser from './project-conversation-workspace-browser.svelte'
import {
  ensureResizeObserver,
  mockWorkspaceMetadata,
  workspaceDiff,
} from './project-conversation-workspace-browser.test-helpers'

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3

  readyState = MockWebSocket.OPEN
  onmessage: ((event: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null

  constructor(public readonly url: string) {}

  send() {}

  close() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.()
  }
}

describe('ProjectConversationWorkspaceBrowser terminal layout', () => {
  beforeAll(() => {
    ensureResizeObserver()
    vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket)
  })

  beforeEach(() => {
    getProjectConversationWorkspaceRepoRefs.mockResolvedValue({
      repoRefs: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'agent/conv-123',
          cacheKey: 'branch:refs/heads/agent/conv-123',
          branchName: 'agent/conv-123',
          branchFullName: 'refs/heads/agent/conv-123',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Add workspace browser scaffolding',
        },
        localBranches: [],
        remoteBranches: [],
      },
    })
    getProjectConversationWorkspaceGitGraph.mockResolvedValue({
      gitGraph: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        limit: 40,
        commits: [],
      },
    })
    checkoutProjectConversationWorkspaceBranch.mockResolvedValue({
      checkout: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        currentRef: {
          kind: 'branch',
          displayName: 'agent/conv-123',
          cacheKey: 'branch:refs/heads/agent/conv-123',
          branchName: 'agent/conv-123',
          branchFullName: 'refs/heads/agent/conv-123',
          commitId: '123456789abc',
          shortCommitId: '123456789abc',
          subject: 'Add workspace browser scaffolding',
        },
        createdLocalBranch: '',
      },
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('keeps the bottom terminal panel stretched to the configured height', async () => {
    mockWorkspaceMetadata(getProjectConversationWorkspace)
    listProjectConversationWorkspaceTree.mockResolvedValue({
      workspaceTree: {
        conversationId: 'conversation-1',
        repoPath: 'services/openase',
        path: '',
        entries: [{ path: 'README.md', name: 'README.md', kind: 'file', sizeBytes: 64 }],
      },
    })
    createProjectConversationTerminalSession.mockResolvedValue({
      terminalSession: {
        id: 'terminal-1',
        mode: 'shell',
        cwd: '/tmp/conversation-1',
        wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-1/attach',
        attachToken: 'attach-token-1',
      },
    })

    const view = render(ProjectConversationWorkspaceBrowser, {
      props: {
        conversationId: 'conversation-1',
        workspaceDiff,
        workspaceDiffLoading: false,
      },
    })

    await waitFor(() => expect(getProjectConversationWorkspace).toHaveBeenCalledTimes(1))
    await fireEvent.click(view.getByRole('button', { name: 'Toggle Terminal' }))

    const viewport = await view.findByTestId('workspace-terminal-instance')
    const panelContent = viewport.parentElement as HTMLElement
    const panelRoot = panelContent.parentElement as HTMLElement
    const panelWrapper = panelRoot.parentElement as HTMLElement

    expect(panelRoot.className).toContain('h-full')
    expect(panelWrapper.className).toContain('flex')
    expect(panelWrapper.className).toContain('min-h-0')
    expect(panelWrapper.className).toContain('shrink-0')
    expect(panelWrapper.className).toContain('overflow-hidden')
    expect(panelWrapper.getAttribute('style')).toContain('height: 260px')
  })
})
