import { waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, beforeEach, expect, it, vi } from 'vitest'

const { createProjectConversationTerminalSession } = vi.hoisted(() => ({
  createProjectConversationTerminalSession: vi.fn(),
}))

class MockFitAddon {
  fit = vi.fn()
}

class MockTerminal {
  static instances: MockTerminal[] = []

  cols = 96
  rows = 28
  output = ''
  onDataHandler?: (value: string) => void
  onResizeHandler?: (value: { cols: number; rows: number }) => void

  constructor() {
    MockTerminal.instances.push(this)
  }

  loadAddon() {}
  open() {}
  focus() {}
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
    if (this.readyState === MockWebSocket.CLOSED) {
      return
    }
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

vi.mock('$lib/api/chat', () => ({
  createProjectConversationTerminalSession,
}))

vi.mock('@xterm/xterm', () => ({
  Terminal: MockTerminal,
}))

vi.mock('@xterm/addon-fit', () => ({
  FitAddon: MockFitAddon,
}))

import { createTerminalManager } from './terminal-manager.svelte'

beforeAll(() => {
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket)
})

beforeEach(() => {
  vi.useRealTimers()
})

afterEach(() => {
  vi.clearAllMocks()
  MockWebSocket.instances.length = 0
  MockTerminal.instances.length = 0
})

it('keeps existing terminal tabs connected when another tab opens and closes', async () => {
  createProjectConversationTerminalSession
    .mockResolvedValueOnce({
      terminalSession: {
        id: 'terminal-1',
        mode: 'shell',
        cwd: '/tmp/conversation-1',
        wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-1/attach',
        attachToken: 'attach-token-1',
      },
    })
    .mockResolvedValueOnce({
      terminalSession: {
        id: 'terminal-2',
        mode: 'shell',
        cwd: '/tmp/conversation-1',
        wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-2/attach',
        attachToken: 'attach-token-2',
      },
    })

  const manager = createTerminalManager({
    getConversationId: () => 'conversation-1',
    getWorkspacePath: () => '/tmp/conversation-1',
  })

  const firstId = manager.createInstance()
  await manager.mountTerminal(firstId, document.createElement('div'))
  await manager.connectTerminal(firstId)
  await waitFor(() => expect(MockWebSocket.instances).toHaveLength(1))
  MockWebSocket.instances[0].emitMessage({ type: 'ready' })
  await waitFor(() => expect(manager.instances[0]?.status).toBe('open'))

  const secondId = manager.createInstance()
  await manager.mountTerminal(secondId, document.createElement('div'))
  await manager.connectTerminal(secondId)
  await waitFor(() => expect(MockWebSocket.instances).toHaveLength(2))
  MockWebSocket.instances[1].emitMessage({ type: 'ready' })
  await waitFor(() => expect(manager.getActiveInstance()?.id).toBe(secondId))

  manager.removeInstance(secondId)

  expect(manager.activeId).toBe(firstId)
  expect(MockWebSocket.instances[1]?.sent).toContain(JSON.stringify({ type: 'close' }))
  expect(MockWebSocket.instances[0]?.readyState).toBe(MockWebSocket.OPEN)
})

it('reconnects a terminal after an unexpected socket close', async () => {
  vi.useFakeTimers()
  createProjectConversationTerminalSession.mockResolvedValue({
    terminalSession: {
      id: 'terminal-1',
      mode: 'shell',
      cwd: '/tmp/conversation-1',
      wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-1/attach',
      attachToken: 'attach-token-1',
    },
  })

  const manager = createTerminalManager({
    getConversationId: () => 'conversation-1',
    getWorkspacePath: () => '/tmp/conversation-1',
  })

  const id = manager.createInstance()
  await manager.mountTerminal(id, document.createElement('div'))
  await manager.connectTerminal(id)
  await waitFor(() => expect(MockWebSocket.instances).toHaveLength(1))
  MockWebSocket.instances[0].emitMessage({ type: 'ready' })
  await waitFor(() => expect(manager.instances[0]?.status).toBe('open'))
  MockWebSocket.instances[0].emitMessage({ type: 'output', data: 'YmVmb3Jl' })
  expect(MockTerminal.instances[0]?.output).toContain('before')

  MockWebSocket.instances[0].close()
  await waitFor(() => expect(manager.instances[0]?.status).toBe('connecting'))

  await vi.advanceTimersByTimeAsync(750)
  await waitFor(() => expect(MockWebSocket.instances).toHaveLength(2))

  MockWebSocket.instances[1].emitMessage({ type: 'ready' })
  MockWebSocket.instances[1].emitMessage({ type: 'output', data: 'YWZ0ZXI=' })
  await waitFor(() => {
    expect(manager.instances[0]?.status).toBe('open')
    expect(manager.instances[0]?.sessionID).toBe('terminal-1')
  })
  expect(createProjectConversationTerminalSession).toHaveBeenCalledTimes(1)
  expect(MockWebSocket.instances[1]?.url).toContain('/terminal-1/attach')
  expect(MockTerminal.instances[0]?.output).toContain('beforeafter')
})

it('does not leak a websocket when a tab is removed during session creation', async () => {
  let resolveSession: ((value: unknown) => void) | undefined
  createProjectConversationTerminalSession.mockImplementation(
    () =>
      new Promise((resolve) => {
        resolveSession = resolve
      }),
  )

  const manager = createTerminalManager({
    getConversationId: () => 'conversation-1',
    getWorkspacePath: () => '/tmp/conversation-1',
  })

  const id = manager.createInstance()
  await manager.mountTerminal(id, document.createElement('div'))
  const connectPromise = manager.connectTerminal(id)

  manager.removeInstance(id)
  resolveSession?.({
    terminalSession: {
      id: 'terminal-1',
      mode: 'shell',
      cwd: '/tmp/conversation-1',
      wsPath: '/api/v1/chat/conversations/conversation-1/terminal-sessions/terminal-1/attach',
      attachToken: 'attach-token-1',
    },
  })

  await connectPromise
  await Promise.resolve()

  expect(manager.instances).toHaveLength(0)
  expect(MockWebSocket.instances).toHaveLength(0)
})
