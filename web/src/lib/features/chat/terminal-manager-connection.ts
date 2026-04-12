import {
  buildExitMessage,
  buildTerminalWebSocketURL,
  parseTerminalServerFrame,
} from './project-conversation-terminal-panel-helpers'
import type { TerminalInstance, TerminalInstanceRuntime } from './terminal-manager-types'
import { createProjectConversationTerminalSession } from '$lib/api/chat'

export function createTerminalConnectionHelpers(input: {
  getConversationId: () => string
  hasInstance: (id: string) => boolean
  listInstances: () => TerminalInstance[]
  runtimeMap: Map<string, TerminalInstanceRuntime>
  socketMap: Map<string, WebSocket>
  scheduleReconnect: (id: string, label: string) => void
  updateInstance: (id: string, updates: Partial<TerminalInstance>) => void
}) {
  function matchesConnectionState(id: string, conversationId: string, connectRevision: number) {
    return (
      input.hasInstance(id) &&
      input.getConversationId() === conversationId &&
      input.runtimeMap.get(id)?.connectRevision === connectRevision
    )
  }

  function matchesSocketEvent(id: string, socket: WebSocket, connectRevision: number) {
    return (
      input.socketMap.get(id) === socket &&
      input.runtimeMap.get(id)?.connectRevision === connectRevision &&
      input.hasInstance(id)
    )
  }

  function setConnectingStatus(id: string, label: string, isReconnect: boolean) {
    input.updateInstance(id, {
      status: 'connecting',
      statusMessage: isReconnect
        ? `Reconnecting shell in ${label}...`
        : `Starting shell in ${label}...`,
    })
  }

  async function resolveTerminalSession(inputState: {
    id: string
    conversationId: string
    connectRevision: number
    runtime: TerminalInstanceRuntime
    terminal: import('@xterm/xterm').Terminal
  }) {
    if (inputState.runtime.session) {
      return inputState.runtime.session
    }

    inputState.terminal.reset()
    try {
      const payload = await createProjectConversationTerminalSession(inputState.conversationId, {
        mode: 'shell',
        cols: inputState.terminal.cols > 0 ? inputState.terminal.cols : 120,
        rows: inputState.terminal.rows > 0 ? inputState.terminal.rows : 32,
      })
      inputState.runtime.session = payload.terminalSession
      return payload.terminalSession
    } catch (error) {
      if (
        !matchesConnectionState(
          inputState.id,
          inputState.conversationId,
          inputState.connectRevision,
        )
      ) {
        return null
      }
      input.updateInstance(inputState.id, {
        status: 'error',
        statusMessage:
          error instanceof Error ? error.message : 'Failed to create terminal session.',
      })
      return null
    }
  }

  function handleSocketMessage(inputState: {
    id: string
    socket: WebSocket
    connectRevision: number
    terminal: import('@xterm/xterm').Terminal
    event: MessageEvent
    runtime: TerminalInstanceRuntime
  }) {
    if (!matchesSocketEvent(inputState.id, inputState.socket, inputState.connectRevision)) {
      return
    }
    try {
      const frame = parseTerminalServerFrame(inputState.event.data)
      const inst = input.listInstances().find((instance) => instance.id === inputState.id)
      switch (frame.type) {
        case 'ready':
          inputState.runtime.reconnectAttempts = 0
          input.updateInstance(inputState.id, {
            status: 'open',
            statusMessage: `Shell attached to ${inst?.label}.`,
          })
          inputState.terminal.focus()
          return
        case 'output':
          inputState.terminal.write(frame.data)
          return
        case 'exit':
          inputState.runtime.reconnectEnabled = false
          inputState.runtime.session = null
          input.updateInstance(inputState.id, {
            status: 'closed',
            statusMessage: buildExitMessage(frame.exitCode, frame.signal),
            sessionID: '',
          })
          inputState.socket.close()
          return
        case 'error':
          inputState.runtime.reconnectEnabled = false
          inputState.runtime.session = null
          input.updateInstance(inputState.id, {
            status: 'error',
            statusMessage: frame.message,
            sessionID: '',
          })
          inputState.socket.close()
      }
    } catch (error) {
      inputState.runtime.reconnectEnabled = false
      inputState.runtime.session = null
      input.updateInstance(inputState.id, {
        status: 'error',
        statusMessage: error instanceof Error ? error.message : 'Failed to parse terminal output.',
      })
      inputState.socket.close()
    }
  }

  function handleSocketError(
    id: string,
    socket: WebSocket,
    connectRevision: number,
    label: string,
  ) {
    if (!matchesSocketEvent(id, socket, connectRevision)) {
      return
    }
    input.updateInstance(id, {
      status: 'connecting',
      statusMessage: `Reconnecting shell in ${label}...`,
    })
  }

  function handleSocketClose(
    id: string,
    socket: WebSocket,
    connectRevision: number,
    runtime: TerminalInstanceRuntime,
  ) {
    if (!matchesSocketEvent(id, socket, connectRevision)) {
      return
    }
    input.socketMap.delete(id)
    const inst = input.listInstances().find((instance) => instance.id === id)
    if (!inst) {
      return
    }
    if (runtime.reconnectEnabled && (inst.status === 'connecting' || inst.status === 'open')) {
      input.scheduleReconnect(id, inst.label)
      return
    }
    if (inst.status === 'connecting' || inst.status === 'open') {
      input.updateInstance(id, {
        status: 'closed',
        statusMessage: 'Terminal disconnected.',
        sessionID: '',
      })
    }
  }

  function attachSocket(inputState: {
    id: string
    session: import('$lib/api/chat').ProjectConversationTerminalSession
    connectRevision: number
    terminal: import('@xterm/xterm').Terminal
    runtime: TerminalInstanceRuntime
    label: string
  }) {
    const socket = new WebSocket(buildTerminalWebSocketURL(inputState.session))
    input.socketMap.set(inputState.id, socket)
    socket.onmessage = (event) =>
      handleSocketMessage({
        id: inputState.id,
        socket,
        connectRevision: inputState.connectRevision,
        terminal: inputState.terminal,
        event,
        runtime: inputState.runtime,
      })
    socket.onerror = () =>
      handleSocketError(inputState.id, socket, inputState.connectRevision, inputState.label)
    socket.onclose = () =>
      handleSocketClose(inputState.id, socket, inputState.connectRevision, inputState.runtime)
  }

  return {
    attachSocket,
    matchesConnectionState,
    resolveTerminalSession,
    setConnectingStatus,
  }
}
