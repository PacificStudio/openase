import {
  encodeTerminalPayload,
  mountProjectConversationTerminal,
} from './project-conversation-terminal-panel-helpers'
import { createTerminalConnectionHelpers } from './terminal-manager-connection'
import { createTerminalManagerPanelState } from './terminal-manager-panel-state.svelte'
import {
  TERMINAL_RECONNECT_ATTEMPT_LIMIT,
  clearTerminalReconnectTimer,
  ensureTerminalRuntime,
  forgetTerminalRuntime,
  nextTerminalReconnectDelay,
} from './terminal-manager-runtime'
import type { MountedTerminal, TerminalInstanceRuntime } from './terminal-manager-types'

export function createTerminalManager(input: {
  getConversationId: () => string
  getWorkspacePath: () => string
}) {
  const panelState = createTerminalManagerPanelState()

  // Internal state per instance (not reactive, keyed by id)
  const xtermMap = new Map<string, MountedTerminal>()
  const socketMap = new Map<string, WebSocket>()
  const elementMap = new Map<string, HTMLDivElement>()
  const resizeObserverMap = new Map<string, ResizeObserver>()
  const runtimeMap = new Map<string, TerminalInstanceRuntime>()

  const { attachSocket, matchesConnectionState, resolveTerminalSession, setConnectingStatus } =
    createTerminalConnectionHelpers({
      getConversationId: input.getConversationId,
      hasInstance: panelState.hasInstance,
      listInstances: () => panelState.instances,
      runtimeMap,
      socketMap,
      scheduleReconnect,
      updateInstance: panelState.updateInstance,
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
      !panelState.hasInstance(id) ||
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
      panelState.updateInstance(id, {
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
      !panelState.hasInstance(id) ||
      !xtermMap.has(id)
    ) {
      return
    }

    if (runtime.reconnectAttempts >= TERMINAL_RECONNECT_ATTEMPT_LIMIT) {
      runtime.reconnectEnabled = false
      panelState.updateInstance(id, {
        status: 'error',
        statusMessage: 'Terminal disconnected. Reconnect attempts exhausted.',
        sessionID: '',
      })
      return
    }

    runtime.reconnectAttempts += 1
    const delay = nextTerminalReconnectDelay(runtime.reconnectAttempts)
    panelState.updateInstance(id, {
      status: 'connecting',
      statusMessage: `Reconnecting shell in ${label}...`,
      sessionID: '',
    })
    runtime.reconnectTimer = setTimeout(() => {
      runtime.reconnectTimer = null
      if (!runtime.reconnectEnabled || !panelState.hasInstance(id) || !xtermMap.has(id)) {
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
    if (!conversationId || !entry || !panelState.hasInstance(id)) return

    closeSocket(id, { updateStatus: false, reconnect: false, terminate: false })
    runtime.connectRevision += 1
    const connectRevision = runtime.connectRevision
    runtime.reconnectEnabled = true
    clearTerminalReconnectTimer(runtimeMap, id)
    entry.fitAddon.fit()

    const label = workspacePath || 'workspace root'
    panelState.updateInstance(id, { label })
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
    panelState.updateInstance(id, { sessionID: session.id })
    attachSocket({
      id,
      session,
      connectRevision,
      terminal: currentEntry.terminal,
      runtime,
      label,
    })
  }

  function removeInstance(id: string) {
    unmountTerminal(id, true)
    panelState.removeInstance(id)
  }

  function disposeAll() {
    for (const inst of panelState.instances) {
      unmountTerminal(inst.id, true)
    }
    panelState.reset()
  }

  /** Refits all visible terminals (call after panel resize). */
  function refitAll() {
    for (const [, entry] of xtermMap) {
      entry.fitAddon.fit()
    }
  }

  return {
    get instances() {
      return panelState.instances
    },
    get activeId() {
      return panelState.activeId
    },
    set activeId(id: string) {
      panelState.activeId = id
    },
    get panelOpen() {
      return panelState.panelOpen
    },
    getActiveInstance: panelState.getActiveInstance,
    mountTerminal,
    connectTerminal,
    createInstance: panelState.createInstance,
    removeInstance,
    openPanel: panelState.openPanel,
    togglePanel: panelState.togglePanel,
    closePanel: panelState.closePanel,
    disposeAll,
    refitAll,
  }
}

export type TerminalManager = ReturnType<typeof createTerminalManager>
