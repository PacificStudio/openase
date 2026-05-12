import {
  encodeTerminalPayload,
  mountProjectConversationTerminal,
} from './project-conversation-terminal-panel-helpers'
import {
  TERMINAL_RECONNECT_ATTEMPT_LIMIT,
  clearTerminalReconnectTimer,
  ensureTerminalRuntime,
  forgetTerminalRuntime,
  nextTerminalReconnectDelay,
} from './terminal-manager-runtime'
import type {
  MountedTerminal,
  TerminalInstance,
  TerminalInstanceRuntime,
} from './terminal-manager-types'

type UpdateInstance = (id: string, updates: Partial<TerminalInstance>) => void

type TerminalLifecycleHelpersInput = {
  runtimeMap: Map<string, TerminalInstanceRuntime>
  socketMap: Map<string, WebSocket>
  xtermMap: Map<string, MountedTerminal>
  elementMap: Map<string, HTMLDivElement>
  resizeObserverMap: Map<string, ResizeObserver>
  hasInstance: (id: string) => boolean
  updateInstance: UpdateInstance
  connectTerminal: (id: string, isReconnect?: boolean) => Promise<void>
}

export function createTerminalLifecycleHelpers(input: TerminalLifecycleHelpersInput) {
  function closeSocket(
    id: string,
    options: {
      updateStatus: boolean
      reconnect: boolean
      terminate: boolean
    },
  ) {
    const runtime = ensureTerminalRuntime(input.runtimeMap, id)
    runtime.reconnectEnabled = options.reconnect
    clearTerminalReconnectTimer(input.runtimeMap, id)

    const socket = input.socketMap.get(id)
    input.socketMap.delete(id)
    if (socket?.readyState === WebSocket.OPEN) {
      if (options.terminate) {
        socket.send(JSON.stringify({ type: 'close' }))
      }
      socket.close()
    } else if (socket?.readyState === WebSocket.CONNECTING) {
      socket.close()
    }

    if (options.terminate) {
      runtime.session = null
    }
    if (options.updateStatus) {
      input.updateInstance(id, {
        status: 'closed',
        statusMessage: 'Terminal closed.',
        sessionID: '',
      })
    }
  }

  function unmountTerminal(id: string, forget: boolean) {
    input.resizeObserverMap.get(id)?.disconnect()
    input.resizeObserverMap.delete(id)
    closeSocket(id, { updateStatus: false, reconnect: false, terminate: true })
    input.xtermMap.get(id)?.dispose()
    input.xtermMap.delete(id)
    input.elementMap.delete(id)
    if (forget) {
      forgetTerminalRuntime(input.runtimeMap, id)
    }
  }

  async function mountTerminal(id: string, element: HTMLDivElement) {
    if (input.xtermMap.has(id) && input.elementMap.get(id) === element) {
      return
    }

    const runtime = ensureTerminalRuntime(input.runtimeMap, id)
    runtime.mountRevision += 1
    const mountRevision = runtime.mountRevision

    unmountTerminal(id, false)
    input.elementMap.set(id, element)

    const mounted = await mountProjectConversationTerminal({
      element,
      onData: (data) => {
        const socket = input.socketMap.get(id)
        if (socket?.readyState !== WebSocket.OPEN) return
        socket.send(JSON.stringify({ type: 'input', data: encodeTerminalPayload(data) }))
      },
      onResize: ({ cols, rows }) => {
        const socket = input.socketMap.get(id)
        if (socket?.readyState !== WebSocket.OPEN) return
        socket.send(JSON.stringify({ type: 'resize', cols, rows }))
      },
    })

    if (
      !input.hasInstance(id) ||
      input.runtimeMap.get(id)?.mountRevision !== mountRevision ||
      input.elementMap.get(id) !== element
    ) {
      mounted.dispose()
      return
    }

    input.xtermMap.set(id, mounted)

    const resizeObserver = new ResizeObserver(() => {
      const entry = input.xtermMap.get(id)
      if (!entry) return
      entry.fitAddon.fit()
      const socket = input.socketMap.get(id)
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({ type: 'resize', cols: entry.terminal.cols, rows: entry.terminal.rows }),
        )
      }
    })
    resizeObserver.observe(element)
    input.resizeObserverMap.set(id, resizeObserver)
  }

  function scheduleReconnect(id: string, label: string) {
    const runtime = input.runtimeMap.get(id)
    if (
      !runtime ||
      !runtime.reconnectEnabled ||
      !runtime.session ||
      !input.hasInstance(id) ||
      !input.xtermMap.has(id)
    ) {
      return
    }

    if (runtime.reconnectAttempts >= TERMINAL_RECONNECT_ATTEMPT_LIMIT) {
      runtime.reconnectEnabled = false
      input.updateInstance(id, {
        status: 'error',
        statusMessage: 'Terminal disconnected. Reconnect attempts exhausted.',
        sessionID: '',
      })
      return
    }

    runtime.reconnectAttempts += 1
    const delay = nextTerminalReconnectDelay(runtime.reconnectAttempts)
    input.updateInstance(id, {
      status: 'connecting',
      statusMessage: `Reconnecting shell in ${label}...`,
      sessionID: '',
    })
    runtime.reconnectTimer = setTimeout(() => {
      runtime.reconnectTimer = null
      if (!runtime.reconnectEnabled || !input.hasInstance(id) || !input.xtermMap.has(id)) {
        return
      }
      void input.connectTerminal(id, true)
    }, delay)
  }

  return {
    closeSocket,
    mountTerminal,
    scheduleReconnect,
    unmountTerminal,
  }
}
