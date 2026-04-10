import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { createProjectConversationTerminalSession } = vi.hoisted(() => ({
  createProjectConversationTerminalSession: vi.fn(),
}))

class MockFitAddon {
  fit = vi.fn()
}

class MockTerminal {
  cols = 96
  rows = 28
  output = ''
  onDataHandler?: (value: string) => void
  onResizeHandler?: (value: { cols: number; rows: number }) => void

  loadAddon() {}
  open() {}
  focus() {}
  clear() {
    this.output = ''
  }
  reset() {
    this.output = ''
  }
  dispose() {}
  write(value: string | Uint8Array) {
    this.output += value instanceof Uint8Array ? new TextDecoder().decode(value) : value
  }
  onData(handler: (value: string) => void) {
    this.onDataHandler = handler
    return { dispose() {} }
  }
  onResize(handler: (value: { cols: number; rows: number }) => void) {
    this.onResizeHandler = handler
    return { dispose() {} }
  }
}

class MockWebSocket {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSING = 2
  static CLOSED = 3
  static instances: MockWebSocket[] = []

  readyState = MockWebSocket.OPEN
  sent: string[] = []
  onmessage: ((event: { data: string }) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null

  constructor(public readonly url: string) {
    MockWebSocket.instances.push(this)
  }

  send(value: string) {
    this.sent.push(value)
  }

  close() {
    this.readyState = MockWebSocket.CLOSED
    this.onclose?.()
  }

  emitMessage(payload: Record<string, unknown>) {
    this.onmessage?.({ data: JSON.stringify(payload) })
  }

  emitError() {
    this.onerror?.()
  }
}

const terminalInstances: MockTerminal[] = []

vi.mock('$lib/api/chat', () => ({
  createProjectConversationTerminalSession,
}))

vi.mock('@xterm/xterm', () => ({
  Terminal: class extends MockTerminal {
    constructor() {
      super()
      terminalInstances.push(this)
    }
  },
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: MockFitAddon,
}))

vi.mock('@xterm/xterm/css/xterm.css', () => ({}))

import ProjectConversationTerminalPanel from './project-conversation-terminal-panel.svelte'

describe('ProjectConversationTerminalPanel', () => {
  beforeAll(() => {
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket)
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    MockWebSocket.instances.length = 0
    terminalInstances.length = 0
  })

  it('opens a terminal at the current repo directory and renders output plus exit state', async () => {
    createProjectConversationTerminalSession.mockResolvedValue({
      terminalSession: {
        id: 'terminal-1',
        mode: 'shell',
        cwd: '/tmp/conversation-1/services/openase/src',
        wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-1/attach',
        attachToken: 'attach-token-1',
      },
    })

    render(ProjectConversationTerminalPanel, {
      props: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
        selectedRepoPath: 'services/openase',
        selectedFilePath: 'src/main.ts',
      },
    })

    await waitFor(() => expect(terminalInstances).toHaveLength(1))
    await fireEvent.click(screen.getByRole('button', { name: /open here/i }))

    await waitFor(() =>
      expect(createProjectConversationTerminalSession).toHaveBeenCalledWith(
        'conversation-1',
        expect.objectContaining({
          mode: 'shell',
          repoPath: 'services/openase',
          cwdPath: 'src',
          cols: expect.any(Number),
          rows: expect.any(Number),
        }),
      ),
    )

    expect(MockWebSocket.instances).toHaveLength(1)
    const socket = MockWebSocket.instances[0]
    expect(socket.url).toContain('attach_token=attach-token-1')

    socket.emitMessage({ type: 'ready' })
    socket.emitMessage({ type: 'output', data: btoa('pwd\r\n') })
    socket.emitMessage({ type: 'exit', exit_code: 0 })

    await waitFor(() => expect(screen.getByText(/exit code 0/i)).toBeTruthy())
    expect(terminalInstances[0].output).toContain('pwd')
  })

  it('shows server error frames in the terminal status overlay', async () => {
    createProjectConversationTerminalSession.mockResolvedValue({
      terminalSession: {
        id: 'terminal-2',
        mode: 'shell',
        cwd: '/tmp/conversation-1',
        wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-2/attach',
        attachToken: 'attach-token-2',
      },
    })

    render(ProjectConversationTerminalPanel, {
      props: {
        conversationId: 'conversation-1',
        workspacePath: '/tmp/conversation-1',
      },
    })

    await waitFor(() => expect(terminalInstances).toHaveLength(1))
    await fireEvent.click(screen.getByRole('button', { name: /workspace root/i }))

    await waitFor(() => expect(MockWebSocket.instances).toHaveLength(1))
    MockWebSocket.instances[0].emitMessage({
      type: 'error',
      message: 'attach token invalid',
    })

    await waitFor(() => expect(screen.getByText('attach token invalid')).toBeTruthy())
  })
})
