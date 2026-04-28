import {
  encodeTerminalPayload,
  mountProjectConversationTerminal,
} from './project-conversation-terminal-panel-helpers'
import { createTerminalConnectionHelpers } from './terminal-manager-connection'
import {
  TERMINAL_RECONNECT_ATTEMPT_LIMIT,
  clearTerminalReconnectTimer,
  ensureTerminalRuntime,
  forgetTerminalRuntime,
  nextTerminalReconnectDelay,
} from './terminal-manager-runtime'
import { createTerminalManagerPanelState } from './terminal-manager-panel-state.svelte'
import type { MountedTerminal, TerminalInstanceRuntime } from './terminal-manager-types'

export function createTerminalManager(input: {
  getConversationId: () => string
  getWorkspacePath: () => string
}) {
  // Internal state per instance (not reactive, keyed by id)
  const xtermMap = new Map<string, MountedTerminal>()
  const socketMap = new Map<string, WebSocket>()
  const elementMap = new Map<string, HTMLDivElement>()
  const resizeObserverMap = new Map<string, ResizeObserver>()
  const runtimeMap = new Map<string, TerminalInstanceRuntime>()

  const state = createTerminalManagerPanelState({
    forgetInstance: (id) => unmountTerminal(id, true),
  })

  const { attachSocket, matchesConnectionState, resolveTerminalSession, setConnectingStatus } =
    createTerminalConnectionHelpers({
      getConversationId: input.getConversationId,
      hasInstance: state.hasInstance,
      listInstances: () => state.instances,
      runtimeMap,
      socketMap,
      scheduleReconnect,
      updateInstance: state.updateInstance,
    })

  async function mountTerminal(id: string, element: HTMLDivElement) {
    // Prevent double-mount
    if (xtermMap.has(id) && elementMap.get(id) === element) return

    const runtime = ensureTerminalRuntime(runtimeMap, id)
    runtime.mountRevision += 1
    const mountRevision = runtime.mountRevision

    // Clean up previous mount for this id
    unmountTerminal(id, false)
    elementMap.set(id, element)

    const mounted = await mountProjectConversationTerminal({
      element,
      onData: (data) => {
        const socket = socketMap.get(id)
        if (socket?.readyState !== WebSocket.OPEN) return
        socket.send(JSON.stringify({ type: 'input', data: encodeTerminalPayload(data) }))
      },
      onResize: ({ cols, rows }) => {
        const socket = socketMap.get(id)
        if (socket?.readyState !== WebSocket.OPEN) return
        socket.send(JSON.stringify({ type: 'resize', cols, rows }))
      },
    })

    if (
      !state.hasInstance(id) ||
      runtimeMap.get(id)?.mountRevision !== mountRevision ||
      elementMap.get(id) !== element
    ) {
      mounted.dispose()
      return
    }

    xtermMap.set(id, mounted)

    const ro = new ResizeObserver(() => {
      const entry = xtermMap.get(id)
      if (!entry) return
      entry.fitAddon.fit()
      const socket = socketMap.get(id)
      if (socket?.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({ type: 'resize', cols: entry.terminal.cols, rows: entry.terminal.rows }),
        )
      }
    })
    ro.observe(element)
    resizeObserverMap.set(id, ro)
  }

  function unmountTerminal(id: string, forget: boolean) {
    resizeObserverMap.get(id)?.disconnect()
    resizeObserverMap.delete(id)
    closeSocket(id, { updateStatus: false, reconnect: false, terminate: true })
    xtermMap.get(id)?.dispose()
    xtermMap.delete(id)
    elementMap.delete(id)
    if (forget) {
      forgetTerminalRuntime(runtimeMap, id)
    }
  }

  function closeSocket(
    id: string,
    options: {
      updateStatus: boolean
      reconnect: boolean
      terminate: boolean
    },
  ) {
    const runtime = ensureTerminalRuntime(runtimeMap, id)
    runtime.reconnectEnabled = options.reconnect
    clearTerminalReconnectTimer(runtimeMap, id)

    const socket = socketMap.get(id)
    socketMap.delete(id)
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
      state.updateInstance(id, {
        status: 'closed',
        statusMessage: 'Terminal closed.',
        sessionID: '',
      })
    }
  }

  function scheduleReconnect(id: string, label: string) {
    const runtime = runtimeMap.get(id)
    if (
      !runtime ||
      !runtime.reconnectEnabled ||
      !runtime.session ||
      !state.hasInstance(id) ||
      !xtermMap.has(id)
    ) {
      return
    }

    if (runtime.reconnectAttempts >= TERMINAL_RECONNECT_ATTEMPT_LIMIT) {
      runtime.reconnectEnabled = false
      state.updateInstance(id, {
        status: 'error',
        statusMessage: 'Terminal disconnected. Reconnect attempts exhausted.',
        sessionID: '',
      })
      return
    }

    runtime.reconnectAttempts += 1
    const delay = nextTerminalReconnectDelay(runtime.reconnectAttempts)
    state.updateInstance(id, {
      status: 'connecting',
      statusMessage: `Reconnecting shell in ${label}...`,
      sessionID: '',
    })
    runtime.reconnectTimer = setTimeout(() => {
      runtime.reconnectTimer = null
      if (!runtime.reconnectEnabled || !state.hasInstance(id) || !xtermMap.has(id)) {
        return
      }
      void connectTerminal(id, true)
    }, delay)
  }

  async function connectTerminal(id: string, isReconnect = false) {
    const conversationId = input.getConversationId()
    const workspacePath = input.getWorkspacePath()
    const runtime = ensureTerminalRuntime(runtimeMap, id)
    const entry = xtermMap.get(id)
    if (!conversationId || !entry || !state.hasInstance(id)) return

    closeSocket(id, { updateStatus: false, reconnect: false, terminate: false })
    runtime.connectRevision += 1
    const connectRevision = runtime.connectRevision
    runtime.reconnectEnabled = true
    clearTerminalReconnectTimer(runtimeMap, id)
    entry.fitAddon.fit()

    const label = workspacePath || 'workspace root'
    state.updateInstance(id, { label })
    setConnectingStatus(id, label, isReconnect)

    const session = await resolveTerminalSession({
      id,
      conversationId,
      connectRevision,
      runtime,
      terminal: entry.terminal,
    })
    if (!session) {
      return
    }

    const currentEntry = xtermMap.get(id)
    if (!currentEntry || !matchesConnectionState(id, conversationId, connectRevision)) {
      return
    }

    runtime.session = session
    state.updateInstance(id, { sessionID: session.id })
    attachSocket({
      id,
      session,
      connectRevision,
      terminal: currentEntry.terminal,
      runtime,
      label,
    })
  }

  /** Refits all visible terminals (call after panel resize). */
  function refitAll() {
    for (const [, entry] of xtermMap) {
      entry.fitAddon.fit()
    }
  }

  return {
    get instances() {
      return state.instances
    },
    get activeId() {
      return state.activeId
    },
    set activeId(id: string) {
      state.activeId = id
    },
    get panelOpen() {
      return state.panelOpen
    },
    getActiveInstance: state.getActiveInstance,
    mountTerminal,
    connectTerminal,
    createInstance: state.createInstance,
    removeInstance: state.removeInstance,
    openPanel: state.openPanel,
    togglePanel: state.togglePanel,
    closePanel: state.closePanel,
    disposeAll: state.disposeAll,
    refitAll,
  }
}

export type TerminalManager = ReturnType<typeof createTerminalManager>
