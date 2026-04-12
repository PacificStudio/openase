import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const {
  createProjectConversationTerminalSession,
  createProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
} = vi.hoisted(() => ({
  createProjectConversationTerminalSession: vi.fn(),
  createProjectConversationWorkspaceFile: vi.fn(),
  deleteProjectConversationWorkspaceFile: vi.fn(),
  getProjectConversationWorkspace: vi.fn(),
  getProjectConversationWorkspaceFilePatch: vi.fn(),
  getProjectConversationWorkspaceFilePreview: vi.fn(),
  listProjectConversationWorkspaceTree: vi.fn(),
  renameProjectConversationWorkspaceFile: vi.fn(),
  searchProjectConversationWorkspacePaths: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  createProjectConversationTerminalSession,
  createProjectConversationWorkspaceFile,
  deleteProjectConversationWorkspaceFile,
  getProjectConversationWorkspace,
  getProjectConversationWorkspaceFilePatch,
  getProjectConversationWorkspaceFilePreview,
  listProjectConversationWorkspaceTree,
  renameProjectConversationWorkspaceFile,
  searchProjectConversationWorkspacePaths,
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
    await fireEvent.click(view.getByRole('button', { name: 'Toggle terminal' }))

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
